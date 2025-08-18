// Package transport provides a unified interface for connecting to sshx servers
// via either gRPC or WebSocket protocols, with automatic fallback capability.
package transport

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"sshx-go/pkg/proto"
)

// SshxTransport provides a unified interface for both gRPC and WebSocket
// transports, allowing seamless fallback between connection types.
type SshxTransport interface {
	// Open a new session on the server.
	Open(ctx context.Context, request *proto.OpenRequest) (*proto.OpenResponse, error)

	// Channel establishes a bidirectional streaming channel for real-time communication.
	// Returns inbound channel for server messages and outbound channel for client messages.
	Channel(ctx context.Context) (chan *proto.ServerUpdate, chan *proto.ClientUpdate, error)

	// Close an existing session on the server.
	Close(ctx context.Context, request *proto.CloseRequest) error

	// ConnectionType returns the connection type for logging/debugging purposes.
	ConnectionType() string

	// Cleanup any resources held by the transport.
	Cleanup() error
}

// ConnectionMethod represents the method used to establish the connection.
type ConnectionMethod int

const (
	// MethodGrpc indicates direct gRPC connection succeeded.
	MethodGrpc ConnectionMethod = iota
	// MethodWebSocketFallback indicates WebSocket fallback was used after gRPC failed.
	MethodWebSocketFallback
)

func (m ConnectionMethod) String() string {
	switch m {
	case MethodGrpc:
		return "gRPC"
	case MethodWebSocketFallback:
		return "WebSocket"
	default:
		return "Unknown"
	}
}

// ConnectionResult contains the result of a connection attempt.
type ConnectionResult struct {
	// Transport is the established transport connection.
	Transport SshxTransport
	// Method is the connection method that was used.
	Method ConnectionMethod
}

// ConnectionConfig holds configuration for creating a connection.
type ConnectionConfig struct {
	// VerboseErrors enables verbose error reporting during fallback attempts.
	VerboseErrors bool
	// GrpcTimeout is custom timeout for gRPC connection attempts.
	GrpcTimeout time.Duration
	// WebSocketTimeout is custom timeout for WebSocket connection attempts.
	WebSocketTimeout time.Duration
}

// DefaultConnectionConfig returns a default connection configuration.
func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		VerboseErrors:    false,
		GrpcTimeout:     3 * time.Second,
		WebSocketTimeout: 5 * time.Second,
	}
}

// CliRequest represents a CLI WebSocket request message with correlation ID.
type CliRequest struct {
	// ID is unique request ID for correlation.
	ID string `json:"id"`
	// Message is the actual request message.
	Message CliMessage `json:"message"`
}

// CliResponse represents a CLI WebSocket response message with correlation ID.
type CliResponse struct {
	// ID is request ID this response corresponds to.
	ID string `json:"id"`
	// Message is the actual response message.
	Message CliResponseMessage `json:"message"`
}

// CliMessage represents CLI-specific request message types.
type CliMessage struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// CliResponseMessage represents CLI-specific response message types.
type CliResponseMessage struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// StreamingMessage is for messages sent during the streaming phase (no correlation ID).
type StreamingMessage struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// OpenSessionRequest matches the Rust CliMessage::OpenSession
type OpenSessionRequest struct {
	Origin             string `json:"origin"`
	EncryptedZeros     []byte `json:"encryptedZeros"`
	Name               string `json:"name"`
	WritePasswordHash  []byte `json:"writePasswordHash,omitempty"`
}

// OpenSessionResponse matches the Rust CliResponseMessage::OpenSession
type OpenSessionResponse struct {
	Name  string `json:"name"`
	Token string `json:"token"`
	URL   string `json:"url"`
}

// CloseSessionRequest matches the Rust CliMessage::CloseSession
type CloseSessionRequest struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

// StartChannelRequest matches the Rust CliMessage::StartChannel
type StartChannelRequest struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

// TerminalDataMessage matches the Rust CliMessage::TerminalData
type TerminalDataMessage struct {
	ID   uint32 `json:"id"`
	Data []byte `json:"data"`
	Seq  uint64 `json:"seq"`
}

// CreatedShellMessage matches the Rust CliMessage::CreatedShell
type CreatedShellMessage struct {
	ID uint32 `json:"id"`
	X  int32  `json:"x"`
	Y  int32  `json:"y"`
}

// ClosedShellMessage matches the Rust CliMessage::ClosedShell
type ClosedShellMessage struct {
	ID uint32 `json:"id"`
}

// PongMessage matches the Rust CliMessage::Pong
type PongMessage struct {
	Timestamp uint64 `json:"timestamp"`
}

// ErrorMessage matches the Rust CliMessage::Error
type ErrorMessage struct {
	Message string `json:"message"`
}

// TerminalInputMessage matches the Rust CliResponseMessage::TerminalInput
type TerminalInputMessage struct {
	ID     uint32 `json:"id"`
	Data   []byte `json:"data"`
	Offset uint64 `json:"offset"`
}

// CreateShellMessage matches the Rust CliResponseMessage::CreateShell
type CreateShellMessage struct {
	ID uint32 `json:"id"`
	X  int32  `json:"x"`
	Y  int32  `json:"y"`
}

// CloseShellMessage matches the Rust CliResponseMessage::CloseShell
type CloseShellMessage struct {
	ID uint32 `json:"id"`
}

// SyncMessage matches the Rust CliResponseMessage::Sync
type SyncMessage struct {
	SequenceNumbers map[uint32]uint64 `json:"sequenceNumbers"`
}

// ResizeMessage matches the Rust CliResponseMessage::Resize
type ResizeMessage struct {
	ID   uint32 `json:"id"`
	Rows uint32 `json:"rows"`
	Cols uint32 `json:"cols"`
}

// PingMessage matches the Rust CliResponseMessage::Ping
type PingMessage struct {
	Timestamp uint64 `json:"timestamp"`
}

// GrpcToWebSocketURL converts a gRPC server URL to its corresponding WebSocket CLI endpoint.
//
// Examples:
//   grpcToWebSocketURL("https://example.com:8051", "my-session") -> "wss://example.com:8051/api/cli/my-session"
//   grpcToWebSocketURL("http://localhost:8051", "test") -> "ws://localhost:8051/api/cli/test"
func GrpcToWebSocketURL(grpcURL, sessionName string) string {
	url := strings.ReplaceAll(grpcURL, "https://", "wss://")
	url = strings.ReplaceAll(url, "http://", "ws://")
	
	// Handle the case where the URL might end with a slash
	base := strings.TrimSuffix(url, "/")
	
	return fmt.Sprintf("%s/api/cli/%s", base, sessionName)
}

// Helper functions for message conversion

// ClientUpdateToCliMessage converts a gRPC ClientUpdate to CLI message format for WebSocket.
func ClientUpdateToCliMessage(update *proto.ClientUpdate) (CliMessage, error) {
	switch msg := update.ClientMessage.(type) {
	case *proto.ClientUpdate_Hello:
		// Parse "name,token" format and convert to StartChannel
		parts := strings.Split(msg.Hello, ",")
		if len(parts) != 2 {
			return CliMessage{}, fmt.Errorf("invalid hello format")
		}
		return CliMessage{
			Type: "startChannel",
			Data: StartChannelRequest{
				Name:  parts[0],
				Token: parts[1],
			},
		}, nil
	case *proto.ClientUpdate_Data:
		return CliMessage{
			Type: "terminalData",
			Data: TerminalDataMessage{
				ID:   msg.Data.Id,
				Data: msg.Data.Data,
				Seq:  msg.Data.Seq,
			},
		}, nil
	case *proto.ClientUpdate_CreatedShell:
		return CliMessage{
			Type: "createdShell",
			Data: CreatedShellMessage{
				ID: msg.CreatedShell.Id,
				X:  msg.CreatedShell.X,
				Y:  msg.CreatedShell.Y,
			},
		}, nil
	case *proto.ClientUpdate_ClosedShell:
		return CliMessage{
			Type: "closedShell",
			Data: ClosedShellMessage{
				ID: msg.ClosedShell,
			},
		}, nil
	case *proto.ClientUpdate_Pong:
		return CliMessage{
			Type: "pong",
			Data: PongMessage{
				Timestamp: msg.Pong,
			},
		}, nil
	case *proto.ClientUpdate_Error:
		return CliMessage{
			Type: "error",
			Data: ErrorMessage{
				Message: msg.Error,
			},
		}, nil
	default:
		// Handle empty ClientUpdate (heartbeat)
		return CliMessage{Type: "heartbeat"}, nil
	}
}

// CliResponseToServerUpdate converts a CLI response message to ServerUpdate for gRPC compatibility.
func CliResponseToServerUpdate(cliMsg CliResponseMessage) (*proto.ServerUpdate, error) {
	switch cliMsg.Type {
	case "terminalInput":
		data, ok := cliMsg.Data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid terminalInput data")
		}
		
		idFloat, ok := data["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid id in terminalInput")
		}
		
		dataBytes, ok := data["data"].([]byte)
		if !ok {
			// Try to handle base64 or string data
			if dataStr, ok := data["data"].(string); ok {
				dataBytes = []byte(dataStr)
			} else {
				return nil, fmt.Errorf("invalid data in terminalInput")
			}
		}
		
		offsetFloat, ok := data["offset"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid offset in terminalInput")
		}
		
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_Input{
				Input: &proto.TerminalInput{
					Id:     uint32(idFloat),
					Data:   dataBytes,
					Offset: uint64(offsetFloat),
				},
			},
		}, nil

	case "createShell":
		data, ok := cliMsg.Data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid createShell data")
		}
		
		idFloat, ok := data["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid id in createShell")
		}
		
		xFloat, ok := data["x"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid x in createShell")
		}
		
		yFloat, ok := data["y"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid y in createShell")
		}
		
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_CreateShell{
				CreateShell: &proto.NewShell{
					Id: uint32(idFloat),
					X:  int32(xFloat),
					Y:  int32(yFloat),
				},
			},
		}, nil

	case "closeShell":
		data, ok := cliMsg.Data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid closeShell data")
		}
		
		idFloat, ok := data["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid id in closeShell")
		}
		
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_CloseShell{
				CloseShell: uint32(idFloat),
			},
		}, nil

	case "sync":
		data, ok := cliMsg.Data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid sync data")
		}
		
		seqNums, ok := data["sequenceNumbers"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid sequenceNumbers in sync")
		}
		
		sequenceMap := make(map[uint32]uint64)
		for k, v := range seqNums {
			// JSON unmarshaling converts numbers to float64
			if vFloat, ok := v.(float64); ok {
				if id, err := parseUint32(k); err == nil {
					sequenceMap[id] = uint64(vFloat)
				}
			}
		}
		
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_Sync{
				Sync: &proto.SequenceNumbers{
					Map: sequenceMap,
				},
			},
		}, nil

	case "resize":
		data, ok := cliMsg.Data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid resize data")
		}
		
		idFloat, ok := data["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid id in resize")
		}
		
		rowsFloat, ok := data["rows"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid rows in resize")
		}
		
		colsFloat, ok := data["cols"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid cols in resize")
		}
		
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_Resize{
				Resize: &proto.TerminalSize{
					Id:   uint32(idFloat),
					Rows: uint32(rowsFloat),
					Cols: uint32(colsFloat),
				},
			},
		}, nil

	case "ping":
		data, ok := cliMsg.Data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid ping data")
		}
		
		timestampFloat, ok := data["timestamp"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid timestamp in ping")
		}
		
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_Ping{
				Ping: uint64(timestampFloat),
			},
		}, nil

	case "error":
		data, ok := cliMsg.Data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid error data")
		}
		
		message, ok := data["message"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid message in error")
		}
		
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_Error{
				Error: message,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported CLI response message type: %s", cliMsg.Type)
	}
}

// Helper function to parse uint32 from string (for JSON map keys)
func parseUint32(s string) (uint32, error) {
	var result uint32
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// responseWriter is a helper for managing correlated WebSocket responses
type responseWriter struct {
	pendingRequests map[string]chan CliResponseMessage
	mu              sync.RWMutex
	nextID          uint64
	nextIDMu        sync.Mutex
}

func newResponseWriter() *responseWriter {
	return &responseWriter{
		pendingRequests: make(map[string]chan CliResponseMessage),
	}
}

func (rw *responseWriter) nextRequestID() string {
	rw.nextIDMu.Lock()
	defer rw.nextIDMu.Unlock()
	rw.nextID++
	return fmt.Sprintf("req_%d", rw.nextID)
}

func (rw *responseWriter) addPendingRequest(id string, ch chan CliResponseMessage) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.pendingRequests[id] = ch
}

func (rw *responseWriter) handleResponse(response CliResponse) {
	rw.mu.Lock()
	ch, exists := rw.pendingRequests[response.ID]
	if exists {
		delete(rw.pendingRequests, response.ID)
	}
	rw.mu.Unlock()
	
	if exists {
		select {
		case ch <- response.Message:
		default:
			// Channel might be closed
		}
	}
}

func (rw *responseWriter) removePendingRequest(id string) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	delete(rw.pendingRequests, id)
}
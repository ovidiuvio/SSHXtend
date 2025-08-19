package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"sshx-go/pkg/proto"
	"sshx-go/pkg/util"
)

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

// TerminalDataMessage for streaming terminal data
type TerminalDataMessage struct {
	ID   uint32       `json:"id"`
	Data BytesAsArray `json:"data"`  // Back to JSON arrays - this was working better
	Seq  uint64       `json:"seq"`
}

// CreatedShellMessage for acknowledging shell creation
type CreatedShellMessage struct {
	ID uint32 `json:"id"`
	X  int32  `json:"x"`
	Y  int32  `json:"y"`
}

// ClosedShellMessage for acknowledging shell closure
type ClosedShellMessage struct {
	ID uint32 `json:"id"`
}

// PongMessage for latency measurement
type PongMessage struct {
	Timestamp uint64 `json:"timestamp"`
}

// ErrorMessage for client errors
type ErrorMessage struct {
	Message string `json:"message"`
}

// CliMessage represents CLI-specific request message types using tagged union format.
// This matches the Rust enum CliMessage exactly using tagged union serialization
type CliMessage struct {
	OpenSession  *OpenSessionRequest  `json:"openSession,omitempty"`
	CloseSession *CloseSessionRequest `json:"closeSession,omitempty"`
	StartChannel *StartChannelRequest `json:"startChannel,omitempty"`
	TerminalData *TerminalDataMessage `json:"terminalData,omitempty"`
	CreatedShell *CreatedShellMessage `json:"createdShell,omitempty"`
	ClosedShell  *ClosedShellMessage  `json:"closedShell,omitempty"`
	Pong         *PongMessage         `json:"pong,omitempty"`
	Error        *ErrorMessage        `json:"error,omitempty"`
}

// CliResponseMessage represents CLI-specific response message types using tagged union format.
// This matches the Rust enum CliResponseMessage exactly using tagged union serialization
type CliResponseMessage struct {
	OpenSession   *OpenSessionResponse   `json:"openSession,omitempty"`
	CloseSession  *CloseSessionResponse  `json:"closeSession,omitempty"`
	StartChannel  *StartChannelResponse  `json:"startChannel,omitempty"`
	TerminalInput *TerminalInputResponse `json:"terminalInput,omitempty"`
	CreateShell   *CreateShellResponse   `json:"createShell,omitempty"`
	CloseShell    *CloseShellResponse    `json:"closeShell,omitempty"`
	Sync          *SyncResponse          `json:"sync,omitempty"`
	Resize        *ResizeResponse        `json:"resize,omitempty"`
	Ping          *PingResponse          `json:"ping,omitempty"`
	Error         *ErrorResponse         `json:"error,omitempty"`
}

type OpenSessionResponse struct {
	Name  string `json:"name"`
	Token string `json:"token"`
	URL   string `json:"url"`
}

type CloseSessionResponse struct{}

type StartChannelResponse struct{}

// Typed response structures to preserve precision during JSON unmarshaling
type TerminalInputResponse struct {
	ID     uint32 `json:"id"`
	Data   []uint8 `json:"data"`   // JSON array of integers, not base64
	Offset uint64  `json:"offset"` // Proper uint64 to preserve large integer precision
}


type CreateShellResponse struct {
	ID uint32 `json:"id"`
	X  int32  `json:"x"`
	Y  int32  `json:"y"`
}

type CloseShellResponse struct {
	ID uint32 `json:"id"`
}

type SyncResponse struct {
	SequenceNumbers map[string]uint64 `json:"sequenceNumbers"`
}

type ResizeResponse struct {
	ID   uint32 `json:"id"`
	Rows uint32 `json:"rows"`
	Cols uint32 `json:"cols"`
}

type PingResponse struct {
	Timestamp uint64 `json:"timestamp"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

// BytesAsArray is a custom type that serializes []byte as JSON array
type BytesAsArray []byte

// MarshalJSON implements custom JSON marshaling to send []byte as array
func (b BytesAsArray) MarshalJSON() ([]byte, error) {
	if b == nil {
		return []byte("null"), nil
	}
	
	// Convert to slice of integers for JSON array format
	result := make([]int, len(b))
	for i, v := range b {
		result[i] = int(v)
	}
	return json.Marshal(result)
}

// UnmarshalJSON implements custom JSON unmarshaling from array format
func (b *BytesAsArray) UnmarshalJSON(data []byte) error {
	var result []int
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	
	*b = make([]byte, len(result))
	for i, v := range result {
		(*b)[i] = byte(v)
	}
	return nil
}

// OpenSessionRequest matches the Rust CliMessage::OpenSession
type OpenSessionRequest struct {
	Origin            string         `json:"origin"`
	EncryptedZeros    BytesAsArray   `json:"encrypted_zeros"`
	Name              string         `json:"name"`
	WritePasswordHash *BytesAsArray  `json:"write_password_hash,omitempty"`
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

// WebSocketTransport implements the SshxTransport interface using WebSocket communication.
type WebSocketTransport struct {
	conn            *websocket.Conn
	responseWriter  *responseWriter
	serverUpdates   chan *proto.ServerUpdate
	done            chan struct{}
	mu              sync.RWMutex
	closed          bool
}

// ConnectWebSocket creates a new WebSocket transport by connecting to a server.
func ConnectWebSocket(endpoint string) (*WebSocketTransport, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WebSocket URL: %w", err)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	// Configure WebSocket connection for proper keep-alive
	// We'll update the read deadline on every message received in readLoop
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	transport := &WebSocketTransport{
		conn:           conn,
		responseWriter: newResponseWriter(),
		serverUpdates:  make(chan *proto.ServerUpdate, 256),
		done:           make(chan struct{}),
	}

	// Start background tasks to handle WebSocket communication
	go transport.readLoop()
	go transport.pingLoop()

	return transport, nil
}

// Open opens a new session on the server.
func (w *WebSocketTransport) Open(ctx context.Context, request *proto.OpenRequest) (*proto.OpenResponse, error) {
	// Send raw bytes as JSON arrays to match server expectations
	var writePasswordHash *BytesAsArray
	if request.WritePasswordHash != nil {
		hash := BytesAsArray(request.WritePasswordHash)
		writePasswordHash = &hash
	}
	
	req := CliRequest{
		ID: w.responseWriter.nextRequestID(),
		Message: CliMessage{
			OpenSession: &OpenSessionRequest{
				Origin:            request.Origin,
				EncryptedZeros:    BytesAsArray(request.EncryptedZeros),
				Name:              request.Name,
				WritePasswordHash: writePasswordHash,
			},
		},
	}

	// Debug: Log the request being sent
	jsonBytes, _ := json.Marshal(req)
	util.DebugLog("WebSocket sending Open request: %s", string(jsonBytes))
	util.DebugLog("Go client encrypted_zeros length: %d bytes", len(request.EncryptedZeros))

	response, err := w.sendRequestWithResponse(ctx, req, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("WebSocket open request failed: %w", err)
	}

	// Handle tagged union response format
	if response.OpenSession != nil {
		util.DebugLog("WebSocket Open response: Name=%s, Token=%s, URL=%s", 
			response.OpenSession.Name, response.OpenSession.Token, response.OpenSession.URL)
		util.DebugLog("WebSocket session validation - Server returned session name: %s", response.OpenSession.Name)
		return &proto.OpenResponse{
			Name:  response.OpenSession.Name,
			Token: response.OpenSession.Token,
			Url:   response.OpenSession.URL,
		}, nil

	} else if response.Error != nil {
		return nil, fmt.Errorf("server error: %s", response.Error.Message)

	} else {
		return nil, fmt.Errorf("unexpected response type for open request")
	}
}

// Channel establishes a bidirectional streaming channel for real-time communication.
func (w *WebSocketTransport) Channel(ctx context.Context) (chan *proto.ServerUpdate, chan *proto.ClientUpdate, error) {
	// Create channels for this streaming session
	serverChan := make(chan *proto.ServerUpdate, 256)
	clientChan := make(chan *proto.ClientUpdate, 256)
	
	// Handle the protocol in a separate goroutine
	go func() {
		defer func() {
			util.DebugLog("WebSocket channel protocol goroutine exiting")
			close(serverChan)
		}()
		
		// Wait for the first Hello message from the controller via clientChan
		var hello string
		var helloReceived bool
		
		for !helloReceived {
			select {
			case firstUpdate := <-clientChan:
				if firstUpdate.ClientMessage == nil {
					continue // Skip heartbeats
				}
				hello = firstUpdate.GetHello()
				if hello != "" {
					helloReceived = true
					util.DebugLog("WebSocket received Hello: %s", hello)
				} else {
					// Not a Hello, this is an error in protocol
					util.DebugLog("WebSocket received non-Hello message while waiting for Hello: %T", firstUpdate.ClientMessage)
					return
				}
			case <-ctx.Done():
				return
			case <-w.done:
				return
			}
		}
		
		// Parse name and token from Hello message
		parts := strings.Split(hello, ",")
		if len(parts) != 2 {
			log.Printf("Invalid hello format: %s", hello)
			return
		}
		name, token := parts[0], parts[1]
		
		// Send StartChannel request and wait for response
		req := CliRequest{
			ID: w.responseWriter.nextRequestID(),
			Message: CliMessage{
				StartChannel: &StartChannelRequest{
					Name:  name,
					Token: token,
				},
			},
		}
		
		response, err := w.sendRequestWithResponse(ctx, req, 30*time.Second)
		if err != nil {
			log.Printf("Failed to start WebSocket channel: %v", err)
			return
		}
		
		// Verify we got the expected response
		if response.StartChannel != nil {
			util.DebugLog("WebSocket channel started successfully")
		} else if response.Error != nil {
			log.Printf("Server error starting channel: %s", response.Error.Message)
			return
		} else {
			log.Printf("Unexpected response to StartChannel")
			return
		}
		
		// Now handle remaining outbound messages
		util.DebugLog("WebSocket entering streaming phase")
		var messageCount int64
		for {
			select {
			case update, ok := <-clientChan:
				if !ok {
					util.DebugLog("WebSocket clientChan closed after %d messages", messageCount)
					return
				}
				
				// Skip heartbeats
				if update.ClientMessage == nil {
					continue
				}
				
				messageCount++
				
				cliMsg, err := ClientUpdateToCliMessage(update)
				if err != nil {
					log.Printf("WebSocket failed to convert client message #%d: %v", messageCount, err)
					continue
				}
				
				// Skip empty messages (Hello already handled)
				if isEmptyCliMessage(cliMsg) {
					continue
				}
				
				// Create streaming request - these don't get individual responses
				requestID := fmt.Sprintf("stream_%d", time.Now().UnixNano())
				req := CliRequest{
					ID:      requestID,
					Message: cliMsg,
				}
				
				// Serialize to JSON
				jsonData, err := json.Marshal(req)
				if err != nil {
					log.Printf("Failed to serialize client message: %v", err)
					continue
				}
				
				// Write to WebSocket
				w.mu.Lock()
				if w.closed {
					w.mu.Unlock()
					log.Printf("WebSocket transport closed while sending message #%d", messageCount)
					return
				}
				err = w.conn.WriteMessage(websocket.TextMessage, jsonData)
				w.mu.Unlock()
				
				if err != nil {
					log.Printf("WebSocket failed to send outbound message #%d: %v", messageCount, err)
					return
				}
				util.DebugLog("WebSocket sent streaming message #%d: %s", messageCount, string(jsonData))
				
			case <-ctx.Done():
				return
			case <-w.done:
				return
			}
		}
	}()
	
	// Start goroutine to forward server messages
	go func() {
		defer func() {
			util.DebugLog("WebSocket server message forwarder exiting")
		}()
		
		var serverMessageCount int64
		for {
			select {
			case update, ok := <-w.serverUpdates:
				if !ok {
					util.DebugLog("WebSocket serverUpdates channel closed after %d messages", serverMessageCount)
					return
				}
				serverMessageCount++
				util.DebugLog("WebSocket forwarding server message #%d: %T to controller", serverMessageCount, update.ServerMessage)
				select {
				case serverChan <- update:
					util.DebugLog("WebSocket successfully forwarded server message #%d", serverMessageCount)
				case <-ctx.Done():
					return
				case <-w.done:
					return
				}
			case <-ctx.Done():
				return
			case <-w.done:
				return
			}
		}
	}()
	
	// Return channels immediately
	return serverChan, clientChan, nil
}

// Close closes an existing session on the server.
func (w *WebSocketTransport) Close(ctx context.Context, request *proto.CloseRequest) error {
	req := CliRequest{
		ID: w.responseWriter.nextRequestID(),
		Message: CliMessage{
			CloseSession: &CloseSessionRequest{
				Name:  request.Name,
				Token: request.Token,
			},
		},
	}

	response, err := w.sendRequestWithResponse(ctx, req, 30*time.Second)
	if err != nil {
		return fmt.Errorf("WebSocket close request failed: %w", err)
	}

	// Handle tagged union response format
	if response.CloseSession != nil {
		return nil
	} else if response.Error != nil {
		return fmt.Errorf("server error: %s", response.Error.Message)
	} else {
		return fmt.Errorf("unexpected response type for close request")
	}
}

// ConnectionType returns the connection type for logging/debugging purposes.
func (w *WebSocketTransport) ConnectionType() string {
	return "WebSocket"
}

// Cleanup any resources held by the transport.
func (w *WebSocketTransport) Cleanup() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true

	// Close the done channel to signal all goroutines to stop
	select {
	case <-w.done:
		// Channel already closed
	default:
		close(w.done)
	}
	
	// Send proper WebSocket close frame before closing connection
	closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	w.conn.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(5*time.Second))
	
	// Close the WebSocket connection
	err := w.conn.Close()
	
	// Don't close the channels here - let the goroutines handle their own cleanup
	// to avoid race conditions
	
	return err
}

// sendRequestWithResponse sends a request and waits for a correlated response.
func (w *WebSocketTransport) sendRequestWithResponse(ctx context.Context, req CliRequest, timeout time.Duration) (CliResponseMessage, error) {
	responseCh := make(chan CliResponseMessage, 1)
	w.responseWriter.addPendingRequest(req.ID, responseCh)

	// Send the request
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return CliResponseMessage{}, fmt.Errorf("transport is closed")
	}

	if err := w.conn.WriteJSON(req); err != nil {
		w.mu.RUnlock()
		w.responseWriter.removePendingRequest(req.ID)
		return CliResponseMessage{}, fmt.Errorf("failed to send request: %w", err)
	}
	w.mu.RUnlock()

	// Wait for response with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case response := <-responseCh:
		return response, nil
	case <-timeoutCtx.Done():
		w.responseWriter.removePendingRequest(req.ID)
		return CliResponseMessage{}, fmt.Errorf("request timed out")
	case <-w.done:
		return CliResponseMessage{}, fmt.Errorf("transport closed")
	}
}

// readLoop handles incoming WebSocket messages.
func (w *WebSocketTransport) readLoop() {
	defer func() {
		// Signal that the connection is broken
		w.mu.Lock()
		if !w.closed {
			w.closed = true
			// Close done channel to signal other goroutines
			select {
			case <-w.done:
				// Already closed
			default:
				close(w.done)
			}
			// Close server updates channel
			close(w.serverUpdates)
		}
		w.mu.Unlock()
	}()

	for {
		select {
		case <-w.done:
			return
		default:
		}

		// Update read deadline to detect stale connections
		w.conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		
		_, message, err := w.conn.ReadMessage()
		if err != nil {
			// Don't log expected close errors or "use of closed network connection" errors
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) &&
				!strings.Contains(err.Error(), "use of closed network connection") &&
				!strings.Contains(err.Error(), "timeout") {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		if err := w.handleIncomingMessage(message); err != nil {
			log.Printf("Error handling WebSocket message: %v", err)
		}
	}
}

// handleIncomingMessage processes incoming WebSocket messages.
func (w *WebSocketTransport) handleIncomingMessage(message []byte) error {
	// Try to parse as a correlated response first
	var cliResponse CliResponse
	if err := json.Unmarshal(message, &cliResponse); err == nil && cliResponse.ID != "" {
		util.DebugLog("Successfully parsed CliResponse with ID: %s", cliResponse.ID)
		// Handle streaming messages (sent with "server_update" ID) - matches Rust implementation
		if cliResponse.ID == "server_update" {
			util.DebugLog("WebSocket received server_update message: %+v", cliResponse.Message)
			serverUpdate, err := CliResponseToServerUpdate(cliResponse.Message)
			if err != nil {
				log.Printf("Failed to convert server_update to ServerUpdate: %v, message: %+v", err, cliResponse.Message)
				return fmt.Errorf("failed to convert CLI response to server update: %w", err)
			}
			util.DebugLog("WebSocket converted to ServerUpdate: %T", serverUpdate.ServerMessage)

			select {
			case w.serverUpdates <- serverUpdate:
				util.DebugLog("WebSocket forwarded server update to channel")
			case <-w.done:
			}
			return nil
		}
		
		// Handle regular request-response messages
		w.responseWriter.handleResponse(cliResponse)
		return nil
	}

	// If we get here, the message format was invalid - matches Rust debug logging
	log.Printf("Failed to parse WebSocket message: %s", string(message))
	return nil
}

// parseJSONBytes converts JSON data back to []byte, handling JSON arrays
func parseJSONBytes(value interface{}) []byte {
	switch v := value.(type) {
	case []interface{}:
		// JSON array format [137,10,116,...] - used when server sends data
		result := make([]byte, len(v))
		for i, item := range v {
			if num, ok := item.(float64); ok {
				result[i] = byte(num)
			}
		}
		return result
	default:
		log.Printf("Warning: parseJSONBytes received unsupported type: %T", value)
		return nil
	}
}

// isEmptyCliMessage checks if a CliMessage has no fields set
func isEmptyCliMessage(msg CliMessage) bool {
	return msg.OpenSession == nil &&
		msg.CloseSession == nil &&
		msg.StartChannel == nil &&
		msg.TerminalData == nil &&
		msg.CreatedShell == nil &&
		msg.ClosedShell == nil &&
		msg.Pong == nil &&
		msg.Error == nil
}

// ClientUpdateToCliMessage converts a proto.ClientUpdate to a CliMessage.
func ClientUpdateToCliMessage(update *proto.ClientUpdate) (CliMessage, error) {
	if update == nil {
		return CliMessage{}, fmt.Errorf("nil client update")
	}
	
	// Handle heartbeat messages (empty ClientUpdate with no ClientMessage)
	if update.ClientMessage == nil {
		// Skip heartbeat messages - they don't need to be sent over WebSocket
		return CliMessage{}, nil
	}

	switch msg := update.ClientMessage.(type) {
	case *proto.ClientUpdate_Hello:
		// Hello is handled separately in handleOutgoingClientUpdate
		return CliMessage{}, nil
	case *proto.ClientUpdate_Data:
		return CliMessage{
			TerminalData: &TerminalDataMessage{
				ID:   msg.Data.Id,
				Data: BytesAsArray(msg.Data.Data),  // Back to JSON arrays
				Seq:  msg.Data.Seq,
			},
		}, nil
	case *proto.ClientUpdate_CreatedShell:
		return CliMessage{
			CreatedShell: &CreatedShellMessage{
				ID: msg.CreatedShell.Id,
				X:  msg.CreatedShell.X,
				Y:  msg.CreatedShell.Y,
			},
		}, nil
	case *proto.ClientUpdate_ClosedShell:
		return CliMessage{
			ClosedShell: &ClosedShellMessage{
				ID: msg.ClosedShell,
			},
		}, nil
	case *proto.ClientUpdate_Pong:
		return CliMessage{
			Pong: &PongMessage{
				Timestamp: msg.Pong,
			},
		}, nil
	case *proto.ClientUpdate_Error:
		return CliMessage{
			Error: &ErrorMessage{
				Message: msg.Error,
			},
		}, nil
	default:
		return CliMessage{}, fmt.Errorf("unsupported client message type: %T", msg)
	}
}

// CliResponseToServerUpdate converts a CliResponseMessage to a proto.ServerUpdate.
func CliResponseToServerUpdate(cliMsg CliResponseMessage) (*proto.ServerUpdate, error) {
	// Handle tagged union format - check each field for the message type
	if cliMsg.TerminalInput != nil {
		// Convert []uint8 to []byte  
		dataBytes := make([]byte, len(cliMsg.TerminalInput.Data))
		for i, v := range cliMsg.TerminalInput.Data {
			dataBytes[i] = byte(v)
		}
		
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_Input{
				Input: &proto.TerminalInput{
					Id:     cliMsg.TerminalInput.ID,
					Data:   dataBytes,
					Offset: cliMsg.TerminalInput.Offset, // ← Now preserves full uint64 precision!
				},
			},
		}, nil
	}
	
	if cliMsg.CreateShell != nil {
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_CreateShell{
				CreateShell: &proto.NewShell{
					Id: cliMsg.CreateShell.ID,
					X:  cliMsg.CreateShell.X,
					Y:  cliMsg.CreateShell.Y,
				},
			},
		}, nil
	}
	
	if cliMsg.CloseShell != nil {
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_CloseShell{
				CloseShell: cliMsg.CloseShell.ID,
			},
		}, nil
	}
	
	if cliMsg.Sync != nil {
		syncMap := make(map[uint32]uint64)
		for k, v := range cliMsg.Sync.SequenceNumbers {
			var key uint32
			fmt.Sscanf(k, "%d", &key)
			syncMap[key] = v // Now preserves uint64 precision
		}
		
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_Sync{
				Sync: &proto.SequenceNumbers{
					Map: syncMap,
				},
			},
		}, nil
	}
	
	if cliMsg.Resize != nil {
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_Resize{
				Resize: &proto.TerminalSize{
					Id:   cliMsg.Resize.ID,
					Rows: cliMsg.Resize.Rows,
					Cols: cliMsg.Resize.Cols,
				},
			},
		}, nil
	}
	
	if cliMsg.Ping != nil {
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_Ping{
				Ping: cliMsg.Ping.Timestamp, // Now preserves uint64 precision
			},
		}, nil
	}
	
	if cliMsg.Error != nil {
		return &proto.ServerUpdate{
			ServerMessage: &proto.ServerUpdate_Error{
				Error: cliMsg.Error.Message,
			},
		}, nil
	}
	
	return nil, fmt.Errorf("unknown message type in CliResponseMessage - all fields are nil: %+v", cliMsg)
}

// pingLoop sends periodic ping frames to keep the WebSocket connection alive.
func (w *WebSocketTransport) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.mu.Lock()
			if w.closed {
				w.mu.Unlock()
				return
			}
			
			err := w.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
			w.mu.Unlock()
			
			if err != nil {
				log.Printf("WebSocket ping failed: %v", err)
				return
			}
		case <-w.done:
			return
		}
	}
}

// GrpcToWebSocketURL converts a gRPC server URL to its corresponding WebSocket CLI endpoint.
func GrpcToWebSocketURL(grpcURL, sessionName string) string {
	wsURL := strings.Replace(grpcURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	
	// Handle the case where the URL might end with a slash
	base := strings.TrimSuffix(wsURL, "/")
	
	return fmt.Sprintf("%s/api/cli/%s", base, sessionName)
}
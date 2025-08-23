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
	"google.golang.org/protobuf/proto"
	pb "sshx-go/pkg/proto"
	"sshx-go/pkg/util"
)

// Using protobuf CliRequest directly from pb package

// Using protobuf CliResponse directly from pb package

// Using protobuf types directly from pb package now

// Using protobuf pb.CliRequest_CliMessage directly

// Using protobuf pb.CliResponse_CliResponseMessage directly

// All response types now use protobuf pb.* types directly

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
	pendingRequests map[string]chan *pb.CliResponse
	mu              sync.RWMutex
	nextID          uint64
	nextIDMu        sync.Mutex
}

func newResponseWriter() *responseWriter {
	return &responseWriter{
		pendingRequests: make(map[string]chan *pb.CliResponse),
	}
}

func (rw *responseWriter) nextRequestID() string {
	rw.nextIDMu.Lock()
	defer rw.nextIDMu.Unlock()
	rw.nextID++
	return fmt.Sprintf("req_%d", rw.nextID)
}

func (rw *responseWriter) addPendingRequest(id string, ch chan *pb.CliResponse) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.pendingRequests[id] = ch
}

func (rw *responseWriter) handleResponse(response *pb.CliResponse) {
	rw.mu.Lock()
	ch, exists := rw.pendingRequests[response.Id]
	if exists {
		delete(rw.pendingRequests, response.Id)
	}
	rw.mu.Unlock()
	
	if exists {
		select {
		case ch <- response:
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
	serverUpdates   chan *pb.ServerUpdate
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
		serverUpdates:  make(chan *pb.ServerUpdate, 256),
		done:           make(chan struct{}),
	}

	// Start background tasks to handle WebSocket communication
	go transport.readLoop()
	go transport.pingLoop()

	return transport, nil
}

// Open opens a new session on the server.
func (w *WebSocketTransport) Open(ctx context.Context, request *pb.OpenRequest) (*pb.OpenResponse, error) {
	// Create protobuf CLI request
	req := &pb.CliRequest{
		Id: w.responseWriter.nextRequestID(),
		CliMessage: &pb.CliRequest_OpenSession{
			OpenSession: request,
		},
	}

	// Debug: Log the request being sent
	util.DebugLog("WebSocket sending Open request with session: %s", request.Name)
	util.DebugLog("Go client encrypted_zeros length: %d bytes", len(request.EncryptedZeros))

	response, err := w.sendRequestWithResponse(ctx, req, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("WebSocket open request failed: %w", err)
	}

	// Handle protobuf response format using type assertion
	switch resp := response.CliResponseMessage.(type) {
	case *pb.CliResponse_OpenSession:
		openResp := resp.OpenSession
		util.DebugLog("WebSocket Open response: Name=%s, Token=%s, URL=%s", 
			openResp.Name, openResp.Token, openResp.Url)
		util.DebugLog("WebSocket session validation - Server returned session name: %s", openResp.Name)
		return openResp, nil

	case *pb.CliResponse_Error:
		return nil, fmt.Errorf("server error: %s", resp.Error)

	default:
		return nil, fmt.Errorf("unexpected response type for open request")
	}
}

// Channel establishes a bidirectional streaming channel for real-time communication.
func (w *WebSocketTransport) Channel(ctx context.Context) (chan *pb.ServerUpdate, chan *pb.ClientUpdate, error) {
	// Create channels for this streaming session
	serverChan := make(chan *pb.ServerUpdate, 256)
	clientChan := make(chan *pb.ClientUpdate, 256)
	
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
		req := &pb.CliRequest{
			Id: w.responseWriter.nextRequestID(),
			CliMessage: &pb.CliRequest_StartChannel{
				StartChannel: &pb.ChannelStartRequest{
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
		switch response.CliResponseMessage.(type) {
		case *pb.CliResponse_StartChannel:
			util.DebugLog("WebSocket channel started successfully")
		case *pb.CliResponse_Error:
			log.Printf("Server error starting channel: %s", response.GetError())
			return
		default:
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
				
				// Skip if no cli message was created
				if cliMsg == nil {
					continue
				}
				
				// Create streaming request - these don't get individual responses
				requestID := fmt.Sprintf("stream_%d", time.Now().UnixNano())
				// Convert interface{} to the right protobuf oneof type
				var cliMessage interface{}
				if cliMsg != nil {
					cliMessage = cliMsg
				}
				
				// Type assert to the correct protobuf oneof interface
				req := &pb.CliRequest{
					Id: requestID,
				}
				
				// Set the cli message field based on type
				switch msg := cliMessage.(type) {
				case *pb.CliRequest_TerminalData:
					req.CliMessage = msg
				case *pb.CliRequest_CreatedShell:
					req.CliMessage = msg
				case *pb.CliRequest_ClosedShell:
					req.CliMessage = msg
				case *pb.CliRequest_Pong:
					req.CliMessage = msg
				case *pb.CliRequest_Error:
					req.CliMessage = msg
				default:
					continue // Skip unsupported message types
				}
				
				// Serialize to protobuf binary
				data, err := proto.Marshal(req)
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
				err = w.conn.WriteMessage(websocket.BinaryMessage, data)
				w.mu.Unlock()
				
				if err != nil {
					log.Printf("WebSocket failed to send outbound message #%d: %v", messageCount, err)
					return
				}
				util.DebugLog("WebSocket sent streaming message #%d (%d bytes)", messageCount, len(data))
				
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
func (w *WebSocketTransport) Close(ctx context.Context, request *pb.CloseRequest) error {
	req := &pb.CliRequest{
		Id: w.responseWriter.nextRequestID(),
		CliMessage: &pb.CliRequest_CloseSession{
			CloseSession: request,
		},
	}

	response, err := w.sendRequestWithResponse(ctx, req, 30*time.Second)
	if err != nil {
		return fmt.Errorf("WebSocket close request failed: %w", err)
	}

	// Handle tagged union response format
	switch response.CliResponseMessage.(type) {
	case *pb.CliResponse_CloseSession:
		return nil
	case *pb.CliResponse_Error:
		return fmt.Errorf("server error: %s", response.GetError())
	default:
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
func (w *WebSocketTransport) sendRequestWithResponse(ctx context.Context, req *pb.CliRequest, timeout time.Duration) (*pb.CliResponse, error) {
	responseCh := make(chan *pb.CliResponse, 1)
	w.responseWriter.addPendingRequest(req.Id, responseCh)

	// Send the request as binary protobuf
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return nil, fmt.Errorf("transport is closed")
	}

	// Marshal protobuf to binary
	data, err := proto.Marshal(req)
	if err != nil {
		w.mu.RUnlock()
		w.responseWriter.removePendingRequest(req.Id)
		return nil, fmt.Errorf("failed to marshal protobuf request: %w", err)
	}

	// Send binary message
	if err := w.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		w.mu.RUnlock()
		w.responseWriter.removePendingRequest(req.Id)
		return nil, fmt.Errorf("failed to send binary request: %w", err)
	}
	w.mu.RUnlock()

	// Wait for response with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case response := <-responseCh:
		return response, nil
	case <-timeoutCtx.Done():
		w.responseWriter.removePendingRequest(req.Id)
		return nil, fmt.Errorf("request timed out")
	case <-w.done:
		return nil, fmt.Errorf("transport closed")
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
	// Try to parse as a protobuf CliResponse first
	var cliResponse pb.CliResponse
	if err := proto.Unmarshal(message, &cliResponse); err == nil && cliResponse.Id != "" {
		util.DebugLog("Successfully parsed CliResponse with ID: %s", cliResponse.Id)
		// Handle streaming messages (sent with "server_update" ID) - matches Rust implementation
		if cliResponse.Id == "server_update" {
			util.DebugLog("WebSocket received server_update message: %+v", cliResponse.CliResponseMessage)
			// Convert the CliResponse to ServerUpdate directly - with protobuf, the types are compatible
			if cliResponse.CliResponseMessage == nil {
				return fmt.Errorf("received server_update with no response message")
			}
			
			serverUpdate, err := CliResponseToServerUpdate(cliResponse.CliResponseMessage)
			if err != nil {
				log.Printf("Failed to convert server_update to ServerUpdate: %v, message: %+v", err, cliResponse.CliResponseMessage)
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
		w.responseWriter.handleResponse(&cliResponse)
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

// isEmptyCliMessage checks if a CliMessage has no fields set (protobuf version)
func isEmptyCliMessage(msg interface{}) bool {
	return msg == nil
}

// ClientUpdateToCliMessage converts a pb.ClientUpdate to a protobuf CliMessage.
func ClientUpdateToCliMessage(update *pb.ClientUpdate) (interface{}, error) {
	if update == nil {
		return nil, fmt.Errorf("nil client update")
	}
	
	// Handle heartbeat messages (empty ClientUpdate with no ClientMessage)
	if update.ClientMessage == nil {
		// Skip heartbeat messages - they don't need to be sent over WebSocket
		return nil, nil
	}

	switch msg := update.ClientMessage.(type) {
	case *pb.ClientUpdate_Hello:
		// Hello is handled separately in handleOutgoingClientUpdate
		return nil, nil
	case *pb.ClientUpdate_Data:
		return &pb.CliRequest_TerminalData{
			TerminalData: msg.Data,
		}, nil
	case *pb.ClientUpdate_CreatedShell:
		return &pb.CliRequest_CreatedShell{
			CreatedShell: msg.CreatedShell,
		}, nil
	case *pb.ClientUpdate_ClosedShell:
		return &pb.CliRequest_ClosedShell{
			ClosedShell: msg.ClosedShell,
		}, nil
	case *pb.ClientUpdate_Pong:
		return &pb.CliRequest_Pong{
			Pong: msg.Pong,
		}, nil
	case *pb.ClientUpdate_Error:
		return &pb.CliRequest_Error{
			Error: msg.Error,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported client message type: %T", msg)
	}
}

// CliResponseToServerUpdate converts a protobuf CliResponseMessage to a pb.ServerUpdate.
func CliResponseToServerUpdate(cliMsg interface{}) (*pb.ServerUpdate, error) {
	if cliMsg == nil {
		return nil, fmt.Errorf("nil CLI response message")
	}

	// Use type switch to handle protobuf oneof interface
	switch msg := cliMsg.(type) {
	case *pb.CliResponse_TerminalInput:
		return &pb.ServerUpdate{
			ServerMessage: &pb.ServerUpdate_Input{
				Input: msg.TerminalInput,
			},
		}, nil
	case *pb.CliResponse_CreateShell:
		return &pb.ServerUpdate{
			ServerMessage: &pb.ServerUpdate_CreateShell{
				CreateShell: msg.CreateShell,
			},
		}, nil
	case *pb.CliResponse_CloseShell:
		return &pb.ServerUpdate{
			ServerMessage: &pb.ServerUpdate_CloseShell{
				CloseShell: msg.CloseShell,
			},
		}, nil
	case *pb.CliResponse_Sync:
		return &pb.ServerUpdate{
			ServerMessage: &pb.ServerUpdate_Sync{
				Sync: msg.Sync,
			},
		}, nil
	case *pb.CliResponse_Resize:
		return &pb.ServerUpdate{
			ServerMessage: &pb.ServerUpdate_Resize{
				Resize: msg.Resize,
			},
		}, nil
	case *pb.CliResponse_Ping:
		return &pb.ServerUpdate{
			ServerMessage: &pb.ServerUpdate_Ping{
				Ping: msg.Ping,
			},
		}, nil
	case *pb.CliResponse_Error:
		return &pb.ServerUpdate{
			ServerMessage: &pb.ServerUpdate_Error{
				Error: msg.Error,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported CLI response message type: %T", msg)
	}
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
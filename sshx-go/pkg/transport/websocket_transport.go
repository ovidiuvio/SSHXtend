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
)

// WebSocketTransport implements the SshxTransport interface using WebSocket communication.
type WebSocketTransport struct {
	conn            *websocket.Conn
	responseWriter  *responseWriter
	serverUpdates   chan *proto.ServerUpdate
	clientUpdates   chan *proto.ClientUpdate
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

	transport := &WebSocketTransport{
		conn:           conn,
		responseWriter: newResponseWriter(),
		serverUpdates:  make(chan *proto.ServerUpdate, 256),
		clientUpdates:  make(chan *proto.ClientUpdate, 256),
		done:           make(chan struct{}),
	}

	// Start background tasks to handle WebSocket communication
	go transport.readLoop()
	go transport.writeLoop()

	return transport, nil
}

// Open opens a new session on the server.
func (w *WebSocketTransport) Open(ctx context.Context, request *proto.OpenRequest) (*proto.OpenResponse, error) {
	req := CliRequest{
		ID: w.responseWriter.nextRequestID(),
		Message: CliMessage{
			Type: "openSession",
			Data: OpenSessionRequest{
				Origin:            request.Origin,
				EncryptedZeros:    request.EncryptedZeros,
				Name:              request.Name,
				WritePasswordHash: request.WritePasswordHash,
			},
		},
	}

	response, err := w.sendRequestWithResponse(ctx, req, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("WebSocket open request failed: %w", err)
	}

	switch response.Type {
	case "openSession":
		data, ok := response.Data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid openSession response data")
		}

		name, _ := data["name"].(string)
		token, _ := data["token"].(string)
		url, _ := data["url"].(string)

		return &proto.OpenResponse{
			Name:  name,
			Token: token,
			Url:   url,
		}, nil

	case "error":
		data, ok := response.Data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid error response data")
		}
		message, _ := data["message"].(string)
		return nil, fmt.Errorf("server error: %s", message)

	default:
		return nil, fmt.Errorf("unexpected response type for open request: %s", response.Type)
	}
}

// Channel establishes a bidirectional streaming channel for real-time communication.
func (w *WebSocketTransport) Channel(ctx context.Context) (chan *proto.ServerUpdate, chan *proto.ClientUpdate, error) {
	// For WebSocket, we return the existing channels since they're already set up in the constructor
	return w.serverUpdates, w.clientUpdates, nil
}

// Close closes an existing session on the server.
func (w *WebSocketTransport) Close(ctx context.Context, request *proto.CloseRequest) error {
	req := CliRequest{
		ID: w.responseWriter.nextRequestID(),
		Message: CliMessage{
			Type: "closeSession",
			Data: CloseSessionRequest{
				Name:  request.Name,
				Token: request.Token,
			},
		},
	}

	response, err := w.sendRequestWithResponse(ctx, req, 30*time.Second)
	if err != nil {
		return fmt.Errorf("WebSocket close request failed: %w", err)
	}

	switch response.Type {
	case "closeSession":
		return nil
	case "error":
		data, ok := response.Data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid error response data")
		}
		message, _ := data["message"].(string)
		return fmt.Errorf("server error: %s", message)
	default:
		return fmt.Errorf("unexpected response type for close request: %s", response.Type)
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

	close(w.done)
	
	// Close the WebSocket connection
	err := w.conn.Close()
	
	// Close channels
	close(w.serverUpdates)
	close(w.clientUpdates)

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
	defer close(w.serverUpdates)

	for {
		select {
		case <-w.done:
			return
		default:
		}

		_, message, err := w.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		if err := w.handleIncomingMessage(message); err != nil {
			log.Printf("Error handling WebSocket message: %v", err)
		}
	}
}

// writeLoop handles outgoing client messages.
func (w *WebSocketTransport) writeLoop() {
	for {
		select {
		case update, ok := <-w.clientUpdates:
			if !ok {
				return // Channel closed
			}

			if err := w.handleOutgoingClientUpdate(update); err != nil {
				log.Printf("Error handling outgoing client update: %v", err)
				continue
			}

		case <-w.done:
			return
		}
	}
}

// handleIncomingMessage processes incoming WebSocket messages.
func (w *WebSocketTransport) handleIncomingMessage(message []byte) error {
	// Try to parse as a correlated response first
	var cliResponse CliResponse
	if err := json.Unmarshal(message, &cliResponse); err == nil && cliResponse.ID != "" {
		w.responseWriter.handleResponse(cliResponse)
		return nil
	}

	// Try to parse as a streaming message (no correlation ID)
	var streamingMsg CliResponseMessage
	if err := json.Unmarshal(message, &streamingMsg); err == nil {
		serverUpdate, err := CliResponseToServerUpdate(streamingMsg)
		if err != nil {
			return fmt.Errorf("failed to convert CLI response to server update: %w", err)
		}

		select {
		case w.serverUpdates <- serverUpdate:
		case <-w.done:
		}
		return nil
	}

	return fmt.Errorf("failed to parse incoming message")
}

// handleOutgoingClientUpdate converts and sends client updates.
func (w *WebSocketTransport) handleOutgoingClientUpdate(update *proto.ClientUpdate) error {
	// Handle special case for Hello message (start channel)
	if hello := update.GetHello(); hello != "" {
		// Parse "name,token" format
		parts := strings.Split(hello, ",")
		if len(parts) != 2 {
			return fmt.Errorf("invalid hello format")
		}

		req := CliRequest{
			ID: w.responseWriter.nextRequestID(),
			Message: CliMessage{
				Type: "startChannel",
				Data: StartChannelRequest{
					Name:  parts[0],
					Token: parts[1],
				},
			},
		}

		w.mu.RLock()
		defer w.mu.RUnlock()
		if w.closed {
			return fmt.Errorf("transport is closed")
		}
		
		// Send start channel request and wait for response
		return w.conn.WriteJSON(req)
	}

	// For other messages, convert and send as streaming messages (no correlation)
	cliMsg, err := ClientUpdateToCliMessage(update)
	if err != nil {
		return fmt.Errorf("failed to convert client update: %w", err)
	}

	// Send as streaming message without correlation ID
	streamingMsg := StreamingMessage{
		Type: cliMsg.Type,
		Data: cliMsg.Data,
	}

	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.closed {
		return fmt.Errorf("transport is closed")
	}

	return w.conn.WriteJSON(streamingMsg)
}
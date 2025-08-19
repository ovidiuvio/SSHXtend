// Package client provides the core client functionality with transport abstraction.
package client

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"sshx-go/pkg/encrypt"
	"sshx-go/pkg/proto"
	"sshx-go/pkg/transport"
	"sshx-go/pkg/util"
)

const (
	heartbeatInterval = 2 * time.Second
	reconnectInterval = 60 * time.Second
)

// ControllerConfig holds configuration for creating a controller.
type ControllerConfig struct {
	Origin        string
	Name          string
	Runner        Runner
	EnableReaders bool
}

// Controller handles a single session's communication with the remote server using transport abstraction.
type Controller struct {
	transport     transport.SshxTransport
	config        ControllerConfig
	encrypt       *encrypt.Encrypt
	encryptionKey string

	name     string
	token    string
	url      string
	writeURL *string

	// Channels with backpressure routing messages to each shell task
	shellsTx map[uint32]chan ShellData
	shellsMu sync.RWMutex

	// Channel shared with tasks to allow them to output client messages
	outputTx chan ClientMessage
	outputRx chan ClientMessage

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Connection method used
	connectionMethod transport.ConnectionMethod
}

// NewController constructs a new controller using transport abstraction, connecting to the remote server.
// This version automatically tries gRPC first, then falls back to WebSocket if gRPC fails.
func NewController(config ControllerConfig) (*Controller, error) {
	return NewControllerWithConnection(config, transport.DefaultConnectionConfig())
}

// NewControllerWithConnection constructs a new controller with custom connection configuration.
func NewControllerWithConnection(config ControllerConfig, connConfig transport.ConnectionConfig) (*Controller, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Generate encryption key - matches Rust implementation
	encryptionKey := randAlphanumeric(14) // 83.3 bits of entropy

	// Create encryptor in background task (matches Rust spawn_blocking)
	encryptor := encrypt.New(encryptionKey)

	var writePassword *string
	var writePasswordHash []byte
	if config.EnableReaders {
		writePasswordVal := randAlphanumeric(14) // 83.3 bits of entropy
		writePassword = &writePasswordVal
		writeEncrypt := encrypt.New(writePasswordVal)
		writePasswordHash = writeEncrypt.Zeros()
	}

	// Connect to server with fallback
	connectionResult, err := transport.ConnectWithFallback(config.Origin, config.Name, connConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	log.Printf("Connected to %s using %s transport", config.Origin, connectionResult.Method)

	// Open session - matches Rust OpenRequest exactly
	openReq := &proto.OpenRequest{
		Origin:            config.Origin,
		EncryptedZeros:    encryptor.Zeros(),
		Name:              config.Name,
		WritePasswordHash: writePasswordHash,
	}

	resp, err := connectionResult.Transport.Open(ctx, openReq)
	if err != nil {
		cancel()
		connectionResult.Transport.Cleanup()
		return nil, fmt.Errorf("failed to open session: %w", err)
	}

	// Build URLs exactly like Rust implementation
	url := resp.Url + "#" + encryptionKey
	var writeURL *string
	if writePassword != nil {
		writeURLVal := url + "," + *writePassword
		writeURL = &writeURLVal
	}

	// Create channels with same buffer sizes as Rust
	outputTx := make(chan ClientMessage, 64)
	outputRx := make(chan ClientMessage, 64)

	controller := &Controller{
		transport:        connectionResult.Transport,
		config:           config,
		encrypt:          encryptor,
		encryptionKey:    encryptionKey,
		name:             resp.Name,
		token:            resp.Token,
		url:              url,
		writeURL:         writeURL,
		shellsTx:         make(map[uint32]chan ShellData),
		outputTx:         outputTx,
		outputRx:         outputRx,
		ctx:              ctx,
		cancel:           cancel,
		connectionMethod: connectionResult.Method,
	}

	return controller, nil
}

// Name returns the name of the session.
func (c *Controller) Name() string {
	return c.name
}

// URL returns the URL of the session.
func (c *Controller) URL() string {
	return c.url
}

// WriteURL returns the write URL of the session, if it exists.
func (c *Controller) WriteURL() *string {
	return c.writeURL
}

// EncryptionKey returns the encryption key for this session.
func (c *Controller) EncryptionKey() string {
	return c.encryptionKey
}

// ConnectionMethod returns the connection method used.
func (c *Controller) ConnectionMethod() transport.ConnectionMethod {
	return c.connectionMethod
}

// Run runs the controller forever, listening for requests from the server.
// This matches the Rust Controller::run method exactly.
func (c *Controller) Run() error {
	lastRetry := time.Now()
	retries := 0

	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}

		if err := c.tryChannel(); err != nil {
			if time.Since(lastRetry) >= 10*time.Second {
				retries = 0
			}
			secs := 1 << min(retries, 4) // Exponential backoff, max 16 seconds
			log.Printf("disconnected, retrying in %ds: %v", secs, err)

			select {
			case <-time.After(time.Duration(secs) * time.Second):
			case <-c.ctx.Done():
				return c.ctx.Err()
			}
			retries++
		}
		lastRetry = time.Now()
	}
}

// tryChannel helper function used by Run() that can return errors.
// This matches the Rust Controller::try_channel method exactly.
func (c *Controller) tryChannel() error {
	// For WebSocket connections, we need to recreate the transport on each attempt
	// since WebSocket connections can't be reused after failure
	if c.connectionMethod == transport.MethodWebSocketFallback {
		// Cleanup old transport
		c.transport.Cleanup()

		// Reconnect using the specific transport type that worked initially
		wsURL := transport.GrpcToWebSocketURL(c.config.Origin, c.config.Name)
		util.DebugLog("Reconnecting via WebSocket (remembered preference): %s", wsURL)
		newTransport, err := transport.ConnectWebSocket(wsURL)
		if err != nil {
			return fmt.Errorf("failed to reconnect via WebSocket: %w", err)
		}
		c.transport = newTransport
	}

	// Get bidirectional channels from transport
	serverUpdates, clientUpdates, err := c.transport.Channel(c.ctx)
	if err != nil {
		return fmt.Errorf("failed to create channel: %w", err)
	}

	// Send hello message first - matches Rust implementation
	hello := fmt.Sprintf("%s,%s", c.name, c.token)
	helloMsg := &proto.ClientUpdate{
		ClientMessage: &proto.ClientUpdate_Hello{Hello: hello},
	}

	select {
	case clientUpdates <- helloMsg:
	case <-c.ctx.Done():
		return c.ctx.Err()
	}

	// Main loop - matches Rust tokio::select! exactly
	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	reconnectTimer := time.NewTimer(reconnectInterval)
	defer reconnectTimer.Stop()

	for {
		select {
		case <-heartbeat.C:
			// Send heartbeat - matches Rust interval.tick()
			select {
			case clientUpdates <- &proto.ClientUpdate{}:
			case <-c.ctx.Done():
				return c.ctx.Err()
			}

		case msg := <-c.outputRx:
			// Send client message - matches Rust output_rx.recv()
			update := c.clientMessageToUpdate(msg)
			select {
			case clientUpdates <- update:
			case <-c.ctx.Done():
				return c.ctx.Err()
			}

		case resp, ok := <-serverUpdates:
			// Receive server message - matches Rust messages.next()
			if !ok {
				return fmt.Errorf("server updates channel closed")
			}
			if err := c.handleServerMessage(resp); err != nil {
				log.Printf("error handling server message: %v", err)
			}

		case <-reconnectTimer.C:
			// Force reconnection - matches Rust reconnect timer
			return nil

		case <-c.ctx.Done():
			return c.ctx.Err()
		}
	}
}

// handleServerMessage processes a message received from the server.
// This matches the Rust message handling logic exactly.
func (c *Controller) handleServerMessage(msg *proto.ServerUpdate) error {
	switch serverMsg := msg.ServerMessage.(type) {
	case *proto.ServerUpdate_Input:
		// Decrypt input data - matches Rust implementation exactly
		util.DebugLog("CONTROLLER[%s]: Received Input - id=%d, offset=%d, encrypted_len=%d, encrypted_data=%v", 
			c.transport.ConnectionType(), serverMsg.Input.Id, serverMsg.Input.Offset, 
			len(serverMsg.Input.Data), serverMsg.Input.Data)
		
		data := c.encrypt.Segment(0x200000000, serverMsg.Input.Offset, serverMsg.Input.Data)
		
		util.DebugLog("CONTROLLER[%s]: Decrypted Input - id=%d, decrypted_len=%d, decrypted_data=%q, raw=%v", 
			c.transport.ConnectionType(), serverMsg.Input.Id, len(data), string(data), data)
		
		c.shellsMu.RLock()
		if sender, ok := c.shellsTx[serverMsg.Input.Id]; ok {
			select {
			case sender <- ShellData{Type: ShellDataTypeData, Data: data}:
				util.DebugLog("CONTROLLER[%s]: Sent data to shell %d", c.transport.ConnectionType(), serverMsg.Input.Id)
			default:
				log.Printf("shell %d channel full, dropping input", serverMsg.Input.Id)
			}
		} else {
			log.Printf("received data for non-existing shell %d", serverMsg.Input.Id)
		}
		c.shellsMu.RUnlock()

	case *proto.ServerUpdate_CreateShell:
		id := serverMsg.CreateShell.Id
		center := [2]int32{serverMsg.CreateShell.X, serverMsg.CreateShell.Y}

		c.shellsMu.Lock()
		if _, exists := c.shellsTx[id]; !exists {
			c.spawnShellTask(id, center)
		} else {
			log.Printf("server asked to create duplicate shell %d", id)
		}
		c.shellsMu.Unlock()

	case *proto.ServerUpdate_CloseShell:
		id := serverMsg.CloseShell
		c.shellsMu.Lock()
		if ch, exists := c.shellsTx[id]; exists {
			close(ch)
			delete(c.shellsTx, id)
		}
		c.shellsMu.Unlock()

		// Send acknowledgment - matches Rust send_msg().await?
		select {
		case c.outputRx <- ClientMessage{
			Type:    ClientMessageTypeClosedShell,
			ShellID: id,
		}:
		case <-c.ctx.Done():
		}

	case *proto.ServerUpdate_Sync:
		for id, seq := range serverMsg.Sync.Map {
			c.shellsMu.RLock()
			if sender, ok := c.shellsTx[id]; ok {
				select {
				case sender <- ShellData{Type: ShellDataTypeSync, Seq: seq}:
				default:
					// Channel full, skip sync
				}
			} else {
				log.Printf("received sequence number for non-existing shell %d", id)
				// Send close acknowledgment for non-existing shell - matches Rust send_msg().await?
				select {
				case c.outputRx <- ClientMessage{
					Type:    ClientMessageTypeClosedShell,
					ShellID: id,
				}:
				case <-c.ctx.Done():
				}
			}
			c.shellsMu.RUnlock()
		}

	case *proto.ServerUpdate_Resize:
		c.shellsMu.RLock()
		if sender, ok := c.shellsTx[serverMsg.Resize.Id]; ok {
			select {
			case sender <- ShellData{
				Type: ShellDataTypeSize,
				Rows: serverMsg.Resize.Rows,
				Cols: serverMsg.Resize.Cols,
			}:
			default:
				// Channel full, skip resize
			}
		} else {
			log.Printf("received resize for non-existing shell %d", serverMsg.Resize.Id)
		}
		c.shellsMu.RUnlock()

	case *proto.ServerUpdate_Ping:
		// Echo back the timestamp for latency measurement
		// Block until send succeeds, matching Rust send_msg().await?
		select {
		case c.outputRx <- ClientMessage{
			Type: ClientMessageTypePong,
			Pong: serverMsg.Ping,
		}:
		case <-c.ctx.Done():
		}

	case *proto.ServerUpdate_Error:
		log.Printf("error received from server: %s", serverMsg.Error)
	}

	return nil
}

// spawnShellTask starts a new terminal task on the client.
// This matches the Rust Controller::spawn_shell_task method exactly.
func (c *Controller) spawnShellTask(id uint32, center [2]int32) {
	shellTx := make(chan ShellData, 16) // Same buffer size as Rust
	c.shellsTx[id] = shellTx

	go func() {
		defer func() {
			c.shellsMu.Lock()
			delete(c.shellsTx, id)
			c.shellsMu.Unlock()

			// Block until send succeeds, matching Rust output_tx.send().await.ok()
			select {
			case c.outputRx <- ClientMessage{
				Type:    ClientMessageTypeClosedShell,
				ShellID: id,
			}:
			case <-c.ctx.Done():
			}
		}()

		util.DebugLog("spawning new shell %d using %s transport", id, c.transport.ConnectionType())

		// Send shell creation acknowledgment - matches Rust NewShell exactly
		newShell := &proto.NewShell{
			Id: id,
			X:  center[0],
			Y:  center[1],
		}
		// Block until send succeeds, matching Rust output_tx.send().await
		select {
		case c.outputRx <- ClientMessage{
			Type:  ClientMessageTypeCreatedShell,
			Shell: newShell,
		}:
		case <-c.ctx.Done():
			return
		}

		// Run the shell
		if err := c.config.Runner.Run(c.ctx, id, c.encrypt, shellTx, c.outputRx); err != nil {
			if c.ctx.Err() == nil { // Only send error if not due to context cancellation
				errMsg := ClientMessage{
					Type:  ClientMessageTypeError,
					Error: fmt.Sprintf("shell %d: %v", id, err),
				}
				// Block until send succeeds, matching Rust output_tx.send().await.ok()
				select {
				case c.outputRx <- errMsg:
				case <-c.ctx.Done():
				}
			}
		}
	}()
}

// clientMessageToUpdate converts a ClientMessage to a ClientUpdate protobuf message.
func (c *Controller) clientMessageToUpdate(msg ClientMessage) *proto.ClientUpdate {
	switch msg.Type {
	case ClientMessageTypeHello:
		return &proto.ClientUpdate{
			ClientMessage: &proto.ClientUpdate_Hello{Hello: msg.Hello},
		}
	case ClientMessageTypeData:
		util.DebugLog("CONTROLLER[%s]: Sending outbound Data - id=%d, len=%d, data=%q, raw=%v, seq=%d", 
			c.transport.ConnectionType(), msg.Data.ID, len(msg.Data.Data), 
			string(msg.Data.Data), msg.Data.Data, msg.Data.Seq)
		
		return &proto.ClientUpdate{
			ClientMessage: &proto.ClientUpdate_Data{
				Data: &proto.TerminalData{
					Id:   msg.Data.ID,
					Data: msg.Data.Data,
					Seq:  msg.Data.Seq,
				},
			},
		}
	case ClientMessageTypeCreatedShell:
		return &proto.ClientUpdate{
			ClientMessage: &proto.ClientUpdate_CreatedShell{
				CreatedShell: msg.Shell,
			},
		}
	case ClientMessageTypeClosedShell:
		return &proto.ClientUpdate{
			ClientMessage: &proto.ClientUpdate_ClosedShell{ClosedShell: msg.ShellID},
		}
	case ClientMessageTypePong:
		return &proto.ClientUpdate{
			ClientMessage: &proto.ClientUpdate_Pong{Pong: msg.Pong},
		}
	case ClientMessageTypeError:
		return &proto.ClientUpdate{
			ClientMessage: &proto.ClientUpdate_Error{Error: msg.Error},
		}
	default:
		return &proto.ClientUpdate{}
	}
}

// Close terminates this session gracefully.
// This matches the Rust Controller::close method exactly.
func (c *Controller) Close() error {
	defer c.cancel()
	defer c.transport.Cleanup()

	req := &proto.CloseRequest{
		Name:  c.name,
		Token: c.token,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.transport.Close(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	return nil
}

// randAlphanumeric generates a cryptographically-secure, random alphanumeric value.
// This matches the Rust rand_alphanumeric function exactly.
func randAlphanumeric(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)

	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(fmt.Sprintf("failed to generate random number: %v", err))
		}
		result[i] = charset[num.Int64()]
	}

	return string(result)
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

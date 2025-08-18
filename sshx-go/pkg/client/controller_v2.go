// Package client provides the core client functionality with transport abstraction.
package client

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"sshx-go/pkg/encrypt"
	"sshx-go/pkg/proto"
	"sshx-go/pkg/transport"
)

// Note: heartbeatInterval and reconnectInterval are already defined in controller.go

// ControllerV2 handles a single session's communication with the remote server using transport abstraction.
// This is the new version that uses the transport abstraction layer.
type ControllerV2 struct {
	transport transport.SshxTransport
	config    ControllerConfig
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

// NewControllerV2 constructs a new controller using transport abstraction, connecting to the remote server.
// This version automatically tries gRPC first, then falls back to WebSocket if gRPC fails.
func NewControllerV2(config ControllerConfig) (*ControllerV2, error) {
	return NewControllerV2WithConnection(config, transport.DefaultConnectionConfig())
}

// NewControllerV2WithConnection constructs a new controller with custom connection configuration.
func NewControllerV2WithConnection(config ControllerConfig, connConfig transport.ConnectionConfig) (*ControllerV2, error) {
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

	controller := &ControllerV2{
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
func (c *ControllerV2) Name() string {
	return c.name
}

// URL returns the URL of the session.
func (c *ControllerV2) URL() string {
	return c.url
}

// WriteURL returns the write URL of the session, if it exists.
func (c *ControllerV2) WriteURL() *string {
	return c.writeURL
}

// EncryptionKey returns the encryption key for this session.
func (c *ControllerV2) EncryptionKey() string {
	return c.encryptionKey
}

// ConnectionMethod returns the connection method used.
func (c *ControllerV2) ConnectionMethod() transport.ConnectionMethod {
	return c.connectionMethod
}

// Run runs the controller forever, listening for requests from the server.
// This matches the Rust Controller::run method exactly.
func (c *ControllerV2) Run() error {
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
func (c *ControllerV2) tryChannel() error {
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
func (c *ControllerV2) handleServerMessage(msg *proto.ServerUpdate) error {
	switch serverMsg := msg.ServerMessage.(type) {
	case *proto.ServerUpdate_Input:
		// Decrypt input data - matches Rust implementation exactly
		data := c.encrypt.Segment(0x200000000, serverMsg.Input.Offset, serverMsg.Input.Data)
		c.shellsMu.RLock()
		if sender, ok := c.shellsTx[serverMsg.Input.Id]; ok {
			select {
			case sender <- ShellData{Type: ShellDataTypeData, Data: data}:
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
func (c *ControllerV2) spawnShellTask(id uint32, center [2]int32) {
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

		log.Printf("spawning new shell %d using %s transport", id, c.transport.ConnectionType())

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
func (c *ControllerV2) clientMessageToUpdate(msg ClientMessage) *proto.ClientUpdate {
	switch msg.Type {
	case ClientMessageTypeHello:
		return &proto.ClientUpdate{
			ClientMessage: &proto.ClientUpdate_Hello{Hello: msg.Hello},
		}
	case ClientMessageTypeData:
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
func (c *ControllerV2) Close() error {
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

// Note: randAlphanumeric and min are already defined in controller.go
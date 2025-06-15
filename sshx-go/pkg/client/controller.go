package client

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"sshx-go/pkg/encrypt"
	"sshx-go/pkg/proto"
)

const (
	heartbeatInterval = 2 * time.Second
	reconnectInterval = 60 * time.Second
)

// Controller handles a single session's communication with the remote server.
// This matches the Rust Controller struct exactly.
type Controller struct {
	origin        string
	runner        Runner
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
}

// ControllerConfig holds configuration for creating a controller.
type ControllerConfig struct {
	Origin        string
	Name          string
	Runner        Runner
	EnableReaders bool
	SessionID     *string
	Secret        *string
}

// NewController constructs a new controller, connecting to the remote server.
// This matches the Rust Controller::new method exactly.
func NewController(config ControllerConfig) (*Controller, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Generate encryption key - matches Rust implementation
	encryptionKey := ""
	if config.Secret != nil {
		encryptionKey = *config.Secret
	} else {
		encryptionKey = randAlphanumeric(14) // 83.3 bits of entropy
	}

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

	// Connect to server
	client, err := connectGRPC(config.Origin)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	// Open session - matches Rust OpenRequest exactly
	openReq := &proto.OpenRequest{
		Origin:             config.Origin,
		EncryptedZeros:     encryptor.Zeros(),
		Name:               config.Name,
		WritePasswordHash:  writePasswordHash,
		SessionId:          config.SessionID,
	}

	resp, err := client.Open(ctx, openReq)
	if err != nil {
		cancel()
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
		origin:        config.Origin,
		runner:        config.Runner,
		encrypt:       encryptor,
		encryptionKey: encryptionKey,
		name:          resp.Name,
		token:         resp.Token,
		url:           url,
		writeURL:      writeURL,
		shellsTx:      make(map[uint32]chan ShellData),
		outputTx:      outputTx,
		outputRx:      outputRx,
		ctx:           ctx,
		cancel:        cancel,
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
	// Create channel to send client updates - matches Rust mpsc::channel(16)
	sendCh := make(chan *proto.ClientUpdate, 16)
	
	// Send hello message first - matches Rust implementation
	hello := fmt.Sprintf("%s,%s", c.name, c.token)
	helloMsg := &proto.ClientUpdate{
		ClientMessage: &proto.ClientUpdate_Hello{Hello: hello},
	}
	
	select {
	case sendCh <- helloMsg:
	case <-c.ctx.Done():
		return c.ctx.Err()
	}

	// Connect to server
	target := parseGRPCTarget(c.origin)
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	client := proto.NewSshxServiceClient(conn)
	stream, err := client.Channel(c.ctx)
	if err != nil {
		return fmt.Errorf("failed to create channel: %w", err)
	}

	// Start sending goroutine that drains sendCh
	go func() {
		for msg := range sendCh {
			if err := stream.Send(msg); err != nil {
				log.Printf("failed to send message: %v", err)
				return
			}
		}
	}()
	defer close(sendCh)

	// Main loop - matches Rust tokio::select! exactly
	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()
	
	reconnectTimer := time.NewTimer(reconnectInterval)
	defer reconnectTimer.Stop()

	// Start receiving goroutine to put messages in a channel
	recvCh := make(chan *proto.ServerUpdate, 16)
	recvErr := make(chan error, 1)
	go func() {
		defer close(recvCh)
		defer close(recvErr)
		for {
			resp, err := stream.Recv()
			if err != nil {
				recvErr <- err
				return
			}
			select {
			case recvCh <- resp:
			case <-c.ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-heartbeat.C:
			// Send heartbeat - matches Rust interval.tick()
			// Block until send succeeds, matching Rust tx.send().await?
			select {
			case sendCh <- &proto.ClientUpdate{}:
			case <-c.ctx.Done():
				return c.ctx.Err()
			}
			
		case msg := <-c.outputRx:
			// Send client message - matches Rust output_rx.recv()
			// Block until send succeeds, matching Rust send_msg().await?
			update := c.clientMessageToUpdate(msg)
			select {
			case sendCh <- update:
			case <-c.ctx.Done():
				return c.ctx.Err()
			}
			
		case resp, ok := <-recvCh:
			// Receive server message - matches Rust messages.next()
			if !ok {
				return fmt.Errorf("receive channel closed")
			}
			if err := c.handleServerMessage(resp); err != nil {
				log.Printf("error handling server message: %v", err)
			}
			
		case err := <-recvErr:
			// Receive goroutine failed
			if err == io.EOF {
				return fmt.Errorf("server closed connection")
			}
			return fmt.Errorf("receive error: %w", err)
			
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

		log.Printf("spawning new shell %d", id)

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
		if err := c.runner.Run(c.ctx, id, c.encrypt, shellTx, c.outputRx); err != nil {
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

	target := parseGRPCTarget(c.origin)
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect for close: %w", err)
	}
	defer conn.Close()

	client := proto.NewSshxServiceClient(conn)
	req := &proto.CloseRequest{
		Name:  c.name,
		Token: c.token,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Close(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	return nil
}

// connectGRPC creates a new gRPC client connection.
func connectGRPC(origin string) (proto.SshxServiceClient, error) {
	// Parse the origin URL to extract host:port for gRPC
	target := parseGRPCTarget(origin)
	
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return proto.NewSshxServiceClient(conn), nil
}

// parseGRPCTarget extracts the host:port from a URL for gRPC dialing
func parseGRPCTarget(origin string) string {
	// Remove protocol prefix if present
	if strings.HasPrefix(origin, "http://") {
		origin = origin[7:]
	} else if strings.HasPrefix(origin, "https://") {
		origin = origin[8:]
	}
	
	// Remove any path component
	if idx := strings.Index(origin, "/"); idx != -1 {
		origin = origin[:idx]
	}
	
	// If no port is specified, add default port
	if !strings.Contains(origin, ":") {
		// Default to port 8051 for local development, 443 for HTTPS, 80 for HTTP
		if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
			origin += ":8051"
		} else {
			origin += ":443" // Assume HTTPS for external servers
		}
	}
	
	return origin
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
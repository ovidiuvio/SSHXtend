// Package transport provides connection management with automatic gRPC→WebSocket fallback.
//
// This module provides high-level connection management that automatically
// attempts gRPC first, then falls back to WebSocket if gRPC fails.
package transport

import (
	"context"
	"fmt"
	"log"
	"time"

	"sshx-go/pkg/proto"
)

const (
	// DefaultGrpcTimeout is the default timeout for gRPC connectivity test.
	DefaultGrpcTimeout = 3 * time.Second
	// DefaultWebSocketTimeout is the default timeout for WebSocket connection.
	DefaultWebSocketTimeout = 5 * time.Second
)

// ConnectWithFallback connects to an sshx server with automatic gRPC→WebSocket fallback.
//
// This function attempts to connect using gRPC first, and if that fails,
// automatically falls back to WebSocket. The connection method is determined
// by testing actual connectivity to the server.
//
// Behavior:
// 1. Attempts gRPC connection with 3-second timeout
// 2. Tests gRPC connectivity by making an actual Open call
// 3. If gRPC fails, converts URL and attempts WebSocket connection
// 4. Returns the first successful connection method
//
// Arguments:
//   - origin: The server URL to connect to (e.g., "https://sshx.io")
//   - sessionName: Name for the session (used for WebSocket endpoint)
//   - config: Connection configuration options
//
// Returns:
//   - ConnectionResult containing the transport and connection method used
func ConnectWithFallback(origin, sessionName string, config ConnectionConfig) (*ConnectionResult, error) {
	log.Printf("Attempting connection to %s with fallback for session %s", origin, sessionName)

	// Apply default timeouts if not specified
	if config.GrpcTimeout == 0 {
		config.GrpcTimeout = DefaultGrpcTimeout
	}
	if config.WebSocketTimeout == 0 {
		config.WebSocketTimeout = DefaultWebSocketTimeout
	}

	// First, try gRPC connection
	if transport, err := tryGrpcConnection(origin, config); err == nil {
		if config.VerboseErrors {
			log.Printf("✓ gRPC connection successful to %s", origin)
		}
		return &ConnectionResult{
			Transport: transport,
			Method:    MethodGrpc,
		}, nil
	} else {
		if config.VerboseErrors {
			log.Printf("⚠ gRPC connection failed to %s: %v, attempting WebSocket fallback", origin, err)
		} else {
			log.Printf("gRPC connection failed, attempting WebSocket fallback: %v", err)
		}
	}

	// If gRPC failed, try WebSocket fallback
	if transport, err := tryWebSocketConnection(origin, sessionName, config); err == nil {
		if config.VerboseErrors {
			log.Printf("✓ WebSocket fallback connection successful to %s", origin)
		}
		return &ConnectionResult{
			Transport: transport,
			Method:    MethodWebSocketFallback,
		}, nil
	} else {
		if config.VerboseErrors {
			log.Printf("✗ WebSocket fallback also failed to %s: %v", origin, err)
		}
		return nil, fmt.Errorf("both gRPC and WebSocket connections failed for %s: %w", origin, err)
	}
}

// tryGrpcConnection attempts to establish a gRPC connection and test its connectivity.
//
// This function not only connects to the gRPC endpoint but also performs
// a real connectivity test by attempting an Open call to ensure the
// connection is actually working.
func tryGrpcConnection(origin string, config ConnectionConfig) (SshxTransport, error) {
	log.Printf("Attempting gRPC connection to %s (timeout: %v)", origin, config.GrpcTimeout)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.GrpcTimeout)
	defer cancel()

	// Attempt to connect
	transport, err := ConnectGrpc(origin)
	if err != nil {
		return nil, fmt.Errorf("gRPC connection failed: %w", err)
	}

	// Test connectivity with a dummy Open request
	// This ensures the connection is actually working, not just established
	log.Printf("Testing gRPC connectivity to %s with Open call", origin)
	testRequest := &proto.OpenRequest{
		Origin:         origin,
		EncryptedZeros: make([]byte, 32), // Dummy encrypted zeros for connectivity test
		Name:           "connectivity-test",
	}

	// We expect this to either succeed or fail with a meaningful error
	// Either way, it proves the gRPC connection is working
	_, err = transport.Open(ctx, testRequest)
	if err != nil {
		transport.Cleanup()
		return nil, fmt.Errorf("gRPC connectivity test failed: %w", err)
	}

	log.Printf("gRPC connectivity test succeeded for %s", origin)
	return transport, nil
}

// tryWebSocketConnection attempts to establish a WebSocket connection.
func tryWebSocketConnection(origin, sessionName string, config ConnectionConfig) (SshxTransport, error) {
	wsURL := GrpcToWebSocketURL(origin, sessionName)
	log.Printf("Attempting WebSocket connection to %s (timeout: %v)", wsURL, config.WebSocketTimeout)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.WebSocketTimeout)
	defer cancel()

	// Create a channel to receive the result
	result := make(chan struct {
		transport SshxTransport
		err       error
	}, 1)

	go func() {
		transport, err := ConnectWebSocket(wsURL)
		result <- struct {
			transport SshxTransport
			err       error
		}{transport, err}
	}()

	select {
	case res := <-result:
		if res.err != nil {
			return nil, fmt.Errorf("WebSocket connection failed: %w", res.err)
		}
		return res.transport, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("WebSocket connection timed out after %v", config.WebSocketTimeout)
	}
}

// TestConnectivity tests gRPC connectivity to a server without establishing a full connection.
//
// This is a lightweight function for testing if gRPC is available without
// the overhead of creating a full transport connection.
//
// Arguments:
//   - origin: The server URL to test
//   - timeoutDuration: Maximum time to wait for the test
//
// Returns:
//   - true if gRPC connectivity is available, false otherwise
func TestConnectivity(origin string, timeoutDuration time.Duration) bool {
	log.Printf("Testing gRPC connectivity to %s", origin)
	
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()
	
	transport, err := ConnectGrpc(origin)
	if err != nil {
		log.Printf("gRPC connectivity test failed: %v", err)
		return false
	}
	defer transport.Cleanup()
	
	// Test with a dummy Open request to verify actual connectivity
	testRequest := &proto.OpenRequest{
		Origin:         origin,
		EncryptedZeros: make([]byte, 32), // Dummy encrypted zeros for connectivity test
		Name:           "connectivity-test",
	}
	
	_, err = transport.Open(ctx, testRequest)
	if err != nil {
		log.Printf("gRPC connectivity test failed on Open call: %v", err)
		return false
	}
	
	log.Printf("gRPC connectivity test succeeded for %s", origin)
	return true
}

// VerboseConfig creates a connection configuration for verbose error reporting.
//
// This is useful for debugging connection issues or when you want
// to show detailed error information to users.
func VerboseConfig() ConnectionConfig {
	return ConnectionConfig{
		VerboseErrors:    true,
		GrpcTimeout:     DefaultGrpcTimeout,
		WebSocketTimeout: DefaultWebSocketTimeout,
	}
}

// CustomTimeoutConfig creates a connection configuration with custom timeouts.
//
// Arguments:
//   - grpcTimeout: Timeout for gRPC connection attempts
//   - websocketTimeout: Timeout for WebSocket connection attempts
func CustomTimeoutConfig(grpcTimeout, websocketTimeout time.Duration) ConnectionConfig {
	return ConnectionConfig{
		VerboseErrors:    false,
		GrpcTimeout:     grpcTimeout,
		WebSocketTimeout: websocketTimeout,
	}
}

// QuickConnectGrpc is a convenience function for connecting via gRPC only.
func QuickConnectGrpc(origin string) (SshxTransport, error) {
	return ConnectGrpc(origin)
}

// QuickConnectWebSocket is a convenience function for connecting via WebSocket only.
func QuickConnectWebSocket(origin, sessionName string) (SshxTransport, error) {
	wsURL := GrpcToWebSocketURL(origin, sessionName)
	return ConnectWebSocket(wsURL)
}
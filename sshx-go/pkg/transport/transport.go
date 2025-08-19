// Package transport provides a unified interface for connecting to sshx servers
// via either gRPC or WebSocket protocols, with automatic fallback capability.
package transport

import (
	"context"
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

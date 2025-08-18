package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"sshx-go/pkg/proto"
)

// GrpcTransport wraps the existing gRPC client implementation.
type GrpcTransport struct {
	client proto.SshxServiceClient
	conn   *grpc.ClientConn
}

// NewGrpcTransport creates a new gRPC transport from an existing client.
func NewGrpcTransport(client proto.SshxServiceClient, conn *grpc.ClientConn) *GrpcTransport {
	return &GrpcTransport{
		client: client,
		conn:   conn,
	}
}

// ConnectGrpc creates a new gRPC transport by connecting to a server.
func ConnectGrpc(origin string) (*GrpcTransport, error) {
	target := parseGRPCTarget(origin)
	
	// Use TLS for HTTPS origins, insecure for others
	var opts []grpc.DialOption
	if strings.HasPrefix(origin, "https://") {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	
	conn, err := grpc.Dial(target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}
	
	client := proto.NewSshxServiceClient(conn)
	return &GrpcTransport{
		client: client,
		conn:   conn,
	}, nil
}

// Open opens a new session on the server.
func (g *GrpcTransport) Open(ctx context.Context, request *proto.OpenRequest) (*proto.OpenResponse, error) {
	resp, err := g.client.Open(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("gRPC open request failed: %w", err)
	}
	return resp, nil
}

// Channel establishes a bidirectional streaming channel for real-time communication.
func (g *GrpcTransport) Channel(ctx context.Context) (chan *proto.ServerUpdate, chan *proto.ClientUpdate, error) {
	stream, err := g.client.Channel(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("gRPC channel request failed: %w", err)
	}

	// Create channels for bidirectional communication
	serverUpdates := make(chan *proto.ServerUpdate, 256)
	clientUpdates := make(chan *proto.ClientUpdate, 256)

	// Start goroutine to handle outbound messages (client -> server)
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				log.Printf("Failed to close send stream: %v", err)
			}
		}()
		
		for {
			select {
			case update, ok := <-clientUpdates:
				if !ok {
					return // Channel closed
				}
				if err := stream.Send(update); err != nil {
					log.Printf("Failed to send client update: %v", err)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start goroutine to handle inbound messages (server -> client)
	go func() {
		defer close(serverUpdates)
		
		for {
			update, err := stream.Recv()
			if err != nil {
				if err.Error() != "EOF" {
					log.Printf("Failed to receive server update: %v", err)
				}
				return
			}
			
			select {
			case serverUpdates <- update:
			case <-ctx.Done():
				return
			}
		}
	}()

	return serverUpdates, clientUpdates, nil
}

// Close closes an existing session on the server.
func (g *GrpcTransport) Close(ctx context.Context, request *proto.CloseRequest) error {
	_, err := g.client.Close(ctx, request)
	if err != nil {
		return fmt.Errorf("gRPC close request failed: %w", err)
	}
	return nil
}

// ConnectionType returns the connection type for logging/debugging purposes.
func (g *GrpcTransport) ConnectionType() string {
	return "gRPC"
}

// Cleanup any resources held by the transport.
func (g *GrpcTransport) Cleanup() error {
	if g.conn != nil {
		return g.conn.Close()
	}
	return nil
}

// parseGRPCTarget extracts the host:port from a URL for gRPC dialing
// This is copied from the existing controller.go to maintain compatibility
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

// TestGrpcConnectivity tests if gRPC connectivity is available to a server.
func TestGrpcConnectivity(origin string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	transport, err := ConnectGrpc(origin)
	if err != nil {
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
	// We expect this to either succeed or fail with a meaningful error
	// Either way, it proves the gRPC connection is working
	return err == nil
}
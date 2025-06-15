// Package client provides the core client functionality.
package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"unicode/utf8"

	"sshx-go/pkg/encrypt"
	"sshx-go/pkg/proto"
	"sshx-go/pkg/terminal"
)

const (
	contentChunkSize    = 1 << 16  // Send at most this many bytes at a time
	contentRollingBytes = 8 << 20  // Store at least this much content
	contentPruneBytes   = 12 << 20 // Prune when we exceed this length
)

// Runner variants define different terminal behaviors.
type Runner interface {
	Run(ctx context.Context, id uint32, encrypt *encrypt.Encrypt, shellRx <-chan ShellData, outputTx chan<- ClientMessage) error
}

// ShellRunner implements the shell variant that spawns a subprocess.
type ShellRunner struct {
	Shell string
}

// EchoRunner implements a mock runner that echoes input, useful for testing.
type EchoRunner struct{}

// ShellData represents internal messages routed to shell runners.
type ShellData struct {
	Type ShellDataType
	Data []byte
	Seq  uint64
	Rows uint32
	Cols uint32
}

type ShellDataType int

const (
	ShellDataTypeData ShellDataType = iota
	ShellDataTypeSync
	ShellDataTypeSize
)

// ClientMessage represents messages sent from client to server.
type ClientMessage struct {
	Type    ClientMessageType
	Hello   string
	Data    *TerminalData
	Shell   *proto.NewShell
	ShellID uint32
	Pong    uint64
	Error   string
}

type ClientMessageType int

const (
	ClientMessageTypeHello ClientMessageType = iota
	ClientMessageTypeData
	ClientMessageTypeCreatedShell
	ClientMessageTypeClosedShell
	ClientMessageTypePong
	ClientMessageTypeError
)

// TerminalData represents terminal output data.
type TerminalData struct {
	ID   uint32
	Data []byte
	Seq  uint64
}

// Run implements the Runner interface for ShellRunner.
// This matches the Rust shell_task function exactly.
func (sr *ShellRunner) Run(ctx context.Context, id uint32, encrypt *encrypt.Encrypt, shellRx <-chan ShellData, outputTx chan<- ClientMessage) error {
	return shellTask(ctx, id, encrypt, sr.Shell, shellRx, outputTx)
}

// Run implements the Runner interface for EchoRunner.
// This matches the Rust echo_task function exactly.
func (er *EchoRunner) Run(ctx context.Context, id uint32, encrypt *encrypt.Encrypt, shellRx <-chan ShellData, outputTx chan<- ClientMessage) error {
	return echoTask(ctx, id, encrypt, shellRx, outputTx)
}

// shellTask handles a single shell within the session.
// This matches the Rust shell_task function exactly.
func shellTask(ctx context.Context, id uint32, encrypt *encrypt.Encrypt, shell string, shellRx <-chan ShellData, outputTx chan<- ClientMessage) error {
	term, err := terminal.New(shell)
	if err != nil {
		return fmt.Errorf("failed to create terminal: %w", err)
	}
	defer term.Close()

	// Set initial window size - matches Rust implementation
	if err := term.SetWinsize(24, 80); err != nil {
		log.Printf("failed to set initial window size: %v", err)
	}

	var content strings.Builder // content from the terminal
	var contentOffset int       // bytes before the first character of content
	var seq int                 // our log of the server's sequence number
	var seqOutdated int         // number of times seq has been outdated
	buf := make([]byte, 4096)   // buffer for reading - same size as Rust
	finished := false           // set when this is done

	// Start a goroutine to read from terminal
	termOutput := make(chan []byte, 100)
	termError := make(chan error, 1)
	
	go func() {
		defer close(termOutput)
		for {
			n, err := term.Read(buf)
			if err != nil {
				if err != io.EOF {
					termError <- err
				}
				return
			}
			if n == 0 {
				return
			}
			
			// Make a copy of the data
			data := make([]byte, n)
			copy(data, buf[:n])
			
			select {
			case termOutput <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	for !finished {
		select {
		case <-ctx.Done():
			return ctx.Err()
			
		case data, ok := <-termOutput:
			if !ok {
				finished = true
			} else {
				// Process UTF-8 decoding like Rust implementation
				validData := make([]byte, 0, len(data))
				for len(data) > 0 {
					r, size := utf8.DecodeRune(data)
					if r == utf8.RuneError && size == 1 {
						// Skip invalid UTF-8 byte
						data = data[1:]
					} else {
						validData = append(validData, data[:size]...)
						data = data[size:]
					}
				}
				content.Write(validData)
			}
			
		case err := <-termError:
			return fmt.Errorf("terminal read error: %w", err)
			
		case item, ok := <-shellRx:
			if !ok {
				finished = true
				break
			}
			
			switch item.Type {
			case ShellDataTypeData:
				if _, err := term.Write(item.Data); err != nil {
					return fmt.Errorf("failed to write to terminal: %w", err)
				}
				
			case ShellDataTypeSync:
				// Sync logic matches Rust implementation exactly
				if item.Seq < uint64(seq) {
					seqOutdated++
					if seqOutdated >= 3 {
						seq = int(item.Seq)
					}
				}
				
			case ShellDataTypeSize:
				if err := term.SetWinsize(uint16(item.Rows), uint16(item.Cols)); err != nil {
					log.Printf("failed to resize terminal: %v", err)
				}
			}
		}

		// Send data if the server has fallen behind - matches Rust logic exactly
		contentStr := content.String()
		if contentOffset+len(contentStr) > seq {
			start := prevCharBoundary(contentStr, seq-contentOffset)
			end := prevCharBoundary(contentStr, min(start+contentChunkSize, len(contentStr)))
			
			// Encrypt segment exactly like Rust implementation
			data := encrypt.Segment(
				0x100000000|uint64(id), // stream number - matches Rust
				uint64(contentOffset+start),
				[]byte(contentStr[start:end]),
			)
			
			termData := &TerminalData{
				ID:   id,
				Data: data,
				Seq:  uint64(contentOffset + start),
			}
			
			msg := ClientMessage{
				Type: ClientMessageTypeData,
				Data: termData,
			}
			
			select {
			case outputTx <- msg:
			case <-ctx.Done():
				return ctx.Err()
			}
			
			seq = contentOffset + end
			seqOutdated = 0
		}

		// Prune content if it gets too large - matches Rust logic exactly
		if len(contentStr) > contentPruneBytes && seq-contentRollingBytes > contentOffset {
			pruned := (seq - contentRollingBytes) - contentOffset
			pruned = prevCharBoundary(contentStr, pruned)
			contentOffset += pruned
			
			// Rebuild content without the pruned part
			newContent := contentStr[pruned:]
			content.Reset()
			content.WriteString(newContent)
		}
	}
	
	return nil
}

// echoTask implements the echo runner for testing.
// This matches the Rust echo_task function exactly.
func echoTask(ctx context.Context, id uint32, encrypt *encrypt.Encrypt, shellRx <-chan ShellData, outputTx chan<- ClientMessage) error {
	var seq uint64
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
			
		case item, ok := <-shellRx:
			if !ok {
				return nil
			}
			
			switch item.Type {
			case ShellDataTypeData:
				msg := string(item.Data)
				
				termData := &TerminalData{
					ID:   id,
					Data: encrypt.Segment(0x100000000|uint64(id), seq, []byte(msg)),
					Seq:  seq,
				}
				
				clientMsg := ClientMessage{
					Type: ClientMessageTypeData,
					Data: termData,
				}
				
				select {
				case outputTx <- clientMsg:
				case <-ctx.Done():
					return ctx.Err()
				}
				
				seq += uint64(len(msg))
				
			case ShellDataTypeSync:
				// Ignore sync messages in echo mode
				
			case ShellDataTypeSize:
				// Ignore resize messages in echo mode
			}
		}
	}
}

// prevCharBoundary finds the last UTF-8 character boundary before an index.
// This matches the Rust prev_char_boundary function exactly.
func prevCharBoundary(s string, i int) int {
	if i >= len(s) {
		return len(s)
	}
	if i <= 0 {
		return 0
	}
	
	// In Go, we can use utf8.RuneStart to find character boundaries
	for i > 0 && !utf8.RuneStart(s[i]) {
		i--
	}
	return i
}


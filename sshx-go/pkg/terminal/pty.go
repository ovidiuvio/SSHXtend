// Package terminal provides platform-specific terminal/PTY handling.
// This implementation uses proper PTY support via github.com/creack/pty.
package terminal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
)

// Terminal represents a PTY terminal with an attached process.
type Terminal struct {
	cmd *exec.Cmd
	pty *os.File
}

// New creates a new terminal with the specified shell command using PTY.
func New(shell string) (*Terminal, error) {
	cmd := exec.Command(shell)
	
	// Set environment variables
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"TERM_PROGRAM=sshx",
	)
	
	// Start the command with a PTY - this matches the Rust implementation
	ptty, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}
	
	return &Terminal{
		cmd: cmd,
		pty: ptty,
	}, nil
}

// Read reads data from the terminal.
func (t *Terminal) Read(p []byte) (int, error) {
	return t.pty.Read(p)
}

// Write writes data to the terminal.
func (t *Terminal) Write(p []byte) (int, error) {
	return t.pty.Write(p)
}

// SetWinsize sets the window size of the terminal.
func (t *Terminal) SetWinsize(rows, cols uint16) error {
	size := &pty.Winsize{
		Rows: rows,
		Cols: cols,
	}
	return pty.Setsize(t.pty, size)
}

// GetWinsize gets the current window size of the terminal.
func (t *Terminal) GetWinsize() (rows, cols uint16, err error) {
	size, err := pty.GetsizeFull(t.pty)
	if err != nil {
		return 0, 0, err
	}
	return size.Rows, size.Cols, nil
}

// Close closes the terminal and terminates the process.
func (t *Terminal) Close() error {
	var firstErr error
	
	// Close the PTY first to signal the process
	if t.pty != nil {
		if err := t.pty.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		t.pty = nil
	}
	
	// Kill the process if it's still running
	if t.cmd != nil && t.cmd.Process != nil {
		// Try graceful termination first
		t.cmd.Process.Signal(os.Interrupt)
		
		// Wait a bit for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- t.cmd.Wait()
		}()
		
		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(2 * time.Second):
			// Force kill if graceful shutdown failed
			if err := t.cmd.Process.Kill(); err != nil && firstErr == nil {
				firstErr = err
			}
			<-done // Wait for the killed process
		}
		
		t.cmd = nil
	}
	
	return firstErr
}

// Wait waits for the terminal process to exit.
func (t *Terminal) Wait() error {
	return t.cmd.Wait()
}

// GetDefaultShell returns the default shell for the current system.
func GetDefaultShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	
	// Try common shell locations
	shells := []string{
		"/bin/bash",
		"/bin/sh",
		"/usr/local/bin/bash",
		"/usr/local/bin/sh",
	}
	
	for _, shell := range shells {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}
	
	return "sh"
}

// Process returns the underlying process.
func (t *Terminal) Process() *os.Process {
	return t.cmd.Process
}

// ProcessState returns the process state.
func (t *Terminal) ProcessState() *os.ProcessState {
	return t.cmd.ProcessState
}

// Ensure Terminal implements io.ReadWriteCloser
var _ io.ReadWriteCloser = (*Terminal)(nil)
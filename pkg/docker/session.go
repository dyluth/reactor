package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/moby/term"
)

// TTYSize represents terminal dimensions
type TTYSize struct {
	Rows uint16
	Cols uint16
}

// TerminalState manages terminal state and cleanup
type TerminalState struct {
	OriginalState *term.State
	RawModeSet    bool
	Size          TTYSize
	SignalChan    chan os.Signal
	ResizeChan    chan TTYSize
	mutex         sync.Mutex
}

// NewTerminalState creates a new terminal state manager
func NewTerminalState() *TerminalState {
	return &TerminalState{
		SignalChan: make(chan os.Signal, 1),
		ResizeChan: make(chan TTYSize, 1),
	}
}

// Setup configures terminal for interactive session
func (ts *TerminalState) Setup() error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	if !term.IsTerminal(os.Stdin.Fd()) {
		return nil // Not a terminal, skip TTY setup
	}

	// Save current terminal state
	oldState, err := term.SaveState(os.Stdin.Fd())
	if err != nil {
		return fmt.Errorf("failed to save terminal state: %w", err)
	}
	ts.OriginalState = oldState

	// Set terminal to raw mode
	_, err = term.SetRawTerminal(os.Stdin.Fd())
	if err != nil {
		return fmt.Errorf("failed to set raw terminal: %w", err)
	}
	ts.RawModeSet = true

	// Get initial terminal size
	size, err := ts.GetTerminalSize()
	if err != nil {
		// Don't fail on size detection, use defaults
		size = TTYSize{Rows: 24, Cols: 80}
	}
	ts.Size = size

	return nil
}

// Cleanup restores terminal state
func (ts *TerminalState) Cleanup() error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	if ts.OriginalState != nil && ts.RawModeSet {
		if err := term.RestoreTerminal(os.Stdin.Fd(), ts.OriginalState); err != nil {
			return fmt.Errorf("failed to restore terminal state: %w", err)
		}
		ts.RawModeSet = false
	}

	// Close channels
	if ts.SignalChan != nil {
		signal.Stop(ts.SignalChan)
		close(ts.SignalChan)
		ts.SignalChan = nil
	}
	if ts.ResizeChan != nil {
		close(ts.ResizeChan)
		ts.ResizeChan = nil
	}

	return nil
}

// GetTerminalSize returns current terminal dimensions
func (ts *TerminalState) GetTerminalSize() (TTYSize, error) {
	if !term.IsTerminal(os.Stdin.Fd()) {
		return TTYSize{Rows: 24, Cols: 80}, nil
	}

	ws, err := term.GetWinsize(os.Stdin.Fd())
	if err != nil {
		return TTYSize{}, fmt.Errorf("failed to get terminal size: %w", err)
	}

	return TTYSize{
		Rows: ws.Height,
		Cols: ws.Width,
	}, nil
}

// StartSignalHandling begins signal forwarding to container
func (ts *TerminalState) StartSignalHandling() {
	// Register for signals we want to forward
	signal.Notify(ts.SignalChan,
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGTERM, // Termination
		syscall.SIGQUIT, // Ctrl+\
		syscall.SIGTSTP, // Ctrl+Z
	)
}

// AttachInteractiveSession attaches to a running container with enhanced TTY support
func (s *Service) AttachInteractiveSession(ctx context.Context, containerID string) error {
	// Check if container is running
	containerInfo, err := s.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	if !containerInfo.State.Running {
		return fmt.Errorf("container %s is not running", containerID)
	}

	// Initialize enhanced terminal state
	termState := NewTerminalState()
	defer func() {
		if err := termState.Cleanup(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}()

	// Setup terminal for interactive session
	if err := termState.Setup(); err != nil {
		return fmt.Errorf("failed to setup terminal: %w", err)
	}

	isTerminal := term.IsTerminal(os.Stdin.Fd())

	// Create exec instance for interactive shell
	execConfig := types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          isTerminal,
		Cmd:          []string{"/bin/bash"}, // Default to bash, could be configurable
	}

	execResp, err := s.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance: %w", err)
	}

	// Attach to the exec instance
	attachResp, err := s.client.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{
		Detach: false,
		Tty:    isTerminal,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to exec instance: %w", err)
	}
	defer attachResp.Close()

	// Start signal handling for terminal
	if isTerminal {
		termState.StartSignalHandling()
	}

	// Channel for coordinating goroutines and handling errors
	errChan := make(chan error, 5)
	var wg sync.WaitGroup

	// Start exec process
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.client.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{
			Detach: false,
			Tty:    isTerminal,
		})
		if err != nil {
			errChan <- fmt.Errorf("exec start failed: %w", err)
		}
	}()

	// Copy stdin to container
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.Copy(attachResp.Conn, os.Stdin)
		if err != nil && err != io.EOF {
			errChan <- fmt.Errorf("stdin copy failed: %w", err)
		}
	}()

	// Copy container output to stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.Copy(os.Stdout, attachResp.Reader)
		if err != nil && err != io.EOF {
			errChan <- fmt.Errorf("stdout copy failed: %w", err)
		}
	}()

	// Handle signals and terminal resize if in TTY mode
	if isTerminal {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.handleTerminalEvents(ctx, containerID, execResp.ID, termState, errChan)
		}()
	}

	// Wait for completion or error
	go func() {
		wg.Wait()
		errChan <- nil // Signal normal completion
	}()

	// Return first error or nil on success
	return <-errChan
}

// handleTerminalEvents processes signals and terminal resize events
func (s *Service) handleTerminalEvents(ctx context.Context, containerID, execID string, termState *TerminalState, errChan chan<- error) {
	// Monitor for terminal resize events
	go s.monitorTerminalResize(ctx, containerID, execID, termState)

	// Handle signals
	for {
		select {
		case sig := <-termState.SignalChan:
			if sig == nil {
				return // Channel closed
			}
			
			// Forward signal to container process
			if err := s.forwardSignal(ctx, execID, sig); err != nil {
				// Log warning but don't fail the session
				fmt.Fprintf(os.Stderr, "Warning: failed to forward signal %v: %v\n", sig, err)
			}

		case <-ctx.Done():
			return // Context cancelled
		}
	}
}

// monitorTerminalResize watches for terminal size changes and updates container TTY
func (s *Service) monitorTerminalResize(ctx context.Context, containerID, execID string, termState *TerminalState) {
	// Initial resize to current terminal size
	if err := s.resizeContainerTTY(ctx, containerID, execID, termState.Size); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed initial TTY resize: %v\n", err)
	}

	// TODO: Implement actual terminal resize monitoring
	// This would require platform-specific code to detect SIGWINCH or terminal changes
	// For now, we set the initial size correctly
}

// forwardSignal forwards a signal to the container exec process
func (s *Service) forwardSignal(ctx context.Context, execID string, sig os.Signal) error {
	// Convert os.Signal to the signal name that Docker API expects
	var signalStr string
	switch sig {
	case syscall.SIGINT:
		signalStr = "INT"
	case syscall.SIGTERM:
		signalStr = "TERM"
	case syscall.SIGQUIT:
		signalStr = "QUIT"
	case syscall.SIGTSTP:
		signalStr = "TSTP"
	default:
		return fmt.Errorf("unsupported signal: %v", sig)
	}

	// Use Docker API to send signal to exec process
	// Note: This sends to the exec process, not the main container process
	return s.client.ContainerKill(ctx, execID, signalStr)
}

// resizeContainerTTY resizes the container's TTY to match terminal dimensions
func (s *Service) resizeContainerTTY(ctx context.Context, containerID, execID string, size TTYSize) error {
	// Use ContainerExecResize for exec sessions
	return s.client.ContainerExecResize(ctx, execID, container.ResizeOptions{
		Height: uint(size.Rows),
		Width:  uint(size.Cols),
	})
}

// ResizeTerminal provides external interface for terminal resizing
func (s *Service) ResizeTerminal(ctx context.Context, containerID string, size TTYSize) error {
	// This can be used by external callers to resize container TTY
	return s.client.ContainerResize(ctx, containerID, container.ResizeOptions{
		Height: uint(size.Rows),
		Width:  uint(size.Cols),
	})
}
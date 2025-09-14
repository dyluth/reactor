package docker

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

// TerminalSize represents the dimensions of a terminal
type TerminalSize struct {
	Width  int
	Height int
}

// getTerminalSize detects host terminal dimensions using golang.org/x/term
func getTerminalSize() (*TerminalSize, error) {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal size: %w", err)
	}
	return &TerminalSize{Width: width, Height: height}, nil
}

// isTerminalAvailable checks if we have a terminal available for TTY operations
func isTerminalAvailable() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// createTTYEnvironment creates environment variables for proper TTY detection
func createTTYEnvironment(terminalSize *TerminalSize) []string {
	env := []string{
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"FORCE_COLOR=1",
	}

	if terminalSize != nil {
		env = append(env,
			fmt.Sprintf("COLUMNS=%d", terminalSize.Width),
			fmt.Sprintf("LINES=%d", terminalSize.Height),
		)
	}

	return env
}

// enableRawMode puts host terminal in raw mode for proper character forwarding
func enableRawMode() (*term.State, error) {
	if !isTerminalAvailable() {
		return nil, fmt.Errorf("terminal not available")
	}
	return term.MakeRaw(int(os.Stdin.Fd()))
}

// restoreTerminalMode restores terminal to original state
func restoreTerminalMode(state *term.State) error {
	if state == nil {
		return nil
	}
	return term.Restore(int(os.Stdin.Fd()), state)
}

// TTYManager manages terminal state and signal handling for interactive sessions
type TTYManager struct {
	originalState *term.State
	currentSize   *TerminalSize
	resizeChan    chan os.Signal
	signalChan    chan os.Signal
}

// NewTTYManager creates a new TTY manager for an interactive session
func NewTTYManager() (*TTYManager, error) {
	// Detect initial terminal size
	terminalSize, err := getTerminalSize()
	if err != nil {
		// Use safe defaults if detection fails
		terminalSize = &TerminalSize{Width: 80, Height: 24}
	}

	manager := &TTYManager{
		currentSize: terminalSize,
		resizeChan:  make(chan os.Signal, 1),
		signalChan:  make(chan os.Signal, 1),
	}

	// Enable raw mode if terminal is available
	if isTerminalAvailable() {
		state, err := enableRawMode()
		if err != nil {
			return nil, fmt.Errorf("failed to enable raw mode: %w", err)
		}
		manager.originalState = state
	}

	// Set up signal handling
	signal.Notify(manager.resizeChan, syscall.SIGWINCH)
	signal.Notify(manager.signalChan, syscall.SIGINT, syscall.SIGTERM)

	return manager, nil
}

// Close cleans up the TTY manager and restores terminal state
func (tm *TTYManager) Close() error {
	// Stop signal handling (idempotent)
	if tm.resizeChan != nil {
		signal.Stop(tm.resizeChan)
		// Only close if not already closed
		select {
		case <-tm.resizeChan:
			// Channel already closed
		default:
			close(tm.resizeChan)
		}
		tm.resizeChan = nil
	}
	if tm.signalChan != nil {
		signal.Stop(tm.signalChan)
		// Only close if not already closed
		select {
		case <-tm.signalChan:
			// Channel already closed
		default:
			close(tm.signalChan)
		}
		tm.signalChan = nil
	}

	// Restore terminal state (idempotent)
	if tm.originalState != nil {
		err := restoreTerminalMode(tm.originalState)
		tm.originalState = nil // Prevent double restore
		return err
	}

	return nil
}

// GetCurrentSize returns the current terminal size
func (tm *TTYManager) GetCurrentSize() *TerminalSize {
	if tm.currentSize == nil {
		return &TerminalSize{Width: 80, Height: 24}
	}
	return tm.currentSize
}

// GetEnvironment returns environment variables for TTY detection
func (tm *TTYManager) GetEnvironment() []string {
	return createTTYEnvironment(tm.currentSize)
}

// WatchResize monitors terminal resize events and calls the provided handler
func (tm *TTYManager) WatchResize(handler func(width, height int)) {
	if tm.resizeChan == nil {
		return
	}

	go func() {
		for range tm.resizeChan {
			// Get new terminal size
			newSize, err := getTerminalSize()
			if err != nil {
				continue // Skip this resize event
			}

			// Only handle resize if dimensions actually changed
			if newSize.Width != tm.currentSize.Width || newSize.Height != tm.currentSize.Height {
				tm.currentSize = newSize
				handler(newSize.Width, newSize.Height)
			}
		}
	}()
}

// WatchSignals monitors interrupt signals and calls the provided handler
func (tm *TTYManager) WatchSignals(handler func(sig os.Signal)) {
	if tm.signalChan == nil {
		return
	}

	go func() {
		for sig := range tm.signalChan {
			handler(sig)
		}
	}()
}

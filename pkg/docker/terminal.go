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
// with comprehensive error handling and validation
func enableRawMode() (*term.State, error) {
	if !isTerminalAvailable() {
		return nil, fmt.Errorf("terminal not available - cannot enable raw mode")
	}

	// Get current terminal state before making changes
	stdinFd := int(os.Stdin.Fd())
	originalState, err := term.GetState(stdinFd)
	if err != nil {
		return nil, fmt.Errorf("failed to get current terminal state: %w", err)
	}

	// Enable raw mode with proper error handling
	rawState, err := term.MakeRaw(stdinFd)
	if err != nil {
		// If raw mode fails, we still have the original state
		return originalState, fmt.Errorf("failed to enable raw mode: %w", err)
	}

	return rawState, nil
}

// restoreTerminalMode restores terminal to original state with enhanced error handling
func restoreTerminalMode(state *term.State) error {
	if state == nil {
		return nil // Nothing to restore
	}

	if !isTerminalAvailable() {
		return fmt.Errorf("terminal not available - cannot restore mode")
	}

	stdinFd := int(os.Stdin.Fd())
	if err := term.Restore(stdinFd, state); err != nil {
		return fmt.Errorf("failed to restore terminal mode: %w", err)
	}

	return nil
}

// safeRestoreTerminalMode provides panic-safe terminal restoration
// This should be used in defer statements to ensure terminal is always restored
func safeRestoreTerminalMode(state *term.State) {
	if state == nil {
		return
	}

	// Catch any panics during terminal restoration
	defer func() {
		if r := recover(); r != nil {
			// Log the panic but don't re-panic - terminal restoration is critical
			fmt.Fprintf(os.Stderr, "WARNING: panic during terminal restoration: %v\n", r)
		}
	}()

	if err := restoreTerminalMode(state); err != nil {
		// Log error but don't fail - this is used in cleanup scenarios
		fmt.Fprintf(os.Stderr, "WARNING: failed to restore terminal mode: %v\n", err)
	}
}

// TTYManager manages terminal state and signal handling for interactive sessions
type TTYManager struct {
	originalState *term.State
	currentSize   *TerminalSize
	resizeChan    chan os.Signal
	signalChan    chan os.Signal
}

// NewTTYManager creates a new TTY manager for an interactive session
// with enhanced error handling and panic recovery
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

	// Enable raw mode if terminal is available with enhanced error handling
	if isTerminalAvailable() {
		state, err := enableRawMode()
		if err != nil {
			// Clean up channels before returning error
			close(manager.resizeChan)
			close(manager.signalChan)
			return nil, fmt.Errorf("failed to enable raw mode: %w", err)
		}
		manager.originalState = state

		// Set up panic recovery for the TTY manager
		// This ensures terminal is restored even if something goes wrong
		defer func() {
			if r := recover(); r != nil {
				safeRestoreTerminalMode(state)
				panic(r) // Re-panic after cleanup
			}
		}()
	}

	// Set up signal handling with error checking
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

	// Restore terminal state with panic recovery (idempotent)
	if tm.originalState != nil {
		state := tm.originalState
		tm.originalState = nil // Prevent double restore

		// Use safe restoration to handle any potential panics
		safeRestoreTerminalMode(state)
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

// ResizeHandler defines the interface for handling terminal resize events
type ResizeHandler interface {
	HandleResize(width, height int) error
}

// ResizeHandlerFunc is a function adapter for ResizeHandler interface
type ResizeHandlerFunc func(width, height int) error

// HandleResize implements ResizeHandler interface
func (f ResizeHandlerFunc) HandleResize(width, height int) error {
	return f(width, height)
}

// WatchResize monitors terminal resize events and calls the provided handler with enhanced error handling
func (tm *TTYManager) WatchResize(handler ResizeHandler) {
	if tm.resizeChan == nil {
		return
	}

	go func() {
		// Panic recovery for resize handling goroutine
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "WARNING: panic in resize handler: %v\n", r)
			}
		}()

		for range tm.resizeChan {
			// Get new terminal size with error handling
			newSize, err := getTerminalSize()
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARNING: failed to get terminal size during resize: %v\n", err)
				continue // Skip this resize event
			}

			// Only handle resize if dimensions actually changed
			if newSize.Width != tm.currentSize.Width || newSize.Height != tm.currentSize.Height {
				tm.currentSize = newSize

				// Call handler with error handling
				if err := handler.HandleResize(newSize.Width, newSize.Height); err != nil {
					// Log error but continue processing resize events
					fmt.Fprintf(os.Stderr, "WARNING: resize handler error (size %dx%d): %v\n",
						newSize.Width, newSize.Height, err)
				}
			}
		}
	}()
}

// WatchResizeFunc provides a convenient function-based resize watching interface
// This maintains backward compatibility with the original function signature
func (tm *TTYManager) WatchResizeFunc(handler func(width, height int)) {
	tm.WatchResize(ResizeHandlerFunc(func(width, height int) error {
		handler(width, height)
		return nil
	}))
}

// WatchResizeWithErrorHandling provides resize watching with explicit error handling
func (tm *TTYManager) WatchResizeWithErrorHandling(handler func(width, height int) error) {
	tm.WatchResize(ResizeHandlerFunc(handler))
}

// SignalHandler defines the interface for handling terminal signals
type SignalHandler interface {
	HandleSignal(sig os.Signal) error
}

// SignalHandlerFunc is a function adapter for SignalHandler interface
type SignalHandlerFunc func(sig os.Signal) error

// HandleSignal implements SignalHandler interface
func (f SignalHandlerFunc) HandleSignal(sig os.Signal) error {
	return f(sig)
}

// WatchSignals monitors interrupt signals and calls the provided handler with enhanced error handling
func (tm *TTYManager) WatchSignals(handler SignalHandler) {
	if tm.signalChan == nil {
		return
	}

	go func() {
		// Panic recovery for signal handling goroutine
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "WARNING: panic in signal handler: %v\n", r)
			}
		}()

		for sig := range tm.signalChan {
			// Handle signal with error reporting but don't stop signal processing
			if err := handler.HandleSignal(sig); err != nil {
				fmt.Fprintf(os.Stderr, "WARNING: signal handler error for %v: %v\n", sig, err)
			}
		}
	}()
}

// WatchSignalsFunc provides a convenient function-based signal watching interface
// This maintains backward compatibility with the original function signature
func (tm *TTYManager) WatchSignalsFunc(handler func(sig os.Signal)) {
	tm.WatchSignals(SignalHandlerFunc(func(sig os.Signal) error {
		handler(sig)
		return nil
	}))
}

// WatchSignalsWithErrorHandling provides signal watching with explicit error handling
func (tm *TTYManager) WatchSignalsWithErrorHandling(handler func(sig os.Signal) error) {
	tm.WatchSignals(SignalHandlerFunc(handler))
}

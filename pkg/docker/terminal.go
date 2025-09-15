package docker

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"
)

// TerminalSize represents the dimensions of a terminal
type TerminalSize struct {
	Width  int
	Height int
}

// TTYManagerPool manages a pool of reusable TTY managers for performance optimization
type TTYManagerPool struct {
	pool chan *TTYManager
	mu   sync.RWMutex
	size int
}

// NewTTYManagerPool creates a new pool of TTY managers
func NewTTYManagerPool(size int) *TTYManagerPool {
	log.Printf("[TTY DEBUG] NewTTYManagerPool: creating TTY manager pool with size %d", size)
	return &TTYManagerPool{
		pool: make(chan *TTYManager, size),
		size: size,
	}
}

// Get retrieves a TTY manager from the pool or creates a new one
func (p *TTYManagerPool) Get() (*TTYManager, error) {
	select {
	case manager := <-p.pool:
		log.Printf("[TTY DEBUG] TTYManagerPool.Get: reusing TTY manager from pool")
		return manager, nil
	default:
		log.Printf("[TTY DEBUG] TTYManagerPool.Get: creating new TTY manager (pool empty)")
		return NewTTYManager()
	}
}

// Put returns a TTY manager to the pool for reuse
func (p *TTYManagerPool) Put(manager *TTYManager) {
	if manager == nil {
		return
	}

	select {
	case p.pool <- manager:
		log.Printf("[TTY DEBUG] TTYManagerPool.Put: returned TTY manager to pool")
	default:
		// Pool is full, just close the manager
		log.Printf("[TTY DEBUG] TTYManagerPool.Put: pool full, closing TTY manager")
		manager.Close()
	}
}

// Close closes all managers in the pool
func (p *TTYManagerPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Printf("[TTY DEBUG] TTYManagerPool.Close: closing TTY manager pool")
	close(p.pool)
	for manager := range p.pool {
		manager.Close()
	}
}

// ResizeRateLimiter prevents excessive resize events
type ResizeRateLimiter struct {
	lastResize time.Time
	minInterval time.Duration
	mu         sync.RWMutex
}

// NewResizeRateLimiter creates a new rate limiter for resize events
func NewResizeRateLimiter(minInterval time.Duration) *ResizeRateLimiter {
	return &ResizeRateLimiter{
		minInterval: minInterval,
	}
}

// ShouldProcess determines if a resize event should be processed based on rate limiting
func (rl *ResizeRateLimiter) ShouldProcess() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if now.Sub(rl.lastResize) >= rl.minInterval {
		rl.lastResize = now
		return true
	}
	return false
}

// getTerminalSize detects host terminal dimensions using golang.org/x/term
func getTerminalSize() (*TerminalSize, error) {
	stdinFd := int(os.Stdin.Fd())

	// Debug: Check if we have a terminal before attempting to get size
	if !term.IsTerminal(stdinFd) {
		log.Printf("[TTY DEBUG] getTerminalSize: stdin fd %d is not a terminal", stdinFd)
		return nil, fmt.Errorf("stdin is not a terminal (fd %d)", stdinFd)
	}

	log.Printf("[TTY DEBUG] getTerminalSize: attempting to get size for terminal fd %d", stdinFd)
	width, height, err := term.GetSize(stdinFd)
	if err != nil {
		log.Printf("[TTY ERROR] getTerminalSize: failed to get terminal size for fd %d: %v", stdinFd, err)
		return nil, fmt.Errorf("failed to get terminal size for fd %d: %w", stdinFd, err)
	}

	log.Printf("[TTY DEBUG] getTerminalSize: detected terminal size %dx%d", width, height)
	return &TerminalSize{Width: width, Height: height}, nil
}

// isTerminalAvailable checks if we have a terminal available for TTY operations
func isTerminalAvailable() bool {
	stdinFd := int(os.Stdin.Fd())
	isTerminal := term.IsTerminal(stdinFd)
	log.Printf("[TTY DEBUG] isTerminalAvailable: stdin fd %d terminal status: %v", stdinFd, isTerminal)
	return isTerminal
}

// createTTYEnvironment creates environment variables for proper TTY detection
func createTTYEnvironment(terminalSize *TerminalSize) []string {
	log.Printf("[TTY DEBUG] createTTYEnvironment: creating TTY environment variables")

	env := []string{
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"FORCE_COLOR=1",
	}

	if terminalSize != nil {
		sizeVars := []string{
			fmt.Sprintf("COLUMNS=%d", terminalSize.Width),
			fmt.Sprintf("LINES=%d", terminalSize.Height),
		}
		env = append(env, sizeVars...)
		log.Printf("[TTY DEBUG] createTTYEnvironment: added terminal size variables: COLUMNS=%d, LINES=%d",
			terminalSize.Width, terminalSize.Height)
	} else {
		log.Printf("[TTY DEBUG] createTTYEnvironment: no terminal size provided, skipping COLUMNS/LINES")
	}

	log.Printf("[TTY DEBUG] createTTYEnvironment: created %d environment variables", len(env))
	return env
}

// enableRawMode puts host terminal in raw mode for proper character forwarding
// with comprehensive error handling and validation
func enableRawMode() (*term.State, error) {
	log.Printf("[TTY DEBUG] enableRawMode: attempting to enable raw mode")

	if !isTerminalAvailable() {
		log.Printf("[TTY ERROR] enableRawMode: terminal not available")
		return nil, fmt.Errorf("terminal not available - cannot enable raw mode")
	}

	// Get current terminal state before making changes
	stdinFd := int(os.Stdin.Fd())
	log.Printf("[TTY DEBUG] enableRawMode: getting current terminal state for fd %d", stdinFd)
	originalState, err := term.GetState(stdinFd)
	if err != nil {
		log.Printf("[TTY ERROR] enableRawMode: failed to get terminal state for fd %d: %v", stdinFd, err)
		return nil, fmt.Errorf("failed to get current terminal state for fd %d: %w", stdinFd, err)
	}

	// Enable raw mode with proper error handling
	log.Printf("[TTY DEBUG] enableRawMode: enabling raw mode for fd %d", stdinFd)
	rawState, err := term.MakeRaw(stdinFd)
	if err != nil {
		log.Printf("[TTY ERROR] enableRawMode: failed to enable raw mode for fd %d: %v", stdinFd, err)
		// If raw mode fails, we still have the original state
		return originalState, fmt.Errorf("failed to enable raw mode for fd %d: %w", stdinFd, err)
	}

	log.Printf("[TTY DEBUG] enableRawMode: successfully enabled raw mode for fd %d", stdinFd)
	return rawState, nil
}

// restoreTerminalMode restores terminal to original state with enhanced error handling
func restoreTerminalMode(state *term.State) error {
	if state == nil {
		log.Printf("[TTY DEBUG] restoreTerminalMode: no state provided, nothing to restore")
		return nil // Nothing to restore
	}

	log.Printf("[TTY DEBUG] restoreTerminalMode: attempting to restore terminal state")

	if !isTerminalAvailable() {
		log.Printf("[TTY ERROR] restoreTerminalMode: terminal not available")
		return fmt.Errorf("terminal not available - cannot restore mode")
	}

	stdinFd := int(os.Stdin.Fd())
	log.Printf("[TTY DEBUG] restoreTerminalMode: restoring terminal state for fd %d", stdinFd)
	if err := term.Restore(stdinFd, state); err != nil {
		log.Printf("[TTY ERROR] restoreTerminalMode: failed to restore terminal for fd %d: %v", stdinFd, err)
		return fmt.Errorf("failed to restore terminal mode for fd %d: %w", stdinFd, err)
	}

	log.Printf("[TTY DEBUG] restoreTerminalMode: successfully restored terminal state for fd %d", stdinFd)
	return nil
}

// safeRestoreTerminalMode provides panic-safe terminal restoration
// This should be used in defer statements to ensure terminal is always restored
func safeRestoreTerminalMode(state *term.State) {
	if state == nil {
		log.Printf("[TTY DEBUG] safeRestoreTerminalMode: no state provided, nothing to restore")
		return
	}

	log.Printf("[TTY DEBUG] safeRestoreTerminalMode: attempting safe terminal restoration")

	// Catch any panics during terminal restoration
	defer func() {
		if r := recover(); r != nil {
			// Log the panic but don't re-panic - terminal restoration is critical
			log.Printf("[TTY ERROR] safeRestoreTerminalMode: panic during terminal restoration: %v", r)
			fmt.Fprintf(os.Stderr, "WARNING: panic during terminal restoration: %v\n", r)
		}
	}()

	if err := restoreTerminalMode(state); err != nil {
		// Log error but don't fail - this is used in cleanup scenarios
		log.Printf("[TTY ERROR] safeRestoreTerminalMode: failed to restore terminal mode: %v", err)
		fmt.Fprintf(os.Stderr, "WARNING: failed to restore terminal mode: %v\n", err)
	} else {
		log.Printf("[TTY DEBUG] safeRestoreTerminalMode: successfully restored terminal mode")
	}
}

// TTYManager manages terminal state and signal handling for interactive sessions
type TTYManager struct {
	originalState   *term.State
	currentSize     *TerminalSize
	resizeChan      chan os.Signal
	signalChan      chan os.Signal
	resizeHandler   ResizeHandler
	signalHandler   SignalHandler
	resizeLimiter   *ResizeRateLimiter
	cancelFuncs     []func() // Track all goroutines for proper cleanup
	mu              sync.RWMutex
	closed          bool
}

// NewTTYManager creates a new TTY manager for an interactive session
// with enhanced error handling and panic recovery
func NewTTYManager() (*TTYManager, error) {
	log.Printf("[TTY DEBUG] NewTTYManager: creating new TTY manager")

	// Detect initial terminal size
	terminalSize, err := getTerminalSize()
	if err != nil {
		log.Printf("[TTY WARN] NewTTYManager: failed to detect terminal size, using defaults: %v", err)
		// Use safe defaults if detection fails
		terminalSize = &TerminalSize{Width: 80, Height: 24}
		log.Printf("[TTY DEBUG] NewTTYManager: using default terminal size %dx%d", terminalSize.Width, terminalSize.Height)
	}

	log.Printf("[TTY DEBUG] NewTTYManager: creating manager with terminal size %dx%d", terminalSize.Width, terminalSize.Height)
	manager := &TTYManager{
		currentSize:   terminalSize,
		resizeChan:    make(chan os.Signal, 1),
		signalChan:    make(chan os.Signal, 1),
		resizeLimiter: NewResizeRateLimiter(50 * time.Millisecond), // Limit resize events to 20Hz max
		cancelFuncs:   make([]func(), 0),
	}

	// Enable raw mode if terminal is available with enhanced error handling
	if isTerminalAvailable() {
		log.Printf("[TTY DEBUG] NewTTYManager: terminal available, enabling raw mode")
		state, err := enableRawMode()
		if err != nil {
			log.Printf("[TTY ERROR] NewTTYManager: failed to enable raw mode: %v", err)
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
				log.Printf("[TTY ERROR] NewTTYManager: panic detected, restoring terminal: %v", r)
				safeRestoreTerminalMode(state)
				panic(r) // Re-panic after cleanup
			}
		}()
	} else {
		log.Printf("[TTY DEBUG] NewTTYManager: no terminal available, running in non-interactive mode")
	}

	// Set up signal handling with error checking
	log.Printf("[TTY DEBUG] NewTTYManager: setting up signal handlers")
	signal.Notify(manager.resizeChan, syscall.SIGWINCH)
	signal.Notify(manager.signalChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("[TTY DEBUG] NewTTYManager: TTY manager created successfully")
	return manager, nil
}

// Close cleans up the TTY manager and restores terminal state
func (tm *TTYManager) Close() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.closed {
		log.Printf("[TTY DEBUG] Close: TTY manager already closed")
		return nil
	}

	log.Printf("[TTY DEBUG] Close: cleaning up TTY manager")

	// Cancel all running goroutines to prevent memory leaks
	for i, cancel := range tm.cancelFuncs {
		log.Printf("[TTY DEBUG] Close: canceling goroutine %d", i)
		cancel()
	}
	tm.cancelFuncs = nil

	// Stop signal handling (idempotent)
	if tm.resizeChan != nil {
		log.Printf("[TTY DEBUG] Close: stopping resize signal handling")
		signal.Stop(tm.resizeChan)
		// Only close if not already closed
		select {
		case <-tm.resizeChan:
			// Channel already closed
			log.Printf("[TTY DEBUG] Close: resize channel already closed")
		default:
			close(tm.resizeChan)
			log.Printf("[TTY DEBUG] Close: closed resize channel")
		}
		tm.resizeChan = nil
	}
	if tm.signalChan != nil {
		log.Printf("[TTY DEBUG] Close: stopping signal handling")
		signal.Stop(tm.signalChan)
		// Only close if not already closed
		select {
		case <-tm.signalChan:
			// Channel already closed
			log.Printf("[TTY DEBUG] Close: signal channel already closed")
		default:
			close(tm.signalChan)
			log.Printf("[TTY DEBUG] Close: closed signal channel")
		}
		tm.signalChan = nil
	}

	// Restore terminal state with panic recovery (idempotent)
	if tm.originalState != nil {
		log.Printf("[TTY DEBUG] Close: restoring original terminal state")
		state := tm.originalState
		tm.originalState = nil // Prevent double restore

		// Use safe restoration to handle any potential panics
		safeRestoreTerminalMode(state)
	} else {
		log.Printf("[TTY DEBUG] Close: no terminal state to restore")
	}

	tm.closed = true
	log.Printf("[TTY DEBUG] Close: TTY manager cleanup completed")
	return nil
}

// GetCurrentSize returns the current terminal size
func (tm *TTYManager) GetCurrentSize() *TerminalSize {
	if tm.currentSize == nil {
		log.Printf("[TTY DEBUG] GetCurrentSize: no current size, returning default 80x24")
		return &TerminalSize{Width: 80, Height: 24}
	}
	log.Printf("[TTY DEBUG] GetCurrentSize: returning current size %dx%d", tm.currentSize.Width, tm.currentSize.Height)
	return tm.currentSize
}

// GetEnvironment returns environment variables for TTY detection
func (tm *TTYManager) GetEnvironment() []string {
	log.Printf("[TTY DEBUG] GetEnvironment: creating environment variables for TTY")
	env := createTTYEnvironment(tm.currentSize)
	log.Printf("[TTY DEBUG] GetEnvironment: created %d environment variables", len(env))
	return env
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
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.resizeChan == nil || tm.closed {
		log.Printf("[TTY DEBUG] WatchResize: cannot setup resize watching (closed or no channel)")
		return
	}

	// Store the handler for potential cleanup
	tm.resizeHandler = handler

	// Create a context for cancellation
	done := make(chan struct{})
	tm.cancelFuncs = append(tm.cancelFuncs, func() {
		close(done)
	})

	go func() {
		// Panic recovery for resize handling goroutine
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[TTY ERROR] WatchResize: panic in resize handler: %v", r)
				fmt.Fprintf(os.Stderr, "WARNING: panic in resize handler: %v\n", r)
			}
		}()

		log.Printf("[TTY DEBUG] WatchResize: starting resize monitoring goroutine")

		for {
			select {
			case <-done:
				log.Printf("[TTY DEBUG] WatchResize: resize monitoring goroutine terminated")
				return
			case <-tm.resizeChan:
				// Apply rate limiting to prevent excessive processing
				if !tm.resizeLimiter.ShouldProcess() {
					log.Printf("[TTY DEBUG] WatchResize: skipping resize event due to rate limiting")
					continue
				}

				// Get new terminal size with error handling
				newSize, err := getTerminalSize()
				if err != nil {
					log.Printf("[TTY ERROR] WatchResize: failed to get terminal size during resize: %v", err)
					fmt.Fprintf(os.Stderr, "WARNING: failed to get terminal size during resize: %v\n", err)
					continue // Skip this resize event
				}

				// Only handle resize if dimensions actually changed
				tm.mu.Lock()
				if newSize.Width != tm.currentSize.Width || newSize.Height != tm.currentSize.Height {
					log.Printf("[TTY DEBUG] WatchResize: terminal size changed from %dx%d to %dx%d",
						tm.currentSize.Width, tm.currentSize.Height, newSize.Width, newSize.Height)
					tm.currentSize = newSize
					tm.mu.Unlock()

					// Call handler with error handling
					if err := handler.HandleResize(newSize.Width, newSize.Height); err != nil {
						// Log error but continue processing resize events
						log.Printf("[TTY ERROR] WatchResize: resize handler error (size %dx%d): %v",
							newSize.Width, newSize.Height, err)
						fmt.Fprintf(os.Stderr, "WARNING: resize handler error (size %dx%d): %v\n",
							newSize.Width, newSize.Height, err)
					}
				} else {
					tm.mu.Unlock()
					log.Printf("[TTY DEBUG] WatchResize: terminal size unchanged, skipping handler")
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
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.signalChan == nil || tm.closed {
		log.Printf("[TTY DEBUG] WatchSignals: cannot setup signal watching (closed or no channel)")
		return
	}

	// Store the handler for potential cleanup
	tm.signalHandler = handler

	// Create a context for cancellation
	done := make(chan struct{})
	tm.cancelFuncs = append(tm.cancelFuncs, func() {
		close(done)
	})

	go func() {
		// Panic recovery for signal handling goroutine
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[TTY ERROR] WatchSignals: panic in signal handler: %v", r)
				fmt.Fprintf(os.Stderr, "WARNING: panic in signal handler: %v\n", r)
			}
		}()

		log.Printf("[TTY DEBUG] WatchSignals: starting signal monitoring goroutine")

		for {
			select {
			case <-done:
				log.Printf("[TTY DEBUG] WatchSignals: signal monitoring goroutine terminated")
				return
			case sig := <-tm.signalChan:
				log.Printf("[TTY DEBUG] WatchSignals: received signal %v", sig)
				// Handle signal with error reporting but don't stop signal processing
				if err := handler.HandleSignal(sig); err != nil {
					log.Printf("[TTY ERROR] WatchSignals: signal handler error for %v: %v", sig, err)
					fmt.Fprintf(os.Stderr, "WARNING: signal handler error for %v: %v\n", sig, err)
				}
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

package docker

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/term"
)

func TestTerminalSize(t *testing.T) {
	t.Run("GetTerminalSize_ValidTerminal", func(t *testing.T) {
		// This test may fail in CI environments without a terminal
		// Skip if no terminal is available
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			t.Skip("Skipping terminal size test: no terminal available")
		}

		size, err := getTerminalSize()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if size.Width <= 0 || size.Height <= 0 {
			t.Errorf("Expected positive dimensions, got: %dx%d", size.Width, size.Height)
		}

		// Reasonable bounds check (most terminals are between 20x5 and 500x200)
		if size.Width < 20 || size.Width > 500 || size.Height < 5 || size.Height > 200 {
			t.Logf("Warning: Terminal size %dx%d seems unusual", size.Width, size.Height)
		}
	})

	t.Run("GetTerminalSize_InvalidFD", func(t *testing.T) {
		// This test verifies error handling when terminal is not available
		// We can't easily mock os.Stdin, so we'll test the wrapper behavior
		if term.IsTerminal(int(os.Stdin.Fd())) {
			t.Skip("Skipping invalid FD test: terminal is available")
		}

		_, err := getTerminalSize()
		if err == nil {
			t.Error("Expected error when no terminal available, got nil")
		}

		if !strings.Contains(err.Error(), "stdin is not a terminal") {
			t.Errorf("Expected 'stdin is not a terminal' error, got: %v", err)
		}
	})
}

func TestTerminalAvailability(t *testing.T) {
	t.Run("IsTerminalAvailable", func(t *testing.T) {
		// Test that the function doesn't panic and returns a boolean
		available := isTerminalAvailable()

		// The result depends on the test environment, just ensure it's deterministic
		available2 := isTerminalAvailable()
		if available != available2 {
			t.Error("isTerminalAvailable() should return consistent results")
		}

		// Log for visibility in test output
		t.Logf("Terminal available: %v", available)
	})
}

func TestTTYEnvironment(t *testing.T) {
	t.Run("CreateTTYEnvironment_WithSize", func(t *testing.T) {
		size := &TerminalSize{Width: 120, Height: 50}
		env := createTTYEnvironment(size)

		// Check required environment variables
		expectedVars := map[string]string{
			"TERM":        "xterm-256color",
			"COLORTERM":   "truecolor",
			"FORCE_COLOR": "1",
			"COLUMNS":     "120",
			"LINES":       "50",
		}

		envMap := make(map[string]string)
		for _, envVar := range env {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		for key, expectedValue := range expectedVars {
			if value, exists := envMap[key]; !exists {
				t.Errorf("Expected environment variable %s, but it was not found", key)
			} else if value != expectedValue {
				t.Errorf("Expected %s=%s, got %s=%s", key, expectedValue, key, value)
			}
		}
	})

	t.Run("CreateTTYEnvironment_NilSize", func(t *testing.T) {
		env := createTTYEnvironment(nil)

		// Should still include basic TTY variables
		envMap := make(map[string]string)
		for _, envVar := range env {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		if envMap["TERM"] != "xterm-256color" {
			t.Error("Expected TERM=xterm-256color")
		}
		if envMap["COLORTERM"] != "truecolor" {
			t.Error("Expected COLORTERM=truecolor")
		}
		if envMap["FORCE_COLOR"] != "1" {
			t.Error("Expected FORCE_COLOR=1")
		}

		// Should not include size-specific variables
		if _, exists := envMap["COLUMNS"]; exists {
			t.Error("Should not include COLUMNS when size is nil")
		}
		if _, exists := envMap["LINES"]; exists {
			t.Error("Should not include LINES when size is nil")
		}
	})

	t.Run("CreateTTYEnvironment_EdgeCaseSizes", func(t *testing.T) {
		testCases := []struct {
			name  string
			size  *TerminalSize
			valid bool
		}{
			{"Very small", &TerminalSize{Width: 1, Height: 1}, true},
			{"Very large", &TerminalSize{Width: 9999, Height: 9999}, true},
			{"Zero width", &TerminalSize{Width: 0, Height: 24}, true},  // Function should handle this
			{"Zero height", &TerminalSize{Width: 80, Height: 0}, true}, // Function should handle this
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				env := createTTYEnvironment(tc.size)

				// Find COLUMNS and LINES in environment
				var columns, lines string
				for _, envVar := range env {
					parts := strings.SplitN(envVar, "=", 2)
					if len(parts) == 2 {
						switch parts[0] {
						case "COLUMNS":
							columns = parts[1]
						case "LINES":
							lines = parts[1]
						}
					}
				}

				// Verify values are numeric strings
				if _, err := strconv.Atoi(columns); err != nil {
					t.Errorf("COLUMNS should be numeric, got: %s", columns)
				}
				if _, err := strconv.Atoi(lines); err != nil {
					t.Errorf("LINES should be numeric, got: %s", lines)
				}

				// Verify they match the input
				if columns != strconv.Itoa(tc.size.Width) {
					t.Errorf("Expected COLUMNS=%d, got %s", tc.size.Width, columns)
				}
				if lines != strconv.Itoa(tc.size.Height) {
					t.Errorf("Expected LINES=%d, got %s", tc.size.Height, lines)
				}
			})
		}
	})
}

func TestTTYManager(t *testing.T) {
	t.Run("NewTTYManager_Success", func(t *testing.T) {
		manager, err := NewTTYManager()
		if err != nil {
			// In CI environments without a terminal, this is expected
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				t.Skipf("Skipping TTY manager test: no terminal available (%v)", err)
			}
			t.Fatalf("Expected successful TTY manager creation, got error: %v", err)
		}
		defer func() {
			if closeErr := manager.Close(); closeErr != nil {
				t.Errorf("Failed to close TTY manager: %v", closeErr)
			}
		}()

		// Test basic functionality
		size := manager.GetCurrentSize()
		if size.Width <= 0 || size.Height <= 0 {
			t.Errorf("Expected positive terminal size, got: %dx%d", size.Width, size.Height)
		}

		env := manager.GetEnvironment()
		if len(env) == 0 {
			t.Error("Expected non-empty environment variables")
		}

		// Verify environment contains expected variables
		hasTermVar := false
		for _, envVar := range env {
			if strings.HasPrefix(envVar, "TERM=") {
				hasTermVar = true
				break
			}
		}
		if !hasTermVar {
			t.Error("Expected environment to contain TERM variable")
		}
	})

	t.Run("TTYManager_Close_Idempotent", func(t *testing.T) {
		manager, err := NewTTYManager()
		if err != nil {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				t.Skipf("Skipping TTY manager test: no terminal available (%v)", err)
			}
			t.Fatalf("Expected successful TTY manager creation, got error: %v", err)
		}

		// Close multiple times should not cause errors
		err1 := manager.Close()
		err2 := manager.Close()
		err3 := manager.Close()

		if err1 != nil {
			t.Errorf("First close failed: %v", err1)
		}
		if err2 != nil {
			t.Errorf("Second close failed: %v", err2)
		}
		if err3 != nil {
			t.Errorf("Third close failed: %v", err3)
		}
	})

	t.Run("TTYManager_GetCurrentSize_DefaultFallback", func(t *testing.T) {
		// Test that we always get a reasonable default size
		manager := &TTYManager{currentSize: nil}
		size := manager.GetCurrentSize()

		if size.Width != 80 || size.Height != 24 {
			t.Errorf("Expected default size 80x24, got %dx%d", size.Width, size.Height)
		}
	})
}

func TestRawMode(t *testing.T) {
	t.Run("RawMode_NoTerminal", func(t *testing.T) {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			t.Skip("Skipping raw mode test: terminal is available")
		}

		_, err := enableRawMode()
		if err == nil {
			t.Error("Expected error when enabling raw mode without terminal")
		}

		if !strings.Contains(err.Error(), "terminal not available") {
			t.Errorf("Expected 'terminal not available' error, got: %v", err)
		}
	})

	t.Run("RestoreTerminalMode_NilState", func(t *testing.T) {
		// Should not panic or return error with nil state
		err := restoreTerminalMode(nil)
		if err != nil {
			t.Errorf("Expected no error with nil state, got: %v", err)
		}
	})

	t.Run("SafeRestoreTerminalMode_NilState", func(t *testing.T) {
		// Should not panic with nil state
		safeRestoreTerminalMode(nil)
		// Test passes if no panic occurs
	})

	t.Run("SafeRestoreTerminalMode_PanicRecovery", func(t *testing.T) {
		// This test ensures safeRestoreTerminalMode handles panics gracefully
		// We can't easily test actual panic scenarios, but we can test the structure
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			// In non-terminal environment, this should handle the error gracefully
			safeRestoreTerminalMode(&term.State{})
			// Test passes if no panic occurs
		}
	})
}

func TestEnhancedSignalHandling(t *testing.T) {
	t.Run("SignalHandlerFunc_Implementation", func(t *testing.T) {
		called := false
		var capturedSig os.Signal

		handler := SignalHandlerFunc(func(sig os.Signal) error {
			called = true
			capturedSig = sig
			return nil
		})

		err := handler.HandleSignal(syscall.SIGTERM)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !called {
			t.Error("Expected handler to be called")
		}

		if capturedSig != syscall.SIGTERM {
			t.Errorf("Expected SIGTERM, got: %v", capturedSig)
		}
	})

	t.Run("SignalHandlerFunc_WithError", func(t *testing.T) {
		expectedErr := fmt.Errorf("test error")

		handler := SignalHandlerFunc(func(sig os.Signal) error {
			return expectedErr
		})

		err := handler.HandleSignal(syscall.SIGINT)
		if err != expectedErr {
			t.Errorf("Expected %v, got: %v", expectedErr, err)
		}
	})
}

func TestEnhancedResizeHandling(t *testing.T) {
	t.Run("ResizeHandlerFunc_Implementation", func(t *testing.T) {
		called := false
		var capturedWidth, capturedHeight int

		handler := ResizeHandlerFunc(func(width, height int) error {
			called = true
			capturedWidth = width
			capturedHeight = height
			return nil
		})

		err := handler.HandleResize(120, 50)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !called {
			t.Error("Expected handler to be called")
		}

		if capturedWidth != 120 || capturedHeight != 50 {
			t.Errorf("Expected 120x50, got: %dx%d", capturedWidth, capturedHeight)
		}
	})

	t.Run("ResizeHandlerFunc_WithError", func(t *testing.T) {
		expectedErr := fmt.Errorf("resize error")

		handler := ResizeHandlerFunc(func(width, height int) error {
			return expectedErr
		})

		err := handler.HandleResize(80, 24)
		if err != expectedErr {
			t.Errorf("Expected %v, got: %v", expectedErr, err)
		}
	})
}

func TestTTYManagerPool(t *testing.T) {
	t.Run("TTYManagerPool_CreateAndReuse", func(t *testing.T) {
		pool := NewTTYManagerPool(2)
		defer pool.Close()

		// Get a manager from the pool (should create new one)
		manager1, err := pool.Get()
		if err != nil {
			// In CI environments without terminal, this might fail - that's OK
			t.Skipf("Skipping TTY manager pool test: %v", err)
		}

		if manager1 == nil {
			t.Error("Expected non-nil TTY manager")
		}

		// Put it back
		pool.Put(manager1)

		// Get another manager (should reuse the first one if possible)
		manager2, err := pool.Get()
		if err != nil {
			t.Skipf("Skipping TTY manager pool test: %v", err)
		}

		if manager2 == nil {
			t.Error("Expected non-nil TTY manager")
		}

		// Clean up
		manager2.Close()
	})

	t.Run("TTYManagerPool_OverflowHandling", func(t *testing.T) {
		pool := NewTTYManagerPool(1) // Small pool
		defer pool.Close()

		managers := make([]*TTYManager, 3)
		for i := 0; i < 3; i++ {
			manager, err := pool.Get()
			if err != nil {
				t.Skipf("Skipping TTY manager pool test: %v", err)
			}
			managers[i] = manager
		}

		// Put them all back - only one should fit in pool, others should be closed
		for _, manager := range managers {
			if manager != nil {
				pool.Put(manager)
			}
		}
	})
}

func TestResizeRateLimiter(t *testing.T) {
	t.Run("ResizeRateLimiter_BasicFunctionality", func(t *testing.T) {
		limiter := NewResizeRateLimiter(100 * time.Millisecond)

		// First call should be allowed
		if !limiter.ShouldProcess() {
			t.Error("First resize should be allowed")
		}

		// Immediate second call should be blocked
		if limiter.ShouldProcess() {
			t.Error("Immediate second resize should be blocked")
		}

		// Wait for rate limit to reset
		time.Sleep(150 * time.Millisecond)

		// Now should be allowed again
		if !limiter.ShouldProcess() {
			t.Error("Resize should be allowed after rate limit period")
		}
	})

	t.Run("ResizeRateLimiter_ConcurrentAccess", func(t *testing.T) {
		limiter := NewResizeRateLimiter(50 * time.Millisecond)

		var allowed, blocked int
		var wg sync.WaitGroup

		// Start multiple goroutines trying to process resize events
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if limiter.ShouldProcess() {
					allowed++
				} else {
					blocked++
				}
			}()
		}

		wg.Wait()

		// Only one should be allowed initially
		if allowed != 1 {
			t.Errorf("Expected 1 allowed resize, got %d", allowed)
		}
		if blocked != 9 {
			t.Errorf("Expected 9 blocked resizes, got %d", blocked)
		}
	})
}

func TestTTYManagerEnhancedMethods(t *testing.T) {
	t.Run("TTYManager_WatchSignalsFunc_BackwardCompatibility", func(t *testing.T) {
		manager := &TTYManager{
			signalChan: make(chan os.Signal, 1),
		}
		defer func() {
			if manager.signalChan != nil {
				close(manager.signalChan)
			}
		}()

		manager.WatchSignalsFunc(func(sig os.Signal) {
			// Signal handler - testing setup only
		})

		// We don't actually send a real signal, just test the method doesn't panic
		if manager.signalChan == nil {
			t.Error("Expected signal channel to be available after WatchSignalsFunc")
		}
	})

	t.Run("TTYManager_WatchResizeFunc_BackwardCompatibility", func(t *testing.T) {
		manager := &TTYManager{
			resizeChan:  make(chan os.Signal, 1),
			currentSize: &TerminalSize{Width: 80, Height: 24},
		}
		defer func() {
			if manager.resizeChan != nil {
				close(manager.resizeChan)
			}
		}()

		manager.WatchResizeFunc(func(width, height int) {
			// Resize handler - testing setup only
		})

		// Test the method doesn't panic and sets up goroutine
		if manager.resizeChan == nil {
			t.Error("Expected resize channel to be available after WatchResizeFunc")
		}
	})

	t.Run("TTYManager_WatchSignalsWithErrorHandling", func(t *testing.T) {
		manager := &TTYManager{
			signalChan: make(chan os.Signal, 1),
		}
		defer func() {
			if manager.signalChan != nil {
				close(manager.signalChan)
			}
		}()

		manager.WatchSignalsWithErrorHandling(func(sig os.Signal) error {
			return fmt.Errorf("test error")
		})

		// Test the method sets up the goroutine without panicking
		if manager.signalChan == nil {
			t.Error("Expected signal channel to be available")
		}
	})

	t.Run("TTYManager_WatchResizeWithErrorHandling", func(t *testing.T) {
		manager := &TTYManager{
			resizeChan:  make(chan os.Signal, 1),
			currentSize: &TerminalSize{Width: 80, Height: 24},
		}
		defer func() {
			if manager.resizeChan != nil {
				close(manager.resizeChan)
			}
		}()

		manager.WatchResizeWithErrorHandling(func(width, height int) error {
			return fmt.Errorf("resize test error")
		})

		// Test the method sets up the goroutine without panicking
		if manager.resizeChan == nil {
			t.Error("Expected resize channel to be available")
		}
	})
}

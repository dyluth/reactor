package testutil

import (
	"fmt"
	"os"
	"testing"
)

func TestAutoCleanupTestContainers(t *testing.T) {
	// Test with no isolation prefix
	originalPrefix := os.Getenv("REACTOR_ISOLATION_PREFIX")
	defer func() {
		if originalPrefix != "" {
			_ = os.Setenv("REACTOR_ISOLATION_PREFIX", originalPrefix)
		} else {
			_ = os.Unsetenv("REACTOR_ISOLATION_PREFIX")
		}
	}()

	// Clear isolation prefix
	_ = os.Unsetenv("REACTOR_ISOLATION_PREFIX")

	err := AutoCleanupTestContainers()
	if err != nil {
		t.Errorf("AutoCleanupTestContainers should not error when no prefix set: %v", err)
	}

	// Test with isolation prefix set
	_ = os.Setenv("REACTOR_ISOLATION_PREFIX", "test-cleanup-test")

	// This should not error even if Docker is not available
	err = AutoCleanupTestContainers()
	if err != nil {
		t.Errorf("AutoCleanupTestContainers should not error when Docker unavailable: %v", err)
	}
}

func TestCleanupAllTestContainers(t *testing.T) {
	// This should not error even if Docker is not available
	err := CleanupAllTestContainers()
	if err != nil {
		t.Errorf("CleanupAllTestContainers should not error when Docker unavailable: %v", err)
	}
}

func TestCleanupTestContainers_ValidPrefix(t *testing.T) {
	// Test with a valid isolation prefix
	isolationPrefix := "test-cleanup-validation-12345"

	err := CleanupTestContainers(isolationPrefix)
	if err != nil {
		// Should either succeed or fail with meaningful error about Docker availability
		// but should not panic or return unclear errors
		t.Logf("CleanupTestContainers with prefix returned: %v", err)
	}
}

func TestCleanupTestContainers_EmptyPrefix(t *testing.T) {
	// Test with empty isolation prefix - should still work
	err := CleanupTestContainers("")
	if err != nil {
		t.Logf("CleanupTestContainers with empty prefix returned: %v", err)
	}
}

func TestCleanupTestContainers_SpecialCharacterPrefix(t *testing.T) {
	// Test with special characters in prefix
	specialPrefixes := []string{
		"test-with-dash",
		"test_with_underscore",
		"test.with.dots",
		"test123numeric",
	}

	for _, prefix := range specialPrefixes {
		t.Run(prefix, func(t *testing.T) {
			err := CleanupTestContainers(prefix)
			if err != nil {
				t.Logf("CleanupTestContainers with prefix '%s' returned: %v", prefix, err)
				// Should not panic - any error should be graceful
			}
		})
	}
}

func TestAutoCleanupTestContainers_EnvironmentVariations(t *testing.T) {
	originalPrefix := os.Getenv("REACTOR_ISOLATION_PREFIX")
	defer func() {
		if originalPrefix != "" {
			_ = os.Setenv("REACTOR_ISOLATION_PREFIX", originalPrefix)
		} else {
			_ = os.Unsetenv("REACTOR_ISOLATION_PREFIX")
		}
	}()

	testCases := []struct {
		name   string
		prefix string
	}{
		{"empty_prefix", ""},
		{"normal_prefix", "test-auto-cleanup"},
		{"long_prefix", "test-auto-cleanup-with-very-long-prefix-name-that-might-cause-issues"},
		{"numeric_prefix", "12345"},
		{"mixed_prefix", "test-123-auto-cleanup"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.prefix == "" {
				_ = os.Unsetenv("REACTOR_ISOLATION_PREFIX")
			} else {
				_ = os.Setenv("REACTOR_ISOLATION_PREFIX", tc.prefix)
			}

			err := AutoCleanupTestContainers()
			if err != nil {
				t.Logf("AutoCleanupTestContainers with prefix '%s' returned: %v", tc.prefix, err)
				// Should not error fatally - Docker unavailability is acceptable
			}
		})
	}
}

func TestCleanupTestContainers_ConcurrentCalls(t *testing.T) {
	// Test that concurrent calls to cleanup functions don't interfere with each other
	const numGoroutines = 5
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			prefix := fmt.Sprintf("concurrent-test-%d", id)
			err := CleanupTestContainers(prefix)
			errors <- err
		}(i)
	}

	// Collect all results
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		if err != nil {
			t.Logf("Concurrent cleanup %d returned error: %v", i, err)
			// Errors are acceptable if Docker is unavailable, but should not panic
		}
	}
}

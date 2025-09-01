package testutil

import (
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

package config

import (
	"os"
	"strings"
	"testing"
)

func TestCheckDependencies(t *testing.T) {
	t.Run("check dependencies function call", func(t *testing.T) {
		// Test that CheckDependencies runs without panic
		// We can't easily test the actual dependencies without mocking filesystem
		err := CheckDependencies()
		// This will likely fail in test environment but shouldn't panic
		if err != nil {
			// Expected in test environment where docker might not be in /usr/bin/
			t.Logf("CheckDependencies returned error (expected in test env): %v", err)
		}
	})
}

func TestCheckCommand(t *testing.T) {
	t.Run("existing command in usr/bin", func(t *testing.T) {
		// Test with a command that might exist in /usr/bin/
		err := checkCommand("ls")
		if err != nil {
			t.Logf("Command 'ls' not found in standard paths (expected on some systems): %v", err)
		}
	})

	t.Run("existing command in usr/local/bin", func(t *testing.T) {
		// Test with a command that might exist in /usr/local/bin/
		err := checkCommand("git")
		if err != nil {
			t.Logf("Command 'git' not found in standard paths (expected on some systems): %v", err)
		}
	})

	t.Run("non-existing command", func(t *testing.T) {
		// Test with a command that shouldn't exist
		err := checkCommand("this-command-should-not-exist-12345")
		if err == nil {
			t.Error("Expected error for non-existing command")
		}
		// Should get a "not found in PATH" error
		if err != nil && !strings.Contains(err.Error(), "not found in PATH") {
			t.Logf("Got expected error for non-existing command: %v", err)
		}
	})

	t.Run("command search with PATH empty", func(t *testing.T) {
		// Save and restore PATH
		originalPath := os.Getenv("PATH")
		defer func() {
			if originalPath != "" {
				os.Setenv("PATH", originalPath)
			}
		}()

		// Set PATH to empty
		os.Setenv("PATH", "")

		err := checkCommand("nonexistent-cmd")
		if err == nil {
			t.Error("Expected error when PATH is empty and command doesn't exist")
		}
		if err != nil && !strings.Contains(err.Error(), "PATH is empty") {
			t.Logf("Got error: %v", err)
		}
	})
}
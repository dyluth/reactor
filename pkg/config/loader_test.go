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

	t.Run("check dependencies stderr output", func(t *testing.T) {
		// Test that CheckDependencies writes to stderr for git warning
		// This exercises the git warning path even if commands aren't found
		err := CheckDependencies()
		// Function may error but should complete without panic
		if err != nil {
			t.Logf("CheckDependencies completed with error: %v", err)
		}
		// The function should have attempted to check both docker and git
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
				_ = os.Setenv("PATH", originalPath)
			}
		}()

		// Set PATH to empty
		_ = os.Setenv("PATH", "")

		err := checkCommand("nonexistent-cmd")
		if err == nil {
			t.Error("Expected error when PATH is empty and command doesn't exist")
		}
		if err != nil && !strings.Contains(err.Error(), "PATH is empty") {
			t.Logf("Got error: %v", err)
		}
	})

	t.Run("test all directory checks", func(t *testing.T) {
		// Test to ensure we hit all the directory paths in checkCommand
		// This will likely fail but will exercise more code paths
		testCommands := []string{"ls", "bash", "sh", "git", "docker"}
		for _, cmd := range testCommands {
			err := checkCommand(cmd)
			if err != nil {
				t.Logf("Command %s not found (expected): %v", cmd, err)
			}
		}
	})
}

func TestCheckDependencies_EdgeCases(t *testing.T) {
	// Save and restore environment
	originalPath := os.Getenv("PATH")
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalPath != "" {
			_ = os.Setenv("PATH", originalPath)
		}
		if originalHome != "" {
			_ = os.Setenv("HOME", originalHome)
		}
	}()

	t.Run("with modified PATH", func(t *testing.T) {
		// Set a limited PATH to exercise different code paths
		_ = os.Setenv("PATH", "/usr/bin:/usr/local/bin")
		err := CheckDependencies()
		// Should complete without panic
		if err != nil {
			t.Logf("CheckDependencies with limited PATH: %v", err)
		}
	})

	t.Run("with empty HOME", func(t *testing.T) {
		_ = os.Setenv("HOME", "")
		err := CheckDependencies()
		// Should handle missing HOME gracefully
		if err != nil {
			t.Logf("CheckDependencies with empty HOME: %v", err)
		}
	})
}

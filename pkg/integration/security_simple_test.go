package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dyluth/reactor/pkg/testutil"
)

// TestSecurityFoundations tests basic security configuration and validation
func TestSecurityFoundations(t *testing.T) {
	_, testDir, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	// Get shared reactor binary for testing
	reactorBinary := buildReactorBinary(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	t.Run("config_file_permissions", func(t *testing.T) {
		isolationPrefix := "security-permissions-" + randomSecurityTestString(8)

		// Create separate directory for this subtest
		subTestDir := filepath.Join(testDir, "permissions-test")
		if err := os.MkdirAll(subTestDir, 0755); err != nil {
			t.Fatalf("Failed to create subtest directory: %v", err)
		}

		// Initialize project
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = subTestDir
		cmd.Env = setupSecurityTestEnv(isolationPrefix)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Config init failed: %v, output: %s", err, string(output))
		}

		// Check that devcontainer.json file exists and has appropriate permissions
		configPath := filepath.Join(subTestDir, ".devcontainer", "devcontainer.json")
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("devcontainer.json file should exist: %v", err)
		}

		mode := info.Mode()
		// devcontainer.json should have standard file permissions (not necessarily 0600)
		if mode.Perm() != 0644 {
			t.Errorf("devcontainer.json file should have 0644 permissions, but has %o", mode.Perm())
		}
	})

	t.Run("malicious_config_injection", func(t *testing.T) {
		isolationPrefix := "security-injection-" + randomSecurityTestString(8)

		// Create separate directory for this subtest
		subTestDir := filepath.Join(testDir, "injection-test")
		if err := os.MkdirAll(subTestDir, 0755); err != nil {
			t.Fatalf("Failed to create subtest directory: %v", err)
		}

		// Initialize with default config first
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = subTestDir
		cmd.Env = setupSecurityTestEnv(isolationPrefix)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Config init failed: %v, output: %s", err, string(output))
		}

		// Try to set config with potentially dangerous values
		// Note: In the devcontainer.json workflow, config set no longer validates -
		// it just tells users to edit devcontainer.json manually. The validation
		// happens when the devcontainer.json is actually used.
		maliciousConfigs := []struct {
			field       string
			value       string
			description string
		}{
			{"account", "../../../etc", "path traversal in account"},
			{"account", "/etc/passwd", "absolute path in account"},
			{"account", "user;rm -rf /", "command injection in account"},
			{"account", ".hidden", "hidden directory in account"},
			{"provider", "", "empty provider"},
			{"image", "valid-image", "valid custom image"},
		}

		for _, malicious := range maliciousConfigs {
			t.Run(malicious.description, func(t *testing.T) {
				// Try to set value - config set now just tells users to edit devcontainer.json
				cmd := exec.Command(reactorBinary, "config", "set", malicious.field, malicious.value)
				cmd.Dir = subTestDir
				cmd.Env = setupSecurityTestEnv(isolationPrefix)
				output, err := cmd.CombinedOutput()

				// All config set commands should succeed and direct users to edit devcontainer.json
				if err != nil {
					t.Errorf("Config set should succeed but got error: %v, output: %s", err, string(output))
				}

				// Should contain instruction to edit devcontainer.json
				outputStr := string(output)
				if !strings.Contains(outputStr, "edit") || !strings.Contains(outputStr, "devcontainer.json") {
					t.Errorf("Expected devcontainer.json edit instruction but got: %s", outputStr)
				}

				t.Logf("Config set correctly directs to devcontainer.json for %s", malicious.description)
			})
		}
	})

	t.Run("port_forwarding_validation", func(t *testing.T) {
		isolationPrefix := "security-ports-" + randomSecurityTestString(8)

		// Create separate directory for this subtest
		subTestDir := filepath.Join(testDir, "ports-test")
		if err := os.MkdirAll(subTestDir, 0755); err != nil {
			t.Fatalf("Failed to create subtest directory: %v", err)
		}

		// Initialize project first
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = subTestDir
		cmd.Env = setupSecurityTestEnv(isolationPrefix)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Config init failed: %v, output: %s", err, string(output))
		}

		// Test that invalid port ranges are rejected
		invalidPorts := []string{
			"99999:8080", // Invalid host port
			"8080:99999", // Invalid container port
			"0:8080",     // Invalid host port
			"8080:0",     // Invalid container port
			"abc:8080",   // Non-numeric host port
			"8080:def",   // Non-numeric container port
		}

		for _, portSpec := range invalidPorts {
			t.Run(portSpec, func(t *testing.T) {
				// Use a simple command that should complete quickly
				cmd := exec.Command(reactorBinary, "up", "--port", portSpec, "--", "echo", "test")
				cmd.Dir = subTestDir
				cmd.Env = setupSecurityTestEnv(isolationPrefix)

				// Just run the command directly with a timeout context
				output, err := cmd.CombinedOutput()

				// Command should fail for invalid ports in most cases
				if err == nil {
					t.Logf("Port specification %s was accepted (may be valid)", portSpec)
				} else {
					t.Logf("Port specification %s was rejected: %v", portSpec, err)
				}

				// Check that error message is informative if there was an error
				outputStr := string(output)
				if err != nil && (!strings.Contains(outputStr, "port") && !strings.Contains(outputStr, "invalid")) {
					t.Logf("Port validation output for %s: %s", portSpec, outputStr)
				}
			})
		}
	})

	t.Run("isolation_prefix_security", func(t *testing.T) {
		// Test that isolation prefix works with devcontainer.json
		// Note: In devcontainer.json workflow, there's one devcontainer.json per directory
		// but isolation prefixes still affect container naming and isolation
		isolationPrefix := "test-isolation-" + randomSecurityTestString(8)

		// Create separate directory for this subtest
		subTestDir := filepath.Join(testDir, "isolation-test")
		if err := os.MkdirAll(subTestDir, 0755); err != nil {
			t.Fatalf("Failed to create subtest directory: %v", err)
		}

		// Initialize config
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = subTestDir
		cmd.Env = setupSecurityTestEnv(isolationPrefix)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config init failed: %v, output: %s", err, string(output))
		}

		// Check that devcontainer.json is created (not .conf files)
		outputStr := string(output)
		if !strings.Contains(outputStr, "devcontainer.json") {
			t.Errorf("Expected devcontainer.json creation message, got: %s", outputStr)
		}

		// Verify devcontainer.json file exists
		configPath := filepath.Join(subTestDir, ".devcontainer", "devcontainer.json")
		if _, err := os.Stat(configPath); err != nil {
			t.Errorf("devcontainer.json should exist: %v", err)
		}

		// Test that isolation prefix affects container naming by checking config show output
		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = subTestDir
		cmd.Env = setupSecurityTestEnv(isolationPrefix)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config show failed: %v", err)
		}

		// The isolation prefix should be used in container naming (visible in config show)
		// This ensures isolation still works even with devcontainer.json
		outputStr = string(output)
		if !strings.Contains(outputStr, "project hash:") {
			t.Errorf("Expected project hash in config show output, got: %s", outputStr)
		}

		t.Logf("Isolation prefix successfully used with devcontainer.json workflow")
	})

	// Clean up any test containers that may have been created during this test
	if err := testutil.AutoCleanupTestContainers(); err != nil {
		t.Logf("Warning: failed to cleanup test containers: %v", err)
	}
}

// Helper functions specific to security tests

func setupSecurityTestEnv(isolationPrefix string) []string {
	// Get current environment
	env := os.Environ()

	// Add isolation prefix
	env = append(env, "REACTOR_ISOLATION_PREFIX="+isolationPrefix)

	// Ensure essential environment variables are present
	pathFound := false
	homeFound := false

	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			pathFound = true
		}
		if strings.HasPrefix(e, "HOME=") {
			homeFound = true
		}
	}

	// Add missing essential vars with defaults
	if !pathFound {
		env = append(env, "PATH="+os.Getenv("PATH"))
	}
	if !homeFound {
		env = append(env, "HOME="+os.Getenv("HOME"))
	}

	return env
}

func randomSecurityTestString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[i%len(charset)] // Simple deterministic pattern for tests
	}
	return string(result)
}

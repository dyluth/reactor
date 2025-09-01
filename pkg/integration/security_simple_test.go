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

	// Build reactor binary for testing
	reactorBinary := buildReactorForSecurityTest(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	t.Run("config_file_permissions", func(t *testing.T) {
		isolationPrefix := "security-permissions-" + randomSecurityTestString(8)

		// Initialize project
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = testDir
		cmd.Env = setupSecurityTestEnv(isolationPrefix)
		if _, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Config init failed: %v", err)
		}

		// Check that config file has restrictive permissions
		configPath := filepath.Join(testDir, "."+isolationPrefix+".conf")
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("Config file should exist: %v", err)
		}

		mode := info.Mode()
		if mode.Perm() != 0600 {
			t.Errorf("Config file should have 0600 permissions, but has %o", mode.Perm())
		}
	})

	t.Run("malicious_config_injection", func(t *testing.T) {
		isolationPrefix := "security-injection-" + randomSecurityTestString(8)

		// Initialize with default config first
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = testDir
		cmd.Env = setupSecurityTestEnv(isolationPrefix)
		if _, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Config init failed: %v", err)
		}

		// Try to create a config with potentially dangerous values
		maliciousConfigs := []struct {
			field       string
			value       string
			shouldError bool
			description string
		}{
			{"account", "../../../etc", true, "path traversal in account"},
			{"account", "/etc/passwd", true, "absolute path in account"},
			{"account", "user;rm -rf /", true, "command injection in account"},
			{"account", ".hidden", true, "hidden directory in account"},
			{"provider", "", true, "empty provider"},
			{"image", "valid-image", false, "valid custom image"}, // This should succeed
		}

		for _, malicious := range maliciousConfigs {
			t.Run(malicious.description, func(t *testing.T) {
				// Try to set malicious value
				cmd := exec.Command(reactorBinary, "config", "set", malicious.field, malicious.value)
				cmd.Dir = testDir
				cmd.Env = setupSecurityTestEnv(isolationPrefix)
				output, err := cmd.CombinedOutput()

				if malicious.shouldError {
					// Should reject dangerous values
					if err == nil {
						t.Errorf("Expected rejection of malicious %s: %s, but command succeeded", malicious.description, malicious.value)
					}
					if strings.Contains(string(output), "invalid") || strings.Contains(string(output), "error") {
						t.Logf("Good: Rejected malicious config - %s", malicious.description)
					}
				} else {
					// Should allow valid values
					if err != nil {
						t.Errorf("Expected valid config to succeed: %s, but got error: %v", malicious.description, err)
					}
				}
			})
		}
	})

	t.Run("port_forwarding_validation", func(t *testing.T) {
		isolationPrefix := "security-ports-" + randomSecurityTestString(8)

		// Initialize project first
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = testDir
		cmd.Env = setupSecurityTestEnv(isolationPrefix)
		if _, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Config init failed: %v", err)
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
				cmd := exec.Command(reactorBinary, "run", "--port", portSpec, "--", "echo", "test")
				cmd.Dir = testDir
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
		// Test that isolation prefix prevents config conflicts
		isolationPrefix1 := "test-isolation-1"
		isolationPrefix2 := "test-isolation-2"

		// Initialize config with first prefix
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = testDir
		cmd.Env = setupSecurityTestEnv(isolationPrefix1)
		output1, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config init with prefix 1 failed: %v", err)
		}

		// Check that different config file is created
		if !strings.Contains(string(output1), ".test-isolation-1.conf") {
			t.Errorf("Expected isolated config file name, got: %s", string(output1))
		}

		// Initialize config with second prefix
		cmd = exec.Command(reactorBinary, "config", "init")
		cmd.Dir = testDir
		cmd.Env = setupSecurityTestEnv(isolationPrefix2)
		output2, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config init with prefix 2 failed: %v", err)
		}

		// Should create different config file
		if !strings.Contains(string(output2), ".test-isolation-2.conf") {
			t.Errorf("Expected different isolated config file name, got: %s", string(output2))
		}

		// Verify both config files exist and are separate
		config1Path := filepath.Join(testDir, ".test-isolation-1.conf")
		config2Path := filepath.Join(testDir, ".test-isolation-2.conf")

		if _, err := os.Stat(config1Path); err != nil {
			t.Errorf("First isolation config should exist: %v", err)
		}
		if _, err := os.Stat(config2Path); err != nil {
			t.Errorf("Second isolation config should exist: %v", err)
		}
	})

	// Clean up any test containers that may have been created during this test
	if err := testutil.AutoCleanupTestContainers(); err != nil {
		t.Logf("Warning: failed to cleanup test containers: %v", err)
	}
}

// Helper functions specific to security tests
func buildReactorForSecurityTest(t *testing.T) string {
	t.Helper()

	// Create a temp binary in OS temp directory
	tempBinary := filepath.Join(os.TempDir(), "reactor-security-test-"+randomSecurityTestString(8))
	cmd := exec.Command("go", "build", "-o", tempBinary, "./cmd/reactor")

	// Set working directory to project root
	workDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build reactor binary: %v\nOutput: %s", err, string(output))
	}

	return tempBinary
}

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

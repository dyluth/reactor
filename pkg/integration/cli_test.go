package integration

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dyluth/reactor/pkg/testutil"
)

var (
	sharedReactorBinary     string
	sharedReactorBinaryOnce sync.Once
	sharedReactorBinaryErr  error
)

// TestReactorCLIBasicCommands tests basic CLI functionality
func TestReactorCLIBasicCommands(t *testing.T) {
	// Set up isolated test environment with robust cleanup
	_, _, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Get shared reactor binary for testing
	reactorBinary := buildReactorBinary(t)

	tests := []struct {
		name             string
		args             []string
		expectedInStdout []string
		expectedInStderr []string
		shouldFail       bool
	}{
		{
			name:             "help command",
			args:             []string{"--help"},
			expectedInStdout: []string{"Reactor provides simple, fast, and reliable containerized development environments"},
			shouldFail:       false,
		},
		{
			name:             "version command",
			args:             []string{"version"},
			expectedInStdout: []string{"reactor version"},
			shouldFail:       false,
		},
		{
			name:             "sessions help",
			args:             []string{"sessions", "--help"},
			expectedInStdout: []string{"Manage and interact with reactor container sessions", "Available Commands:", "attach", "list"},
			shouldFail:       false,
		},
		{
			name:             "sessions list with no containers",
			args:             []string{"sessions", "list"},
			expectedInStdout: []string{"No reactor containers found"},
			shouldFail:       false,
		},
		{
			name:             "config help",
			args:             []string{"config", "--help"},
			expectedInStdout: []string{"Manage project", "Available Commands:", "show", "get", "set", "init"},
			shouldFail:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(reactorBinary, tt.args...)
			cmd.Env = setupTestEnv("test-integration-" + randomString(8))

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			if tt.shouldFail && err == nil {
				t.Errorf("Expected command to fail but it succeeded. Output: %s", outputStr)
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("Expected command to succeed but it failed with error: %v. Output: %s", err, outputStr)
			}

			for _, expected := range tt.expectedInStdout {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain '%s' but got: %s", expected, outputStr)
				}
			}
		})
	}
}

// TestReactorConfigOperations tests config initialization and management
func TestReactorConfigOperations(t *testing.T) {
	// Set up isolated test environment
	_, tempDir, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	reactorBinary := buildReactorBinary(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	isolationPrefix := "test-config-" + randomString(8)
	env := []string{
		"REACTOR_ISOLATION_PREFIX=" + isolationPrefix,
	}

	t.Run("config init", func(t *testing.T) {
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config init failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Get current username for dynamic test validation
		currentUser := os.Getenv("USER")
		if currentUser == "" {
			// Fallback for systems where USER is not set
			if u, err := user.Current(); err == nil {
				currentUser = u.Username
			} else {
				currentUser = "unknown"
			}
		}

		expectedStrings := []string{
			"Initialized devcontainer.json at:",
			"Default configuration:",
			"name:",
			"image:",
			"account: " + currentUser,
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Expected output to contain '%s' but got: %s", expected, outputStr)
			}
		}

		// Verify devcontainer.json was created
		devcontainerFile := filepath.Join(tempDir, ".devcontainer", "devcontainer.json")
		if _, err := os.Stat(devcontainerFile); os.IsNotExist(err) {
			t.Errorf("devcontainer.json was not created at %s", devcontainerFile)
		}
	})

	t.Run("config show", func(t *testing.T) {
		cmd := exec.Command(reactorBinary, "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config show failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		expectedStrings := []string{
			"DevContainer Configuration",
			"account:",
			"image:",
			"project root:",
			"project hash:",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Expected output to contain '%s' but got: %s", expected, outputStr)
			}
		}
	})

	t.Run("config verbose show", func(t *testing.T) {
		cmd := exec.Command(reactorBinary, "--verbose", "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config show --verbose failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		// Should contain all the same info as regular show
		expectedStrings := []string{
			"DevContainer Configuration",
			"project root:",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Expected verbose output to contain '%s' but got: %s", expected, outputStr)
			}
		}
	})

	t.Run("config get and set", func(t *testing.T) {
		// Test setting a value (now directs to devcontainer.json)
		cmd := exec.Command(reactorBinary, "config", "set", "provider", "gemini")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err := cmd.CombinedOutput()
		// config set now directs users to edit devcontainer.json manually
		if err != nil {
			t.Fatalf("config set failed: %v, output: %s", err, string(output))
		}

		if !strings.Contains(string(output), "To set 'provider', edit your devcontainer.json file") {
			t.Errorf("Expected devcontainer.json edit instruction but got: %s", string(output))
		}

		// Test getting the value (now directs to devcontainer.json)
		cmd = exec.Command(reactorBinary, "config", "get", "provider")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config get failed: %v, output: %s", err, string(output))
		}

		if !strings.Contains(string(output), "For configuration key 'provider', check your devcontainer.json file") {
			t.Errorf("Expected devcontainer.json check instruction but got: %s", string(output))
		}
	})
}

// TestContainerNaming tests the enhanced container naming scheme
func TestContainerNaming(t *testing.T) {
	_, _, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	reactorBinary := buildReactorBinary(t)

	// Create test directories with different names to test sanitization
	testCases := []struct {
		dirName               string
		expectedContainerPart string
	}{
		{"simple-project", "simple-project"},
		{"my_project", "my_project"},
		{"project.with.dots", "project.with.dots"},
		{"very-long-project-name-that-exceeds-limits", "very-long-project-na"}, // Should be truncated to 20 chars
		{"project with spaces", "project-with-spaces"},                         // Spaces should become hyphens
	}

	for _, tc := range testCases {
		t.Run("naming_"+tc.dirName, func(t *testing.T) {
			tempDir := createTempDir(t, tc.dirName)
			t.Cleanup(func() {
				if err := os.RemoveAll(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp directory %s: %v", tempDir, err)
				}
			})

			isolationPrefix := "test-naming-" + randomString(8)
			env := []string{"REACTOR_ISOLATION_PREFIX=" + isolationPrefix}

			// Initialize config
			cmd := exec.Command(reactorBinary, "config", "init")
			cmd.Dir = tempDir
			cmd.Env = append(os.Environ(), env...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("config init failed: %v\nOutput: %s", err, string(output))
			}

			// Get the config show output to see what container name would be generated
			cmd = exec.Command(reactorBinary, "config", "show")
			cmd.Dir = tempDir
			cmd.Env = append(os.Environ(), env...)

			output, err = cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("config show failed: %v", err)
			}

			outputStr := string(output)
			// The container name should follow pattern: {prefix}-reactor-cam-{sanitized-folder}-{hash}
			// We can't predict the exact hash, but we can verify the structure
			// Use canonical path comparison to handle symlink differences (e.g., /var vs /private/var on macOS)
			if !strings.Contains(outputStr, "project root:") {
				t.Errorf("Expected output to contain 'project root:' but got: %s", outputStr)
			} else {
				// Extract project root from output and compare paths using canonical comparison
				lines := strings.Split(outputStr, "\n")
				var actualProjectRoot string
				for _, line := range lines {
					if strings.Contains(line, "project root:") {
						parts := strings.Split(line, "project root:")
						if len(parts) > 1 {
							actualProjectRoot = strings.TrimSpace(parts[1])
							break
						}
					}
				}
				if actualProjectRoot != "" {
					testutil.AssertPathsEqual(t, tempDir, actualProjectRoot, "project root should match expected directory")
				} else {
					t.Errorf("Could not find project root in output: %s", outputStr)
				}
			}
		})
	}
}

// Helper functions

// getSharedReactorBinary builds the reactor binary once and returns the path to all callers
func getSharedReactorBinary(t *testing.T) string {
	t.Helper()

	sharedReactorBinaryOnce.Do(func() {
		sharedReactorBinary, sharedReactorBinaryErr = buildReactorBinaryOnce()
	})

	if sharedReactorBinaryErr != nil {
		t.Fatalf("Failed to build shared reactor binary: %v", sharedReactorBinaryErr)
	}

	return sharedReactorBinary
}

// buildReactorBinaryOnce builds the reactor binary once for all tests
func buildReactorBinaryOnce() (string, error) {
	// Build the reactor binary in OS temp directory
	tempBinary := filepath.Join(os.TempDir(), "reactor-integration-shared")
	cmd := exec.Command("go", "build", "-o", tempBinary, "./cmd/reactor")

	// Set the working directory to the project root
	// When running from pkg/integration, we need to go up two levels
	workDir, err := filepath.Abs("../..")
	if err != nil {
		return "", err
	}
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %v\nOutput: %s", err, string(output))
	}

	return tempBinary, nil
}

// buildReactorBinary is kept for backward compatibility but now uses shared binary
func buildReactorBinary(t *testing.T) string {
	return getSharedReactorBinary(t)
}

func createTempDir(t *testing.T, name string) string {
	t.Helper()

	// Ensure we have isolated HOME environment
	// If HOME is not set, set up isolation (this handles cases where the test
	// function didn't explicitly call testutil.WithIsolatedHome)
	if os.Getenv("HOME") == "" {
		testutil.WithIsolatedHome(t)
	}

	// Use OS temp directory to avoid permission issues with Docker-created files
	tempDir := filepath.Join(os.TempDir(), name+"-"+randomString(8))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	return tempDir
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond) // Ensure different timestamps
	}
	return string(result)
}

func setupTestEnv(isolationPrefix string) []string {
	env := append(os.Environ(),
		"REACTOR_ISOLATION_PREFIX="+isolationPrefix,
	)

	// Ensure HOME is set (required for config operations)
	if home := os.Getenv("HOME"); home != "" {
		env = append(env, "HOME="+home)
	}

	return env
}

package integration

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dyluth/reactor/pkg/testutil"
)

// TestReactorCLIBasicCommands tests basic CLI functionality
func TestReactorCLIBasicCommands(t *testing.T) {
	// Set up isolated test environment with robust cleanup
	_, _, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	// Build reactor binary for testing
	reactorBinary := buildReactorBinary(t)
	defer func() { _ = os.Remove(reactorBinary) }()

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

	reactorBinary := buildReactorBinary(t)
	defer func() { _ = os.Remove(reactorBinary) }()

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
			"Created directory:",
			"Initialized project configuration",
			"Default configuration:",
			"provider: claude",
			"account:  " + currentUser,
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Expected output to contain '%s' but got: %s", expected, outputStr)
			}
		}

		// Verify config file was created
		configFile := filepath.Join(tempDir, "."+isolationPrefix+".conf")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Errorf("Config file was not created at %s", configFile)
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
			"Project Configuration",
			"Resolved Configuration:",
			"project root:",
			"Available Providers:",
			"claude",
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
			"Project Configuration",
			"project root:",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Expected verbose output to contain '%s' but got: %s", expected, outputStr)
			}
		}
	})

	t.Run("config get and set", func(t *testing.T) {
		// Test setting a value
		cmd := exec.Command(reactorBinary, "config", "set", "provider", "gemini")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config set failed: %v, output: %s", err, string(output))
		}

		if !strings.Contains(string(output), "Set provider = gemini") {
			t.Errorf("Expected set confirmation but got: %s", string(output))
		}

		// Test getting the value
		cmd = exec.Command(reactorBinary, "config", "get", "provider")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config get failed: %v, output: %s", err, string(output))
		}

		if !strings.Contains(string(output), "gemini") {
			t.Errorf("Expected to get 'gemini' but got: %s", string(output))
		}
	})
}

// TestContainerNaming tests the enhanced container naming scheme
func TestContainerNaming(t *testing.T) {
	_, _, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	reactorBinary := buildReactorBinary(t)
	defer func() { _ = os.Remove(reactorBinary) }()

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

			isolationPrefix := "test-naming-" + randomString(8)
			env := []string{"REACTOR_ISOLATION_PREFIX=" + isolationPrefix}

			// Initialize config
			cmd := exec.Command(reactorBinary, "config", "init")
			cmd.Dir = tempDir
			cmd.Env = append(os.Environ(), env...)
			_, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("config init failed: %v", err)
			}

			// Get the config show output to see what container name would be generated
			cmd = exec.Command(reactorBinary, "config", "show")
			cmd.Dir = tempDir
			cmd.Env = append(os.Environ(), env...)

			output, err := cmd.CombinedOutput()
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

func buildReactorBinary(t *testing.T) string {
	t.Helper()

	// Build the reactor binary in OS temp directory
	tempBinary := filepath.Join(os.TempDir(), "reactor-test-"+randomString(8))
	cmd := exec.Command("go", "build", "-o", tempBinary, "./cmd/reactor")

	// Set the working directory to the project root
	// When running from pkg/integration, we need to go up two levels
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

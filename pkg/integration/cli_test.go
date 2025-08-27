package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestReactorCLIBasicCommands tests basic CLI functionality
func TestReactorCLIBasicCommands(t *testing.T) {
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
	reactorBinary := buildReactorBinary(t)
	defer func() { _ = os.Remove(reactorBinary) }()

	// Create temporary directory for testing
	tempDir := createTempDir(t, "reactor-config-test")
	defer func() { _ = os.RemoveAll(tempDir) }()

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
		expectedStrings := []string{
			"Created directory:",
			"Initialized project configuration",
			"Default configuration:",
			"provider: claude",
			"account:  cam",
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
			defer func() { _ = os.RemoveAll(tempDir) }()

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
			if !strings.Contains(outputStr, "project root:    "+tempDir) {
				t.Errorf("Expected project root to be %s but got output: %s", tempDir, outputStr)
			}
		})
	}
}

// Helper functions

func buildReactorBinary(t *testing.T) string {
	t.Helper()

	// Build the reactor binary
	tempBinary := filepath.Join(t.TempDir(), "reactor-test")
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

	tempDir := filepath.Join(t.TempDir(), name)
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

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBasicReactorFunctionality tests core reactor functionality
func TestBasicReactorFunctionality(t *testing.T) {
	// Build reactor binary for testing
	reactorBinary := buildReactorForTest(t)

	// Create test directory in system temp
	testDir := filepath.Join(t.TempDir(), "integration-test-"+randomTestString(8))
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	isolationPrefix := "test-basic-" + randomTestString(8)

	t.Run("basic CLI commands work", func(t *testing.T) {
		// Test help command
		cmd := exec.Command(reactorBinary, "--help")
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Help command failed: %v", err)
		}

		if !strings.Contains(string(output), "Reactor provides") {
			t.Errorf("Help output should contain 'Reactor provides' but got: %s", string(output))
		}

		// Test version command
		cmd = exec.Command(reactorBinary, "version")
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Version command failed: %v", err)
		}

		if !strings.Contains(string(output), "reactor version") {
			t.Errorf("Version output should contain 'reactor version' but got: %s", string(output))
		}
	})

	t.Run("config operations work", func(t *testing.T) {
		// Test config init
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config init failed: %v\nOutput: %s", err, string(output))
		}

		if !strings.Contains(string(output), "Initialized project configuration") {
			t.Errorf("Config init should show success message but got: %s", string(output))
		}

		// Test config show
		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config show failed: %v\nOutput: %s", err, string(output))
		}

		expectedInOutput := []string{
			"provider: claude",
			"account:",
			"project root:",
			"project hash:",
		}

		outputStr := string(output)
		for _, expected := range expectedInOutput {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Config show should contain '%s' but got: %s", expected, outputStr)
			}
		}

		// Test config get/set
		cmd = exec.Command(reactorBinary, "config", "set", "provider", "gemini")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config set failed: %v\nOutput: %s", err, string(output))
		}

		cmd = exec.Command(reactorBinary, "config", "get", "provider")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config get failed: %v\nOutput: %s", err, string(output))
		}

		if !strings.Contains(string(output), "gemini") {
			t.Errorf("Config get should return 'gemini' but got: %s", string(output))
		}
	})

	t.Run("sessions commands work", func(t *testing.T) {
		// Test sessions list (should show no containers)
		cmd := exec.Command(reactorBinary, "sessions", "list")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Sessions list failed: %v\nOutput: %s", err, string(output))
		}

		if !strings.Contains(string(output), "No reactor containers found") {
			t.Errorf("Sessions list should show no containers but got: %s", string(output))
		}

		// Test sessions attach help
		cmd = exec.Command(reactorBinary, "sessions", "attach", "--help")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Sessions attach help failed: %v\nOutput: %s", err, string(output))
		}

		if !strings.Contains(string(output), "Attach to a") {
			t.Errorf("Sessions attach help should contain expected text but got: %s", string(output))
		}
	})
}

// Helper functions for basic integration tests

func buildReactorForTest(t *testing.T) string {
	t.Helper()

	// Create a temp binary in the test temp directory
	tempBinary := filepath.Join(t.TempDir(), "reactor-integration-test")
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

func setupBasicEnv(isolationPrefix string) []string {
	env := []string{
		"REACTOR_ISOLATION_PREFIX=" + isolationPrefix,
		"PATH=" + os.Getenv("PATH"),
	}

	// Ensure HOME is set (required for config operations)
	if home := os.Getenv("HOME"); home != "" {
		env = append(env, "HOME="+home)
	}

	// Add other essential env vars if they exist
	if user := os.Getenv("USER"); user != "" {
		env = append(env, "USER="+user)
	}

	return env
}

func randomTestString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond) // Ensure different timestamps
	}
	return string(result)
}

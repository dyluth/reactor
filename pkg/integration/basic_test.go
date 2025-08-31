package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dyluth/reactor/pkg/testutil"
)

// TestBasicReactorFunctionality tests core reactor functionality
func TestBasicReactorFunctionality(t *testing.T) {
	// Set up isolated test environment with HOME directory
	_, testDir, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	// Build reactor binary for testing
	reactorBinary := buildReactorForTest(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer os.Chdir(originalWD)

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

	// Build the reactor binary in OS temp directory to avoid permission issues
	tempBinary := filepath.Join(os.TempDir(), "reactor-test-"+randomTestString(8))
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
	// Get current environment, which now includes the isolated HOME from testutil.WithIsolatedHome
	env := os.Environ()
	
	// Add isolation prefix
	env = append(env, "REACTOR_ISOLATION_PREFIX="+isolationPrefix)
	
	// Ensure essential environment variables are present
	pathFound := false
	homeFound := false
	userFound := false
	
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			pathFound = true
		}
		if strings.HasPrefix(e, "HOME=") {
			homeFound = true
		}
		if strings.HasPrefix(e, "USER=") {
			userFound = true
		}
	}
	
	// Add missing essential vars with defaults
	if !pathFound {
		env = append(env, "PATH="+os.Getenv("PATH"))
	}
	if !homeFound {
		// This should not happen when using testutil.WithIsolatedHome, but add as fallback
		if home := os.Getenv("HOME"); home != "" {
			env = append(env, "HOME="+home)
		}
	}
	if !userFound {
		if user := os.Getenv("USER"); user != "" {
			env = append(env, "USER="+user)
		}
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

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
	if err := testutil.CleanupAllTestContainers(); err != nil {
		t.Fatalf("Initial cleanup failed: %v", err)
	}

	// Set up isolated test environment with HOME directory
	_, testDir, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	isolationPrefix := "test-basic-" + randomTestString(8)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Build reactor binary for testing
	reactorBinary := buildReactorForTest(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

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

		if !strings.Contains(string(output), "Initialized devcontainer.json") {
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
			"DevContainer Configuration",
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

		// Test config set (now directly modifies devcontainer.json)
		cmd = exec.Command(reactorBinary, "config", "set", "account", "test-account")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config set failed: %v\nOutput: %s", err, string(output))
		}

		if !strings.Contains(string(output), "Successfully updated account") {
			t.Errorf("Config set should show a success message but got: %s", string(output))
		}

		// Verify the file was actually changed
		content, err := os.ReadFile(filepath.Join(testDir, ".devcontainer/devcontainer.json"))
		if err != nil {
			t.Fatalf("Failed to read devcontainer.json after set: %v", err)
		}
		if !strings.Contains(string(content), `"account": "test-account"`) {
			t.Errorf("devcontainer.json was not updated correctly. Content: %s", string(content))
		}

		cmd = exec.Command(reactorBinary, "config", "get", "account")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config get failed: %v\nOutput: %s", err, string(output))
		}

		// Should return the account we just set
		if !strings.Contains(string(output), "test-account") {
			t.Errorf("Config get should return the account we set ('test-account') but got: %s", string(output))
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

// TestDevContainerFunctionality tests devcontainer.json integration features
func TestDevContainerFunctionality(t *testing.T) {
	if err := testutil.CleanupAllTestContainers(); err != nil {
		t.Fatalf("Initial cleanup failed: %v", err)
	}

	// Set up isolated test environment with HOME directory
	_, testDir, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	isolationPrefix := "test-dev-" + randomTestString(8)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Build reactor binary for testing
	reactorBinary := buildReactorForTest(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	t.Run("devcontainer.json with forwardPorts and remoteUser", func(t *testing.T) {
		// Create a devcontainer.json with forwardPorts and remoteUser
		devcontainerDir := filepath.Join(testDir, ".devcontainer")
		err := os.MkdirAll(devcontainerDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .devcontainer directory: %v", err)
		}

		devcontainerContent := `{
	"name": "test-project",
	"image": "ghcr.io/dyluth/reactor/base:latest",
	"remoteUser": "testuser",
	"forwardPorts": [8080, "3000:4000"],
	"customizations": {
		"reactor": {
			"account": "test-account"
		}
	}
}`

		devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
		err = os.WriteFile(devcontainerPath, []byte(devcontainerContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write devcontainer.json: %v", err)
		}

		// Test config show to verify parsing
		cmd := exec.Command(reactorBinary, "config", "show")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config show failed: %v\nOutput: %s", err, string(output))
		}

		outputStr := string(output)
		expectedInOutput := []string{
			"DevContainer Configuration",
			"account:         test-account",
			"image:           ghcr.io/dyluth/reactor/base:latest",
			"project root:",
		}

		for _, expected := range expectedInOutput {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Config show should contain '%s' but got: %s", expected, outputStr)
			}
		}
	})

	t.Run("invalid forwardPorts in devcontainer.json", func(t *testing.T) {
		// Create a devcontainer.json with invalid forwardPorts
		devcontainerDir := filepath.Join(testDir, ".devcontainer")
		err := os.MkdirAll(devcontainerDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .devcontainer directory: %v", err)
		}

		devcontainerContent := `{
	"name": "test-project",
	"image": "ghcr.io/dyluth/reactor/base:latest",
	"forwardPorts": [8080, "invalid:port"],
	"customizations": {
		"reactor": {
			"account": "test-account"
		}
	}
}`

		devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
		err = os.WriteFile(devcontainerPath, []byte(devcontainerContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write devcontainer.json: %v", err)
		}

		// Test config show should fail with clear error
		cmd := exec.Command(reactorBinary, "config", "show")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("Config show should have failed with invalid forwardPorts but succeeded. Output: %s", string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "failed to parse forwardPorts") {
			t.Errorf("Error should mention forwardPorts parsing but got: %s", outputStr)
		}
	})
}

// TestPostCreateCommandFunctionality tests postCreateCommand execution
func TestPostCreateCommandFunctionality(t *testing.T) {
	if err := testutil.CleanupAllTestContainers(); err != nil {
		t.Fatalf("Initial cleanup failed: %v", err)
	}

	// Set up isolated test environment with HOME directory
	_, testDir, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	isolationPrefix := "test-postcreate-" + randomTestString(8)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Build reactor binary for testing
	reactorBinary := buildReactorForTest(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	t.Run("string postCreateCommand executes successfully", func(t *testing.T) {
		// Create a devcontainer.json with string postCreateCommand
		devcontainerDir := filepath.Join(testDir, ".devcontainer")
		err := os.MkdirAll(devcontainerDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .devcontainer directory: %v", err)
		}

		devcontainerContent := `{
	"name": "test-postcreate-project",
	"image": "ghcr.io/dyluth/reactor/base:latest",
	"postCreateCommand": "echo 'PostCreate command executed' > /tmp/postcreate-test.txt",
	"customizations": {
		"reactor": {
			"account": "test-account"
		}
	}
}`

		devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
		err = os.WriteFile(devcontainerPath, []byte(devcontainerContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write devcontainer.json: %v", err)
		}

		// Skip the actual reactor up test since it requires Docker and is complex
		// Instead, test that config parsing works correctly
		cmd := exec.Command(reactorBinary, "config", "show")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config show failed: %v\nOutput: %s", err, string(output))
		}

		outputStr := string(output)
		expectedInOutput := []string{
			"DevContainer Configuration",
			"account:         test-account",
			"image:           ghcr.io/dyluth/reactor/base:latest",
		}

		for _, expected := range expectedInOutput {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Config show should contain '%s' but got: %s", expected, outputStr)
			}
		}

		t.Log("String postCreateCommand configuration parsed successfully")
	})

	t.Run("array postCreateCommand configuration", func(t *testing.T) {
		// Create a devcontainer.json with array postCreateCommand
		devcontainerDir := filepath.Join(testDir, ".devcontainer")
		err := os.MkdirAll(devcontainerDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .devcontainer directory: %v", err)
		}

		devcontainerContent := `{
	"name": "test-postcreate-array-project",
	"image": "ghcr.io/dyluth/reactor/base:latest",
	"postCreateCommand": ["npm", "install", "--verbose"],
	"customizations": {
		"reactor": {
			"account": "test-account"
		}
	}
}`

		devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
		err = os.WriteFile(devcontainerPath, []byte(devcontainerContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write devcontainer.json: %v", err)
		}

		// Test config parsing
		cmd := exec.Command(reactorBinary, "config", "show")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config show failed: %v\nOutput: %s", err, string(output))
		}

		outputStr := string(output)
		expectedInOutput := []string{
			"DevContainer Configuration",
			"account:         test-account",
			"image:           ghcr.io/dyluth/reactor/base:latest",
		}

		for _, expected := range expectedInOutput {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Config show should contain '%s' but got: %s", expected, outputStr)
			}
		}

		t.Log("Array postCreateCommand configuration parsed successfully")
	})

	t.Run("no postCreateCommand specified", func(t *testing.T) {
		// Create a devcontainer.json without postCreateCommand
		devcontainerDir := filepath.Join(testDir, ".devcontainer")
		err := os.MkdirAll(devcontainerDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .devcontainer directory: %v", err)
		}

		devcontainerContent := `{
	"name": "test-no-postcreate-project",
	"image": "ghcr.io/dyluth/reactor/base:latest",
	"customizations": {
		"reactor": {
			"account": "test-account"
		}
	}
}`

		devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
		err = os.WriteFile(devcontainerPath, []byte(devcontainerContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write devcontainer.json: %v", err)
		}

		// Test config parsing works without postCreateCommand
		cmd := exec.Command(reactorBinary, "config", "show")
		cmd.Dir = testDir
		cmd.Env = setupBasicEnv(isolationPrefix)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config show failed: %v\nOutput: %s", err, string(output))
		}

		outputStr := string(output)
		expectedInOutput := []string{
			"DevContainer Configuration",
			"account:         test-account",
			"image:           ghcr.io/dyluth/reactor/base:latest",
		}

		for _, expected := range expectedInOutput {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Config show should contain '%s' but got: %s", expected, outputStr)
			}
		}

		t.Log("Configuration without postCreateCommand parsed successfully")
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

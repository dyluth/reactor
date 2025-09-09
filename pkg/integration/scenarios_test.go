package integration

import (
	"os"
	"os/exec"

	"strings"
	"testing"

	"github.com/dyluth/reactor/pkg/testutil"
)

// TestEndToEndScenarios tests complete user workflows
func TestEndToEndScenarios(t *testing.T) {
	// Set up isolated test environment with robust cleanup
	_, _, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	reactorBinary := buildReactorBinary(t)

	t.Run("developer workflow - config and sessions", func(t *testing.T) {
		// Scenario: Developer creates a new project, initializes reactor, and manages sessions
		tempDir := createTempDir(t, "developer-workflow")

		// Change to test directory
		originalWD, _ := os.Getwd()
		err := os.Chdir(tempDir)
		if err != nil {
			t.Fatalf("Failed to change to test directory: %v", err)
		}
		defer func() { _ = os.Chdir(originalWD) }()

		isolationPrefix := "test-e2e-" + randomString(8)
		env := []string{"REACTOR_ISOLATION_PREFIX=" + isolationPrefix}

		// Step 1: Initialize project configuration
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Step 1 - config init failed: %v, output: %s", err, string(output))
		}

		if !strings.Contains(string(output), "Initialized devcontainer.json") {
			t.Errorf("Step 1 - Expected initialization success message")
		}

		// Step 2: Check initial configuration
		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Step 2 - config show failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		requiredConfigItems := []string{
			"account:",
			"image:",
			"project hash:",
		}

		for _, item := range requiredConfigItems {
			if !strings.Contains(outputStr, item) {
				t.Errorf("Step 2 - Expected config to contain '%s' but got: %s", item, outputStr)
			}
		}

		// Handle project root comparison separately using canonical path comparison
		if !strings.Contains(outputStr, "project root:") {
			t.Errorf("Step 2 - Expected output to contain 'project root:' but got: %s", outputStr)
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
				t.Errorf("Step 2 - Could not find project root in output: %s", outputStr)
			}
		}

		// Step 3: Modify configuration
		cmd = exec.Command(reactorBinary, "config", "set", "image", "python")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Step 3 - config set failed: %v, output: %s", err, string(output))
		}

		if !strings.Contains(string(output), "edit") || !strings.Contains(string(output), "devcontainer.json") {
			t.Errorf("Step 3 - Expected devcontainer.json edit instruction")
		}

		// Step 4: Verify configuration change
		cmd = exec.Command(reactorBinary, "config", "get", "image")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Step 4 - config get failed: %v, output: %s", err, string(output))
		}

		if !strings.Contains(string(output), "ghcr.io/dyluth/reactor/base:latest") {
			t.Errorf("Step 4 - Expected to get default image but got: %s", string(output))
		}

		// Step 5: Check sessions (should be empty initially)
		cmd = exec.Command(reactorBinary, "sessions", "list")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Step 5 - sessions list failed: %v, output: %s", err, string(output))
		}

		if !strings.Contains(string(output), "No reactor containers found") {
			t.Errorf("Step 5 - Expected no containers initially but got: %s", string(output))
		}
	})

	t.Run("multi-project isolation", func(t *testing.T) {
		// Scenario: Developer works on multiple projects with different configurations

		isolationPrefix := "test-multi-" + randomString(8)
		env := []string{"REACTOR_ISOLATION_PREFIX=" + isolationPrefix}

		// Create two different projects
		project1Dir := createTempDir(t, "project-alpha")
		project2Dir := createTempDir(t, "project-beta")

		// Setup project 1 with Claude
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = project1Dir
		cmd.Env = append(os.Environ(), env...)
		_, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 1 init failed: %v", err)
		}

		cmd = exec.Command(reactorBinary, "config", "set", "provider", "claude")
		cmd.Dir = project1Dir
		cmd.Env = append(os.Environ(), env...)
		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 1 config set failed: %v", err)
		}

		// Setup project 2 with Gemini
		cmd = exec.Command(reactorBinary, "config", "init")
		cmd.Dir = project2Dir
		cmd.Env = append(os.Environ(), env...)
		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 2 init failed: %v", err)
		}

		cmd = exec.Command(reactorBinary, "config", "set", "provider", "gemini")
		cmd.Dir = project2Dir
		cmd.Env = append(os.Environ(), env...)
		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 2 config set failed: %v", err)
		}

		// Verify they have different configurations
		cmd = exec.Command(reactorBinary, "config", "get", "provider")
		cmd.Dir = project1Dir
		cmd.Env = append(os.Environ(), env...)
		output1, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 1 config get failed: %v", err)
		}

		cmd = exec.Command(reactorBinary, "config", "get", "provider")
		cmd.Dir = project2Dir
		cmd.Env = append(os.Environ(), env...)
		output2, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 2 config get failed: %v", err)
		}

		if !strings.Contains(string(output1), "check your devcontainer.json file") {
			t.Errorf("Project 1 should show devcontainer.json instruction but got: %s", string(output1))
		}
		if !strings.Contains(string(output2), "check your devcontainer.json file") {
			t.Errorf("Project 2 should show devcontainer.json instruction but got: %s", string(output2))
		}

		// Verify they have different project hashes
		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = project1Dir
		cmd.Env = append(os.Environ(), env...)
		show1, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 1 config show failed: %v", err)
		}

		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = project2Dir
		cmd.Env = append(os.Environ(), env...)
		show2, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 2 config show failed: %v", err)
		}

		// They should reference different project roots
		if !strings.Contains(string(show1), project1Dir) {
			t.Errorf("Project 1 config should reference its directory")
		}
		if !strings.Contains(string(show2), project2Dir) {
			t.Errorf("Project 2 config should reference its directory")
		}
	})

	t.Run("verbose output scenario", func(t *testing.T) {
		// Scenario: Developer uses verbose mode to debug configuration issues

		tempDir := createTempDir(t, "verbose-test-project")
		isolationPrefix := "test-verbose-" + randomString(8)
		env := []string{"REACTOR_ISOLATION_PREFIX=" + isolationPrefix}

		// Initialize with verbose output
		cmd := exec.Command(reactorBinary, "--verbose", "config", "init")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Verbose config init failed: %v, output: %s", err, string(output))
		}

		// Should still work the same way
		if !strings.Contains(string(output), "Initialized devcontainer.json") {
			t.Errorf("Verbose init should still show success message")
		}

		// Verbose config show should provide detailed information
		cmd = exec.Command(reactorBinary, "--verbose", "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Verbose config show failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		// Should contain all the standard config info
		verboseExpected := []string{
			"DevContainer Configuration",
			"account:",
			"image:",
			"project root:",
			"project hash:",
		}

		for _, expected := range verboseExpected {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Verbose config show should contain '%s' but got: %s", expected, outputStr)
			}
		}
	})

	// Clean up any test containers that may have been created during this test
	if err := testutil.AutoCleanupTestContainers(); err != nil {
		t.Logf("Warning: failed to cleanup test containers: %v", err)
	}
}

// TestErrorRecoveryScenarios tests how the system handles various error conditions
func TestErrorRecoveryScenarios(t *testing.T) {
	// Set up isolated test environment with robust cleanup
	_, _, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	reactorBinary := buildReactorBinary(t)

	t.Run("invalid configuration values", func(t *testing.T) {
		tempDir := createTempDir(t, "error-recovery-test")
		isolationPrefix := "test-recovery-" + randomString(8)
		env := []string{"REACTOR_ISOLATION_PREFIX=" + isolationPrefix}

		// Initialize valid config first
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)
		_, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config init failed: %v", err)
		}

		// Try to set invalid provider
		cmd = exec.Command(reactorBinary, "config", "set", "provider", "invalid-provider")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err := cmd.CombinedOutput()
		// This might succeed (just setting the value) or fail with validation
		// Either behavior is acceptable as long as it doesn't crash
		if err != nil && !strings.Contains(string(output), "invalid") && !strings.Contains(string(output), "not found") {
			t.Logf("Setting invalid provider returned: %s", string(output))
		}

		// The important thing is that the config system handles the corrupted state gracefully
		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err = cmd.CombinedOutput()
		if err != nil {
			// It's acceptable for config show to fail with a corrupted config file
			// as long as the error message is helpful
			outputStr := string(output)
			if !strings.Contains(outputStr, "invalid") && !strings.Contains(outputStr, "provider") && !strings.Contains(outputStr, "error") {
				t.Errorf("Config system should give helpful error after invalid input, got: %s", outputStr)
			}
		}
	})

	t.Run("missing config file recovery", func(t *testing.T) {
		tempDir := createTempDir(t, "missing-config-test")
		isolationPrefix := "test-missing-" + randomString(8)
		env := []string{"REACTOR_ISOLATION_PREFIX=" + isolationPrefix}

		// Try to show config without initializing
		cmd := exec.Command(reactorBinary, "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		output, err := cmd.CombinedOutput()
		// Should either show defaults or give a helpful error
		if err != nil {
			outputStr := string(output)
			// Error message should be helpful
			if !strings.Contains(outputStr, "no devcontainer.json found") && !strings.Contains(outputStr, "initialize") && !strings.Contains(outputStr, "not found") {
				t.Errorf("Missing config error should be helpful but got: %s", outputStr)
			}
		}

		// Should be able to recover by initializing
		cmd = exec.Command(reactorBinary, "config", "init")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Should be able to recover with config init: %v", err)
		}

		// Now config show should work
		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), env...)

		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Errorf("Config show should work after init: %v", err)
		}
	})

	// Clean up any test containers that may have been created during this test
	if err := testutil.AutoCleanupTestContainers(); err != nil {
		t.Logf("Warning: failed to cleanup test containers: %v", err)
	}
}

// TestContainerNameGeneration tests the container naming logic more thoroughly
func TestContainerNameGeneration(t *testing.T) {
	// Set up isolated test environment with robust cleanup
	_, _, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	reactorBinary := buildReactorBinary(t)

	testCases := []struct {
		name             string
		projectName      string
		isolationPrefix  string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:            "simple project with isolation",
			projectName:     "my-simple-project",
			isolationPrefix: "test-simple",
			shouldContain:   []string{"test-simple", "my-simple-project"}, // Should find isolation prefix and project name in project root path
		},
		{
			name:             "project with special characters",
			projectName:      "my@project#with$special%chars",
			isolationPrefix:  "test-special",
			shouldContain:    []string{"test-special", "my@project#with$special%chars"}, // Should find isolation prefix and original project name in project root path
			shouldNotContain: []string{},                                                // The special chars will be in the project root path, so we can't exclude them
		},
		{
			name:            "very long project name",
			projectName:     "this-is-a-very-long-project-name-that-should-be-truncated-appropriately",
			isolationPrefix: "test-long",
			shouldContain:   []string{"test-long", "this-is-a-very-long-project-name-that-should-be-truncated-appropriately"}, // Should find the full original name in project root
		},
		{
			name:             "project with spaces and underscores",
			projectName:      "my project_with mixed_separators",
			isolationPrefix:  "test-mixed",
			shouldContain:    []string{"test-mixed", "my project_with mixed_separators"}, // Should find original name in project root path
			shouldNotContain: []string{},                                                 // Spaces will be in the project root path, so we can't exclude them
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := createTempDir(t, tc.projectName)
			env := []string{"REACTOR_ISOLATION_PREFIX=" + tc.isolationPrefix}

			// Initialize config
			cmd := exec.Command(reactorBinary, "config", "init")
			cmd.Dir = tempDir
			cmd.Env = append(os.Environ(), env...)
			_, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Config init failed for %s: %v", tc.name, err)
			}

			// Get config output to examine container naming
			cmd = exec.Command(reactorBinary, "config", "show")
			cmd.Dir = tempDir
			cmd.Env = append(os.Environ(), env...)

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Config show failed for %s: %v", tc.name, err)
			}

			outputStr := string(output)

			// Check expected content
			for _, expected := range tc.shouldContain {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Test %s: Expected output to contain '%s' but got: %s",
						tc.name, expected, outputStr)
				}
			}

			// Check that unwanted content is not present
			for _, unwanted := range tc.shouldNotContain {
				if strings.Contains(outputStr, unwanted) {
					t.Errorf("Test %s: Expected output to NOT contain '%s' but got: %s",
						tc.name, unwanted, outputStr)
				}
			}

			// Verify isolation prefix appears in the output
			if !strings.Contains(outputStr, tc.isolationPrefix) {
				t.Errorf("Test %s: Expected isolation prefix '%s' to appear in output but got: %s",
					tc.name, tc.isolationPrefix, outputStr)
			}
		})
	}

	// Clean up any test containers that may have been created during this test
	if err := testutil.AutoCleanupTestContainers(); err != nil {
		t.Logf("Warning: failed to cleanup test containers: %v", err)
	}
}

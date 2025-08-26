package integration

import (
	"os/exec"
	"strings"
	"testing"
)

// TestEndToEndScenarios tests complete user workflows
func TestEndToEndScenarios(t *testing.T) {
	reactorBinary := buildReactorBinary(t)

	t.Run("developer workflow - config and sessions", func(t *testing.T) {
		// Scenario: Developer creates a new project, initializes reactor, and manages sessions
		tempDir := createTempDir(t, "dev-workflow-project")
		isolationPrefix := "test-e2e-" + randomString(8)
		env := []string{"REACTOR_ISOLATION_PREFIX=" + isolationPrefix}

		// Step 1: Initialize project configuration
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, env...)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Step 1 - config init failed: %v, output: %s", err, string(output))
		}

		if !strings.Contains(string(output), "Initialized project configuration") {
			t.Errorf("Step 1 - Expected initialization success message")
		}

		// Step 2: Check initial configuration
		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, env...)
		
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Step 2 - config show failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		requiredConfigItems := []string{
			"provider: claude",
			"account:  cam", 
			"project root:    " + tempDir,
			"project hash:",
		}

		for _, item := range requiredConfigItems {
			if !strings.Contains(outputStr, item) {
				t.Errorf("Step 2 - Expected config to contain '%s' but got: %s", item, outputStr)
			}
		}

		// Step 3: Modify configuration
		cmd = exec.Command(reactorBinary, "config", "set", "image", "python")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, env...)
		
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Step 3 - config set failed: %v, output: %s", err, string(output))
		}

		if !strings.Contains(string(output), "Set image = python") {
			t.Errorf("Step 3 - Expected set confirmation")
		}

		// Step 4: Verify configuration change
		cmd = exec.Command(reactorBinary, "config", "get", "image")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, env...)
		
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Step 4 - config get failed: %v, output: %s", err, string(output))
		}

		if !strings.Contains(string(output), "python") {
			t.Errorf("Step 4 - Expected to get 'python' but got: %s", string(output))
		}

		// Step 5: Check sessions (should be empty initially)
		cmd = exec.Command(reactorBinary, "sessions", "list")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, env...)
		
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
		cmd.Env = append(cmd.Env, env...)
		_, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 1 init failed: %v", err)
		}

		cmd = exec.Command(reactorBinary, "config", "set", "provider", "claude")
		cmd.Dir = project1Dir
		cmd.Env = append(cmd.Env, env...)
		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 1 config set failed: %v", err)
		}

		// Setup project 2 with Gemini
		cmd = exec.Command(reactorBinary, "config", "init")
		cmd.Dir = project2Dir
		cmd.Env = append(cmd.Env, env...)
		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 2 init failed: %v", err)
		}

		cmd = exec.Command(reactorBinary, "config", "set", "provider", "gemini")
		cmd.Dir = project2Dir
		cmd.Env = append(cmd.Env, env...)
		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 2 config set failed: %v", err)
		}

		// Verify they have different configurations
		cmd = exec.Command(reactorBinary, "config", "get", "provider")
		cmd.Dir = project1Dir
		cmd.Env = append(cmd.Env, env...)
		output1, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 1 config get failed: %v", err)
		}

		cmd = exec.Command(reactorBinary, "config", "get", "provider")
		cmd.Dir = project2Dir
		cmd.Env = append(cmd.Env, env...)
		output2, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 2 config get failed: %v", err)
		}

		if !strings.Contains(string(output1), "claude") {
			t.Errorf("Project 1 should have claude provider but got: %s", string(output1))
		}
		if !strings.Contains(string(output2), "gemini") {
			t.Errorf("Project 2 should have gemini provider but got: %s", string(output2))
		}

		// Verify they have different project hashes
		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = project1Dir
		cmd.Env = append(cmd.Env, env...)
		show1, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Project 1 config show failed: %v", err)
		}

		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = project2Dir
		cmd.Env = append(cmd.Env, env...)
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
		cmd.Env = append(cmd.Env, env...)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Verbose config init failed: %v, output: %s", err, string(output))
		}

		// Should still work the same way
		if !strings.Contains(string(output), "Initialized project configuration") {
			t.Errorf("Verbose init should still show success message")
		}

		// Verbose config show should provide detailed information
		cmd = exec.Command(reactorBinary, "--verbose", "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, env...)
		
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Verbose config show failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		// Should contain all the standard config info
		verboseExpected := []string{
			"Project Configuration",
			"Resolved Configuration:", 
			"project root:",
			"project hash:",
			"Available Providers:",
			"Available Images:",
		}

		for _, expected := range verboseExpected {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Verbose config show should contain '%s' but got: %s", expected, outputStr)
			}
		}
	})
}

// TestErrorRecoveryScenarios tests how the system handles various error conditions
func TestErrorRecoveryScenarios(t *testing.T) {
	reactorBinary := buildReactorBinary(t)

	t.Run("invalid configuration values", func(t *testing.T) {
		tempDir := createTempDir(t, "error-recovery-test")
		isolationPrefix := "test-recovery-" + randomString(8)
		env := []string{"REACTOR_ISOLATION_PREFIX=" + isolationPrefix}

		// Initialize valid config first
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, env...)
		_, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Config init failed: %v", err)
		}

		// Try to set invalid provider
		cmd = exec.Command(reactorBinary, "config", "set", "provider", "invalid-provider")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, env...)
		
		output, err := cmd.CombinedOutput()
		// This might succeed (just setting the value) or fail with validation
		// Either behavior is acceptable as long as it doesn't crash
		if err != nil && !strings.Contains(string(output), "invalid") && !strings.Contains(string(output), "not found") {
			t.Logf("Setting invalid provider returned: %s", string(output))
		}

		// The important thing is that the config system remains functional
		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, env...)
		
		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Errorf("Config system should remain functional after invalid input: %v", err)
		}
	})

	t.Run("missing config file recovery", func(t *testing.T) {
		tempDir := createTempDir(t, "missing-config-test")
		isolationPrefix := "test-missing-" + randomString(8)
		env := []string{"REACTOR_ISOLATION_PREFIX=" + isolationPrefix}

		// Try to show config without initializing
		cmd := exec.Command(reactorBinary, "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, env...)
		
		output, err := cmd.CombinedOutput()
		// Should either show defaults or give a helpful error
		if err != nil {
			outputStr := string(output)
			// Error message should be helpful
			if !strings.Contains(outputStr, "not found") && !strings.Contains(outputStr, "initialize") {
				t.Errorf("Missing config error should be helpful but got: %s", outputStr)
			}
		}

		// Should be able to recover by initializing
		cmd = exec.Command(reactorBinary, "config", "init")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, env...)
		
		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Should be able to recover with config init: %v", err)
		}

		// Now config show should work
		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, env...)
		
		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Errorf("Config show should work after init: %v", err)
		}
	})
}

// TestContainerNameGeneration tests the container naming logic more thoroughly
func TestContainerNameGeneration(t *testing.T) {
	reactorBinary := buildReactorBinary(t)

	testCases := []struct {
		name            string
		projectName     string
		isolationPrefix string
		shouldContain   []string
		shouldNotContain []string
	}{
		{
			name:            "simple project with isolation",
			projectName:     "my-simple-project", 
			isolationPrefix: "test-simple",
			shouldContain:   []string{"my-simple-project"},
		},
		{
			name:            "project with special characters",
			projectName:     "my@project#with$special%chars",
			isolationPrefix: "test-special",
			shouldContain:   []string{"my-project-with-special-chars"}, // Should be sanitized
			shouldNotContain: []string{"@", "#", "$", "%"},
		},
		{
			name:            "very long project name",
			projectName:     "this-is-a-very-long-project-name-that-should-be-truncated-appropriately",
			isolationPrefix: "test-long",
			shouldContain:   []string{"this-is-a-very-long"}, // Should be truncated
		},
		{
			name:            "project with spaces and underscores",
			projectName:     "my project_with mixed_separators",
			isolationPrefix: "test-mixed",
			shouldContain:   []string{"my-project_with-mixed"}, // Spaces become hyphens
			shouldNotContain: []string{" "},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := createTempDir(t, tc.projectName)
			env := []string{"REACTOR_ISOLATION_PREFIX=" + tc.isolationPrefix}

			// Initialize config
			cmd := exec.Command(reactorBinary, "config", "init")
			cmd.Dir = tempDir
			cmd.Env = append(cmd.Env, env...)
			_, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Config init failed for %s: %v", tc.name, err)
			}

			// Get config output to examine container naming
			cmd = exec.Command(reactorBinary, "config", "show")
			cmd.Dir = tempDir
			cmd.Env = append(cmd.Env, env...)
			
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
}
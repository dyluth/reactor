package integration

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
)

// TestDockerIntegration tests Docker-dependent functionality
// This test requires Docker to be running
func TestDockerIntegration(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping Docker integration tests")
	}

	reactorBinary := buildReactorBinary(t)

	t.Run("docker health check", func(t *testing.T) {
		// This should pass if Docker is running
		cmd := exec.Command(reactorBinary, "sessions", "list")
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX=test-docker-"+randomString(8))
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("sessions list failed when Docker should be available: %v, output: %s", err, string(output))
		}

		// Should show "No reactor containers found" since we haven't created any
		if !strings.Contains(string(output), "No reactor containers found") {
			t.Errorf("Expected 'No reactor containers found' but got: %s", string(output))
		}
	})

	t.Run("container discovery patterns", func(t *testing.T) {
		isolationPrefix := "test-discovery-" + randomString(8)
		
		// Test that the container discovery recognizes the correct naming patterns
		cmd := exec.Command(reactorBinary, "sessions", "list")
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+isolationPrefix)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("sessions list failed: %v, output: %s", err, string(output))
		}

		// Should show no containers initially
		outputStr := string(output)
		if !strings.Contains(outputStr, "No reactor containers found") {
			t.Errorf("Expected no containers initially but got: %s", outputStr)
		}
	})
}

// TestSessionsListOutput tests the formatting of sessions list output
func TestSessionsListOutput(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping sessions list output test")
	}

	reactorBinary := buildReactorBinary(t)
	isolationPrefix := "test-sessions-" + randomString(8)

	t.Run("sessions list table format", func(t *testing.T) {
		cmd := exec.Command(reactorBinary, "sessions", "list")
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+isolationPrefix)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("sessions list failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		// When no containers, should show helpful message
		expectedStrings := []string{
			"No reactor containers found",
			"Run 'reactor run' to create",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Expected sessions list output to contain '%s' but got: %s", expected, outputStr)
			}
		}
	})

	t.Run("sessions attach help", func(t *testing.T) {
		cmd := exec.Command(reactorBinary, "sessions", "attach", "--help")
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+isolationPrefix)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("sessions attach --help failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		expectedStrings := []string{
			"Attach to a container session",
			"Examples:",
			"reactor sessions attach",
			"Auto-attach to current project",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Expected sessions attach help to contain '%s' but got: %s", expected, outputStr)
			}
		}
	})
}

// TestContainerNameSanitization tests the container name sanitization logic
func TestContainerNameSanitization(t *testing.T) {
	// We can test this without Docker by examining config output
	reactorBinary := buildReactorBinary(t)

	testCases := []struct {
		projectName     string
		shouldContain   []string
		shouldNotContain []string
	}{
		{
			projectName:   "simple-project",
			shouldContain: []string{"simple-project"},
		},
		{
			projectName:   "project with spaces",
			shouldContain: []string{"project-with-spaces"}, // Spaces become hyphens
		},
		{
			projectName:   "project@#$%special",
			shouldContain: []string{"project"}, // Special chars should be sanitized
		},
	}

	for _, tc := range testCases {
		t.Run("sanitize_"+tc.projectName, func(t *testing.T) {
			tempDir := createTempDir(t, tc.projectName)
			isolationPrefix := "test-sanitize-" + randomString(8)

			// Init config in the test directory
			cmd := exec.Command(reactorBinary, "config", "init")
			cmd.Dir = tempDir
			cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+isolationPrefix)
			
			_, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("config init failed: %v", err)
			}

			// Get the config to see the generated names
			cmd = exec.Command(reactorBinary, "config", "show")
			cmd.Dir = tempDir
			cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+isolationPrefix)
			
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("config show failed: %v", err)
			}

			outputStr := string(output)
			for _, should := range tc.shouldContain {
				if !strings.Contains(outputStr, should) {
					t.Errorf("Expected config output to contain '%s' for project '%s' but got: %s", 
						should, tc.projectName, outputStr)
				}
			}

			for _, shouldNot := range tc.shouldNotContain {
				if strings.Contains(outputStr, shouldNot) {
					t.Errorf("Expected config output to NOT contain '%s' for project '%s' but got: %s", 
						shouldNot, tc.projectName, outputStr)
				}
			}
		})
	}
}

// TestErrorHandling tests error scenarios and messages
func TestErrorHandling(t *testing.T) {
	reactorBinary := buildReactorBinary(t)

	t.Run("sessions attach non-existent container", func(t *testing.T) {
		cmd := exec.Command(reactorBinary, "sessions", "attach", "non-existent-container")
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX=test-errors-"+randomString(8))
		
		output, err := cmd.CombinedOutput()
		// This should fail
		if err == nil {
			t.Errorf("Expected attach to non-existent container to fail but it succeeded. Output: %s", string(output))
		}

		outputStr := string(output)
		// Should contain helpful error message
		if !strings.Contains(outputStr, "not found") {
			t.Errorf("Expected error message about container not found but got: %s", outputStr)
		}
	})

	t.Run("config operations in non-initialized directory", func(t *testing.T) {
		tempDir := createTempDir(t, "non-initialized")
		
		// Try to get config without initializing
		cmd := exec.Command(reactorBinary, "config", "get", "provider")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX=test-errors-"+randomString(8))
		
		output, err := cmd.CombinedOutput()
		// This might fail or return default values - either is acceptable
		// The important thing is that it doesn't crash
		outputStr := string(output)
		
		// Should either succeed with default or fail gracefully
		if err != nil && !strings.Contains(outputStr, "not found") && !strings.Contains(outputStr, "default") {
			t.Errorf("Expected graceful handling of non-initialized config but got: %s", outputStr)
		}
	})
}

// TestIsolationPrefix tests that isolation prefixes work correctly
func TestIsolationPrefix(t *testing.T) {
	reactorBinary := buildReactorBinary(t)

	t.Run("isolation prefix in container names", func(t *testing.T) {
		tempDir := createTempDir(t, "isolation-test")
		isolationPrefix := "test-isolation-" + randomString(8)

		// Init with isolation prefix
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+isolationPrefix)
		
		_, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config init with isolation failed: %v", err)
		}

		// Check that config file uses isolation prefix
		cmd = exec.Command(reactorBinary, "config", "show")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+isolationPrefix)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config show with isolation failed: %v", err)
		}

		outputStr := string(output)
		// Should show the isolation prefix in paths and config
		if !strings.Contains(outputStr, isolationPrefix) {
			t.Errorf("Expected isolation prefix '%s' to appear in config output but got: %s", 
				isolationPrefix, outputStr)
		}
	})

	t.Run("different isolation prefixes create separate configs", func(t *testing.T) {
		tempDir := createTempDir(t, "multi-isolation-test")
		
		prefix1 := "test-multi1-" + randomString(8)
		prefix2 := "test-multi2-" + randomString(8)

		// Initialize with first prefix
		cmd := exec.Command(reactorBinary, "config", "init")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+prefix1)
		_, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config init with prefix1 failed: %v", err)
		}

		// Set a value with first prefix
		cmd = exec.Command(reactorBinary, "config", "set", "provider", "claude")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+prefix1)
		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config set with prefix1 failed: %v", err)
		}

		// Initialize with second prefix
		cmd = exec.Command(reactorBinary, "config", "init")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+prefix2)
		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config init with prefix2 failed: %v", err)
		}

		// Set a different value with second prefix
		cmd = exec.Command(reactorBinary, "config", "set", "provider", "gemini")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+prefix2)
		_, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config set with prefix2 failed: %v", err)
		}

		// Verify they have different values
		cmd = exec.Command(reactorBinary, "config", "get", "provider")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+prefix1)
		output1, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config get with prefix1 failed: %v", err)
		}

		cmd = exec.Command(reactorBinary, "config", "get", "provider")
		cmd.Dir = tempDir
		cmd.Env = append(cmd.Env, "REACTOR_ISOLATION_PREFIX="+prefix2)
		output2, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("config get with prefix2 failed: %v", err)
		}

		if !strings.Contains(string(output1), "claude") {
			t.Errorf("Expected prefix1 config to contain 'claude' but got: %s", string(output1))
		}
		if !strings.Contains(string(output2), "gemini") {
			t.Errorf("Expected prefix2 config to contain 'gemini' but got: %s", string(output2))
		}
	})
}

// Helper function to check if Docker is available
func isDockerAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false
	}
	defer cli.Close()
	
	_, err = cli.Ping(ctx)
	return err == nil
}
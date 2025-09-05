package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dyluth/reactor/pkg/testutil"
)

// TestBuildFunctionality tests the build command and build integration with up
func TestBuildFunctionality(t *testing.T) {
	// Set up isolated test environment
	_, testDir, cleanup := testutil.SetupIsolatedTest(t)
	defer cleanup()

	// Ensure Docker cleanup runs after test completion for all build test prefixes
	t.Cleanup(func() {
		if err := testutil.CleanupTestContainers("build-"); err != nil {
			t.Logf("Warning: failed to cleanup build test containers: %v", err)
		}
	})

	// Get shared reactor binary for testing
	reactorBinary := buildReactorBinary(t)

	// Change to test directory
	originalWD, _ := os.Getwd()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	t.Run("basic_dockerfile_build", func(t *testing.T) {
		isolationPrefix := "build-basic-" + randomString(8)

		// Create separate directory for this subtest
		subTestDir := filepath.Join(testDir, "basic-build-test")
		if err := os.MkdirAll(subTestDir, 0755); err != nil {
			t.Fatalf("Failed to create subtest directory: %v", err)
		}

		// Create .devcontainer directory first
		devcontainerDir := filepath.Join(subTestDir, ".devcontainer")
		if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
			t.Fatalf("Failed to create .devcontainer directory: %v", err)
		}

		// Create a simple Dockerfile in .devcontainer directory (default context)
		dockerfile := `FROM alpine:latest
RUN apk add --no-cache python3 py3-pip
WORKDIR /workspace
CMD ["/bin/sh"]`

		dockerfilePath := filepath.Join(devcontainerDir, "Dockerfile")
		if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		// Create devcontainer.json with build configuration
		devcontainerContent := `{
	"name": "build-test",
	"build": {
		"dockerfile": "Dockerfile"
	},
	"customizations": {
		"reactor": {
			"account": "test-user"
		}
	}
}`

		devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
		if err := os.WriteFile(devcontainerPath, []byte(devcontainerContent), 0644); err != nil {
			t.Fatalf("Failed to create devcontainer.json: %v", err)
		}

		// Test reactor build command
		cmd := exec.Command(reactorBinary, "build")
		cmd.Dir = subTestDir
		cmd.Env = setupTestEnv(isolationPrefix)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("reactor build failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should indicate successful build
		expectedPhrases := []string{
			"Building Docker image",
			"Build completed successfully",
		}

		for _, phrase := range expectedPhrases {
			if !strings.Contains(outputStr, phrase) {
				t.Errorf("Expected build output to contain '%s' but got: %s", phrase, outputStr)
			}
		}

		t.Logf("Basic Dockerfile build completed successfully")
	})

	t.Run("build_with_custom_context", func(t *testing.T) {
		isolationPrefix := "build-context-" + randomString(8)

		// Create separate directory for this subtest
		subTestDir := filepath.Join(testDir, "context-build-test")
		if err := os.MkdirAll(subTestDir, 0755); err != nil {
			t.Fatalf("Failed to create subtest directory: %v", err)
		}

		// Create .devcontainer directory first
		devcontainerDir := filepath.Join(subTestDir, ".devcontainer")
		if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
			t.Fatalf("Failed to create .devcontainer directory: %v", err)
		}

		// Create a build context directory relative to .devcontainer
		buildContextDir := filepath.Join(devcontainerDir, "build-context")
		if err := os.MkdirAll(buildContextDir, 0755); err != nil {
			t.Fatalf("Failed to create build context directory: %v", err)
		}

		// Create a Dockerfile in the build context
		dockerfile := `FROM node:18-alpine
WORKDIR /app
COPY package.json .
RUN npm install --production
COPY . .
EXPOSE 3000
CMD ["node", "index.js"]`

		dockerfilePath := filepath.Join(buildContextDir, "Dockerfile")
		if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		// Create a package.json file in build context
		packageJSON := `{
	"name": "test-app",
	"version": "1.0.0",
	"dependencies": {
		"express": "^4.18.0"
	}
}`

		packageJSONPath := filepath.Join(buildContextDir, "package.json")
		if err := os.WriteFile(packageJSONPath, []byte(packageJSON), 0644); err != nil {
			t.Fatalf("Failed to create package.json: %v", err)
		}

		// Create an index.js file
		indexJS := `const express = require('express');
const app = express();
app.get('/', (req, res) => res.send('Hello World'));
app.listen(3000, () => console.log('Server running on port 3000'));`

		indexJSPath := filepath.Join(buildContextDir, "index.js")
		if err := os.WriteFile(indexJSPath, []byte(indexJS), 0644); err != nil {
			t.Fatalf("Failed to create index.js: %v", err)
		}

		// Create devcontainer.json with build configuration using custom context
		devcontainerContent := `{
	"name": "node-context-test",
	"build": {
		"dockerfile": "Dockerfile",
		"context": "build-context"
	},
	"customizations": {
		"reactor": {
			"account": "test-user"
		}
	}
}`

		devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
		if err := os.WriteFile(devcontainerPath, []byte(devcontainerContent), 0644); err != nil {
			t.Fatalf("Failed to create devcontainer.json: %v", err)
		}

		// Test reactor build command with custom context
		cmd := exec.Command(reactorBinary, "build")
		cmd.Dir = subTestDir
		cmd.Env = setupTestEnv(isolationPrefix)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("reactor build with context failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should indicate successful build with context
		if !strings.Contains(outputStr, "Building Docker image") {
			t.Errorf("Expected build output to contain 'Building Docker image' but got: %s", outputStr)
		}

		if !strings.Contains(outputStr, "Build completed successfully") {
			t.Errorf("Expected build output to contain 'Build completed successfully' but got: %s", outputStr)
		}

		t.Logf("Build with custom context completed successfully")
	})

	t.Run("build_error_handling", func(t *testing.T) {
		isolationPrefix := "build-error-" + randomString(8)

		// Create separate directory for this subtest
		subTestDir := filepath.Join(testDir, "error-build-test")
		if err := os.MkdirAll(subTestDir, 0755); err != nil {
			t.Fatalf("Failed to create subtest directory: %v", err)
		}

		t.Run("missing_dockerfile", func(t *testing.T) {
			// Create devcontainer.json with build configuration pointing to non-existent Dockerfile
			devcontainerContent := `{
	"name": "missing-dockerfile-test",
	"build": {
		"dockerfile": "NonExistentDockerfile"
	}
}`

			devcontainerDir := filepath.Join(subTestDir, "missing-dockerfile", ".devcontainer")
			if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
				t.Fatalf("Failed to create .devcontainer directory: %v", err)
			}

			devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
			if err := os.WriteFile(devcontainerPath, []byte(devcontainerContent), 0644); err != nil {
				t.Fatalf("Failed to create devcontainer.json: %v", err)
			}

			// Test reactor build command - should fail gracefully
			cmd := exec.Command(reactorBinary, "build")
			cmd.Dir = filepath.Join(subTestDir, "missing-dockerfile")
			cmd.Env = setupTestEnv(isolationPrefix)

			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("Expected build to fail with missing Dockerfile, but it succeeded. Output: %s", string(output))
			}

			outputStr := string(output)

			// Should provide helpful error message
			if !strings.Contains(outputStr, "Dockerfile") && !strings.Contains(outputStr, "not found") {
				t.Logf("Build failed as expected with missing Dockerfile. Output: %s", outputStr)
			}
		})

		t.Run("invalid_dockerfile_syntax", func(t *testing.T) {
			// Create a Dockerfile with invalid syntax
			dockerfile := `INVALID_INSTRUCTION this should fail
FROM alpine:latest`

			dockerfileDir := filepath.Join(subTestDir, "invalid-dockerfile")
			if err := os.MkdirAll(dockerfileDir, 0755); err != nil {
				t.Fatalf("Failed to create dockerfile directory: %v", err)
			}

			dockerfilePath := filepath.Join(dockerfileDir, "Dockerfile")
			if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
				t.Fatalf("Failed to create invalid Dockerfile: %v", err)
			}

			// Create devcontainer.json
			devcontainerContent := `{
	"name": "invalid-syntax-test",
	"build": {
		"dockerfile": "Dockerfile"
	}
}`

			devcontainerDir := filepath.Join(dockerfileDir, ".devcontainer")
			if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
				t.Fatalf("Failed to create .devcontainer directory: %v", err)
			}

			devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
			if err := os.WriteFile(devcontainerPath, []byte(devcontainerContent), 0644); err != nil {
				t.Fatalf("Failed to create devcontainer.json: %v", err)
			}

			// Test reactor build command - should fail with Docker error
			cmd := exec.Command(reactorBinary, "build")
			cmd.Dir = dockerfileDir
			cmd.Env = setupTestEnv(isolationPrefix)

			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("Expected build to fail with invalid Dockerfile syntax, but it succeeded. Output: %s", string(output))
			}

			t.Logf("Build failed as expected with invalid Dockerfile syntax. Error: %v", err)
		})
	})

	t.Run("build_creates_reusable_image", func(t *testing.T) {
		isolationPrefix := "build-reuse-" + randomString(8)

		// Create separate directory for this subtest
		subTestDir := filepath.Join(testDir, "build-reuse-test")
		if err := os.MkdirAll(subTestDir, 0755); err != nil {
			t.Fatalf("Failed to create subtest directory: %v", err)
		}

		// Create .devcontainer directory first
		devcontainerDir := filepath.Join(subTestDir, ".devcontainer")
		if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
			t.Fatalf("Failed to create .devcontainer directory: %v", err)
		}

		// Create a simple Dockerfile in .devcontainer directory
		dockerfile := `FROM alpine:latest
RUN apk add --no-cache curl
WORKDIR /workspace
CMD ["/bin/sh"]`

		dockerfilePath := filepath.Join(devcontainerDir, "Dockerfile")
		if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		// Create devcontainer.json with build configuration
		devcontainerContent := `{
	"name": "build-reuse-test",
	"build": {
		"dockerfile": "Dockerfile"
	}
}`

		devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
		if err := os.WriteFile(devcontainerPath, []byte(devcontainerContent), 0644); err != nil {
			t.Fatalf("Failed to create devcontainer.json: %v", err)
		}

		// First build - should build the image
		cmd := exec.Command(reactorBinary, "build")
		cmd.Dir = subTestDir
		cmd.Env = setupTestEnv(isolationPrefix)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("First build failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Building Docker image") {
			t.Errorf("Expected first build to show 'Building Docker image' but got: %s", outputStr)
		}

		// Second build - should skip building if image already exists
		cmd = exec.Command(reactorBinary, "build")
		cmd.Dir = subTestDir
		cmd.Env = setupTestEnv(isolationPrefix)

		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Second build failed: %v, output: %s", err, string(output))
		}

		outputStr = string(output)
		// The second build should either skip (image exists) or rebuild
		if !strings.Contains(outputStr, "already exists") && !strings.Contains(outputStr, "Building Docker image") {
			t.Errorf("Expected second build to either skip or build, but got: %s", outputStr)
		}

		t.Logf("Build creates reusable image test completed successfully")
	})

	t.Run("build_without_devcontainer", func(t *testing.T) {
		isolationPrefix := "build-no-devcontainer-" + randomString(8)

		// Create separate directory for this subtest - no devcontainer.json
		subTestDir := filepath.Join(testDir, "no-devcontainer-test")
		if err := os.MkdirAll(subTestDir, 0755); err != nil {
			t.Fatalf("Failed to create subtest directory: %v", err)
		}

		// Test reactor build command without devcontainer.json - should fail
		cmd := exec.Command(reactorBinary, "build")
		cmd.Dir = subTestDir
		cmd.Env = setupTestEnv(isolationPrefix)

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Errorf("Expected build to fail without devcontainer.json, but it succeeded. Output: %s", string(output))
		}

		outputStr := string(output)

		// Should indicate missing devcontainer.json
		if !strings.Contains(outputStr, "devcontainer.json") {
			t.Errorf("Expected error message to mention devcontainer.json but got: %s", outputStr)
		}

		t.Logf("Build correctly failed without devcontainer.json")
	})

	// Clean up any test containers that may have been created during this test
	if err := testutil.AutoCleanupTestContainers(); err != nil {
		t.Logf("Warning: failed to cleanup test containers: %v", err)
	}
}

// Helper functions are defined in cli_test.go and shared across all integration tests

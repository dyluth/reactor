package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dyluth/reactor/pkg/config"
	"github.com/dyluth/reactor/pkg/testutil"
	"github.com/dyluth/reactor/pkg/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceParser(t *testing.T) {
	testutil.SetupIsolatedTest(t)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	t.Run("FindWorkspaceFile", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		// Test with no workspace file
		_, found, err := workspace.FindWorkspaceFile(tmpDir)
		require.NoError(t, err)
		assert.False(t, found)

		// Create reactor-workspace.yml
		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(workspaceFile, []byte("version: \"1\"\nservices: {}"), 0644)
		require.NoError(t, err)

		// Should find the file
		foundPath, found, err := workspace.FindWorkspaceFile(tmpDir)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, workspaceFile, foundPath)

		// Test with .yaml extension
		err = os.Remove(workspaceFile)
		require.NoError(t, err)
		workspaceFileYAML := filepath.Join(tmpDir, "reactor-workspace.yaml")
		err = os.WriteFile(workspaceFileYAML, []byte("version: \"1\"\nservices: {}"), 0644)
		require.NoError(t, err)

		foundPath, found, err = workspace.FindWorkspaceFile(tmpDir)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, workspaceFileYAML, foundPath)

		// Test with current directory (empty string)
		cwd, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.Chdir(cwd)
			require.NoError(t, err)
		})

		foundPath, found, err = workspace.FindWorkspaceFile("")
		require.NoError(t, err)
		assert.True(t, found)
		// Use EvalSymlinks to resolve any symlinks for comparison (macOS /var -> /private/var)
		expectedPath, _ := filepath.EvalSymlinks(workspaceFileYAML)
		actualPath, _ := filepath.EvalSymlinks(foundPath)
		assert.Equal(t, expectedPath, actualPath)
	})

	t.Run("ParseWorkspaceFile_ValidFile", func(t *testing.T) {
		// Create temporary directory structure
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		// Create service directories with devcontainer.json
		apiDir := filepath.Join(tmpDir, "services", "api", ".devcontainer")
		frontendDir := filepath.Join(tmpDir, "services", "frontend", ".devcontainer")
		require.NoError(t, os.MkdirAll(apiDir, 0755))
		require.NoError(t, os.MkdirAll(frontendDir, 0755))

		// Create devcontainer.json files
		apiDevcontainer := filepath.Join(apiDir, "devcontainer.json")
		frontendDevcontainer := filepath.Join(frontendDir, "devcontainer.json")

		err = os.WriteFile(apiDevcontainer, []byte(`{
			"name": "api-service",
			"image": "node:18",
			"customizations": {
				"reactor": {
					"account": "api-account"
				}
			}
		}`), 0644)
		require.NoError(t, err)

		err = os.WriteFile(frontendDevcontainer, []byte(`{
			"name": "frontend-service", 
			"image": "node:18"
		}`), 0644)
		require.NoError(t, err)

		// Create valid workspace file
		workspaceContent := `version: "1"
services:
  api:
    path: ./services/api
    account: work-account
  frontend:
    path: ./services/frontend`

		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(workspaceFile, []byte(workspaceContent), 0644)
		require.NoError(t, err)

		// Parse workspace file
		ws, err := workspace.ParseWorkspaceFile(workspaceFile)
		require.NoError(t, err)
		assert.Equal(t, "1", ws.Version)
		assert.Len(t, ws.Services, 2)

		// Check API service
		apiService, exists := ws.Services["api"]
		require.True(t, exists)
		assert.Equal(t, "./services/api", apiService.Path)
		assert.Equal(t, "work-account", apiService.Account)

		// Check frontend service
		frontendService, exists := ws.Services["frontend"]
		require.True(t, exists)
		assert.Equal(t, "./services/frontend", frontendService.Path)
		assert.Empty(t, frontendService.Account)
	})

	t.Run("ParseWorkspaceFile_InvalidVersions", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		testCases := []struct {
			name        string
			content     string
			expectedErr string
		}{
			{
				name:        "UnsupportedVersion",
				content:     "version: \"2\"\nservices:\n  api:\n    path: \"./api\"",
				expectedErr: "unsupported workspace version '2', expected '1'",
			},
			{
				name:        "NoServices",
				content:     "version: \"1\"\nservices: {}",
				expectedErr: "workspace must define at least one service",
			},
			{
				name:        "ServiceMissingPath",
				content:     "version: \"1\"\nservices:\n  api: {}",
				expectedErr: "service 'api' must define a path",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				workspaceFile := filepath.Join(tmpDir, "test-workspace.yml")
				err := os.WriteFile(workspaceFile, []byte(tc.content), 0644)
				require.NoError(t, err)
				t.Cleanup(func() {
					err := os.Remove(workspaceFile)
					require.NoError(t, err)
				})

				_, err = workspace.ParseWorkspaceFile(workspaceFile)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			})
		}
	})

	t.Run("ParseWorkspaceFile_SecurityValidation", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		testCases := []struct {
			name        string
			servicePath string
			expectedErr string
		}{
			{
				name:        "PathTraversal",
				servicePath: "../../../etc",
				expectedErr: "must be within the workspace directory",
			},
			{
				name:        "AbsolutePathOutsideWorkspace",
				servicePath: "/etc",
				expectedErr: "must be within the workspace directory",
			},
			{
				name:        "NonexistentPath",
				servicePath: "./nonexistent",
				expectedErr: "does not exist",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				workspaceContent := "version: \"1\"\nservices:\n  test:\n    path: " + tc.servicePath

				workspaceFile := filepath.Join(tmpDir, "test-workspace.yml")
				err := os.WriteFile(workspaceFile, []byte(workspaceContent), 0644)
				require.NoError(t, err)
				t.Cleanup(func() {
					err := os.Remove(workspaceFile)
					require.NoError(t, err)
				})

				_, err = workspace.ParseWorkspaceFile(workspaceFile)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			})
		}
	})

	t.Run("GenerateWorkspaceHash", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(workspaceFile, []byte("version: \"1\"\nservices: {}"), 0644)
		require.NoError(t, err)

		// Generate hash
		hash1, err := workspace.GenerateWorkspaceHash(workspaceFile)
		require.NoError(t, err)
		assert.Len(t, hash1, 64) // SHA256 hex string length

		// Same file should generate same hash
		hash2, err := workspace.GenerateWorkspaceHash(workspaceFile)
		require.NoError(t, err)
		assert.Equal(t, hash1, hash2)

		// Different file should generate different hash
		workspaceFile2 := filepath.Join(tmpDir, "other-workspace.yml")
		err = os.WriteFile(workspaceFile2, []byte("version: \"1\"\nservices: {}"), 0644)
		require.NoError(t, err)

		hash3, err := workspace.GenerateWorkspaceHash(workspaceFile2)
		require.NoError(t, err)
		assert.NotEqual(t, hash1, hash3)
	})
}

func TestWorkspaceValidationIntegration(t *testing.T) {
	testutil.SetupIsolatedTest(t)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	t.Run("ValidateWorkspaceWithValidServices", func(t *testing.T) {
		// Create workspace structure
		tmpDir, err := os.MkdirTemp("", "workspace-validation-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		// Create service directories and devcontainer.json files
		setupValidWorkspace(t, tmpDir)

		// Test workspace validation
		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		ws, err := workspace.ParseWorkspaceFile(workspaceFile)
		require.NoError(t, err)

		// Validate each service's devcontainer.json
		validServices := 0
		for serviceName, service := range ws.Services {
			servicePath := filepath.Join(tmpDir, service.Path)

			// Check for devcontainer.json
			_, found, err := config.FindDevContainerFile(servicePath)
			require.NoError(t, err, "Service %s should have devcontainer.json", serviceName)
			require.True(t, found, "Service %s devcontainer.json should be found", serviceName)

			// Validate devcontainer.json can be parsed
			configService := config.NewServiceWithRoot(servicePath)
			_, err = configService.ResolveConfiguration()
			require.NoError(t, err, "Service %s devcontainer.json should be valid", serviceName)

			validServices++
		}

		assert.Equal(t, 2, validServices)
	})

	t.Run("ValidateWorkspaceWithInvalidDevcontainer", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-validation-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		// Create basic structure
		serviceDir := filepath.Join(tmpDir, "services", "broken", ".devcontainer")
		require.NoError(t, os.MkdirAll(serviceDir, 0755))

		// Create invalid devcontainer.json
		invalidDevcontainer := filepath.Join(serviceDir, "devcontainer.json")
		err = os.WriteFile(invalidDevcontainer, []byte(`{invalid json`), 0644)
		require.NoError(t, err)

		// Create workspace file
		workspaceContent := `version: "1"
services:
  broken:
    path: ./services/broken`

		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(workspaceFile, []byte(workspaceContent), 0644)
		require.NoError(t, err)

		// Parse workspace (should succeed)
		ws, err := workspace.ParseWorkspaceFile(workspaceFile)
		require.NoError(t, err)

		// Validate service devcontainer.json (should fail)
		service := ws.Services["broken"]
		servicePath := filepath.Join(tmpDir, service.Path)

		configService := config.NewServiceWithRoot(servicePath)
		_, err = configService.ResolveConfiguration()
		assert.Error(t, err, "Invalid devcontainer.json should cause validation error")
	})
}

func TestWorkspaceContainerNaming(t *testing.T) {
	testutil.SetupIsolatedTest(t)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	t.Run("GenerateWorkspaceContainerNames", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-naming-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		setupValidWorkspace(t, tmpDir)

		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		ws, err := workspace.ParseWorkspaceFile(workspaceFile)
		require.NoError(t, err)

		// Generate workspace hash
		workspaceHash, err := workspace.GenerateWorkspaceHash(workspaceFile)
		require.NoError(t, err)
		assert.NotEmpty(t, workspaceHash)

		// Test container naming for each service
		for serviceName, service := range ws.Services {
			servicePath := filepath.Join(tmpDir, service.Path)
			projectHash := config.GenerateProjectHash(servicePath)

			expectedContainerName := "reactor-ws-" + serviceName + "-" + projectHash

			// Verify project hash generation
			assert.NotEmpty(t, projectHash)
			assert.Len(t, projectHash, 8) // Should be 8 characters

			// Verify container name format
			assert.Contains(t, expectedContainerName, serviceName)
			assert.Contains(t, expectedContainerName, "reactor-ws-")
			assert.Contains(t, expectedContainerName, projectHash)
		}
	})
}

// setupValidWorkspace creates a complete valid workspace structure for testing
func setupValidWorkspace(t *testing.T, tmpDir string) {
	// Create service directories
	apiDir := filepath.Join(tmpDir, "services", "api", ".devcontainer")
	frontendDir := filepath.Join(tmpDir, "services", "frontend", ".devcontainer")
	require.NoError(t, os.MkdirAll(apiDir, 0755))
	require.NoError(t, os.MkdirAll(frontendDir, 0755))

	// Create devcontainer.json files
	apiDevcontainer := filepath.Join(apiDir, "devcontainer.json")
	frontendDevcontainer := filepath.Join(frontendDir, "devcontainer.json")

	err := os.WriteFile(apiDevcontainer, []byte(`{
		"name": "api-service",
		"image": "node:18",
		"customizations": {
			"reactor": {
				"account": "api-account"
			}
		}
	}`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(frontendDevcontainer, []byte(`{
		"name": "frontend-service",
		"image": "node:18",
		"customizations": {
			"reactor": {
				"account": "frontend-account"
			}
		}
	}`), 0644)
	require.NoError(t, err)

	// Create workspace file
	workspaceContent := `version: "1"
services:
  api:
    path: ./services/api
    account: work-account
  frontend:
    path: ./services/frontend`

	workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
	err = os.WriteFile(workspaceFile, []byte(workspaceContent), 0644)
	require.NoError(t, err)
}

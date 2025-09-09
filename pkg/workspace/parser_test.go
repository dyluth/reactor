package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindWorkspaceFile(t *testing.T) {
	t.Run("NoWorkspaceFile", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		path, found, err := FindWorkspaceFile(tmpDir)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, path)
	})

	t.Run("WithReactorWorkspaceYml", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		expectedFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(expectedFile, []byte("test"), 0644)
		require.NoError(t, err)

		path, found, err := FindWorkspaceFile(tmpDir)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, expectedFile, path)
	})

	t.Run("WithReactorWorkspaceYaml", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		expectedFile := filepath.Join(tmpDir, "reactor-workspace.yaml")
		err = os.WriteFile(expectedFile, []byte("test"), 0644)
		require.NoError(t, err)

		path, found, err := FindWorkspaceFile(tmpDir)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, expectedFile, path)
	})

	t.Run("PreferYmlOverYaml", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		ymlFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		yamlFile := filepath.Join(tmpDir, "reactor-workspace.yaml")

		err = os.WriteFile(yamlFile, []byte("yaml"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(ymlFile, []byte("yml"), 0644)
		require.NoError(t, err)

		path, found, err := FindWorkspaceFile(tmpDir)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, ymlFile, path) // Should prefer .yml
	})

	t.Run("EmptyDirectoryUsesCurrentDir", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		// Create workspace file in temp dir
		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(workspaceFile, []byte("test"), 0644)
		require.NoError(t, err)

		// Change to temp dir
		cwd, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.Chdir(cwd)
			require.NoError(t, err)
		})

		path, found, err := FindWorkspaceFile("")
		require.NoError(t, err)
		assert.True(t, found)
		// Use EvalSymlinks to resolve any symlinks for comparison (macOS /var -> /private/var)
		expectedPath, _ := filepath.EvalSymlinks(workspaceFile)
		actualPath, _ := filepath.EvalSymlinks(path)
		assert.Equal(t, expectedPath, actualPath)
	})
}

func TestParseWorkspaceFile(t *testing.T) {
	t.Run("ValidWorkspaceFile", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		// Create service directories
		apiDir := filepath.Join(tmpDir, "services", "api")
		frontendDir := filepath.Join(tmpDir, "services", "frontend")
		require.NoError(t, os.MkdirAll(apiDir, 0755))
		require.NoError(t, os.MkdirAll(frontendDir, 0755))

		content := `version: "1"
services:
  api:
    path: ./services/api
    account: work-account
  frontend:
    path: ./services/frontend`

		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(workspaceFile, []byte(content), 0644)
		require.NoError(t, err)

		ws, err := ParseWorkspaceFile(workspaceFile)
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

	t.Run("InvalidVersion", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		content := `version: "2"
services:
  api:
    path: ./services/api`

		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(workspaceFile, []byte(content), 0644)
		require.NoError(t, err)

		_, err = ParseWorkspaceFile(workspaceFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported workspace version '2'")
	})

	t.Run("NoServices", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		content := `version: "1"
services: {}`

		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(workspaceFile, []byte(content), 0644)
		require.NoError(t, err)

		_, err = ParseWorkspaceFile(workspaceFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "workspace must define at least one service")
	})

	t.Run("ServiceMissingPath", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		content := `version: "1"
services:
  api: {}`

		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(workspaceFile, []byte(content), 0644)
		require.NoError(t, err)

		_, err = ParseWorkspaceFile(workspaceFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service 'api' must define a path")
	})

	t.Run("ServicePathDoesNotExist", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		content := `version: "1"
services:
  api:
    path: ./nonexistent`

		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(workspaceFile, []byte(content), 0644)
		require.NoError(t, err)

		_, err = ParseWorkspaceFile(workspaceFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("PathTraversalSecurity", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		testCases := []string{
			"../../../etc",
			"/../etc",
			"/etc",
			"./../../etc",
		}

		for _, maliciousPath := range testCases {
			t.Run("Path_"+maliciousPath, func(t *testing.T) {
				content := "version: \"1\"\nservices:\n  evil:\n    path: " + maliciousPath

				workspaceFile := filepath.Join(tmpDir, "test-workspace.yml")
				err = os.WriteFile(workspaceFile, []byte(content), 0644)
				require.NoError(t, err)
				t.Cleanup(func() {
					err := os.Remove(workspaceFile)
					require.NoError(t, err)
				})

				_, err = ParseWorkspaceFile(workspaceFile)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "must be within the workspace directory")
			})
		}
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		content := `version: "1"
services:
  api:
    path: ./services/api
    invalid yaml structure here`

		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(workspaceFile, []byte(content), 0644)
		require.NoError(t, err)

		_, err = ParseWorkspaceFile(workspaceFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse workspace YAML")
	})
}

func TestGenerateWorkspaceHash(t *testing.T) {
	t.Run("ConsistentHashGeneration", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
		err = os.WriteFile(workspaceFile, []byte("test"), 0644)
		require.NoError(t, err)

		hash1, err := GenerateWorkspaceHash(workspaceFile)
		require.NoError(t, err)
		assert.Len(t, hash1, 64) // SHA256 hex string

		// Same file should generate same hash
		hash2, err := GenerateWorkspaceHash(workspaceFile)
		require.NoError(t, err)
		assert.Equal(t, hash1, hash2)
	})

	t.Run("DifferentPathsDifferentHashes", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "workspace-test-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.RemoveAll(tmpDir)
			require.NoError(t, err)
		})

		file1 := filepath.Join(tmpDir, "workspace1.yml")
		file2 := filepath.Join(tmpDir, "workspace2.yml")

		err = os.WriteFile(file1, []byte("test"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(file2, []byte("test"), 0644)
		require.NoError(t, err)

		hash1, err := GenerateWorkspaceHash(file1)
		require.NoError(t, err)

		hash2, err := GenerateWorkspaceHash(file2)
		require.NoError(t, err)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("NonexistentFile", func(t *testing.T) {
		// GenerateWorkspaceHash doesn't validate file existence, only generates hash of absolute path
		// This is by design - the hash is based on the canonical path, not file contents
		hash, err := GenerateWorkspaceHash("/nonexistent/file.yml")
		require.NoError(t, err)
		assert.Len(t, hash, 64) // Should still generate valid hash
		assert.NotEmpty(t, hash)
	})
}

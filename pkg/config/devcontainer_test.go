package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindDevContainerFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "reactor-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(tmpDir)) })

	t.Run("finds file in .devcontainer directory", func(t *testing.T) {
		// Create .devcontainer directory and file
		devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))

		configFile := filepath.Join(devcontainerDir, "devcontainer.json")
		require.NoError(t, os.WriteFile(configFile, []byte(`{"image": "ubuntu"}`), 0644))

		// Test finding the file
		foundPath, found, err := FindDevContainerFile(tmpDir)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, configFile, foundPath)
	})

	t.Run("finds file in root directory", func(t *testing.T) {
		// Create a new temp dir for this test
		tmpDir2, err := os.MkdirTemp("", "reactor-test-*")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, os.RemoveAll(tmpDir2)) })

		// Create .devcontainer.json in root
		configFile := filepath.Join(tmpDir2, ".devcontainer.json")
		require.NoError(t, os.WriteFile(configFile, []byte(`{"image": "ubuntu"}`), 0644))

		// Test finding the file
		foundPath, found, err := FindDevContainerFile(tmpDir2)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, configFile, foundPath)
	})

	t.Run("prefers .devcontainer directory over root", func(t *testing.T) {
		// Create a new temp dir for this test
		tmpDir3, err := os.MkdirTemp("", "reactor-test-*")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, os.RemoveAll(tmpDir3)) })

		// Create both files
		devcontainerDir := filepath.Join(tmpDir3, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))

		preferredFile := filepath.Join(devcontainerDir, "devcontainer.json")
		require.NoError(t, os.WriteFile(preferredFile, []byte(`{"image": "preferred"}`), 0644))

		rootFile := filepath.Join(tmpDir3, ".devcontainer.json")
		require.NoError(t, os.WriteFile(rootFile, []byte(`{"image": "root"}`), 0644))

		// Test that it prefers the .devcontainer directory
		foundPath, found, err := FindDevContainerFile(tmpDir3)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, preferredFile, foundPath)
	})

	t.Run("returns false when no file found", func(t *testing.T) {
		// Create a new temp dir for this test
		tmpDir4, err := os.MkdirTemp("", "reactor-test-*")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, os.RemoveAll(tmpDir4)) })

		// Test with empty directory
		foundPath, found, err := FindDevContainerFile(tmpDir4)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, foundPath)
	})
}

func TestLoadDevContainerConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "reactor-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(tmpDir)) })

	t.Run("loads valid JSON config", func(t *testing.T) {
		configContent := `{
			"name": "test-project",
			"image": "ubuntu:latest",
			"remoteUser": "testuser",
			"forwardPorts": [8080, "3000:3001"],
			"customizations": {
				"reactor": {
					"account": "testaccount",
					"defaultCommand": "bash"
				}
			}
		}`

		configFile := filepath.Join(tmpDir, "valid.json")
		require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

		// Test loading the config
		config, err := LoadDevContainerConfig(configFile)
		require.NoError(t, err)
		assert.Equal(t, "test-project", config.Name)
		assert.Equal(t, "ubuntu:latest", config.Image)
		assert.Equal(t, "testuser", config.RemoteUser)
		assert.Len(t, config.ForwardPorts, 2)
		assert.NotNil(t, config.Customizations)
		assert.NotNil(t, config.Customizations.Reactor)
		assert.Equal(t, "testaccount", config.Customizations.Reactor.Account)
		assert.Equal(t, "bash", config.Customizations.Reactor.DefaultCommand)
	})

	t.Run("loads JSON with comments (JSONC)", func(t *testing.T) {
		configContent := `{
			// This is a test configuration
			"name": "test-with-comments",
			"image": "node:18", // Node.js image
			/* Multi-line comment
			   explaining the configuration */
			"remoteUser": "node"
		}`

		configFile := filepath.Join(tmpDir, "commented.json")
		require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

		// Test loading the config
		config, err := LoadDevContainerConfig(configFile)
		require.NoError(t, err)
		assert.Equal(t, "test-with-comments", config.Name)
		assert.Equal(t, "node:18", config.Image)
		assert.Equal(t, "node", config.RemoteUser)
	})

	t.Run("returns error for malformed JSON", func(t *testing.T) {
		configContent := `{
			"name": "malformed-json"
			"image": "ubuntu" // Missing comma
		}`

		configFile := filepath.Join(tmpDir, "malformed.json")
		require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

		// Test that it returns an error
		_, err := LoadDevContainerConfig(configFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse JSONC")
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		nonexistentFile := filepath.Join(tmpDir, "nonexistent.json")

		_, err := LoadDevContainerConfig(nonexistentFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read devcontainer file")
	})

	t.Run("handles minimal config", func(t *testing.T) {
		configContent := `{
			"image": "alpine"
		}`

		configFile := filepath.Join(tmpDir, "minimal.json")
		require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

		// Test loading minimal config
		config, err := LoadDevContainerConfig(configFile)
		require.NoError(t, err)
		assert.Equal(t, "alpine", config.Image)
		assert.Empty(t, config.Name)
		assert.Empty(t, config.RemoteUser)
		assert.Nil(t, config.Customizations)
	})
}

func TestServiceResolveConfiguration(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "reactor-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(tmpDir)) })

	t.Run("resolves basic devcontainer config", func(t *testing.T) {
		// Create devcontainer.json
		configContent := `{
			"name": "test-project",
			"image": "ubuntu:latest",
			"remoteUser": "testuser"
		}`

		configFile := filepath.Join(tmpDir, ".devcontainer.json")
		require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

		// Create service with test directory as project root
		service := &Service{
			projectRoot: tmpDir,
		}

		// Test resolution
		resolved, err := service.ResolveConfiguration()
		require.NoError(t, err)
		assert.Equal(t, "ubuntu:latest", resolved.Image)
		assert.Equal(t, tmpDir, resolved.ProjectRoot)
		assert.NotEmpty(t, resolved.ProjectHash)
		assert.NotEmpty(t, resolved.Account)                           // Should use system username
		assert.Equal(t, BuiltinProviders["claude"], resolved.Provider) // Default provider
	})

	t.Run("uses reactor customizations for account", func(t *testing.T) {
		// Create a new temp dir for this test
		tmpDir2, err := os.MkdirTemp("", "reactor-test-*")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, os.RemoveAll(tmpDir2)) })

		// Create devcontainer.json with reactor customizations
		configContent := `{
			"image": "node:18",
			"customizations": {
				"reactor": {
					"account": "custom-account"
				}
			}
		}`

		configFile := filepath.Join(tmpDir2, ".devcontainer.json")
		require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

		// Create service
		service := &Service{
			projectRoot: tmpDir2,
		}

		// Test resolution
		resolved, err := service.ResolveConfiguration()
		require.NoError(t, err)
		assert.Equal(t, "custom-account", resolved.Account)
		assert.Equal(t, "node:18", resolved.Image)
	})

	t.Run("returns error when no devcontainer.json found", func(t *testing.T) {
		// Create a new temp dir for this test
		tmpDir3, err := os.MkdirTemp("", "reactor-test-*")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, os.RemoveAll(tmpDir3)) })

		// Create service with empty directory
		service := &Service{
			projectRoot: tmpDir3,
		}

		// Test that it returns an error
		_, err = service.ResolveConfiguration()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no devcontainer.json found")
	})

	t.Run("uses default image when not specified", func(t *testing.T) {
		// Create a new temp dir for this test
		tmpDir4, err := os.MkdirTemp("", "reactor-test-*")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, os.RemoveAll(tmpDir4)) })

		// Create devcontainer.json without image
		configContent := `{
			"name": "no-image-project"
		}`

		configFile := filepath.Join(tmpDir4, ".devcontainer.json")
		require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

		// Create service
		service := &Service{
			projectRoot: tmpDir4,
		}

		// Test resolution uses default image
		resolved, err := service.ResolveConfiguration()
		require.NoError(t, err)
		assert.Equal(t, BuiltinProviders["claude"].DefaultImage, resolved.Image)
	})
}

func TestCompleteDataFlowTransformation(t *testing.T) {
	// Create comprehensive devcontainer.json
	configContent := `{
		"name": "full-test-project",
		"image": "ubuntu:22.04",
		"remoteUser": "developer",
		"forwardPorts": [8080, "3000:3001"],
		"postCreateCommand": "npm install",
		"build": {
			"dockerfile": "Dockerfile",
			"context": "."
		},
		"customizations": {
			"reactor": {
				"account": "test-account",
				"defaultCommand": "/bin/zsh"
			}
		}
	}`

	// Test both file locations
	testCases := []struct {
		name     string
		filePath string
	}{
		{"devcontainer directory", ".devcontainer/devcontainer.json"},
		{"root directory", ".devcontainer.json"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create isolated temporary directory for this sub-test
			tmpDir, err := os.MkdirTemp("", "reactor-test-*")
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, os.RemoveAll(tmpDir)) })

			// Create directory if needed
			dir := filepath.Dir(filepath.Join(tmpDir, tc.filePath))
			if dir != tmpDir {
				require.NoError(t, os.MkdirAll(dir, 0755))
			}

			// Create config file
			configFile := filepath.Join(tmpDir, tc.filePath)
			require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

			// Test complete flow: devcontainer.json -> DevContainerConfig -> ResolvedConfig

			// 1. Find file
			foundPath, found, err := FindDevContainerFile(tmpDir)
			require.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, configFile, foundPath)

			// 2. Load DevContainerConfig
			devConfig, err := LoadDevContainerConfig(foundPath)
			require.NoError(t, err)
			assert.Equal(t, "full-test-project", devConfig.Name)
			assert.Equal(t, "ubuntu:22.04", devConfig.Image)
			assert.Equal(t, "developer", devConfig.RemoteUser)
			assert.Equal(t, "npm install", devConfig.PostCreateCommand)
			assert.NotNil(t, devConfig.Build)
			assert.Equal(t, "Dockerfile", devConfig.Build.Dockerfile)
			assert.Equal(t, ".", devConfig.Build.Context)
			assert.NotNil(t, devConfig.Customizations)
			assert.NotNil(t, devConfig.Customizations.Reactor)
			assert.Equal(t, "test-account", devConfig.Customizations.Reactor.Account)
			assert.Equal(t, "/bin/zsh", devConfig.Customizations.Reactor.DefaultCommand)

			// 3. Transform to ResolvedConfig via service
			service := &Service{
				projectRoot: tmpDir,
			}

			resolved, err := service.ResolveConfiguration()
			require.NoError(t, err)

			// Verify the transformation
			assert.Equal(t, "ubuntu:22.04", resolved.Image)
			assert.Equal(t, "test-account", resolved.Account)
			assert.Equal(t, tmpDir, resolved.ProjectRoot)
			assert.NotEmpty(t, resolved.ProjectHash)
			assert.Contains(t, resolved.AccountConfigDir, "test-account")
			assert.Contains(t, resolved.ProjectConfigDir, resolved.ProjectHash)
			assert.Equal(t, BuiltinProviders["claude"], resolved.Provider)
			assert.False(t, resolved.Danger) // Default to safe mode
		})
	}
}

package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dyluth/reactor/pkg/config"
	"github.com/dyluth/reactor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateService_ValidateDirectories_Success(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	// Create temporary directories
	tempDir := t.TempDir()
	accountDir := filepath.Join(tempDir, "account")
	projectDir := filepath.Join(tempDir, "project")
	providerDir1 := filepath.Join(projectDir, "provider1")
	providerDir2 := filepath.Join(projectDir, "provider2")

	// Create all required directories
	require.NoError(t, os.MkdirAll(accountDir, 0755))
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	require.NoError(t, os.MkdirAll(providerDir1, 0755))
	require.NoError(t, os.MkdirAll(providerDir2, 0755))

	// Create resolved config with provider mounts
	resolved := &config.ResolvedConfig{
		AccountConfigDir: accountDir,
		ProjectConfigDir: projectDir,
		Provider: config.ProviderInfo{
			Mounts: []config.MountPoint{
				{Source: "provider1", Target: "/container/provider1"},
				{Source: "provider2", Target: "/container/provider2"},
			},
		},
	}

	service := NewStateService(resolved)
	err := service.ValidateDirectories()

	assert.NoError(t, err)
}

func TestStateService_ValidateDirectories_MissingAccountDir(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	tempDir := t.TempDir()
	nonExistentAccountDir := filepath.Join(tempDir, "nonexistent-account")
	projectDir := filepath.Join(tempDir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	resolved := &config.ResolvedConfig{
		AccountConfigDir: nonExistentAccountDir,
		ProjectConfigDir: projectDir,
		Provider: config.ProviderInfo{
			Mounts: []config.MountPoint{},
		},
	}

	service := NewStateService(resolved)
	err := service.ValidateDirectories()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account directory does not exist")
	assert.Contains(t, err.Error(), nonExistentAccountDir)
}

func TestStateService_ValidateDirectories_MissingProjectDir(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	tempDir := t.TempDir()
	accountDir := filepath.Join(tempDir, "account")
	nonExistentProjectDir := filepath.Join(tempDir, "nonexistent-project")
	require.NoError(t, os.MkdirAll(accountDir, 0755))

	resolved := &config.ResolvedConfig{
		AccountConfigDir: accountDir,
		ProjectConfigDir: nonExistentProjectDir,
		Provider: config.ProviderInfo{
			Mounts: []config.MountPoint{},
		},
	}

	service := NewStateService(resolved)
	err := service.ValidateDirectories()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project configuration directory does not exist")
	assert.Contains(t, err.Error(), nonExistentProjectDir)
}

func TestStateService_ValidateDirectories_MissingProviderDir(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	tempDir := t.TempDir()
	accountDir := filepath.Join(tempDir, "account")
	projectDir := filepath.Join(tempDir, "project")
	
	// Create base directories but NOT the provider directory
	require.NoError(t, os.MkdirAll(accountDir, 0755))
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	resolved := &config.ResolvedConfig{
		AccountConfigDir: accountDir,
		ProjectConfigDir: projectDir,
		Provider: config.ProviderInfo{
			Mounts: []config.MountPoint{
				{Source: "nonexistent-provider", Target: "/container/provider"},
			},
		},
	}

	service := NewStateService(resolved)
	err := service.ValidateDirectories()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider directory does not exist")
	assert.Contains(t, err.Error(), "nonexistent-provider")
}

func TestStateService_ValidateDirectories_MultipleProviderDirs(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	tempDir := t.TempDir()
	accountDir := filepath.Join(tempDir, "account")
	projectDir := filepath.Join(tempDir, "project")
	
	// Create base directories and first provider directory
	require.NoError(t, os.MkdirAll(accountDir, 0755))
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "provider1"), 0755))
	// Intentionally NOT create provider2

	resolved := &config.ResolvedConfig{
		AccountConfigDir: accountDir,
		ProjectConfigDir: projectDir,
		Provider: config.ProviderInfo{
			Mounts: []config.MountPoint{
				{Source: "provider1", Target: "/container/provider1"}, // exists
				{Source: "provider2", Target: "/container/provider2"}, // doesn't exist
			},
		},
	}

	service := NewStateService(resolved)
	err := service.ValidateDirectories()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider directory does not exist")
	assert.Contains(t, err.Error(), "provider2")
}

func TestStateService_GetMounts(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	projectRoot := filepath.Join(tempDir, "workspace")

	resolved := &config.ResolvedConfig{
		ProjectConfigDir: projectDir,
		ProjectRoot:      projectRoot,
		Provider: config.ProviderInfo{
			Mounts: []config.MountPoint{
				{Source: "claude-config", Target: "/home/claude/.claude"},
				{Source: "ssh-keys", Target: "/home/claude/.ssh"},
				{Source: "git-config", Target: "/home/claude/.gitconfig"},
			},
		},
	}

	service := NewStateService(resolved)
	mounts := service.GetMounts()

	// Should have provider mounts + project root mount
	require.Len(t, mounts, 4)

	// Verify provider mounts
	expectedProviderMounts := []MountSpec{
		{
			Source: filepath.Join(projectDir, "claude-config"),
			Target: "/home/claude/.claude",
			Type:   "bind",
		},
		{
			Source: filepath.Join(projectDir, "ssh-keys"),
			Target: "/home/claude/.ssh",
			Type:   "bind",
		},
		{
			Source: filepath.Join(projectDir, "git-config"),
			Target: "/home/claude/.gitconfig",
			Type:   "bind",
		},
	}

	for i, expected := range expectedProviderMounts {
		assert.Equal(t, expected, mounts[i], "Provider mount %d should match", i)
	}

	// Verify project root mount (last one)
	projectRootMount := mounts[len(mounts)-1]
	assert.Equal(t, projectRoot, projectRootMount.Source)
	assert.Equal(t, "/workspace", projectRootMount.Target)
	assert.Equal(t, "bind", projectRootMount.Type)
}

func TestStateService_GetMounts_NoProviderMounts(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	projectRoot := "/home/user/myproject"

	resolved := &config.ResolvedConfig{
		ProjectConfigDir: "/config/dir",
		ProjectRoot:      projectRoot,
		Provider: config.ProviderInfo{
			Mounts: []config.MountPoint{}, // No provider mounts
		},
	}

	service := NewStateService(resolved)
	mounts := service.GetMounts()

	// Should only have project root mount
	require.Len(t, mounts, 1)

	expected := MountSpec{
		Source: projectRoot,
		Target: "/workspace",
		Type:   "bind",
	}
	assert.Equal(t, expected, mounts[0])
}

func TestStateService_GetMounts_EmptyProvider(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	projectRoot := "/home/user/project"

	resolved := &config.ResolvedConfig{
		ProjectConfigDir: "/config",
		ProjectRoot:      projectRoot,
		Provider: config.ProviderInfo{}, // Empty provider config
	}

	service := NewStateService(resolved)
	mounts := service.GetMounts()

	// Should only have project root mount
	require.Len(t, mounts, 1)
	assert.Equal(t, projectRoot, mounts[0].Source)
	assert.Equal(t, "/workspace", mounts[0].Target)
	assert.Equal(t, "bind", mounts[0].Type)
}

func TestStateService_GetAccount(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	resolved := &config.ResolvedConfig{
		Account: "testuser",
	}

	service := NewStateService(resolved)
	account := service.GetAccount()

	assert.Equal(t, "testuser", account)
}

func TestStateService_GetProjectHash(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	resolved := &config.ResolvedConfig{
		ProjectHash: "abcdef123456",
	}

	service := NewStateService(resolved)
	hash := service.GetProjectHash()

	assert.Equal(t, "abcdef123456", hash)
}

func TestStateService_GetProjectRoot(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	resolved := &config.ResolvedConfig{
		ProjectRoot: "/home/user/myproject",
	}

	service := NewStateService(resolved)
	root := service.GetProjectRoot()

	assert.Equal(t, "/home/user/myproject", root)
}

func TestNewStateService(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	resolved := &config.ResolvedConfig{
		Account:     "test",
		ProjectRoot: "/test",
	}

	service := NewStateService(resolved)

	assert.NotNil(t, service)
	assert.Equal(t, resolved, service.resolved)
}

// Edge case and integration tests

func TestStateService_ValidateDirectories_SymbolicLinks(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	tempDir := t.TempDir()
	actualAccountDir := filepath.Join(tempDir, "actual-account")
	symlinkAccountDir := filepath.Join(tempDir, "symlink-account")
	
	// Create actual directory and symbolic link
	require.NoError(t, os.MkdirAll(actualAccountDir, 0755))
	require.NoError(t, os.Symlink(actualAccountDir, symlinkAccountDir))
	
	// Create project directory
	projectDir := filepath.Join(tempDir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	resolved := &config.ResolvedConfig{
		AccountConfigDir: symlinkAccountDir, // Using symbolic link
		ProjectConfigDir: projectDir,
		Provider: config.ProviderInfo{
			Mounts: []config.MountPoint{},
		},
	}

	service := NewStateService(resolved)
	err := service.ValidateDirectories()

	// Should work with symbolic links
	assert.NoError(t, err)
}

func TestStateService_ValidateDirectories_FileInsteadOfDirectory(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	tempDir := t.TempDir()
	
	// Create a file where we expect a directory
	accountFile := filepath.Join(tempDir, "account-file")
	require.NoError(t, os.WriteFile(accountFile, []byte("not a directory"), 0644))

	resolved := &config.ResolvedConfig{
		AccountConfigDir: accountFile, // This is a file, not a directory
		ProjectConfigDir: "/nonexistent",
		Provider: config.ProviderInfo{
			Mounts: []config.MountPoint{},
		},
	}

	service := NewStateService(resolved)
	err := service.ValidateDirectories()

	// Should fail because account path is a file, not a directory
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project configuration directory does not exist")
}

func TestStateService_GetMounts_AbsolutePaths(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	projectConfigDir := "/absolute/path/to/config"
	projectRoot := "/absolute/path/to/project"

	resolved := &config.ResolvedConfig{
		ProjectConfigDir: projectConfigDir,
		ProjectRoot:      projectRoot,
		Provider: config.ProviderInfo{
			Mounts: []config.MountPoint{
				{Source: "relative/path", Target: "/container/target"},
			},
		},
	}

	service := NewStateService(resolved)
	mounts := service.GetMounts()

	require.Len(t, mounts, 2)
	
	// Verify that relative source path gets joined with ProjectConfigDir
	expectedProviderMount := MountSpec{
		Source: filepath.Join(projectConfigDir, "relative/path"),
		Target: "/container/target",
		Type:   "bind",
	}
	assert.Equal(t, expectedProviderMount, mounts[0])
	
	// Verify project root mount uses absolute path
	expectedProjectMount := MountSpec{
		Source: projectRoot,
		Target: "/workspace",
		Type:   "bind",
	}
	assert.Equal(t, expectedProjectMount, mounts[1])
}
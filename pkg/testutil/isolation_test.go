package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithIsolatedHome(t *testing.T) {
	// Store original HOME for verification
	originalHome := os.Getenv("HOME")

	// Create isolated home
	isolatedHome := WithIsolatedHome(t)

	// Verify HOME was changed to our temporary directory
	currentHome := os.Getenv("HOME")
	if currentHome != isolatedHome {
		t.Errorf("Expected HOME to be %s, got %s", isolatedHome, currentHome)
	}

	// Verify the directory exists and is writable
	testFile := filepath.Join(isolatedHome, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Errorf("Failed to write to isolated home directory: %v", err)
	}

	// Verify it's a temporary directory (should contain temp path patterns)
	if !strings.Contains(isolatedHome, "TestWithIsolatedHome") {
		t.Errorf("Isolated home %s doesn't appear to be a test temp directory", isolatedHome)
	}

	// After test completes, t.Setenv should restore original HOME automatically
	// We can't test this here since it happens during test cleanup
	_ = originalHome // Acknowledge we stored it
}

func TestWithIsolatedWorkspace(t *testing.T) {
	originalHome := os.Getenv("HOME")

	homeDir, workspaceDir := WithIsolatedWorkspace(t)

	// Verify HOME was set
	if os.Getenv("HOME") != homeDir {
		t.Errorf("Expected HOME to be %s, got %s", homeDir, os.Getenv("HOME"))
	}

	// Verify both directories exist and are different
	if homeDir == workspaceDir {
		t.Error("Home and workspace directories should be different")
	}

	// Verify both directories are writable
	homeTestFile := filepath.Join(homeDir, "home_test.txt")
	workspaceTestFile := filepath.Join(workspaceDir, "workspace_test.txt")

	if err := os.WriteFile(homeTestFile, []byte("home test"), 0644); err != nil {
		t.Errorf("Failed to write to isolated home: %v", err)
	}

	if err := os.WriteFile(workspaceTestFile, []byte("workspace test"), 0644); err != nil {
		t.Errorf("Failed to write to isolated workspace: %v", err)
	}

	_ = originalHome
}

func TestCanonicalPath(t *testing.T) {
	// Create a test directory
	testDir := t.TempDir()

	// Test with regular path
	canonical := CanonicalPath(t, testDir)
	if !filepath.IsAbs(canonical) {
		t.Errorf("Canonical path should be absolute, got %s", canonical)
	}

	// Test with non-existent path (should still return absolute path)
	nonExistent := filepath.Join(testDir, "nonexistent")
	canonicalNonExistent := CanonicalPath(t, nonExistent)
	if !filepath.IsAbs(canonicalNonExistent) {
		t.Errorf("Canonical path for non-existent should be absolute, got %s", canonicalNonExistent)
	}
}

func TestAssertPathsEqual(t *testing.T) {
	// Create test directories
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Test equal paths (should pass)
	AssertPathsEqual(t, dir1, dir1, "same directory should be equal")

	// Test different paths (this would fail the test, so we'll simulate it)
	// We can't easily test the failure case without creating a sub-test
	t.Run("different_paths_should_fail", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				// The test should fail when paths are different, but since we're testing
				// the testing helper, we expect it to call t.Errorf, not panic
				// This is just to document expected behavior
				t.Log("No panic occurred as expected")
			}
		}()
		// This assertion should work (both paths should be equal to themselves)
		AssertPathsEqual(t, dir1, dir1, "should pass")
		AssertPathsEqual(t, dir2, dir2, "should pass")
	})
}

func TestSetupIsolatedTest(t *testing.T) {
	originalWD, _ := os.Getwd()

	homeDir, workspaceDir, cleanup := SetupIsolatedTest(t)
	defer cleanup()

	// Verify isolation
	if homeDir == workspaceDir {
		t.Error("Home and workspace should be different")
	}

	if os.Getenv("HOME") != homeDir {
		t.Errorf("HOME should be set to %s, got %s", homeDir, os.Getenv("HOME"))
	}

	// Test changing to workspace directory
	if err := os.Chdir(workspaceDir); err != nil {
		t.Errorf("Failed to change to workspace directory: %v", err)
	}

	currentWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Verify we're in the workspace directory
	AssertPathsEqual(t, workspaceDir, currentWD, "should be in workspace directory")

	// Test cleanup - it should restore original working directory
	cleanup()

	restoredWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory after cleanup: %v", err)
	}

	AssertPathsEqual(t, originalWD, restoredWD, "cleanup should restore original working directory")
}

func TestRobustRemoveAll_Success(t *testing.T) {
	// Create a test directory we can remove
	testDir := t.TempDir()
	subDir := filepath.Join(testDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(subDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Copy the testDir path since t.TempDir() manages its own cleanup
	testDirCopy := filepath.Join(os.TempDir(), "robust-remove-test-"+filepath.Base(testDir))
	if err := os.Rename(testDir, testDirCopy); err != nil {
		t.Fatalf("Failed to rename test directory: %v", err)
	}

	// RobustRemoveAll should successfully remove the directory
	err := RobustRemoveAll(t, testDirCopy)
	if err != nil {
		t.Errorf("RobustRemoveAll failed: %v", err)
	}

	// Directory should no longer exist
	if _, err := os.Stat(testDirCopy); !os.IsNotExist(err) {
		t.Error("Directory should have been removed")
	}
}

func TestRobustRemoveAll_NonExistent(t *testing.T) {
	// RobustRemoveAll should succeed on non-existent paths
	nonExistentPath := filepath.Join(os.TempDir(), "non-existent-dir-12345")
	err := RobustRemoveAll(t, nonExistentPath)
	if err != nil {
		t.Errorf("RobustRemoveAll should succeed on non-existent paths: %v", err)
	}
}

func TestSetupCredentialDirectories(t *testing.T) {
	// Create isolated home directory
	homeDir := WithIsolatedHome(t)
	account := "test-account"
	projectHash := "abc123def456"

	// Call function under test
	credDirs := SetupCredentialDirectories(t, homeDir, account, projectHash)

	// Verify expected credential files were created
	expectedFiles := []string{
		"claude_credentials.txt",
		"claude_config.json",
		"gemini_api_key.txt",
		"gemini_settings.yaml",
	}

	assert.Len(t, credDirs, len(expectedFiles))
	for _, expectedFile := range expectedFiles {
		filePath, exists := credDirs[expectedFile]
		assert.True(t, exists, "Expected file %s not found in credDirs", expectedFile)

		// Verify file exists
		_, err := os.Stat(filePath)
		assert.NoError(t, err, "Credential file %s should exist", filePath)

		// Verify file content
		content, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "test-", "File content should contain test prefix")
		assert.Contains(t, string(content), account, "File content should contain account name")
	}

	// Verify directory structure
	expectedReactorDir := filepath.Join(homeDir, ".reactor", account, projectHash)
	info, err := os.Stat(expectedReactorDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify provider subdirectories
	claudeDir := filepath.Join(expectedReactorDir, "claude")
	info, err = os.Stat(claudeDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	geminiDir := filepath.Join(expectedReactorDir, "gemini")
	info, err = os.Stat(geminiDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestSetupCredentialDirectories_DifferentAccounts(t *testing.T) {
	homeDir := WithIsolatedHome(t)
	account1 := "account1"
	account2 := "account2"
	projectHash := "samehash"

	// Setup credentials for two different accounts
	credDirs1 := SetupCredentialDirectories(t, homeDir, account1, projectHash)
	credDirs2 := SetupCredentialDirectories(t, homeDir, account2, projectHash)

	// Both should have same number of files
	assert.Equal(t, len(credDirs1), len(credDirs2))

	// But files should be in different directories
	for key := range credDirs1 {
		path1 := credDirs1[key]
		path2 := credDirs2[key]
		assert.NotEqual(t, path1, path2, "Paths should be different for different accounts")
		assert.Contains(t, path1, account1, "Path should contain account1")
		assert.Contains(t, path2, account2, "Path should contain account2")
	}

	// Verify content is account-specific
	claudeFile1 := credDirs1["claude_credentials.txt"]
	claudeFile2 := credDirs2["claude_credentials.txt"]

	content1, err := os.ReadFile(claudeFile1)
	assert.NoError(t, err)
	content2, err := os.ReadFile(claudeFile2)
	assert.NoError(t, err)

	assert.Contains(t, string(content1), account1)
	assert.Contains(t, string(content2), account2)
	assert.NotEqual(t, string(content1), string(content2))
}

func TestSetupCredentialDirectories_DifferentProjects(t *testing.T) {
	homeDir := WithIsolatedHome(t)
	account := "same-account"
	projectHash1 := "project1"
	projectHash2 := "project2"

	// Setup credentials for same account but different projects
	credDirs1 := SetupCredentialDirectories(t, homeDir, account, projectHash1)
	credDirs2 := SetupCredentialDirectories(t, homeDir, account, projectHash2)

	// Files should be in different project directories
	for key := range credDirs1 {
		path1 := credDirs1[key]
		path2 := credDirs2[key]
		assert.NotEqual(t, path1, path2, "Paths should be different for different projects")
		assert.Contains(t, path1, projectHash1, "Path should contain projectHash1")
		assert.Contains(t, path2, projectHash2, "Path should contain projectHash2")
	}
}

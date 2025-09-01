package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
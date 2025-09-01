// Package testutil provides utilities for creating isolated, hermetic test environments
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// WithIsolatedHome creates a temporary home directory and sets the HOME environment
// variable for the duration of the test. This ensures tests run in a clean, isolated
// environment without depending on the actual user's home directory.
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    testutil.WithIsolatedHome(t)
//	    // Test code that depends on HOME directory
//	}
func WithIsolatedHome(t *testing.T) string {
	t.Helper()

	// Create a temporary directory that will serve as the isolated HOME
	tempHome := t.TempDir()

	// Set HOME environment variable to the temporary directory for this test
	t.Setenv("HOME", tempHome)

	return tempHome
}

// WithIsolatedWorkspace creates both an isolated home directory and a temporary
// workspace directory for tests that need both. Returns both paths.
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    homeDir, workspaceDir := testutil.WithIsolatedWorkspace(t)
//	    // Test code that needs both home and workspace isolation
//	}
func WithIsolatedWorkspace(t *testing.T) (homeDir, workspaceDir string) {
	t.Helper()

	// Create isolated home directory
	homeDir = WithIsolatedHome(t)

	// Create isolated workspace directory
	workspaceDir = t.TempDir()

	return homeDir, workspaceDir
}

// CanonicalPath resolves symlinks and returns the canonical absolute path.
// This is useful for path comparisons in tests that might encounter symlink
// differences between different operating systems (e.g., /var vs /private/var on macOS).
//
// Usage:
//
//	expected := testutil.CanonicalPath(t, expectedPath)
//	actual := testutil.CanonicalPath(t, actualPath)
//	assert.Equal(t, expected, actual)
func CanonicalPath(t *testing.T, path string) string {
	t.Helper()

	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If we can't resolve symlinks, return the absolute path
		abs, absErr := filepath.Abs(path)
		if absErr != nil {
			t.Fatalf("Failed to get canonical or absolute path for %s: symlink error: %v, abs error: %v", path, err, absErr)
		}
		return abs
	}

	// Ensure the result is absolute
	if !filepath.IsAbs(canonical) {
		abs, err := filepath.Abs(canonical)
		if err != nil {
			t.Fatalf("Failed to get absolute path for resolved symlink %s: %v", canonical, err)
		}
		canonical = abs
	}

	return canonical
}

// AssertPathsEqual compares two paths after canonicalizing them to handle
// symlink differences across operating systems.
//
// Usage:
//
//	testutil.AssertPathsEqual(t, expectedPath, actualPath, "project root should match")
func AssertPathsEqual(t *testing.T, expected, actual, message string) {
	t.Helper()

	expectedCanonical := CanonicalPath(t, expected)
	actualCanonical := CanonicalPath(t, actual)

	if expectedCanonical != actualCanonical {
		t.Errorf("%s: expected path %q (canonical: %q) but got %q (canonical: %q)",
			message, expected, expectedCanonical, actual, actualCanonical)
	}
}

// SetupIsolatedTest provides complete test isolation including HOME directory,
// workspace directory, and robust cleanup for Docker-created files. This is the most
// comprehensive test isolation helper.
//
// The cleanup function now uses RobustRemoveAll which handles permission issues
// caused by Docker containers creating files with different ownership.
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    homeDir, workspaceDir, cleanup := testutil.SetupIsolatedTest(t)
//	    defer cleanup() // Recommended for tests that use Docker
//
//	    // Change to workspace directory for the test
//	    originalWD, _ := os.Getwd()
//	    os.Chdir(workspaceDir)
//	    defer os.Chdir(originalWD)
//	}
func SetupIsolatedTest(t *testing.T) (homeDir, workspaceDir string, cleanup func()) {
	t.Helper()

	// Get original working directory to restore later
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Create isolated directories manually to avoid conflict between
	// Go's t.TempDir() cleanup and our Docker-aware RobustRemoveAll
	tempBase := os.TempDir()

	// Sanitize test name for use in file paths - replace path separators and other invalid chars
	sanitizedTestName := strings.ReplaceAll(t.Name(), "/", "_")
	sanitizedTestName = strings.ReplaceAll(sanitizedTestName, "\\", "_") // Windows compatibility
	testPrefix := fmt.Sprintf("%s_%d_", sanitizedTestName, time.Now().UnixNano())

	homeDir, err = os.MkdirTemp(tempBase, testPrefix+"home_")
	if err != nil {
		t.Fatalf("Failed to create isolated home directory: %v", err)
	}

	workspaceDir, err = os.MkdirTemp(tempBase, testPrefix+"workspace_")
	if err != nil {
		t.Fatalf("Failed to create isolated workspace directory: %v", err)
	}

	// Set HOME environment variable to the temporary directory for this test
	t.Setenv("HOME", homeDir)

	// Register robust cleanup that can handle Docker-created files
	t.Cleanup(func() {
		// Restore original working directory first
		if err := os.Chdir(originalWD); err != nil {
			t.Logf("Warning: Failed to restore original working directory: %v", err)
		}

		// Use robust removal for both directories - this handles Docker-created files
		if err := RobustRemoveAll(t, homeDir); err != nil {
			t.Logf("Warning: Robust cleanup failed for home directory %s: %v", homeDir, err)
		}

		if err := RobustRemoveAll(t, workspaceDir); err != nil {
			t.Logf("Warning: Robust cleanup failed for workspace directory %s: %v", workspaceDir, err)
		}
	})

	// Create legacy cleanup function for backward compatibility
	cleanup = func() {
		// This function is now mostly a no-op since cleanup is registered with t.Cleanup()
		// but we keep it for tests that explicitly call it
		if err := os.Chdir(originalWD); err != nil {
			t.Logf("Warning: Failed to restore original working directory: %v", err)
		}
	}

	return homeDir, workspaceDir, cleanup
}

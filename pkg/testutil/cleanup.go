package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// isSafeToRemove performs comprehensive safety checks before allowing forceful removal
// of a directory path using Docker. This function implements multiple safety mechanisms
// to ensure we only ever delete test-created temporary directories.
func isSafeToRemove(t *testing.T, path string) bool {
	t.Helper()

	// 1. Must be an absolute path to avoid any ambiguity.
	if !filepath.IsAbs(path) {
		t.Logf("Safety Check Failed: Path is not absolute: %s", path)
		return false
	}

	// 2. Resolve any symbolic links in both paths for a canonical comparison.
	evalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Logf("Safety Check Failed: Could not evaluate symlinks for path %s: %v", path, err)
		return false
	}
	evalTempDir, _ := filepath.EvalSymlinks(os.TempDir())

	// 3. The evaluated path MUST be a subdirectory of the OS's temp directory.
	if !strings.HasPrefix(evalPath, evalTempDir) {
		t.Logf("Safety Check Failed: Path is not inside OS temp dir. Path: %s, TempDir: %s", evalPath, evalTempDir)
		return false
	}

	// 4. The path MUST contain the sanitized name of the running test.
	// Go's t.TempDir() creates paths like /tmp/TestName123456/001/
	// We need to check if the test name appears anywhere in the path components.
	sanitizedTestName := strings.ReplaceAll(t.Name(), "/", "_")
	pathContainsTestName := false
	
	// Check all path components from the temp dir down
	relPath, err := filepath.Rel(evalTempDir, evalPath)
	if err == nil {
		pathComponents := strings.Split(relPath, string(filepath.Separator))
		for _, component := range pathComponents {
			if strings.Contains(component, sanitizedTestName) {
				pathContainsTestName = true
				break
			}
		}
	}
	
	if !pathContainsTestName {
		t.Logf("Safety Check Failed: Path does not contain sanitized test name '%s' in any component. Path: %s", sanitizedTestName, evalPath)
		return false
	}

	return true
}

// forceRemoveAll uses a Docker container to forcefully remove a directory when
// standard os.RemoveAll() fails due to permission issues. This is a fallback mechanism
// for test cleanup when containers create files with root ownership.
func forceRemoveAll(t *testing.T, path string) error {
	t.Helper()

	// CRITICAL: Perform comprehensive safety checks
	if !isSafeToRemove(t, path) {
		t.Fatalf("Safety check failed: refusing to force remove path %s", path)
	}

	// Create a new Docker client specifically for cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("Failed to create Docker client for force cleanup: %v", err)
	}
	defer func() { _ = dockerClient.Close() }()

	// Ensure we can connect to Docker daemon
	if _, err := dockerClient.Ping(ctx); err != nil {
		t.Fatalf("Docker daemon not available for force cleanup: %v", err)
	}

	// Pull alpine image if not available (minimal logging to avoid test noise)
	if os.Getenv("REACTOR_VERBOSE_CLEANUP") != "" {
		t.Logf("Force removing directory %s using Docker cleaner container", path)
	}
	
	// Attempt to pull alpine image (this will be fast if already present)
	if os.Getenv("REACTOR_VERBOSE_CLEANUP") != "" {
		t.Logf("Ensuring alpine image is available for cleanup...")
	}
	pullReader, err := dockerClient.ImagePull(ctx, "alpine:latest", types.ImagePullOptions{})
	if err != nil {
		t.Fatalf("Failed to pull alpine image for cleanup: %v", err)
	}
	defer func() { _ = pullReader.Close() }()
	
	// Drain the pull response to complete the operation
	buffer := make([]byte, 1024)
	for {
		_, err := pullReader.Read(buffer)
		if err != nil {
			break // EOF or error, pull completed
		}
	}

	// Create and run the cleaner container
	config := &container.Config{
		Image: "alpine:latest",
		Cmd:   []string{"rm", "-rf", "/work/*"},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{path + ":/work"},
		// Don't use AutoRemove - we'll remove manually after waiting
	}

	containerResp, err := dockerClient.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		t.Fatalf("Failed to create cleaner container: %v", err)
	}

	// Ensure container cleanup even if something fails
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = dockerClient.ContainerRemove(cleanupCtx, containerResp.ID, container.RemoveOptions{Force: true})
	}()

	// Start the container
	if err := dockerClient.ContainerStart(ctx, containerResp.ID, container.StartOptions{}); err != nil {
		t.Fatalf("Failed to start cleaner container: %v", err)
	}

	// Wait for the container to complete
	waitCh, errCh := dockerClient.ContainerWait(ctx, containerResp.ID, container.WaitConditionNotRunning)
	select {
	case result := <-waitCh:
		if result.StatusCode != 0 {
			t.Fatalf("Cleaner container exited with non-zero status: %d", result.StatusCode)
		}
	case err := <-errCh:
		t.Fatalf("Error waiting for cleaner container: %v", err)
	case <-ctx.Done():
		t.Fatalf("Timeout waiting for cleaner container to complete")
	}

	if os.Getenv("REACTOR_VERBOSE_CLEANUP") != "" {
		t.Logf("Successfully force-removed directory using Docker cleaner container")
	}
	return nil
}

// RobustRemoveAll attempts standard removal first, falling back to Docker-based
// force removal if permission errors occur. This is the primary cleanup function
// that should be used for test directories that may contain Docker-created files.
func RobustRemoveAll(t *testing.T, path string) error {
	t.Helper()

	// First attempt: standard Go removal
	if err := os.RemoveAll(path); err != nil {
		// Check if this is a permission error that warrants force removal
		if os.IsPermission(err) {
			// Use verbose logging only when explicitly requested
			if os.Getenv("REACTOR_VERBOSE_CLEANUP") != "" {
				t.Logf("Standard removal failed with permission error, attempting force removal: %v", err)
			}
			
			// Fallback: force removal using Docker cleaner container
			if err := forceRemoveAll(t, path); err != nil {
				return fmt.Errorf("both standard and force removal failed: %w", err)
			}
			return nil
		}
		
		// Other types of errors should be propagated
		return fmt.Errorf("standard removal failed: %w", err)
	}

	// Standard removal succeeded
	return nil
}
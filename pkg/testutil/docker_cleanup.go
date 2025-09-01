package testutil

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dyluth/reactor/pkg/docker"
)

// CleanupTestContainers provides comprehensive cleanup for reactor containers created during integration tests.
// This function implements the self-cleaning container approach to resolve permission issues
// where containers create files with different ownership that Go test cleanup cannot remove.
func CleanupTestContainers(isolationPrefix string) error {
	if isolationPrefix == "" {
		return fmt.Errorf("isolation prefix cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Initialize Docker service
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() { _ = dockerService.Close() }()

	// Check if Docker is available - if not, skip cleanup silently
	if err := dockerService.CheckHealth(ctx); err != nil {
		// Docker not available - this is not an error for tests that don't use Docker
		return nil
	}

	// List all reactor containers
	allContainers, err := dockerService.ListReactorContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list reactor containers: %w", err)
	}

	// Filter containers that match our isolation prefix
	var testContainers []docker.ContainerInfo
	for _, container := range allContainers {
		if strings.Contains(container.Name, isolationPrefix) {
			testContainers = append(testContainers, container)
		}
	}

	if len(testContainers) == 0 {
		// No test containers found - this is normal
		return nil
	}

	fmt.Fprintf(os.Stderr, "Cleaning up %d test containers with prefix %s\n", len(testContainers), isolationPrefix)

	// Clean up each test container using self-cleaning approach
	for _, container := range testContainers {
		if err := cleanupTestContainer(ctx, dockerService, container); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to cleanup container %s: %v\n", container.Name, err)
			// Continue with other containers - don't fail entire cleanup for one container
		}
	}

	return nil
}

// cleanupTestContainer removes a single test container using standard removal
func cleanupTestContainer(ctx context.Context, dockerService *docker.Service, container docker.ContainerInfo) error {
	fmt.Fprintf(os.Stderr, "Cleaning container %s (status: %s)\n", container.Name, container.Status)

	// Use standard container removal - file cleanup is now handled by RobustRemoveAll
	return dockerService.RemoveContainer(ctx, container.ID)
}

// AutoCleanupTestContainers automatically cleans up test containers based on the current
// REACTOR_ISOLATION_PREFIX environment variable. This should be called in test cleanup functions.
func AutoCleanupTestContainers() error {
	isolationPrefix := os.Getenv("REACTOR_ISOLATION_PREFIX")
	if isolationPrefix == "" {
		// No isolation prefix - nothing to clean up
		return nil
	}

	return CleanupTestContainers(isolationPrefix)
}

// CleanupAllTestContainers cleans up all test containers that match common test prefixes.
// This is useful for cleaning up accumulated state from previous failed test runs.
func CleanupAllTestContainers() error {
	testPrefixes := []string{
		"test-",
		"security-test-",
		"test-e2e-",
		"test-multi-",
		"test-verbose-",
		"test-recovery-",
		"test-naming-",
		"test-docker-",
		"test-sessions-",
		"test-sanitize-",
		"test-integration-",
		"test-config-",
		"test-isolation-",
		"test-errors-",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Initialize Docker service
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() { _ = dockerService.Close() }()

	// Check if Docker is available - if not, skip cleanup silently
	if err := dockerService.CheckHealth(ctx); err != nil {
		// Docker not available - this is not an error
		return nil
	}

	// List all reactor containers
	allContainers, err := dockerService.ListReactorContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list reactor containers: %w", err)
	}

	// Filter containers that match test prefixes
	var testContainers []docker.ContainerInfo
	for _, container := range allContainers {
		for _, prefix := range testPrefixes {
			if strings.Contains(container.Name, prefix) {
				testContainers = append(testContainers, container)
				break // Found a match, don't check other prefixes
			}
		}
	}

	if len(testContainers) == 0 {
		// No test containers found
		return nil
	}

	fmt.Fprintf(os.Stderr, "Cleaning up %d accumulated test containers\n", len(testContainers))

	// Clean up each test container using self-cleaning approach
	for _, container := range testContainers {
		if err := cleanupTestContainer(ctx, dockerService, container); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to cleanup container %s: %v\n", container.Name, err)
			// Continue with other containers - don't fail entire cleanup for one container
		}
	}

	return nil
}

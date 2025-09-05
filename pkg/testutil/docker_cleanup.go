package testutil

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dyluth/reactor/pkg/docker"
)

// CleanupTestContainers provides comprehensive cleanup for reactor containers created during integration tests.
// This function uses Docker labels to identify test containers, providing a more robust cleanup mechanism
// that doesn't rely on name patterns which can be unreliable.
func CleanupTestContainers(isolationPrefix string) error {
	// Note: isolationPrefix is kept for backward compatibility but we now use Docker labels
	// for more reliable test container identification

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

	// Find all containers with the test label
	testContainers, err := dockerService.ListContainersByLabel(ctx, "com.reactor.test", "true")
	if err != nil {
		return fmt.Errorf("failed to list test containers by label: %w", err)
	}

	if len(testContainers) == 0 {
		// No test containers found - this is normal
		return nil
	}

	fmt.Fprintf(os.Stderr, "Cleaning up %d test containers with test label\n", len(testContainers))

	// Clean up each test container with force removal
	for _, container := range testContainers {
		if err := cleanupTestContainer(ctx, dockerService, container); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to cleanup container %s: %v\n", container.Name, err)
			// Continue with other containers - don't fail entire cleanup for one container
		}
	}

	return nil
}

// cleanupTestContainer removes a single test container with force removal to ensure cleanup
func cleanupTestContainer(ctx context.Context, dockerService *docker.Service, container docker.ContainerInfo) error {
	fmt.Fprintf(os.Stderr, "Force cleaning container %s (status: %s)\n", container.Name, container.Status)

	// Use force removal to ensure container is cleaned up even if running
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

// CleanupAllTestContainers cleans up all test containers using Docker labels.
// This is useful for cleaning up accumulated state from previous failed test runs.
func CleanupAllTestContainers() error {
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

	// Find all containers with the test label
	testContainers, err := dockerService.ListContainersByLabel(ctx, "com.reactor.test", "true")
	if err != nil {
		return fmt.Errorf("failed to list test containers by label: %w", err)
	}

	if len(testContainers) == 0 {
		// No test containers found
		return nil
	}

	fmt.Fprintf(os.Stderr, "Cleaning up %d accumulated test containers with test label\n", len(testContainers))

	// Clean up each test container with force removal
	for _, container := range testContainers {
		if err := cleanupTestContainer(ctx, dockerService, container); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to cleanup container %s: %v\n", container.Name, err)
			// Continue with other containers - don't fail entire cleanup for one container
		}
	}

	return nil
}

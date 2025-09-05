package integration

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/dyluth/reactor/pkg/docker"
)

// TestMain provides global test setup and cleanup for integration tests.
// This ensures that all test containers are cleaned up even if tests panic.
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Perform global cleanup after all tests complete
	if err := globalTestCleanup(); err != nil {
		log.Printf("Warning: Global test cleanup failed: %v", err)
	}

	os.Exit(code)
}

// globalTestCleanup finds and force-removes all containers with com.reactor.test=true label
func globalTestCleanup() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Initialize Docker service
	dockerService, err := docker.NewService()
	if err != nil {
		// If Docker service fails to initialize, try direct docker command as fallback
		return fallbackDockerCleanup()
	}
	defer func() { _ = dockerService.Close() }()

	// Check if Docker is available
	if err := dockerService.CheckHealth(ctx); err != nil {
		// Docker not available - use fallback
		return fallbackDockerCleanup()
	}

	// Find all containers with the test label using Docker SDK
	testContainers, err := dockerService.ListContainersByLabel(ctx, "com.reactor.test", "true")
	if err != nil {
		// SDK failed - use fallback
		return fallbackDockerCleanup()
	}

	if len(testContainers) == 0 {
		// No test containers found - this is normal
		return nil
	}

	fmt.Printf("Global cleanup: Found %d test containers to remove\n", len(testContainers))

	// Force remove each test container
	for _, container := range testContainers {
		fmt.Printf("Global cleanup: Force removing container %s (%s)\n", container.Name, container.ID)
		if err := dockerService.RemoveContainer(ctx, container.ID); err != nil {
			fmt.Printf("Warning: Failed to remove container %s: %v\n", container.Name, err)
			// Continue with other containers
		}
	}

	return nil
}

// fallbackDockerCleanup uses direct docker commands as a fallback when SDK fails
func fallbackDockerCleanup() error {
	// Find containers with test label
	cmd := exec.Command("docker", "ps", "-aq", "--filter", "label=com.reactor.test=true")
	output, err := cmd.Output()
	if err != nil {
		// Docker command failed - this might be normal if Docker isn't available
		return fmt.Errorf("failed to find test containers via docker command: %w", err)
	}

	containerIDs := string(output)
	if containerIDs == "" {
		// No containers found
		return nil
	}

	// Force remove all test containers
	fmt.Println("Global cleanup: Using fallback docker command to remove test containers")
	cmd = exec.Command("docker", "rm", "-f")
	cmd.Stdin = strings.NewReader(containerIDs)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to force remove test containers: %w", err)
	}

	fmt.Println("Global cleanup: Successfully removed test containers via fallback")
	return nil
}

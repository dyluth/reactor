package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/dyluth/reactor/pkg/config"
	"github.com/dyluth/reactor/pkg/docker"
	"github.com/dyluth/reactor/pkg/orchestrator"
	"github.com/dyluth/reactor/pkg/testutil"
	"github.com/dyluth/reactor/pkg/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceUpBasicFunctionality(t *testing.T) {
	testutil.SetupIsolatedTest(t)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Create workspace structure
	tmpDir, err := os.MkdirTemp("", "workspace-up-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		require.NoError(t, err)
	})

	// Setup complete workspace structure
	setupTestWorkspaceWithImages(t, tmpDir)

	workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")

	// Parse workspace to get services
	ws, err := workspace.ParseWorkspaceFile(workspaceFile)
	require.NoError(t, err)
	require.Len(t, ws.Services, 2)

	// Generate workspace hash for container tracking
	workspaceHash, err := workspace.GenerateWorkspaceHash(workspaceFile)
	require.NoError(t, err)
	require.NotEmpty(t, workspaceHash)

	ctx := context.Background()
	dockerService, err := docker.NewService()
	require.NoError(t, err)
	defer func() {
		err := dockerService.Close()
		require.NoError(t, err)
	}()

	// Test starting individual services using orchestrator directly
	// This mimics what workspace up command does
	successCount := 0

	for serviceName, service := range ws.Services {
		// Resolve service path
		servicePath := service.Path
		if !filepath.IsAbs(servicePath) {
			servicePath = filepath.Join(tmpDir, service.Path)
		}
		servicePath = filepath.Clean(servicePath)

		// Create service-specific orchestrator config
		serviceConfig := orchestrator.UpConfig{
			ProjectDirectory:      servicePath,
			AccountOverride:       service.Account,
			NamePrefix:            fmt.Sprintf("reactor-ws-%s-", serviceName),
			ForceRebuild:          false,
			DiscoveryMode:         true, // Use discovery mode for testing (no mounts)
			DockerHostIntegration: false,
			Verbose:               true,
		}

		// Add workspace labels
		serviceConfig.Labels = make(map[string]string)
		serviceConfig.Labels["com.reactor.workspace.instance"] = workspaceHash
		serviceConfig.Labels["com.reactor.workspace.service"] = serviceName
		serviceConfig.Labels["com.reactor.test"] = "true" // For cleanup

		t.Run(fmt.Sprintf("StartService_%s", serviceName), func(t *testing.T) {
			// Start the service
			resolved, containerID, err := orchestrator.Up(ctx, serviceConfig)
			require.NoError(t, err)
			require.NotEmpty(t, containerID)
			require.NotNil(t, resolved)

			// Verify workspace labels are applied using Docker API directly
			client := dockerService.GetClient()
			containerDetails, err := client.ContainerInspect(ctx, containerID)
			require.NoError(t, err)
			assert.Equal(t, workspaceHash, containerDetails.Config.Labels["com.reactor.workspace.instance"])
			assert.Equal(t, serviceName, containerDetails.Config.Labels["com.reactor.workspace.service"])

			// Verify container name follows workspace convention by checking the actual running container
			actualContainerName := containerDetails.Name
			if actualContainerName != "" && actualContainerName[0] == '/' {
				actualContainerName = actualContainerName[1:] // Remove leading slash if present
			}
			expectedNamePattern := fmt.Sprintf("reactor-ws-%s-", serviceName)
			assert.Contains(t, actualContainerName, expectedNamePattern)

			// Also verify the container is actually running by checking state directly
			assert.True(t, containerDetails.State.Running, "Container should be running")
			assert.Equal(t, "running", containerDetails.State.Status)

			successCount++
		})
	}

	// Verify both services started successfully
	assert.Equal(t, 2, successCount)

	// Test workspace hash consistency
	workspaceHash2, err := workspace.GenerateWorkspaceHash(workspaceFile)
	require.NoError(t, err)
	assert.Equal(t, workspaceHash, workspaceHash2, "Workspace hash should be consistent")
}

func TestWorkspaceUpPortConflictDetection(t *testing.T) {
	testutil.SetupIsolatedTest(t)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Create workspace with port conflicts
	tmpDir, err := os.MkdirTemp("", "workspace-port-conflict-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		require.NoError(t, err)
	})

	// Create service directories with conflicting ports
	setupWorkspaceWithPortConflicts(t, tmpDir)

	workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")

	// Parse workspace
	ws, err := workspace.ParseWorkspaceFile(workspaceFile)
	require.NoError(t, err)

	// Test port conflict detection logic
	workspaceDir := filepath.Dir(workspaceFile)
	allHostPorts := make(map[int][]string)

	// Simulate the port validation logic from workspace up
	for serviceName, service := range ws.Services {
		servicePath := service.Path
		if !filepath.IsAbs(servicePath) {
			servicePath = filepath.Join(workspaceDir, service.Path)
		}
		servicePath = filepath.Clean(servicePath)

		// Load service configuration
		configService := config.NewServiceWithRoot(servicePath)
		resolved, err := configService.ResolveConfiguration()
		require.NoError(t, err)

		// Collect port mappings
		for _, port := range resolved.ForwardPorts {
			if existing, exists := allHostPorts[port.HostPort]; exists {
				allHostPorts[port.HostPort] = append(existing, serviceName)
			} else {
				allHostPorts[port.HostPort] = []string{serviceName}
			}
		}
	}

	// Check for conflicts
	conflictFound := false
	for hostPort, services := range allHostPorts {
		if len(services) > 1 {
			t.Logf("Port conflict detected: port %d used by services: %v", hostPort, services)
			conflictFound = true
		}
	}

	assert.True(t, conflictFound, "Expected port conflicts to be detected")
}

func TestWorkspaceContainerNamingConventions(t *testing.T) {
	testutil.SetupIsolatedTest(t)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Create workspace structure
	tmpDir, err := os.MkdirTemp("", "workspace-naming-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		require.NoError(t, err)
	})

	setupTestWorkspaceWithImages(t, tmpDir)

	workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
	ws, err := workspace.ParseWorkspaceFile(workspaceFile)
	require.NoError(t, err)

	// Test container name generation
	for serviceName, service := range ws.Services {
		servicePath := service.Path
		if !filepath.IsAbs(servicePath) {
			servicePath = filepath.Join(tmpDir, service.Path)
		}
		servicePath = filepath.Clean(servicePath)

		// Generate expected container name
		projectHash := config.GenerateProjectHash(servicePath)
		expectedContainerName := fmt.Sprintf("reactor-ws-%s-%s", serviceName, projectHash)

		// Verify name components
		assert.NotEmpty(t, projectHash)
		assert.Len(t, projectHash, 8, "Project hash should be 8 characters")
		assert.Contains(t, expectedContainerName, serviceName)
		assert.Contains(t, expectedContainerName, "reactor-ws-")
		assert.Contains(t, expectedContainerName, projectHash)

		t.Logf("Service %s: expected container name %s", serviceName, expectedContainerName)
	}
}

// setupTestWorkspaceWithImages creates a complete workspace structure for testing
func setupTestWorkspaceWithImages(t *testing.T, tmpDir string) {
	// Create service directories
	apiDir := filepath.Join(tmpDir, "services", "api", ".devcontainer")
	frontendDir := filepath.Join(tmpDir, "services", "frontend", ".devcontainer")
	require.NoError(t, os.MkdirAll(apiDir, 0755))
	require.NoError(t, os.MkdirAll(frontendDir, 0755))

	// Create devcontainer.json files with simple images (no port conflicts)
	apiDevcontainer := filepath.Join(apiDir, "devcontainer.json")
	frontendDevcontainer := filepath.Join(frontendDir, "devcontainer.json")

	err := os.WriteFile(apiDevcontainer, []byte(`{
		"name": "api-service",
		"image": "node:18-alpine",
		"remoteUser": "node",
		"customizations": {
			"reactor": {
				"account": "api-account",
				"defaultCommand": "sleep infinity"
			}
		}
	}`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(frontendDevcontainer, []byte(`{
		"name": "frontend-service",
		"image": "node:18-alpine", 
		"remoteUser": "node",
		"customizations": {
			"reactor": {
				"account": "frontend-account",
				"defaultCommand": "sleep infinity"
			}
		}
	}`), 0644)
	require.NoError(t, err)

	// Create workspace file
	workspaceContent := `version: "1"
services:
  api:
    path: ./services/api
    account: work-account
  frontend:
    path: ./services/frontend`

	workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
	err = os.WriteFile(workspaceFile, []byte(workspaceContent), 0644)
	require.NoError(t, err)
}

// setupWorkspaceWithPortConflicts creates a workspace with conflicting ports
func setupWorkspaceWithPortConflicts(t *testing.T, tmpDir string) {
	// Create service directories
	service1Dir := filepath.Join(tmpDir, "service1", ".devcontainer")
	service2Dir := filepath.Join(tmpDir, "service2", ".devcontainer")
	require.NoError(t, os.MkdirAll(service1Dir, 0755))
	require.NoError(t, os.MkdirAll(service2Dir, 0755))

	// Both services use port 3000 - this should cause conflict
	service1Devcontainer := filepath.Join(service1Dir, "devcontainer.json")
	service2Devcontainer := filepath.Join(service2Dir, "devcontainer.json")

	err := os.WriteFile(service1Devcontainer, []byte(`{
		"name": "service1",
		"image": "node:18-alpine",
		"forwardPorts": [3000]
	}`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(service2Devcontainer, []byte(`{
		"name": "service2", 
		"image": "node:18-alpine",
		"forwardPorts": [3000]
	}`), 0644)
	require.NoError(t, err)

	// Create workspace file
	workspaceContent := `version: "1"
services:
  service1:
    path: ./service1
  service2:
    path: ./service2`

	workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
	err = os.WriteFile(workspaceFile, []byte(workspaceContent), 0644)
	require.NoError(t, err)
}
func TestWorkspaceExecBasicFunctionality(t *testing.T) {
	testutil.SetupIsolatedTest(t)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Create workspace structure
	tmpDir, err := os.MkdirTemp("", "workspace-exec-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		require.NoError(t, err)
	})

	setupTestWorkspaceWithImages(t, tmpDir)

	workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
	ws, err := workspace.ParseWorkspaceFile(workspaceFile)
	require.NoError(t, err)

	// Generate workspace hash for container labeling
	workspaceHash, err := workspace.GenerateWorkspaceHash(workspaceFile)
	require.NoError(t, err)

	ctx := context.Background()
	dockerService, err := docker.NewService()
	require.NoError(t, err)
	defer func() {
		err := dockerService.Close()
		require.NoError(t, err)
	}()

	// Start one service first
	serviceName := "api"
	service := ws.Services[serviceName]

	// Resolve service path
	servicePath := service.Path
	if !filepath.IsAbs(servicePath) {
		servicePath = filepath.Join(tmpDir, service.Path)
	}
	servicePath = filepath.Clean(servicePath)

	// Create service-specific orchestrator config
	serviceConfig := orchestrator.UpConfig{
		ProjectDirectory:      servicePath,
		AccountOverride:       service.Account,
		NamePrefix:            fmt.Sprintf("reactor-ws-%s-", serviceName),
		ForceRebuild:          false,
		DiscoveryMode:         true, // Use discovery mode for testing
		DockerHostIntegration: false,
		Verbose:               false,
	}

	// Add workspace labels
	serviceConfig.Labels = make(map[string]string)
	serviceConfig.Labels["com.reactor.workspace.instance"] = workspaceHash
	serviceConfig.Labels["com.reactor.workspace.service"] = serviceName
	serviceConfig.Labels["com.reactor.test"] = "true" // For cleanup

	// Start the service
	resolved, containerID, err := orchestrator.Up(ctx, serviceConfig)
	require.NoError(t, err)
	require.NotEmpty(t, containerID)
	require.NotNil(t, resolved)

	// Now test exec functionality by simulating what the exec command would do

	// Verify the container is running using the container ID we got from orchestrator.Up
	client := dockerService.GetClient()
	containerDetails, err := client.ContainerInspect(ctx, containerID)
	require.NoError(t, err)
	require.True(t, containerDetails.State.Running, "Container should be running")

	// Test simple command execution using the docker service directly
	command := []string{"echo", "hello from workspace exec"}

	// Create exec config for simple command (not interactive)
	execConfig := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          command,
	}

	execResp, err := client.ContainerExecCreate(ctx, containerID, execConfig)
	require.NoError(t, err)

	// Start the exec
	err = client.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{})
	require.NoError(t, err)

	// Wait for completion and check exit code
	inspectResp, err := client.ContainerExecInspect(ctx, execResp.ID)
	require.NoError(t, err)

	// If it's still running, wait a bit
	if inspectResp.Running {
		time.Sleep(100 * time.Millisecond)
		inspectResp, err = client.ContainerExecInspect(ctx, execResp.ID)
		require.NoError(t, err)
	}

	// Verify command executed successfully
	assert.False(t, inspectResp.Running, "Command should have completed")
	assert.Equal(t, 0, inspectResp.ExitCode, "Command should succeed")

	t.Logf("Exec test completed successfully for service %s in container %s", serviceName, containerDetails.Name)
}

func TestWorkspaceFullLifecycleEndToEnd(t *testing.T) {
	testutil.SetupIsolatedTest(t)

	// Ensure Docker cleanup runs after test completion
	t.Cleanup(func() {
		if err := testutil.CleanupAllTestContainers(); err != nil {
			t.Logf("Warning: failed to cleanup test containers: %v", err)
		}
	})

	// Create workspace structure
	tmpDir, err := os.MkdirTemp("", "workspace-e2e-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		require.NoError(t, err)
	})

	setupTestWorkspaceWithImages(t, tmpDir)

	workspaceFile := filepath.Join(tmpDir, "reactor-workspace.yml")
	ws, err := workspace.ParseWorkspaceFile(workspaceFile)
	require.NoError(t, err)

	// Generate workspace hash for container tracking
	workspaceHash, err := workspace.GenerateWorkspaceHash(workspaceFile)
	require.NoError(t, err)

	ctx := context.Background()
	dockerService, err := docker.NewService()
	require.NoError(t, err)
	defer func() {
		err := dockerService.Close()
		require.NoError(t, err)
	}()

	// Step 1: Start all services using workspace up
	t.Logf("=== Step 1: Starting workspace services ===")

	for serviceName, service := range ws.Services {
		// Resolve service path
		servicePath := service.Path
		if !filepath.IsAbs(servicePath) {
			servicePath = filepath.Join(tmpDir, service.Path)
		}
		servicePath = filepath.Clean(servicePath)

		// Create service-specific orchestrator config
		serviceConfig := orchestrator.UpConfig{
			ProjectDirectory:      servicePath,
			AccountOverride:       service.Account,
			NamePrefix:            fmt.Sprintf("reactor-ws-%s-", serviceName),
			ForceRebuild:          false,
			DiscoveryMode:         true, // Use discovery mode for testing
			DockerHostIntegration: false,
			Verbose:               false,
		}

		// Add workspace labels
		serviceConfig.Labels = make(map[string]string)
		serviceConfig.Labels["com.reactor.workspace.instance"] = workspaceHash
		serviceConfig.Labels["com.reactor.workspace.service"] = serviceName
		serviceConfig.Labels["com.reactor.test"] = "true" // For cleanup

		// Start the service
		resolved, containerID, err := orchestrator.Up(ctx, serviceConfig)
		require.NoError(t, err, "Failed to start service %s", serviceName)
		require.NotEmpty(t, containerID, "Container ID should not be empty for service %s", serviceName)
		require.NotNil(t, resolved, "Resolved config should not be nil for service %s", serviceName)

		t.Logf("✅ Started service %s with container ID %s", serviceName, containerID[:12])
	}

	// Step 2: Verify all containers are running and have correct labels
	t.Logf("=== Step 2: Verifying all containers are running ===")
	client := dockerService.GetClient()

	for serviceName := range ws.Services {
		// Find containers using workspace labels
		filterArgs := filters.NewArgs()
		filterArgs.Add("label", fmt.Sprintf("com.reactor.workspace.instance=%s", workspaceHash))
		filterArgs.Add("label", fmt.Sprintf("com.reactor.workspace.service=%s", serviceName))
		filterArgs.Add("status", "running")

		containers, err := client.ContainerList(ctx, container.ListOptions{
			Filters: filterArgs,
		})
		require.NoError(t, err, "Failed to list containers for service %s", serviceName)
		require.Len(t, containers, 1, "Expected exactly 1 running container for service %s", serviceName)

		container := containers[0]
		assert.Equal(t, workspaceHash, container.Labels["com.reactor.workspace.instance"],
			"Workspace instance label should match for service %s", serviceName)
		assert.Equal(t, serviceName, container.Labels["com.reactor.workspace.service"],
			"Service label should match for service %s", serviceName)
		assert.Equal(t, "running", container.State, "Container should be running for service %s", serviceName)

		t.Logf("✅ Verified service %s is running with correct labels", serviceName)
	}

	// Step 3: Execute a command in one of the services to verify exec functionality
	t.Logf("=== Step 3: Testing exec functionality ===")
	testServiceName := "api" // Use API service for exec test

	// Find the container for exec test
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.reactor.workspace.instance=%s", workspaceHash))
	filterArgs.Add("label", fmt.Sprintf("com.reactor.workspace.service=%s", testServiceName))

	execContainers, err := client.ContainerList(ctx, container.ListOptions{
		Filters: filterArgs,
	})
	require.NoError(t, err, "Failed to list containers for exec test")
	require.Len(t, execContainers, 1, "Expected exactly 1 container for exec test")

	execContainer := execContainers[0]
	require.True(t, execContainer.State == "running", "Container should be running for exec test")

	// Test simple command execution
	execConfig := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"echo", "workspace-exec-test"},
	}

	execResp, err := client.ContainerExecCreate(ctx, execContainer.ID, execConfig)
	require.NoError(t, err, "Failed to create exec")

	err = client.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{})
	require.NoError(t, err, "Failed to start exec")

	// Wait for completion
	time.Sleep(100 * time.Millisecond)
	inspectResp, err := client.ContainerExecInspect(ctx, execResp.ID)
	require.NoError(t, err, "Failed to inspect exec")
	assert.Equal(t, 0, inspectResp.ExitCode, "Exec command should succeed")

	t.Logf("✅ Successfully executed command in service %s", testServiceName)

	// Step 4: Stop all services using workspace down functionality
	t.Logf("=== Step 4: Stopping workspace services ===")

	// Simulate the workspace down functionality by stopping containers with workspace labels
	allFilterArgs := filters.NewArgs()
	allFilterArgs.Add("label", fmt.Sprintf("com.reactor.workspace.instance=%s", workspaceHash))

	allContainers, err := client.ContainerList(ctx, container.ListOptions{
		Filters: allFilterArgs,
		All:     true,
	})
	require.NoError(t, err, "Failed to list all workspace containers")

	// Stop and remove all containers
	for _, container := range allContainers {
		serviceName := container.Labels["com.reactor.workspace.service"]
		t.Logf("Stopping service %s container %s", serviceName, container.ID[:12])

		// Stop the container
		if container.State == "running" {
			err := dockerService.StopContainer(ctx, container.ID)
			require.NoError(t, err, "Failed to stop container for service %s", serviceName)
		}

		// Remove the container
		err := dockerService.RemoveContainer(ctx, container.ID)
		require.NoError(t, err, "Failed to remove container for service %s", serviceName)

		t.Logf("✅ Stopped and removed service %s", serviceName)
	}

	// Step 5: Verify all containers have been removed
	t.Logf("=== Step 5: Verifying all containers are removed ===")

	for serviceName := range ws.Services {
		// Check that no containers exist for this service
		serviceFilterArgs := filters.NewArgs()
		serviceFilterArgs.Add("label", fmt.Sprintf("com.reactor.workspace.instance=%s", workspaceHash))
		serviceFilterArgs.Add("label", fmt.Sprintf("com.reactor.workspace.service=%s", serviceName))

		remainingContainers, err := client.ContainerList(ctx, container.ListOptions{
			Filters: serviceFilterArgs,
			All:     true, // Include stopped containers
		})
		require.NoError(t, err, "Failed to list containers after cleanup")
		assert.Len(t, remainingContainers, 0, "No containers should remain for service %s", serviceName)

		t.Logf("✅ Verified service %s containers are completely removed", serviceName)
	}

	t.Logf("=== End-to-End Test Complete ===")
	t.Logf("✅ Successfully tested full workspace lifecycle:")
	t.Logf("   - workspace up (all services started)")
	t.Logf("   - container verification (labels and state)")
	t.Logf("   - exec functionality (command execution)")
	t.Logf("   - workspace down (all services stopped)")
	t.Logf("   - cleanup verification (all containers removed)")
}

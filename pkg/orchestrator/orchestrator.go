package orchestrator

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dyluth/reactor/pkg/config"
	"github.com/dyluth/reactor/pkg/core"
	"github.com/dyluth/reactor/pkg/docker"
)

// UpConfig contains all necessary, pre-resolved parameters for an 'up' operation.
type UpConfig struct {
	// The absolute path to the service's project directory (the one containing .devcontainer).
	ProjectDirectory string

	// An optional account override from the workspace file. If empty, the account
	// from the devcontainer.json file will be used.
	AccountOverride string

	// A flag to force a rebuild of the container image.
	ForceRebuild bool

	// An optional map of labels to apply to the container (for workspace tracking).
	Labels map[string]string

	// An optional name prefix for the container (e.g., "reactor-ws-api-").
	NamePrefix string

	// CLI-provided port mappings that override devcontainer.json ports
	CLIPortMappings []string

	// Enable discovery mode (no mounts)
	DiscoveryMode bool

	// Enable Docker host integration (dangerous)
	DockerHostIntegration bool

	// Enable verbose output
	Verbose bool
}

// PortMapping represents a port forwarding configuration
type PortMapping struct {
	HostPort      int
	ContainerPort int
}

// Up orchestrates the entire 'reactor up' logic for a single service.
// It returns the final resolved config and container ID on success.
func Up(ctx context.Context, upConfig UpConfig) (*config.ResolvedConfig, string, error) {
	// Check dependencies first
	if err := config.CheckDependencies(); err != nil {
		return nil, "", err
	}

	// Validate flag combinations
	if upConfig.DiscoveryMode {
		if len(upConfig.CLIPortMappings) > 0 {
			return nil, "", fmt.Errorf("discovery mode cannot be used with port forwarding")
		}
		if upConfig.DockerHostIntegration {
			return nil, "", fmt.Errorf("discovery mode cannot be used with docker host integration")
		}
	}

	// Parse and validate CLI port mappings
	cliPorts, err := parsePortMappings(upConfig.CLIPortMappings)
	if err != nil {
		return nil, "", fmt.Errorf("CLI port mapping error: %w", err)
	}

	// Load and validate configuration using new devcontainer.json workflow
	// First change to the project directory to ensure relative paths work correctly
	originalWD, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	if err := os.Chdir(upConfig.ProjectDirectory); err != nil {
		return nil, "", fmt.Errorf("failed to change to project directory %s: %w", upConfig.ProjectDirectory, err)
	}

	configService := config.NewService()
	resolved, err := configService.ResolveConfiguration()
	if err != nil {
		return nil, "", err
	}

	// Apply account override if provided
	if upConfig.AccountOverride != "" {
		resolved.Account = upConfig.AccountOverride
		// TODO: In future milestones, we might need to recalculate paths when account changes
	}

	// Merge devcontainer.json ports with CLI ports (CLI takes precedence on conflicts)
	finalPorts := mergePortMappings(resolved.ForwardPorts, cliPorts)

	// Check for port conflicts on final merged list
	if len(finalPorts) > 0 {
		conflictPorts := checkPortConflicts(finalPorts)
		if len(conflictPorts) > 0 {
			fmt.Printf("⚠️  WARNING: The following host ports may already be in use:\n")
			for _, port := range conflictPorts {
				fmt.Printf("   Port %d - containers may fail to start or port forwarding may not work\n", port)
			}
			fmt.Printf("   Consider using different host ports or stopping conflicting services.\n\n")
		}
	}

	// Security warning for Docker host integration
	if upConfig.DockerHostIntegration {
		fmt.Printf("⚠️  WARNING: Docker host integration enabled!\n")
		fmt.Printf("   This gives the container full access to your host Docker daemon.\n")
		fmt.Printf("   Only use this flag with trusted images and AI agents.\n")
		fmt.Printf("   The container can create, modify, and delete other containers.\n\n")
	}

	// Display resolved configuration for debugging
	if upConfig.Verbose {
		fmt.Printf("Resolved configuration:\n")
		fmt.Printf("  Provider: %s\n", resolved.Provider.Name)
		fmt.Printf("  Account: %s\n", resolved.Account)
		fmt.Printf("  Image: %s\n", resolved.Image)
		fmt.Printf("  Danger: %t\n", resolved.Danger)
		fmt.Printf("  Project: %s\n", resolved.ProjectRoot)
		fmt.Printf("  Config Dir: %s\n", resolved.ProjectConfigDir)
		if upConfig.ForceRebuild {
			fmt.Printf("  Rebuild: enabled\n")
		}
	}

	// Initialize Docker service
	dockerService, err := docker.NewService()
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() {
		if err := dockerService.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close Docker service: %v\n", err)
		}
	}()

	// Check Docker daemon health
	if err := dockerService.CheckHealth(ctx); err != nil {
		return nil, "", fmt.Errorf("docker daemon not available: %w", err)
	}

	// Handle image building if build configuration is present
	finalImageName := resolved.Image // Default to resolved image
	if resolved.Build != nil {
		// Build takes precedence over image
		buildSpec, err := createBuildSpecFromConfig(resolved)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create build specification: %w", err)
		}

		// Check if we should force rebuild
		forceRebuild := upConfig.ForceRebuild
		if err := dockerService.BuildImage(ctx, buildSpec, forceRebuild); err != nil {
			return nil, "", fmt.Errorf("build failed: %w", err)
		}

		// Use the built image for container creation
		finalImageName = buildSpec.ImageName
		if upConfig.Verbose {
			fmt.Printf("[INFO] Using built image: %s\n", finalImageName)
		}
	}

	// Update resolved config to use final image name
	resolved.Image = finalImageName

	// Convert final merged port mappings to core format
	corePortMappings := make([]core.PortMapping, len(finalPorts))
	for i, pm := range finalPorts {
		corePortMappings[i] = core.PortMapping{
			HostPort:      pm.HostPort,
			ContainerPort: pm.ContainerPort,
		}
	}

	// Create container blueprint with internal mount construction
	blueprint := core.NewContainerBlueprint(resolved, upConfig.DiscoveryMode, upConfig.DockerHostIntegration, corePortMappings)
	containerSpec := blueprint.ToContainerSpec()

	// Apply workspace labels if provided
	if len(upConfig.Labels) > 0 {
		if containerSpec.Labels == nil {
			containerSpec.Labels = make(map[string]string)
		}
		for k, v := range upConfig.Labels {
			containerSpec.Labels[k] = v
		}
	}

	// Apply name prefix if provided
	if upConfig.NamePrefix != "" {
		containerSpec.Name = upConfig.NamePrefix + containerSpec.Name
	}

	// Enhanced verbose output showing container naming and discovery
	if upConfig.Verbose {
		fmt.Printf("[INFO] Project: %s (%s)\n", filepath.Base(resolved.ProjectRoot), resolved.ProjectRoot)
		fmt.Printf("[INFO] Container name: %s\n", containerSpec.Name)
		if upConfig.DiscoveryMode {
			fmt.Printf("[INFO] Discovery mode: no mounts will be created\n")
		}
		if upConfig.DockerHostIntegration {
			fmt.Printf("[INFO] Docker host integration: Docker socket will be mounted\n")
		}
		if len(finalPorts) > 0 {
			fmt.Printf("[INFO] Port forwarding: ")
			for i, pm := range finalPorts {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%d->%d", pm.HostPort, pm.ContainerPort)
			}
			fmt.Printf("\n")
		}
	}

	// Check for existing container first for enhanced verbose feedback
	if upConfig.Verbose {
		existingContainer, err := dockerService.ContainerExists(ctx, containerSpec.Name)
		if err == nil {
			switch existingContainer.Status {
			case docker.StatusRunning:
				fmt.Printf("[INFO] Found existing container: running\n")
			case docker.StatusStopped:
				fmt.Printf("[INFO] Found existing container: stopped (will be restarted)\n")
			case docker.StatusNotFound:
				fmt.Printf("[INFO] No existing container found (will create new one)\n")
			}
		}
	}

	// Provision container using recovery strategy (with cleanup for discovery mode)
	var containerInfo docker.ContainerInfo
	if upConfig.DiscoveryMode {
		// In discovery mode, check if we need to clean up existing container
		existingContainer, checkErr := dockerService.ContainerExists(ctx, containerSpec.Name)
		if checkErr == nil && existingContainer.Status != docker.StatusNotFound {
			fmt.Printf("Discovery mode: removing existing container for clean environment\n")
		}
		containerInfo, err = dockerService.ProvisionContainerWithCleanup(ctx, containerSpec, true)
	} else {
		containerInfo, err = dockerService.ProvisionContainer(ctx, containerSpec)
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to provision container: %w", err)
	}

	fmt.Printf("Container provisioned: %s\n", containerInfo.Name)
	if upConfig.Verbose {
		fmt.Printf("Container ID: %s\n", containerInfo.ID)
		fmt.Printf("Status: %s\n", containerInfo.Status)
	}

	// Execute postCreateCommand if specified
	if resolved.PostCreateCommand != nil {
		if upConfig.Verbose {
			fmt.Printf("[INFO] Executing postCreateCommand...\n")
		} else {
			fmt.Printf("Running postCreateCommand...\n")
		}

		if err := dockerService.ExecutePostCreateCommand(ctx, containerInfo.ID, resolved.PostCreateCommand); err != nil {
			return nil, "", fmt.Errorf("postCreateCommand execution failed: %w", err)
		}

		if upConfig.Verbose {
			fmt.Printf("[INFO] postCreateCommand completed successfully\n")
		} else {
			fmt.Printf("postCreateCommand completed.\n")
		}
	}

	return resolved, containerInfo.ID, nil
}

// Down orchestrates the 'reactor down' logic for a single service.
func Down(ctx context.Context, projectDirectory string) error {
	// Check dependencies first
	if err := config.CheckDependencies(); err != nil {
		return err
	}

	// Load configuration to get container information
	// First change to the project directory to ensure relative paths work correctly
	originalWD, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	if err := os.Chdir(projectDirectory); err != nil {
		return fmt.Errorf("failed to change to project directory %s: %w", projectDirectory, err)
	}

	configService := config.NewService()
	resolved, err := configService.ResolveConfiguration()
	if err != nil {
		return err
	}

	// Initialize Docker service
	dockerService, err := docker.NewService()
	if err != nil {
		return fmt.Errorf("failed to initialize Docker service: %w", err)
	}
	defer func() {
		if err := dockerService.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close Docker service: %v\n", err)
		}
	}()

	// Check Docker daemon health
	if err := dockerService.CheckHealth(ctx); err != nil {
		return fmt.Errorf("docker daemon not available: %w", err)
	}

	// Create a basic container blueprint to get the expected container name
	blueprint := core.NewContainerBlueprint(resolved, false, false, nil)
	containerSpec := blueprint.ToContainerSpec()

	// Check if container exists
	containerInfo, err := dockerService.ContainerExists(ctx, containerSpec.Name)
	if err != nil {
		return fmt.Errorf("failed to check container existence: %w", err)
	}

	if containerInfo.Status == docker.StatusNotFound {
		fmt.Printf("No container found for project: %s\n", containerSpec.Name)
		return nil
	}

	// Stop and remove the container
	fmt.Printf("Stopping and removing container: %s\n", containerInfo.Name)
	if err := dockerService.RemoveContainer(ctx, containerInfo.ID); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	fmt.Printf("Container removed successfully.\n")
	return nil
}

// parsePortMappings parses and validates port mapping strings in the format "host:container"
func parsePortMappings(portStrings []string) ([]PortMapping, error) {
	var mappings []PortMapping

	for _, portStr := range portStrings {
		parts := strings.Split(portStr, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid port mapping format '%s': expected 'host:container'", portStr)
		}

		hostPort, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid host port '%s': must be a number", parts[0])
		}

		containerPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid container port '%s': must be a number", parts[1])
		}

		// Validate port ranges
		if hostPort < 1 || hostPort > 65535 {
			return nil, fmt.Errorf("host port %d is out of valid range (1-65535)", hostPort)
		}
		if containerPort < 1 || containerPort > 65535 {
			return nil, fmt.Errorf("container port %d is out of valid range (1-65535)", containerPort)
		}

		mappings = append(mappings, PortMapping{
			HostPort:      hostPort,
			ContainerPort: containerPort,
		})
	}

	return mappings, nil
}

// checkPortConflicts checks if any of the host ports are already in use
func checkPortConflicts(mappings []PortMapping) []int {
	var conflictPorts []int

	for _, pm := range mappings {
		// Try to listen on the host port briefly to check if it's available
		addr := fmt.Sprintf(":%d", pm.HostPort)
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			// Port is open (something is listening)
			_ = conn.Close()
			conflictPorts = append(conflictPorts, pm.HostPort)
		} else {
			// Try to bind to the port to see if it's available
			listener, err := net.Listen("tcp", addr)
			if err != nil {
				// Port is likely in use or restricted
				conflictPorts = append(conflictPorts, pm.HostPort)
			} else {
				_ = listener.Close()
			}
		}
	}

	return conflictPorts
}

// mergePortMappings merges devcontainer.json ports with CLI ports
// CLI ports take precedence on host port conflicts
func mergePortMappings(devcontainerPorts []config.PortMapping, cliPorts []PortMapping) []PortMapping {
	// Start with devcontainer.json ports as the base
	result := make([]PortMapping, 0, len(devcontainerPorts)+len(cliPorts))

	// Convert devcontainer ports to CLI PortMapping type
	for _, port := range devcontainerPorts {
		result = append(result, PortMapping{
			HostPort:      port.HostPort,
			ContainerPort: port.ContainerPort,
		})
	}

	// Add CLI ports, overriding any devcontainer ports with same host port
	for _, cliPort := range cliPorts {
		conflictIndex := -1

		// Check if CLI port conflicts with existing port by host port
		for i, existingPort := range result {
			if existingPort.HostPort == cliPort.HostPort {
				conflictIndex = i
				break
			}
		}

		if conflictIndex >= 0 {
			// Override the existing port mapping
			result[conflictIndex] = cliPort
		} else {
			// Add the new port mapping
			result = append(result, cliPort)
		}
	}

	return result
}

// createBuildSpecFromConfig creates a BuildSpec from ResolvedConfig
func createBuildSpecFromConfig(resolved *config.ResolvedConfig) (docker.BuildSpec, error) {
	if resolved.Build == nil {
		return docker.BuildSpec{}, fmt.Errorf("build configuration is nil")
	}

	// Find the devcontainer.json file to determine context base directory
	configPath, found, err := config.FindDevContainerFile(resolved.ProjectRoot)
	if err != nil {
		return docker.BuildSpec{}, fmt.Errorf("failed to find devcontainer.json: %w", err)
	}
	if !found {
		return docker.BuildSpec{}, fmt.Errorf("devcontainer.json not found")
	}

	// Get directory containing devcontainer.json
	configDir := filepath.Dir(configPath)

	// Resolve build context relative to devcontainer.json directory
	var contextPath string
	if resolved.Build.Context != "" {
		if filepath.IsAbs(resolved.Build.Context) {
			contextPath = resolved.Build.Context
		} else {
			contextPath = filepath.Join(configDir, resolved.Build.Context)
		}
	} else {
		// Default context to same directory as devcontainer.json
		contextPath = configDir
	}

	// Clean the path
	contextPath = filepath.Clean(contextPath)

	// Dockerfile defaults to "Dockerfile" if not specified
	dockerfile := resolved.Build.Dockerfile
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}

	// Create image name using project hash
	imageName := fmt.Sprintf("reactor-build:%s", resolved.ProjectHash)

	return docker.BuildSpec{
		Dockerfile: dockerfile,
		Context:    contextPath,
		ImageName:  imageName,
	}, nil
}

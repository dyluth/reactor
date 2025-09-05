package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dyluth/reactor/pkg/config"
	"github.com/dyluth/reactor/pkg/docker"
)

// PortMapping represents a port forwarding configuration
type PortMapping struct {
	HostPort      int
	ContainerPort int
}

// ContainerBlueprint defines the complete specification for creating a container
type ContainerBlueprint struct {
	Name         string        // Deterministic container name with isolation support
	Image        string        // Resolved container image
	Command      []string      // Command to run in container
	WorkDir      string        // Working directory in container
	User         string        // Container user (e.g., "claude")
	Environment  []string      // Environment variables
	Mounts       []string      // Volume mounts in "source:target:type" format
	PortMappings []PortMapping // Port forwarding configurations
	NetworkMode  string        // Network configuration
}

// NewContainerBlueprint creates a container blueprint from resolved configuration
func NewContainerBlueprint(resolved *config.ResolvedConfig, isDiscovery bool, dockerHostIntegration bool, portMappings []PortMapping) *ContainerBlueprint {
	// Generate appropriate container name based on mode
	var containerName string
	if isDiscovery {
		containerName = GenerateDiscoveryContainerName(resolved.Account, resolved.ProjectRoot, resolved.ProjectHash)
	} else {
		containerName = GenerateContainerName(resolved.Account, resolved.ProjectRoot, resolved.ProjectHash)
	}

	// Construct all mounts internally (empty for discovery mode)
	dockerMounts := []string{}
	if !isDiscovery {
		// 1. Add workspace mount first
		dockerMounts = append(dockerMounts, formatDockerMount(resolved.ProjectRoot, "/workspace"))

		// 2. Add provider credential mounts for ALL providers
		for _, provider := range config.BuiltinProviders {
			for _, mount := range provider.Mounts {
				hostPath := filepath.Join(resolved.ProjectConfigDir, mount.Source)
				dockerMounts = append(dockerMounts, formatDockerMount(hostPath, mount.Target))
			}
		}
	}

	// Add Docker socket mount if host integration is enabled
	if dockerHostIntegration {
		dockerMounts = append(dockerMounts, formatDockerMount("/var/run/docker.sock", "/var/run/docker.sock"))
	}

	// Set up environment variables
	environment := []string{}
	if dockerHostIntegration {
		environment = append(environment, "REACTOR_DOCKER_HOST_INTEGRATION=true")
	}

	// Determine container user: use RemoteUser from devcontainer.json or default to "claude"
	user := resolved.RemoteUser
	if user == "" {
		user = "claude" // Default fallback for backward compatibility
	}

	// Determine container command: use DefaultCommand from reactor customizations or default to sh
	command := []string{"/bin/sh"} // Default interactive shell (more universal than bash)
	if resolved.DefaultCommand != "" {
		// For defaultCommand, wrap it in a shell to handle complex commands
		command = []string{"/bin/sh", "-c", resolved.DefaultCommand}
	}

	return &ContainerBlueprint{
		Name:         containerName,
		Image:        resolved.Image,
		Command:      command,
		WorkDir:      "/workspace", // Default to mounted project directory
		User:         user,         // Use remoteUser from devcontainer.json with fallback
		Environment:  environment,
		Mounts:       dockerMounts,
		PortMappings: portMappings,
		NetworkMode:  "bridge", // Default Docker network
	}
}

// ToContainerSpec converts the blueprint to a Docker ContainerSpec
func (b *ContainerBlueprint) ToContainerSpec() *docker.ContainerSpec {
	// Convert port mappings to docker format
	dockerPortMappings := make([]docker.PortMapping, len(b.PortMappings))
	for i, pm := range b.PortMappings {
		dockerPortMappings[i] = docker.PortMapping{
			HostPort:      pm.HostPort,
			ContainerPort: pm.ContainerPort,
		}
	}

	return &docker.ContainerSpec{
		Name:         b.Name,
		Image:        b.Image,
		Command:      b.Command,
		WorkDir:      b.WorkDir,
		User:         b.User,
		Environment:  b.Environment,
		Mounts:       b.Mounts,
		PortMappings: dockerPortMappings,
		NetworkMode:  b.NetworkMode,
	}
}

// GenerateContainerName creates a deterministic container name with project folder name and optional isolation prefix
func GenerateContainerName(account, projectPath, projectHash string) string {
	// Extract and sanitize project folder name
	folderName := filepath.Base(projectPath)
	safeFolderName := sanitizeContainerName(folderName)

	baseName := fmt.Sprintf("reactor-%s-%s-%s", account, safeFolderName, projectHash)
	if prefix := os.Getenv("REACTOR_ISOLATION_PREFIX"); prefix != "" {
		return fmt.Sprintf("%s-%s", prefix, baseName)
	}
	return baseName
}

// GenerateDiscoveryContainerName creates a unique container name for discovery mode
func GenerateDiscoveryContainerName(account, projectPath, projectHash string) string {
	// Extract and sanitize project folder name
	folderName := filepath.Base(projectPath)
	safeFolderName := sanitizeContainerName(folderName)

	baseName := fmt.Sprintf("reactor-discovery-%s-%s-%s", account, safeFolderName, projectHash)
	if prefix := os.Getenv("REACTOR_ISOLATION_PREFIX"); prefix != "" {
		return fmt.Sprintf("%s-%s", prefix, baseName)
	}
	return baseName
}

// sanitizeContainerName ensures the folder name is safe for use in container names
func sanitizeContainerName(name string) string {
	// Docker container names must match: [a-zA-Z0-9][a-zA-Z0-9_.-]*
	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	sanitized := reg.ReplaceAllString(name, "-")

	// Ensure it starts with alphanumeric
	if len(sanitized) > 0 && !regexp.MustCompile(`^[a-zA-Z0-9]`).MatchString(sanitized) {
		sanitized = "project-" + sanitized
	}

	// Limit length to prevent overly long container names (keep reasonable length)
	const maxFolderNameLength = 20
	if len(sanitized) > maxFolderNameLength {
		sanitized = sanitized[:maxFolderNameLength]
		// Ensure it doesn't end with a hyphen after truncation
		sanitized = strings.TrimRight(sanitized, "-")
	}

	// Fallback if somehow empty
	if sanitized == "" {
		sanitized = "project"
	}

	return sanitized
}

// formatDockerMount creates a properly formatted Docker bind mount string
// that handles paths with spaces and special characters
func formatDockerMount(hostPath, containerPath string) string {
	// Quote paths that contain spaces or other special characters
	// Docker mount parsing handles quoted paths correctly
	if needsQuoting(hostPath) || needsQuoting(containerPath) {
		return fmt.Sprintf(`"%s:%s"`, hostPath, containerPath)
	}
	return fmt.Sprintf("%s:%s", hostPath, containerPath)
}

// needsQuoting checks if a path contains characters that require quoting
func needsQuoting(path string) bool {
	// Check for spaces and other characters that can cause parsing issues
	return strings.ContainsAny(path, " \t\n\r\"'\\")
}

package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/anthropics/reactor/pkg/config"
	"github.com/anthropics/reactor/pkg/docker"
)

// ContainerBlueprint defines the complete specification for creating a container
type ContainerBlueprint struct {
	Name        string   // Deterministic container name with isolation support
	Image       string   // Resolved container image
	Command     []string // Command to run in container
	WorkDir     string   // Working directory in container
	User        string   // Container user (e.g., "claude")
	Environment []string // Environment variables
	Mounts      []string // Volume mounts in "source:target:type" format
	NetworkMode string   // Network configuration
}

// NewContainerBlueprint creates a container blueprint from resolved configuration and mounts
func NewContainerBlueprint(resolved *config.ResolvedConfig, mounts []MountSpec, isDiscovery bool) *ContainerBlueprint {
	// Generate appropriate container name based on mode
	var containerName string
	if isDiscovery {
		containerName = GenerateDiscoveryContainerName(resolved.Account, resolved.ProjectRoot, resolved.ProjectHash)
	} else {
		containerName = GenerateContainerName(resolved.Account, resolved.ProjectRoot, resolved.ProjectHash)
	}

	// Convert mount specifications to Docker bind format (empty for discovery mode)
	dockerMounts := []string{}
	if !isDiscovery {
		for _, mount := range mounts {
			// Format: "source:target:type" (e.g., "/home/user/.reactor/cam/abc123/claude:/home/claude/.claude:bind")
			dockerMounts = append(dockerMounts, fmt.Sprintf("%s:%s:%s", mount.Source, mount.Target, mount.Type))
		}
	}

	return &ContainerBlueprint{
		Name:        containerName,
		Image:       resolved.Image,
		Command:     []string{"/bin/bash"}, // Default interactive shell
		WorkDir:     "/workspace",          // Default to mounted project directory
		User:        "claude",              // Default container user
		Environment: []string{},            // No special environment variables by default
		Mounts:      dockerMounts,
		NetworkMode: "bridge", // Default Docker network
	}
}

// ToContainerSpec converts the blueprint to a Docker ContainerSpec
func (b *ContainerBlueprint) ToContainerSpec() *docker.ContainerSpec {
	return &docker.ContainerSpec{
		Name:        b.Name,
		Image:       b.Image,
		Command:     b.Command,
		WorkDir:     b.WorkDir,
		User:        b.User,
		Environment: b.Environment,
		Mounts:      b.Mounts,
		NetworkMode: b.NetworkMode,
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

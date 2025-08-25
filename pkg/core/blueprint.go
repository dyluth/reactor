package core

import (
	"fmt"
	"os"

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
func NewContainerBlueprint(resolved *config.ResolvedConfig, mounts []MountSpec) *ContainerBlueprint {
	// Generate deterministic container name with isolation prefix support
	containerName := GenerateContainerName(resolved.Account, resolved.ProjectHash)

	// Convert mount specifications to Docker bind format
	dockerMounts := []string{}
	for _, mount := range mounts {
		// Format: "source:target:type" (e.g., "/home/user/.reactor/cam/abc123/claude:/home/claude/.claude:bind")
		dockerMounts = append(dockerMounts, fmt.Sprintf("%s:%s:%s", mount.Source, mount.Target, mount.Type))
	}

	return &ContainerBlueprint{
		Name:        containerName,
		Image:       resolved.Image,
		Command:     []string{"/bin/bash"}, // Default interactive shell
		WorkDir:     "/workspace",          // Default to mounted project directory
		User:        "claude",              // Default container user
		Environment: []string{},            // No special environment variables by default
		Mounts:      dockerMounts,
		NetworkMode: "bridge",              // Default Docker network
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

// GenerateContainerName creates a deterministic container name with optional isolation prefix
func GenerateContainerName(account, projectHash string) string {
	baseName := fmt.Sprintf("reactor-%s-%s", account, projectHash)
	if prefix := os.Getenv("REACTOR_ISOLATION_PREFIX"); prefix != "" {
		return fmt.Sprintf("%s-%s", prefix, baseName)
	}
	return baseName
}
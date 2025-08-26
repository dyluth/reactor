package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Service manages Docker daemon interactions
type Service struct {
	client *client.Client
}

// NewService creates a new Docker service
func NewService() (*Service, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Service{
		client: cli,
	}, nil
}

// Close closes the Docker client connection
func (s *Service) Close() error {
	return s.client.Close()
}

// CheckHealth verifies Docker daemon is accessible and running
func (s *Service) CheckHealth(ctx context.Context) error {
	// Set timeout to prevent hanging
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Ping Docker daemon
	ping, err := s.client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("docker daemon is not accessible. Please ensure Docker is running and you have proper permissions: %w", err)
	}

	// Verify we can communicate with daemon
	if ping.APIVersion == "" {
		return fmt.Errorf("docker daemon responded but API version is unknown. Please check your Docker installation")
	}

	return nil
}

// ContainerExists checks if a container with the given name exists
func (s *Service) ContainerExists(ctx context.Context, name string) (ContainerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	containers, err := s.client.ContainerList(ctx, container.ListOptions{
		All: true, // Include stopped containers
	})
	if err != nil {
		return ContainerInfo{}, fmt.Errorf("failed to list containers: %w", err)
	}

	for _, container := range containers {
		for _, containerName := range container.Names {
			// Container names have leading slash, so check with and without
			if containerName == "/"+name || containerName == name {
				var status ContainerStatus
				switch container.State {
				case "running":
					status = StatusRunning
				case "exited", "stopped":
					status = StatusStopped
				default:
					status = StatusNotFound
				}

				return ContainerInfo{
					ID:     container.ID,
					Name:   name,
					Status: status,
					Image:  container.Image,
				}, nil
			}
		}
	}

	return ContainerInfo{
		Status: StatusNotFound,
	}, nil
}

// ContainerInfo holds information about a container
type ContainerInfo struct {
	ID     string
	Name   string
	Status ContainerStatus
	Image  string
}

// ContainerStatus represents the status of a container
type ContainerStatus string

const (
	StatusRunning  ContainerStatus = "running"
	StatusStopped  ContainerStatus = "stopped"
	StatusNotFound ContainerStatus = "not_found"
)

// CreateContainer creates a new container with the given specifications
func (s *Service) CreateContainer(ctx context.Context, spec *ContainerSpec) (ContainerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Create container configuration
	containerConfig := &container.Config{
		Image:      spec.Image,
		Cmd:        spec.Command,
		WorkingDir: spec.WorkDir,
		User:       spec.User,
		Env:        spec.Environment,
	}

	// Create host configuration (mounts, network, etc.)
	hostConfig := &container.HostConfig{
		Binds:       spec.Mounts,
		NetworkMode: container.NetworkMode(spec.NetworkMode),
	}

	// Create the container
	resp, err := s.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, spec.Name)
	if err != nil {
		return ContainerInfo{}, fmt.Errorf("failed to create container %s: %w", spec.Name, err)
	}

	return ContainerInfo{
		ID:     resp.ID,
		Name:   spec.Name,
		Status: StatusStopped,
		Image:  spec.Image,
	}, nil
}

// StartContainer starts a stopped container
func (s *Service) StartContainer(ctx context.Context, containerID string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := s.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}

	return nil
}

// StopContainer stops a running container
func (s *Service) StopContainer(ctx context.Context, containerID string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	timeout := 10 // Give container 10 seconds to stop gracefully
	if err := s.client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	}); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}

	return nil
}

// RemoveContainer removes a container (must be stopped first)
func (s *Service) RemoveContainer(ctx context.Context, containerID string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := s.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: true, // Force removal even if running
	}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}

	return nil
}

// ContainerSpec defines the specification for creating a container
type ContainerSpec struct {
	Name        string
	Image       string
	Command     []string
	WorkDir     string
	User        string
	Environment []string
	Mounts      []string // In "source:target:mode" format
	NetworkMode string
}

// ListReactorContainers returns all containers that match the reactor naming pattern
func (s *Service) ListReactorContainers(ctx context.Context) ([]ContainerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	containers, err := s.client.ContainerList(ctx, container.ListOptions{
		All: true, // Include stopped containers
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var reactorContainers []ContainerInfo
	for _, c := range containers {
		for _, containerName := range c.Names {
			// Container names have leading slash, so remove it
			name := strings.TrimPrefix(containerName, "/")
			
			// Check if this is a reactor container (with or without isolation prefix)
			if s.isReactorContainer(name) {
				var status ContainerStatus
				switch c.State {
				case "running":
					status = StatusRunning
				case "exited", "stopped":
					status = StatusStopped
				default:
					status = StatusNotFound
				}

				reactorContainers = append(reactorContainers, ContainerInfo{
					ID:     c.ID,
					Name:   name,
					Status: status,
					Image:  c.Image,
				})
				break // Found matching name, no need to check other names for this container
			}
		}
	}

	return reactorContainers, nil
}

// FindProjectContainer finds a container for a specific project path
func (s *Service) FindProjectContainer(ctx context.Context, account, projectPath, projectHash string) (*ContainerInfo, error) {
	// Generate expected container name for this project
	expectedName := s.generateContainerNameForProject(account, projectPath, projectHash)
	
	// Use existing ContainerExists method
	containerInfo, err := s.ContainerExists(ctx, expectedName)
	if err != nil {
		return nil, err
	}
	
	if containerInfo.Status == StatusNotFound {
		return nil, nil // No container found, but no error
	}
	
	return &containerInfo, nil
}

// isReactorContainer checks if a container name matches reactor naming pattern
func (s *Service) isReactorContainer(name string) bool {
	// Match patterns:
	// reactor-{account}-{folder}-{hash}
	// {prefix}-reactor-{account}-{folder}-{hash} (with isolation prefix)
	
	// Check for isolation prefix pattern first
	if isolationPrefix := os.Getenv("REACTOR_ISOLATION_PREFIX"); isolationPrefix != "" {
		expectedPrefix := isolationPrefix + "-reactor-"
		if strings.HasPrefix(name, expectedPrefix) {
			return true
		}
	}
	
	// Check for standard reactor pattern
	if strings.HasPrefix(name, "reactor-") {
		// Verify it has the expected number of components
		// reactor-{account}-{folder}-{hash} = 4 parts minimum
		parts := strings.Split(name, "-")
		return len(parts) >= 4 && parts[0] == "reactor"
	}
	
	return false
}

// generateContainerNameForProject creates the expected container name for a project
func (s *Service) generateContainerNameForProject(account, projectPath, projectHash string) string {
	// This should match the logic in pkg/core/blueprint.go
	folderName := filepath.Base(projectPath)
	safeFolderName := s.sanitizeContainerName(folderName)
	
	baseName := fmt.Sprintf("reactor-%s-%s-%s", account, safeFolderName, projectHash)
	if prefix := os.Getenv("REACTOR_ISOLATION_PREFIX"); prefix != "" {
		return fmt.Sprintf("%s-%s", prefix, baseName)
	}
	return baseName
}

// sanitizeContainerName mirrors the logic from pkg/core/blueprint.go
func (s *Service) sanitizeContainerName(name string) string {
	// Docker container names must match: [a-zA-Z0-9][a-zA-Z0-9_.-]*
	reg := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	sanitized := reg.ReplaceAllString(name, "-")
	
	// Ensure it starts with alphanumeric
	if len(sanitized) > 0 && !regexp.MustCompile(`^[a-zA-Z0-9]`).MatchString(sanitized) {
		sanitized = "project-" + sanitized
	}
	
	// Limit length
	const maxFolderNameLength = 20
	if len(sanitized) > maxFolderNameLength {
		sanitized = sanitized[:maxFolderNameLength]
		sanitized = strings.TrimRight(sanitized, "-")
	}
	
	if sanitized == "" {
		sanitized = "project"
	}
	
	return sanitized
}
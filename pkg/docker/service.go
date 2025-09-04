package docker

import (
	"archive/tar"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Service manages Docker daemon interactions
type Service struct {
	client DockerClient
}

// NewService creates a new Docker service with a real Docker client
func NewService() (*Service, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Service{
		client: cli,
	}, nil
}

// NewServiceWithClient creates a new Docker service with the provided client.
// This constructor is primarily used for testing with mock clients.
func NewServiceWithClient(client DockerClient) *Service {
	return &Service{
		client: client,
	}
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

	// Create port bindings for container and host configuration
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	for _, pm := range spec.PortMappings {
		hostPortStr := strconv.Itoa(pm.HostPort)

		containerPort, err := nat.NewPort("tcp", strconv.Itoa(pm.ContainerPort))
		if err != nil {
			return ContainerInfo{}, fmt.Errorf("invalid container port %d: %w", pm.ContainerPort, err)
		}

		exposedPorts[containerPort] = struct{}{}
		portBindings[containerPort] = []nat.PortBinding{
			{
				HostIP:   "",
				HostPort: hostPortStr,
			},
		}
	}

	// Create container configuration
	containerConfig := &container.Config{
		Image:        spec.Image,
		Cmd:          spec.Command,
		WorkingDir:   spec.WorkDir,
		User:         spec.User,
		Env:          spec.Environment,
		ExposedPorts: exposedPorts,
	}

	// Create host configuration (mounts, network, ports, etc.)
	hostConfig := &container.HostConfig{
		Binds:        spec.Mounts,
		NetworkMode:  container.NetworkMode(spec.NetworkMode),
		PortBindings: portBindings,
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
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if err := s.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: true, // Force removal even if running
	}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}

	return nil
}

// BuildSpec defines the specification for building a Docker image
type BuildSpec struct {
	Dockerfile string // Path to Dockerfile relative to context
	Context    string // Path to build context directory
	ImageName  string // Name to tag the built image with
}

// ContainerSpec defines the specification for creating a container
// PortMapping represents a port forwarding configuration
type PortMapping struct {
	HostPort      int
	ContainerPort int
}

type ContainerSpec struct {
	Name         string
	Image        string
	Command      []string
	WorkDir      string
	User         string
	Environment  []string
	Mounts       []string      // In "source:target:mode" format
	PortMappings []PortMapping // Port forwarding configurations
	NetworkMode  string
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

// FileChange represents a filesystem change in a container
type FileChange struct {
	Kind string // A (Added), D (Deleted), C (Changed)
	Path string // Path to the changed file
}

// ContainerDiff returns filesystem changes made to a container
func (s *Service) ContainerDiff(ctx context.Context, containerID string) ([]FileChange, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get container diff from Docker
	changes, err := s.client.ContainerDiff(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container diff: %w", err)
	}

	// Convert to our FileChange format
	var fileChanges []FileChange
	for _, change := range changes {
		var kind string
		switch change.Kind {
		case container.ChangeModify:
			kind = "C"
		case container.ChangeAdd:
			kind = "A"
		case container.ChangeDelete:
			kind = "D"
		default:
			kind = "?"
		}

		fileChanges = append(fileChanges, FileChange{
			Kind: kind,
			Path: change.Path,
		})
	}

	return fileChanges, nil
}

// ImageExists checks if an image with the given name/tag exists locally
func (s *Service) ImageExists(ctx context.Context, imageName string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	images, err := s.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to list images: %w", err)
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == imageName {
				return true, nil
			}
		}
	}

	return false, nil
}

// BuildImage builds a Docker image from the given BuildSpec
// It checks if the image already exists and skips building if found, unless forceRebuild is true
func (s *Service) BuildImage(ctx context.Context, spec BuildSpec, forceRebuild bool) error {
	// Check if image already exists (unless forcing rebuild)
	if !forceRebuild {
		exists, err := s.ImageExists(ctx, spec.ImageName)
		if err != nil {
			return fmt.Errorf("failed to check if image exists: %w", err)
		}
		if exists {
			fmt.Printf("Image %s already exists, skipping build\n", spec.ImageName)
			return nil
		}
	}

	// Validate context directory exists
	if _, err := os.Stat(spec.Context); os.IsNotExist(err) {
		return fmt.Errorf("build context directory does not exist: %s", spec.Context)
	}

	// Validate Dockerfile exists
	dockerfilePath := filepath.Join(spec.Context, spec.Dockerfile)
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("dockerfile does not exist: %s", dockerfilePath)
	}

	fmt.Printf("Building Docker image: %s\n", spec.ImageName)
	fmt.Printf("Context: %s\n", spec.Context)
	fmt.Printf("Dockerfile: %s\n", spec.Dockerfile)

	// Create build context tar archive
	buildContext, err := s.createBuildContext(spec.Context)
	if err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}
	defer func() { _ = buildContext.Close() }()

	// Build the image
	buildOptions := types.ImageBuildOptions{
		Context:    buildContext,
		Dockerfile: spec.Dockerfile,
		Tags:       []string{spec.ImageName},
		Remove:     true, // Remove intermediate containers
	}

	response, err := s.client.ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	// Stream build output to console with real-time feedback
	if err := s.streamBuildOutput(response.Body); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("Successfully built image: %s\n", spec.ImageName)
	return nil
}

// createBuildContext creates a tar archive of the build context directory
func (s *Service) createBuildContext(contextPath string) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		defer func() { _ = pw.Close() }()
		tw := tar.NewWriter(pw)
		defer func() { _ = tw.Close() }()

		err := filepath.Walk(contextPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Get relative path from context directory
			relPath, err := filepath.Rel(contextPath, path)
			if err != nil {
				return err
			}

			// Skip if this is the context directory itself
			if relPath == "." {
				return nil
			}

			// Create tar header
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}

			// Use forward slashes for tar paths (required by Docker)
			header.Name = filepath.ToSlash(relPath)

			// Write header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			// If it's a regular file, write its content
			if info.Mode().IsRegular() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer func() { _ = file.Close() }()

				if _, err := io.Copy(tw, file); err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr, nil
}

// streamBuildOutput processes Docker build output and streams it to console
func (s *Service) streamBuildOutput(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		var buildOutput struct {
			Stream string `json:"stream,omitempty"`
			Error  string `json:"error,omitempty"`
		}

		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &buildOutput); err != nil {
			// If we can't parse as JSON, just print the raw line
			fmt.Print(line + "\n")
			continue
		}

		// Handle build errors
		if buildOutput.Error != "" {
			return fmt.Errorf("build error: %s", buildOutput.Error)
		}

		// Stream build output preserving ANSI colors
		if buildOutput.Stream != "" {
			fmt.Print(buildOutput.Stream)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading build output: %w", err)
	}

	return nil
}

// ExecutePostCreateCommand runs the postCreateCommand in the specified container
// postCreateCommand can be either a string or []string (array of strings)
func (s *Service) ExecutePostCreateCommand(ctx context.Context, containerID string, postCreateCommand interface{}) error {
	if postCreateCommand == nil {
		// No postCreateCommand specified, nothing to do
		return nil
	}

	// Parse postCreateCommand into command array
	var cmdArray []string
	switch cmd := postCreateCommand.(type) {
	case string:
		if strings.TrimSpace(cmd) == "" {
			// Empty command, nothing to do
			return nil
		}
		// For string commands, we'll execute them through the shell to handle complex commands
		cmdArray = []string{"/bin/sh", "-c", cmd}
	case []interface{}:
		if len(cmd) == 0 {
			// Empty array, nothing to do
			return nil
		}
		// Convert []interface{} to []string
		for _, v := range cmd {
			if str, ok := v.(string); ok {
				cmdArray = append(cmdArray, str)
			} else {
				return fmt.Errorf("postCreateCommand array contains non-string element: %v", v)
			}
		}
	case []string:
		if len(cmd) == 0 {
			// Empty array, nothing to do
			return nil
		}
		cmdArray = cmd
	default:
		return fmt.Errorf("postCreateCommand must be a string or array of strings, got %T", postCreateCommand)
	}

	// Check if container is running
	containerInfo, err := s.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	if !containerInfo.State.Running {
		return fmt.Errorf("container %s is not running, cannot execute postCreateCommand", containerID)
	}

	fmt.Printf("Executing postCreateCommand: %v\n", cmdArray)

	// Create exec instance for postCreateCommand
	execConfig := types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmdArray,
	}

	execResp, err := s.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance for postCreateCommand: %w", err)
	}

	// Start the exec instance
	if err := s.client.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{}); err != nil {
		return fmt.Errorf("failed to start postCreateCommand execution: %w", err)
	}

	// Attach to get output
	attachResp, err := s.client.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	if err != nil {
		return fmt.Errorf("failed to attach to postCreateCommand execution: %w", err)
	}
	defer attachResp.Close()

	// Stream the output
	scanner := bufio.NewScanner(attachResp.Reader)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading postCreateCommand output: %w", err)
	}

	// Wait for the exec to complete and check exit code
	inspectResp, err := s.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect postCreateCommand execution: %w", err)
	}

	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("postCreateCommand failed with exit code %d", inspectResp.ExitCode)
	}

	fmt.Println("postCreateCommand completed successfully")
	return nil
}

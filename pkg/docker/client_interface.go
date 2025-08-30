package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// DockerClient defines the interface for Docker API operations needed by our Service.
// This interface allows us to mock the Docker client in unit tests while
// using the real Docker client in production.
//
// The interface is focused on the critical path operations needed for container recovery.
type DockerClient interface {
	// Health and connection management
	Ping(ctx context.Context) (types.Ping, error)
	Close() error
	
	// Core container lifecycle operations - CRITICAL PATH
	ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error)
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
	
	// Session and interaction operations
	ContainerAttach(ctx context.Context, containerID string, options container.AttachOptions) (types.HijackedResponse, error)
	ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	
	// Exec operations for session management
	ContainerExecCreate(ctx context.Context, containerID string, options types.ExecConfig) (types.IDResponse, error)
	ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error)
	ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error
	
	// Additional operations for discovery and debugging
	ContainerDiff(ctx context.Context, containerID string) ([]container.FilesystemChange, error)
	ContainerKill(ctx context.Context, containerID string, signal string) error
	ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error
	ContainerResize(ctx context.Context, containerID string, options container.ResizeOptions) error
	
	// Image management
	ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error)
}

// Ensure that *client.Client implements our DockerClient interface at compile time
var _ DockerClient = (*client.Client)(nil)
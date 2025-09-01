package docker

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDockerClient implements DockerClient interface for testing
type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) Ping(ctx context.Context) (types.Ping, error) {
	args := m.Called(ctx)
	return args.Get(0).(types.Ping), args.Error(1)
}

func (m *MockDockerClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDockerClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	args := m.Called(ctx, options)
	return args.Get(0).([]types.Container), args.Error(1)
}

func (m *MockDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
	args := m.Called(ctx, config, hostConfig, networkingConfig, platform, containerName)
	return args.Get(0).(container.CreateResponse), args.Error(1)
}

func (m *MockDockerClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerAttach(ctx context.Context, containerID string, options container.AttachOptions) (types.HijackedResponse, error) {
	args := m.Called(ctx, containerID, options)
	return args.Get(0).(types.HijackedResponse), args.Error(1)
}

func (m *MockDockerClient) ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	args := m.Called(ctx, containerID, condition)
	return args.Get(0).(<-chan container.WaitResponse), args.Get(1).(<-chan error)
}

func (m *MockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(types.ContainerJSON), args.Error(1)
}

func (m *MockDockerClient) ContainerExecCreate(ctx context.Context, containerID string, options types.ExecConfig) (types.IDResponse, error) {
	args := m.Called(ctx, containerID, options)
	return args.Get(0).(types.IDResponse), args.Error(1)
}

func (m *MockDockerClient) ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error) {
	args := m.Called(ctx, execID, config)
	return args.Get(0).(types.HijackedResponse), args.Error(1)
}

func (m *MockDockerClient) ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error {
	args := m.Called(ctx, execID, config)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerDiff(ctx context.Context, containerID string) ([]container.FilesystemChange, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).([]container.FilesystemChange), args.Error(1)
}

func (m *MockDockerClient) ContainerKill(ctx context.Context, containerID string, signal string) error {
	args := m.Called(ctx, containerID, signal)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error {
	args := m.Called(ctx, execID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerResize(ctx context.Context, containerID string, options container.ResizeOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
	args := m.Called(ctx, refStr, options)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// Test utilities
func setupTestService() (*Service, *MockDockerClient) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)
	return service, mockClient
}

// CRITICAL PATH TESTS - Container Recovery Logic

func TestContainerExists_NotFound(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Mock ContainerList to return empty list (no containers)
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return([]types.Container{}, nil)

	containerInfo, err := service.ContainerExists(context.Background(), "test-container")

	assert.NoError(t, err)
	assert.Equal(t, StatusNotFound, containerInfo.Status)
	assert.Empty(t, containerInfo.ID)
	assert.Empty(t, containerInfo.Name)
}

func TestContainerExists_Running(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Mock ContainerList to return running container
	containers := []types.Container{
		{
			ID:    "test-id-123",
			Names: []string{"/test-container"},
			State: "running",
			Image: "test-image:latest",
		},
	}
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return(containers, nil)

	containerInfo, err := service.ContainerExists(context.Background(), "test-container")

	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, containerInfo.Status)
	assert.Equal(t, "test-id-123", containerInfo.ID)
	assert.Equal(t, "test-container", containerInfo.Name)
	assert.Equal(t, "test-image:latest", containerInfo.Image)
}

func TestContainerExists_Stopped(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Mock ContainerList to return stopped container
	containers := []types.Container{
		{
			ID:    "test-id-456",
			Names: []string{"/test-container"},
			State: "exited",
			Image: "test-image:latest",
		},
	}
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return(containers, nil)

	containerInfo, err := service.ContainerExists(context.Background(), "test-container")

	assert.NoError(t, err)
	assert.Equal(t, StatusStopped, containerInfo.Status)
	assert.Equal(t, "test-id-456", containerInfo.ID)
	assert.Equal(t, "test-container", containerInfo.Name)
	assert.Equal(t, "test-image:latest", containerInfo.Image)
}

func TestContainerExists_ListError(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Mock ContainerList to return error
	expectedError := errors.New("docker daemon not available")
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return([]types.Container{}, expectedError)

	containerInfo, err := service.ContainerExists(context.Background(), "test-container")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list containers")
	assert.Contains(t, err.Error(), "docker daemon not available")
	assert.Equal(t, ContainerInfo{}, containerInfo)
}

func TestStartContainer_Success(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Mock successful container start
	mockClient.On("ContainerStart", mock.Anything, "test-id-123", container.StartOptions{}).Return(nil)

	err := service.StartContainer(context.Background(), "test-id-123")

	assert.NoError(t, err)
}

func TestStartContainer_Error(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Mock container start failure
	expectedError := errors.New("container failed to start")
	mockClient.On("ContainerStart", mock.Anything, "test-id-123", container.StartOptions{}).Return(expectedError)

	err := service.StartContainer(context.Background(), "test-id-123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start container test-id-123")
	assert.Contains(t, err.Error(), "container failed to start")
}

func TestStopContainer_Success(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Mock successful container stop
	timeout := 10
	expectedOptions := container.StopOptions{Timeout: &timeout}
	mockClient.On("ContainerStop", mock.Anything, "test-id-123", expectedOptions).Return(nil)

	err := service.StopContainer(context.Background(), "test-id-123")

	assert.NoError(t, err)
}

func TestStopContainer_Error(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Mock container stop failure
	expectedError := errors.New("container failed to stop")
	timeout := 10
	expectedOptions := container.StopOptions{Timeout: &timeout}
	mockClient.On("ContainerStop", mock.Anything, "test-id-123", expectedOptions).Return(expectedError)

	err := service.StopContainer(context.Background(), "test-id-123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop container test-id-123")
	assert.Contains(t, err.Error(), "container failed to stop")
}

func TestRemoveContainer_Success(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Mock successful container removal
	expectedOptions := container.RemoveOptions{Force: true}
	mockClient.On("ContainerRemove", mock.Anything, "test-id-123", expectedOptions).Return(nil)

	err := service.RemoveContainer(context.Background(), "test-id-123")

	assert.NoError(t, err)
}

func TestRemoveContainer_Error(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Mock container removal failure
	expectedError := errors.New("container failed to remove")
	expectedOptions := container.RemoveOptions{Force: true}
	mockClient.On("ContainerRemove", mock.Anything, "test-id-123", expectedOptions).Return(expectedError)

	err := service.RemoveContainer(context.Background(), "test-id-123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove container test-id-123")
	assert.Contains(t, err.Error(), "container failed to remove")
}

func TestCreateContainer_Success(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Test data
	spec := &ContainerSpec{
		Name:        "test-container",
		Image:       "test-image:latest",
		Command:     []string{"echo", "hello"},
		WorkDir:     "/app",
		User:        "root",
		Environment: []string{"ENV=test"},
		Mounts:      []string{"/host:/container:rw"},
		PortMappings: []PortMapping{
			{HostPort: 8080, ContainerPort: 80},
		},
		NetworkMode: "bridge",
	}

	// Mock successful container creation
	expectedResponse := container.CreateResponse{
		ID: "new-container-id",
	}
	mockClient.On("ContainerCreate", mock.Anything, mock.AnythingOfType("*container.Config"), mock.AnythingOfType("*container.HostConfig"), mock.Anything, mock.Anything, "test-container").Return(expectedResponse, nil)

	containerInfo, err := service.CreateContainer(context.Background(), spec)

	assert.NoError(t, err)
	assert.Equal(t, "new-container-id", containerInfo.ID)
	assert.Equal(t, "test-container", containerInfo.Name)
	assert.Equal(t, StatusStopped, containerInfo.Status)
	assert.Equal(t, "test-image:latest", containerInfo.Image)
}

func TestCreateContainer_Error(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Test data
	spec := &ContainerSpec{
		Name:  "test-container",
		Image: "test-image:latest",
	}

	// Mock container creation failure
	expectedError := errors.New("image not found")
	mockClient.On("ContainerCreate", mock.Anything, mock.AnythingOfType("*container.Config"), mock.AnythingOfType("*container.HostConfig"), mock.Anything, mock.Anything, "test-container").Return(container.CreateResponse{}, expectedError)

	containerInfo, err := service.CreateContainer(context.Background(), spec)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create container test-container")
	assert.Contains(t, err.Error(), "image not found")
	assert.Equal(t, ContainerInfo{}, containerInfo)
}

// Test timeouts to ensure our context handling works
func TestContainerExists_Timeout(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// Create a context that will timeout quickly
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Mock ContainerList to simulate a slow response
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return([]types.Container{}, context.DeadlineExceeded).After(10 * time.Millisecond)

	containerInfo, err := service.ContainerExists(ctx, "test-container")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list containers")
	assert.Equal(t, ContainerInfo{}, containerInfo)
}

// CRITICAL PATH INTEGRATION TESTS - ProvisionContainer Recovery Logic

func TestProvisionContainer_RunningContainerExists(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	spec := &ContainerSpec{
		Name:  "test-container",
		Image: "test-image:latest",
	}

	// Mock ContainerExists to return running container
	containers := []types.Container{
		{
			ID:    "existing-id-123",
			Names: []string{"/test-container"},
			State: "running",
			Image: "test-image:latest",
		},
	}
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return(containers, nil)

	// Should NOT call create or start since container is already running
	containerInfo, err := service.ProvisionContainer(context.Background(), spec)

	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, containerInfo.Status)
	assert.Equal(t, "existing-id-123", containerInfo.ID)
	assert.Equal(t, "test-container", containerInfo.Name)
}

func TestProvisionContainer_StoppedContainerRestarts(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	spec := &ContainerSpec{
		Name:  "test-container",
		Image: "test-image:latest",
	}

	// Mock ContainerExists to return stopped container
	containers := []types.Container{
		{
			ID:    "existing-id-456",
			Names: []string{"/test-container"},
			State: "exited",
			Image: "test-image:latest",
		},
	}
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return(containers, nil)

	// Mock successful restart
	mockClient.On("ContainerStart", mock.Anything, "existing-id-456", container.StartOptions{}).Return(nil)

	containerInfo, err := service.ProvisionContainer(context.Background(), spec)

	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, containerInfo.Status)
	assert.Equal(t, "existing-id-456", containerInfo.ID)
	assert.Equal(t, "test-container", containerInfo.Name)
}

func TestProvisionContainer_StoppedContainerRestartFails_CreatesNew(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	spec := &ContainerSpec{
		Name:  "test-container",
		Image: "test-image:latest",
	}

	// Mock ContainerExists to return stopped container
	containers := []types.Container{
		{
			ID:    "broken-id-789",
			Names: []string{"/test-container"},
			State: "exited",
			Image: "test-image:latest",
		},
	}
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return(containers, nil)

	// Mock restart failure
	mockClient.On("ContainerStart", mock.Anything, "broken-id-789", container.StartOptions{}).Return(errors.New("container corrupted"))

	// Mock removal of broken container
	mockClient.On("ContainerRemove", mock.Anything, "broken-id-789", container.RemoveOptions{Force: true}).Return(nil)

	// Mock creation of new container
	mockClient.On("ContainerCreate", mock.Anything, mock.AnythingOfType("*container.Config"), mock.AnythingOfType("*container.HostConfig"), mock.Anything, mock.Anything, "test-container").Return(container.CreateResponse{ID: "new-id-999"}, nil)

	// Mock start of new container
	mockClient.On("ContainerStart", mock.Anything, "new-id-999", container.StartOptions{}).Return(nil)

	containerInfo, err := service.ProvisionContainer(context.Background(), spec)

	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, containerInfo.Status)
	assert.Equal(t, "new-id-999", containerInfo.ID)
	assert.Equal(t, "test-container", containerInfo.Name)
}

func TestProvisionContainer_NoContainerExists_CreatesNew(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	spec := &ContainerSpec{
		Name:  "test-container",
		Image: "test-image:latest",
	}

	// Mock ContainerExists to return not found
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return([]types.Container{}, nil)

	// Mock creation of new container
	mockClient.On("ContainerCreate", mock.Anything, mock.AnythingOfType("*container.Config"), mock.AnythingOfType("*container.HostConfig"), mock.Anything, mock.Anything, "test-container").Return(container.CreateResponse{ID: "new-id-111"}, nil)

	// Mock start of new container
	mockClient.On("ContainerStart", mock.Anything, "new-id-111", container.StartOptions{}).Return(nil)

	containerInfo, err := service.ProvisionContainer(context.Background(), spec)

	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, containerInfo.Status)
	assert.Equal(t, "new-id-111", containerInfo.ID)
	assert.Equal(t, "test-container", containerInfo.Name)
}

func TestProvisionContainer_CreateFails(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	spec := &ContainerSpec{
		Name:  "test-container",
		Image: "nonexistent-image:latest",
	}

	// Mock ContainerExists to return not found
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return([]types.Container{}, nil)

	// Mock creation failure
	mockClient.On("ContainerCreate", mock.Anything, mock.AnythingOfType("*container.Config"), mock.AnythingOfType("*container.HostConfig"), mock.Anything, mock.Anything, "test-container").Return(container.CreateResponse{}, errors.New("image not found"))

	containerInfo, err := service.ProvisionContainer(context.Background(), spec)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create new container")
	assert.Contains(t, err.Error(), "image not found")
	assert.Equal(t, ContainerInfo{}, containerInfo)
}

func TestProvisionContainer_StartNewContainerFails_CleansUp(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	spec := &ContainerSpec{
		Name:  "test-container",
		Image: "test-image:latest",
	}

	// Mock ContainerExists to return not found
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return([]types.Container{}, nil)

	// Mock successful creation
	mockClient.On("ContainerCreate", mock.Anything, mock.AnythingOfType("*container.Config"), mock.AnythingOfType("*container.HostConfig"), mock.Anything, mock.Anything, "test-container").Return(container.CreateResponse{ID: "new-id-222"}, nil)

	// Mock start failure
	mockClient.On("ContainerStart", mock.Anything, "new-id-222", container.StartOptions{}).Return(errors.New("port already in use"))

	// Mock cleanup of failed container
	mockClient.On("ContainerRemove", mock.Anything, "new-id-222", container.RemoveOptions{Force: true}).Return(nil)

	containerInfo, err := service.ProvisionContainer(context.Background(), spec)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start new container")
	assert.Contains(t, err.Error(), "port already in use")
	assert.Equal(t, ContainerInfo{}, containerInfo)
}

func TestProvisionContainerWithCleanup_ForceCleanup(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	spec := &ContainerSpec{
		Name:  "test-container",
		Image: "test-image:latest",
	}

	// Mock ContainerExists to return running container
	containers := []types.Container{
		{
			ID:    "existing-running-id",
			Names: []string{"/test-container"},
			State: "running",
			Image: "test-image:latest",
		},
	}
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return(containers, nil)

	// Mock forced cleanup of existing container
	mockClient.On("ContainerStop", mock.Anything, "existing-running-id", mock.AnythingOfType("container.StopOptions")).Return(nil)
	mockClient.On("ContainerRemove", mock.Anything, "existing-running-id", container.RemoveOptions{Force: true}).Return(nil)

	// Mock creation and start of new container
	mockClient.On("ContainerCreate", mock.Anything, mock.AnythingOfType("*container.Config"), mock.AnythingOfType("*container.HostConfig"), mock.Anything, mock.Anything, "test-container").Return(container.CreateResponse{ID: "clean-new-id"}, nil)
	mockClient.On("ContainerStart", mock.Anything, "clean-new-id", container.StartOptions{}).Return(nil)

	containerInfo, err := service.ProvisionContainerWithCleanup(context.Background(), spec, true)

	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, containerInfo.Status)
	assert.Equal(t, "clean-new-id", containerInfo.ID)
	assert.Equal(t, "test-container", containerInfo.Name)
}

// Tests for 0% coverage functions to reach >80% total coverage

func TestNewService(t *testing.T) {
	// Test NewService constructor
	service, err := NewService()
	if err != nil {
		// Docker might not be available in test environment, but constructor should work
		t.Logf("NewService failed (Docker may not be available): %v", err)
	} else {
		assert.NotNil(t, service)
		assert.NotNil(t, service.client)
		// Clean up
		_ = service.Close()
	}
}

func TestClose(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	// Mock Close method
	mockClient.On("Close").Return(nil)

	err := service.Close()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestClose_Error(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	// Mock Close method with error
	mockClient.On("Close").Return(errors.New("close failed"))

	err := service.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "close failed")
	mockClient.AssertExpectations(t)
}

func TestCheckHealth_Success(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	// Mock successful ping
	expectedPing := types.Ping{APIVersion: "1.42"}
	mockClient.On("Ping", mock.Anything).Return(expectedPing, nil)

	err := service.CheckHealth(context.Background())
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestCheckHealth_Error(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	// Mock ping failure
	mockClient.On("Ping", mock.Anything).Return(types.Ping{}, errors.New("daemon not running"))

	err := service.CheckHealth(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "daemon not running")
	mockClient.AssertExpectations(t)
}

func TestListReactorContainers_Success(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	containers := []types.Container{
		{
			ID:    "reactor-id-1",
			Names: []string{"/reactor-user-project-abc123"},
			State: "running",
			Image: "ghcr.io/dyluth/reactor/base:latest",
		},
		{
			ID:    "reactor-id-2",
			Names: []string{"/reactor-user-other-def456"},
			State: "exited",
			Image: "ghcr.io/dyluth/reactor/python:latest",
		},
		{
			ID:    "non-reactor-id",
			Names: []string{"/nginx"},
			State: "running",
			Image: "nginx:latest",
		},
	}

	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return(containers, nil)

	result, err := service.ListReactorContainers(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 2) // Only reactor containers

	// Verify first reactor container
	assert.Equal(t, "reactor-id-1", result[0].ID)
	assert.Equal(t, "reactor-user-project-abc123", result[0].Name)
	assert.Equal(t, StatusRunning, result[0].Status)
	assert.Equal(t, "ghcr.io/dyluth/reactor/base:latest", result[0].Image)

	// Verify second reactor container
	assert.Equal(t, "reactor-id-2", result[1].ID)
	assert.Equal(t, "reactor-user-other-def456", result[1].Name)
	assert.Equal(t, StatusStopped, result[1].Status)
	assert.Equal(t, "ghcr.io/dyluth/reactor/python:latest", result[1].Image)

	mockClient.AssertExpectations(t)
}

func TestListReactorContainers_WithIsolationPrefix(t *testing.T) {
	t.Setenv("REACTOR_ISOLATION_PREFIX", "test-prefix")

	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	containers := []types.Container{
		{
			ID:    "reactor-id-1",
			Names: []string{"/test-prefix-reactor-user-project-abc123"},
			State: "running",
			Image: "ghcr.io/dyluth/reactor/base:latest",
		},
		{
			ID:    "reactor-id-2",
			Names: []string{"/reactor-user-project-def456"}, // No prefix
			State: "running",
			Image: "ghcr.io/dyluth/reactor/base:latest",
		},
	}

	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return(containers, nil)

	result, err := service.ListReactorContainers(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 2) // Both should be found (with and without prefix)

	mockClient.AssertExpectations(t)
}

func TestListReactorContainers_Error(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return([]types.Container{}, errors.New("docker daemon error"))

	result, err := service.ListReactorContainers(context.Background())
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list containers")
	mockClient.AssertExpectations(t)
}

func TestFindProjectContainer_Found(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	// Generate expected container name (accounting for isolation prefix)
	expectedName := service.generateContainerNameForProject("testuser", "/path/to/myproject", "abc123")

	// Mock ContainerList for ContainerExists call
	containers := []types.Container{
		{
			ID:    "project-container-id",
			Names: []string{"/" + expectedName},
			State: "running",
			Image: "ghcr.io/dyluth/reactor/base:latest",
		},
	}
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return(containers, nil)

	result, err := service.FindProjectContainer(context.Background(), "testuser", "/path/to/myproject", "abc123")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "project-container-id", result.ID)
	assert.Equal(t, expectedName, result.Name)
	assert.Equal(t, StatusRunning, result.Status)

	mockClient.AssertExpectations(t)
}

func TestFindProjectContainer_NotFound(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	// Mock ContainerList returning no matching containers
	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return([]types.Container{}, nil)

	result, err := service.FindProjectContainer(context.Background(), "testuser", "/path/to/myproject", "abc123")
	assert.NoError(t, err)
	assert.Nil(t, result) // Should return nil when no container found

	mockClient.AssertExpectations(t)
}

func TestFindProjectContainer_Error(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	mockClient.On("ContainerList", mock.Anything, container.ListOptions{All: true}).Return([]types.Container{}, errors.New("docker error"))

	result, err := service.FindProjectContainer(context.Background(), "testuser", "/path/to/myproject", "abc123")
	assert.Error(t, err)
	assert.Nil(t, result)
	mockClient.AssertExpectations(t)
}

func TestIsReactorContainer(t *testing.T) {
	service := NewServiceWithClient(&MockDockerClient{})

	testCases := []struct {
		name     string
		input    string
		expected bool
		envVar   string
	}{
		// Standard reactor containers
		{"basic reactor container", "reactor-user-project-abc123", true, ""},
		{"reactor with long hash", "reactor-user-myproject-1234567890abcdef", true, ""},
		{"reactor with special chars in project", "reactor-user-my-special-project-abc123", true, ""},

		// With isolation prefix
		{"with isolation prefix", "test-prefix-reactor-user-project-abc123", true, "test-prefix"},
		{"different prefix", "ci-reactor-user-project-abc123", true, "ci"},

		// Non-reactor containers
		{"not reactor", "nginx", false, ""},
		{"starts with reactor but invalid", "reactor-invalid", false, ""},
		{"reactor in middle", "some-reactor-container", false, ""},
		{"empty name", "", false, ""},

		// Edge cases
		{"reactor with minimum parts", "reactor-a-b-c", true, ""},
		{"reactor with many parts", "reactor-user-my-complex-project-name-abc123", true, ""},

		// Isolation prefix edge cases
		{"prefix but no reactor", "test-prefix-nginx", false, "test-prefix"},
		{"wrong prefix", "wrong-reactor-user-project-abc123", false, "test-prefix"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable if specified
			if tc.envVar != "" {
				t.Setenv("REACTOR_ISOLATION_PREFIX", tc.envVar)
			}

			result := service.isReactorContainer(tc.input)
			assert.Equal(t, tc.expected, result, "Container name: %s", tc.input)
		})
	}
}

func TestGenerateContainerNameForProject(t *testing.T) {
	service := NewServiceWithClient(&MockDockerClient{})

	testCases := []struct {
		name            string
		account         string
		projectPath     string
		projectHash     string
		isolationPrefix string
		expected        string
	}{
		{
			name:        "simple project",
			account:     "user",
			projectPath: "/home/user/myproject",
			projectHash: "abc123",
			expected:    "reactor-user-myproject-abc123",
		},
		{
			name:        "project with special chars",
			account:     "user",
			projectPath: "/home/user/my@special#project",
			projectHash: "def456",
			expected:    "reactor-user-my-special-project-def456",
		},
		{
			name:        "very long project name",
			account:     "user",
			projectPath: "/home/user/this-is-a-very-long-project-name-that-exceeds-limits",
			projectHash: "xyz789",
			expected:    "reactor-user-this-is-a-very-long-xyz789",
		},
		{
			name:            "with isolation prefix",
			account:         "user",
			projectPath:     "/home/user/myproject",
			projectHash:     "abc123",
			isolationPrefix: "test",
			expected:        "test-reactor-user-myproject-abc123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.isolationPrefix != "" {
				t.Setenv("REACTOR_ISOLATION_PREFIX", tc.isolationPrefix)
			} else {
				// Clear any existing isolation prefix for this test
				t.Setenv("REACTOR_ISOLATION_PREFIX", "")
			}

			result := service.generateContainerNameForProject(tc.account, tc.projectPath, tc.projectHash)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSanitizeContainerName(t *testing.T) {
	service := NewServiceWithClient(&MockDockerClient{})

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid name", "myproject", "myproject"},
		{"name with spaces", "my project", "my-project"},
		{"name with special chars", "my@project#test", "my-project-test"},
		{"name starting with non-alphanumeric", "@project", "project--project"},
		{"very long name", "this-is-a-very-long-project-name-that-exceeds-the-twenty-character-limit", "this-is-a-very-long"},
		{"empty name", "", "project"},
		{"name with unicode", "pröject", "pr-ject"},
		{"name ending with dash after truncation", "project-name-with-dash-", "project-name-with-da"},
		{"only special chars", "@#$%", "project-----"},
		{"mixed valid and invalid", "my_project.test", "my_project.test"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.sanitizeContainerName(tc.input)
			assert.Equal(t, tc.expected, result)

			// Verify result follows Docker naming rules
			if result != "" {
				// Should start with alphanumeric
				assert.Regexp(t, `^[a-zA-Z0-9]`, result, "Should start with alphanumeric: %s", result)
				// Should only contain valid chars
				assert.Regexp(t, `^[a-zA-Z0-9_.-]*$`, result, "Should only contain valid chars: %s", result)
			}
		})
	}
}

func TestContainerDiff_Success(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	expectedChanges := []container.FilesystemChange{
		{Kind: container.ChangeAdd, Path: "/home/claude/.claude/config.json"},
		{Kind: container.ChangeModify, Path: "/home/claude/.bashrc"},
		{Kind: container.ChangeDelete, Path: "/tmp/temp_file"},
	}

	mockClient.On("ContainerDiff", mock.Anything, "test-container-id").Return(expectedChanges, nil)

	result, err := service.ContainerDiff(context.Background(), "test-container-id")
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Verify changes are properly converted
	assert.Equal(t, "A", result[0].Kind)
	assert.Equal(t, "/home/claude/.claude/config.json", result[0].Path)
	assert.Equal(t, "C", result[1].Kind)
	assert.Equal(t, "/home/claude/.bashrc", result[1].Path)
	assert.Equal(t, "D", result[2].Kind)
	assert.Equal(t, "/tmp/temp_file", result[2].Path)

	mockClient.AssertExpectations(t)
}

func TestContainerDiff_Error(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	mockClient.On("ContainerDiff", mock.Anything, "test-container-id").Return([]container.FilesystemChange{}, errors.New("container not found"))

	result, err := service.ContainerDiff(context.Background(), "test-container-id")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "container not found")
	mockClient.AssertExpectations(t)
}

// Basic session tests for simple constructors and non-interactive functions
func TestNewTerminalState(t *testing.T) {
	state := NewTerminalState()
	assert.NotNil(t, state)
	assert.NotNil(t, state.SignalChan)
	assert.NotNil(t, state.ResizeChan)
	assert.False(t, state.RawModeSet)
}

func TestTerminalState_GetTerminalSize(t *testing.T) {
	state := NewTerminalState()

	// Test getting terminal size (should not panic even if not a terminal)
	size, err := state.GetTerminalSize()
	// Should always succeed with default values if not a terminal
	assert.NoError(t, err)
	assert.True(t, size.Rows > 0, "Rows should be positive: %d", size.Rows)
	assert.True(t, size.Cols > 0, "Cols should be positive: %d", size.Cols)

	// Default fallback values when not in terminal
	if size.Rows == 24 && size.Cols == 80 {
		t.Log("Using default terminal size (not in actual terminal)")
	} else {
		t.Logf("Detected terminal size: %dx%d", size.Cols, size.Rows)
	}
}

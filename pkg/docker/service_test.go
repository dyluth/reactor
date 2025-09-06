package docker

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDockerClient implements DockerClient interface for testing
type MockDockerClient struct {
	mock.Mock
}

// MockConn implements net.Conn for testing
type MockConn struct {
	*strings.Reader
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (m *MockConn) Close() error {
	return nil
}

func (m *MockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
}

func (m *MockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
}

func (m *MockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// NewMockHijackedResponse creates a mock with readable content
func NewMockHijackedResponse(output string) types.HijackedResponse {
	reader := strings.NewReader(output)
	conn := &MockConn{Reader: strings.NewReader(output)}
	return types.HijackedResponse{
		Conn:   conn,
		Reader: bufio.NewReader(reader),
	}
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

func (m *MockDockerClient) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	args := m.Called(ctx, execID)
	return args.Get(0).(types.ContainerExecInspect), args.Error(1)
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

func (m *MockDockerClient) ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	args := m.Called(ctx, buildContext, options)
	return args.Get(0).(types.ImageBuildResponse), args.Error(1)
}

func (m *MockDockerClient) ImageList(ctx context.Context, options types.ImageListOptions) ([]image.Summary, error) { //nolint:staticcheck // image.Summary not available in this Docker client version
	args := m.Called(ctx, options)
	return args.Get(0).([]image.Summary), args.Error(1) //nolint:staticcheck // image.Summary not available in this Docker client version
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

// Test terminal state setup and cleanup in non-interactive environments
func TestTerminalState_Setup_NonInteractive(t *testing.T) {
	state := NewTerminalState()

	// Setup should succeed in non-interactive environment
	err := state.Setup()
	assert.NoError(t, err)

	// In non-interactive mode, raw mode should not be set
	assert.False(t, state.RawModeSet)
	assert.Nil(t, state.OriginalState)

	// Size should be set to defaults or actual size (may be 0 if not set yet)
	// Just check that we don't panic
	assert.NotNil(t, state)
}

func TestTerminalState_Cleanup_SafeMultipleCalls(t *testing.T) {
	state := NewTerminalState()

	// Cleanup should be safe to call without setup
	err := state.Cleanup()
	assert.NoError(t, err)

	// Multiple cleanup calls should be safe
	err = state.Cleanup()
	assert.NoError(t, err)

	// Setup then cleanup should work
	err = state.Setup()
	assert.NoError(t, err)

	err = state.Cleanup()
	assert.NoError(t, err)

	// Channels should be closed after cleanup
	assert.Nil(t, state.SignalChan)
	assert.Nil(t, state.ResizeChan)
}

func TestTerminalState_GetTerminalSize_NonInteractive(t *testing.T) {
	state := NewTerminalState()

	// Should return default size in non-interactive environment
	size, err := state.GetTerminalSize()
	assert.NoError(t, err)
	assert.Equal(t, uint16(24), size.Rows)
	assert.Equal(t, uint16(80), size.Cols)
}

func TestTerminalState_StartSignalHandling_NonInteractive(t *testing.T) {
	state := NewTerminalState()

	// Should not panic in non-interactive environment
	assert.NotPanics(t, func() {
		state.StartSignalHandling()
	})

	// Signal channel should be initialized
	assert.NotNil(t, state.SignalChan)

	// Cleanup should work after signal handling setup
	err := state.Cleanup()
	assert.NoError(t, err)
}

func TestService_ResizeTerminal(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	// Set up mock expectation for ContainerResize
	mockClient.On("ContainerResize", mock.Anything, "test-container", mock.AnythingOfType("container.ResizeOptions")).Return(nil)

	// ResizeTerminal should handle non-interactive environment gracefully
	newSize := TTYSize{Rows: 30, Cols: 100}
	err := service.ResizeTerminal(context.Background(), "test-container", newSize)

	// Should succeed with proper mock setup
	assert.NoError(t, err)

	// Test should complete without panicking
	assert.NotNil(t, service)
	mockClient.AssertExpectations(t)
}

func TestService_AttachInteractiveSession_NonInteractive(t *testing.T) {
	// This tests the parameter validation and early exit behavior
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	containerID := "test-container-id"

	// Set up mock expectation for ContainerInspect to return a running container
	containerState := types.ContainerState{Running: false} // Not running
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &containerState,
		},
	}
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(containerJSON, nil)

	// Should return error when container is not running
	err := service.AttachInteractiveSession(context.Background(), containerID)

	// Should get "container is not running" error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not running")
	mockClient.AssertExpectations(t)
}

func TestService_AttachInteractiveSession_RunningContainer(t *testing.T) {
	// Test deeper into the function with running container but fail at exec creation
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	containerID := "test-container-id"

	// Set up mock for ContainerInspect to return running container
	containerState := types.ContainerState{Running: true}
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &containerState,
		},
	}
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(containerJSON, nil)

	// Set up mock for ContainerExecCreate to fail (simulates exec creation error)
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("types.ExecConfig")).Return(types.IDResponse{}, errors.New("exec creation failed"))

	// Should get exec creation failure
	err := service.AttachInteractiveSession(context.Background(), containerID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create exec instance")
	mockClient.AssertExpectations(t)
}

func TestService_AttachInteractiveSession_AttachFailure(t *testing.T) {
	// Test even deeper into the function - pass exec creation but fail at attach
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	containerID := "test-container-id"
	execID := "test-exec-id"

	// Set up mock for ContainerInspect to return running container
	containerState := types.ContainerState{Running: true}
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &containerState,
		},
	}
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(containerJSON, nil)

	// Set up mock for ContainerExecCreate to succeed
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("types.ExecConfig")).Return(types.IDResponse{ID: execID}, nil)

	// Set up mock for ContainerExecAttach to fail
	mockClient.On("ContainerExecAttach", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(types.HijackedResponse{}, errors.New("attach failed"))

	// Should get attach failure error
	err := service.AttachInteractiveSession(context.Background(), containerID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to attach to exec instance")
	mockClient.AssertExpectations(t)
}

func TestTerminalState_ChannelInitialization(t *testing.T) {
	state := NewTerminalState()

	// Channels should be initialized
	assert.NotNil(t, state.SignalChan)
	assert.NotNil(t, state.ResizeChan)

	// Channels should be buffered
	select {
	case state.ResizeChan <- TTYSize{Rows: 25, Cols: 81}:
		// Should not block
	default:
		t.Error("Resize channel should be buffered")
	}
}

func TestTerminalState_ConcurrentAccess(t *testing.T) {
	state := NewTerminalState()

	// Test concurrent setup/cleanup doesn't panic
	var wg sync.WaitGroup
	errors := make(chan error, 4)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := state.Setup(); err != nil {
				errors <- err
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := state.Cleanup(); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors (some might be expected due to concurrency)
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
			t.Logf("Concurrent operation error (may be expected): %v", err)
		}
	}

	// The important thing is that it didn't panic
	t.Logf("Concurrent access test completed with %d errors (panics would be more serious)", errorCount)
}

// Test NewService error path (currently at 75% coverage)
func TestNewService_ErrorPath(t *testing.T) {
	// This test is tricky because NewService typically either works or doesn't
	// But we can test the construction path
	service, err := NewService()
	if err != nil {
		// Expected in environments without Docker
		t.Logf("NewService error (expected in CI): %v", err)
		assert.Nil(t, service)
	} else {
		// Success path
		assert.NotNil(t, service)
		assert.NotNil(t, service.client)
		_ = service.Close()
	}
}

// Test CheckHealth error conditions (currently at 87.5% coverage)
func TestCheckHealth_DetailedError(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	// Test ping failure
	mockClient.On("Ping", mock.Anything).Return(types.Ping{}, context.Canceled)

	err := service.CheckHealth(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "docker daemon is not accessible")

	mockClient.AssertExpectations(t)
}

// Test ContainerExists edge cases (currently at 93.3% coverage)
func TestContainerExists_EdgeCases(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	// Set up mock to return empty container list (no containers found)
	mockClient.On("ContainerList", mock.Anything, mock.AnythingOfType("container.ListOptions")).Return([]types.Container{}, nil)

	// Test with empty container name
	containerInfo, err := service.ContainerExists(context.Background(), "")
	assert.Equal(t, StatusNotFound, containerInfo.Status)
	assert.NoError(t, err) // Current implementation doesn't validate empty names

	// Test with whitespace-only name
	containerInfo, err = service.ContainerExists(context.Background(), "   ")
	assert.Equal(t, StatusNotFound, containerInfo.Status)
	assert.NoError(t, err) // Current implementation doesn't validate whitespace names

	mockClient.AssertExpectations(t)
}

// Test CreateContainer edge cases (currently at 94.1% coverage)
func TestCreateContainer_EdgeCases(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	// Test with empty name - should be allowed to continue (Docker will handle it)
	spec := &ContainerSpec{
		Name:  "",
		Image: "test:latest",
	}

	// Set up mock to simulate successful container creation with empty name
	mockClient.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, "").Return(container.CreateResponse{ID: "test-id"}, nil)

	containerInfo, err := service.CreateContainer(context.Background(), spec)
	assert.NoError(t, err)
	assert.Equal(t, "test-id", containerInfo.ID)
	assert.Equal(t, "", containerInfo.Name) // Empty name passed through

	mockClient.AssertExpectations(t)
}

// Test ContainerDiff error cases (currently at 93.3% coverage)
func TestContainerDiff_ErrorCases(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	// Test with empty container name - Docker will handle validation
	mockClient.On("ContainerDiff", mock.Anything, "").Return(
		[]container.FilesystemChange{}, errors.New("no such container"))

	_, err := service.ContainerDiff(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get container diff")

	// Test with container that doesn't exist
	mockClient.On("ContainerDiff", mock.Anything, "nonexistent").Return(
		[]container.FilesystemChange{}, errors.New("no such container"))

	_, err = service.ContainerDiff(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get container diff")

	mockClient.AssertExpectations(t)
}

func TestProvisionContainerWithCleanup_StopFailure(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	spec := &ContainerSpec{
		Name:  "test-container",
		Image: "test:latest",
	}

	// Mock container exists running
	mockClient.On("ContainerList", mock.Anything, mock.AnythingOfType("container.ListOptions")).Return(
		[]types.Container{{
			ID:    "existing-id",
			Names: []string{"/test-container"},
			State: "running",
			Image: "test:latest",
		}}, nil)

	// Mock stop container failure
	mockClient.On("ContainerStop", mock.Anything, "existing-id", mock.AnythingOfType("container.StopOptions")).Return(errors.New("failed to stop"))

	// Should get error when stop fails during force cleanup
	_, err := service.ProvisionContainerWithCleanup(context.Background(), spec, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop existing container for cleanup")

	mockClient.AssertExpectations(t)
}

func TestProvisionContainerWithCleanup_RemoveFailureDuringCleanup(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	spec := &ContainerSpec{
		Name:  "test-container",
		Image: "test:latest",
	}

	// Mock container exists stopped
	mockClient.On("ContainerList", mock.Anything, mock.AnythingOfType("container.ListOptions")).Return(
		[]types.Container{{
			ID:    "existing-id",
			Names: []string{"/test-container"},
			State: "exited",
			Image: "test:latest",
		}}, nil)

	// Mock remove container failure during force cleanup
	mockClient.On("ContainerRemove", mock.Anything, "existing-id", mock.AnythingOfType("container.RemoveOptions")).Return(errors.New("failed to remove"))

	// Should get error when remove fails during force cleanup
	_, err := service.ProvisionContainerWithCleanup(context.Background(), spec, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove existing container for cleanup")

	mockClient.AssertExpectations(t)
}

// Test ListReactorContainers error edge case (currently at 94.4% coverage)
func TestListReactorContainers_FilterError(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := NewServiceWithClient(mockClient)

	// Test container list error
	mockClient.On("ContainerList", mock.Anything, mock.AnythingOfType("container.ListOptions")).Return(
		[]types.Container{}, context.DeadlineExceeded)

	_, err := service.ListReactorContainers(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list containers")

	mockClient.AssertExpectations(t)
}

// BUILD FUNCTIONALITY TESTS

func TestImageExists_Found(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	imageName := "reactor-build:abc12345"

	// Mock image list with matching image
	mockClient.On("ImageList", mock.Anything, types.ImageListOptions{}).Return(
		[]image.Summary{ //nolint:staticcheck // image.Summary not available in this Docker client version
			{RepoTags: []string{"reactor-build:abc12345", "reactor-build:latest"}},
			{RepoTags: []string{"other-image:latest"}},
		}, nil)

	exists, err := service.ImageExists(context.Background(), imageName)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestImageExists_NotFound(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	imageName := "reactor-build:notfound"

	// Mock image list without matching image
	mockClient.On("ImageList", mock.Anything, types.ImageListOptions{}).Return(
		[]image.Summary{ //nolint:staticcheck // image.Summary not available in this Docker client version
			{RepoTags: []string{"reactor-build:other", "reactor-build:latest"}},
			{RepoTags: []string{"different-image:latest"}},
		}, nil)

	exists, err := service.ImageExists(context.Background(), imageName)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestImageExists_Error(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	imageName := "reactor-build:abc12345"

	// Mock image list error
	mockClient.On("ImageList", mock.Anything, types.ImageListOptions{}).Return(
		[]image.Summary{}, errors.New("docker daemon error")) //nolint:staticcheck // image.Summary not available in this Docker client version

	exists, err := service.ImageExists(context.Background(), imageName)
	assert.Error(t, err)
	assert.False(t, exists)
	assert.Contains(t, err.Error(), "failed to list images")
}

// POST CREATE COMMAND FUNCTIONALITY TESTS

func TestExecutePostCreateCommand_NilCommand(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// No mocks needed since function should return early
	err := service.ExecutePostCreateCommand(context.Background(), "test-container", nil)
	assert.NoError(t, err)
}

func TestExecutePostCreateCommand_EmptyStringCommand(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// No mocks needed since function should return early
	err := service.ExecutePostCreateCommand(context.Background(), "test-container", "")
	assert.NoError(t, err)
}

func TestExecutePostCreateCommand_WhitespaceStringCommand(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	// No mocks needed since function should return early
	err := service.ExecutePostCreateCommand(context.Background(), "test-container", "   \t\n  ")
	assert.NoError(t, err)
}

func TestExecutePostCreateCommand_StringCommand_Success(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	containerID := "test-container"
	command := "echo 'Hello World'"

	// Mock container running check
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: true},
		},
	}
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(containerJSON, nil)

	// Mock exec creation
	execID := "exec-123"
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.MatchedBy(func(config types.ExecConfig) bool {
		return len(config.Cmd) == 3 &&
			config.Cmd[0] == "/bin/sh" &&
			config.Cmd[1] == "-c" &&
			config.Cmd[2] == command &&
			config.AttachStdout && config.AttachStderr && !config.AttachStdin
	})).Return(types.IDResponse{ID: execID}, nil)

	// Mock exec attach
	mockAttachResp := NewMockHijackedResponse("postCreateCommand output\n")
	mockClient.On("ContainerExecAttach", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(mockAttachResp, nil)

	// Mock exec start
	mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(nil)

	// Mock exec inspect for exit code
	mockClient.On("ContainerExecInspect", mock.Anything, execID).Return(types.ContainerExecInspect{ExitCode: 0}, nil)

	err := service.ExecutePostCreateCommand(context.Background(), containerID, command)
	assert.NoError(t, err)
}

func TestExecutePostCreateCommand_ArrayCommand_Success(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	containerID := "test-container"
	command := []string{"npm", "install"}

	// Mock container running check
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: true},
		},
	}
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(containerJSON, nil)

	// Mock exec creation
	execID := "exec-456"
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.MatchedBy(func(config types.ExecConfig) bool {
		return len(config.Cmd) == 2 &&
			config.Cmd[0] == "npm" &&
			config.Cmd[1] == "install" &&
			config.AttachStdout && config.AttachStderr && !config.AttachStdin
	})).Return(types.IDResponse{ID: execID}, nil)

	// Mock exec attach
	mockAttachResp := NewMockHijackedResponse("postCreateCommand output\n")
	mockClient.On("ContainerExecAttach", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(mockAttachResp, nil)

	// Mock exec start
	mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(nil)

	// Mock exec inspect for exit code
	mockClient.On("ContainerExecInspect", mock.Anything, execID).Return(types.ContainerExecInspect{ExitCode: 0}, nil)

	err := service.ExecutePostCreateCommand(context.Background(), containerID, command)
	assert.NoError(t, err)
}

func TestExecutePostCreateCommand_InterfaceArrayCommand_Success(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	containerID := "test-container"
	// Simulate how JSON unmarshaling creates []interface{} from devcontainer.json
	command := []interface{}{"pip", "install", "-r", "requirements.txt"}

	// Mock container running check
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: true},
		},
	}
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(containerJSON, nil)

	// Mock exec creation
	execID := "exec-789"
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.MatchedBy(func(config types.ExecConfig) bool {
		return len(config.Cmd) == 4 &&
			config.Cmd[0] == "pip" &&
			config.Cmd[1] == "install" &&
			config.Cmd[2] == "-r" &&
			config.Cmd[3] == "requirements.txt"
	})).Return(types.IDResponse{ID: execID}, nil)

	// Mock exec attach
	mockAttachResp := NewMockHijackedResponse("postCreateCommand output\n")
	mockClient.On("ContainerExecAttach", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(mockAttachResp, nil)

	// Mock exec start
	mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(nil)

	// Mock exec inspect for exit code
	mockClient.On("ContainerExecInspect", mock.Anything, execID).Return(types.ContainerExecInspect{ExitCode: 0}, nil)

	err := service.ExecutePostCreateCommand(context.Background(), containerID, command)
	assert.NoError(t, err)
}

func TestExecutePostCreateCommand_ContainerNotRunning(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	containerID := "test-container"
	command := "echo test"

	// Mock container not running
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: false},
		},
	}
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(containerJSON, nil)

	err := service.ExecutePostCreateCommand(context.Background(), containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container is not running")
}

func TestExecutePostCreateCommand_ContainerInspectFails(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	containerID := "test-container"
	command := "echo test"

	// Mock container inspect failure
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(types.ContainerJSON{}, errors.New("container not found"))

	err := service.ExecutePostCreateCommand(context.Background(), containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to inspect container")
}

func TestExecutePostCreateCommand_ExecCreateFails(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	containerID := "test-container"
	command := "echo test"

	// Mock container running check
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: true},
		},
	}
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(containerJSON, nil)

	// Mock exec create failure
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("types.ExecConfig")).Return(types.IDResponse{}, errors.New("exec create failed"))

	err := service.ExecutePostCreateCommand(context.Background(), containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create exec instance for postCreateCommand")
}

func TestExecutePostCreateCommand_ExecAttachFails(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	containerID := "test-container"
	command := "echo test"

	// Mock container running check
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: true},
		},
	}
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(containerJSON, nil)

	// Mock exec creation
	execID := "exec-123"
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("types.ExecConfig")).Return(types.IDResponse{ID: execID}, nil)

	// Mock exec start (which happens before attach)
	mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(nil)

	// Mock exec attach failure
	mockClient.On("ContainerExecAttach", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(types.HijackedResponse{}, errors.New("attach failed"))

	err := service.ExecutePostCreateCommand(context.Background(), containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to attach to postCreateCommand execution")
}

func TestExecutePostCreateCommand_ExecStartFails(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	containerID := "test-container"
	command := "echo test"

	// Mock container running check
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: true},
		},
	}
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(containerJSON, nil)

	// Mock exec creation
	execID := "exec-123"
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("types.ExecConfig")).Return(types.IDResponse{ID: execID}, nil)

	// Mock exec start failure (attach won't be called because start fails)
	mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(errors.New("start failed"))

	err := service.ExecutePostCreateCommand(context.Background(), containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start postCreateCommand execution")
}

func TestExecutePostCreateCommand_NonZeroExitCode(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	containerID := "test-container"
	command := "exit 1"

	// Mock container running check
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: true},
		},
	}
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(containerJSON, nil)

	// Mock exec creation
	execID := "exec-123"
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("types.ExecConfig")).Return(types.IDResponse{ID: execID}, nil)

	// Mock exec attach
	mockAttachResp := NewMockHijackedResponse("postCreateCommand output\n")
	mockClient.On("ContainerExecAttach", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(mockAttachResp, nil)

	// Mock exec start
	mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(nil)

	// Mock exec inspect with non-zero exit code
	mockClient.On("ContainerExecInspect", mock.Anything, execID).Return(types.ContainerExecInspect{ExitCode: 1}, nil)

	err := service.ExecutePostCreateCommand(context.Background(), containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postCreateCommand failed with exit code 1")
}

func TestExecutePostCreateCommand_ExecInspectFails(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	containerID := "test-container"
	command := "echo test"

	// Mock container running check
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: true},
		},
	}
	mockClient.On("ContainerInspect", mock.Anything, containerID).Return(containerJSON, nil)

	// Mock exec creation
	execID := "exec-123"
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("types.ExecConfig")).Return(types.IDResponse{ID: execID}, nil)

	// Mock exec attach
	mockAttachResp := NewMockHijackedResponse("postCreateCommand output\n")
	mockClient.On("ContainerExecAttach", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(mockAttachResp, nil)

	// Mock exec start
	mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("types.ExecStartCheck")).Return(nil)

	// Mock exec inspect failure
	mockClient.On("ContainerExecInspect", mock.Anything, execID).Return(types.ContainerExecInspect{}, errors.New("inspect failed"))

	err := service.ExecutePostCreateCommand(context.Background(), containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to inspect postCreateCommand execution")
}

func TestExecutePostCreateCommand_InvalidCommandType(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	containerID := "test-container"
	command := 12345 // Invalid type (int)

	// No mocks needed since function should return early
	err := service.ExecutePostCreateCommand(context.Background(), containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postCreateCommand must be a string or array of strings, got int")
}

func TestExecutePostCreateCommand_InterfaceArrayWithInvalidType(t *testing.T) {
	service, mockClient := setupTestService()
	defer mockClient.AssertExpectations(t)

	containerID := "test-container"
	command := []interface{}{"npm", "install", 123} // Invalid type in array

	// No mocks needed since function should return early
	err := service.ExecutePostCreateCommand(context.Background(), containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postCreateCommand array contains non-string element: 123")
}

// TestBuildImage test suite

func TestBuildImage_Success(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()

	// Create temporary build context directory
	tempDir := os.TempDir()
	workspaceDir := filepath.Join(tempDir, "reactor-test-build-"+strings.ReplaceAll(t.Name(), "/", "-"))
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(workspaceDir) }()

	// Create Dockerfile
	dockerfilePath := filepath.Join(workspaceDir, "Dockerfile")
	err = os.WriteFile(dockerfilePath, []byte("FROM alpine:latest\nRUN echo 'test'\n"), 0644)
	assert.NoError(t, err)

	spec := BuildSpec{
		Context:    workspaceDir,
		Dockerfile: "Dockerfile",
		ImageName:  "test-image:latest",
	}

	// Mock ImageExists to return false (image doesn't exist)
	mockClient.On("ImageList", mock.Anything, types.ImageListOptions{}).Return([]image.Summary{}, nil)

	// Mock successful build
	buildOutput := `{"stream":"Step 1/2 : FROM alpine:latest\n"}` + "\n" +
		`{"stream":"Successfully built abc123\n"}` + "\n" +
		`{"stream":"Successfully tagged test-image:latest\n"}` + "\n"

	mockResponse := types.ImageBuildResponse{
		Body: io.NopCloser(strings.NewReader(buildOutput)),
	}
	mockClient.On("ImageBuild", mock.Anything, mock.Anything, mock.MatchedBy(func(opts types.ImageBuildOptions) bool {
		return opts.Dockerfile == "Dockerfile" &&
			len(opts.Tags) == 1 && opts.Tags[0] == "test-image:latest"
	})).Return(mockResponse, nil)

	err = service.BuildImage(ctx, spec, false)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestBuildImage_ForceRebuild(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()

	// Create temporary build context directory
	tempDir := os.TempDir()
	workspaceDir := filepath.Join(tempDir, "reactor-test-build-"+strings.ReplaceAll(t.Name(), "/", "-"))
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(workspaceDir) }()

	// Create Dockerfile
	dockerfilePath := filepath.Join(workspaceDir, "Dockerfile")
	err = os.WriteFile(dockerfilePath, []byte("FROM alpine:latest\n"), 0644)
	assert.NoError(t, err)

	spec := BuildSpec{
		Context:    workspaceDir,
		Dockerfile: "Dockerfile",
		ImageName:  "test-image:latest",
	}

	// With forceRebuild=true, should skip ImageExists check and build anyway
	buildOutput := `{"stream":"Successfully built abc123\n"}`
	mockResponse := types.ImageBuildResponse{
		Body: io.NopCloser(strings.NewReader(buildOutput)),
	}
	mockClient.On("ImageBuild", mock.Anything, mock.Anything, mock.Anything).Return(mockResponse, nil)

	err = service.BuildImage(ctx, spec, true)
	assert.NoError(t, err)

	// Should not call ImageList when forceRebuild=true
	mockClient.AssertNotCalled(t, "ImageList")
	mockClient.AssertExpectations(t)
}

func TestBuildImage_ImageExistsSkipBuild(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()

	// Create temporary build context directory
	tempDir := os.TempDir()
	workspaceDir := filepath.Join(tempDir, "reactor-test-build-"+strings.ReplaceAll(t.Name(), "/", "-"))
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(workspaceDir) }()

	// Create Dockerfile
	dockerfilePath := filepath.Join(workspaceDir, "Dockerfile")
	err = os.WriteFile(dockerfilePath, []byte("FROM alpine:latest\n"), 0644)
	assert.NoError(t, err)

	spec := BuildSpec{
		Context:    workspaceDir,
		Dockerfile: "Dockerfile",
		ImageName:  "test-image:latest",
	}

	// Mock ImageExists to return true (image exists)
	mockClient.On("ImageList", mock.Anything, types.ImageListOptions{}).Return([]image.Summary{
		{ID: "abc123", RepoTags: []string{"test-image:latest"}},
	}, nil)

	err = service.BuildImage(ctx, spec, false)
	assert.NoError(t, err)

	// Should not call ImageBuild when image exists and forceRebuild=false
	mockClient.AssertNotCalled(t, "ImageBuild")
	mockClient.AssertExpectations(t)
}

func TestBuildImage_ContextDoesNotExist(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()

	// Mock ImageExists to return false (since we check image existence first)
	mockClient.On("ImageList", mock.Anything, types.ImageListOptions{}).Return([]image.Summary{}, nil)

	spec := BuildSpec{
		Context:    "/nonexistent/path",
		Dockerfile: "Dockerfile",
		ImageName:  "test-image:latest",
	}

	err := service.BuildImage(ctx, spec, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "build context directory does not exist")
}

func TestBuildImage_DockerfileDoesNotExist(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()

	// Mock ImageExists to return false (since we check image existence first)
	mockClient.On("ImageList", mock.Anything, types.ImageListOptions{}).Return([]image.Summary{}, nil)

	// Create temporary build context directory without Dockerfile
	tempDir := os.TempDir()
	workspaceDir := filepath.Join(tempDir, "reactor-test-build-"+strings.ReplaceAll(t.Name(), "/", "-"))
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(workspaceDir) }()

	spec := BuildSpec{
		Context:    workspaceDir,
		Dockerfile: "Dockerfile",
		ImageName:  "test-image:latest",
	}

	err = service.BuildImage(ctx, spec, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dockerfile does not exist")
}

func TestBuildImage_BuildFails(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()

	// Create temporary build context directory
	tempDir := os.TempDir()
	workspaceDir := filepath.Join(tempDir, "reactor-test-build-"+strings.ReplaceAll(t.Name(), "/", "-"))
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(workspaceDir) }()

	// Create Dockerfile
	dockerfilePath := filepath.Join(workspaceDir, "Dockerfile")
	err = os.WriteFile(dockerfilePath, []byte("FROM alpine:latest\n"), 0644)
	assert.NoError(t, err)

	spec := BuildSpec{
		Context:    workspaceDir,
		Dockerfile: "Dockerfile",
		ImageName:  "test-image:latest",
	}

	// Mock ImageExists to return false
	mockClient.On("ImageList", mock.Anything, types.ImageListOptions{}).Return([]image.Summary{}, nil)

	// Mock build failure
	mockClient.On("ImageBuild", mock.Anything, mock.Anything, mock.Anything).Return(types.ImageBuildResponse{}, errors.New("build failed"))

	err = service.BuildImage(ctx, spec, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build image")
	mockClient.AssertExpectations(t)
}

func TestBuildImage_StreamOutputError(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()

	// Create temporary build context directory
	tempDir := os.TempDir()
	workspaceDir := filepath.Join(tempDir, "reactor-test-build-"+strings.ReplaceAll(t.Name(), "/", "-"))
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(workspaceDir) }()

	// Create Dockerfile
	dockerfilePath := filepath.Join(workspaceDir, "Dockerfile")
	err = os.WriteFile(dockerfilePath, []byte("FROM alpine:latest\n"), 0644)
	assert.NoError(t, err)

	spec := BuildSpec{
		Context:    workspaceDir,
		Dockerfile: "Dockerfile",
		ImageName:  "test-image:latest",
	}

	// Mock ImageExists to return false
	mockClient.On("ImageList", mock.Anything, types.ImageListOptions{}).Return([]image.Summary{}, nil)

	// Mock build with error in output stream
	buildOutput := `{"errorDetail":{"message":"build error"},"error":"build error"}`
	mockResponse := types.ImageBuildResponse{
		Body: io.NopCloser(strings.NewReader(buildOutput)),
	}
	mockClient.On("ImageBuild", mock.Anything, mock.Anything, mock.Anything).Return(mockResponse, nil)

	err = service.BuildImage(ctx, spec, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "build failed")
	mockClient.AssertExpectations(t)
}

// TestExecuteInteractiveCommand test suite

func TestExecuteInteractiveCommand_Success(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()
	containerID := "test-container"
	command := []string{"bash", "-c", "echo hello"}

	// Mock container inspect - running container
	mockClient.On("ContainerInspect", ctx, containerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: true},
		},
	}, nil)

	// Mock exec create
	execResp := types.IDResponse{ID: "exec-123"}
	mockClient.On("ContainerExecCreate", ctx, containerID, mock.MatchedBy(func(config types.ExecConfig) bool {
		return config.AttachStdout && config.AttachStderr && config.AttachStdin && config.Tty &&
			len(config.Cmd) == 3 && config.Cmd[0] == "bash"
	})).Return(execResp, nil)

	// Mock exec attach
	attachResp := NewMockHijackedResponse("hello world\n")
	mockClient.On("ContainerExecAttach", ctx, "exec-123", mock.MatchedBy(func(config types.ExecStartCheck) bool {
		return config.Tty
	})).Return(attachResp, nil)

	// Mock exec start
	mockClient.On("ContainerExecStart", ctx, "exec-123", mock.MatchedBy(func(config types.ExecStartCheck) bool {
		return config.Tty
	})).Return(nil)

	// Mock exec inspect (command completed successfully)
	mockClient.On("ContainerExecInspect", ctx, "exec-123").Return(types.ContainerExecInspect{
		Running:  false,
		ExitCode: 0,
	}, nil)

	err := service.ExecuteInteractiveCommand(ctx, containerID, command)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestExecuteInteractiveCommand_EmptyCommand(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()
	containerID := "test-container"
	command := []string{}

	err := service.ExecuteInteractiveCommand(ctx, containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command array cannot be empty")

	// Should not call any Docker methods
	mockClient.AssertExpectations(t)
}

func TestExecuteInteractiveCommand_ContainerNotRunning(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()
	containerID := "test-container"
	command := []string{"bash"}

	// Mock container inspect - stopped container
	mockClient.On("ContainerInspect", ctx, containerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: false},
		},
	}, nil)

	err := service.ExecuteInteractiveCommand(ctx, containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container test-container is not running")
	mockClient.AssertExpectations(t)
}

func TestExecuteInteractiveCommand_ContainerInspectFails(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()
	containerID := "test-container"
	command := []string{"bash"}

	// Mock container inspect failure
	mockClient.On("ContainerInspect", ctx, containerID).Return(types.ContainerJSON{}, errors.New("container not found"))

	err := service.ExecuteInteractiveCommand(ctx, containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to inspect container")
	mockClient.AssertExpectations(t)
}

func TestExecuteInteractiveCommand_ExecCreateFails(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()
	containerID := "test-container"
	command := []string{"bash"}

	// Mock container inspect - running container
	mockClient.On("ContainerInspect", ctx, containerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: true},
		},
	}, nil)

	// Mock exec create failure
	mockClient.On("ContainerExecCreate", ctx, containerID, mock.Anything).Return(types.IDResponse{}, errors.New("exec create failed"))

	err := service.ExecuteInteractiveCommand(ctx, containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create exec instance")
	mockClient.AssertExpectations(t)
}

func TestExecuteInteractiveCommand_ExecAttachFails(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()
	containerID := "test-container"
	command := []string{"bash"}

	// Mock container inspect - running container
	mockClient.On("ContainerInspect", ctx, containerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: true},
		},
	}, nil)

	// Mock exec create
	execResp := types.IDResponse{ID: "exec-123"}
	mockClient.On("ContainerExecCreate", ctx, containerID, mock.Anything).Return(execResp, nil)

	// Mock exec attach failure
	mockClient.On("ContainerExecAttach", ctx, "exec-123", mock.Anything).Return(types.HijackedResponse{}, errors.New("attach failed"))

	err := service.ExecuteInteractiveCommand(ctx, containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to attach to exec instance")
	mockClient.AssertExpectations(t)
}

func TestExecuteInteractiveCommand_ExecStartFails(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()
	containerID := "test-container"
	command := []string{"bash"}

	// Mock container inspect - running container
	mockClient.On("ContainerInspect", ctx, containerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{Running: true},
		},
	}, nil)

	// Mock exec create
	execResp := types.IDResponse{ID: "exec-123"}
	mockClient.On("ContainerExecCreate", ctx, containerID, mock.Anything).Return(execResp, nil)

	// Mock exec attach
	attachResp := NewMockHijackedResponse("test output")
	mockClient.On("ContainerExecAttach", ctx, "exec-123", mock.Anything).Return(attachResp, nil)

	// Mock exec start failure
	mockClient.On("ContainerExecStart", ctx, "exec-123", mock.Anything).Return(errors.New("start failed"))

	err := service.ExecuteInteractiveCommand(ctx, containerID, command)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start command execution")
	mockClient.AssertExpectations(t)
}

// TestListContainersByLabel test suite

func TestListContainersByLabel_Success(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()
	labelKey := "com.reactor.workspace.service"
	labelValue := "api"

	// Mock container list
	mockContainers := []types.Container{
		{
			ID:    "container1",
			Names: []string{"/test-api-1"},
			Image: "test:latest",
			State: "running",
			Labels: map[string]string{
				"com.reactor.workspace.service":  "api",
				"com.reactor.workspace.instance": "abc123",
			},
		},
		{
			ID:    "container2",
			Names: []string{"/test-api-2"},
			Image: "test:latest",
			State: "exited",
			Labels: map[string]string{
				"com.reactor.workspace.service": "api",
			},
		},
		{
			ID:    "container3",
			Names: []string{"/other-service"},
			Image: "other:latest",
			State: "running",
			Labels: map[string]string{
				"com.reactor.workspace.service": "frontend", // Different service
			},
		},
		{
			ID:     "container4",
			Names:  []string{"/no-labels"},
			Image:  "none:latest",
			State:  "running",
			Labels: nil, // No labels
		},
	}

	mockClient.On("ContainerList", mock.Anything, mock.MatchedBy(func(opts container.ListOptions) bool {
		return opts.All == true
	})).Return(mockContainers, nil)

	result, err := service.ListContainersByLabel(ctx, labelKey, labelValue)
	assert.NoError(t, err)
	assert.Len(t, result, 2) // Only containers with matching label

	// Check first container (running)
	assert.Equal(t, "container1", result[0].ID)
	assert.Equal(t, "test-api-1", result[0].Name)
	assert.Equal(t, StatusRunning, result[0].Status)
	assert.Equal(t, "test:latest", result[0].Image)

	// Check second container (stopped)
	assert.Equal(t, "container2", result[1].ID)
	assert.Equal(t, "test-api-2", result[1].Name)
	assert.Equal(t, StatusStopped, result[1].Status)

	mockClient.AssertExpectations(t)
}

func TestListContainersByLabel_NoMatches(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()
	labelKey := "com.reactor.workspace.service"
	labelValue := "nonexistent"

	// Mock container list with containers that don't match
	mockContainers := []types.Container{
		{
			ID:    "container1",
			Names: []string{"/other-service"},
			State: "running",
			Labels: map[string]string{
				"com.reactor.workspace.service": "api", // Different value
			},
		},
		{
			ID:     "container2",
			Names:  []string{"/no-labels"},
			State:  "running",
			Labels: nil, // No labels
		},
	}

	mockClient.On("ContainerList", mock.Anything, mock.Anything).Return(mockContainers, nil)

	result, err := service.ListContainersByLabel(ctx, labelKey, labelValue)
	assert.NoError(t, err)
	assert.Len(t, result, 0) // No matching containers

	mockClient.AssertExpectations(t)
}

func TestListContainersByLabel_ContainerListError(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()
	labelKey := "com.reactor.workspace.service"
	labelValue := "api"

	// Mock container list error
	mockClient.On("ContainerList", mock.Anything, mock.Anything).Return([]types.Container{}, errors.New("docker daemon error"))

	result, err := service.ListContainersByLabel(ctx, labelKey, labelValue)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list containers by label")
	assert.Nil(t, result)

	mockClient.AssertExpectations(t)
}

func TestListContainersByLabel_StatusMapping(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()
	labelKey := "test.label"
	labelValue := "test"

	// Mock containers with different states
	mockContainers := []types.Container{
		{
			ID:     "running-container",
			Names:  []string{"/running"},
			State:  "running",
			Labels: map[string]string{"test.label": "test"},
		},
		{
			ID:     "stopped-container",
			Names:  []string{"/stopped"},
			State:  "stopped",
			Labels: map[string]string{"test.label": "test"},
		},
		{
			ID:     "exited-container",
			Names:  []string{"/exited"},
			State:  "exited",
			Labels: map[string]string{"test.label": "test"},
		},
		{
			ID:     "unknown-container",
			Names:  []string{"/unknown"},
			State:  "paused", // Unknown state
			Labels: map[string]string{"test.label": "test"},
		},
	}

	mockClient.On("ContainerList", mock.Anything, mock.Anything).Return(mockContainers, nil)

	result, err := service.ListContainersByLabel(ctx, labelKey, labelValue)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	// Check status mapping
	assert.Equal(t, StatusRunning, result[0].Status)
	assert.Equal(t, StatusStopped, result[1].Status)
	assert.Equal(t, StatusStopped, result[2].Status)  // exited -> stopped
	assert.Equal(t, StatusNotFound, result[3].Status) // unknown -> not found

	mockClient.AssertExpectations(t)
}

func TestListContainersByLabel_EmptyNames(t *testing.T) {
	mockClient := &MockDockerClient{}
	service := &Service{client: mockClient}
	ctx := context.Background()
	labelKey := "test.label"
	labelValue := "test"

	// Mock container with no names
	mockContainers := []types.Container{
		{
			ID:     "no-name-container",
			Names:  []string{}, // Empty names array
			State:  "running",
			Labels: map[string]string{"test.label": "test"},
		},
	}

	mockClient.On("ContainerList", mock.Anything, mock.Anything).Return(mockContainers, nil)

	result, err := service.ListContainersByLabel(ctx, labelKey, labelValue)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "", result[0].Name) // Empty name when Names array is empty

	mockClient.AssertExpectations(t)
}

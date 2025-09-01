package core

import (
	"os"
	"testing"

	"github.com/dyluth/reactor/pkg/config"
	"github.com/dyluth/reactor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeContainerName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid alphanumeric name",
			input:    "myproject123",
			expected: "myproject123",
		},
		{
			name:     "name with valid special characters",
			input:    "my-project_v1.0",
			expected: "my-project_v1.0",
		},
		{
			name:     "name with invalid characters",
			input:    "my@project#2024",
			expected: "my-project-2024",
		},
		{
			name:     "name starting with invalid character",
			input:    "-invalid-start",
			expected: "project--invalid-sta",
		},
		{
			name:     "name with special characters at start",
			input:    "@#$project",
			expected: "project----project",
		},
		{
			name:     "very long name gets truncated",
			input:    "verylongprojectnamethatshouldbetruncated",
			expected: "verylongprojectnamet",
		},
		{
			name:     "long name ending with hyphen after truncation",
			input:    "longproject-name-that-gets-truncated",
			expected: "longproject-name-tha",
		},
		{
			name:     "empty string fallback",
			input:    "",
			expected: "project",
		},
		{
			name:     "only invalid characters",
			input:    "@#$%^&*()",
			expected: "project----------",
		},
		{
			name:     "unicode characters",
			input:    "프로젝트",
			expected: "project-----",
		},
		{
			name:     "spaces get replaced with hyphens",
			input:    "my project name",
			expected: "my-project-name",
		},
		{
			name:     "mixed case preserved",
			input:    "MyProject123",
			expected: "MyProject123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeContainerName(tt.input)
			assert.Equal(t, tt.expected, result)
			
			// Verify result meets Docker container name requirements
			assert.LessOrEqual(t, len(result), 20, "name should not exceed max length")
			assert.NotEmpty(t, result, "name should not be empty")
			assert.Regexp(t, `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`, result, "name should match Docker naming pattern")
		})
	}
}

func TestGenerateContainerName(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	tests := []struct {
		name                string
		account             string
		projectPath         string
		projectHash         string
		isolationPrefix     string
		expectedPattern     string
	}{
		{
			name:            "basic container name",
			account:         "testuser",
			projectPath:     "/home/user/myproject",
			projectHash:     "abc123",
			expectedPattern: "^reactor-testuser-myproject-abc123$",
		},
		{
			name:            "with isolation prefix",
			account:         "testuser",
			projectPath:     "/home/user/myproject",
			projectHash:     "def456",
			isolationPrefix: "ci-test",
			expectedPattern: "^ci-test-reactor-testuser-myproject-def456$",
		},
		{
			name:            "project with special characters",
			account:         "user",
			projectPath:     "/path/to/my@project#2024",
			projectHash:     "xyz789",
			expectedPattern: "^reactor-user-my-project-2024-xyz789$",
		},
		{
			name:            "deeply nested project path",
			account:         "dev",
			projectPath:     "/very/deeply/nested/path/to/project",
			projectHash:     "nested123",
			expectedPattern: "^reactor-dev-project-nested123$",
		},
		{
			name:            "long folder name gets truncated",
			account:         "user",
			projectPath:     "/home/verylongprojectnamethatshouldbetruncated",
			projectHash:     "long123",
			expectedPattern: "^reactor-user-verylongprojectnamet-long123$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set isolation prefix if specified
			if tt.isolationPrefix != "" {
				t.Setenv("REACTOR_ISOLATION_PREFIX", tt.isolationPrefix)
			} else {
				_ = os.Unsetenv("REACTOR_ISOLATION_PREFIX")
			}

			result := GenerateContainerName(tt.account, tt.projectPath, tt.projectHash)
			assert.Regexp(t, tt.expectedPattern, result)
			
			// Verify Docker naming compliance
			assert.Regexp(t, `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`, result, "container name should be Docker compliant")
		})
	}
}

func TestGenerateDiscoveryContainerName(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	tests := []struct {
		name                string
		account             string
		projectPath         string
		projectHash         string
		isolationPrefix     string
		expectedPattern     string
	}{
		{
			name:            "basic discovery container name",
			account:         "testuser",
			projectPath:     "/home/user/myproject",
			projectHash:     "abc123",
			expectedPattern: "^reactor-discovery-testuser-myproject-abc123$",
		},
		{
			name:            "with isolation prefix",
			account:         "testuser",
			projectPath:     "/home/user/myproject", 
			projectHash:     "def456",
			isolationPrefix: "discovery-test",
			expectedPattern: "^discovery-test-reactor-discovery-testuser-myproject-def456$",
		},
		{
			name:            "discovery with special characters in project name",
			account:         "user",
			projectPath:     "/path/to/my@project#2024",
			projectHash:     "xyz789",
			expectedPattern: "^reactor-discovery-user-my-project-2024-xyz789$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set isolation prefix if specified
			if tt.isolationPrefix != "" {
				t.Setenv("REACTOR_ISOLATION_PREFIX", tt.isolationPrefix)
			} else {
				_ = os.Unsetenv("REACTOR_ISOLATION_PREFIX")
			}

			result := GenerateDiscoveryContainerName(tt.account, tt.projectPath, tt.projectHash)
			assert.Regexp(t, tt.expectedPattern, result)
			
			// Verify Docker naming compliance
			assert.Regexp(t, `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`, result, "discovery container name should be Docker compliant")
			
			// Verify discovery containers always have "discovery" in the name
			assert.Contains(t, result, "discovery", "discovery container names should contain 'discovery'")
		})
	}
}

func TestNewContainerBlueprint(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	// Create test resolved config
	resolved := &config.ResolvedConfig{
		Account:     "testuser",
		ProjectRoot: "/home/user/testproject",
		ProjectHash: "testhash123",
		Image:       "test-image:latest",
	}

	// Test mount specifications
	mounts := []MountSpec{
		{
			Source: "/host/path/config",
			Target: "/container/config",
			Type:   "bind",
		},
		{
			Source: "/host/path/data",
			Target: "/container/data", 
			Type:   "bind",
		},
	}

	// Test port mappings
	portMappings := []PortMapping{
		{HostPort: 8080, ContainerPort: 80},
		{HostPort: 3000, ContainerPort: 3000},
	}

	tests := []struct {
		name                   string
		isDiscovery           bool
		dockerHostIntegration bool
		isolationPrefix       string
		expectedNamePattern   string
		expectedDockerMounts  int
		expectedEnvironment   int
	}{
		{
			name:                 "regular container",
			isDiscovery:          false,
			dockerHostIntegration: false,
			expectedNamePattern: "^reactor-testuser-testproject-testhash123$",
			expectedDockerMounts: 2, // 2 mount specs
			expectedEnvironment:  0, // no special env vars
		},
		{
			name:                 "discovery container (no mounts)",
			isDiscovery:          true,
			dockerHostIntegration: false,
			expectedNamePattern: "^reactor-discovery-testuser-testproject-testhash123$",
			expectedDockerMounts: 0, // discovery mode has no mounts
			expectedEnvironment:  0, // no special env vars
		},
		{
			name:                 "regular container with Docker host integration",
			isDiscovery:          false,
			dockerHostIntegration: true,
			expectedNamePattern: "^reactor-testuser-testproject-testhash123$",
			expectedDockerMounts: 3, // 2 mount specs + Docker socket
			expectedEnvironment:  1, // REACTOR_DOCKER_HOST_INTEGRATION=true
		},
		{
			name:                 "discovery with Docker host integration",
			isDiscovery:          true,
			dockerHostIntegration: true,
			expectedNamePattern: "^reactor-discovery-testuser-testproject-testhash123$",
			expectedDockerMounts: 1, // only Docker socket mount
			expectedEnvironment:  1, // REACTOR_DOCKER_HOST_INTEGRATION=true
		},
		{
			name:                 "with isolation prefix",
			isDiscovery:          false,
			dockerHostIntegration: false,
			isolationPrefix:     "test-prefix",
			expectedNamePattern: "^test-prefix-reactor-testuser-testproject-testhash123$",
			expectedDockerMounts: 2, // 2 mount specs
			expectedEnvironment:  0, // no special env vars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set isolation prefix if specified
			if tt.isolationPrefix != "" {
				t.Setenv("REACTOR_ISOLATION_PREFIX", tt.isolationPrefix)
			} else {
				_ = os.Unsetenv("REACTOR_ISOLATION_PREFIX")
			}

			blueprint := NewContainerBlueprint(resolved, mounts, tt.isDiscovery, tt.dockerHostIntegration, portMappings)

			// Verify container name
			assert.Regexp(t, tt.expectedNamePattern, blueprint.Name)
			
			// Verify basic properties
			assert.Equal(t, "test-image:latest", blueprint.Image)
			assert.Equal(t, []string{"/bin/bash"}, blueprint.Command)
			assert.Equal(t, "/workspace", blueprint.WorkDir)
			assert.Equal(t, "claude", blueprint.User)
			assert.Equal(t, "bridge", blueprint.NetworkMode)
			
			// Verify port mappings
			assert.Equal(t, portMappings, blueprint.PortMappings)
			
			// Verify mounts count
			assert.Len(t, blueprint.Mounts, tt.expectedDockerMounts)
			
			// Verify environment count
			assert.Len(t, blueprint.Environment, tt.expectedEnvironment)
			
			// Verify Docker host integration environment
			if tt.dockerHostIntegration {
				assert.Contains(t, blueprint.Environment, "REACTOR_DOCKER_HOST_INTEGRATION=true")
				assert.Contains(t, blueprint.Mounts, "/var/run/docker.sock:/var/run/docker.sock:bind")
			} else {
				assert.NotContains(t, blueprint.Environment, "REACTOR_DOCKER_HOST_INTEGRATION=true")
				assert.NotContains(t, blueprint.Mounts, "/var/run/docker.sock:/var/run/docker.sock:bind")
			}
			
			// Verify mount format for non-discovery containers
			if !tt.isDiscovery {
				expectedMounts := []string{
					"/host/path/config:/container/config:bind",
					"/host/path/data:/container/data:bind",
				}
				for _, expectedMount := range expectedMounts {
					assert.Contains(t, blueprint.Mounts, expectedMount)
				}
			}
		})
	}
}

func TestContainerBlueprintToContainerSpec(t *testing.T) {
	portMappings := []PortMapping{
		{HostPort: 8080, ContainerPort: 80},
		{HostPort: 3000, ContainerPort: 3000},
	}

	blueprint := &ContainerBlueprint{
		Name:         "test-container",
		Image:        "test-image:latest",
		Command:      []string{"/bin/bash"},
		WorkDir:      "/workspace",
		User:         "claude",
		Environment:  []string{"ENV=test"},
		Mounts:       []string{"/host:/container:bind"},
		PortMappings: portMappings,
		NetworkMode:  "bridge",
	}

	spec := blueprint.ToContainerSpec()

	// Verify all fields are correctly mapped
	assert.Equal(t, blueprint.Name, spec.Name)
	assert.Equal(t, blueprint.Image, spec.Image)
	assert.Equal(t, blueprint.Command, spec.Command)
	assert.Equal(t, blueprint.WorkDir, spec.WorkDir)
	assert.Equal(t, blueprint.User, spec.User)
	assert.Equal(t, blueprint.Environment, spec.Environment)
	assert.Equal(t, blueprint.Mounts, spec.Mounts)
	assert.Equal(t, blueprint.NetworkMode, spec.NetworkMode)
	
	// Verify port mappings conversion
	require.Len(t, spec.PortMappings, 2)
	assert.Equal(t, 8080, spec.PortMappings[0].HostPort)
	assert.Equal(t, 80, spec.PortMappings[0].ContainerPort)
	assert.Equal(t, 3000, spec.PortMappings[1].HostPort)
	assert.Equal(t, 3000, spec.PortMappings[1].ContainerPort)
}

func TestContainerBlueprintValidation_EdgeCases(t *testing.T) {
	testutil.WithIsolatedHome(t)
	
	// Test with minimal config
	resolved := &config.ResolvedConfig{
		Account:     "",
		ProjectRoot: "",
		ProjectHash: "",
		Image:       "",
	}

	blueprint := NewContainerBlueprint(resolved, []MountSpec{}, false, false, []PortMapping{})

	// Should handle empty values gracefully
	assert.NotEmpty(t, blueprint.Name) // sanitizer should provide fallback
	assert.Equal(t, "", blueprint.Image)
	assert.Equal(t, []string{"/bin/bash"}, blueprint.Command)
	assert.Equal(t, "/workspace", blueprint.WorkDir)
	assert.Equal(t, "claude", blueprint.User)
	
	// Should convert to valid Docker spec
	spec := blueprint.ToContainerSpec()
	assert.NotNil(t, spec)
	assert.Equal(t, blueprint.Name, spec.Name)
}
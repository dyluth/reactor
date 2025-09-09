package core

import (
	"fmt"
	"os"
	"path/filepath"
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
		name            string
		account         string
		projectPath     string
		projectHash     string
		isolationPrefix string
		expectedPattern string
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
		name            string
		account         string
		projectPath     string
		projectHash     string
		isolationPrefix string
		expectedPattern string
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
		Account:          "testuser",
		ProjectRoot:      "/home/user/testproject",
		ProjectHash:      "testhash123",
		ProjectConfigDir: "/home/.reactor/testuser/testhash123",
		Image:            "test-image:latest",
	}

	// Note: Mount specifications are now constructed internally by NewContainerBlueprint

	// Test port mappings
	portMappings := []PortMapping{
		{HostPort: 8080, ContainerPort: 80},
		{HostPort: 3000, ContainerPort: 3000},
	}

	tests := []struct {
		name                  string
		isDiscovery           bool
		dockerHostIntegration bool
		isolationPrefix       string
		expectedNamePattern   string
		expectedDockerMounts  int
		expectedEnvironment   int
	}{
		{
			name:                  "regular container",
			isDiscovery:           false,
			dockerHostIntegration: false,
			expectedNamePattern:   "^reactor-testuser-testproject-testhash123$",
			expectedDockerMounts:  3, // workspace + 2 providers (claude, gemini)
			expectedEnvironment:   0, // no special env vars
		},
		{
			name:                  "discovery container (no mounts)",
			isDiscovery:           true,
			dockerHostIntegration: false,
			expectedNamePattern:   "^reactor-discovery-testuser-testproject-testhash123$",
			expectedDockerMounts:  0, // discovery mode has no mounts
			expectedEnvironment:   0, // no special env vars
		},
		{
			name:                  "regular container with Docker host integration",
			isDiscovery:           false,
			dockerHostIntegration: true,
			expectedNamePattern:   "^reactor-testuser-testproject-testhash123$",
			expectedDockerMounts:  4, // workspace + 2 providers + Docker socket
			expectedEnvironment:   1, // REACTOR_DOCKER_HOST_INTEGRATION=true
		},
		{
			name:                  "discovery with Docker host integration",
			isDiscovery:           true,
			dockerHostIntegration: true,
			expectedNamePattern:   "^reactor-discovery-testuser-testproject-testhash123$",
			expectedDockerMounts:  1, // only Docker socket mount
			expectedEnvironment:   1, // REACTOR_DOCKER_HOST_INTEGRATION=true
		},
		{
			name:                  "with isolation prefix",
			isDiscovery:           false,
			dockerHostIntegration: false,
			isolationPrefix:       "test-prefix",
			expectedNamePattern:   "^test-prefix-reactor-testuser-testproject-testhash123$",
			expectedDockerMounts:  3, // workspace + 2 providers (claude, gemini)
			expectedEnvironment:   0, // no special env vars
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

			blueprint := NewContainerBlueprint(resolved, tt.isDiscovery, tt.dockerHostIntegration, portMappings)

			// Verify container name
			assert.Regexp(t, tt.expectedNamePattern, blueprint.Name)

			// Verify basic properties
			assert.Equal(t, "test-image:latest", blueprint.Image)
			assert.Equal(t, []string{"/bin/sh"}, blueprint.Command)
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
				assert.Contains(t, blueprint.Mounts, "/var/run/docker.sock:/var/run/docker.sock")
			} else {
				assert.NotContains(t, blueprint.Environment, "REACTOR_DOCKER_HOST_INTEGRATION=true")
				assert.NotContains(t, blueprint.Mounts, "/var/run/docker.sock:/var/run/docker.sock")
			}

			// Verify mount format for non-discovery containers
			if !tt.isDiscovery {
				// Should have workspace mount
				assert.Contains(t, blueprint.Mounts, "/home/user/testproject:/workspace")
				// Should have provider credential mounts
				assert.Contains(t, blueprint.Mounts, "/home/.reactor/testuser/testhash123/claude:/home/claude/.claude")
				assert.Contains(t, blueprint.Mounts, "/home/.reactor/testuser/testhash123/gemini:/home/claude/.gemini")
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
		Command:      []string{"/bin/sh"},
		WorkDir:      "/workspace",
		User:         "claude",
		Environment:  []string{"ENV=test"},
		Mounts:       []string{"/host:/container"},
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

	blueprint := NewContainerBlueprint(resolved, false, false, []PortMapping{})

	// Should handle empty values gracefully
	assert.NotEmpty(t, blueprint.Name) // sanitizer should provide fallback
	assert.Equal(t, "", blueprint.Image)
	assert.Equal(t, []string{"/bin/sh"}, blueprint.Command)
	assert.Equal(t, "/workspace", blueprint.WorkDir)
	assert.Equal(t, "claude", blueprint.User)

	// Should convert to valid Docker spec
	spec := blueprint.ToContainerSpec()
	assert.NotNil(t, spec)
	assert.Equal(t, blueprint.Name, spec.Name)
}

func TestNewContainerBlueprint_RemoteUser(t *testing.T) {
	testutil.WithIsolatedHome(t)

	tests := []struct {
		name         string
		remoteUser   string
		expectedUser string
	}{
		{
			name:         "with remoteUser specified",
			remoteUser:   "myuser",
			expectedUser: "myuser",
		},
		{
			name:         "with root user",
			remoteUser:   "root",
			expectedUser: "root",
		},
		{
			name:         "empty remoteUser falls back to claude",
			remoteUser:   "",
			expectedUser: "claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test configuration with specific remoteUser
			resolved := &config.ResolvedConfig{
				Provider: config.ProviderInfo{
					Name:         "claude",
					DefaultImage: "test-image",
					Mounts:       []config.MountPoint{},
				},
				Account:          "testuser",
				Image:            "test-image",
				ProjectRoot:      "/test/project",
				ProjectHash:      "testhash123",
				AccountConfigDir: "/test/account",
				ProjectConfigDir: "/test/project/config",
				ForwardPorts:     []config.PortMapping{},
				RemoteUser:       tt.remoteUser,
				Danger:           false,
			}

			// Create blueprint
			blueprint := NewContainerBlueprint(resolved, false, false, []PortMapping{})

			// Verify user is set correctly
			assert.Equal(t, tt.expectedUser, blueprint.User)
		})
	}
}

func TestNewContainerBlueprint_ForwardPortsIntegration(t *testing.T) {
	testutil.WithIsolatedHome(t)

	// Test that the function accepts port mappings correctly
	resolved := &config.ResolvedConfig{
		Provider: config.ProviderInfo{
			Name:         "claude",
			DefaultImage: "test-image",
			Mounts:       []config.MountPoint{},
		},
		Account:          "testuser",
		Image:            "test-image",
		ProjectRoot:      "/test/project",
		ProjectHash:      "testhash123",
		AccountConfigDir: "/test/account",
		ProjectConfigDir: "/test/project/config",
		ForwardPorts:     []config.PortMapping{}, // Not used directly in blueprint construction
		RemoteUser:       "testuser",
		Danger:           false,
	}

	portMappings := []PortMapping{
		{HostPort: 8080, ContainerPort: 8080},
		{HostPort: 3000, ContainerPort: 4000},
	}

	blueprint := NewContainerBlueprint(resolved, false, false, portMappings)

	// Verify port mappings are preserved
	require.Len(t, blueprint.PortMappings, 2)
	assert.Equal(t, 8080, blueprint.PortMappings[0].HostPort)
	assert.Equal(t, 8080, blueprint.PortMappings[0].ContainerPort)
	assert.Equal(t, 3000, blueprint.PortMappings[1].HostPort)
	assert.Equal(t, 4000, blueprint.PortMappings[1].ContainerPort)

	// Verify other fields
	assert.Equal(t, "testuser", blueprint.User)
	assert.Equal(t, "test-image", blueprint.Image)
}

func TestNewContainerBlueprint_DefaultCommand(t *testing.T) {
	testutil.WithIsolatedHome(t)

	tests := []struct {
		name            string
		defaultCommand  string
		expectedCommand []string
	}{
		{
			name:            "with defaultCommand specified",
			defaultCommand:  "claude",
			expectedCommand: []string{"/bin/sh", "-c", "claude"},
		},
		{
			name:            "with custom shell command",
			defaultCommand:  "/bin/zsh",
			expectedCommand: []string{"/bin/sh", "-c", "/bin/zsh"},
		},
		{
			name:            "with complex command",
			defaultCommand:  "echo 'hello world'",
			expectedCommand: []string{"/bin/sh", "-c", "echo 'hello world'"},
		},
		{
			name:            "empty defaultCommand falls back to bash",
			defaultCommand:  "",
			expectedCommand: []string{"/bin/sh"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test configuration with specific defaultCommand
			resolved := &config.ResolvedConfig{
				Account:          "testuser",
				Image:            "test-image",
				ProjectRoot:      "/test/project",
				ProjectHash:      "testhash123",
				ProjectConfigDir: "/test/project/config",
				DefaultCommand:   tt.defaultCommand,
			}

			// Create blueprint
			blueprint := NewContainerBlueprint(resolved, false, false, []PortMapping{})

			// Verify command is set correctly
			assert.Equal(t, tt.expectedCommand, blueprint.Command)
		})
	}
}

func TestNewContainerBlueprint_MultiProviderMounts(t *testing.T) {
	testutil.WithIsolatedHome(t)

	resolved := &config.ResolvedConfig{
		Account:          "work-account",
		Image:            "test-image",
		ProjectRoot:      "/home/user/myproject",
		ProjectHash:      "abc123",
		ProjectConfigDir: "/home/.reactor/work-account/abc123",
	}

	blueprint := NewContainerBlueprint(resolved, false, false, []PortMapping{})

	// Verify that ALL providers get mounted
	expectedMounts := []string{
		// Workspace mount
		"/home/user/myproject:/workspace",
		// All builtin provider mounts
		"/home/.reactor/work-account/abc123/claude:/home/claude/.claude",
		"/home/.reactor/work-account/abc123/gemini:/home/claude/.gemini",
	}

	assert.Len(t, blueprint.Mounts, len(expectedMounts), "Should have mounts for workspace + all providers")

	for _, expectedMount := range expectedMounts {
		assert.Contains(t, blueprint.Mounts, expectedMount, "Should contain mount: %s", expectedMount)
	}
}

func TestNewContainerBlueprint_DiscoveryModeSkipsAllMounts(t *testing.T) {
	testutil.WithIsolatedHome(t)

	resolved := &config.ResolvedConfig{
		Account:          "testuser",
		Image:            "test-image",
		ProjectRoot:      "/home/user/myproject",
		ProjectHash:      "abc123",
		ProjectConfigDir: "/home/.reactor/testuser/abc123",
		DefaultCommand:   "claude",
	}

	blueprint := NewContainerBlueprint(resolved, true, false, []PortMapping{})

	// Discovery mode should have no mounts at all
	assert.Empty(t, blueprint.Mounts, "Discovery mode should have no mounts")

	// But should still respect other settings like defaultCommand
	assert.Equal(t, []string{"/bin/sh", "-c", "claude"}, blueprint.Command)
	assert.Contains(t, blueprint.Name, "discovery", "Discovery container should have discovery in name")
}

func TestNewContainerBlueprint_EdgeCaseCoverage(t *testing.T) {
	testutil.WithIsolatedHome(t)

	tests := []struct {
		name        string
		resolved    *config.ResolvedConfig
		isDiscovery bool
		dockerHost  bool
		ports       []PortMapping
		description string
	}{
		{
			name: "empty_project_config_dir",
			resolved: &config.ResolvedConfig{
				Account:          "testuser",
				Image:            "test-image",
				ProjectRoot:      "/project",
				ProjectHash:      "hash123",
				ProjectConfigDir: "", // Empty to test edge case
				DefaultCommand:   "",
			},
			isDiscovery: false,
			dockerHost:  false,
			ports:       []PortMapping{},
			description: "should handle empty ProjectConfigDir gracefully",
		},
		{
			name: "nil_resolved_config_fields",
			resolved: &config.ResolvedConfig{
				Account:          "",
				Image:            "",
				ProjectRoot:      "",
				ProjectHash:      "",
				ProjectConfigDir: "/test/config",
				DefaultCommand:   "",
			},
			isDiscovery: false,
			dockerHost:  false,
			ports:       []PortMapping{},
			description: "should handle empty string fields gracefully",
		},
		{
			name: "discovery_mode_with_docker_host_and_ports",
			resolved: &config.ResolvedConfig{
				Account:          "user",
				Image:            "alpine",
				ProjectRoot:      "/project",
				ProjectHash:      "hash",
				ProjectConfigDir: "/config",
				DefaultCommand:   "echo test",
			},
			isDiscovery: true,
			dockerHost:  true,
			ports:       []PortMapping{{HostPort: 8080, ContainerPort: 80}},
			description: "discovery mode with docker host should only mount docker socket",
		},
		{
			name: "all_features_enabled",
			resolved: &config.ResolvedConfig{
				Account:          "power-user",
				Image:            "custom:latest",
				ProjectRoot:      "/complex/project/path",
				ProjectHash:      "complex123",
				ProjectConfigDir: "/home/.reactor/power-user/complex123",
				DefaultCommand:   "/usr/local/bin/custom-shell --interactive",
			},
			isDiscovery: false,
			dockerHost:  true,
			ports: []PortMapping{
				{HostPort: 3000, ContainerPort: 3000},
				{HostPort: 8080, ContainerPort: 80},
			},
			description: "all features enabled should work together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprint := NewContainerBlueprint(tt.resolved, tt.isDiscovery, tt.dockerHost, tt.ports)

			// Verify basic structure is always valid
			assert.NotNil(t, blueprint, tt.description)
			assert.NotEmpty(t, blueprint.Name, "container name should never be empty")
			assert.Equal(t, tt.resolved.Image, blueprint.Image, "image should match resolved config")
			assert.Equal(t, tt.ports, blueprint.PortMappings, "port mappings should be preserved")
			assert.Equal(t, "bridge", blueprint.NetworkMode, "network mode should default to bridge")
			assert.Equal(t, "/workspace", blueprint.WorkDir, "work dir should default to /workspace")

			// Verify command logic
			if tt.resolved.DefaultCommand != "" {
				assert.Equal(t, []string{"/bin/sh", "-c", tt.resolved.DefaultCommand}, blueprint.Command, "should use custom default command")
			} else {
				assert.Equal(t, []string{"/bin/sh"}, blueprint.Command, "should fallback to sh")
			}

			// Verify user logic (should always have a fallback)
			if tt.resolved.RemoteUser != "" {
				assert.Equal(t, tt.resolved.RemoteUser, blueprint.User, "should use remote user when specified")
			} else {
				assert.Equal(t, "claude", blueprint.User, "should fallback to claude user")
			}

			// Verify mount logic based on mode
			if tt.isDiscovery {
				// Discovery mode: only docker socket if enabled
				if tt.dockerHost {
					assert.Len(t, blueprint.Mounts, 1, "discovery + docker host should have 1 mount")
					assert.Contains(t, blueprint.Mounts, "/var/run/docker.sock:/var/run/docker.sock")
				} else {
					assert.Empty(t, blueprint.Mounts, "discovery mode should have no mounts")
				}
			} else {
				// Regular mode: workspace + providers + optional docker socket
				expectedMountCount := 3 // workspace + claude + gemini
				if tt.dockerHost {
					expectedMountCount++ // + docker socket
				}
				assert.Len(t, blueprint.Mounts, expectedMountCount, "regular mode should have all expected mounts")

				// Should have workspace mount
				workspaceMount := fmt.Sprintf("%s:/workspace", tt.resolved.ProjectRoot)
				assert.Contains(t, blueprint.Mounts, workspaceMount, "should have workspace mount")

				// Should have provider mounts (if ProjectConfigDir is not empty)
				if tt.resolved.ProjectConfigDir != "" {
					claudeMount := fmt.Sprintf("%s/claude:/home/claude/.claude", tt.resolved.ProjectConfigDir)
					geminiMount := fmt.Sprintf("%s/gemini:/home/claude/.gemini", tt.resolved.ProjectConfigDir)
					assert.Contains(t, blueprint.Mounts, claudeMount, "should have claude mount")
					assert.Contains(t, blueprint.Mounts, geminiMount, "should have gemini mount")
				}

				// Docker socket mount if enabled
				if tt.dockerHost {
					assert.Contains(t, blueprint.Mounts, "/var/run/docker.sock:/var/run/docker.sock")
				}
			}

			// Verify environment variables
			if tt.dockerHost {
				assert.Contains(t, blueprint.Environment, "REACTOR_DOCKER_HOST_INTEGRATION=true")
			} else {
				assert.NotContains(t, blueprint.Environment, "REACTOR_DOCKER_HOST_INTEGRATION=true")
			}
		})
	}
}

func TestNewContainerBlueprint_ProviderIterationComplete(t *testing.T) {
	testutil.WithIsolatedHome(t)

	// Test that ALL providers from BuiltinProviders are mounted
	resolved := &config.ResolvedConfig{
		Account:          "test-account",
		Image:            "test-image",
		ProjectRoot:      "/test/project",
		ProjectHash:      "test123",
		ProjectConfigDir: "/test/.reactor/test-account/test123",
	}

	blueprint := NewContainerBlueprint(resolved, false, false, []PortMapping{})

	// Count expected mounts: workspace + all builtin providers
	expectedProviderMounts := len(config.BuiltinProviders)
	expectedTotalMounts := 1 + expectedProviderMounts // workspace + providers

	assert.Len(t, blueprint.Mounts, expectedTotalMounts,
		"Should mount workspace + all %d builtin providers", expectedProviderMounts)

	// Verify each provider gets mounted
	for providerName, providerInfo := range config.BuiltinProviders {
		for _, mountPoint := range providerInfo.Mounts {
			expectedMount := fmt.Sprintf("%s/%s:%s",
				resolved.ProjectConfigDir, mountPoint.Source, mountPoint.Target)
			assert.Contains(t, blueprint.Mounts, expectedMount,
				"Should mount provider %s at %s", providerName, expectedMount)
		}
	}
}

func TestNewContainerBlueprint_NestedMountPointIteration(t *testing.T) {
	testutil.WithIsolatedHome(t)

	// This test ensures we handle providers with multiple mount points correctly
	// Even though current providers only have 1 mount each, the code should handle multiple

	resolved := &config.ResolvedConfig{
		Account:          "multi-mount-test",
		Image:            "test-image",
		ProjectRoot:      "/test",
		ProjectHash:      "multi123",
		ProjectConfigDir: "/test/.reactor/multi-mount-test/multi123",
	}

	blueprint := NewContainerBlueprint(resolved, false, false, []PortMapping{})

	// Calculate expected mounts by iterating the same way the implementation does
	expectedMounts := []string{
		"/test:/workspace", // workspace mount
	}

	// Add all provider mounts
	for _, provider := range config.BuiltinProviders {
		for _, mount := range provider.Mounts {
			hostPath := filepath.Join(resolved.ProjectConfigDir, mount.Source)
			expectedMount := fmt.Sprintf("%s:%s", hostPath, mount.Target)
			expectedMounts = append(expectedMounts, expectedMount)
		}
	}

	assert.Len(t, blueprint.Mounts, len(expectedMounts), "Should have exact expected mount count")

	for _, expectedMount := range expectedMounts {
		assert.Contains(t, blueprint.Mounts, expectedMount, "Should contain mount: %s", expectedMount)
	}
}

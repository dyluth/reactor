package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dyluth/reactor/pkg/config"
	"github.com/dyluth/reactor/pkg/docker"
)

func TestMergePortMappings(t *testing.T) {
	tests := []struct {
		name              string
		devcontainerPorts []config.PortMapping
		cliPorts          []PortMapping
		expected          []PortMapping
	}{
		{
			name:              "empty inputs",
			devcontainerPorts: []config.PortMapping{},
			cliPorts:          []PortMapping{},
			expected:          []PortMapping{},
		},
		{
			name: "only devcontainer ports",
			devcontainerPorts: []config.PortMapping{
				{HostPort: 8080, ContainerPort: 8080},
				{HostPort: 3000, ContainerPort: 3000},
			},
			cliPorts: []PortMapping{},
			expected: []PortMapping{
				{HostPort: 8080, ContainerPort: 8080},
				{HostPort: 3000, ContainerPort: 3000},
			},
		},
		{
			name:              "only CLI ports",
			devcontainerPorts: []config.PortMapping{},
			cliPorts: []PortMapping{
				{HostPort: 9000, ContainerPort: 4000},
				{HostPort: 5000, ContainerPort: 5000},
			},
			expected: []PortMapping{
				{HostPort: 9000, ContainerPort: 4000},
				{HostPort: 5000, ContainerPort: 5000},
			},
		},
		{
			name: "no conflicts - simple merge",
			devcontainerPorts: []config.PortMapping{
				{HostPort: 8080, ContainerPort: 8080},
				{HostPort: 3000, ContainerPort: 3000},
			},
			cliPorts: []PortMapping{
				{HostPort: 9000, ContainerPort: 4000},
				{HostPort: 5000, ContainerPort: 5000},
			},
			expected: []PortMapping{
				{HostPort: 8080, ContainerPort: 8080},
				{HostPort: 3000, ContainerPort: 3000},
				{HostPort: 9000, ContainerPort: 4000},
				{HostPort: 5000, ContainerPort: 5000},
			},
		},
		{
			name: "CLI overrides devcontainer port - single conflict",
			devcontainerPorts: []config.PortMapping{
				{HostPort: 8080, ContainerPort: 8000},
				{HostPort: 3000, ContainerPort: 3000},
			},
			cliPorts: []PortMapping{
				{HostPort: 8080, ContainerPort: 3000}, // Override 8080:8000 with 8080:3000
			},
			expected: []PortMapping{
				{HostPort: 8080, ContainerPort: 3000}, // CLI port wins
				{HostPort: 3000, ContainerPort: 3000}, // Unchanged
			},
		},
		{
			name: "CLI overrides multiple devcontainer ports",
			devcontainerPorts: []config.PortMapping{
				{HostPort: 8080, ContainerPort: 8000},
				{HostPort: 3000, ContainerPort: 3000},
				{HostPort: 9000, ContainerPort: 9000},
			},
			cliPorts: []PortMapping{
				{HostPort: 8080, ContainerPort: 3000}, // Override first
				{HostPort: 9000, ContainerPort: 4000}, // Override third
				{HostPort: 5000, ContainerPort: 5000}, // Add new
			},
			expected: []PortMapping{
				{HostPort: 8080, ContainerPort: 3000}, // CLI override
				{HostPort: 3000, ContainerPort: 3000}, // Unchanged
				{HostPort: 9000, ContainerPort: 4000}, // CLI override
				{HostPort: 5000, ContainerPort: 5000}, // CLI addition
			},
		},
		{
			name: "complex scenario - multiple conflicts and additions",
			devcontainerPorts: []config.PortMapping{
				{HostPort: 8080, ContainerPort: 8080},
				{HostPort: 3000, ContainerPort: 3000},
				{HostPort: 4000, ContainerPort: 4000},
			},
			cliPorts: []PortMapping{
				{HostPort: 8080, ContainerPort: 3000}, // Override existing
				{HostPort: 5000, ContainerPort: 5000}, // Add new
				{HostPort: 3000, ContainerPort: 8000}, // Override existing
				{HostPort: 6000, ContainerPort: 6000}, // Add new
			},
			expected: []PortMapping{
				{HostPort: 8080, ContainerPort: 3000}, // CLI override
				{HostPort: 3000, ContainerPort: 8000}, // CLI override
				{HostPort: 4000, ContainerPort: 4000}, // Unchanged
				{HostPort: 5000, ContainerPort: 5000}, // CLI addition
				{HostPort: 6000, ContainerPort: 6000}, // CLI addition
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergePortMappings(tt.devcontainerPorts, tt.cliPorts)

			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d port mappings, got %d", len(tt.expected), len(result))
			}

			// Create maps for easier comparison since order might not be preserved exactly
			resultMap := make(map[int]int)
			expectedMap := make(map[int]int)

			for _, pm := range result {
				resultMap[pm.HostPort] = pm.ContainerPort
			}
			for _, pm := range tt.expected {
				expectedMap[pm.HostPort] = pm.ContainerPort
			}

			if len(resultMap) != len(expectedMap) {
				t.Fatalf("Maps have different sizes: result=%d, expected=%d", len(resultMap), len(expectedMap))
			}

			for hostPort, expectedContainerPort := range expectedMap {
				if resultContainerPort, exists := resultMap[hostPort]; !exists {
					t.Errorf("Missing host port %d in result", hostPort)
				} else if resultContainerPort != expectedContainerPort {
					t.Errorf("Host port %d: expected container port %d, got %d",
						hostPort, expectedContainerPort, resultContainerPort)
				}
			}
		})
	}
}

func TestCreateBuildSpecFromConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "reactor-build-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create .devcontainer directory and devcontainer.json
	devcontainerDir := filepath.Join(tempDir, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		t.Fatalf("Failed to create .devcontainer directory: %v", err)
	}

	devcontainerFile := filepath.Join(devcontainerDir, "devcontainer.json")
	devcontainerContent := `{
		"name": "test",
		"build": {
			"dockerfile": "Dockerfile",
			"context": ".."
		}
	}`
	if err := os.WriteFile(devcontainerFile, []byte(devcontainerContent), 0644); err != nil {
		t.Fatalf("Failed to write devcontainer.json: %v", err)
	}

	tests := []struct {
		name     string
		config   *config.ResolvedConfig
		expected docker.BuildSpec
		wantErr  bool
	}{
		{
			name: "basic build config",
			config: &config.ResolvedConfig{
				ProjectRoot: tempDir,
				ProjectHash: "abc12345",
				Build: &config.Build{
					Dockerfile: "Dockerfile",
					Context:    "..",
				},
			},
			expected: docker.BuildSpec{
				Dockerfile: "Dockerfile",
				Context:    tempDir, // Context ".." resolves to project root
				ImageName:  "reactor-build:abc12345",
			},
			wantErr: false,
		},
		{
			name: "build config with empty dockerfile (should default)",
			config: &config.ResolvedConfig{
				ProjectRoot: tempDir,
				ProjectHash: "def67890",
				Build: &config.Build{
					Dockerfile: "",
					Context:    ".",
				},
			},
			expected: docker.BuildSpec{
				Dockerfile: "Dockerfile", // Should default to "Dockerfile"
				Context:    devcontainerDir,
				ImageName:  "reactor-build:def67890",
			},
			wantErr: false,
		},
		{
			name: "build config with empty context (should default)",
			config: &config.ResolvedConfig{
				ProjectRoot: tempDir,
				ProjectHash: "ghi24680",
				Build: &config.Build{
					Dockerfile: "Dockerfile.dev",
					Context:    "",
				},
			},
			expected: docker.BuildSpec{
				Dockerfile: "Dockerfile.dev",
				Context:    devcontainerDir, // Should default to devcontainer directory
				ImageName:  "reactor-build:ghi24680",
			},
			wantErr: false,
		},
		{
			name: "nil build config",
			config: &config.ResolvedConfig{
				ProjectRoot: tempDir,
				ProjectHash: "nil12345",
				Build:       nil,
			},
			expected: docker.BuildSpec{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := createBuildSpecFromConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

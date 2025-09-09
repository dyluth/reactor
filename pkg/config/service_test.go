package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewService(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWd) }()

	tempDir := t.TempDir()
	_ = os.Chdir(tempDir)

	service := NewService()

	// Check that project root is set to some form of tempDir (may have symlink resolution)
	if !strings.Contains(service.projectRoot, "TestNewService") {
		t.Errorf("Expected project root to contain test dir, got %s", service.projectRoot)
	}
}

func TestService_InitializeProject(t *testing.T) {
	tempDir := t.TempDir()

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWd) }()

	_ = os.Chdir(tempDir)
	service := NewService()

	t.Run("successful initialization", func(t *testing.T) {
		err := service.InitializeProject()
		if err != nil {
			t.Fatalf("InitializeProject failed: %v", err)
		}

		// Check that devcontainer.json was created
		configPath := filepath.Join(tempDir, ".devcontainer", "devcontainer.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Expected devcontainer.json to be created at %s", configPath)
		}

		// Check that the created config can be loaded
		_, err = LoadDevContainerConfig(configPath)
		if err != nil {
			t.Errorf("Failed to load created devcontainer.json: %v", err)
		}
	})

	t.Run("project already initialized", func(t *testing.T) {
		// Try to initialize again - should fail
		err := service.InitializeProject()
		if err == nil {
			t.Error("Expected error when initializing already initialized project")
		}
		if !strings.Contains(err.Error(), "already initialized") {
			t.Errorf("Expected 'already initialized' error, got: %v", err)
		}
	})
}

func TestService_ShowConfiguration(t *testing.T) {
	tempDir := t.TempDir()

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWd) }()

	_ = os.Chdir(tempDir)
	service := NewService()

	t.Run("show configuration without devcontainer.json", func(t *testing.T) {
		err := service.ShowConfiguration()
		if err == nil {
			t.Error("Expected error when no devcontainer.json exists")
		}
	})

	t.Run("show configuration with devcontainer.json", func(t *testing.T) {
		// First initialize the project
		err := service.InitializeProject()
		if err != nil {
			t.Fatalf("Failed to initialize project: %v", err)
		}

		// Now show configuration should work
		err = service.ShowConfiguration()
		if err != nil {
			t.Errorf("ShowConfiguration failed: %v", err)
		}
	})
}

func TestService_ListAccounts(t *testing.T) {
	service := NewService()

	t.Run("list accounts without error", func(t *testing.T) {
		// This should not error even if no accounts exist
		err := service.ListAccounts()
		if err != nil {
			t.Errorf("ListAccounts failed: %v", err)
		}
	})
}

func TestParseForwardPorts(t *testing.T) {
	tests := []struct {
		name          string
		input         []interface{}
		expected      []PortMapping
		expectError   bool
		errorContains string
	}{
		{
			name:     "empty array",
			input:    []interface{}{},
			expected: []PortMapping{},
		},
		{
			name:  "single int port",
			input: []interface{}{8080.0}, // JSON numbers are float64 in Go
			expected: []PortMapping{
				{HostPort: 8080, ContainerPort: 8080},
			},
		},
		{
			name:  "multiple int ports",
			input: []interface{}{8080.0, 3000.0, 9000.0},
			expected: []PortMapping{
				{HostPort: 8080, ContainerPort: 8080},
				{HostPort: 3000, ContainerPort: 3000},
				{HostPort: 9000, ContainerPort: 9000},
			},
		},
		{
			name:  "single string port mapping",
			input: []interface{}{"8080:3000"},
			expected: []PortMapping{
				{HostPort: 8080, ContainerPort: 3000},
			},
		},
		{
			name:  "multiple string port mappings",
			input: []interface{}{"8080:3000", "9000:4000"},
			expected: []PortMapping{
				{HostPort: 8080, ContainerPort: 3000},
				{HostPort: 9000, ContainerPort: 4000},
			},
		},
		{
			name:  "mixed int and string ports",
			input: []interface{}{8080.0, "9000:4000", 5000.0},
			expected: []PortMapping{
				{HostPort: 8080, ContainerPort: 8080},
				{HostPort: 9000, ContainerPort: 4000},
				{HostPort: 5000, ContainerPort: 5000},
			},
		},
		{
			name:          "invalid type - boolean",
			input:         []interface{}{true},
			expectError:   true,
			errorContains: "invalid type bool",
		},
		{
			name:          "invalid type - object",
			input:         []interface{}{map[string]interface{}{"port": 8080}},
			expectError:   true,
			errorContains: "invalid type",
		},
		{
			name:          "invalid string format - no colon",
			input:         []interface{}{"8080"},
			expectError:   true,
			errorContains: "invalid string format '8080', expected 'host:container'",
		},
		{
			name:          "invalid string format - too many parts",
			input:         []interface{}{"8080:3000:extra"},
			expectError:   true,
			errorContains: "invalid string format '8080:3000:extra', expected 'host:container'",
		},
		{
			name:          "invalid host port - not a number",
			input:         []interface{}{"abc:3000"},
			expectError:   true,
			errorContains: "invalid host port 'abc', must be a number",
		},
		{
			name:          "invalid container port - not a number",
			input:         []interface{}{"8080:def"},
			expectError:   true,
			errorContains: "invalid container port 'def', must be a number",
		},
		{
			name:          "port out of range - int too low",
			input:         []interface{}{0.0},
			expectError:   true,
			errorContains: "port 0 is out of valid range (1-65535)",
		},
		{
			name:          "port out of range - int too high",
			input:         []interface{}{70000.0},
			expectError:   true,
			errorContains: "port 70000 is out of valid range (1-65535)",
		},
		{
			name:          "host port out of range - string too low",
			input:         []interface{}{"0:8080"},
			expectError:   true,
			errorContains: "host port 0 is out of valid range (1-65535)",
		},
		{
			name:          "container port out of range - string too high",
			input:         []interface{}{"8080:70000"},
			expectError:   true,
			errorContains: "container port 70000 is out of valid range (1-65535)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseForwardPorts(tt.input)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %s", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d port mappings, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				if result[i].HostPort != expected.HostPort || result[i].ContainerPort != expected.ContainerPort {
					t.Errorf("Port mapping %d: expected %d:%d, got %d:%d",
						i, expected.HostPort, expected.ContainerPort,
						result[i].HostPort, result[i].ContainerPort)
				}
			}
		})
	}
}

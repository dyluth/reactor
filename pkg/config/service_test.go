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
package config

import (
	"fmt"
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

	expectedConfigPath := GetProjectConfigPath()
	if service.configPath != expectedConfigPath {
		t.Errorf("Expected config path %s, got %s", expectedConfigPath, service.configPath)
	}
}

func TestService_InitializeProject(t *testing.T) {
	// Create temp directory and change to it
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()
	_ = os.Chdir(tempDir)

	service := NewService()

	t.Run("successful initialization", func(t *testing.T) {
		err := service.InitializeProject()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Verify config file was created
		configPath := GetProjectConfigPath()
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file should have been created")
		}

		// Verify config is valid
		config, err := LoadProjectConfig(configPath)
		if err != nil {
			t.Errorf("Should be able to load created config: %v", err)
		}

		if config.Provider == "" {
			t.Error("Provider should be set")
		}
		if config.Account == "" {
			t.Error("Account should be set")
		}
		if config.Image == "" {
			t.Error("Image should be set")
		}
	})

	t.Run("project already initialized", func(t *testing.T) {
		// Try to initialize again
		err := service.InitializeProject()
		if err == nil {
			t.Error("Expected error when trying to initialize twice")
		}

		if !strings.Contains(err.Error(), "already initialized") {
			t.Errorf("Expected 'already initialized' error, got: %v", err)
		}
	})
}

func TestService_GetConfigValue(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()
	_ = os.Chdir(tempDir)

	// Create test config
	configPath := GetProjectConfigPath()
	testConfig := &ProjectConfig{
		Provider: "claude",
		Account:  "testuser",
		Image:    "python",
		Danger:   true,
	}
	_ = SaveProjectConfig(testConfig, configPath)

	service := NewService()

	testCases := []struct {
		key           string
		expectedValue interface{}
		expectError   bool
	}{
		{"provider", "claude", false},
		{"account", "testuser", false},
		{"image", "python", false},
		{"danger", true, false},
		{"invalid_key", nil, true},
		{"", nil, true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("get_%s", tc.key), func(t *testing.T) {
			value, err := service.GetConfigValue(tc.key)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for key '%s'", tc.key)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for key '%s', got: %v", tc.key, err)
				}
				if value != tc.expectedValue {
					t.Errorf("Expected value %v for key '%s', got %v", tc.expectedValue, tc.key, value)
				}
			}
		})
	}
}

func TestService_SetConfigValue(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()
	_ = os.Chdir(tempDir)

	// Create initial config
	service := NewService()
	_ = service.InitializeProject()

	t.Run("set valid provider", func(t *testing.T) {
		err := service.SetConfigValue("provider", "gemini")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify the change
		value, _ := service.GetConfigValue("provider")
		if value != "gemini" {
			t.Errorf("Expected provider 'gemini', got '%v'", value)
		}
	})

	t.Run("set valid account", func(t *testing.T) {
		err := service.SetConfigValue("account", "newuser")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		value, _ := service.GetConfigValue("account")
		if value != "newuser" {
			t.Errorf("Expected account 'newuser', got '%v'", value)
		}
	})

	t.Run("set valid image", func(t *testing.T) {
		err := service.SetConfigValue("image", "python")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		value, _ := service.GetConfigValue("image")
		if value != "python" {
			t.Errorf("Expected image 'python', got '%v'", value)
		}
	})

	t.Run("set danger to true", func(t *testing.T) {
		err := service.SetConfigValue("danger", "true")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		value, _ := service.GetConfigValue("danger")
		if value != true {
			t.Errorf("Expected danger true, got %v", value)
		}
	})

	t.Run("set danger with various boolean values", func(t *testing.T) {
		boolTests := []struct {
			input    string
			expected bool
		}{
			{"true", true},
			{"1", true},
			{"yes", true},
			{"on", true},
			{"false", false},
			{"0", false},
			{"no", false},
			{"off", false},
		}

		for _, bt := range boolTests {
			err := service.SetConfigValue("danger", bt.input)
			if err != nil {
				t.Errorf("Expected no error for '%s', got: %v", bt.input, err)
			}

			value, _ := service.GetConfigValue("danger")
			if value != bt.expected {
				t.Errorf("Expected danger %v for input '%s', got %v", bt.expected, bt.input, value)
			}
		}
	})

	t.Run("set invalid boolean", func(t *testing.T) {
		err := service.SetConfigValue("danger", "invalid")
		if err == nil {
			t.Error("Expected error for invalid boolean")
		}

		if !strings.Contains(err.Error(), "invalid boolean value") {
			t.Errorf("Expected boolean error, got: %v", err)
		}
	})

	t.Run("set invalid provider", func(t *testing.T) {
		err := service.SetConfigValue("provider", "")
		if err == nil {
			t.Error("Expected error for empty provider")
		}
	})

	t.Run("set invalid account", func(t *testing.T) {
		err := service.SetConfigValue("account", "../malicious")
		if err == nil {
			t.Error("Expected error for malicious account")
		}
	})

	t.Run("set invalid image", func(t *testing.T) {
		err := service.SetConfigValue("image", "")
		if err == nil {
			t.Error("Expected error for empty image")
		}
	})

	t.Run("set unknown key", func(t *testing.T) {
		err := service.SetConfigValue("unknown", "value")
		if err == nil {
			t.Error("Expected error for unknown key")
		}

		if !strings.Contains(err.Error(), "unknown configuration key") {
			t.Errorf("Expected unknown key error, got: %v", err)
		}
	})
}

func TestService_LoadConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()
	_ = os.Chdir(tempDir)

	service := NewService()
	_ = service.InitializeProject()

	t.Run("load configuration with no overrides", func(t *testing.T) {
		resolved, err := service.LoadConfiguration("", "", "", false)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if resolved.Provider.Name == "" {
			t.Error("Provider should be resolved")
		}
		if resolved.Account == "" {
			t.Error("Account should be set")
		}
		if resolved.Image == "" {
			t.Error("Image should be resolved")
		}
		if resolved.ProjectRoot == "" {
			t.Error("Project root should be set")
		}
		if resolved.ProjectHash == "" {
			t.Error("Project hash should be generated")
		}
	})

	t.Run("load configuration with CLI overrides", func(t *testing.T) {
		resolved, err := service.LoadConfiguration("gemini", "cliuser", "python", true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if resolved.Provider.Name != "gemini" {
			t.Errorf("Expected provider 'gemini', got '%s'", resolved.Provider.Name)
		}
		if resolved.Account != "cliuser" {
			t.Errorf("Expected account 'cliuser', got '%s'", resolved.Account)
		}
		if !strings.Contains(resolved.Image, "python") {
			t.Errorf("Expected python image, got '%s'", resolved.Image)
		}
		if !resolved.Danger {
			t.Error("Expected danger to be true")
		}

		// Verify CLI overrides were persisted to config file
		config, _ := LoadProjectConfig(service.configPath)
		if config.Provider != "gemini" {
			t.Error("CLI provider override should have been saved to config")
		}
	})

	t.Run("load configuration with invalid provider", func(t *testing.T) {
		// First set an invalid provider in the config
		_ = service.SetConfigValue("provider", "claude") // Reset to valid

		// Now try with invalid CLI override (this should fail at resolution, not validation)
		_, err := service.LoadConfiguration("nonexistent", "", "", false)
		if err == nil {
			t.Error("Expected error for nonexistent provider")
		}

		if !strings.Contains(err.Error(), "unknown provider") {
			t.Errorf("Expected unknown provider error, got: %v", err)
		}
	})
}

func TestService_ShowConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()
	_ = os.Chdir(tempDir)

	service := NewService()
	_ = service.InitializeProject()

	t.Run("show configuration without error", func(t *testing.T) {
		// We can't easily test output, but we can ensure it doesn't error
		err := service.ShowConfiguration()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}

func TestService_ListAccounts(t *testing.T) {
	service := NewService()

	t.Run("list accounts without error", func(t *testing.T) {
		// We can't easily mock the filesystem structure,
		// but we can ensure it doesn't panic or error unexpectedly
		err := service.ListAccounts()
		// Error is OK if ~/.reactor doesn't exist
		if err != nil && !strings.Contains(err.Error(), "failed to read") {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestService_createProjectDirectories(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()
	_ = os.Chdir(tempDir)

	// Set up isolated reactor home for testing
	_ = os.Setenv("REACTOR_ISOLATION_PREFIX", "test-service")
	defer func() { _ = os.Unsetenv("REACTOR_ISOLATION_PREFIX") }()

	service := NewService()

	config := &ProjectConfig{
		Provider: "claude",
		Account:  "testuser",
		Image:    "base",
		Danger:   false,
	}

	t.Run("create directories successfully", func(t *testing.T) {
		err := service.createProjectDirectories(config)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify directories were created
		reactorHome, _ := GetReactorHomeDir()
		projectHash := GenerateProjectHash(service.projectRoot)
		expectedDir := filepath.Join(reactorHome, config.Account, projectHash, "claude")

		if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
			t.Errorf("Directory should have been created: %s", expectedDir)
		}
	})
}

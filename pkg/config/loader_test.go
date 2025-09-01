package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectConfig(t *testing.T) {
	// Create temp directory for test files
	tempDir := t.TempDir()

	t.Run("valid config loads successfully", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "valid.conf")
		content := `provider: claude
account: testuser
image: base
danger: false`

		err := os.WriteFile(configPath, []byte(content), 0600)
		if err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		config, err := LoadProjectConfig(configPath)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if config.Provider != "claude" {
			t.Errorf("Expected provider 'claude', got '%s'", config.Provider)
		}
		if config.Account != "testuser" {
			t.Errorf("Expected account 'testuser', got '%s'", config.Account)
		}
		if config.Image != "base" {
			t.Errorf("Expected image 'base', got '%s'", config.Image)
		}
		if config.Danger != false {
			t.Errorf("Expected danger false, got %v", config.Danger)
		}
	})

	t.Run("missing config file", func(t *testing.T) {
		nonexistentPath := filepath.Join(tempDir, "nonexistent.conf")

		_, err := LoadProjectConfig(nonexistentPath)
		if err == nil {
			t.Error("Expected error for missing config file")
		}

		if !strings.Contains(err.Error(), "no project configuration found") {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("malformed YAML", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "malformed.conf")
		content := `provider: claude
account: testuser
image: base
danger: false
invalid_yaml: [unclosed bracket`

		err := os.WriteFile(configPath, []byte(content), 0600)
		if err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err = LoadProjectConfig(configPath)
		if err == nil {
			t.Error("Expected error for malformed YAML")
		}

		if !strings.Contains(err.Error(), "failed to parse YAML") {
			t.Errorf("Expected YAML parse error, got: %v", err)
		}
	})

	t.Run("invalid provider in config", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "invalid_provider.conf")
		content := `provider: ""
account: testuser
image: base`

		err := os.WriteFile(configPath, []byte(content), 0600)
		if err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err = LoadProjectConfig(configPath)
		if err == nil {
			t.Error("Expected error for empty provider")
		}

		if !strings.Contains(err.Error(), "invalid configuration") {
			t.Errorf("Expected validation error, got: %v", err)
		}
	})

	t.Run("invalid account in config", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "invalid_account.conf")
		content := `provider: claude
account: "../malicious"
image: base`

		err := os.WriteFile(configPath, []byte(content), 0600)
		if err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err = LoadProjectConfig(configPath)
		if err == nil {
			t.Error("Expected error for malicious account path")
		}

		if !strings.Contains(err.Error(), "invalid configuration") {
			t.Errorf("Expected validation error, got: %v", err)
		}
	})

	t.Run("unreadable config file", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "unreadable.conf")
		content := `provider: claude
account: testuser
image: base`

		err := os.WriteFile(configPath, []byte(content), 0000) // No read permissions
		if err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		// Clean up permissions after test
		defer func() { _ = os.Chmod(configPath, 0600) }()

		_, err = LoadProjectConfig(configPath)
		if err == nil {
			t.Error("Expected error for unreadable file")
		}

		if !strings.Contains(err.Error(), "failed to read config file") {
			t.Errorf("Expected read error, got: %v", err)
		}
	})
}

func TestSaveProjectConfig(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("save valid config", func(t *testing.T) {
		config := &ProjectConfig{
			Provider: "claude",
			Account:  "testuser",
			Image:    "python",
			Danger:   true,
		}

		configPath := filepath.Join(tempDir, "save_test.conf")
		err := SaveProjectConfig(config, configPath)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Verify file exists and has correct permissions
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("Config file should exist: %v", err)
		}

		expectedMode := os.FileMode(0600)
		if info.Mode().Perm() != expectedMode {
			t.Errorf("Expected permissions %v, got %v", expectedMode, info.Mode().Perm())
		}

		// Verify content can be loaded back
		loadedConfig, err := LoadProjectConfig(configPath)
		if err != nil {
			t.Fatalf("Should be able to load saved config: %v", err)
		}

		if loadedConfig.Provider != config.Provider {
			t.Errorf("Provider mismatch: expected %s, got %s", config.Provider, loadedConfig.Provider)
		}
	})

	t.Run("save nil config", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "nil_test.conf")
		err := SaveProjectConfig(nil, configPath)
		if err == nil {
			t.Error("Expected error for nil config")
		}

		if !strings.Contains(err.Error(), "invalid configuration") {
			t.Errorf("Expected validation error, got: %v", err)
		}
	})

	t.Run("save to invalid path", func(t *testing.T) {
		config := &ProjectConfig{
			Provider: "claude",
			Account:  "testuser",
			Image:    "base",
		}

		// Try to write to a directory that doesn't exist
		invalidPath := "/nonexistent/directory/config.conf"
		err := SaveProjectConfig(config, invalidPath)
		if err == nil {
			t.Error("Expected error for invalid path")
		}

		if !strings.Contains(err.Error(), "failed to write config file") {
			t.Errorf("Expected write error, got: %v", err)
		}
	})
}

func TestValidateProjectConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		err := ValidateProjectConfig(nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}

		if !strings.Contains(err.Error(), "config cannot be nil") {
			t.Errorf("Expected nil error, got: %v", err)
		}
	})

	t.Run("valid config", func(t *testing.T) {
		config := &ProjectConfig{
			Provider: "claude",
			Account:  "testuser",
			Image:    "base",
			Danger:   false,
		}

		err := ValidateProjectConfig(config)
		if err != nil {
			t.Errorf("Expected no error for valid config, got: %v", err)
		}
	})

	t.Run("invalid provider", func(t *testing.T) {
		config := &ProjectConfig{
			Provider: "",
			Account:  "testuser",
			Image:    "base",
		}

		err := ValidateProjectConfig(config)
		if err == nil {
			t.Error("Expected error for empty provider")
		}

		if !strings.Contains(err.Error(), "invalid provider") {
			t.Errorf("Expected provider error, got: %v", err)
		}
	})

	t.Run("invalid account", func(t *testing.T) {
		config := &ProjectConfig{
			Provider: "claude",
			Account:  "/..",
			Image:    "base",
		}

		err := ValidateProjectConfig(config)
		if err == nil {
			t.Error("Expected error for malicious account")
		}

		if !strings.Contains(err.Error(), "invalid account") {
			t.Errorf("Expected account error, got: %v", err)
		}
	})

	t.Run("invalid image", func(t *testing.T) {
		config := &ProjectConfig{
			Provider: "claude",
			Account:  "testuser",
			Image:    "",
		}

		err := ValidateProjectConfig(config)
		if err == nil {
			t.Error("Expected error for empty image")
		}

		if !strings.Contains(err.Error(), "invalid image") {
			t.Errorf("Expected image error, got: %v", err)
		}
	})
}

func TestCreateDefaultProjectConfig(t *testing.T) {
	t.Run("creates valid defaults", func(t *testing.T) {
		config, err := CreateDefaultProjectConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if config.Provider != "claude" {
			t.Errorf("Expected default provider 'claude', got '%s'", config.Provider)
		}

		if config.Image != "base" {
			t.Errorf("Expected default image 'base', got '%s'", config.Image)
		}

		if config.Danger != false {
			t.Errorf("Expected default danger false, got %v", config.Danger)
		}

		// Account should be set to system username (non-empty)
		if config.Account == "" {
			t.Error("Expected account to be set to system username")
		}

		// Validate the generated config
		err = ValidateProjectConfig(config)
		if err != nil {
			t.Errorf("Default config should be valid: %v", err)
		}
	})
}

func TestCheckDependencies(t *testing.T) {
	t.Run("dependency check runs without panic", func(t *testing.T) {
		// We can't reliably test the actual docker/git detection
		// but we can ensure the function doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("CheckDependencies panicked: %v", r)
			}
		}()

		// This may or may not return an error depending on system
		_ = CheckDependencies()
	})
}

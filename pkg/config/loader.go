package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadProjectConfig loads and parses the .reactor.conf file
func LoadProjectConfig(configPath string) (*ProjectConfig, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no project configuration found at %s. Run 'reactor config init' to create one", configPath)
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Parse YAML
	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", configPath, err)
	}

	// Validate the loaded configuration
	if err := ValidateProjectConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration in %s: %w", configPath, err)
	}

	return &config, nil
}

// SaveProjectConfig saves a ProjectConfig to the specified path
func SaveProjectConfig(config *ProjectConfig, configPath string) error {
	// Validate before saving
	if err := ValidateProjectConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write to file with restrictive permissions
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", configPath, err)
	}

	return nil
}

// ValidateProjectConfig validates a ProjectConfig struct
func ValidateProjectConfig(config *ProjectConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate provider
	if err := ValidateProvider(config.Provider); err != nil {
		return fmt.Errorf("invalid provider: %w", err)
	}

	// Validate account
	if err := ValidateAccount(config.Account); err != nil {
		return fmt.Errorf("invalid account: %w", err)
	}

	// Validate image
	if err := ValidateImage(config.Image); err != nil {
		return fmt.Errorf("invalid image: %w", err)
	}

	return nil
}

// CreateDefaultProjectConfig creates a ProjectConfig with sensible defaults
func CreateDefaultProjectConfig() (*ProjectConfig, error) {
	// Get system username for default account
	username, err := GetSystemUsername()
	if err != nil {
		return nil, fmt.Errorf("failed to get system username: %w", err)
	}

	return &ProjectConfig{
		Provider: "claude",           // Default to Claude
		Account:  username,           // Use system username
		Image:    "base",             // Default to base image
		Danger:   false,              // Default to safe mode
	}, nil
}

// CheckDependencies verifies that required system dependencies are available
func CheckDependencies() error {
	// Check for Docker
	if err := checkCommand("docker"); err != nil {
		return fmt.Errorf("Docker is required but not found. Please install Docker and ensure the daemon is running: %w", err)
	}

	// Check for Git (for project hash generation and version info)
	if err := checkCommand("git"); err != nil {
		// Git is not strictly required, but warn the user
		fmt.Fprintf(os.Stderr, "Warning: Git not found. Some features may not work properly.\n")
	}

	return nil
}

// checkCommand verifies if a command is available in PATH
func checkCommand(command string) error {
	_, err := os.Stat("/usr/bin/" + command)
	if err == nil {
		return nil
	}

	_, err = os.Stat("/usr/local/bin/" + command)
	if err == nil {
		return nil
	}

	// Try to find in PATH
	path := os.Getenv("PATH")
	if path == "" {
		return fmt.Errorf("command %s not found and PATH is empty", command)
	}

	for _, dir := range []string{"/usr/bin", "/usr/local/bin", "/opt/homebrew/bin"} {
		if _, err := os.Stat(dir + "/" + command); err == nil {
			return nil
		}
	}

	return fmt.Errorf("command %s not found in PATH", command)
}
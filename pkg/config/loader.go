package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tailscale/hujson"
)

// CheckDependencies verifies that required system dependencies are available
func CheckDependencies() error {
	// Check for Docker
	if err := checkCommand("docker"); err != nil {
		return fmt.Errorf("docker is required but not found. Please install Docker and ensure the daemon is running: %w", err)
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

// FindDevContainerFile searches for devcontainer.json in the specified directory
// Search order: .devcontainer/devcontainer.json, then .devcontainer.json
func FindDevContainerFile(dir string) (string, bool, error) {
	// First try .devcontainer/devcontainer.json
	devcontainerPath := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	if _, err := os.Stat(devcontainerPath); err == nil {
		return devcontainerPath, true, nil
	}

	// Then try .devcontainer.json
	rootPath := filepath.Join(dir, ".devcontainer.json")
	if _, err := os.Stat(rootPath); err == nil {
		return rootPath, true, nil
	}

	return "", false, nil
}

// LoadDevContainerConfig loads and parses a devcontainer.json file
func LoadDevContainerConfig(filePath string) (*DevContainerConfig, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read devcontainer file %s: %w", filePath, err)
	}

	// Parse JSONC using hujson to convert to standard JSON
	standardJSON, err := hujson.Standardize(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSONC in %s: %w", filePath, err)
	}

	// Unmarshal into DevContainerConfig struct
	var config DevContainerConfig
	if err := json.Unmarshal(standardJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal devcontainer config in %s: %w", filePath, err)
	}

	return &config, nil
}

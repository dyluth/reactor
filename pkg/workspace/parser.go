package workspace

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	workspaceFileYML  = "reactor-workspace.yml"
	workspaceFileYAML = "reactor-workspace.yaml"
	requiredVersion   = "1"
)

// FindWorkspaceFile looks for reactor-workspace.yml or reactor-workspace.yaml in the specified directory.
// Returns the absolute path to the found file, whether it was found, and any error.
func FindWorkspaceFile(directory string) (string, bool, error) {
	if directory == "" {
		var err error
		directory, err = os.Getwd()
		if err != nil {
			return "", false, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Convert to absolute path for consistent results
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return "", false, fmt.Errorf("failed to get absolute path for directory %s: %w", directory, err)
	}

	// Try .yml first, then .yaml
	candidates := []string{
		filepath.Join(absDir, workspaceFileYML),
		filepath.Join(absDir, workspaceFileYAML),
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true, nil
		}
	}

	return "", false, nil
}

// ParseWorkspaceFile reads and parses a workspace file into a Workspace struct.
// It validates the version and ensures services are defined.
func ParseWorkspaceFile(filePath string) (*Workspace, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace file: %w", err)
	}

	var workspace Workspace
	if err := yaml.Unmarshal(data, &workspace); err != nil {
		return nil, fmt.Errorf("failed to parse workspace YAML: %w", err)
	}

	// Validate version
	if workspace.Version != requiredVersion {
		return nil, fmt.Errorf("unsupported workspace version '%s', expected '%s'", workspace.Version, requiredVersion)
	}

	// Validate services map is not empty
	if len(workspace.Services) == 0 {
		return nil, fmt.Errorf("workspace must define at least one service")
	}

	// Validate each service
	workspaceDir := filepath.Dir(filePath)
	for serviceName, service := range workspace.Services {
		if service.Path == "" {
			return nil, fmt.Errorf("service '%s' must define a path", serviceName)
		}

		// Resolve service path relative to workspace file
		servicePath := service.Path
		if !filepath.IsAbs(servicePath) {
			servicePath = filepath.Join(workspaceDir, service.Path)
		}

		// Clean the path to resolve any . or .. elements
		servicePath = filepath.Clean(servicePath)

		// Security check: ensure service path is within workspace directory or its subdirectories
		absWorkspaceDir, err := filepath.Abs(workspaceDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for workspace directory: %w", err)
		}

		absServicePath, err := filepath.Abs(servicePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for service '%s': %w", serviceName, err)
		}

		// Check if service path starts with workspace directory path
		relPath, err := filepath.Rel(absWorkspaceDir, absServicePath)
		if err != nil || filepath.IsAbs(relPath) || len(relPath) > 0 && relPath[0] == '.' {
			return nil, fmt.Errorf("service '%s' path '%s' must be within the workspace directory", serviceName, service.Path)
		}

		// Check if service directory exists
		if info, err := os.Stat(absServicePath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("service '%s' path '%s' does not exist", serviceName, service.Path)
			}
			return nil, fmt.Errorf("failed to check service '%s' path '%s': %w", serviceName, service.Path, err)
		} else if !info.IsDir() {
			return nil, fmt.Errorf("service '%s' path '%s' is not a directory", serviceName, service.Path)
		}
	}

	return &workspace, nil
}

// GenerateWorkspaceHash creates a SHA256 hash of the canonical, absolute path of the workspace file.
// This is used for workspace instance labeling.
func GenerateWorkspaceHash(workspaceFilePath string) (string, error) {
	absPath, err := filepath.Abs(workspaceFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Use the canonical absolute path for consistent hashing
	hash := sha256.Sum256([]byte(absPath))
	return fmt.Sprintf("%x", hash), nil
}

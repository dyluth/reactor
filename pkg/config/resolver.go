package config

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// ResolveImage determines the final container image to use based on precedence:
// CLI override > project config > provider default
func ResolveImage(projectImage, providerDefault, cliImage string) string {
	// CLI override takes highest precedence
	if cliImage != "" {
		if resolved, exists := BuiltinImages[cliImage]; exists {
			return resolved
		}
		return cliImage
	}
	
	// Project config takes second precedence
	if projectImage != "" {
		if resolved, exists := BuiltinImages[projectImage]; exists {
			return resolved
		}
		return projectImage
	}
	
	// Provider default is the fallback
	if resolved, exists := BuiltinImages[providerDefault]; exists {
		return resolved
	}
	return providerDefault
}

// GenerateProjectHash creates a consistent hash for the project directory
// This is used to isolate configurations between different projects for the same account
func GenerateProjectHash(projectRoot string) string {
	// Use absolute path to ensure consistency
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		// Fallback to the original path if absolute path fails
		absPath = projectRoot
	}
	
	hash := sha256.Sum256([]byte(absPath))
	// Return first 8 characters of hex-encoded hash for readability
	return fmt.Sprintf("%x", hash[:4])
}

// GetReactorHomeDir returns the reactor configuration directory path with optional isolation prefix
func GetReactorHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	dirname := ".reactor"
	if prefix := os.Getenv("REACTOR_ISOLATION_PREFIX"); prefix != "" {
		dirname = ".reactor-" + prefix
	}
	
	return filepath.Join(homeDir, dirname), nil
}

// GetProjectConfigPath returns the path to the project configuration file with optional isolation prefix
func GetProjectConfigPath() string {
	filename := ".reactor.conf"
	if prefix := os.Getenv("REACTOR_ISOLATION_PREFIX"); prefix != "" {
		filename = "." + prefix + ".conf"
	}
	return filename
}
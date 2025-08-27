package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/reactor/pkg/config"
)

// StateService validates existence of state directories and provides mount specifications
type StateService struct {
	resolved *config.ResolvedConfig
}

// NewStateService creates a new state validation service
func NewStateService(resolved *config.ResolvedConfig) *StateService {
	return &StateService{
		resolved: resolved,
	}
}

// ValidateDirectories checks that all required state directories exist
// Returns error if directories are missing, directing user to run `reactor config init`
func (s *StateService) ValidateDirectories() error {
	// Check if account configuration directory exists
	if _, err := os.Stat(s.resolved.AccountConfigDir); os.IsNotExist(err) {
		return fmt.Errorf("account directory does not exist: %s", s.resolved.AccountConfigDir)
	}

	// Check if project-specific directory exists
	if _, err := os.Stat(s.resolved.ProjectConfigDir); os.IsNotExist(err) {
		return fmt.Errorf("project configuration directory does not exist: %s", s.resolved.ProjectConfigDir)
	}

	// Check if provider-specific directories exist
	for _, mount := range s.resolved.Provider.Mounts {
		providerDir := filepath.Join(s.resolved.ProjectConfigDir, mount.Source)
		if _, err := os.Stat(providerDir); os.IsNotExist(err) {
			return fmt.Errorf("provider directory does not exist: %s", providerDir)
		}
	}

	return nil
}

// GetMounts generates mount specifications for the container
// Returns mount specifications in Docker bind mount format
func (s *StateService) GetMounts() []MountSpec {
	mounts := []MountSpec{}

	// Add provider-specific mounts
	for _, mount := range s.resolved.Provider.Mounts {
		sourcePath := filepath.Join(s.resolved.ProjectConfigDir, mount.Source)
		mounts = append(mounts, MountSpec{
			Source: sourcePath,
			Target: mount.Target,
			Type:   "bind",
		})
	}

	// Add project root mount (read-only by default)
	mounts = append(mounts, MountSpec{
		Source: s.resolved.ProjectRoot,
		Target: "/workspace",
		Type:   "bind",
	})

	return mounts
}

// MountSpec defines a container mount specification
type MountSpec struct {
	Source string // Host path (absolute)
	Target string // Container path
	Type   string // Mount type ("bind", "volume", etc.)
}

// GetAccount returns the account name from resolved configuration
func (s *StateService) GetAccount() string {
	return s.resolved.Account
}

// GetProjectHash returns the project hash from resolved configuration
func (s *StateService) GetProjectHash() string {
	return s.resolved.ProjectHash
}

// GetProjectRoot returns the project root directory
func (s *StateService) GetProjectRoot() string {
	return s.resolved.ProjectRoot
}

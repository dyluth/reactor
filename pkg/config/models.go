package config

import (
	"fmt"
	"os/user"
)

// ProjectConfig represents the project-level configuration stored in .reactor.conf
type ProjectConfig struct {
	Provider string `yaml:"provider"`         // claude, gemini, or custom
	Account  string `yaml:"account"`          // account name for isolation
	Image    string `yaml:"image"`            // base, python, go, or custom image URL
	Danger   bool   `yaml:"danger,omitempty"` // enable dangerous permissions
}

// MountPoint defines a directory mount for providers
type MountPoint struct {
	Source string // subdirectory under ~/.reactor/<account>/<project-hash>/
	Target string // path in container
}

// ProviderInfo defines built-in provider configuration
type ProviderInfo struct {
	Name         string       // claude, gemini
	DefaultImage string       // suggested default image
	Mounts       []MountPoint // multiple mount points for this provider
}

// ResolvedConfig contains fully resolved configuration with all paths
type ResolvedConfig struct {
	Provider         ProviderInfo
	Account          string
	Image            string
	ProjectRoot      string
	ProjectHash      string // first 8 chars of project path hash
	AccountConfigDir string // ~/.reactor/<account>/
	ProjectConfigDir string // ~/.reactor/<account>/<project-hash>/
	Danger           bool
}

// Built-in provider mappings (hardcoded but extensible)
var BuiltinProviders = map[string]ProviderInfo{
	"claude": {
		Name:         "claude",
		DefaultImage: "ghcr.io/dyluth/reactor/base:latest",
		Mounts: []MountPoint{
			{Source: "claude", Target: "/home/claude/.claude"},
			// Additional mounts can be added if claude stores files elsewhere
		},
	},
	"gemini": {
		Name:         "gemini",
		DefaultImage: "ghcr.io/dyluth/reactor/base:latest",
		Mounts: []MountPoint{
			{Source: "gemini", Target: "/home/claude/.gemini"},
			// Additional mounts can be added if gemini stores files elsewhere
		},
	},
	// Future providers (openai, etc.) will be added here with code changes
}

// Built-in image mappings for convenience
var BuiltinImages = map[string]string{
	"base":   "ghcr.io/dyluth/reactor/base:latest",
	"python": "ghcr.io/dyluth/reactor/python:latest",
	"node":   "ghcr.io/dyluth/reactor/node:latest",
	"go":     "ghcr.io/dyluth/reactor/go:latest",
}

// GetSystemUsername returns the current system username as default account
func GetSystemUsername() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	return currentUser.Username, nil
}

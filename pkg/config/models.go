package config

import (
	"fmt"
	"os/user"
)

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

// DevContainerConfig represents the structure of a devcontainer.json file
type DevContainerConfig struct {
	Name              string                   `json:"name"`
	Image             string                   `json:"image"`
	Build             *Build                   `json:"build"`
	ForwardPorts      []interface{}            `json:"forwardPorts"` // Can be int or string "host:container"
	RemoteUser        string                   `json:"remoteUser"`
	PostCreateCommand string                   `json:"postCreateCommand"`
	Customizations    *Customizations          `json:"customizations"`
}

// Build defines Docker build properties
type Build struct {
	Dockerfile string `json:"dockerfile"`
	Context    string `json:"context"`
}

// Customizations block for tool-specific settings
type Customizations struct {
	Reactor *ReactorCustomizations `json:"reactor"`
}

// ReactorCustomizations defines reactor-specific settings
type ReactorCustomizations struct {
	Account        string `json:"account"`
	DefaultCommand string `json:"defaultCommand"`
}

// GetSystemUsername returns the current system username as default account
func GetSystemUsername() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	return currentUser.Username, nil
}

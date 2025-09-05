package workspace

// Workspace defines the structure of the reactor-workspace.yml file.
type Workspace struct {
	Version  string             `yaml:"version"`
	Services map[string]Service `yaml:"services"`
}

// Service defines the configuration for a single service within the workspace.
type Service struct {
	Path    string `yaml:"path"`
	Account string `yaml:"account,omitempty"`
}

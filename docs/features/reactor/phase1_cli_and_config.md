# **Feature Design Document: Reactor Phase 1 - CLI Structure and Configuration Management**

Version: 1.0  
Status: Draft  
Author(s): Claude, cam  
Date: 2025-08-24

## **1. The 'Why': Rationale & User Focus**

*This section defines the purpose of the feature, the target user, and the value it delivers. It ensures we are solving the right problem for the right person.*

### **1.1. High-level summary**

Phase 1 establishes the foundational CLI structure for Reactor and implements robust configuration management. This phase creates the command-line interface using Cobra framework with all primary commands (`run`, `diff`, `accounts`, `config`) and implements the account-based configuration system with built-in provider mappings and simple project configuration files (`<project-dir>/.reactor.conf`). This foundation enables users to configure AI providers and manage account-isolated settings while providing a consistent, extensible CLI experience.

### **1.2. User personas**

* **Primary Persona: The AI-Powered Developer ("Dev")**: A software engineer who uses various AI CLI tools and needs to configure providers, manage accounts, and set up project-specific environments. They value clear configuration paths and want to understand what settings are being used.

* **Secondary Persona: The Tool Onboarder ("Ops")**: A developer or DevOps engineer who needs to understand how Reactor's configuration works to set up standardized environments for their team. They need clear configuration file locations and structures.

### **1.3. Problem statement & user stories**

**Problem Statement:**
Developers need a consistent way to configure AI providers and manage project-specific settings for containerized development environments. The configuration system must be discoverable, explicit, and support multiple providers and accounts without conflicts.

**User Stories:**

* As a **Dev**, I want to run `reactor config show` to see my current configuration, so that I understand what provider and settings are active for my project.
* As a **Dev**, I want to permanently switch my project's default tool, so I can run `reactor config set provider claude`.
* As a **Dev**, I want to configure my project and run it in a single command, so I can use a flag like `reactor run --provider gemini` to set the provider in my `.reactor.conf` and immediately launch the container.
* As a **Dev**, I want to run `reactor run --image myimage` and get a clear "not implemented yet" message, so that I know the command structure is ready for future phases.
* As an **Ops**, I want to run `reactor config show` to understand available providers and current configuration, so that I can standardize team configurations without complex setup.
* As a **Dev**, I want clear error messages when configuration is missing or invalid, so that I can fix issues quickly.

### **1.4. Success metrics**

**Business Metrics:**

* Users can successfully configure their first provider within 2 minutes of installation
* Zero configuration-related support questions during Phase 1 testing

**Technical Metrics:**

* All CLI commands execute in <100ms for configuration operations
* Configuration loading succeeds 100% of the time with valid config files
* Error messages provide actionable guidance for 100% of common misconfigurations

## **2. The 'How': Technical Design & Architecture**

*This section details the proposed technical solution, exploring the system context, alternatives, and the specific changes required across the stack.*

### **2.1. System context & constraints**

* **Technology Stack:** Go 1.21+, Cobra CLI Framework, YAML parsing (gopkg.in/yaml.v3)
* **Current State:** This is a new tool being built as part of the reactor monorepo. The `cmd/reactor/` directory will house the main binary.
* **Technical Constraints:** Must follow the monorepo structure with shared code in `pkg/`. Configuration files must be human-readable and version-controllable. CLI must follow standard POSIX conventions.
* **Distribution:** Single binary with build-time metadata injection via linker flags.

### **2.2. Guiding design principles**

* **Simplicity over Complexity (YAGNI):** Configuration system will be explicit and straightforward - no complex inheritance or auto-discovery mechanisms beyond documented fallback behavior.
* **Consistency with Existing Code:** Follow the established modular, layered architecture from the Project Charter with clear separation between `cmd`, `core`, and service layers.
* **Clarity and Readability:** Configuration files will be self-documenting with clear field names. Error messages will be actionable and specific.

### **2.3. Alternatives considered**

**Option 1: Single Configuration File Approach**
* **Description:** Store all configuration (global and project) in a single `~/.reactor/config.yaml` file with nested sections.
* **Pros:** Simpler to implement, single source of truth, easier to backup.
* **Cons:** Conflicts with the multi-account isolation requirement, harder to manage project-specific overrides.

**Option 2: Account-Based Directory Structure with Simple Config (Chosen Approach)**
* **Description:** Account-isolated directories at `~/.reactor/<account>/` contain actual agent config files. Simple project config file `<project-dir>/.reactor.conf` specifies account/provider/image. Built-in image mappings for convenience (base, python, go) plus support for custom images.
* **Pros:** Perfect account isolation, agents manage their own config files via setup wizards, simple configuration model, supports both built-in and custom images.
* **Cons:** Requires understanding of account-based directory structure.

**Chosen Approach Justification:**
This approach provides perfect account isolation while letting AI agents manage their own configuration through their native setup processes. The `~/.reactor/<account>/<provider>/` directories are mounted into containers where agents expect their config, allowing seamless setup and persistent configuration.

### **2.4. Detailed design**

The implementation consists of four main components:

**1. CLI Command Structure (cmd layer)**
```
reactor/
├── run [--image IMAGE] [--account ACCOUNT] [--provider PROVIDER] [--danger]
│   └── (validates config, prepares for container provisioning, returns "Container provisioning not implemented yet")
│   └── Note: Flags like --provider, --account, and --image serve as temporary, single-command overrides for the settings in .reactor.conf.
├── diff [--discovery]
│   └── (validates config, returns "Container diff not implemented yet")  
├── accounts
│   ├── list (shows configured accounts)
│   ├── set <account-name> (sets active account)
│   └── show (shows current account)
└── config
    ├── show (displays resolved configuration hierarchy + shows config file locations)
    ├── set <key> <value> (modifies project config only, supports danger=true/false)
    ├── get <key> (retrieves project config values)
    └── init (creates project config + account directories as needed)
```

**2. Core Configuration Service**
- `pkg/config/service.go` - Main configuration management
- `pkg/config/models.go` - Configuration structs
- `pkg/config/loader.go` - File loading and parsing logic
- `pkg/config/validator.go` - Configuration validation

**3. Configuration Resolution Logic**
```
1. Load built-in provider mappings (claude->base, gemini->base, custom images)
2. Check for project config at .reactor.conf
   - If missing: Error "No project configuration found. Run 'reactor config init' to create one."
3. Apply and persist CLI flag overrides. Any flags passed to 'run' (e.g., --provider, --image) will update the .reactor.conf file before the configuration is resolved.
4. Resolve image: Use config.image, fallback to provider default, then CLI override
5. Generate project hash from absolute project path (first 8 chars of SHA-256)
6. Resolve directory structure: ~/.reactor/<account>/<project-hash>/<provider>/
7. Create account/provider directories only during 'reactor config init' when needed
8. Return resolved configuration with all mount paths and final image selection
```

**4. Error Handling Strategy**
- Missing .reactor.conf: Error with instruction to run `reactor config init`
- Invalid YAML syntax: Show syntax error with file path and line numbers  
- Invalid provider: Show available providers (claude, gemini, custom)
- Invalid image: For built-in providers, show valid images; for custom, validate image format

**2.4.5. Test & Automation Isolation**
To ensure that automated tests do not interfere with a user's local configuration or with other concurrent test runs, the tool will support an isolation mode via an environment variable.

- **Environment Variable:** `REACTOR_ISOLATION_PREFIX`
- **Behavior:** If this variable is set to a value (e.g., `test-run-123`), all file paths and resource names will be prefixed with this value.
  - **Host Directory:** `~/.reactor/` becomes `~/.reactor-test-run-123/`
  - **Config File:** `.reactor.conf` becomes `.reactor-test-run-123.conf`
- **Default Behavior:** If the variable is not set or empty, the tool functions with its default paths and names.
- **Implementation:** The Makefile is responsible for setting this environment variable correctly for all test targets. Phase 2 will extend this isolation to container names and other resources.

#### **2.4.1. Data model updates**

```go
// pkg/config/models.go

// Simple project configuration
type ProjectConfig struct {
    Provider string `yaml:"provider"` // claude, gemini, or custom
    Account  string `yaml:"account"`  // account name for isolation
    Image    string `yaml:"image"`    // base, python, go, or custom image URL
    Danger   bool   `yaml:"danger,omitempty"` // enable dangerous permissions (e.g., --dangerously-skip-permissions)
}

// Mount point definition for providers
type MountPoint struct {
    Source string // subdirectory under ~/.reactor/<account>/<project-hash>/
    Target string // path in container
}

// Built-in provider definitions (in code, not config files)
type ProviderInfo struct {
    Name         string      // claude, gemini
    DefaultImage string      // suggested default image (base, python, go)
    Mounts       []MountPoint // multiple mount points for this provider
}

// Resolved configuration with mount paths
type ResolvedConfig struct {
    Provider         ProviderInfo
    Account          string
    ProjectRoot      string
    ProjectHash      string               // first 8 chars of project path hash
    AccountConfigDir string               // ~/.reactor/<account>/
    ProjectConfigDir string               // ~/.reactor/<account>/<project-hash>/
    Image            string               // resolved from CLI flag or provider default
}

// Built-in provider mappings (hardcoded but extensible)
var BuiltinProviders = map[string]ProviderInfo{
    "claude": {
        Name:         "claude",
        DefaultImage: "base",
        Mounts: []MountPoint{
            {Source: "claude", Target: "/home/claude/.claude"},
            // Additional mounts can be added if claude stores files elsewhere
        },
    },
    "gemini": {
        Name:         "gemini",
        DefaultImage: "base", 
        Mounts: []MountPoint{
            {Source: "gemini", Target: "/home/claude/.gemini"},
            // Additional mounts can be added if gemini stores files elsewhere
        },
    },
    // Future providers (openai, etc.) will be added here with code changes
}

// Built-in image mappings  
var BuiltinImages = map[string]string{
    "base":   "ghcr.io/reactor-suite/base:latest",
    "python": "ghcr.io/reactor-suite/python:latest", 
    "go":     "ghcr.io/reactor-suite/go:latest",
}

// GetConfigPath returns the project config file path with optional isolation prefix
func GetConfigPath() string {
    filename := ".reactor.conf"
    if prefix := os.Getenv("REACTOR_ISOLATION_PREFIX"); prefix != "" {
        filename = "." + prefix + ".conf"
    }
    return filename
}

// GetReactorHomeDir returns the reactor home directory with optional isolation prefix  
func GetReactorHomeDir() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    
    dirname := ".reactor"
    if prefix := os.Getenv("REACTOR_ISOLATION_PREFIX"); prefix != "" {
        dirname = ".reactor-" + prefix
    }
    
    return filepath.Join(homeDir, dirname), nil
}
```

#### **2.4.2. Data migration plan**

N/A - This is a new tool with no existing data.

#### **2.4.3. API & backend changes**

N/A - This is a client-side CLI tool.

#### **2.4.4. Frontend changes**

N/A - This is a CLI tool.

### **2.5. Non-functional requirements (NFRs)**

* **Performance:** All CLI operations must complete in <100ms on standard hardware. Configuration file parsing must handle files up to 1MB.
* **Reliability:** Configuration loading must be atomic - either succeed completely or fail with clear error message. Partial configuration states are not allowed.
* **Operations & Developer Experience:** All configuration operations must provide clear, actionable feedback. The `reactor config show` command must clearly display the configuration hierarchy and source of each setting.

## **3. The 'What': Implementation & Execution**

*This section breaks the work into manageable pieces and defines the strategy for testing, documentation, and quality assurance.*

### **3.1. Phased implementation plan**

**PR 1.1: Cobra CLI Structure Setup**
* [ ] Set up `cmd/reactor/main.go` with Cobra root command
* [ ] Create command structure for `run`, `diff`, `accounts`, `config` with placeholder implementations
* [ ] Add version command with build-time metadata injection
* [ ] Implement basic help text and command descriptions
* [ ] Add Makefile targets for building and testing

**PR 1.2: Configuration Service Implementation**
* [ ] Create `pkg/config/` package with models, service, loader, validator
* [ ] Implement project config loading from `.reactor.conf`
* [ ] Add built-in provider mappings (claude, gemini) with hardcoded mount paths
* [ ] Implement account directory resolution (default to system username)
* [ ] Implement `REACTOR_ISOLATION_PREFIX` environment variable support for test isolation
* [ ] Implement `reactor config` commands:
  * [ ] `show` - displays config + prints account directory locations
  * [ ] `set` - modifies project config, prints global config locations for reference
  * [ ] `get` - retrieves project config values
  * [ ] `init` - creates .reactor.conf + account directories as needed
* [ ] Add comprehensive error handling with actionable messages

### **3.2. Testing strategy**

* **Unit Tests:** 
  * Test configuration loading with various file states (missing, invalid YAML, partial configs)
  * Test configuration resolution logic with different precedence scenarios
  * Test YAML parsing with edge cases (empty files, malformed syntax)
  * Test CLI flag parsing and validation

* **Integration Tests:** 
  * Test full command execution with temporary config directories (using `REACTOR_ISOLATION_PREFIX`)
  * Verify configuration precedence works correctly across global/project/CLI layers
  * Test error handling for common user mistakes
  * Test isolation prefix functionality ensures test runs don't interfere with user config

* **End-to-End (E2E) User Story Tests:**
  * **User Story 1 ("As a Dev, I want to run `reactor config show`..."):** Test script creates project, runs `reactor config show`, verifies output format and content including sensible defaults
  * **User Story 2 ("As a Dev, I want to run `reactor config set provider claude`..."):** Test script sets provider using nested key syntax, verifies configuration persists and shows in subsequent `config show`
  * **User Story 3 ("As a Dev, I want to run `reactor run --image myimage`..."):** Test script runs command, verifies it validates config and returns "Container provisioning not implemented yet" message

## **4. The 'What Ifs': Risks & Mitigation**

*This section addresses potential issues, ensuring the feature is secure, reliable, and can be deployed and managed safely.*

### **4.1. Security & privacy considerations**

* **Configuration File Permissions:** Configuration files will be created with restrictive permissions (0600) to prevent unauthorized access to potentially sensitive provider settings.
* **Input Validation:** All YAML input will be validated against expected schemas to prevent injection attacks or parsing vulnerabilities.
* **Path Traversal:** All file path operations will be validated to prevent directory traversal attacks.

### **4.2. Rollout & deployment**

* **Feature Flags:** N/A - This is foundational functionality that must work for the tool to be usable.
* **Monitoring & Observability:** Structured logging will be implemented with different log levels (INFO, DEBUG, ERROR) for configuration operations.
* **Rollback Plan:** Since this is Phase 1 of a new tool, rollback would involve reverting to previous commit or disabling the specific binary.

### **4.3. Dependencies and integrations**

* **Internal Dependencies:** None - this is the foundation layer.
* **External Dependencies:** 
  * Cobra CLI framework for command structure
  * gopkg.in/yaml.v3 for YAML parsing
  * Standard Go libraries for file operations

### **4.4. Cost and resource analysis**

* **Infrastructure Costs:** None - tool runs entirely on user's local machine
* **Operational Costs:** Minimal - configuration files are lightweight and operations are local

### **4.5. Account-Based Directory Structure**

The key architectural insight is that `~/.reactor/` uses an account-based structure where each account gets isolated directories for each provider:

```
~/.reactor/
├── default/                    # default account (system username)
│   ├── a1b2c3d4/              # project hash (first 8 chars of project path hash)
│   │   ├── claude/            # mounted to /home/claude/.claude
│   │   │   ├── auth.json      # created by claude CLI setup wizard
│   │   │   └── preferences.json # created by claude CLI
│   │   ├── gemini/            # mounted to /home/claude/.gemini
│   │   │   └── config.json    # created by gemini CLI setup wizard
│   │   └── openai/            # mounted to /home/claude/.openai
│   │       └── api_key.txt    # created by openai CLI setup
│   └── e5f6g7h8/              # different project, same account
│       └── claude/            # completely isolated per project
│           ├── auth.json      # can have different settings per project
│           └── preferences.json
├── work-account/              # separate work account
│   └── a1b2c3d4/              # same project but different account
│       ├── claude/            # completely isolated work config
│       │   ├── auth.json      # different auth tokens
│       │   └── preferences.json   
│       └── gemini/
│           └── config.json
└── personal-projects/         # another account
    └── f9a0b1c2/              # different project hash
        └── claude/
            ├── auth.json
            └── preferences.json
```

**Benefits:**
- **Perfect Account Isolation**: Each account has completely separate config directories
- **Project-Level Isolation**: Multiple projects using the same account/provider don't interfere with each other  
- **Agent-Managed Config**: AI agents create and manage their own config files through their setup wizards  
- **Persistent Configuration**: Config survives container restarts and removals
- **Multiple Mount Points**: Each provider can mount multiple directories as needed
- **Simple Mounting**: Each provider directory mounts to where that agent expects its config
- **Easy Backup**: Account/project directories can be copied/shared independently

### **4.6. Open questions & assumptions**

**Implementation Decisions:**
* **Mount Paths**: Hardcoded in `BuiltinProviders` map but extensible for new providers via code changes
* **Account Directory Creation**: Only during `reactor config init` when a project needs specific account/provider setup  
* **Custom Providers**: Require code changes to add new providers (e.g., OpenAI) - no runtime provider definition
* **Default Account**: Uses system username as default account name
* **Config Scope**: `reactor config set` only modifies project-level settings, prints locations of global config files for manual editing
* **Danger Mode**: `--danger` flag enables dangerous AI agent permissions and is stored in project config for persistence

### **4.7. Security & Privacy Considerations**

**Danger Mode Security Risk:**
* **`--danger` flag**: This enables dangerous permissions for AI agents (e.g., Claude's `--dangerously-skip-permissions` flag)
* **Risk**: AI agents may gain excessive system access, potentially modifying or deleting files outside project directory
* **Mitigation**: 
  - Flag requires explicit user consent per project
  - Setting is stored in version-controlled `.reactor.conf` making it visible to team members
  - Clear warning messages when danger mode is enabled
  - Documentation must emphasize risks and appropriate use cases

**Configuration File Security:**
* Configuration files created with restrictive permissions (0600) to prevent unauthorized access
* Account directories isolate configuration to prevent cross-account information leakage
* Project-level isolation prevents configuration conflicts between different projects

**Assumptions:**
* Users understand basic YAML syntax for project configuration editing
* Users have read/write permissions in their home directory and project directories  
* AI agents will properly handle their config directories being mounted from the host
* The 3-image strategy (base, python, go) covers most common development scenarios
* System username is a reasonable default account identifier
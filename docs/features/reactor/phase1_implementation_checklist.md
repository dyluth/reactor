# Phase 1 Implementation Checklist

## Overview
This checklist provides a detailed breakdown of Phase 1 implementation tasks, organized by pull request and component.

## PR 1.1: Cobra CLI Structure Setup

### Core CLI Setup
- [ ] Create `cmd/reactor/main.go` with Cobra root command
- [ ] Set up build-time metadata injection via linker flags (version, commit, build date)
- [ ] Create basic command structure:
  - [ ] `reactor run [--image IMAGE] [--account ACCOUNT] [--provider PROVIDER] [--danger]` (placeholder returning "Container provisioning not implemented yet")
  - [ ] `reactor diff [--discovery]` (placeholder returning "Container diff not implemented yet")
  - [ ] `reactor accounts` (with subcommands)
  - [ ] `reactor config` (with subcommands)
  - [ ] `reactor version` (displays build metadata)

### Command Structure Details
- [ ] `reactor accounts`:
  - [ ] `list` - shows configured accounts (scan ~/.reactor/ directories)
  - [ ] `set <account-name>` - sets active account in project config
  - [ ] `show` - shows current account from project config
- [ ] `reactor config`:
  - [ ] `show` - placeholder for configuration display
  - [ ] `set <key> <value>` - placeholder for configuration setting
  - [ ] `get <key>` - placeholder for configuration retrieval
  - [ ] `init` - placeholder for configuration initialization

### Help & Documentation
- [ ] Implement comprehensive help text for all commands
- [ ] Add usage examples in help output
- [ ] Set up proper command descriptions and flags
- [ ] Add --verbose flag for debug output (structured logging)

### Build System
- [ ] Create Makefile with targets:
  - [ ] `make build` - builds reactor binary
  - [ ] `make test` - runs all tests
  - [ ] `make lint` - runs golangci-lint
  - [ ] `make clean` - cleans build artifacts
  - [ ] `make install` - installs to local system

### Initial Testing
- [ ] Set up basic CLI integration tests
- [ ] Test command structure and help output
- [ ] Verify build-time metadata injection works
- [ ] Test all placeholder commands return expected "not implemented" messages

## PR 1.2: Configuration Service Implementation

### Package Structure
- [ ] Create `pkg/config/` package with:
  - [ ] `models.go` - Configuration structs and built-in mappings
  - [ ] `service.go` - Main configuration service
  - [ ] `loader.go` - File loading and parsing logic
  - [ ] `validator.go` - Configuration validation
  - [ ] `resolver.go` - Account and path resolution

### Built-in Mappings
- [ ] Define `BuiltinProviders` map with:
  - [ ] `claude` provider (base image, /home/claude/.claude mount)
  - [ ] `gemini` provider (base image, /home/claude/.gemini mount)
- [ ] Define `BuiltinImages` map with:
  - [ ] `base` -> `ghcr.io/reactor-suite/base:latest`
  - [ ] `python` -> `ghcr.io/reactor-suite/python:latest`
  - [ ] `go` -> `ghcr.io/reactor-suite/go:latest`

### Configuration Loading
- [ ] Implement project config loading from `.reactor.conf`
- [ ] Add YAML parsing with proper error handling
- [ ] Implement configuration validation (YAML structure only)
- [ ] Add CLI flag override handling (project-level only)

### Account Directory Management
- [ ] Implement account directory resolution:
  - [ ] Default account = system username
  - [ ] Support custom account names
  - [ ] Resolve to `~/.reactor/<account>/`
- [ ] Create account directories during `config init` only
- [ ] Validate directory permissions

### Configuration Commands Implementation
- [ ] `reactor config show`:
  - [ ] Display current project configuration (including danger mode)
  - [ ] Show resolved account directory paths with project hash
  - [ ] Display built-in provider and image mappings
  - [ ] Show configuration hierarchy (project -> built-ins)
- [ ] `reactor config set <key> <value>`:
  - [ ] Support nested keys (e.g., `provider`, `account`, `image`, `danger`)
  - [ ] Support boolean values for danger flag (`danger=true/false`)
  - [ ] Validate image names against built-in images or custom format
  - [ ] Modify project `.reactor.conf` only
  - [ ] Print locations of account config directories for reference
  - [ ] Validate against known configuration keys
- [ ] `reactor config get <key>`:
  - [ ] Retrieve values from project configuration
  - [ ] Support nested key access
  - [ ] Return appropriate exit codes for missing keys
- [ ] `reactor config init`:
  - [ ] Create `.reactor.conf` with prompts for provider/account/image
  - [ ] Create account directories as needed (`~/.reactor/<account>/<project-hash>/<provider>/`)
  - [ ] Generate project hash from absolute project path (first 8 chars of SHA-256)
  - [ ] Set sensible defaults (provider=claude, account=username, image=base, danger=false)

### Account Commands Implementation
- [ ] `reactor accounts list`:
  - [ ] Scan `~/.reactor/` for existing account directories
  - [ ] Display account names and configured providers
- [ ] `reactor accounts set <account>`:
  - [ ] Update project config account field
  - [ ] Validate account directory exists or can be created
- [ ] `reactor accounts show`:
  - [ ] Display current account from project config
  - [ ] Show account directory path

### Error Handling
- [ ] Missing .reactor.conf: Clear error with `reactor config init` instruction
- [ ] Invalid YAML: Show syntax error with file path and line numbers
- [ ] Invalid provider: Show available providers (claude, gemini)
- [ ] Invalid image: Show available images (base, python, go) or validate custom format
- [ ] Permission errors: Clear messages about directory access
- [ ] Missing directories: Instructions for resolution

### Configuration Resolution Logic
- [ ] Implement complete configuration resolution:
  1. [ ] Load built-in provider mappings
  2. [ ] Check for project config (error if missing)
  3. [ ] Apply CLI flag overrides (including --danger and --image)
  4. [ ] Resolve final image (config.image -> provider default -> CLI override)
  5. [ ] Generate project hash from absolute project path
  6. [ ] Resolve account directory path with project hash
  7. [ ] Return `ResolvedConfig` with all mount paths and final image

### Security Implementation
- [ ] Danger mode implementation:
  - [ ] Store danger flag in project configuration
  - [ ] Display clear warnings when danger mode is enabled
  - [ ] Pass dangerous permissions to AI agents when flag is set
  - [ ] Create configuration files with restrictive permissions (0600)
- [ ] Account isolation validation:
  - [ ] Ensure account directories are properly isolated
  - [ ] Validate project hash prevents cross-project contamination

### Integration Updates
- [ ] Update `reactor run` to validate configuration and show resolved paths
- [ ] Update `reactor diff` to validate configuration 
- [ ] Ensure all commands properly handle missing/invalid configuration

## Testing Strategy

### Unit Tests
- [ ] Configuration loading with various file states (missing, invalid YAML, partial configs)
- [ ] Built-in provider and image mapping resolution
- [ ] Account directory path resolution (including edge cases)
- [ ] YAML parsing with malformed input
- [ ] CLI flag parsing and override behavior
- [ ] Nested key access in config get/set operations

### Integration Tests  
- [ ] Full command execution with temporary config directories
- [ ] Configuration creation and modification workflows
- [ ] Account directory creation and management
- [ ] Error handling for common user mistakes
- [ ] CLI flag override behavior across all commands

### End-to-End Tests
- [ ] **Config Show Workflow**: Create project, run `reactor config show`, verify output includes account paths and built-in mappings
- [ ] **Config Set Workflow**: Set provider using `reactor config set provider claude`, verify persistence and display in `config show`
- [ ] **Config Init Workflow**: Run `reactor config init` in empty directory, verify `.reactor.conf` creation and account directories
- [ ] **Placeholder Commands**: Verify `reactor run --image python` validates config and returns "not implemented" message
- [ ] **Account Management**: Test account listing, setting, and switching workflows

## Definition of Done
- [ ] All commands execute without errors (placeholder or functional)
- [ ] Configuration validation works correctly
- [ ] Account directories are created and managed properly
- [ ] Error messages are clear and actionable
- [ ] All tests pass (unit, integration, e2e)
- [ ] Code passes linting and formatting checks
- [ ] Commands provide appropriate help text and usage examples
- [ ] Build system works correctly with proper metadata injection
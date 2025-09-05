# **Feature Design Document: M3 - Reactor-Specific Extensions**

Version: 1.1
Status: Implementation Ready
Author(s): Gemini, cam
Date: 2025-09-02
Updated: 2025-01-17

## **1. The 'Why': Rationale & User Focus**

### **1.1. High-level summary**

This feature implements `reactor`'s primary value-add on top of the Dev Container standard: seamless, account-based credential management and a configurable entrypoint for AI tools. By reading values from a `customizations.reactor` block within `devcontainer.json`, `reactor` can automate the tedious process of managing and mounting secret files for different AI providers and can launch the user directly into their preferred tool, significantly improving workflow efficiency.

### **1.2. User personas**

*   **Primary Persona: The Multi-Account Developer ("Pro-Dev")**: A professional developer who uses different AI tools and accounts for different contexts (e.g., a personal account for experimentation, a work account for company projects). They need to switch between these contexts effortlessly without manually moving or configuring credential files.

### **1.3. Problem statement & user stories**

**Problem Statement:**
While Dev Containers standardize the development environment, they don't have a built-in solution for managing context-specific secrets like AI provider credentials. Developers are forced to manually copy files, set environment variables, or use other ad-hoc methods, which is inefficient, error-prone, and insecure. Furthermore, the default entrypoint is always a shell, requiring extra steps to launch the desired AI tool.

**User Stories:**

*   As a **Pro-Dev**, I want to specify `"account": "work-account"` in my `devcontainer.json`, so that `reactor up` automatically and securely mounts my pre-configured work credentials into the container.
*   As a **Pro-Dev**, I want to specify `"account": "personal-account"` in another project, so I can seamlessly switch between my work and personal AI contexts without any manual steps.
*   As a **Pro-Dev**, I want to set `"defaultCommand": "claude"` in my `devcontainer.json`, so that `reactor up` drops me directly into the Claude AI shell instead of a bash prompt, saving me a step every time I start my environment.

### **1.4. Success metrics**

**Technical Metrics:**

*   When `customizations.reactor.account` is set, `reactor up` correctly mounts the corresponding directories from the host's `~/.reactor/{account}/{project-hash}/{provider}/` into the container for all configured providers.
*   When `customizations.reactor.defaultCommand` is set, the container starts and executes that command as its entrypoint.
*   If the `account` property is omitted, the system gracefully falls back to using the system username, maintaining existing behavior.
*   The implementation is covered by unit and integration tests.

## **2. The 'How': Technical Design & Architecture**

### **2.1. System context & constraints**

*   **Technology Stack:** Go, Cobra, Docker Go SDK.
*   **Current State:** The `DevContainerConfig` and `ResolvedConfig` structs already have the necessary fields within the `ReactorCustomizations` struct. The configuration service correctly parses these fields, but they are not yet used by the core container-creation logic.
*   **Technical Constraints:** The solution must not require any changes to the `devcontainer.json` specification beyond using the standard `customizations` block. The account-based directory structure on the host machine (`~/.reactor/{account}`) is a core concept of `reactor`.

### **2.2. Detailed design**

The implementation will primarily touch the `pkg/config` service and the `pkg/core` blueprint creation logic.

#### **2.2.1. Configuration Flow (`pkg/config`)**

The `config.Service` already parses the `customizations.reactor` block. The key changes are:

1. **Add `DefaultCommand` field to `ResolvedConfig`** - This field must be added to the struct in `pkg/config/models.go`
2. **Update mapping function** - The `mapToResolvedConfig()` function in `pkg/config/service.go` must be updated to extract and map the `defaultCommand` value

*   **`account` fallback:** If `customizations.reactor.account` is not present or is empty, the `account` field in `ResolvedConfig` **must** fall back to the system username. This ensures backward compatibility and a smooth experience for users not using the feature. *(Already implemented)*
*   **`defaultCommand` fallback:** If `customizations.reactor.defaultCommand` is not present or empty, the `defaultCommand` field in `ResolvedConfig` should be empty, allowing the core logic to default to `/bin/bash`.

#### **2.2.2. Core Logic Updates (`pkg/core`)**

**BREAKING CHANGE:** The `NewContainerBlueprint` function signature will be updated to remove the `mounts []MountSpec` parameter, as the function will now be responsible for constructing ALL mounts internally.

**New Signature:**
```go
func NewContainerBlueprint(resolved *config.ResolvedConfig, isDiscovery bool, dockerHostIntegration bool, portMappings []PortMapping) *ContainerBlueprint
```

**Mount Construction Logic:**
The function will construct all mounts internally in the following order:
1. **Workspace Mount** (unless in discovery mode): `/workspace` ← `resolved.ProjectRoot`
2. **Provider Credential Mounts** (unless in discovery mode): For ALL providers in `config.BuiltinProviders`

**Implementation Details:**

1.  **`defaultCommand` Logic:**
    *   The function will check `resolved.DefaultCommand`.
    *   If the command is not empty, `blueprint.Command` will be set to `[]string{resolved.DefaultCommand}`.
    *   If it is empty, `blueprint.Command` will be set to the default shell `[]string{"/bin/bash"}` as it is now.

2.  **Account Mount Logic:**
    *   **Multi-Provider Support:** The function will iterate through ALL providers in the `config.BuiltinProviders` map (`claude`, `gemini`, etc.) to provide a complete credential environment.
    *   **Nested Mount Iteration:** For each provider, iterate through ALL mount points in `provider.Mounts` (supporting providers with multiple mount locations).
    *   **Host Path Construction:** `filepath.Join(resolved.ProjectConfigDir, mount.Source)` where `resolved.ProjectConfigDir` is `~/.reactor/{account}/{project-hash}/`
    *   **Container Target Path:** Use `mount.Target` directly from the provider configuration.
    *   **Path Resolution:** Use existing `config.GetReactorHomeDir()` for consistent home directory handling across test and production environments.

**Directory Management:**
*   **No Pre-validation:** All directory existence validation has been removed. Docker's bind mount functionality will automatically create the entire directory path on the host if missing.
*   **StateService Removal:** The `StateService` has been entirely removed as its responsibilities (directory validation and mount construction) are no longer needed.

### **2.3. Implementation plan**

This will be implemented as a single comprehensive PR to maintain architectural consistency.

**Core Implementation Tasks:**
*   [ ] **Config Layer Updates:**
    *   Add `DefaultCommand string` field to `ResolvedConfig` struct in `pkg/config/models.go`
    *   Update `mapToResolvedConfig()` in `pkg/config/service.go` to extract and map `defaultCommand` from reactor customizations
*   [ ] **Core Logic Updates:**
    *   Update `NewContainerBlueprint` function signature in `pkg/core/blueprint.go` (remove `mounts` parameter)
    *   Implement internal mount construction for workspace and all provider credentials
    *   Add `defaultCommand` logic with fallback to `/bin/bash`
*   [ ] **Architecture Cleanup:**
    *   Remove `StateService` entirely from `pkg/core/state.go`
    *   Update all callsites of `NewContainerBlueprint` (primarily `upCmdHandler` in `cmd/reactor/main.go`)
    *   Remove StateService usage from tests
*   [ ] **Testing:**
    *   Extend `testutil` package for temporary credential directory management
    *   Add unit tests for blueprint creation with multiple providers and mount points
    *   Add integration test for account-based credential mounting
    *   Add integration test for defaultCommand functionality

### **2.4. Testing strategy**

*   **Unit Tests:**
    *   **Mount Path Construction:** Verify mount paths are constructed correctly for different account names and all provider combinations
    *   **Multi-Provider Support:** Test that ALL providers in `BuiltinProviders` get credential mounts created
    *   **Multiple Mount Points:** Test providers with multiple mount points per provider (future-proofing)
    *   **DefaultCommand Logic:** Test command setting with various defaultCommand values and empty fallback
    *   **Discovery Mode:** Verify no mounts are created when `isDiscovery = true`

*   **Integration Tests:**
    *   **Account Mount Test:**
        1.  Use `testutil` to create isolated temporary credential directories: `temp-home/.reactor/{test-account}/{project-hash}/{provider}/`
        2.  Place test files in multiple provider directories (e.g., `claude/test-creds.txt`, `gemini/test-creds.txt`)
        3.  Use a `devcontainer.json` that specifies `"account": "test-account"`
        4.  Run `reactor up`
        5.  Use `docker exec` to verify files exist at ALL expected container locations (`/home/claude/.claude/test-creds.txt`, `/home/claude/.gemini/test-creds.txt`)
    *   **Default Command Test:**
        1.  Use a `devcontainer.json` that specifies `"defaultCommand": "echo 'init command successful'"`
        2.  Run `reactor up` and capture the container's output
        3.  Assert that the output contains "init command successful"
        4.  Assert that `reactor up` exits with code 0 after the command completes

## **3. The 'What Ifs': Risks & Mitigation**

*   **Risk:** A user specifies an account in `devcontainer.json`, but the corresponding directory does not exist on the host.
*   **Mitigation:** This is not an error condition. Docker's bind mount functionality will automatically create the entire directory path on the host (`~/.reactor/{account}/{project-hash}/{provider}/`) with appropriate ownership. This design eliminates the need for pre-validation and simplifies the user experience.

*   **Risk:** A user specifies an invalid `defaultCommand` that doesn't exist in the container's `PATH`.
*   **Mitigation:** The container will fail to start, and the Docker error message will be streamed to the user. This is standard, expected behavior, and `reactor` does not need to add extra validation for it.

*   **Risk:** StateService removal could break existing functionality if it has responsibilities beyond mount management.
*   **Mitigation:** Comprehensive testing during implementation will verify that all StateService responsibilities have been properly migrated or are no longer needed. The service's primary functions (directory validation and mount construction) are being replaced by Docker's native capabilities and centralized blueprint logic.

*   **Risk:** Mounting credentials for ALL providers could create unnecessary container clutter or security concerns.
*   **Mitigation:** This design provides maximum utility by giving users access to all configured AI tools within a single environment. Empty directories (when users don't have specific provider credentials) are harmless and provide a consistent experience. Users maintain control through the account selection mechanism.

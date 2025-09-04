# **Feature Design Document: M3 - Reactor-Specific Extensions**

Version: 1.0
Status: Approved
Author(s): Gemini, cam
Date: 2025-09-02

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

*   When `customizations.reactor.account` is set, `reactor up` correctly mounts the corresponding directories from the host's `~/.reactor/{account}/` into the container.
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

The `config.Service` already parses the `customizations.reactor` block. The key change is to ensure the `account` and `defaultCommand` values are consistently plumbed through to the `ResolvedConfig` struct.

*   **`account` fallback:** If `customizations.reactor.account` is not present or is empty, the `account` field in `ResolvedConfig` **must** fall back to the system username. This ensures backward compatibility and a smooth experience for users not using the feature.
*   **`defaultCommand` fallback:** If `customizations.reactor.defaultCommand` is not present or empty, the `defaultCommand` field in `ResolvedConfig` should be empty, allowing the core logic to default to `/bin/bash`.

#### **2.2.2. Core Logic Updates (`pkg/core`)**

The `NewContainerBlueprint` function in `pkg/core/blueprint.go` will be updated to use the new fields from `ResolvedConfig`.

1.  **`defaultCommand` Logic:**
    *   The function will check `resolved.DefaultCommand`.
    *   If the command is not empty, `blueprint.Command` will be set to `[]string{resolved.DefaultCommand}`.
    *   If it is empty, `blueprint.Command` will be set to the default shell `[]string{"/bin/bash"}` as it is now.

2.  **Account Mount Logic:**
    *   This is the most significant change. The function will now be responsible for constructing the credential mounts.
    *   It will iterate through the `config.BuiltinProviders` map (which contains `claude`, `gemini`, etc.).
    *   For each provider, it will construct a mount spec:
        *   **Host Source Path:** `~/.reactor/{resolved.Account}/{provider.Mounts.Source}`. The path must be resolved to an absolute path on the host.
        *   **Container Target Path:** `{provider.Mounts.Target}`.
    *   These mount specs will be added to the `blueprint.Mounts` slice, in addition to the project workspace mount.
    *   **Important:** The host source directories (e.g., `~/.reactor/work-account/claude`) do **not** need to exist at runtime. Docker will automatically create them on the host if they are missing when the container starts. This simplifies the logic, as we don't need to pre-create them.

### **2.3. Phased implementation plan**

This can be implemented as two separate, sequential PRs.

*   **PR 1: Implement Account-Based Secret Management**
    *   [ ] In `pkg/core/blueprint.go`, update `NewContainerBlueprint` to iterate through the `BuiltinProviders` and construct the appropriate volume mounts based on the `resolved.Account`.
    *   [ ] Ensure the host path is correctly resolved to an absolute path.
    *   [ ] Add an integration test with a fixture that sets `customizations.reactor.account`. The test must create a dummy credential file on the host, run `reactor up`, and verify the file exists at the correct location inside the container.

*   **PR 2: Implement Optional AI Entrypoint**
    *   [ ] In `pkg/config/service.go`, ensure `defaultCommand` is correctly passed to `ResolvedConfig`.
    *   [ ] In `pkg/core/blueprint.go`, update `NewContainerBlueprint` to use `resolved.DefaultCommand` for the container's entrypoint, falling back to a shell if it's not set.
    *   [ ] Add an integration test with a fixture that sets `defaultCommand` to `["echo", "hello reactor"]`. The test will run `reactor up` and assert that the container output is "hello reactor" and that it exits gracefully (i.e., does not start an interactive session).

### **2.4. Testing strategy**

*   **Unit Tests:**
    *   Add unit tests for the blueprint creation logic to verify that the mount paths are constructed correctly for different account names.
    *   Add unit tests to verify the container command is set correctly based on `defaultCommand`.
*   **Integration Tests:**
    *   **Account Mount Test:**
        1.  Create a test-specific account directory on the host (e.g., `/tmp/test-home/.reactor/test-account/claude`).
        2.  Place a file (`test-creds.txt`) inside it.
        3.  Use a `devcontainer.json` that specifies `"account": "test-account"`.
        4.  Run `reactor up`.
        5.  Use `docker exec` to verify that `/home/claude/.claude/test-creds.txt` exists inside the container.
    *   **Default Command Test:**
        1.  Use a `devcontainer.json` that specifies `"defaultCommand": "echo 'init command successful'"`
        2.  Run `reactor up` and capture the container's output.
        3.  Assert that the output contains "init command successful".
        4.  Assert that `reactor up` exits with code 0 after the command completes.

## **3. The 'What Ifs': Risks & Mitigation**

*   **Risk:** A user specifies an account in `devcontainer.json`, but the corresponding directory does not exist on the host.
*   **Mitigation:** This is not an error condition. Docker's bind mount functionality will automatically create the directory on the host with root ownership. This is acceptable default behavior. Our documentation should explain how to pre-populate these directories.
*   **Risk:** A user specifies an invalid `defaultCommand` that doesn't exist in the container's `PATH`.
*   **Mitigation:** The container will fail to start, and the Docker error message will be streamed to the user. This is standard, expected behavior, and `reactor` does not need to add extra validation for it.

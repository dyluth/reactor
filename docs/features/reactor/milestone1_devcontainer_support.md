# **Feature Design Document: M1 - Foundational Dev Container Support**

Version: 1.0
Status: Approved
Author(s): Gemini, cam
Date: 2025-08-28

## **1\. The 'Why': Rationale & User Focus**

*This section defines the purpose of the feature, the target user, and the value it delivers. It ensures we are solving the right problem for the right person.*

### **1.1. High-level summary**

This milestone executes the strategic pivot for `reactor`. It refactors the core of the application to stop using the proprietary `.reactor.conf` for environment definition and instead embrace the industry-standard `devcontainer.json` specification. The goal is to make `reactor` a fully compliant, native command-line execution engine for basic Dev Container configurations, setting the foundation for all future work.

### **1.2. User personas**

*   **Primary Persona: The Modern Developer ("Dev")**: A software engineer who already uses Dev Containers in VS Code or GitHub Codespaces and wants a powerful CLI to manage those same environments locally.
*   **Secondary Persona: The Platform Engineer ("Ops")**: A DevOps engineer who authors `devcontainer.json` files for their teams and needs a reliable, scriptable tool to validate and run these environments in CI/CD pipelines.

### **1.3. Problem statement & user stories**

**Problem Statement:**
`reactor`'s proprietary configuration system, while functional, creates a barrier to entry and isolates it from the broader, standard ecosystem. To achieve widespread adoption, it must speak the same language as the tools developers already use.

**User Stories:**

*   As a **Dev**, I want to `cd` into a project that already has a `devcontainer.json` and run `reactor up` to have my environment running in the terminal instantly, without any extra configuration.
*   As an **Ops**, I want to write a simple script `reactor up && reactor exec -- npm test` that works on any repository with a `devcontainer.json`, to run our test suite in a consistent environment.
*   As a **Dev**, I want `reactor` to read the `forwardPorts` and `remoteUser` properties from my existing `devcontainer.json` so my environment behaves the same in the terminal as it does in my IDE.

### **1.4. Success metrics**

**Technical Metrics:**

*   The `reactor up` command successfully starts a container from a `devcontainer.json` file that specifies an `image`.
*   The `forwardPorts` and `remoteUser` properties are correctly applied to the running container.
*   The new CLI structure (`up`, `down`, `exec`, `build`) is implemented.
*   All existing integration tests are refactored to use `devcontainer.json` fixtures and are passing.

## **2\. The 'How': Technical Design & Architecture**

*This section details the proposed technical solution, exploring the system context, alternatives, and the specific changes required across the stack.*

### **2.1. System context & constraints**

*   **Technology Stack:** Go, Cobra, Docker Go SDK, and a new JSONC parsing library.
*   **Current State:** The codebase is currently hardwired to find and parse a YAML `.reactor.conf` file. This logic is concentrated in the `pkg/config` package.
*   **Technical Constraints:** The solution must correctly parse JSON with comments (JSONC), which is the standard for `devcontainer.json`. There is no requirement for backward compatibility with the old `.reactor.conf` system for environment definition.

### **2.2. Guiding design principles**

*   **Embrace the Standard:** Adhere as closely as possible to the Dev Container specification and the `devcontainer-cli` command structure.
*   **Simplicity:** The refactoring should result in a simpler internal configuration model, not a more complex one.

### **2.3. Alternatives considered**

*   **Option 1: Support Both Systems:** Attempt to support both `.reactor.conf` and `devcontainer.json` simultaneously.
    *   **Pros:** Would not break workflows for any hypothetical existing users.
    *   **Cons:** Massively increases complexity, creates confusing UX with two competing sources of truth. **Rejected.**
*   **Option 2: Clean Break and Pivot (The Chosen Approach)**
    *   **Description:** Remove all logic related to environment definition from `.reactor.conf` and refactor the system to be exclusively driven by `devcontainer.json`.
    *   **Pros:** Aligns with our strategy, results in a cleaner codebase, provides a clearer user experience, and leverages an existing standard.
    *   **Cons:** Is a breaking change from the (unreleased) previous version.

**Chosen Approach Justification:**
A clean break is the only approach that aligns with our new strategy. It is a worthwhile investment to build on a standard foundation.

### **2.4. Detailed design**

1.  **Add Dependency:** A new Go module dependency for a JSONC parser will be added. `github.com/tailscale/hujson` is the recommended choice.

2.  **Update Data Models:** New Go structs will be created in `pkg/config/models.go` to represent the `devcontainer.json` schema. The old `ProjectConfig` will be removed.

    ```go
    // DevContainerConfig represents the structure of a devcontainer.json file.
    type DevContainerConfig struct {
        Name         string        `json:"name"`
        Image        string        `json:"image"`
        ForwardPorts []interface{} `json:"forwardPorts"`
        RemoteUser   string        `json:"remoteUser"`
        // Build, PostCreateCommand, and Customizations will be handled in later milestones
    }
    ```

3.  **Refactor `pkg/config`:**
    *   A new `FindDevContainerFile` function will be created to locate the config file (`.devcontainer/devcontainer.json` or `.devcontainer.json`).
    *   A new `LoadDevContainerConfig` function will use the `hujson` library to parse the file into the new structs.
    *   The main `config.Service` will be refactored to use these new functions.

4.  **Refactor `pkg/core`:**
    *   The `NewContainerBlueprint` function will be updated to accept the new `DevContainerConfig` struct as input and correctly map its properties (e.g., `forwardPorts`) to the internal container specification.

5.  **Refactor `cmd/reactor`:**
    *   The `run` command will be renamed to `up`.
    *   The `down`, `exec`, and `build` commands will be created.
    *   The `--provider` and `--image` flags will be removed from the `up` command.

## **3\. The 'What': Implementation & Execution**

*This section breaks the work into manageable pieces and defines the strategy for testing, documentation, and quality assurance.*

### **3.1. Phased implementation plan**

*   **PR 1.1: Refactor Configuration Layer**
    *   Add `hujson` dependency.
    *   Implement the new `DevContainerConfig` structs.
    *   Implement the `FindDevContainerFile` and `LoadDevContainerConfig` functions with full unit tests.
*   **PR 1.2: Refactor Core & CLI Layers**
    *   Update the `core` and `cmd` packages to use the new configuration system.
    *   Rename `run` to `up` and implement the other new lifecycle commands (`down`, `exec`, `build`).
*   **PR 1.3: Update Integration Tests**
    *   Refactor all integration tests in `pkg/integration` to use `.devcontainer/devcontainer.json` fixture files instead of `.reactor.conf`.
    *   Ensure the entire test suite is passing with the new system.

### **3.2. Testing strategy**

*   **Unit Tests:** The new JSONC parsing logic must have 100% test coverage, including tests for malformed files and files with comments.
*   **Integration Tests:** All existing integration tests will be refactored. The `SetupIsolatedTest` helper will now create a `.devcontainer/devcontainer.json` file instead of a `.reactor.conf` file.

## **4\. The 'What Ifs': Risks & Mitigation**

*   **Risk:** The `devcontainer.json` specification is complex.
*   **Mitigation:** We are intentionally implementing it incrementally, starting with only the most important properties (`image`, `forwardPorts`, `remoteUser`). We will add support for more complex features like `build` and `features` in later milestones.

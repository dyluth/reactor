# **Feature Design Document: `Reactor` - The Developer Environment Tool**

Version: 1.0  
Status: Draft  
Author(s): Gemini, cam  
Date: 2025-08-23

## **1\. The 'Why': Rationale & User Focus**

*This section defines the purpose of the feature, the target user, and the value it delivers. It ensures we are solving the right problem for the right person.*

### **1.1. High-level summary**

`Reactor` is a command-line tool that provides developers with a simple, fast, and reliable way to create isolated, containerized development environments. It allows users to run AI CLI tools (like Claude, Gemini, etc.) in a clean, project-specific container with a single command, ensuring environment consistency, preventing dependency conflicts, and managing all AI-related configurations in a persistent, isolated manner.

### **1.2. User personas**

*   **Primary Persona: The AI-Powered Developer ("Dev")**: A software engineer who uses various AI CLI tools as an integral part of their daily workflow (e.g., for code generation, debugging, and analysis). They often work on multiple projects with different tech stacks. They value speed, environment consistency, and keeping their host machine clean.

*   **Secondary Persona: The Tool Onboarder ("Ops")**: A developer or DevOps engineer responsible for integrating a new AI tool or agent into their team's workflow. They value clear diagnostics and predictable configuration paths. They need to understand an agent's configuration footprint to create standardized environments.

### **1.3. Problem statement & user stories**

**Problem Statement:**
Developers using multiple AI CLI tools face significant friction managing inconsistent environments, conflicting dependencies, and scattered authentication tokens. This complexity slows down their workflow and creates a barrier to adopting new AI tools. Furthermore, it's difficult to determine the exact set of configuration files a new agent creates, making it hard to create reproducible environments.

**User Stories:**

*   As a **Dev**, I want to run my AI agent in a clean, containerized environment with a single command, so that I can start working on my project code immediately without manual setup.
*   As a **Dev**, I want to switch between different projects, accounts, and AI providers seamlessly, so that my configurations and authentication tokens are automatically managed and don't conflict.
*   As an **Ops**, I want to run a new agent in a "discovery mode" with no file mounts, so that I can see exactly what configuration files it creates and where, allowing me to define a proper mounting strategy.

### **1.4. Success metrics**

**Business Metrics:**

*   **Adoption:** Achieve 100 stars on GitHub and receive contributions from at least 5 non-core developers within 6 months of public launch.

**Technical Metrics:**

*   **Velocity:** A developer can go from a clean project directory to an interactive AI session in under 90 seconds for a new container, and under 3 seconds for a recovered container.
*   **Reliability:** The tool achieves a 99.9% success rate for container creation and session management, with graceful error handling for Docker daemon issues.

## **2\. The 'How': Technical Design & Architecture**

*This section details the proposed technical solution, exploring the system context, alternatives, and the specific changes required across the stack.*

### **2.1. System context & constraints**

*   **Technology Stack:** Go, Cobra CLI Framework, Docker Go SDK.
*   **Current State:** This is a new tool being built as part of a monorepo, refactored from the original `claude-reactor` project. It will leverage shared code from a `pkg/` directory but will be a standalone binary.
*   **Technical Constraints:** The tool must run on macOS and Linux (amd64/arm64). It depends on the user having a running Docker daemon installed on their host machine.
*   **Distribution:** The final output will be a single, standalone binary with build-time metadata (version, commit, date) injected via linker flags to aid in debugging.

### **2.2. Guiding design principles**

*   **Simplicity over Complexity (YAGNI):** The tool will not have "smart" features like project auto-detection. The user is in explicit control.
*   **Consistency with Existing Code:** The tool will follow the modular, layered architecture defined in the Project Charter.
*   **Clarity and Readability:** The codebase will be structured into clear, single-responsibility components (`cmd`, `core`, `docker`, `session`) to be easily understood and maintained.

### **2.3. Alternatives considered**

*   **Option 1: Managed State Directories (The Chosen Approach)**
    *   **Description:** `Reactor` actively creates and manages isolated state directories under `~/.reactor/` for each account/provider/project combination. These managed directories are then mounted into the container.
    *   **Pros:** Provides perfect isolation, prevents configuration conflicts, solves the core user problem completely.
    *   **Cons:** More complex to implement than a simple passthrough.

*   **Option 2: Simple Mount Passthrough (The Simplest Possible Approach)**
    *   **Description:** `Reactor` would simply act as a wrapper to mount existing host directories (e.g., `~/.claude`, `~/.config/gcloud`) directly into the container.
    *   **Pros:** Very fast to implement, easy to understand.
    *   **Cons:** Does not solve the core problem of configuration conflicts between different accounts or projects. The user would have to manage this manually.

**Chosen Approach Justification:**
Option 1 was chosen because it directly solves the user story of seamlessly switching between projects and accounts without configuration conflicts. While more complex, this approach provides far more value and is essential for a smooth developer experience, which is a key project goal.

### **2.4. Detailed design**

The internal architecture is composed of several distinct, isolated components:

**1. `cmd` (CLI Layer)**
   - **Responsibility**: Parses user commands (`run`, `diff`, `accounts`) and flags (`--image`, `--discovery-mode`). Delegates to the `core` layer.

**2. `core` (Orchestration Layer)**
   - **Responsibility**: The brain of the application. It uses services for Config, State, and Images to produce a complete `ContainerBlueprint` that defines everything needed to launch the container.

**3. `docker` (Container Provisioning Layer)**
   - **Responsibility**: Executes all interactions with the Docker daemon. This includes a robust container recovery strategy: 1. Check for a running container with the expected deterministic name. 2. If not found, check for a stopped container with that name and attempt to restart it. 3. Only create a new container if no existing one can be recovered. It is also responsible for running `docker diff` for discovery mode.

**4. `session` (Terminal Interaction Layer)**
   - **Responsibility**: Manages the interactive TTY connection to the running container's process.

**5. `entrypoint` (In-Container Client)**
   - **Responsibility**: A separate utility inside the container that acts as the entrypoint, deciding whether to connect to `Reactor-Fabric` or run the native AI tool.

#### **2.4.1. Data model updates**

Configuration uses a simple project-based YAML file with built-in provider mappings.

**`<project-dir>/.reactor.conf`:**
```yaml
provider: claude    # claude, gemini, or custom
account: default    # account name for configuration isolation
image: python       # base, python, go, or custom image URL
danger: false       # enable dangerous permissions (optional, defaults to false)
```

**Built-in provider mappings (in code):**
```go
type ProjectConfig struct {
    Provider string `yaml:"provider"` // claude, gemini, or custom
    Account  string `yaml:"account"`  // account name for isolation
    Image    string `yaml:"image"`    // base, python, go, or custom image URL
    Danger   bool   `yaml:"danger,omitempty"` // enable dangerous permissions
}

// Built-in providers with mount paths
var BuiltinProviders = map[string]ProviderInfo{
    "claude": {
        DefaultImage: "base",
        Mounts: []MountPoint{
            {Source: "claude", Target: "/home/claude/.claude"},
            // Additional mounts can be added for claude
        },
    },
    "gemini": {
        DefaultImage: "base",
        Mounts: []MountPoint{
            {Source: "gemini", Target: "/home/claude/.gemini"},
            // Additional mounts can be added for gemini
        },
    },
}
```

**Account directory structure:**
```
~/.reactor/
└── <account>/           # e.g., cam (system username), work-account
    └── <project-hash>/  # first 8 chars of project path hash
        ├── claude/      # mounted to /home/claude/.claude
        ├── gemini/      # mounted to /home/claude/.gemini  
        └── openai/      # mounted to /home/claude/.openai
```

#### **2.4.2. Data migration plan**

N/A. This is a new tool.

#### **2.4.3. API & backend changes**

N/A. This is a client-side CLI tool.

#### **2.4.4. Frontend changes**

N/A. This is a CLI tool.

### **2.5. Non-functional requirements (NFRs)**

*   **Performance:** P99 latency for `reactor run` (from command execution to interactive session) must be under 90 seconds for a new container and under 3 seconds for a recovered container.
*   **Reliability:** The application must handle Docker daemon connection errors gracefully and provide clear, actionable error messages to the user. All Docker SDK calls must be wrapped in a `context.WithTimeout` (e.g., 30-60 seconds) to prevent the application from hanging.
*   **Operations & Developer Experience:** All common development and testing tasks will be automated via a `Makefile`. The developer onboarding time from `git clone` to a running local instance must be under 10 minutes.

## **3\. The 'What': Implementation & Execution**

*This section breaks the work into manageable pieces and defines the strategy for testing, documentation, and quality assurance.*

### **3.1. Phased implementation plan**

**Phase 1: Core Scaffolding & Config**
*   \[ \] PR 1.1: Set up Cobra CLI structure for all commands (`run`, `diff`, `accounts`, `config`).
*   \[ \] PR 1.2: Implement the `core.ConfigService` to load project config and manage account directories.

**Phase 2: Container Provisioning**
*   \[ \] PR 2.1: Implement the `docker` layer to create, start, and stop a basic container, incorporating the full recovery logic.
*   \[ \] PR 2.2: Implement the `core.StateService` to manage the `~/.reactor/` state directories.
*   \[ \] PR 2.3: Integrate the services so `reactor run` can launch a container with the correct state mounts.

**Phase 3: Interactive Session**
*   \[ \] PR 3.1: Implement the `session` layer to handle TTY attachment to the running container's process.

**Phase 4: Advanced Features & Docs**
*   \[ \] PR 4.1: Implement the `--discovery-mode` flag and the `reactor diff` command.
*   \[ \] PR 4.2: Implement the `--docker-host-integration` flag.
*   \[ \] PR 4.3: Create user-facing documentation for all features.

**Phase 5: Auto-Installation & Enhanced UX**
*   \[ \] PR 5.1: Implement automatic AI agent installation in containers that don't have the specified agent pre-installed.
*   \[ \] PR 5.2: Add interactive configuration flow as alternative to `reactor config init`.

**Phase 6: Integrated In-Session Automation (Future)**
*   [ ] PR 6.1: Introduce a new `automations.yaml` configuration to define rule-based automations (e.g., regex triggers, text-injection actions).
*   [ ] PR 6.2: Add an `automation` key to `.reactor.conf` to allow projects to persist their desired automation setting.
*   [ ] PR 6.3: Enhance `reactor run` to act as a process manager, spawning a managed "automaton" process that sits between the user and the container to apply the configured automation rules.


### **3.2. Testing strategy**

*   **Unit Tests:** Each service in the `core` layer will be unit-tested with mocked dependencies. The `docker` layer will be tested by mocking the Docker SDK.
*   **Integration Tests:** Test the interaction between the `core` services. Test the full `reactor run` command against a live Docker daemon, specifically testing the container recovery logic.
*   **End-to-End (E2E) User Story Tests:**
    *   **Dev Story:** A test script will `cd` into a test project, run `reactor run --image <image> --account test`, verify the container starts, and then run `reactor clean` and verify the container is removed.
    *   **Ops Story:** A test script will run an agent in `--discovery-mode`, create a file inside the container, end the session, run `reactor diff`, and verify the new file is reported.

## **4\. The 'What Ifs': Risks & Mitigation**

*This section addresses potential issues, ensuring the feature is secure, reliable, and can be deployed and managed safely.*

### **4.1. Security & privacy considerations**

*   **Authentication & Authorization:** This tool does not have its own auth layer. It relies on the security of the underlying AI provider's configuration files, which it isolates.
*   **Docker Host Integration (`--docker-host-integration`):** This is the primary security risk. Using this flag mounts the host's Docker socket into the container, giving it full host-level Docker daemon access (not Docker-in-Docker). The documentation must clearly warn users to only use this with trusted images.

### **4.2. Rollout & deployment**

*   **Feature Flags:** N/A. New commands will be added directly.
*   **Monitoring & Observability:** The tool will use structured logging (e.g., Logrus). A `--verbose` flag will enable DEBUG level logging for troubleshooting.
*   **Rollback Plan:** N/A for the initial release. Subsequent releases will be managed via versioned binaries on GitHub Releases.

### **4.3. Dependencies and integrations**

*   **External Dependencies:** The tool has a hard dependency on a running Docker daemon on the host machine.

### **4.4. Cost and resource analysis**

*   **Infrastructure Costs:** N/A. The tool runs entirely on the user's local machine.

### **4.5. Container Images & Configuration**

**Built-in Images:**
Reactor provides three curated images with convenient short names:

* **`base`**: Core development tools + AI agents (Claude, Gemini)
  * Tools: curl, git, ca-certificates, wget, unzip, gnupg2, socat, sudo, ripgrep, jq, fzf, nano, vim, less, procps, htop, build-essential, shellcheck, man-db, node, npm
  * AI Agents: Claude CLI, Gemini CLI pre-installed
  * Image: `ghcr.io/dyluth/claude-reactor-go`

* **`python`**: Base image + Python development environment  
  * Additional: python3, python3-pip, uv, uvx and Python toolchain
  * Image: `ghcr.io/dyluth/claude-reactor-go`

* **`go`**: Base image + Go development environment
  * Additional: Go toolchain with essential Go development tools
  * Image: `ghcr.io/dyluth/claude-reactor-go`

**Custom Images:**
Users can specify any Docker image. The image must:
- Have a `claude` user (or compatible user setup)
- Support the AI agent specified in the provider configuration
- Be compatible with the mounting strategy for agent configuration

**Account-Based Configuration:**
Configuration uses an account-based directory structure under `~/.reactor/<account>/` where each provider gets its own subdirectory. AI agents manage their own config files through their setup wizards when the directories are properly mounted.

### **4.6. Open questions & assumptions**

*   **Assumptions:**
    *   Users have Docker installed and have sufficient permissions to interact with the Docker daemon.
    *   Users are comfortable with the command line and the basic concepts of Docker containers.
    *   AI agents will properly initialize their configuration when their expected config directories are mounted from the host.
# **Feature Design Document: Milestone 1 - Namespaced Core Services**

**Version: 1.0**  
**Status: Draft**  
**Author(s): Gemini, cam**  
**Date: 2025-09-03**

## **1\. The 'Why': Rationale & User Focus**

*This section defines the purpose of the feature, the target user, and the value it delivers. It ensures we are solving the right problem for the right person.*

### **1.1. High-level summary**

This milestone delivers the foundational runtime for `reactor-fabric`. Its core purpose is to provide a simple, robust command-line interface (CLI) for developers and operators to manage the lifecycle of `reactor-fabric` instances. It delivers the essential `start`, `down`, and `list` commands to stand up the core services (Orchestrator, Redis) in an isolated, namespaced manner, making the system testable and usable for further development.

### **1.2. User personas**

*   **Primary Persona: The Agent Developer (Dev)**: Needs a reliable way to spin up the core `reactor-fabric` engine to test the agents they are developing. They value speed, command-line accessibility, and isolation between different test runs.
*   **Secondary Persona: The Platform Operator (Ops)**: Needs a scriptable way to manage the lifecycle of multiple `reactor-fabric` instances on a shared server. They value stability, resource isolation, and clear status reporting.

### **1.3. Problem statement & user stories**

**Problem Statement:**
Before any agent orchestration can happen, a user needs a simple, reliable way to start, stop, and manage the lifecycle of the core `reactor-fabric` engine and its dependencies. This process must be namespaced to prevent resource conflicts between different projects or test runs on a single machine.

**User Stories:**

*   As a Dev, I want to run `reactor-fabric start --name my-test`, so that I can get a clean instance of the engine running and begin developing my agent.
*   As a Dev, I want to run `reactor-fabric down --name my-test`, so that I can completely remove all resources associated with my test instance and ensure a clean state for the next run.
*   As an Ops, I want to run `reactor-fabric list`, so that I can see all the running `reactor-fabric` instances on the machine and monitor resource usage.

### **1.4. Success metrics**

**Business Metrics:**

*   N/A for this foundational, technical-enablement milestone.

**Technical Metrics:**

*   P99 latency for `start`, `down`, and `list` commands is under 15 seconds.
*   `reactor-fabric start` successfully launches namespaced Orchestrator and Redis containers on a shared, namespaced Docker network.
*   `reactor-fabric down` successfully removes all containers and networks associated with a given namespace, leaving no orphaned resources.
*   `start` command fails with a clear error message and non-zero exit code if an instance with the same name is already running.
*   `down` command reports success and exits with a zero exit code if the specified instance does not exist.
*   All commands fail with a user-friendly error message if the Docker daemon is not available.

## **2\. The 'How': Technical Design & Architecture**

*This section details the proposed technical solution, exploring the system context, alternatives, and the specific changes required across the stack.*

### **2.1. System context & constraints**

*   **Technology Stack:** Go, Cobra (for CLI), Docker Go SDK, Docker Engine.
*   **Current State:** This is a net-new feature. It will create the `cmd/fabric/` application entrypoint and may create a new `pkg/fabric/runtime` package to encapsulate the core logic.
*   **Technical Constraints:** Must run on any system with a compatible Docker Engine installed. Must not interfere with other Dockerized applications running on the host.

### **2.2. Guiding design principles**

*   **Simplicity over Complexity:** The CLI commands and flags should be simple and intuitive, mirroring patterns from established tools like `docker-compose`.
*   **Consistency with Existing Code:** Naming and labeling of Docker resources must be consistent and predictable to allow for reliable, atomic management of instance resources.
*   **Clarity and Readability:** The Go code should be easy to understand, with a clear separation between the command-line layer and the runtime-management logic.

### **2.3. Alternatives considered**

*   **Option 1: Shell Scripts**
    *   **Description:** Use a set of bash scripts to wrap `docker` commands.
    *   **Pros:** Very fast to write for basic cases.
    *   **Cons:** Not portable (e.g., Windows compatibility), difficult to test, poor error handling, does not scale with future complexity.

*   **Option 2: Go Application (Chosen)**
    *   **Description:** Use the official Docker Go SDK to programmatically create and manage all Docker resources.
    *   **Pros:** Robust, portable, easily testable, and provides superior error handling and logging. It creates a stable foundation for future milestones.
    *   **Cons:** More initial development effort than a simple script.

**Chosen Approach Justification:**
The Go application approach is chosen because it provides the stable, professional, and extensible foundation required for the rest of the project's vision. It is the only viable option for a tool intended for serious development and operational use.

### **2.4. Detailed design**

#### **2.4.1. Data model updates**

N/A

#### **2.4.2. Data migration plan**

N/A

#### **2.4.3. API & backend changes**

N/A. The Orchestrator container will be a placeholder. For this milestone, its image (e.g., `alpine:latest` with a `sleep` command) will be hardcoded within the `reactor-fabric` binary to ensure simplicity. Configuration of this image will be handled in a future milestone.

#### **2.4.4. Frontend changes**

N/A. This is a CLI-only feature.

### **2.5. Non-functional requirements (NFRs)**

*   **Performance:** Core commands (`start`, `down`, `list`) must complete in under 15 seconds.
*   **Reliability:** The `down` command must reliably clean up all resources to prevent resource leakage.
*   **Operations & Developer Experience:** The developer onboarding time from `git clone` to being able to run `make build` and `reactor-fabric start` must be under 10 minutes.

## **3\. The 'What': Implementation & Execution**

*This section breaks the work into manageable pieces and defines the strategy for testing, documentation, and quality assurance.*

### **3.1. Phased implementation plan**

**Phase 1: Cobra CLI Scaffolding**

*   [ ] PR 1.1: Create `cmd/fabric/main.go`.
*   [ ] PR 1.2: Implement the `start`, `down`, and `list` commands using the Cobra library.
*   [ ] PR 1.3: Add flag parsing for `--name` (required) and `--config`.
*   [ ] PR 1.4: Ensure the CLI commands gracefully handle and display the specific errors returned from the runtime package.

**Phase 2: Docker Runtime Logic**

*   [ ] PR 2.1: Create a new `pkg/fabric/runtime` package.
*   [ ] PR 2.2: Implement `Start(name, configPath)` function that uses the Docker Go SDK to create the network and containers with appropriate labels.
*   [ ] PR 2.3: Implement `Down(name)` function.
*   [ ] PR 2.4: Implement `List()` function.
*   [ ] PR 2.5: Implement robust error handling in the runtime package for common cases (instance exists, instance not found, Docker daemon unavailable).

### **3.2. Testing strategy**

*   **Unit Tests:**
    *   Test the Docker resource naming and labeling logic.
    *   Test the `fabric.yml` config loader to ensure it correctly checks for file existence without parsing content.
*   **Integration Tests:** (Requires a live Docker daemon)
    *   Create a test suite that runs the full `start` -> `list` -> `down` lifecycle.
    *   Verify that containers are created with the correct names, labels, and network settings using `docker inspect`.
    *   Verify that `down` performs a complete cleanup.
    *   Test the defined error scenarios: `start` on an existing instance, `down` on a non-existent instance, and running a command when the Docker daemon is stopped. Verify both the error messages and exit codes.
*   **End-to-End (E2E) User Story Tests:**
    *   **User Story 1 (`start`):** An integration test will execute the compiled binary with `start` and verify via the Docker SDK that the expected resources are running.
    *   **User Story 2 (`down`):** An integration test will execute the compiled binary with `down` and verify via the Docker SDK that all resources have been removed.

## **4\. The 'What Ifs': Risks & Mitigation**

*This section addresses potential issues, ensuring the feature is secure, reliable, and can be deployed and managed safely.*

### **4.1. Security & privacy considerations**

The Orchestrator container will eventually require access to the host's Docker socket. This is a significant security consideration that grants root-level access to the host. For Milestone 1, the placeholder container does **not** require this access. This risk will be formally addressed in the design for the milestone that implements agent scaling.

### **4.2. Rollout & deployment**

*   **Deployment:** This is a CLI tool. Deployment will be managed via GitHub Releases, with compiled binaries provided for major operating systems and architectures.
*   **Monitoring:** N/A for the CLI itself. Monitoring will apply to the services it launches in later milestones.
*   **Rollback Plan:** N/A for the CLI. Users can download and use older versions from GitHub Releases if needed.

### **4.3. Dependencies and integrations**

*   **External Dependencies:** Requires a running Docker Engine on the host machine. The CLI **must** detect if it cannot connect to the Docker daemon and provide a clear, user-friendly error message (e.g., "Error: Cannot connect to the Docker daemon. Is it running?") instead of a raw network or SDK error.

### **4.4. Cost and resource analysis**

*   **Infrastructure Costs:** N/A. All costs are local CPU/memory/disk resources for running Docker containers.

### **4.5. Open questions & assumptions**

*   **Assumption:** The user running the `reactor-fabric` command has sufficient permissions to interact with the Docker daemon on their host machine.
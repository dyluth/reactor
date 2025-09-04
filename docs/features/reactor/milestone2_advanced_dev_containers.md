# **Feature Design Document: M2 - Advanced Dev Container Features**

Version: 1.1
Status: Approved
Author(s): Gemini, cam
Date: 2025-09-02

## **1. The 'Why': Rationale & User Focus**

### **1.1. High-level summary**

This feature elevates `reactor` from a tool that only runs pre-built images to a true Dev Container engine capable of building environments from source. It implements support for two core `devcontainer.json` properties: `build` (for Dockerfile-based builds) and `postCreateCommand` (for setup and initialization hooks). This allows `reactor` to provision fully custom, project-specific environments automatically, which is a fundamental requirement for most real-world development workflows.

### **1.2. User personas**

*   **Primary Persona: The Modern Developer ("Dev")**: Needs their development environment to contain project-specific dependencies and be fully initialized after creation. They expect `reactor` to handle the `Dockerfile` build and run setup scripts (`npm install`, `bundle install`, etc.) automatically.
*   **Secondary Persona: The Platform Engineer ("Ops")**: Authors `devcontainer.json` files that define a complete, reproducible environment, including the Dockerfile build process and all initialization logic. They need `reactor` to correctly interpret these properties to ensure consistency across all developers and CI systems.

### **1.3. Problem statement & user stories**

**Problem Statement:**
Currently, `reactor` can only run pre-existing container images. It lacks the ability to build a project's custom Dockerfile or run setup commands after the container is created. This is a major feature gap that prevents its use for any project requiring more than a generic, pre-built environment.

**User Stories:**

*   As a **Dev**, when I run `reactor up` in a project with a `Dockerfile`, I want `reactor` to automatically build that Dockerfile and start my session in the resulting custom image, so my environment has all the necessary tools and dependencies.
*   As an **Ops**, I want to specify a `postCreateCommand` in `devcontainer.json` (e.g., `"npm install"`), so that `reactor up` will automatically run it, providing every developer with a fully initialized, ready-to-code project.
*   As a **Dev**, I want the `reactor build` command to build my dev container image without starting it, so I can pre-build my environment or use it in CI pipelines.

### **1.4. Success metrics**

**Technical Metrics:**

*   `reactor up` successfully executes a `docker build` when the `build.dockerfile` property is present in `devcontainer.json`.
*   `reactor up` successfully executes the command(s) specified in `postCreateCommand` inside the container after it is created.
*   The `reactor build` command successfully builds the image and exits without starting a container.
*   All new logic is covered by unit and integration tests.

## **2. The 'How': Technical Design & Architecture**

### **2.1. System context & constraints**

*   **Technology Stack:** Go, Cobra, Docker Go SDK.
*   **Current State:** The application can parse a `devcontainer.json` file and start a container from a pre-built `image`. The `DevContainerConfig` struct already contains fields for `Build` and `PostCreateCommand`, but they are not yet plumbed through to the `ResolvedConfig` or used by the core logic.
*   **Technical Constraints:** The implementation must use the Docker Go SDK for all Docker operations, not shell out to the `docker` CLI.

### **2.2. Guiding design principles**

*   **Embrace the Standard:** The implementation of `build` and `postCreateCommand` must adhere to the behavior defined in the Dev Container specification.
*   **Provide Clear Feedback:** Building images and running commands can be slow and produce a lot of output. The user must receive clear, real-time feedback during these processes.

### **2.3. Alternatives considered**

**Option 1: Enhance `docker.Service` and `up` Command (Chosen Approach)**
*   **Description:** Add `BuildImage` and `ExecInContainer` methods to the existing `docker.Service`. The logic within the `up` command will be expanded to conditionally call these new methods based on the contents of the resolved `devcontainer.json`.
*   **Pros:** Keeps all Docker-related logic within the `pkg/docker` package. It's a clean extension of our existing architecture.
*   **Cons:** Makes the `up` command handler more complex, as it has to orchestrate more steps.

**Option 2: Shelling Out to Docker CLI**
*   **Description:** Use `os/exec` to call `docker build` and `docker exec` directly from the `up` command.
*   **Pros:** Might seem faster to implement initially.
*   **Cons:** It's less robust, less secure, and makes it much harder to manage I/O streams, handle errors, and test the logic. It violates our principle of using the official SDK. **Rejected.**

**Chosen Approach Justification:**
Enhancing our existing `docker.Service` is the only approach that maintains architectural integrity and provides the robust error handling and control needed for a production-quality tool.

### **2.4. Detailed design**

#### **2.4.1. Data Model & Config Updates**

1.  **`pkg/config/models.go`**: Both structs must be updated to include the build and lifecycle hook information.
    *   **DevContainerConfig**: Change `PostCreateCommand string` to `PostCreateCommand interface{}` (required to handle both `string` and `[]string` from JSON per Dev Container spec).
    *   **ResolvedConfig**: Add `Build *Build` and `PostCreateCommand interface{}` fields.
2.  **`pkg/config/service.go`**: The mapping function that transforms `DevContainerConfig` to `ResolvedConfig` must be updated to copy these new fields across.

**Path Resolution Logic**: 
- Build context path is relative to the directory containing devcontainer.json
- Dockerfile path is relative to the resolved context directory
- Example: devcontainer.json at `/project/.devcontainer/devcontainer.json` with `{"build": {"context": "..", "dockerfile": "Dockerfile"}}` resolves to context=`/project/` and dockerfile=`/project/Dockerfile`

#### **2.4.2. Docker Service Enhancements (`pkg/docker`)**

1.  **`BuildSpec` Struct:** Create a new `BuildSpec` struct to pass build parameters.
    ```go
    type BuildSpec struct {
        Dockerfile string
        Context    string
        ImageName  string // The name to tag the built image with
    }
    ```
2.  **New Methods in `docker.Service`:**
    *   `BuildImage(ctx context.Context, spec BuildSpec) error`: This method will use the Docker Go SDK to build an image. It must stream the build output with ANSI colors preserved to the user's console in real-time. **Image reuse**: Check if target image exists locally first; only build if not found.
    *   `ExecInContainer(ctx context.Context, containerID string, command interface{}) error`: This method will execute a command in a running container. **Command format handling**: 
        - `string` commands (e.g., `"npm install && npm test"`) executed via shell: `["/bin/sh", "-c", "command"]`
        - `[]string` commands (e.g., `["npm", "install"]`) executed directly without shell
        - Stream stdout/stderr in real-time, preserve ANSI colors
        - On failure: display clear error with exit code, leave container running for debugging

#### **2.4.3. CLI Command Logic Updates (`cmd/reactor`)**

The `up` and `build` command handlers will be updated to orchestrate the new workflow.

1.  **`build` Command (`cmd/reactor/build.go`):**
    *   This command will no longer be a placeholder.
    *   It will load the configuration. If `resolved.Build` is present, it will call `docker.Service.BuildImage`. If not, it will print an error message stating that no build configuration is defined in `devcontainer.json`.

2.  **`up` Command (enhanced `upCmdHandler`):**
    *   The workflow will be extended:
        1.  Load configuration into `ResolvedConfig`.
        2.  **If `resolved.Build` is not nil:**
            *   **Precedence Rule:** If `devcontainer.json` contains both `image` and `build` properties, the `build` property **must** take precedence.
            *   **Image Tagging:** The image **must** be tagged with the name `reactor-build:<project-hash>`, where `<project-hash>` is the existing hash for the project.
            *   **Build Logic**: Check if `reactor-build:<project-hash>` exists locally. If yes, skip build and use existing. If no, call `docker.Service.BuildImage` with the build spec.
            *   Use the image name (`reactor-build:<project-hash>`) for the subsequent container creation step.
        3.  **Else (no build step):**
            *   Use `resolved.Image` as before.
        4.  Provision and start the container.
        5.  **If `resolved.PostCreateCommand` is not nil:**
            *   Print a message to the user (e.g., "Running postCreateCommand...").
            *   Call `docker.Service.ExecInContainer` to run the command(s). **Error handling**: If it fails, terminate `reactor up` with clear error message but leave container running for debugging.
        6.  Attach to the interactive session.

**Additional Commands:**
- **`reactor build`**: Forces build even if image exists (ignores reuse logic)
- **`reactor up --rebuild`**: Forces rebuild before starting container

### **2.5. Non-functional requirements (NFRs)**

*   **Feedback:** The user must see the real-time output from both `docker build` and the `postCreateCommand`. The output should be streamed directly to the console.
*   **Error Handling:** If the Dockerfile build fails or the `postCreateCommand` returns a non-zero exit code, the `reactor up` process must terminate immediately and report the error clearly.

## **3. The 'What': Implementation & Execution**

### **3.1. Phased implementation plan**

This feature can be implemented in two distinct PRs.

*   **PR 1: Implement `build` from Dockerfile**
    *   [ ] Update `ResolvedConfig` to include the `Build` struct.
    *   [ ] Update the `config.Service` to plumb the `Build` struct through.
    *   [ ] Implement the `BuildImage` method in `pkg/docker/service.go`.
    *   [ ] Implement the `reactor build` command logic.
    *   [ ] Update the `reactor up` command to include the build logic.
    *   [ ] Add unit and integration tests for the build functionality.

*   **PR 2: Implement `postCreateCommand`**
    *   [ ] Update `ResolvedConfig` to include the `PostCreateCommand` field.
    *   [ ] Update the `config.Service` to plumb the command through.
    *   [ ] Implement the `ExecInContainer` method in `pkg/docker/service.go`, ensuring it handles both `string` and `[]string`.
    *   [ ] Add the `postCreateCommand` execution logic to the `reactor up` command.
    *   [ ] Add unit and integration tests for the `postCreateCommand` functionality.

### **3.2. Testing strategy**

*   **Unit Tests:**
    *   Add tests for the new methods in `docker.Service`, likely using mocks for the Docker client to verify that the correct SDK functions are called with the right parameters.
    *   Add tests for the `ExecInContainer` logic to ensure it correctly handles both string and slice-of-string command types.
*   **Integration Tests:**
    *   Create a new test fixture directory containing a simple project with a `devcontainer.json` that uses the `build` property and a corresponding `Dockerfile`.
    *   Write an integration test that runs `reactor up` on this fixture and verifies that:
        1.  A Docker image with the tag `reactor-build:<hash>` is built.
        2.  The container is started from the newly built image.
    *   Create another test fixture with a `postCreateCommand` that creates a specific file (e.g., `touch /tmp/post-create-was-here`).
    *   Write an integration test that runs `reactor up` on this fixture and then uses `docker exec` to verify that the file was created, proving the command ran successfully.

## **4. The 'What Ifs': Risks & Mitigation**

*   **Risk:** `postCreateCommand` can execute arbitrary code.
*   **Mitigation:** This is expected behavior and a feature of the Dev Container spec. We will not add extra security layers, but we will ensure that our documentation clearly states that users should only run projects from trusted sources.
*   **Risk:** A long-running `docker build` or `postCreateCommand` could make the `up` command appear to hang.
*   **Mitigation:** The requirement to stream all output directly to the console is the mitigation. As long as the user sees the output from the underlying process, the tool will not appear to be hung.
*   **Risk:** The `postCreateCommand` can be a string or an array of strings.
*   **Mitigation:** The design explicitly requires the data model (`interface{}`) and the `ExecInContainer` function to handle both types, and this must be verified with unit tests.

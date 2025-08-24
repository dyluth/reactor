# Vision and Strategy: Refactoring the Reactor Project

## 1. The Core Problem

The original `claude-reactor` project has evolved into a highly capable but complex system. Its complexity stems from serving two distinct user needs within a single tool:

1.  **A Developer Environment Tool**: A simple way for a developer to create a consistent, containerized, and isolated environment for their daily work.
2.  **An AI Orchestration System**: A powerful system (`Reactor-Fabric`) for orchestrating suites of specialized, containerized AI agents to perform complex, collaborative tasks.

Combining these two roles into one tool creates a steep learning curve, increases maintenance overhead, and makes the system difficult to reason about.

## 2. The Strategic Split

We propose refactoring the project into two (and potentially three) distinct, focused tools:

1.  **`Reactor`**: The new tool for **developer environments**. It will be simple, fast, and provider-agnostic (supporting Claude, Gemini, etc.). Its job is to get a developer into a clean, containerized environment with their code and preferred AI tool as quickly as possible.
2.  **`Reactor-Fabric`**: The **AI orchestration system**. This tool will focus entirely on the multi-agent use case. It will read a `suite.yaml`, manage container lifecycles, and proxy communication, but it will *not* be the primary tool for interactive development.
3.  **`Reactor-Scaffold` (Optional)**: A separate CLI for **project scaffolding**. This extracts the template-generation logic into its own command, keeping `Reactor` lean and focused on its core containerization task.

## 3. How They Interact

These tools are designed to be independent but composable:

- A developer can use `Reactor` on its own for all their daily development tasks.
- A developer can use `Reactor-Fabric` to orchestrate complex workflows. One of the services defined in its `suite.yaml` could very well be a `Reactor` instance, making it a specialized agent in a larger suite.
- A developer can use `Reactor-Scaffold` to start a new project, and then use `Reactor` to work on it.

## 4. Benefits of This Approach

-   **Simplicity**: Each tool has a clear, single purpose.
-   **Extensibility**: The provider-agnostic design of `Reactor` makes it easy to add support for new LLM CLIs.
-   **Maintainability**: Smaller, focused codebases are easier to develop, test, and debug.
-   **Flexibility**: Users can mix and match the tools as needed.

## 5. Monorepo and Code Structure

To facilitate code sharing and simplify dependency management, all of these tools will be developed within the single, existing Go repository. The project will adhere to the standard Go layout for multi-command applications.

-   **Binary Entrypoints**: Each tool will have its own `main.go` file within the `cmd/` directory:
    -   `cmd/reactor/main.go`
    -   `cmd/reactor-fabric/main.go`
    -   `cmd/reactor-scaffold/main.go`

-   **Shared Code**: Logic common to multiple tools (e.g., Docker utilities, configuration models, logging) will be placed in a `pkg/` directory to be imported by each application as needed.

This structure allows us to build and distribute each tool as an independent binary while efficiently reusing code.

## 6. Next Steps

The following design documents in this directory provide a more detailed breakdown of each proposed tool.

-   `01_REACTOR.md`
-   `02_REACTOR_FABRIC.md`
-   `03_REACTOR_SCAFFOLD.md`
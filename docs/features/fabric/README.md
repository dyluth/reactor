
WARNING: this needs more work to consider as a final document - it needs to follow the structure of the `000_Project Charter template.md`

# Design: `Reactor-Fabric` - The AI Orchestration System

## 1. Goal

`Reactor-Fabric` is a standalone orchestration engine that manages a suite of specialized, containerized AI agents (MCP services). It is designed to enable complex, collaborative workflows where multiple AI specialists work in concert to achieve a goal. This design is based entirely on the detailed specification in `ai-prompts/6-distributed-mcp-orchestration-system.md`.

## 2. Core Functionality

-   **Configuration-Driven**: It is driven by a `suite.yaml` file which declaratively defines the suite of available AI services.
-   **On-Demand Spawning**: It dynamically spawns and tears down agent containers based on client requests.
-   **Client Agnostic**: It can serve any MCP-compliant client. A `Reactor` instance can act as a client to the fabric.
-   **Contextual Awareness**: It passes the client's file system context to the agents it spawns, ensuring they operate on the correct files.
-   **Concurrency**: It is designed to handle multiple, concurrent client connections, each with its own isolated session and context.
-   **Advanced Lifecycle Management**: It implements the sophisticated container lifecycle strategies (`fresh_per_call`, `reuse_per_session`, `smart_refresh`) outlined in `docs/CONTAINER_STRATEGIES.md`.

## 3. What It Is NOT

-   It is **not** an interactive development tool. A developer would not typically run `reactor-fabric` to work on their code directly.
-   It does **not** manage a single development environment. Its purpose is to manage a *fleet* of them.

## 4. Command-Line Interface (CLI)

```bash
# Start the orchestrator with a specific suite configuration
reactor-fabric start --config /path/to/my-suite.yaml

# Validate a suite configuration file for correctness
reactor-fabric validate --config /path/to/my-suite.yaml

# Run in the foreground for debugging
reactor-fabric start --foreground
```

## 5. Technical Implementation

The implementation will follow the detailed plan laid out in `ai-prompts/6-distributed-mcp-orchestration-system.md`.

-   **Language**: Go
-   **Standalone Binary**: `reactor-fabric` will be a separate binary from `reactor`.
-   **Shared Code**: It will share common code (e.g., Docker interaction, configuration parsing) with `Reactor` via a shared `pkg/` directory in the monorepo.
-   **Networking**: It will manage its own Docker network to facilitate communication between agent containers if necessary.

This tool represents the "multi-agent" personality of the original `claude-reactor`, now free to evolve independently as a powerful orchestration platform without being burdened by the concerns of a single developer's interactive environment.

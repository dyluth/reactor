
# **Feature Design: `reactor-fabric` - A Container-Native AI Agent Orchestrator**

Version: 2.0
Status: Proposed
Date: 2025-08-28

## **1. Vision & Strategy**

### **1.1. High-Level Summary**

`reactor-fabric` is a standalone, **container-native orchestration engine** designed to manage a suite of specialized, tool-equipped AI agents. It is positioned to fill a major gap in the current market, which is dominated by Python-centric, LLM-chaining frameworks.

The core vision is to provide a robust, scalable, and developer-friendly platform for automating complex software engineering tasks by leveraging the power of containerization and the familiar paradigms of DevOps and Platform Engineering.

### **1.2. Core Problem Solved**

Existing multi-agent frameworks are powerful but share a fundamental limitation: they are **LLM-centric**, designed primarily to orchestrate Python functions. This makes them a poor fit for automating the full spectrum of the software development lifecycle, which relies on a diverse ecosystem of compilers, CLIs, and infrastructure tools (`git`, `docker`, `kubectl`, `terraform`).

`reactor-fabric` solves this by shifting the paradigm. It is not an LLM-chaining library; it is a **container-native orchestration engine**. It enables the automation of real-world DevOps and software engineering tasks by orchestrating agents whose tools are not just Python functions, but any command-line tool that can be packaged into a container.

## **2. Core Functionality & Architecture**

### **2.1. Key Concepts**

*   **The Agent as a Container:** In `reactor-fabric`, an "agent" is a `reactor` instance. Its environment, capabilities, and tools are explicitly defined by a version-controlled `devcontainer.json` and `Dockerfile`. This provides perfect reproducibility, isolation, and true tooling agnosticism.
*   **Declarative Orchestration:** A `fabric.yml` file defines the composition and workflow of your agent team. You declare which agents are available, what their roles are, and how they are triggered and communicate.
*   **On-Demand, Stateful Spawning:** The fabric dynamically spawns agent containers as needed, managing their lifecycle with strategies like `fresh_per_call` or `reuse_per_session` to balance performance and state persistence.
*   **Client/Server Architecture:** `reactor-fabric` runs as a persistent engine. The `reactor` CLI is the primary client for human interaction, but any application (e.g., a GitHub Action, an IDE extension) can interact with the fabric via its API to trigger complex automated workflows.

### **2.2. Declarative Workflows**

Orchestration will be defined in a declarative YAML file, `fabric.yml`. This file will define:
*   **Agents:** A list of named agents, each pointing to a `devcontainer.json` source.
*   **Workflow Graph:** The relationships and data flow between agents.
*   **State Management:** How state is passed between containerized agents (e.g., via a shared volume, a message queue, or a Redis-based blackboard).

### **2.3. Foundational Patterns**

The architecture will draw from established patterns in distributed systems and multi-agent systems:
*   **Hierarchical Task Planning:** A top-level "planner" agent can decompose a high-level goal (e.g., "deploy the web service") into a graph of tasks executed by specialized worker agents (a "build" agent, a "test" agent, a "deploy" agent).
*   **Blackboard Architecture:** Agents will communicate and coordinate by reading from and writing to a shared, persistent state store (the "blackboard"), rather than through complex point-to-point integrations.

### **2.4. Command-Line Interface**

The `reactor-fabric` binary will be a standalone server process.
```bash
# Start the orchestrator with a specific fabric configuration
reactor-fabric start --config /path/to/my-fabric.yml

# Validate a fabric configuration file for correctness
reactor-fabric validate --config /path/to/my-fabric.yml
```

### **2.5. Integration with `reactor`**

The `reactor` CLI can act as a client to the fabric, but `reactor-fabric` is a completely separate system designed for multi-agent, automated workflows, not interactive development.


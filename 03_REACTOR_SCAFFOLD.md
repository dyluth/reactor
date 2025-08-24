WARNING: this needs more work to consider as a final document - it needs to follow the structure of the `000_Project Charter template.md`

# Design: `Reactor-Scaffold` - The Project Scaffolding Tool

## 1. Goal

`Reactor-Scaffold` is a simple, focused command-line tool for generating new projects from predefined templates. It extracts the scaffolding functionality from the original `claude-reactor` project into a separate, optional tool.

## 2. Core Functionality

-   **Template-Based**: It uses a library of built-in templates for common project types (e.g., Go API, Rust CLI, Node.js web app).
-   **Interactive Mode**: An interactive `reactor-scaffold init` command guides the user through the process of creating a new project.
-   **Direct Mode**: A non-interactive `reactor-scaffold new <template> <project-name>` command allows for quick, direct project creation, suitable for scripting.
-   **Extensible**: Users can point it to their own directories of custom templates for company-specific or personal project structures.

## 3. What It Is NOT

-   It does **not** run or manage containers. Its only job is to create files on the local filesystem.
-   It is **not** required to use `Reactor` or `Reactor-Fabric`. It is a completely independent utility.

## 4. Command-Line Interface (CLI)

```bash
# Interactively create a new project
reactor-scaffold init

# List available built-in templates
reactor-scaffold templates list

# Directly create a new project from a template
reactor-scaffold new go-api my-new-api

# Use a custom template directory
reactor-scaffold new --template-dir ~/my-templates/ my-custom-proj
```

## 5. Benefits of Separation

-   **Keeps `Reactor` Lean**: The primary development tool is not burdened with the logic for project generation.
-   **Independent Evolution**: The templating engine can evolve with new features (e.g., more complex logic, integration with version control) without impacting the container runtime tool.
-   **Optionality**: Users who do not need project scaffolding do not need to install or learn this tool.

By separating this functionality, we adhere to the Unix philosophy of "doing one thing and doing it well." `Reactor-Scaffold` creates projects, and `Reactor` runs them.

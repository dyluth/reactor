# **Feature Design Document: M5 - `init` Command Templates**

Version: 1.1
Status: Implementation Ready
Author(s): Gemini, cam
Date: 2025-09-02
Updated: 2025-01-17

## **1. The 'Why': Rationale & User Focus**

### **1.1. High-level summary**

This feature enhances the `reactor init` command to support templates, providing new users with a zero-configuration, "hello world" experience. By running a single command (`reactor init --template go`), a user can generate a complete, runnable Dev Container environment, including a sample application, a `Dockerfile`, and a pre-configured `devcontainer.json`. This dramatically lowers the barrier to entry and makes the power of `reactor` immediately apparent.

### **1.2. User personas**

*   **Primary Persona: The New User**: Any developer who is trying `reactor` for the first time. They may not be an expert in Dev Containers and want to see the value of the tool as quickly as possible.

### **1.3. Problem statement & user stories**

**Problem Statement:**
The current `reactor init` command only creates a minimal `devcontainer.json` file. This is not enough for a new user to get started, as it requires them to also create a `Dockerfile` and application code manually. This creates friction and prevents the user from having a successful first run.

**User Stories:**

*   As a **New User**, I want to run a single command to generate a complete, working sample project, so I can immediately run `reactor up` and see a successful result.
*   As a **New User**, I want to choose a template for my preferred language (`go`, `python`, `node`), so the generated project is relevant to my needs.

### **1.4. Success metrics**

*   The `reactor init --template <name>` command is implemented and functional.
*   Running `reactor up` in a directory created by `init --template` results in a successful build and a running application.
*   The implementation is covered by integration tests.

## **2. The 'How': Technical Design & Architecture**

### **2.1. System context & constraints**

*   **Technology Stack:** Go, Cobra.
*   **Current State:** The `init` command exists but is minimal. This feature will expand its functionality significantly.
*   **Technical Constraints:** The generated files must be simple, idiomatic, and easy for a new user to understand. We will use the official, curated images we created in a previous milestone as the base for the generated Dockerfiles.

### **2.2. Detailed design**

#### **2.2.1. CLI Updates**

The `init` command in `cmd/reactor/main.go` (configInitHandler) will be updated:
*   It will accept a new optional flag: `--template`. Valid values will be `go`, `python`, `node`.
*   The flag will include `ValidArgs` for shell auto-completion and enhanced help text listing available templates.
*   If no template is provided, it will perform its original behavior by calling `configService.InitializeProject()`.
*   If a template is provided, it will call the new `templates.GenerateFromTemplate()` function.

#### **2.2.2. New Package: `pkg/templates`**

A new package will be created to hold the template file content.

*   **`pkg/templates/templates.go`**: This file will contain the static string content for each file to be generated (the `devcontainer.json`, `Dockerfile`, and sample application code for each language).
*   **`pkg/templates/generator.go`**: This file will contain the logic for creating the files and directories based on the chosen template.

#### **2.2.3. Template Specifications**

This is the explicit specification for the files to be created for each template.

**Template: `go`**

*   **`.devcontainer/devcontainer.json`**
    ```json
    {
      "name": "Reactor Go Project",
      "build": {
        "dockerfile": "Dockerfile",
        "context": "."
      },
      "forwardPorts": [8080]
    }
    ```
*   **`.devcontainer/Dockerfile`**
    ```dockerfile
    FROM ghcr.io/dyluth/reactor/go:latest
    WORKDIR /workspace
    COPY . .
    RUN go mod tidy
    CMD ["go", "run", "main.go"]
    ```
*   **`main.go`**
    ```go
    package main

    import (
        "fmt"
        "log"
        "net/http"
    )

    func main() {
        http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
            fmt.Fprintf(w, "Hello, World from your Reactor Go environment!")
        })
        log.Println("Server starting on port 8080...")
        log.Fatal(http.ListenAndServe(":8080", nil))
    }
    ```
*   **`go.mod`**
    ```
    module my-go-app

    go 1.22
    ```

**Template: `python`**

*   **`.devcontainer/devcontainer.json`**
    ```json
    {
      "name": "Reactor Python Project",
      "build": {
        "dockerfile": "Dockerfile",
        "context": "."
      },
      "forwardPorts": [8000]
    }
    ```
*   **`.devcontainer/Dockerfile`**
    ```dockerfile
    FROM ghcr.io/dyluth/reactor/python:latest
    WORKDIR /workspace
    COPY . .
    RUN pip install -r requirements.txt
    CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
    ```
*   **`main.py`**
    ```python
    from fastapi import FastAPI

    app = FastAPI()

    @app.get("/")
    def read_root():
        return {"message": "Hello, World from your Reactor Python environment!"}
    ```
*   **`requirements.txt`**
    ```
    fastapi
    uvicorn[standard]
    ```

**Template: `node`**

*   **`.devcontainer/devcontainer.json`**
    ```json
    {
      "name": "Reactor Node.js Project", 
      "build": {
        "dockerfile": "Dockerfile",
        "context": "."
      },
      "forwardPorts": [3000]
    }
    ```
*   **`.devcontainer/Dockerfile`**
    ```dockerfile
    FROM ghcr.io/dyluth/reactor/node:latest
    WORKDIR /workspace
    COPY . .
    RUN npm install
    CMD [ "node", "index.js" ]
    ```
*   **`index.js`**
    ```javascript
    const express = require('express');
    const app = express();
    const port = 3000;

    app.get('/', (req, res) => {
      res.send('Hello, World from your Reactor Node.js environment!');
    });

    app.listen(port, () => {
      console.log(`Server listening on port ${port}`);
    });
    ```
*   **`package.json`**
    ```json
    {
      "name": "my-node-app",
      "version": "1.0.0",
      "description": "A Reactor Node.js project",
      "main": "index.js",
      "scripts": {
        "start": "node index.js"
      },
      "dependencies": {
        "express": "^4.17.1"
      }
    }
    ```

#### **2.2.4. Dynamic Project Naming**

All templates use dynamic project naming based on the current working directory:
*   Project names are derived from `filepath.Base(os.Getwd())` and sanitized for each language's requirements
*   **Sanitization Rules** (applied in order):
    1. Convert to lowercase
    2. Replace spaces and special characters (non-alphanumeric, non-hyphen) with hyphens
    3. Collapse multiple consecutive hyphens into single hyphens  
    4. Prefix with `app-` if name starts with number or hyphen
    5. Remove leading/trailing hyphens
*   **Examples:**
    - `"My Cool App"` → `"my-cool-app"`
    - `"123-project"` → `"app-123-project"`
    - `"react@app!"` → `"react-app"`
    - `"__my-go-app__"` → `"my-go-app"`

#### **2.2.5. File Conflict Detection**

The template generator performs intelligent file conflict detection:
*   **Conflict Check**: Before writing files, check if any template files already exist
*   **Acceptable Files**: Hidden files (`.git`, `.DS_Store`) and non-conflicting files are acceptable
*   **Error Handling**: Clear error messages listing specific conflicting files
*   **No Force Flag**: No override capability in this milestone for simplicity

### **2.3. Implementation Architecture**

#### **2.3.1. Package Structure**

*   **`pkg/templates/templates.go`**: Template content constants for all languages
*   **`pkg/templates/generator.go`**: Core generation logic with sanitization and conflict detection
*   **`cmd/reactor/main.go`**: Updated configInitHandler dispatcher logic

#### **2.3.2. Core Functions**

*   **`GenerateFromTemplate(templateName, targetDir string) error`**: Main entry point
*   **`sanitizeProjectName(name string) string`**: Project name sanitization
*   **`checkFileConflicts(files []string, targetDir string) error`**: Conflict detection
*   **Template content functions**: `getGoTemplate()`, `getPythonTemplate()`, `getNodeTemplate()`

### **2.4. Phased implementation plan**

*   **PR 1: Go Template + Complete Infrastructure**
    *   [ ] Create complete `pkg/templates` package with sanitization and conflict detection
    *   [ ] Add `--template` flag to init command with ValidArgs and enhanced help text  
    *   [ ] Implement dispatcher logic in configInitHandler
    *   [ ] Add Go template content and generation logic
    *   [ ] Create comprehensive integration test for Go template with HTTP validation
*   **PR 2: Python and Node Templates**
    *   [ ] Add Python and Node template content constants
    *   [ ] Add integration tests for Python and Node templates

### **2.5. Testing strategy**

*   **Integration Tests:** Comprehensive end-to-end validation.
    
    **PR 1: Go Template Test**
    1.  Create isolated temporary directory using testutil helpers
    2.  Run `reactor init --template go` inside it
    3.  Verify all files created with correct content and dynamic project name
    4.  Run `reactor up` and validate successful container build
    5.  Make HTTP request to `localhost:8080` and verify "Hello, World" response
    6.  Run `reactor down` and verify container cleanup
    
    **PR 2: Python and Node Tests**
    - Similar tests for Python (port 8000, FastAPI JSON response)
    - Similar tests for Node.js (port 3000, Express text response)

*   **Unit Tests:**
    - Project name sanitization with comprehensive edge cases
    - File conflict detection logic
    - Template content validation

## **3. The 'What Ifs': Risks & Mitigation**

*   **Risk:** File conflicts when `reactor init --template` is run.
*   **Mitigation:** Intelligent conflict detection that allows hidden files but fails on template file conflicts with clear error messages listing specific conflicting files.

*   **Risk:** Invalid directory names for package managers.
*   **Mitigation:** Comprehensive project name sanitization following consistent rules that work across all target languages and package managers.

*   **Risk:** The base container images don't exist.
*   **Mitigation:** Implementation assumes images exist as specified (ghcr.io/dyluth/reactor/{go,python,node}:latest). Image availability is outside scope of this milestone.

*   **Risk:** Template files become outdated.
*   **Mitigation:** Template content stored as Go string constants, making them easy to update through standard development workflows.

*   **Risk:** Integration tests fail due to port conflicts.
*   **Mitigation:** Use testutil isolation helpers and unique ports. Tests can skip HTTP validation if ports are unavailable, focusing on container build success.
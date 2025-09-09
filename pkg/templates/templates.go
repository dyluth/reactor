package templates

// TemplateFile represents a single file in a template
type TemplateFile struct {
	Path    string // Relative path from project root
	Content string // File content with {{PROJECT_NAME}} placeholder
}

// Template represents a complete project template
type Template struct {
	Name  string
	Files []TemplateFile
}

// getTemplateByName returns the template for the given name
func getTemplateByName(name string) (Template, bool) {
	switch name {
	case "go":
		return getGoTemplate(), true
	case "python":
		return getPythonTemplate(), true
	case "node":
		return getNodeTemplate(), true
	default:
		return Template{}, false
	}
}

// getGoTemplate returns the Go project template
func getGoTemplate() Template {
	return Template{
		Name: "go",
		Files: []TemplateFile{
			{
				Path: ".devcontainer/devcontainer.json",
				Content: `{
  "name": "Reactor Go Project",
  "build": {
    "dockerfile": "Dockerfile",
    "context": "."
  },
  "forwardPorts": [8080]
}`,
			},
			{
				Path: ".devcontainer/Dockerfile",
				Content: `FROM ghcr.io/dyluth/reactor/go:latest
WORKDIR /workspace
COPY . .
RUN go mod tidy
CMD ["go", "run", "main.go"]`,
			},
			{
				Path: "main.go",
				Content: `package main

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
}`,
			},
			{
				Path: "go.mod",
				Content: `module {{PROJECT_NAME}}

go 1.22`,
			},
		},
	}
}

// getPythonTemplate returns the Python project template (placeholder for PR 2)
func getPythonTemplate() Template {
	return Template{
		Name: "python",
		Files: []TemplateFile{
			{
				Path: ".devcontainer/devcontainer.json",
				Content: `{
  "name": "Reactor Python Project",
  "build": {
    "dockerfile": "Dockerfile",
    "context": "."
  },
  "forwardPorts": [8000]
}`,
			},
			{
				Path: ".devcontainer/Dockerfile",
				Content: `FROM ghcr.io/dyluth/reactor/python:latest
WORKDIR /workspace
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY main.py .
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]`,
			},
			{
				Path: ".devcontainer/requirements.txt",
				Content: `fastapi
uvicorn[standard]`,
			},
			{
				Path: ".devcontainer/main.py",
				Content: `from fastapi import FastAPI

app = FastAPI()

@app.get("/")
def read_root():
    return {"message": "Hello, World from your Reactor Python environment!"}`,
			},
		},
	}
}

// getNodeTemplate returns the Node.js project template (placeholder for PR 2)
func getNodeTemplate() Template {
	return Template{
		Name: "node",
		Files: []TemplateFile{
			{
				Path: ".devcontainer/devcontainer.json",
				Content: `{
  "name": "Reactor Node.js Project",
  "build": {
    "dockerfile": "Dockerfile",
    "context": "."
  },
  "forwardPorts": [3000]
}`,
			},
			{
				Path: ".devcontainer/Dockerfile",
				Content: `FROM ghcr.io/dyluth/reactor/node:latest
WORKDIR /workspace
COPY package.json .
RUN npm install
COPY index.js .
CMD [ "node", "index.js" ]`,
			},
			{
				Path: ".devcontainer/package.json",
				Content: `{
  "name": "{{PROJECT_NAME}}",
  "version": "1.0.0",
  "description": "A Reactor Node.js project",
  "main": "index.js",
  "scripts": {
    "start": "node index.js"
  },
  "dependencies": {
    "express": "^4.17.1"
  }
}`,
			},
			{
				Path: ".devcontainer/index.js",
				Content: `const express = require('express');
const app = express();
const port = 3000;

app.get('/', (req, res) => {
  res.send('Hello, World from your Reactor Node.js environment!');
});

app.listen(port, () => {
  console.log(` + "`" + `Server listening on port ${port}` + "`" + `);
});`,
			},
		},
	}
}

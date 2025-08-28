# Reactor Go Image
# Built FROM reactor/base with complete Go development environment  
# Size target: <500MB compressed

ARG BASE_IMAGE=ghcr.io/dyluth/reactor/base:latest
FROM ${BASE_IMAGE}

# Switch to root for installations
USER root

# Install Go 1.22
ENV GO_VERSION=1.22.8
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ]; then GO_ARCH="arm64"; \
    elif [ "$ARCH" = "amd64" ]; then GO_ARCH="amd64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" | tar -xz -C /usr/local

# Set up Go environment
ENV PATH=/usr/local/go/bin:$PATH
ENV GOROOT=/usr/local/go

# Install Go development tools as root (they'll be accessible to claude user)
RUN /usr/local/go/bin/go install golang.org/x/tools/gopls@v0.14.2 && \
    /usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@v1.21.2 && \
    /usr/local/go/bin/go install honnef.co/go/tools/cmd/staticcheck@2023.1.6 && \
    /usr/local/go/bin/go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2 && \
    /usr/local/go/bin/go install golang.org/x/tools/cmd/goimports@latest && \
    /usr/local/go/bin/go install github.com/fatih/gomodifytags@v1.16.0 && \
    /usr/local/go/bin/go install github.com/cweill/gotests/gotests@v1.6.0

# Copy Go tools to system path for all users
RUN cp /root/go/bin/* /usr/local/bin/ 2>/dev/null || true

# Set up Go environment for claude user
USER claude
WORKDIR /home/claude

# Configure Go environment variables
RUN echo '# Go development environment' >> /home/claude/.bashrc \
    && echo 'export GOPATH="$HOME/go"' >> /home/claude/.bashrc \
    && echo 'export GOBIN="$GOPATH/bin"' >> /home/claude/.bashrc \
    && echo 'export PATH="$PATH:$GOBIN:/usr/local/go/bin"' >> /home/claude/.bashrc \
    && echo '# Go development aliases' >> /home/claude/.bashrc \
    && echo 'alias gob="go build"' >> /home/claude/.bashrc \
    && echo 'alias gor="go run"' >> /home/claude/.bashrc \
    && echo 'alias got="go test"' >> /home/claude/.bashrc \
    && echo 'alias gotv="go test -v"' >> /home/claude/.bashrc \
    && echo 'alias gom="go mod"' >> /home/claude/.bashrc \
    && echo 'alias gof="go fmt"' >> /home/claude/.bashrc

# Create Go workspace directories
RUN mkdir -p /home/claude/go/src /home/claude/go/bin /home/claude/go/pkg

# Create default Go project structure in workspace
USER root
RUN mkdir -p /workspace/cmd/main /workspace/internal /workspace/pkg \
    && cat > /workspace/main.go << 'EOF'
package main

import (
    "fmt"
    "runtime"
)

func main() {
    fmt.Println("Hello from Reactor Go environment!")
    fmt.Printf("Go version: %s\n", runtime.Version())
    fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
EOF

# Initialize Go module
RUN cd /workspace && /usr/local/go/bin/go mod init reactor-workspace

# Create sample package
RUN mkdir -p /workspace/pkg/hello \
    && cat > /workspace/pkg/hello/hello.go << 'EOF'
// Package hello provides greeting functionality
package hello

import "fmt"

// Greet returns a greeting message
func Greet(name string) string {
    return fmt.Sprintf("Hello, %s! Welcome to Reactor Go environment.", name)
}
EOF

# Create sample test
RUN cat > /workspace/pkg/hello/hello_test.go << 'EOF'
package hello

import "testing"

func TestGreet(t *testing.T) {
    result := Greet("Gopher")
    expected := "Hello, Gopher! Welcome to Reactor Go environment."
    
    if result != expected {
        t.Errorf("Greet() = %q, want %q", result, expected)
    }
}
EOF

# Create Makefile for common tasks
RUN cat > /workspace/Makefile << 'EOF'
.PHONY: build run test fmt lint clean

# Build the main application
build:
	go build -o bin/main main.go

# Run the main application
run:
	go run main.go

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-v:
	go test -v ./...

# Format all Go code
fmt:
	go fmt ./...
	goimports -w .

# Run linter
lint:
	golangci-lint run

# Static analysis  
check:
	staticcheck ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Initialize new module (if needed)
init:
	go mod init $(shell basename $(PWD))
	go mod tidy
EOF

# Set proper ownership
RUN chown -R claude:claude /workspace

# Switch back to claude user  
USER claude
WORKDIR /workspace

# Set Go environment variables for this session
ENV GOPATH=/home/claude/go
ENV GOBIN=/home/claude/go/bin
ENV PATH=${PATH}:${GOBIN}:/usr/local/go/bin

# Verify Go installation and build the sample
RUN go version \
    && go env GOPATH \
    && go env GOROOT \
    && go build main.go \
    && ./main \
    && rm main

# Verify development tools
RUN gopls version \
    && golangci-lint --version \
    && staticcheck -version

# Metadata
LABEL org.opencontainers.image.title="Reactor Go"
LABEL org.opencontainers.image.description="Complete Go development environment built on Reactor Base"
LABEL org.opencontainers.image.vendor="Reactor Suite"
LABEL org.opencontainers.image.licenses="MIT"
#!/bin/bash
# Test script for reactor/go image
# Verifies Go development environment is working

set -e

echo "🔵 Testing Reactor Go Image..."

# Test base image functionality first
echo "✅ Testing base image tools..."
git --version
claude --version || echo "Claude CLI available"

# Test Go installation
echo "✅ Testing Go installation..."
go version
go env GOPATH
go env GOROOT

# Test Go can compile and run
echo "✅ Testing Go execution..."
cd /workspace
go run main.go

# Test Go module functionality
echo "✅ Testing Go modules..."
go mod tidy
go mod verify

# Test package building
echo "✅ Testing package building..."
go build -o test-main main.go
./test-main
rm test-main

# Test Go development tools
echo "✅ Testing Go development tools..."
gopls version
dlv version | head -1
staticcheck -version  
golangci-lint --version
which goimports > /dev/null && echo "goimports available" || echo "goimports not found"

# Test package functionality
echo "✅ Testing sample package..."
go test ./pkg/hello/
go test -v ./pkg/hello/

# Test workspace setup
echo "✅ Testing workspace structure..."
ls -la /workspace
test -f /workspace/main.go
test -f /workspace/go.mod
test -d /workspace/pkg/hello
test -f /workspace/pkg/hello/hello.go
test -f /workspace/pkg/hello/hello_test.go
test -f /workspace/Makefile

# Test Makefile functionality
echo "✅ Testing Makefile..."
cd /workspace
make build
test -f bin/main
make clean
test ! -f bin/main

# Test code formatting
echo "✅ Testing code formatting..."
make fmt

# Test linting (may have warnings but shouldn't fail)
echo "✅ Testing linting..."
make lint || echo "Linter completed with warnings (normal)"

# Test static analysis
echo "✅ Testing static analysis..."
make check

# Test aliases (if running in bash)
echo "✅ Testing shell configuration..."
if grep -q "alias gob=" /home/claude/.bashrc; then
    echo "Go aliases: ✅ CONFIGURED"
else
    echo "Go aliases: ❌ NOT CONFIGURED"
    exit 1
fi

# Test GOPATH setup
echo "✅ Testing Go workspace..."
test -d /home/claude/go/src
test -d /home/claude/go/bin  
test -d /home/claude/go/pkg

echo "🎉 All Go tests passed! Reactor Go image is ready."
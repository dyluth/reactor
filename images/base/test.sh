#!/bin/bash
# Test script for reactor/base image
# Verifies all essential tools and AI CLIs are working

set -e

echo "🧪 Testing Reactor Base Image..."

# Test basic system tools
echo "✅ Testing basic system tools..."
git --version
curl --version
wget --version
jq --version
rg --version
fzf --version

# Test development tools  
echo "✅ Testing development tools..."
node --version
npm --version
docker --version

# Test Claude CLI
echo "✅ Testing Claude CLI..."
if command -v claude &> /dev/null; then
    claude --version || echo "Claude CLI available but version check failed (expected for unauthenticated)"
    echo "Claude CLI: ✅ INSTALLED"
else
    echo "Claude CLI: ❌ NOT FOUND"
    exit 1
fi

# Test user and permissions
echo "✅ Testing user configuration..."
whoami
id
pwd

# Test sudo access
echo "✅ Testing sudo access..."
sudo echo "Sudo access confirmed"

# Test git-aware prompt setup
echo "✅ Testing shell configuration..."
if grep -q "GITAWAREPROMPT" /home/claude/.bashrc; then
    echo "Git-aware prompt: ✅ CONFIGURED"
else
    echo "Git-aware prompt: ❌ NOT CONFIGURED"
    exit 1
fi

# Test workspace directory
echo "✅ Testing workspace..."
ls -la /workspace
touch /workspace/test-file
rm /workspace/test-file

echo "🎉 All tests passed! Reactor base image is ready."
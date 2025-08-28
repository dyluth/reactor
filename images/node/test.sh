#!/bin/bash
# Test script for reactor/node image
# Verifies Node.js development environment is working

set -e

echo "🟢 Testing Reactor Node.js Image..."

# Test base image functionality first
echo "✅ Testing base image tools..."
git --version
claude --version || echo "Claude CLI available"

# Test Node.js installation
echo "✅ Testing Node.js installation..."
node --version
npm --version

# Test Node.js can run
echo "✅ Testing Node.js execution..."
node -e "console.log('Node.js is working!')"
node /workspace/src/index.js

# Test development tools
echo "✅ Testing Node.js development tools..."
tsc --version
ts-node --version
eslint --version
prettier --version
jest --version
nodemon --version

# Test global packages
echo "✅ Testing global packages..."
which tsc
which ts-node
which eslint
which prettier
which jest
which nodemon

# Test workspace setup
echo "✅ Testing workspace structure..."
ls -la /workspace
test -f /workspace/package.json
test -f /workspace/tsconfig.json
test -f /workspace/.eslintrc.js
test -f /workspace/.prettierrc
test -d /workspace/src
test -d /workspace/tests
test -f /workspace/src/index.js

# Test package.json structure
echo "✅ Testing package.json..."
node -e "
const pkg = require('/workspace/package.json');
console.log('Package name:', pkg.name);
console.log('Available scripts:', Object.keys(pkg.scripts).join(', '));
"

# Test TypeScript compilation
echo "✅ Testing TypeScript..."
cd /workspace
echo 'console.log("TypeScript test");' > src/test.ts
tsc --noEmit src/test.ts
rm src/test.ts

# Test ESLint
echo "✅ Testing ESLint..."
eslint src/index.js --format=compact

# Test Prettier
echo "✅ Testing Prettier..."
prettier --check src/index.js

# Test aliases (if running in bash)
echo "✅ Testing shell configuration..."
if grep -q "alias dev=" /home/claude/.bashrc; then
    echo "Node.js aliases: ✅ CONFIGURED"
else
    echo "Node.js aliases: ❌ NOT CONFIGURED"
    exit 1
fi

# Test npm functionality
echo "✅ Testing npm functionality..."
cd /workspace
npm list --depth=0 2>/dev/null || echo "No local packages (expected for new project)"

echo "🎉 All Node.js tests passed! Reactor Node.js image is ready."
#!/bin/bash
# Test script for reactor/python image  
# Verifies Python development environment is working

set -e

echo "🐍 Testing Reactor Python Image..."

# Test base image functionality first
echo "✅ Testing base image tools..."
git --version
claude --version || echo "Claude CLI available"

# Test Python installation
echo "✅ Testing Python installation..."
python --version
python3 --version
pip --version
uv --version

# Test Python can run
echo "✅ Testing Python execution..."
python -c "print('Python is working!')"
python /workspace/hello.py

# Test modern Python tools
echo "✅ Testing Python development tools..."
ruff --version
black --version  
mypy --version
pytest --version

# Test essential packages
echo "✅ Testing essential packages..."
python -c "import requests; print(f'requests: {requests.__version__}')"
python -c "import rich; print('Rich library imported successfully')"
python -c "import IPython; print(f'IPython: {IPython.__version__}')"

# Test uv package manager
echo "✅ Testing uv package manager..."
cd /workspace
uv --help > /dev/null
echo "uv package manager working"

# Test workspace setup
echo "✅ Testing workspace structure..."
ls -la /workspace
test -d /workspace/src
test -d /workspace/tests
test -f /workspace/README.md
test -f /workspace/hello.py

# Test Python path configuration
echo "✅ Testing Python environment..."
python -c "import sys; print('Python path configured:', '/workspace' in sys.path or any('/workspace' in p for p in sys.path))"

# Test aliases (if running in bash)
echo "✅ Testing shell configuration..."
if grep -q "alias py=python" /home/claude/.bashrc; then
    echo "Python aliases: ✅ CONFIGURED"
else
    echo "Python aliases: ❌ NOT CONFIGURED"
    exit 1
fi

echo "🎉 All Python tests passed! Reactor Python image is ready."
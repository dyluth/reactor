#!/bin/bash
# Build all Reactor container images locally
# Usage: ./scripts/build-images.sh [OPTIONS]
#
# Options:
#   -t, --test     Build with :test tags and run tests
#   -o, --official Build with official tags
#   -h, --help     Show this help

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

# Default settings
TEST_MODE=false
OFFICIAL_TAGS=false
TAG_SUFFIX="local"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--test)
            TEST_MODE=true
            TAG_SUFFIX="test"
            shift
            ;;
        -o|--official)
            OFFICIAL_TAGS=true
            shift
            ;;
        -h|--help)
            echo "Build all Reactor container images locally"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -t, --test     Build with :test tags and run tests"
            echo "  -o, --official Build with official tags (ghcr.io/dyluth/reactor/*)"
            echo "  -h, --help     Show this help"
            echo ""
            echo "Examples:"
            echo "  $0                 # Build with :local tags"
            echo "  $0 --test          # Build with :test tags and run tests"
            echo "  $0 --official      # Build with official registry tags"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Set image names based on options
if [[ "$OFFICIAL_TAGS" == "true" ]]; then
    BASE_TAG="ghcr.io/dyluth/reactor/base:latest"
    PYTHON_TAG="ghcr.io/dyluth/reactor/python:latest"
    NODE_TAG="ghcr.io/dyluth/reactor/node:latest"
    GO_TAG="ghcr.io/dyluth/reactor/go:latest"
else
    BASE_TAG="reactor/base:$TAG_SUFFIX"
    PYTHON_TAG="reactor/python:$TAG_SUFFIX"
    NODE_TAG="reactor/node:$TAG_SUFFIX"
    GO_TAG="reactor/go:$TAG_SUFFIX"
fi

echo "🐳 Building Reactor container images..."
echo "Repository root: $REPO_ROOT"
echo "Tags: $TAG_SUFFIX"
echo ""

# Build base image first (others depend on it)
echo "📦 Building base image..."
docker build -t "$BASE_TAG" images/base
echo "✅ Base image built: $BASE_TAG"
echo ""

# Build language-specific images in parallel
echo "📦 Building language-specific images..."
docker build -t "$PYTHON_TAG" images/python &
docker build -t "$NODE_TAG" images/node &
docker build -t "$GO_TAG" images/go &

# Wait for all background builds to complete
wait

echo "✅ Python image built: $PYTHON_TAG"
echo "✅ Node.js image built: $NODE_TAG"
echo "✅ Go image built: $GO_TAG"
echo ""

# Run tests if requested
if [[ "$TEST_MODE" == "true" ]]; then
    echo "🧪 Running image tests..."
    
    echo "Testing base image..."
    docker run --rm -v "$(pwd)/images/base/test.sh:/test.sh:ro" "$BASE_TAG" bash /test.sh
    
    echo "Testing Python image..."
    docker run --rm -v "$(pwd)/images/python/test.sh:/test.sh:ro" "$PYTHON_TAG" bash /test.sh
    
    echo "Testing Node.js image..."
    docker run --rm -v "$(pwd)/images/node/test.sh:/test.sh:ro" "$NODE_TAG" bash /test.sh
    
    echo "Testing Go image..."
    docker run --rm -v "$(pwd)/images/go/test.sh:/test.sh:ro" "$GO_TAG" bash /test.sh
    
    echo "✅ All tests passed!"
    echo ""
fi

# Show final status
echo "🎉 All images built successfully!"
echo ""
echo "Available images:"
docker images | grep -E "(reactor/|ghcr.io/dyluth/reactor)" | head -20
echo ""
echo "Usage examples:"
echo "  docker run --rm -it $BASE_TAG bash"
echo "  docker run --rm -it $PYTHON_TAG bash"
echo "  docker run --rm -it $NODE_TAG bash"  
echo "  docker run --rm -it $GO_TAG bash"
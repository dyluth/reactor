#!/bin/bash
#
# Reactor Container Entrypoint
# 
# This script acts as the main entrypoint for Reactor containers, handling
# Docker host integration through socat proxy when enabled.
#
# Environment Variables:
#   REACTOR_DOCKER_HOST_INTEGRATION - Set to 'true' to enable Docker socket forwarding
#   DOCKER_HOST - Docker daemon endpoint (when host integration is enabled)
#

set -euo pipefail

# Function to log messages with timestamps
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" >&2
}

# Function to start socat proxy for Docker socket forwarding
start_docker_proxy() {
    if [ ! -S /var/run/docker.sock ]; then
        log "WARNING: Docker socket not found at /var/run/docker.sock"
        log "Docker host integration may not work properly"
        return 1
    fi

    log "Starting Docker socket proxy..."
    
    # Create a Unix socket that forwards to the host Docker daemon
    # This allows the container to communicate with the host Docker daemon
    socat UNIX-LISTEN:/tmp/docker.sock,fork,user=claude,group=claude UNIX-CONNECT:/var/run/docker.sock &
    SOCAT_PID=$!
    
    # Set DOCKER_HOST for applications inside the container
    export DOCKER_HOST=unix:///tmp/docker.sock
    
    log "Docker socket proxy started (PID: $SOCAT_PID)"
    log "DOCKER_HOST set to: $DOCKER_HOST"
    
    # Store PID for cleanup
    echo $SOCAT_PID > /tmp/socat.pid
    
    return 0
}

# Function to stop Docker proxy
stop_docker_proxy() {
    if [ -f /tmp/socat.pid ]; then
        local pid=$(cat /tmp/socat.pid)
        if kill -0 "$pid" 2>/dev/null; then
            log "Stopping Docker socket proxy (PID: $pid)..."
            kill "$pid" 2>/dev/null || true
        fi
        rm -f /tmp/socat.pid
    fi
}

# Function to handle script termination
cleanup() {
    log "Received termination signal, cleaning up..."
    stop_docker_proxy
    exit 0
}

# Set up signal handlers for graceful shutdown
trap cleanup SIGTERM SIGINT SIGQUIT

# Main entrypoint logic
main() {
    log "Reactor container entrypoint starting..."
    
    # Check if Docker host integration is enabled
    if [ "${REACTOR_DOCKER_HOST_INTEGRATION:-false}" = "true" ]; then
        log "Docker host integration enabled"
        
        if start_docker_proxy; then
            log "Docker proxy setup completed successfully"
        else
            log "WARNING: Failed to set up Docker proxy, continuing without host integration"
        fi
    else
        log "Docker host integration disabled"
    fi
    
    # Switch to claude user for running commands
    if [ "$(whoami)" = "root" ]; then
        log "Switching to claude user..."
        # If no command specified, check if we have a TTY
        if [ $# -eq 0 ]; then
            if [ -t 0 ]; then
                # TTY available - use interactive shell
                exec su - claude -c "cd /workspace && exec bash"
            else
                # No TTY - use sleep to keep container alive for exec
                log "No TTY detected, keeping container alive for exec sessions..."
                exec su - claude -c "cd /workspace && exec sleep infinity"
            fi
        else
            exec su - claude -c "cd /workspace && exec \"$@\""
        fi
    else
        log "Running as user: $(whoami)"
        cd /workspace
        # If no command specified, check if we have a TTY
        if [ $# -eq 0 ]; then
            if [ -t 0 ]; then
                # TTY available - use interactive shell
                exec bash
            else
                # No TTY - use sleep to keep container alive for exec
                log "No TTY detected, keeping container alive for exec sessions..."
                exec sleep infinity
            fi
        else
            exec "$@"
        fi
    fi
}

# Execute main function with all arguments
main "$@"
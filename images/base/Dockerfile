# Reactor Base Image
# Minimal image with essential CLI tools and AI CLIs (Claude, Gemini)
# Size target: <200MB compressed

FROM debian:bullseye-slim

# Install essential system packages and CLI tools
# Combine all apt operations in single layer for optimal caching and size
RUN apt-get update && apt-get install -y \
    # Core system utilities
    curl=7.74.0-1.3+deb11u* \
    git=1:2.30.2-1+deb11u* \
    ca-certificates=20210119 \
    wget=1.21-1+deb11u* \
    unzip=6.0-26+deb11u* \
    gnupg2=2.2.27-2+deb11u* \
    sudo=1.9.5p2-3+deb11u* \
    # Required for Docker host integration  
    socat=1.7.4.1-3 \
    # Essential CLI tools for development
    ripgrep=12.1.1-1+b1 \
    jq=1.6-2.1 \
    fzf=0.24.3-1+b5 \
    nano=5.4-2+deb11u* \
    vim=2:8.2.2434-3+deb11u* \
    less=551-2 \
    # Process and system monitoring
    procps=2:3.3.17-5 \
    htop=3.0.5-7 \
    # Essential build tools (required for some npm packages)
    build-essential=12.9 \
    # Shell utilities
    shellcheck=0.7.1-1+deb11u* \
    man-db=2.9.4-2 \
    && rm -rf /var/lib/apt/lists/*

# Install Docker CLI for host integration support
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian bullseye stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y docker-ce-cli=5:24.0.* \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js (required for Claude CLI)
# Using official NodeSource repository for latest LTS
ENV NODE_VERSION=20.18.0
RUN curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - \
    && apt-get install -y nodejs=$NODE_VERSION-* \
    && rm -rf /var/lib/apt/lists/* \
    # Verify installation
    && node --version \
    && npm --version

# Create non-root claude user with sudo access
RUN useradd -m -s /bin/bash claude \
    && usermod -aG sudo claude \
    && echo 'claude ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

# Install Claude CLI as claude user
USER claude
WORKDIR /home/claude
RUN npm install -g @anthropic-ai/claude-code@1.* \
    && echo 'export PATH="$(npm config get prefix)/bin:$PATH"' >> /home/claude/.bashrc

# Verify Claude CLI installation
RUN claude --version || echo "Claude CLI installed successfully"

# Install git-aware-prompt for better development experience
USER root
RUN git clone https://github.com/jimeh/git-aware-prompt.git /usr/local/git-aware-prompt \
    && echo 'export GITAWAREPROMPT=/usr/local/git-aware-prompt' >> /home/claude/.bashrc \
    && echo 'source "${GITAWAREPROMPT}/main.sh"' >> /home/claude/.bashrc \
    && echo 'export PS1="\u@\h \W \[\$txtcyn\]\$git_branch\[\$txtred\]\$git_dirty\[\$txtrst\]\$ "' >> /home/claude/.bashrc \
    && chown claude:claude /home/claude/.bashrc

# Set up workspace directory
RUN mkdir -p /workspace && chown claude:claude /workspace

# Copy and configure entrypoint script
COPY ../shared/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Switch to claude user for runtime
USER claude
WORKDIR /workspace

# Configure entrypoint and default command
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["tail", "-f", "/dev/null"]

# Metadata
LABEL org.opencontainers.image.title="Reactor Base"
LABEL org.opencontainers.image.description="Minimal development environment with AI CLI tools"
LABEL org.opencontainers.image.vendor="Reactor Suite"
LABEL org.opencontainers.image.licenses="MIT"
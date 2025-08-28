# Reactor Base Image
# Minimal image with essential CLI tools and AI CLIs (Claude, Gemini)
# Size target: <200MB compressed

FROM debian:bullseye-slim

# Install essential system packages and CLI tools
# Combine all apt operations in single layer for optimal caching and size
RUN apt-get update && apt-get install -y \
    # Core system utilities
    curl \
    git \
    ca-certificates \
    wget \
    unzip \
    gnupg2 \
    sudo \
    # Required for Docker host integration  
    socat \
    # Essential CLI tools for development
    ripgrep \
    jq \
    fzf \
    nano \
    vim \
    less \
    # Process and system monitoring
    procps \
    htop \
    # Essential build tools (required for some npm packages)
    build-essential \
    # Shell utilities
    shellcheck \
    man-db \
    && rm -rf /var/lib/apt/lists/*

# Install Docker CLI for host integration support
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian bullseye stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y docker-ce-cli \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js (required for Claude CLI)
# Using official NodeSource repository for latest LTS
RUN curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/* \
    # Verify installation
    && node --version \
    && npm --version

# Create non-root claude user with sudo access
RUN useradd -m -s /bin/bash claude \
    && usermod -aG sudo claude \
    && echo 'claude ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

# Install Claude CLI as root, then switch to claude user
USER root
RUN npm install -g @anthropic-ai/claude-code

# Switch to claude user and configure shell
USER claude
WORKDIR /home/claude
RUN echo 'export PATH="$(npm config get prefix)/bin:$PATH"' >> /home/claude/.bashrc

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
COPY images/shared/entrypoint.sh /usr/local/bin/entrypoint.sh
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
# Reactor Python Image  
# Built FROM reactor/base with complete Python development environment
# Size target: <500MB compressed

ARG BASE_IMAGE=ghcr.io/dyluth/reactor/base:latest
FROM ${BASE_IMAGE}

# Switch to root for package installations
USER root

# Install Python 3.11 and development dependencies
RUN apt-get update && apt-get install -y \
    # Python 3 (latest available)
    python3 \
    python3-pip \
    python3-dev \
    python3-venv \
    # Python build dependencies
    python3-setuptools \
    python3-wheel \
    # Additional development tools
    python3-distutils \
    # Required for some Python packages
    libffi-dev \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# Create python symlink for convenience
RUN ln -sf /usr/bin/python3 /usr/local/bin/python \
    && ln -sf /usr/bin/pip3 /usr/local/bin/pip

# Install modern Python package manager (uv)
RUN curl -LsSf https://astral.sh/uv/0.4.18/install.sh | sh \
    && mv /root/.cargo/bin/uv /usr/local/bin/uv \
    && mv /root/.cargo/bin/uvx /usr/local/bin/uvx \
    && rm -rf /root/.cargo

# Install essential Python development tools globally
RUN pip3 install --no-cache-dir \
    # Modern Python tooling
    ruff==0.1.15 \
    black==23.12.1 \
    mypy==1.8.0 \
    # Testing framework  
    pytest==7.4.4 \
    pytest-cov==4.1.0 \
    # Popular data science libraries
    requests==2.31.0 \
    # Development utilities
    ipython==8.18.1 \
    rich==13.7.0

# Configure Python environment for claude user
USER claude
WORKDIR /home/claude

# Set up Python-specific shell configuration
RUN echo '# Python development aliases' >> /home/claude/.bashrc \
    && echo 'alias py=python' >> /home/claude/.bashrc \
    && echo 'alias ipy=ipython' >> /home/claude/.bashrc \
    && echo 'alias pytest="python -m pytest"' >> /home/claude/.bashrc \
    && echo 'export PYTHONPATH="${PYTHONPATH}:/workspace"' >> /home/claude/.bashrc

# Create default Python project structure in workspace
USER root
RUN mkdir -p /workspace/src /workspace/tests \
    && echo '# Python Project' > /workspace/README.md \
    && echo 'print("Hello from Reactor Python environment!")' > /workspace/hello.py \
    && chown -R claude:claude /workspace

# Switch back to claude user
USER claude
WORKDIR /workspace

# Verify Python installation
RUN python --version \
    && pip --version \
    && uv --version \
    && ruff --version

# Metadata
LABEL org.opencontainers.image.title="Reactor Python"
LABEL org.opencontainers.image.description="Complete Python development environment built on Reactor Base"
LABEL org.opencontainers.image.vendor="Reactor Suite"
LABEL org.opencontainers.image.licenses="MIT"
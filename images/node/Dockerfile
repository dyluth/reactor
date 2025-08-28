# Reactor Node.js Image
# Built FROM reactor/base with enhanced Node.js development environment
# Size target: <500MB compressed

ARG BASE_IMAGE=ghcr.io/dyluth/reactor/base:latest
FROM ${BASE_IMAGE}

# Switch to root for package installations
USER root

# Note: Base image already has Node.js 20.18.0 installed
# We'll enhance it with development tools and additional utilities

# Install Node.js development tools globally as claude user
USER claude
WORKDIR /home/claude

# Install essential Node.js development packages
RUN npm install -g \
    # TypeScript ecosystem
    typescript@5.3.3 \
    ts-node@10.9.2 \
    @types/node@20.10.6 \
    # Code quality tools
    eslint@8.56.0 \
    prettier@3.1.1 \
    # Package management utilities  
    npm-check-updates@16.14.12 \
    # Build and development tools
    nodemon@3.0.2 \
    concurrently@8.2.2 \
    # Testing framework
    jest@29.7.0 \
    # Useful utilities
    http-server@14.1.1

# Set up Node.js-specific shell configuration
RUN echo '# Node.js development aliases' >> /home/claude/.bashrc \
    && echo 'alias ncu="npm-check-updates"' >> /home/claude/.bashrc \
    && echo 'alias dev="npm run dev"' >> /home/claude/.bashrc \
    && echo 'alias build="npm run build"' >> /home/claude/.bashrc \
    && echo 'alias test="npm test"' >> /home/claude/.bashrc

# Create default Node.js project structure in workspace
USER root
RUN mkdir -p /workspace/src /workspace/tests \
    && cat > /workspace/package.json << 'EOF'
{
  "name": "reactor-workspace",
  "version": "1.0.0", 
  "description": "Default Reactor Node.js workspace",
  "main": "src/index.js",
  "scripts": {
    "start": "node src/index.js",
    "dev": "nodemon src/index.js",
    "build": "tsc",
    "test": "jest",
    "lint": "eslint src/",
    "format": "prettier --write src/"
  },
  "keywords": ["reactor", "nodejs"],
  "author": "Reactor Suite",
  "license": "MIT"
}
EOF

# Create sample files
RUN mkdir -p /workspace/src \
    && cat > /workspace/src/index.js << 'EOF'
console.log('Hello from Reactor Node.js environment!');
console.log('Node.js version:', process.version);
console.log('Platform:', process.platform);
EOF

RUN cat > /workspace/tsconfig.json << 'EOF'
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "commonjs", 
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
EOF

RUN cat > /workspace/.eslintrc.js << 'EOF'
module.exports = {
  env: {
    node: true,
    es2022: true
  },
  extends: ['eslint:recommended'],
  parserOptions: {
    ecmaVersion: 2022,
    sourceType: 'module'
  },
  rules: {
    'no-console': 'off'
  }
};
EOF

RUN cat > /workspace/.prettierrc << 'EOF'
{
  "semi": true,
  "trailingComma": "es5",
  "singleQuote": true,
  "printWidth": 80,
  "tabWidth": 2
}
EOF

# Set proper ownership
RUN chown -R claude:claude /workspace

# Switch back to claude user
USER claude
WORKDIR /workspace

# Verify Node.js development environment
RUN node --version \
    && npm --version \
    && npx typescript --version \
    && npx eslint --version \
    && npx prettier --version

# Metadata
LABEL org.opencontainers.image.title="Reactor Node.js"
LABEL org.opencontainers.image.description="Complete Node.js development environment built on Reactor Base"
LABEL org.opencontainers.image.vendor="Reactor Suite" 
LABEL org.opencontainers.image.licenses="MIT"
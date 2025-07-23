# justfile for gitea-check-service

# Project configuration
APP_NAME := "gitea-check-service"
APP_MAIN_GO_FILE := "main.go"

# Set environment variables for development
export GITEA_URL := env_var_or_default("GITEA_URL", "https://git.example.com")
export TOKEN := env_var_or_default("TOKEN", "test-token")
export PORT := env_var_or_default("PORT", "8080")

# Import common Go justfile targets
import "common-config/justfile-go"

# Default recipe - show available commands
default:
    @just --list

# Project-specific run target with custom startup message  
run-service: build
    @echo "Starting gitea-check-service on port {{PORT}}..."
    ./{{APP_NAME}}

# Development setup
dev-setup: check-tools
    @echo "Setting up development environment..."
    @if [ ! -f .env ]; then \
        echo "Creating .env from .env.example..."; \
        cp .env.example .env; \
        echo "Please edit .env with your configuration"; \
    fi
    go mod download
    @echo "Development setup complete!"

# Docker operations
docker-build:
    @echo "Building Docker image..."
    docker build -t {{APP_NAME}} .

docker-run: docker-build
    @echo "Running Docker container..."
    docker run --rm -p 8080:8080 --env-file .env {{APP_NAME}}

# Docker compose operations
compose-up:
    @echo "Starting with docker-compose..."
    docker-compose up --build -d

compose-down:
    @echo "Stopping docker-compose..."
    docker-compose down

compose-logs:
    @echo "Showing docker-compose logs..."
    docker-compose logs -f

# Git hooks
install-hooks:
    @echo "Installing git hooks..."
    @mkdir -p .git/hooks
    @echo '#!/bin/sh\njust check' > .git/hooks/pre-commit
    @chmod +x .git/hooks/pre-commit
    @echo "Pre-commit hook installed!"

# Release preparation
prepare-release version:
    @echo "Preparing release {{version}}..."
    @echo "Running all checks..."
    just check
    @echo "Tagging version {{version}}..."
    git tag -a {{version}} -m "Release {{version}}"
    @echo "Release {{version}} prepared. Push with: git push origin {{version}}"

# Show project information
info:
    @echo "Gitea Check Service"
    @echo "=================="
    @echo "Go version: $(go version)"
    @echo "Git commit: $(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
    @echo "Git branch: $(git branch --show-current 2>/dev/null || echo 'unknown')"
    @echo "Build time: $(date)"
    @echo ""
    @echo "Environment:"
    @echo "  GITEA_URL: {{GITEA_URL}}"
    @echo "  PORT: {{PORT}}"
# justfile for gitea-check-service

# Default recipe - show available commands
default:
    @just --list

# Set environment variables for development
export GITEA_URL := env_var_or_default("GITEA_URL", "https://git.example.com")
export TOKEN := env_var_or_default("TOKEN", "test-token")
export PORT := env_var_or_default("PORT", "8080")

# Check that required tools are available
check-tools:
    @echo "Checking required tools..."
    @which golangci-lint > /dev/null || (echo "golangci-lint not found in PATH" && exit 1)
    @which govulncheck > /dev/null || (echo "govulncheck not found in PATH" && exit 1)
    @which goimports-reviser > /dev/null || (echo "goimports-reviser not found in PATH" && exit 1)
    @echo "All required tools available!"

# Run the application
run: build
    @echo "Starting gitea-check-service on port {{PORT}}..."
    ./gitea-check-service

# Build the application
build:
    @echo "Building gitea-check-service..."
    go mod tidy
    go build -o gitea-check-service -ldflags="-s -w" .

# Build for production with optimizations
build-prod:
    @echo "Building production binary..."
    CGO_ENABLED=0 go build -o gitea-check-service \
        -ldflags="-s -w -X main.version={{`git describe --tags --always --dirty`}}" \
        -trimpath .

# Run tests with coverage
test:
    @echo "Running tests with coverage..."
    go test -v -race -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $3}'

# Run tests and generate HTML coverage report
test-coverage: test
    @echo "Generating HTML coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Format Go code
format:
    @echo "Formatting Go code..."
    goimports-reviser -rm-unused -set-alias -format -recursive .
    @echo "Code formatted!"

# Check Go code formatting
format-check:
    @echo "Checking Go code formatting..."
    @if [ -n "$(goimports-reviser -list-diff -rm-unused -set-alias -format -recursive .)" ]; then \
        echo "formatting issues found:"; \
        goimports-reviser -list-diff -rm-unused -set-alias -format -recursive .; \
        exit 1; \
    fi
    @echo "Code formatting is correct!"

# Run linting checks
lint:
    @echo "Running linting checks..."
    @echo "→ Running golangci-lint..."
    golangci-lint run
    @echo "All linting checks passed!"

# Fix common linting issues
lint-fix: format
    @echo "Fixing linting issues..."
    golangci-lint run --fix
    @echo "Linting issues fixed!"

# Check for security vulnerabilities
vulncheck:
    @echo "Checking for vulnerabilities..."
    govulncheck ./...

# Run all checks (format-check, test, lint, vulncheck)
check: format-check test lint vulncheck
    @echo "All checks passed!"

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts..."
    rm -f gitea-check-service
    rm -f coverage.out coverage.html
    go clean -cache
    @echo "Clean complete!"

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
    docker build -t gitea-check-service .

docker-run: docker-build
    @echo "Running Docker container..."
    docker run --rm -p 8080:8080 --env-file .env gitea-check-service

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
    @echo '#!/bin/sh\njust test lint' > .git/hooks/pre-commit
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
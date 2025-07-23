# Gitea Check Service

A lightweight Go web service that provides build status information from Gitea repositories via a simple REST API. The service fetches commit status data from Gitea's API and returns it with appropriate HTTP status codes and visual indicators.

## Features

- **REST API** for querying Gitea repository build status
- **HTTP Status Code Mapping** - Response codes match build status (200 for success, 417 for failure, etc.)
- **Visual Status Indicators** - Unicode symbols for quick status identification
- **Health Check Endpoint** - Built-in `/health` endpoint for monitoring
- **Docker Support** - Multi-architecture containers (AMD64/ARM64)
- **Environment-based Configuration** - Easy deployment configuration
- **Request Logging** - Built-in HTTP request logging middleware

## API Endpoints

### GET /status

Retrieves the build status for a Gitea repository.

**Parameters:**
- `owner` (required) - Repository owner/organization name
- `repo` (required) - Repository name

**Example Request:**
```bash
curl "http://localhost:8080/status?owner=myorg&repo=myproject"
```

**Example Response:**
```json
{
  "owner": "myorg",
  "repository": "myproject",
  "branch": "main",
  "state": "success",
  "symbol": "✓"
}
```

**HTTP Status Codes:**
- `200` - Success or Warning
- `202` - Pending
- `204` - Unknown status
- `417` - Build failure
- `500` - Build error or API error

**Status Symbols:**
- `✓` - Success
- `✗` - Failure/Error
- `●` - Pending
- `⚠` - Warning
- `○` - Unknown
- `?` - Unrecognized state

### GET /health

Health check endpoint for monitoring and load balancers.

**Example Response:**
```json
{
  "status": "ok"
}
```

## Configuration

The service is configured via environment variables:

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `GITEA_URL` | Yes | Base URL of your Gitea instance | `https://git.example.com` |
| `TOKEN` | Yes | Gitea API token with repo access | `abc123...` |
| `PORT` | No | HTTP server port (default: 8080) | `8080` |

### Environment Setup

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your Gitea configuration:
   ```bash
   GITEA_URL=https://your-gitea-instance.com
   TOKEN=your-gitea-api-token
   PORT=8080
   ```

### Gitea API Token

To create a Gitea API token:

1. Log into your Gitea instance
2. Go to Settings → Applications
3. Generate a new token with `repo` scope
4. Copy the token to your `.env` file

## Installation & Deployment

### Option 1: Docker (Recommended)

Using the pre-built multi-architecture image:

```bash
docker run -d \
  --name gitea-check-service \
  -p 8080:8080 \
  -e GITEA_URL=https://your-gitea-instance.com \
  -e TOKEN=your-gitea-api-token \
  ghcr.io/fred-drake/gitea-check-service:latest
```

### Option 2: Docker Compose

```bash
# Copy and configure environment
cp .env.example .env
# Edit .env with your settings

# Start the service
docker-compose up -d
```

### Option 3: Build from Source

**Prerequisites:**
- Go 1.23 or later

**Steps:**
```bash
# Clone the repository
git clone https://github.com/fred-drake/gitea-check-service.git
cd gitea-check-service

# Build the application
go mod download
go build -o gitea-check-service

# Configure environment
cp .env.example .env
# Edit .env with your settings

# Run the service
./gitea-check-service
```

## Usage Examples

### Check Build Status

```bash
# Check status for a repository
curl "http://localhost:8080/status?owner=myorg&repo=myproject"

# Check with response code handling
curl -w "HTTP Status: %{http_code}\n" \
  "http://localhost:8080/status?owner=myorg&repo=myproject"
```

### Health Check

```bash
curl "http://localhost:8080/health"
```

### Integration with Scripts

```bash
#!/bin/bash
RESPONSE=$(curl -s -w "%{http_code}" "http://localhost:8080/status?owner=myorg&repo=myproject")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [ "$HTTP_CODE" -eq 200 ]; then
    echo "✅ Build successful"
    exit 0
elif [ "$HTTP_CODE" -eq 202 ]; then
    echo "⏳ Build pending"
    exit 1
else
    echo "❌ Build failed (HTTP $HTTP_CODE)"
    exit 1
fi
```

## Development

### Project Structure

```
.
├── main.go                 # Main application code
├── main_test.go           # Comprehensive test suite (83% coverage)
├── go.mod                  # Go module definition
├── justfile               # Development task runner
├── Dockerfile              # Container build instructions
├── docker-compose.yml      # Local development setup
├── .env.example           # Environment template
├── .gitignore             # Git ignore rules
├── .github/
│   └── workflows/
│       └── docker.yml     # CI/CD pipeline
└── README.md              # This file
```

### Local Development

This project uses [just](https://github.com/casey/just) as a command runner. Install it first:

```bash
# Install just
curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to ~/bin
# Or via package manager (e.g., brew install just, apt install just)

# Set up development environment
just dev-setup

# View available commands
just --list
```

**Common Commands:**
```bash
# Run the application
just run

# Build the application
just build

# Format Go code
just format

# Run tests with coverage (currently 83%+)
just test

# Run linting checks (golangci-lint, goimports)
just lint

# Run all quality checks (format-check, test, lint, vulncheck)
just check

# Fix formatting and import issues
just lint-fix
```

### Testing

The project maintains high test coverage (83%+) with comprehensive testing of:

- **Unit Tests**: All business logic functions tested
- **Integration Tests**: Full HTTP request/response cycles
- **Error Handling**: Network failures, invalid responses, missing data
- **Edge Cases**: Malformed JSON, connection timeouts, API errors
- **Mock Testing**: External API calls properly mocked for reliable testing

**Running Tests:**
```bash
# Run all tests
go test -v

# Run tests with coverage report
just test

# Generate HTML coverage report
go test -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Docker Development

```bash
# Build Docker image
just docker-build

# Run with Docker
just docker-run

# Use docker-compose
just compose-up
just compose-logs
just compose-down
```

## CI/CD

The project includes a GitHub Actions workflow that:

- **Triggers** on pushes to `main`/`master` and version tags
- **Builds** multi-architecture Docker images (AMD64/ARM64)
- **Publishes** to GitHub Container Registry
- **Tags** images appropriately based on git refs

### Automatic Container Publishing

When you push to the main branch or create a version tag, the workflow automatically:

1. Builds containers for both AMD64 and ARM64
2. Publishes to `ghcr.io/fred-drake/gitea-check-service`
3. Tags with `latest`, branch names, or semantic versions

**Available Tags:**
- `latest` - Latest main/master branch
- `main` - Main branch builds  
- `v1.2.3` - Specific version tags
- `1.2.3`, `1.2`, `1` - Semantic version variants

## Troubleshooting

### Common Issues

**"GITEA_URL environment variable is required"**
- Ensure `.env` file exists and contains `GITEA_URL`
- Verify environment variables are properly exported

**"TOKEN environment variable is required"**
- Create a Gitea API token with `repo` scope
- Add token to `.env` file or environment

**"Failed to get repository info: 401"**
- Check that your API token is valid
- Verify token has access to the requested repository
- Ensure token hasn't expired

**"Failed to get repository info: 404"**
- Verify the owner and repository names are correct
- Check that the repository exists and is accessible
- Ensure your token has read access to the repository

### Debugging

Enable debug logging by checking the service logs:

```bash
# Docker
docker logs gitea-check-service

# Direct execution
./gitea-check-service
```

The service logs all HTTP requests with timing information.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Security

- API tokens are passed via environment variables (not logged)
- HTTP client has reasonable timeouts (10 seconds)
- No sensitive information is exposed in API responses
- Container runs as non-root user in production

For security issues, please email the maintainers directly rather than using public issues.
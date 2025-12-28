# Project Overview

OctoCatalog is a lightweight Go-based service that provides external data sources for Slack select dialog modals. It acts as a webhook endpoint that validates incoming Slack requests and returns filtered catalog options based on JSON configuration.

**Primary Use Case:** Provide dynamic, searchable lists of projects/repositories for Slack interactive components.

# Tech Stack

- **Language:** Go 1.25.5
- **API:** Slack Block Kit API (external select menus)
- **Security:** HMAC-SHA256 signature verification for Slack webhooks
- **Deployment:** Docker with multi-stage builds (scratch-based runtime)
- **Configuration:** Environment variables and JSON catalog files

# Coding Guidelines

## Go Code Style

- Follow standard Go conventions and idiomatic Go patterns
- Use meaningful variable and function names
- Keep functions focused and concise
- Prefer composition over inheritance
- Use the standard library whenever possible

## Structure and Organization

- Main application logic in `main.go`
- Tests in `main_test.go` using Go's standard testing package
- Configuration through environment variables
- External catalog data in JSON files

## Error Handling

- Always wrap errors with context using `fmt.Errorf("context: %w", err)`
- Log errors before returning HTTP error responses
- Use appropriate HTTP status codes (400, 401, 415, 500)

## Security

- **CRITICAL:** Always validate Slack signatures using HMAC-SHA256
- Check timestamp to prevent replay attacks (5-minute tolerance)
- Never log sensitive data like signing secrets
- Use constant-time comparison for signature validation (`hmac.Equal`)

# Project Architecture

```
/
├── main.go           # Main application entry point and HTTP handlers
├── main_test.go      # Comprehensive test suite
├── catalog.json      # Runtime catalog configuration (gitignored)
├── go.mod            # Go module definition
├── Dockerfile        # Multi-stage Docker build
└── docker-compose.yml
```

## Key Components

- **Config:** Environment-based configuration (PORT, SLACK_SIGNING_SECRET, CONFIG_FILE)
- **CatalogEntry:** JSON-based catalog with actionId and options
- **SlackRequest/SlackResponse:** Slack API request/response structures
- **handleRequest:** Main HTTP handler with signature validation and filtering

# Testing Practices

- Use Go's standard `testing` package
- Create table-driven tests when appropriate
- Use `httptest` for HTTP handler testing
- Test helper functions: `setupTestCatalog()`, `generateTestSignature()`
- **Always test:**
  - Happy paths with valid requests
  - Error cases (missing payload, invalid JSON, wrong content-type)
  - Signature validation
  - Filtering logic (case-insensitive, substring matching)
  - Both form-encoded and JSON request formats

## Running Tests

```bash
go test -v
go test -cover
```

# Build and Run

## Development

```bash
# Set required environment variables
export SLACK_SIGNING_SECRET=your_secret_here
export PORT=8080
export CONFIG_FILE=catalog.json

# Run locally
go run main.go

# Run tests
go test -v
```

## Docker

```bash
# Build
docker build -t octocatalog .

# Run
docker run -p 8080:8080 \
  -e SLACK_SIGNING_SECRET=your_secret \
  -e PORT=8080 \
  octocatalog
```

## Docker Compose

```bash
docker-compose up --build
```

# Request/Response Format

## Incoming Slack Request

- **Content-Type:** `application/x-www-form-urlencoded` or `application/json`
- **Headers:** `X-Slack-Request-Timestamp`, `X-Slack-Signature`
- **Payload:** `action_id`, `value` (search query)

## Response Format

Returns JSON with filtered options matching the query (case-insensitive substring match on both text and value fields).

# Important Notes

- The catalog file (`catalog.json`) is gitignored - use `catalog.json.example` as a template
- Signature validation prevents unauthorized requests
- Filtering is case-insensitive and matches on both `text` and `value` fields
- Empty query returns all options
- Multi-stage Docker build keeps runtime image minimal (scratch-based)

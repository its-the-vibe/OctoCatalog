# OctoCatalog

A service which provides a list of projects or repos to use as an external data source for Slack select dialog modals.

## Features

- Written in Go 1.24
- Validates Slack request signatures for security
- Configurable via environment variables
- Docker support with scratch-based runtime image
- JSON-based catalog configuration

## Configuration

### Environment Variables

- `PORT` - Port to run the server on (default: `8080`)
- `SLACK_SIGNING_SECRET` - Slack signing secret for request validation (required)
- `CONFIG_FILE` - Path to the catalog configuration file (default: `catalog.json`)

### Catalog Configuration

The catalog is defined in a JSON file (e.g., `catalog.json`) with the following structure:

```json
[
  {
    "actionId": "SlackCompose",
    "options": [
      {
        "text": "InnerGate",
        "value": "InnerGate"
      },
      {
        "text": "OctoSlack",
        "value": "OctoSlack"
      }
    ]
  }
]
```

## Running the Service

### Using Go

1. Copy `.env.example` to `.env` and set your `SLACK_SIGNING_SECRET`
2. Run the service:

```bash
export SLACK_SIGNING_SECRET=your_secret_here
go run main.go
```

### Using Docker Compose

1. Copy `.env.example` to `.env` and set your `SLACK_SIGNING_SECRET`
2. Run with Docker Compose:

```bash
docker-compose up --build
```

### Using Docker

Build and run the Docker container:

```bash
docker build -t octocatalog .
docker run -p 8080:8080 \
  -e SLACK_SIGNING_SECRET=your_secret_here \
  -e PORT=8080 \
  octocatalog
```

## API

The service responds to POST requests from Slack with the following format:

**Request:**
```json
{
  "type": "block_suggestion",
  "action_id": "your_select_action_id",
  "block_id": "block_identifier",
  "value": "abc"
}
```

**Response:**
```json
{
  "options": [
    {
      "text": {
        "type": "plain_text",
        "text": "Option 1"
      },
      "value": "value1"
    }
  ]
}
```

The service matches the `action_id` from the request to the `actionId` in the catalog configuration and returns the corresponding options.

# Story API

A REST API service that consumes Hacker News stories from Kafka and exposes them via HTTP endpoints.

## Building

```bash
go build -o story-api
```

## Running

```bash
./story-api
```

The server listens on port 8080 by default (configurable in `config.yaml`).

## API Endpoints

### GET /stories

Returns all stories with optional filtering and sorting.

**Query Parameters:**

- `minScore` (int): Minimum score threshold (default: 0)
- `maxScore` (int): Maximum score threshold (default: unlimited)
- `since` (RFC3339): Return stories after this timestamp (e.g., `2025-01-13T00:00:00Z`)
- `until` (RFC3339): Return stories before this timestamp
- `sort` (string): Sort order - `latest` (default), `oldest`, or `popularity`

**Examples:**

```bash
# All stories sorted by latest (default)
curl http://localhost:8080/stories

# Stories with score >= 100, sorted by popularity
curl http://localhost:8080/stories?minScore=100&sort=popularity

# Stories from the last hour, oldest first
curl 'http://localhost:8080/stories?since=2025-01-12T23:00:00Z&sort=oldest'

# Stories with score between 50 and 200
curl 'http://localhost:8080/stories?minScore=50&maxScore=200'
```

### GET /health

Health check endpoint that returns `{"status":"ok"}`.

## Configuration

Edit `config.yaml` to customize:

- Kafka broker address, topic, and consumer group
- TLS certificate paths
- API port

Relative certificate paths are resolved relative to the config file location.

## How It Works

1. On startup, the service connects to Kafka using TLS
2. It consumes messages from the configured topic and consumer group
3. Stories are stored in memory (map by ID)
4. The REST API filters and sorts stories on demand
5. Graceful shutdown on SIGINT/SIGTERM

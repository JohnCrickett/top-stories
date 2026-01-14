# Story API

A REST API service that consumes Hacker News stories from Kafka and exposes them via HTTP endpoints.

## Building

```bash
go build -o story-api
```

## Running

```bash
# Run with default config (config.yaml)
./story-api

# Run with a specific config file
./story-api -config config.ask-hn.yaml
./story-api -config config.rust.yaml
```

The server listens on port 8080 by default (configurable in the config file).

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
- **Consumer-side filtering** (new)

Relative certificate paths are resolved relative to the config file location.

### Consumer-Side Filtering

Configure filters in the `filter` section of `config.yaml` to control which stories are consumed and stored. This enables running multiple instances with different filters, each storing only relevant stories.

**Filter Options:**

- `story_types` (list): Story types to consume (e.g., `["story", "ask", "show", "poll"]`)
  - Leave empty to consume all types
  
- `keywords` (list): Keywords to match in story titles (case-insensitive)
  - A story matches if it contains ANY keyword
  - Leave empty to match all stories
  
- `minimum_score` (int): Only consume stories with score >= this value
  - Set to 0 to disable minimum score filtering

**Example Configurations:**

1. **Ask HN Stories Only** - `config.ask-hn.yaml`
   ```yaml
   filter:
     story_types: ["ask"]
     keywords: []
     minimum_score: 0
   ```
   Run: `./story-api` (loads default `config.yaml`, or modify to use this config)

2. **Rust-Related Stories** - `config.rust.yaml`
   ```yaml
   filter:
     story_types: []
     keywords: ["rust"]
     minimum_score: 0
   ```

3. **High-Scoring Stories Only** - `config.top.yaml`
   ```yaml
   filter:
     story_types: []
     keywords: []
     minimum_score: 100
   ```

4. **Show HN with Score Filter** - `config.show-hn.yaml`
   ```yaml
   filter:
     story_types: ["story"]
     keywords: ["show"]
     minimum_score: 10
   ```

**Running Multiple Instances:**

```bash
# Terminal 1: Ask HN stories on port 8081
./story-api -config config.ask-hn.yaml &

# Terminal 2: Rust stories on port 8082
./story-api -config config.rust.yaml &

# Terminal 3: High-scoring stories on port 8083
./story-api -config config.top.yaml &

# Terminal 4: Show HN with score filter on port 8084
./story-api -config config.show-hn.yaml &
```

Or use the provided script:
```bash
./run-instances.sh
```

Each instance consumes from the same Kafka topic but filters at ingest time, storing only matching stories. The `/stories` API returns only stories matching that instance's filters.

## How It Works

1. On startup, the service connects to Kafka using TLS
2. It consumes messages from the configured topic and consumer group
3. **Consumer-side filters are applied at ingest time** - non-matching stories are discarded immediately
4. Matching stories are stored in memory (map by ID)
5. The REST API filters and sorts stored stories on demand
6. Graceful shutdown on SIGINT/SIGTERM

This design allows multiple instances to be deployed with different filters, creating a distributed system where each instance is optimized for its specific story subset.

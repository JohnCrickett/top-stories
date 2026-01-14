# Web Server

A Go-based server that serves the React frontend and provides API configuration.

## Features

- Serves static React frontend files
- Provides `/api/config` endpoint with API configuration
- Single page application (SPA) routing support
- Simple YAML configuration

## Building

```bash
go build -o web-server
```

## Running

```bash
./web-server -config config.yaml
```

Default port is 3000 if not specified in config.

## Configuration

Edit `config.yaml` to configure:

```yaml
port: 3000
refresh_interval: 60

apis:
  - name: "Top Stories"
    url: "http://localhost:8080/stories"
  - name: "Ask HN"
    url: "http://localhost:8081/stories"
```

- `port`: Port to listen on
- `refresh_interval`: Default auto-refresh interval in seconds
- `apis`: List of API sources
  - `name`: Display name for the tab
  - `url`: Full URL to the `/stories` endpoint

## API Endpoints

### GET /api/config

Returns the configuration with list of available APIs and refresh interval.

**Response:**

```json
{
  "apis": [
    {
      "name": "Top Stories",
      "url": "http://localhost:8080/stories"
    }
  ],
  "refreshInterval": 60
}
```

### GET / and other routes

Serves the React frontend static files.

## Frontend Build

The web server expects the frontend to be built. Build the frontend first:

```bash
cd ../frontend
npm run build
```

Then the web server will serve the files from `../frontend/build`.

## Development Workflow

1. Build the frontend: `cd frontend && npm run build`
2. Build the web server: `cd web-server && go build`
3. Run the web server: `./web-server -config config.yaml`
4. Visit `http://localhost:3000`

## Changing API Configuration

Edit `config.yaml` and restart the server. The frontend will fetch the new configuration on load.

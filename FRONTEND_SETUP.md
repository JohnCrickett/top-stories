# Frontend Setup Guide

This guide explains how to set up and run the web frontend for the Top Stories project.

## Architecture Overview

The system has three components:

1. **Backend Scraper** (`backend/`) - Fetches stories from Hacker News and publishes to Kafka
2. **Story APIs** (`story-api/`) - Consume from Kafka and expose HTTP endpoints
3. **Web Frontend** (`frontend/` + `web-server/`) - React UI that displays stories from multiple APIs

## Prerequisites

- Node.js 14+ and npm
- Go 1.16+ (for the web server)

## Quick Start

### Step 1: Build the Frontend

```bash
cd frontend
npm install
npm run build
```

This creates a production build in `frontend/build/`.

### Step 2: Build the Web Server

```bash
cd ../web-server
go build -o web-server
```

### Step 3: Configure API Sources

Edit `web-server/config.yaml` to specify which story APIs to connect to:

```yaml
port: 3000
refresh_interval: 60

apis:
  - name: "Top Stories"
    url: "http://localhost:8080/stories"
  - name: "Ask HN"
    url: "http://localhost:8081/stories"
  - name: "Show HN"
    url: "http://localhost:8082/stories"
```

### Step 4: Run the Backend Components

Start the story API instances in separate terminals:

```bash
# Terminal 1: Top Stories
cd story-api
./story-api -config config.top.yaml

# Terminal 2: Ask HN
./story-api -config config.ask-hn.yaml

# Terminal 3: Show HN
./story-api -config config.show-hn.yaml
```

Or use the provided script:

```bash
cd story-api
./run-instances.sh
```

### Step 5: Run the Web Server

```bash
cd web-server
./web-server -config config.yaml
```

The frontend will be available at `http://localhost:3000`.

## Features

- **Tabbed interface**: Switch between different story sources
- **Auto-refresh**: Stories refresh every minute (configurable)
- **Manual refresh**: Click the "Refresh" button to update immediately
- **Responsive design**: Works on desktop and mobile
- **Hacker News style**: Clean, minimal UI similar to Hacker News

## Configuration Details

### Refresh Interval

The refresh interval (in minutes) can be:
- Set in `web-server/config.yaml` as the default
- Changed in the UI using the interval input field

### Adding New API Sources

1. Start a new story API instance with a different filter/config
2. Add it to `web-server/config.yaml`
3. Restart the web server

Example:

```yaml
apis:
  - name: "Top Stories"
    url: "http://localhost:8080/stories"
  - name: "Rust Stories"
    url: "http://localhost:8084/stories"
```

## Development

### Frontend Development Mode

```bash
cd frontend
npm start
```

This starts a development server with hot reloading on `http://localhost:3000`.

**Note**: The frontend needs to connect to story APIs. Update the API URLs in your code or use a local proxy if needed.

### Debugging

- Open browser DevTools (F12)
- Check the Console tab for errors
- The Network tab shows API calls to `/api/config` and story endpoints

## Troubleshooting

### "Failed to load API configuration"

- Ensure the web server is running
- Check that `web-server/config.yaml` exists and is valid YAML
- Verify the port is correct

### Stories not loading

- Check that story API instances are running
- Verify the URLs in `web-server/config.yaml` match the running APIs
- Check browser console for CORS errors (need to enable CORS on story APIs)

### Port already in use

- Change the port in `web-server/config.yaml`
- Or kill the existing process: `lsof -i :3000` then `kill -9 <PID>`

## Production Deployment

1. Build the frontend:
   ```bash
   cd frontend
   npm run build
   ```

2. Build the web server:
   ```bash
   cd web-server
   go build -o web-server
   ```

3. Deploy the `web-server` binary and `config.yaml`

4. Ensure story API instances are accessible from the deployment environment

5. Update `config.yaml` with the correct story API URLs

## File Structure

```
top-stories/
├── frontend/              # React application
│   ├── public/           # Static HTML
│   ├── src/              # React components
│   │   ├── App.js
│   │   ├── components/   # React components
│   │   │   ├── Tabs.js
│   │   │   ├── Stories.js
│   │   │   ├── StoryItem.js
│   │   │   └── Controls.js
│   │   └── ...
│   ├── package.json
│   └── README.md
├── web-server/           # Go web server
│   ├── main.go
│   ├── config.yaml       # Server & API configuration
│   └── README.md
├── story-api/            # Story API service
├── backend/              # Hacker News scraper
└── FRONTEND_SETUP.md     # This file
```

## Component Overview

### Story Display

Each story shows:
- Rank number
- Story title (clickable link)
- Domain name
- Points/score
- Author
- Time since posted

### Controls

- **Refresh button**: Manually refresh all stories
- **Interval input**: Change auto-refresh interval (1-300 minutes)
- **Last updated**: Shows when stories were last refreshed

### Tabs

Each tab represents one API source and shows stories from that source independently.

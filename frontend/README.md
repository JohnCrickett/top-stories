# Top Stories Frontend

A React-based web frontend for displaying stories from multiple API sources in separate tabs, similar to Hacker News.

## Features

- **Multi-source tabs**: Display stories from multiple APIs in separate tabs
- **Auto-refresh**: Automatically refreshes stories at a configurable interval (default 1 minute)
- **Manual refresh**: Click the Refresh button to update immediately
- **Clean design**: Hacker News-inspired UI with responsive layout
- **Dynamic configuration**: API endpoints are fetched from the backend config

## Setup

### Prerequisites

- Node.js 14+ and npm

### Installation

```bash
cd frontend
npm install
```

### Building for Production

```bash
npm run build
```

This creates a `build` directory with optimized production files.

### Development

```bash
npm start
```

Starts the development server on `http://localhost:3000` (or next available port).

## Configuration

The frontend fetches its configuration from the backend at `/api/config`. This includes:

- List of APIs and their URLs
- Default refresh interval

Edit the `web-server/config.yaml` file to change the APIs and refresh interval.

## API Response Format

The frontend expects story APIs to return a JSON array of stories with the following structure:

```json
[
  {
    "id": 123456,
    "title": "Story Title",
    "url": "https://example.com/article",
    "by": "username",
    "score": 42,
    "time": 1705084800,
    "type": "story"
  },
  ...
]
```

- `id`: Unique story identifier
- `title`: Story headline
- `url`: Link to the original story (optional)
- `by`: Username of the poster
- `score`: Number of upvotes/points
- `time`: Unix timestamp when story was posted
- `type`: Story type (story, ask, show, poll, job)

## Component Structure

- `App.js`: Main component, handles config fetching and tab management
- `Tabs.js`: Tab navigation between different APIs
- `Stories.js`: Displays the list of stories for the active tab
- `StoryItem.js`: Individual story row with metadata
- `Controls.js`: Refresh controls and interval configuration

## Styling

The frontend uses Hacker News-inspired styling:

- Orange header (#ff6600)
- Clean white background (#fff)
- Light gray accents (#f6f6ef)
- Minimal, text-focused design

All styling is handled through CSS modules in each component directory.

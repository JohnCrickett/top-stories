# Top Stories

Grab top stories from key developer sites

## Backend - Hacker News Scraper

A Go-based scraper that fetches and displays stories from Hacker News.

### Running the Backend

1. Navigate to the backend directory:
   ```bash
   cd backend
   ```

2. Run the scraper:
   ```bash
   go run main.go
   ```

The scraper will:
- Fetch the top 30 stories from both `/topstories` and `/newstories` endpoints
- Display new stories with their titles and URLs
- Poll for new stories every minute
- Deduplicate stories across polls

### Stopping the Scraper

Press `CTRL-C` to gracefully shutdown the scraper.

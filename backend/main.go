package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	baseURL         = "https://hacker-news.firebaseio.com/v0"
	topStoriesURL   = baseURL + "/topstories.json"
	newStoriesURL   = baseURL + "/newstories.json"
	itemURL         = baseURL + "/item/%d.json"
	pollInterval    = 1 * time.Minute
	storiesToFetch  = 30
)

type Story struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	By    string `json:"by"`
	Score int    `json:"score"`
	Time  int64  `json:"time"`
	Type  string `json:"type"`
}

type Scraper struct {
	client      *http.Client
	seenStories map[int]bool
	mu          sync.Mutex
	done        chan struct{}
}

func NewScraper() *Scraper {
	return &Scraper{
		client:      &http.Client{Timeout: 10 * time.Second},
		seenStories: make(map[int]bool),
		done:        make(chan struct{}),
	}
}

// fetchStoryIDs fetches story IDs from the given endpoint
func (s *Scraper) fetchStoryIDs(url string) ([]int, error) {
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch story IDs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var ids []int
	if err := json.Unmarshal(body, &ids); err != nil {
		return nil, fmt.Errorf("failed to unmarshal story IDs: %w", err)
	}

	return ids, nil
}

// fetchStory fetches details for a single story
func (s *Scraper) fetchStory(id int) (*Story, error) {
	url := fmt.Sprintf(itemURL, id)
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch story: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var story Story
	if err := json.Unmarshal(body, &story); err != nil {
		return nil, fmt.Errorf("failed to unmarshal story: %w", err)
	}

	return &story, nil
}

// addAndPrintStory adds a story to seen map and prints if new
func (s *Scraper) addAndPrintStory(story *Story) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.seenStories[story.ID] {
		s.seenStories[story.ID] = true
		fmt.Printf("[NEW] %s\n", story.Title)
		if story.URL != "" {
			fmt.Printf("      %s\n", story.URL)
		}
	}
}

// pollStories polls both top and new stories
func (s *Scraper) pollStories() {
	urls := []string{topStoriesURL, newStoriesURL}

	for _, url := range urls {
		ids, err := s.fetchStoryIDs(url)
		if err != nil {
			fmt.Printf("Error fetching story IDs from %s: %v\n", url, err)
			continue
		}

		// Only fetch the top 30
		if len(ids) > storiesToFetch {
			ids = ids[:storiesToFetch]
		}

		for _, id := range ids {
			story, err := s.fetchStory(id)
			if err != nil {
				fmt.Printf("Error fetching story %d: %v\n", id, err)
				continue
			}

			// Only print stories with titles (filter out deleted/dead stories)
			if story.Title != "" {
				s.addAndPrintStory(story)
			}
		}
	}
}

// run starts the polling loop
func (s *Scraper) run() {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Poll immediately on startup
	fmt.Println("Starting Hacker News scraper...")
	s.pollStories()

	for {
		select {
		case <-ticker.C:
			s.pollStories()
		case <-s.done:
			fmt.Println("\nScraper shutdown complete.")
			return
		}
	}
}

// stop signals the scraper to stop
func (s *Scraper) stop() {
	close(s.done)
}

func main() {
	scraper := NewScraper()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down gracefully...")
		scraper.stop()
	}()

	scraper.run()
}

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"gopkg.in/yaml.v3"
)

const (
	baseURL       = "https://hacker-news.firebaseio.com/v0"
	topStoriesURL = baseURL + "/topstories.json"
	newStoriesURL = baseURL + "/newstories.json"
	itemURL       = baseURL + "/item/%d.json"
)

type Config struct {
	Kafka  KafkaConfig  `yaml:"kafka"`
	Scraper ScraperConfig `yaml:"scraper"`
}

type KafkaConfig struct {
	Broker        string `yaml:"broker"`
	Topic         string `yaml:"topic"`
	CACertPath    string `yaml:"ca_cert_path"`
	ClientCertPath string `yaml:"client_cert_path"`
	ClientKeyPath string `yaml:"client_key_path"`
}

type ScraperConfig struct {
	PollIntervalSeconds int `yaml:"poll_interval_seconds"`
	StoriesToFetch      int `yaml:"stories_to_fetch"`
}

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
	client       *http.Client
	seenStories  map[int]bool
	mu           sync.Mutex
	config       Config
	kafkaWriter  *kafka.Writer
	pollInterval time.Duration
	storiesToFetch int
	ctx          context.Context
	cancel       context.CancelFunc
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Resolve cert paths relative to config file directory
	configDir := filepath.Dir(path)
	if configDir == "." {
		configDir = ""
	} else if configDir != "" {
		configDir += string(filepath.Separator)
	}

	if cfg.Kafka.CACertPath != "" && cfg.Kafka.CACertPath[0] != '/' {
		cfg.Kafka.CACertPath = configDir + cfg.Kafka.CACertPath
	}
	if cfg.Kafka.ClientCertPath != "" && cfg.Kafka.ClientCertPath[0] != '/' {
		cfg.Kafka.ClientCertPath = configDir + cfg.Kafka.ClientCertPath
	}
	if cfg.Kafka.ClientKeyPath != "" && cfg.Kafka.ClientKeyPath[0] != '/' {
		cfg.Kafka.ClientKeyPath = configDir + cfg.Kafka.ClientKeyPath
	}

	return &cfg, nil
}

func createKafkaWriter(cfg KafkaConfig) (*kafka.Writer, error) {
	keypair, err := tls.LoadX509KeyPair(cfg.ClientCertPath, cfg.ClientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client cert/key: %w", err)
	}

	caCert, err := os.ReadFile(cfg.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA cert")
	}

	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		TLS: &tls.Config{
			Certificates: []tls.Certificate{keypair},
			RootCAs:      caCertPool,
		},
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{cfg.Broker},
		Topic:   cfg.Topic,
		Dialer:  dialer,
	})

	fmt.Printf("[DEBUG] Kafka writer configured for broker: %s, topic: %s\n", cfg.Broker, cfg.Topic)
	return writer, nil
}

func NewScraper(cfg Config) (*Scraper, error) {
	writer, err := createKafkaWriter(cfg.Kafka)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka writer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Scraper{
		client:         &http.Client{Timeout: 10 * time.Second},
		seenStories:    make(map[int]bool),
		config:         cfg,
		kafkaWriter:    writer,
		pollInterval:   time.Duration(cfg.Scraper.PollIntervalSeconds) * time.Second,
		storiesToFetch: cfg.Scraper.StoriesToFetch,
		ctx:            ctx,
		cancel:         cancel,
	}, nil
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

// publishStoryToKafka publishes a story to Kafka with retry logic
func (s *Scraper) publishStoryToKafka(story *Story, source string) error {
	storyJSON, err := json.Marshal(story)
	if err != nil {
		return fmt.Errorf("failed to marshal story: %w", err)
	}

	maxRetries := 5
	backoffDuration := time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check if we're shutting down before each attempt
		if s.ctx.Err() != nil {
			return fmt.Errorf("shutdown in progress")
		}

		msg := kafka.Message{
			Key:   []byte(source),
			Value: storyJSON,
		}

		// Publish with a timeout using goroutine
		errChan := make(chan error, 1)
		go func() {
			writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			errChan <- s.kafkaWriter.WriteMessages(writeCtx, msg)
		}()

		select {
		case err := <-errChan:
			if err == nil {
				fmt.Printf("[PUBLISHED] %s | %s (Story ID: %d)\n", source, story.Title, story.ID)
				return nil
			}
			// Publish failed, will retry
		case <-time.After(6 * time.Second):
			err = fmt.Errorf("publish timeout")
		case <-s.ctx.Done():
			return fmt.Errorf("shutdown in progress")
		}

		// Check if context was cancelled (shutdown signal)
		if s.ctx.Err() != nil {
			return fmt.Errorf("shutdown in progress")
		}

		if attempt < maxRetries {
			fmt.Printf("[RETRY] Publishing story %d (attempt %d/%d, waiting %v): %v\n",
				story.ID, attempt+1, maxRetries, backoffDuration, err)
			
			// Sleep with early exit on shutdown
			select {
			case <-time.After(backoffDuration):
				backoffDuration *= 2
			case <-s.ctx.Done():
				return fmt.Errorf("shutdown in progress")
			}
		}
	}

	return fmt.Errorf("failed to publish story %d after %d retries: %v", story.ID, maxRetries, err)
}

// addAndPublishStory adds a story to seen map and publishes if new
func (s *Scraper) addAndPublishStory(story *Story, source string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.seenStories[story.ID] {
		s.seenStories[story.ID] = true
		fmt.Printf("[NEW] %s\n", story.Title)
		if story.URL != "" {
			fmt.Printf("      %s\n", story.URL)
		}

		// Publish to Kafka
		if err := s.publishStoryToKafka(story, source); err != nil {
			fmt.Printf("[ERROR] %v\n", err)
		}
	}
}

// pollStories polls both top and new stories
func (s *Scraper) pollStories() {
	sources := map[string]string{
		"top": topStoriesURL,
		"new": newStoriesURL,
	}

	for source, url := range sources {
		// Check if shutting down
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		ids, err := s.fetchStoryIDs(url)
		if err != nil {
			fmt.Printf("Error fetching story IDs from %s: %v\n", url, err)
			continue
		}

		// Only fetch the top N
		if len(ids) > s.storiesToFetch {
			ids = ids[:s.storiesToFetch]
		}

		for _, id := range ids {
			// Check if shutting down
			select {
			case <-s.ctx.Done():
				return
			default:
			}

			story, err := s.fetchStory(id)
			if err != nil {
				fmt.Printf("Error fetching story %d: %v\n", id, err)
				continue
			}

			// Only process stories with titles (filter out deleted/dead stories)
			if story.Title != "" {
				s.addAndPublishStory(story, source)
			}
		}
	}
}

// run starts the polling loop
func (s *Scraper) run() {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	// Poll immediately on startup
	fmt.Println("Starting Hacker News scraper...")
	s.pollStories()

	for {
		select {
		case <-ticker.C:
			s.pollStories()
		case <-s.ctx.Done():
			fmt.Println("\nShutting down Kafka writer...")
			// Give pending writes 2 seconds to complete
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			
			// Try to close gracefully
			closeDone := make(chan struct{})
			go func() {
				if err := s.kafkaWriter.Close(); err != nil {
					fmt.Printf("Error closing Kafka writer: %v\n", err)
				}
				closeDone <- struct{}{}
			}()
			
			select {
			case <-closeDone:
				fmt.Println("Scraper shutdown complete.")
			case <-shutdownCtx.Done():
				fmt.Println("Forced shutdown after timeout.")
			}
			return
		}
	}
}

// stop signals the scraper to stop
func (s *Scraper) stop() {
	s.cancel()
}

func main() {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	scraper, err := NewScraper(*cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize scraper: %v\n", err)
		os.Exit(1)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\n[SIGNAL] Received %v, shutting down...\n", sig)
		scraper.stop()
		
		// Force exit if cleanup takes too long
		fmt.Println("[WAITING] Giving 3 seconds for graceful shutdown...")
		time.Sleep(3 * time.Second)
		fmt.Println("[SHUTDOWN] Timeout reached, forcing exit...")
		os.Exit(0)
	}()

	scraper.run()
}

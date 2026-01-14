package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Kafka  KafkaConfig  `yaml:"kafka"`
	API    APIConfig    `yaml:"api"`
	Filter FilterConfig `yaml:"filter"`
}

type KafkaConfig struct {
	Broker         string `yaml:"broker"`
	Topic          string `yaml:"topic"`
	ConsumerGroup  string `yaml:"consumer_group"`
	CACertPath     string `yaml:"ca_cert_path"`
	ClientCertPath string `yaml:"client_cert_path"`
	ClientKeyPath  string `yaml:"client_key_path"`
}

type APIConfig struct {
	Port int `yaml:"port"`
}

type FilterConfig struct {
	StoryTypes   []string `yaml:"story_types"`
	Keywords     []string `yaml:"keywords"`
	MinimumScore int      `yaml:"minimum_score"`
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

type StoryStore struct {
	mu      sync.RWMutex
	stories map[int]*Story // ID -> Story
}

type Server struct {
	store    *StoryStore
	config   Config
	reader   *kafka.Reader
	ctx      context.Context
	cancel   context.CancelFunc
	filter   *StoryFilter
}

type StoryFilter struct {
	storyTypes   map[string]bool // For O(1) lookups
	keywords     []string
	minimumScore int
	enabled      bool // true if any filter is configured
}

// NewStoryFilter creates a filter from config
func NewStoryFilter(cfg FilterConfig) *StoryFilter {
	filter := &StoryFilter{
		storyTypes:   make(map[string]bool),
		keywords:     cfg.Keywords,
		minimumScore: cfg.MinimumScore,
	}

	// Populate storyTypes map for O(1) lookups
	for _, t := range cfg.StoryTypes {
		filter.storyTypes[t] = true
	}

	// Filter is enabled if any filter constraint is specified
	filter.enabled = len(cfg.StoryTypes) > 0 || len(cfg.Keywords) > 0 || cfg.MinimumScore > 0

	return filter
}

// Matches returns true if a story passes all configured filters
func (f *StoryFilter) Matches(story *Story) bool {
	if !f.enabled {
		return true // No filters configured, match everything
	}

	// Check story type filter
	if len(f.storyTypes) > 0 && !f.storyTypes[story.Type] {
		return false
	}

	// Check minimum score filter
	if story.Score < f.minimumScore {
		return false
	}

	// Check keywords filter (match if ANY keyword is found in title)
	if len(f.keywords) > 0 {
		titleLower := strings.ToLower(story.Title)
		matched := false
		for _, keyword := range f.keywords {
			if strings.Contains(titleLower, strings.ToLower(keyword)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
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

func createKafkaReader(cfg KafkaConfig) (*kafka.Reader, error) {
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

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{cfg.Broker},
		Topic:       cfg.Topic,
		StartOffset: kafka.FirstOffset,
		Dialer:      dialer,
		// Note: Not using consumer groups so no offsets are tracked across restarts
		// Each instance always reads from the beginning
	})

	fmt.Printf("[DEBUG] Kafka reader configured for broker: %s, topic: %s\n",
		cfg.Broker, cfg.Topic)
	return reader, nil
}

func NewServer(cfg Config) (*Server, error) {
	reader, err := createKafkaReader(cfg.Kafka)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka reader: %w", err)
	}

	filter := NewStoryFilter(cfg.Filter)

	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		store:  &StoryStore{stories: make(map[int]*Story)},
		config: cfg,
		reader: reader,
		ctx:    ctx,
		cancel: cancel,
		filter: filter,
	}, nil
}

// consumeMessages reads messages from Kafka and adds them to the store
func (s *Server) consumeMessages() {
	fmt.Println("Starting Kafka message consumer...")
	for {
		select {
		case <-s.ctx.Done():
			fmt.Println("Consumer shutting down...")
			return
		default:
		}

		msg, err := s.reader.FetchMessage(s.ctx)
		if err != nil {
			if err == context.Canceled {
				return
			}
			fmt.Printf("[ERROR] Failed to fetch message: %v\n", err)
			continue
		}

		var story Story
		if err := json.Unmarshal(msg.Value, &story); err != nil {
			fmt.Printf("[ERROR] Failed to unmarshal story: %v\n", err)
			continue
		}

		// Apply filter before storing
		if !s.filter.Matches(&story) {
			fmt.Printf("[FILTERED] Story ID %d: %s (Type: %s, Score: %d)\n",
				story.ID, story.Title, story.Type, story.Score)
			continue
		}

		s.store.AddStory(&story)
		fmt.Printf("[STORED] Story ID %d: %s (Score: %d)\n", story.ID, story.Title, story.Score)
	}
}

func (s *StoryStore) AddStory(story *Story) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stories[story.ID] = story
}

func (s *StoryStore) GetAllStories() []*Story {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stories := make([]*Story, 0, len(s.stories))
	for _, story := range s.stories {
		stories = append(stories, story)
	}
	return stories
}

// handleGetStories handles GET /stories with optional filtering and sorting
func (s *Server) handleGetStories(w http.ResponseWriter, r *http.Request) {
	stories := s.store.GetAllStories()

	// Parse query parameters
	minScore := 0
	maxScore := int(^uint32(0) >> 1) // Max int
	var sinceTime, untilTime int64

	if ms := r.URL.Query().Get("minScore"); ms != "" {
		if v, err := strconv.Atoi(ms); err == nil {
			minScore = v
		}
	}
	if ms := r.URL.Query().Get("maxScore"); ms != "" {
		if v, err := strconv.Atoi(ms); err == nil {
			maxScore = v
		}
	}
	if st := r.URL.Query().Get("since"); st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			sinceTime = t.Unix()
		}
	}
	if ut := r.URL.Query().Get("until"); ut != "" {
		if t, err := time.Parse(time.RFC3339, ut); err == nil {
			untilTime = t.Unix()
		}
	}

	// Filter stories
	filtered := make([]*Story, 0, len(stories))
	for _, story := range stories {
		if story.Score < minScore || story.Score > maxScore {
			continue
		}
		if sinceTime > 0 && story.Time < sinceTime {
			continue
		}
		if untilTime > 0 && story.Time > untilTime {
			continue
		}
		filtered = append(filtered, story)
	}

	// Sort stories
	sortBy := r.URL.Query().Get("sort")
	switch sortBy {
	case "oldest":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Time < filtered[j].Time
		})
	case "popularity":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Score > filtered[j].Score
		})
	case "latest":
		fallthrough
	default:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Time > filtered[j].Time
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

func (s *Server) setupRoutes() {
	http.HandleFunc("/stories", s.handleGetStories)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
}

func (s *Server) start() {
	s.setupRoutes()

	// Log filter configuration
	if s.filter.enabled {
		fmt.Println("\n[CONFIG] Consumer-side filtering enabled:")
		if len(s.filter.storyTypes) > 0 {
			fmt.Printf("  Story types: %v\n", s.config.Filter.StoryTypes)
		}
		if len(s.filter.keywords) > 0 {
			fmt.Printf("  Keywords: %v\n", s.config.Filter.Keywords)
		}
		if s.filter.minimumScore > 0 {
			fmt.Printf("  Minimum score: %d\n", s.filter.minimumScore)
		}
		fmt.Println()
	} else {
		fmt.Println("\n[CONFIG] No filters configured - consuming all stories\n")
	}

	addr := fmt.Sprintf(":%d", s.config.API.Port)
	fmt.Printf("Starting API server on %s\n", addr)

	go s.consumeMessages()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	httpServer := &http.Server{
		Addr:         addr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[ERROR] Server error: %v\n", err)
		}
	}()

	<-sigChan
	fmt.Println("\n[SIGNAL] Shutting down server...")
	s.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		fmt.Printf("[ERROR] Server shutdown error: %v\n", err)
	}

	if err := s.reader.Close(); err != nil {
		fmt.Printf("[ERROR] Reader close error: %v\n", err)
	}

	fmt.Println("Server shutdown complete")
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	server, err := NewServer(*cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize server: %v\n", err)
		os.Exit(1)
	}

	server.start()
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type APIConfig struct {
	Name string `yaml:"name" json:"name"`
	URL  string `yaml:"url" json:"url"`
}

type Config struct {
	Port            int        `yaml:"port"`
	RefreshInterval int        `yaml:"refresh_interval"`
	APIs            []APIConfig `yaml:"apis"`
}

type ConfigResponse struct {
	APIs            []APIConfig `json:"apis"`
	RefreshInterval int        `json:"refreshInterval"`
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

	// Set defaults
	if cfg.Port == 0 {
		cfg.Port = 3000
	}
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 60
	}

	return &cfg, nil
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Get the directory of the frontend build
	frontendDir := filepath.Join(filepath.Dir(*configPath), "..", "frontend", "build")

	// API endpoint for config
	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ConfigResponse{
			APIs:            cfg.APIs,
			RefreshInterval: cfg.RefreshInterval,
		})
	})

	// Serve static files
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Default to index.html for root and missing files (SPA routing)
		path := filepath.Join(frontendDir, r.URL.Path)

		if _, err := os.Stat(path); os.IsNotExist(err) || r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(frontendDir, "index.html"))
		} else {
			http.ServeFile(w, r, path)
		}
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Starting web server on http://localhost:%d", cfg.Port)
	log.Fatal(http.ListenAndServe(addr, nil))
}

package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FeedConfig represents a single RSS/Atom feed configuration
type FeedConfig struct {
	ID                   string `yaml:"id"`
	Name                 string `yaml:"name"`
	URL                  string `yaml:"url"`
	Category             string `yaml:"category"`
	Enabled              bool   `yaml:"enabled"`
	RefreshIntervalMinutes *int  `yaml:"refresh_interval_minutes,omitempty"`
}

// UIConfig represents UI-related settings
type UIConfig struct {
	ItemsPerPage      int    `yaml:"items_per_page"`
	DefaultView       string `yaml:"default_view"`
	ShowFilteredCount bool   `yaml:"show_filtered_count"`
}

// Config represents the complete application configuration
type Config struct {
	Feeds     []FeedConfig `yaml:"feeds"`
	Blocklist []string     `yaml:"blocklist"`
	UI        UIConfig     `yaml:"ui"`
}

// DataDir returns the path to the CalmNews data directory
// Checks CALMNEWS_DATA_DIR environment variable first, then defaults to ~/.calmnews/
func DataDir() (string, error) {
	// Check for environment variable (useful for Docker)
	if dataDir := os.Getenv("CALMNEWS_DATA_DIR"); dataDir != "" {
		return dataDir, nil
	}
	
	// Default to home directory
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	return filepath.Join(usr.HomeDir, ".calmnews"), nil
}

// EnsureDataDir creates the data directory if it doesn't exist
func EnsureDataDir() error {
	dir, err := DataDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DefaultConfig returns a default configuration with example feeds
func DefaultConfig() *Config {
	refreshInterval := 10
	return &Config{
		Feeds: []FeedConfig{
			{
				ID:                   "hackernews",
				Name:                 "Hacker News",
				URL:                  "https://hnrss.org/frontpage",
				Category:             "tech",
				Enabled:              true,
				RefreshIntervalMinutes: &refreshInterval,
			},
			{
				ID:                   "lobsters",
				Name:                 "Lobsters",
				URL:                  "https://lobste.rs/rss",
				Category:             "tech",
				Enabled:              true,
				RefreshIntervalMinutes: &refreshInterval,
			},
		},
		Blocklist: []string{
			"donald trump",
			"trump",
		},
		UI: UIConfig{
			ItemsPerPage:      50,
			DefaultView:       "latest",
			ShowFilteredCount: true,
		},
	}
}


package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"calmnews/internal/config"
	"calmnews/internal/feeds"
	"calmnews/internal/storage"
	"calmnews/internal/web"
)

func main() {
	// Get data directory
	dataDir, err := config.DataDir()
	if err != nil {
		log.Fatalf("Failed to get data directory: %v", err)
	}

	// Ensure data directory exists
	if err := config.EnsureDataDir(); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Load or create config
	configPath := filepath.Join(dataDir, "config.yaml")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Config file not found, creating default config at %s", configPath)
			cfg = config.DefaultConfig()
			if err := config.SaveConfig(configPath, cfg); err != nil {
				log.Fatalf("Failed to save default config: %v", err)
			}
		} else {
			log.Fatalf("Failed to load config: %v", err)
		}
	}

	// Initialize database
	dbPath := filepath.Join(dataDir, "news.db")
	db, err := storage.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Printf("Database initialized at %s", dbPath)

	// Sync feeds from config to database
	for _, feedCfg := range cfg.Feeds {
		feed := &storage.Feed{
			ID:       feedCfg.ID,
			Name:     feedCfg.Name,
			URL:      feedCfg.URL,
			Category: feedCfg.Category,
			Enabled:  feedCfg.Enabled,
		}
		if err := storage.UpsertFeed(db, feed); err != nil {
			log.Printf("Warning: Failed to sync feed %s: %v", feedCfg.ID, err)
		}
	}

	log.Printf("Synced %d feeds to database", len(cfg.Feeds))

	// Start background scheduler
	refreshInterval := 10 // default
	if len(cfg.Feeds) > 0 && cfg.Feeds[0].RefreshIntervalMinutes != nil {
		refreshInterval = *cfg.Feeds[0].RefreshIntervalMinutes
	}
	feeds.StartScheduler(db, cfg, refreshInterval)
	log.Printf("Started feed scheduler (refresh interval: %d minutes)", refreshInterval)

	// Create web server
	server := web.NewServer(db, cfg, configPath)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.HandleIndex)
	mux.HandleFunc("/settings", server.HandleSettings)
	mux.HandleFunc("/settings/blocklist", server.HandleUpdateBlocklist)
	mux.HandleFunc("/settings/feeds", server.HandleUpdateFeeds)
	mux.HandleFunc("/article/read", server.HandleMarkArticleRead)
	mux.HandleFunc("/article/save", server.HandleToggleArticleSaved)
	mux.HandleFunc("/static/", web.HandleStatic)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting CalmNews server on http://127.0.0.1:8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}


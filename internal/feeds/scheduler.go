package feeds

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"calmnews/internal/config"
	"calmnews/internal/storage"
)

// StartScheduler starts a background goroutine that periodically fetches and updates feeds
func StartScheduler(db *sql.DB, cfg *config.Config, refreshIntervalMinutes int) {
	go func() {
		ticker := time.NewTicker(time.Duration(refreshIntervalMinutes) * time.Minute)
		defer ticker.Stop()

		// Do an initial fetch immediately
		fetchAllFeeds(db, cfg)

		// Do an initial cleanup
		cleanupExpiredArticles(db)

		for range ticker.C {
			fetchAllFeeds(db, cfg)
			// Cleanup expired articles after each fetch cycle
			cleanupExpiredArticles(db)
		}
	}()
}

// cleanupExpiredArticles removes articles older than 72 hours (except saved ones)
func cleanupExpiredArticles(db *sql.DB) {
	deleted, err := storage.DeleteExpiredArticles(db, 72)
	if err != nil {
		log.Printf("Error cleaning up expired articles: %v", err)
		return
	}
	if deleted > 0 {
		log.Printf("Cleaned up %d expired articles", deleted)
	}
}

func fetchAllFeeds(db *sql.DB, cfg *config.Config) {
	feeds, err := storage.ListFeeds(db, true) // Only enabled feeds
	if err != nil {
		log.Printf("Error listing feeds: %v", err)
		return
	}

	now := time.Now()
	defaultInterval := 10 * time.Minute

	for _, feed := range feeds {
		// Check if enough time has passed since last fetch
		if feed.LastFetchedAt != nil {
			// Find refresh interval for this feed
			interval := defaultInterval
			for _, feedCfg := range cfg.Feeds {
				if feedCfg.ID == feed.ID && feedCfg.RefreshIntervalMinutes != nil {
					interval = time.Duration(*feedCfg.RefreshIntervalMinutes) * time.Minute
					break
				}
			}

			timeSinceLastFetch := now.Sub(*feed.LastFetchedAt)
			if timeSinceLastFetch < interval {
				continue // Skip this feed, not enough time has passed
			}
		}

		// Fetch the feed
		if err := fetchAndStoreFeed(db, feed); err != nil {
			log.Printf("Error fetching feed %s (%s): %v", feed.Name, feed.URL, err)
			continue
		}

		log.Printf("Successfully fetched feed: %s", feed.Name)
	}
}

func fetchAndStoreFeed(db *sql.DB, feed *storage.Feed) error {
	// Fetch feed data
	data, err := FetchFeed(feed.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// Parse feed
	articles, err := ParseFeed(data, feed.URL, feed.ID, feed.Name)
	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	// Filter out duplicate articles by title
	var uniqueArticles []*storage.Article
	for _, article := range articles {
		exists, err := storage.ArticleExistsByTitle(db, article.Title)
		if err != nil {
			log.Printf("Error checking for duplicate article %s: %v", article.Title, err)
			// Continue with other articles, but don't skip this one
		} else if exists {
			log.Printf("Skipping duplicate article: %s", article.Title)
			continue
		}
		uniqueArticles = append(uniqueArticles, article)
	}

	// Store unique articles
	for _, article := range uniqueArticles {
		if err := storage.UpsertArticle(db, article); err != nil {
			log.Printf("Error upserting article %s: %v", article.ID, err)
			// Continue with other articles
		}
	}

	// Update last_fetched_at
	now := time.Now()
	if err := storage.UpdateFeedLastFetched(db, feed.ID, now); err != nil {
		return fmt.Errorf("failed to update last_fetched_at: %w", err)
	}

	return nil
}

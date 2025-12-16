package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// Feed represents a feed in the database
type Feed struct {
	ID            string
	Name          string
	URL           string
	Category      string
	Enabled       bool
	LastFetchedAt *time.Time
}

// Article represents an article in the database
type Article struct {
	ID          string
	FeedID      string
	Title       string
	URL         string
	Summary     string
	Content     string
	PublishedAt time.Time
	FetchedAt   time.Time
	SourceName  string
	Categories   string
	IsRead       bool
}

// hashArticleID generates a unique ID for an article based on feed URL and entry GUID/link
func hashArticleID(feedURL, entryGUID string) string {
	data := feedURL + "|" + entryGUID
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// UpsertFeed inserts or updates a feed in the database
func UpsertFeed(db *sql.DB, feed *Feed) error {
	query := `
	INSERT INTO feeds (id, name, url, category, enabled, last_fetched_at)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name = excluded.name,
		url = excluded.url,
		category = excluded.category,
		enabled = excluded.enabled,
		last_fetched_at = excluded.last_fetched_at;`

	_, err := db.Exec(query, feed.ID, feed.Name, feed.URL, feed.Category, feed.Enabled, feed.LastFetchedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert feed: %w", err)
	}
	return nil
}

// ListFeeds returns all feeds, optionally filtering by enabled status
func ListFeeds(db *sql.DB, enabledOnly bool) ([]*Feed, error) {
	var query string
	var args []interface{}

	if enabledOnly {
		query = `SELECT id, name, url, category, enabled, last_fetched_at FROM feeds WHERE enabled = 1 ORDER BY name;`
	} else {
		query = `SELECT id, name, url, category, enabled, last_fetched_at FROM feeds ORDER BY name;`
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query feeds: %w", err)
	}
	defer rows.Close()

	var feeds []*Feed
	for rows.Next() {
		var f Feed
		var lastFetched sql.NullTime
		err := rows.Scan(&f.ID, &f.Name, &f.URL, &f.Category, &f.Enabled, &lastFetched)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feed: %w", err)
		}
		if lastFetched.Valid {
			f.LastFetchedAt = &lastFetched.Time
		}
		feeds = append(feeds, &f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feeds: %w", err)
	}

	return feeds, nil
}

// GetFeedByID returns a feed by its ID
func GetFeedByID(db *sql.DB, id string) (*Feed, error) {
	query := `SELECT id, name, url, category, enabled, last_fetched_at FROM feeds WHERE id = ?;`

	var f Feed
	var lastFetched sql.NullTime
	err := db.QueryRow(query, id).Scan(&f.ID, &f.Name, &f.URL, &f.Category, &f.Enabled, &lastFetched)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("feed not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}
	if lastFetched.Valid {
		f.LastFetchedAt = &lastFetched.Time
	}
	return &f, nil
}

// UpdateFeedLastFetched updates the last_fetched_at timestamp for a feed
func UpdateFeedLastFetched(db *sql.DB, feedID string, t time.Time) error {
	query := `UPDATE feeds SET last_fetched_at = ? WHERE id = ?;`
	_, err := db.Exec(query, t, feedID)
	if err != nil {
		return fmt.Errorf("failed to update feed last_fetched_at: %w", err)
	}
	return nil
}

// UpsertArticle inserts or updates an article in the database
func UpsertArticle(db *sql.DB, article *Article) error {
	query := `
	INSERT INTO articles (id, feed_id, title, url, summary, content, published_at, fetched_at, source_name, categories, is_read)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		title = excluded.title,
		url = excluded.url,
		summary = excluded.summary,
		content = excluded.content,
		published_at = excluded.published_at,
		fetched_at = excluded.fetched_at,
		source_name = excluded.source_name,
		categories = excluded.categories;`

	_, err := db.Exec(query,
		article.ID, article.FeedID, article.Title, article.URL, article.Summary,
		article.Content, article.PublishedAt, article.FetchedAt, article.SourceName,
		article.Categories, article.IsRead)
	if err != nil {
		return fmt.Errorf("failed to upsert article: %w", err)
	}
	return nil
}

// ListArticlesByView returns articles based on view type and optional feed filter
func ListArticlesByView(db *sql.DB, view string, feedID string, limit int) ([]*Article, error) {
	var query string
	var args []interface{}

	now := time.Now()
	var timeWindow time.Time

	switch view {
	case "today":
		// Start of today
		timeWindow = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		query = `SELECT id, feed_id, title, url, summary, content, published_at, fetched_at, source_name, categories, is_read
			FROM articles
			WHERE published_at >= ?`
	case "week":
		// Last 7 days
		timeWindow = now.AddDate(0, 0, -7)
		query = `SELECT id, feed_id, title, url, summary, content, published_at, fetched_at, source_name, categories, is_read
			FROM articles
			WHERE published_at >= ?`
	case "latest":
		fallthrough
	default:
		// Last 3 days or just limit
		timeWindow = now.AddDate(0, 0, -3)
		query = `SELECT id, feed_id, title, url, summary, content, published_at, fetched_at, source_name, categories, is_read
			FROM articles
			WHERE published_at >= ?`
	}

	args = append(args, timeWindow)

	if feedID != "" && feedID != "all" {
		query += ` AND feed_id = ?`
		args = append(args, feedID)
	}

	query += ` ORDER BY published_at DESC LIMIT ?;`
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}
	defer rows.Close()

	var articles []*Article
	for rows.Next() {
		var a Article
		err := rows.Scan(&a.ID, &a.FeedID, &a.Title, &a.URL, &a.Summary, &a.Content,
			&a.PublishedAt, &a.FetchedAt, &a.SourceName, &a.Categories, &a.IsRead)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating articles: %w", err)
	}

	return articles, nil
}

// GenerateArticleID generates an article ID from feed URL and entry GUID/link
func GenerateArticleID(feedURL, entryGUID string) string {
	return hashArticleID(feedURL, entryGUID)
}


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
	IsSaved      bool
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
	INSERT INTO articles (id, feed_id, title, url, summary, content, published_at, fetched_at, source_name, categories, is_read, is_saved)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		title = excluded.title,
		url = excluded.url,
		summary = excluded.summary,
		content = excluded.content,
		published_at = excluded.published_at,
		fetched_at = COALESCE(articles.fetched_at, excluded.fetched_at),
		source_name = excluded.source_name,
		categories = excluded.categories,
		is_read = COALESCE(excluded.is_read, articles.is_read),
		is_saved = COALESCE(excluded.is_saved, articles.is_saved);`

	isRead := 0
	if article.IsRead {
		isRead = 1
	}
	isSaved := 0
	if article.IsSaved {
		isSaved = 1
	}

	_, err := db.Exec(query,
		article.ID, article.FeedID, article.Title, article.URL, article.Summary,
		article.Content, article.PublishedAt, article.FetchedAt, article.SourceName,
		article.Categories, isRead, isSaved)
	if err != nil {
		return fmt.Errorf("failed to upsert article: %w", err)
	}
	return nil
}

// ListArticlesByView returns articles based on view type and optional feed filter
// readFilter can be "all", "unread", or "read"
func ListArticlesByView(db *sql.DB, view string, feedID string, readFilter string, limit int) ([]*Article, error) {
	var query string
	var args []interface{}

	now := time.Now()
	var timeWindow time.Time

	switch view {
	case "saved":
		// Saved articles view - no time window, just saved articles
		query = `SELECT id, feed_id, title, url, summary, content, published_at, fetched_at, source_name, categories, is_read, is_saved
			FROM articles
			WHERE is_saved = 1`
		// No time window for saved articles
	case "today":
		// Start of today
		timeWindow = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		query = `SELECT id, feed_id, title, url, summary, content, published_at, fetched_at, source_name, categories, is_read, is_saved
			FROM articles
			WHERE published_at >= ?`
	case "week":
		// Last 7 days
		timeWindow = now.AddDate(0, 0, -7)
		query = `SELECT id, feed_id, title, url, summary, content, published_at, fetched_at, source_name, categories, is_read, is_saved
			FROM articles
			WHERE published_at >= ?`
	case "latest":
		fallthrough
	default:
		// Last 3 days or just limit
		timeWindow = now.AddDate(0, 0, -3)
		query = `SELECT id, feed_id, title, url, summary, content, published_at, fetched_at, source_name, categories, is_read, is_saved
			FROM articles
			WHERE published_at >= ?`
	}

	if view != "saved" {
		args = append(args, timeWindow)
	}

	if feedID != "" && feedID != "all" {
		query += ` AND feed_id = ?`
		args = append(args, feedID)
	}

	// Add read filter
	if readFilter == "unread" {
		query += ` AND is_read = 0`
	} else if readFilter == "read" {
		query += ` AND is_read = 1`
	}

	// Sort: unread first (by published_at DESC), then read (by published_at DESC)
	query += ` ORDER BY is_read ASC, published_at DESC LIMIT ?;`
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}
	defer rows.Close()

	var articles []*Article
	for rows.Next() {
		var a Article
		var isRead, isSaved int
		err := rows.Scan(&a.ID, &a.FeedID, &a.Title, &a.URL, &a.Summary, &a.Content,
			&a.PublishedAt, &a.FetchedAt, &a.SourceName, &a.Categories, &isRead, &isSaved)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		a.IsRead = isRead == 1
		a.IsSaved = isSaved == 1
		articles = append(articles, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating articles: %w", err)
	}

	return articles, nil
}

// MarkArticleAsRead marks an article as read
func MarkArticleAsRead(db *sql.DB, articleID string) error {
	query := `UPDATE articles SET is_read = 1 WHERE id = ?;`
	_, err := db.Exec(query, articleID)
	if err != nil {
		return fmt.Errorf("failed to mark article as read: %w", err)
	}
	return nil
}

// MarkArticleAsUnread marks an article as unread
func MarkArticleAsUnread(db *sql.DB, articleID string) error {
	query := `UPDATE articles SET is_read = 0 WHERE id = ?;`
	_, err := db.Exec(query, articleID)
	if err != nil {
		return fmt.Errorf("failed to mark article as unread: %w", err)
	}
	return nil
}

// ToggleArticleSaved toggles the saved status of an article
func ToggleArticleSaved(db *sql.DB, articleID string) error {
	query := `UPDATE articles SET is_saved = NOT is_saved WHERE id = ?;`
	_, err := db.Exec(query, articleID)
	if err != nil {
		return fmt.Errorf("failed to toggle article saved status: %w", err)
	}
	return nil
}

// DeleteExpiredArticles deletes articles older than expirationHours from fetched_at, except saved ones
func DeleteExpiredArticles(db *sql.DB, expirationHours int) (int64, error) {
	query := `DELETE FROM articles 
		WHERE is_saved = 0 
		AND datetime(fetched_at, '+' || ? || ' hours') < datetime('now');`
	
	result, err := db.Exec(query, expirationHours)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired articles: %w", err)
	}
	
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return deleted, nil
}

// GenerateArticleID generates an article ID from feed URL and entry GUID/link
func GenerateArticleID(feedURL, entryGUID string) string {
	return hashArticleID(feedURL, entryGUID)
}

// ArticleExistsByTitle checks if an article with the given title already exists in the database
func ArticleExistsByTitle(db *sql.DB, title string) (bool, error) {
	query := `SELECT COUNT(*) FROM articles WHERE title = ?;`
	var count int
	err := db.QueryRow(query, title).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check article by title: %w", err)
	}
	return count > 0, nil
}


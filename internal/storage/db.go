package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// InitDB initializes a SQLite database connection
func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := RunMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// RunMigrations creates the necessary tables if they don't exist
func RunMigrations(db *sql.DB) error {
	// Create feeds table
	feedsTable := `
	CREATE TABLE IF NOT EXISTS feeds (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		category TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		last_fetched_at DATETIME
	);`

	if _, err := db.Exec(feedsTable); err != nil {
		return fmt.Errorf("failed to create feeds table: %w", err)
	}

	// Create articles table
	articlesTable := `
	CREATE TABLE IF NOT EXISTS articles (
		id TEXT PRIMARY KEY,
		feed_id TEXT NOT NULL,
		title TEXT NOT NULL,
		url TEXT NOT NULL,
		summary TEXT,
		content TEXT,
		published_at DATETIME NOT NULL,
		fetched_at DATETIME NOT NULL,
		source_name TEXT NOT NULL,
		categories TEXT,
		is_read INTEGER DEFAULT 0,
		is_saved INTEGER DEFAULT 0,
		FOREIGN KEY (feed_id) REFERENCES feeds(id)
	);`

	if _, err := db.Exec(articlesTable); err != nil {
		return fmt.Errorf("failed to create articles table: %w", err)
	}

	// Add is_saved column if it doesn't exist (for existing databases)
	_, err := db.Exec(`ALTER TABLE articles ADD COLUMN is_saved INTEGER DEFAULT 0;`)
	if err != nil {
		// Column might already exist, ignore error
	}

	// Create index on published_at for faster queries
	indexQuery := `
	CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles(published_at DESC);`

	if _, err := db.Exec(indexQuery); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Create index on feed_id
	feedIndexQuery := `
	CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id);`

	if _, err := db.Exec(feedIndexQuery); err != nil {
		return fmt.Errorf("failed to create feed_id index: %w", err)
	}

	// Create index on title for duplicate detection
	titleIndexQuery := `
	CREATE INDEX IF NOT EXISTS idx_articles_title ON articles(title);`

	if _, err := db.Exec(titleIndexQuery); err != nil {
		return fmt.Errorf("failed to create title index: %w", err)
	}

	return nil
}


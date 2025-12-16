package feeds

import (
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
	"calmnews/internal/storage"
)

// ParseFeed parses RSS/Atom feed data and returns normalized articles
func ParseFeed(data []byte, feedURL string, feedID string, sourceName string) ([]*storage.Article, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	var articles []*storage.Article
	now := time.Now()

	for _, item := range feed.Items {
		// Use GUID if available, otherwise use link
		entryGUID := item.GUID
		if entryGUID == "" {
			entryGUID = item.Link
		}

		articleID := storage.GenerateArticleID(feedURL, entryGUID)

		// Parse published date
		var publishedAt time.Time
		if item.PublishedParsed != nil {
			publishedAt = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			publishedAt = *item.UpdatedParsed
		} else {
			publishedAt = now
		}

		// Extract summary/description
		summary := ""
		if item.Description != "" {
			summary = item.Description
		} else if item.Content != "" {
			summary = item.Content
		}

		// Extract content
		content := ""
		if item.Content != "" {
			content = item.Content
		} else if item.Description != "" {
			content = item.Description
		}

		article := &storage.Article{
			ID:          articleID,
			FeedID:      feedID,
			Title:       item.Title,
			URL:         item.Link,
			Summary:     summary,
			Content:     content,
			PublishedAt: publishedAt,
			FetchedAt:   now,
			SourceName:  sourceName,
			Categories:  "",
			IsRead:      false,
		}

		articles = append(articles, article)
	}

	return articles, nil
}


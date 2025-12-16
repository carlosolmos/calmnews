package filter

import (
	"strings"

	"calmnews/internal/storage"
)

// ShouldFilter returns true if the article should be filtered out based on the blocklist
func ShouldFilter(article *storage.Article, blocklist []string) bool {
	if len(blocklist) == 0 {
		return false
	}

	// Build a lowercase text blob from title and summary
	textBlob := strings.ToLower(article.Title + " " + article.Summary)

	// Check each phrase in the blocklist
	for _, phrase := range blocklist {
		lowerPhrase := strings.ToLower(strings.TrimSpace(phrase))
		if lowerPhrase == "" {
			continue
		}
		if strings.Contains(textBlob, lowerPhrase) {
			return true
		}
	}

	return false
}

// FilterArticles filters a list of articles based on the blocklist
func FilterArticles(articles []*storage.Article, blocklist []string) ([]*storage.Article, int) {
	var filtered []*storage.Article
	filteredCount := 0

	for _, article := range articles {
		if ShouldFilter(article, blocklist) {
			filteredCount++
			continue
		}
		filtered = append(filtered, article)
	}

	return filtered, filteredCount
}


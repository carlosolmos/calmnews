package feeds

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	maxResponseSize = 10 * 1024 * 1024 // 10MB
	httpTimeout     = 30 * time.Second
)

// FetchFeed fetches an RSS/Atom feed from the given URL
func FetchFeed(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: httpTimeout,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "CalmNews/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}


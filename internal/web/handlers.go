package web

import (
	"database/sql"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"calmnews/internal/config"
	"calmnews/internal/filter"
	"calmnews/internal/storage"
)

// Server holds the dependencies for HTTP handlers
type Server struct {
	db         *sql.DB
	config     *config.Config
	configPath string
}

// NewServer creates a new web server instance
func NewServer(db *sql.DB, cfg *config.Config, configPath string) *Server {
	return &Server{
		db:         db,
		config:     cfg,
		configPath: configPath,
	}
}

// HandleIndex handles the main front page
func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	view := r.URL.Query().Get("view")
	if view == "" {
		view = s.config.UI.DefaultView
	}
	if view != "latest" && view != "today" && view != "week" && view != "saved" {
		view = "latest"
	}

	feedID := r.URL.Query().Get("feed")
	if feedID == "" {
		feedID = "all"
	}

	readFilter := r.URL.Query().Get("read")
	if readFilter == "" {
		readFilter = "all"
	}
	if readFilter != "all" && readFilter != "read" && readFilter != "unread" {
		readFilter = "all"
	}

	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Query articles (get a superset, we'll filter and paginate)
	limit := 300 // Get more than we need for filtering
	articles, err := storage.ListArticlesByView(s.db, view, feedID, readFilter, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error querying articles: %v", err), http.StatusInternalServerError)
		return
	}

	// Apply blocklist filter
	filteredArticles, filteredCount := filter.FilterArticles(articles, s.config.Blocklist)

	// Paginate
	itemsPerPage := s.config.UI.ItemsPerPage
	start := (page - 1) * itemsPerPage
	end := start + itemsPerPage
	if start > len(filteredArticles) {
		start = len(filteredArticles)
	}
	if end > len(filteredArticles) {
		end = len(filteredArticles)
	}

	var pageArticles []*storage.Article
	if start < len(filteredArticles) {
		pageArticles = filteredArticles[start:end]
	}

	// Get all feeds for the filter dropdown
	feeds, _ := storage.ListFeeds(s.db, false)

	// Prepare template data
	data := map[string]interface{}{
		"Articles":          pageArticles,
		"View":              view,
		"FeedID":            feedID,
		"ReadFilter":        readFilter,
		"Feeds":             feeds,
		"Page":              page,
		"NextPage":          page + 1,
		"PrevPage":          page - 1,
		"HasNextPage":       end < len(filteredArticles),
		"HasPrevPage":       page > 1,
		"FilteredCount":     filteredCount,
		"ShowFilteredCount": s.config.UI.ShowFilteredCount,
	}

	if err := s.RenderTemplate(w, "index.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleSettings handles the settings page
func (s *Server) HandleSettings(w http.ResponseWriter, r *http.Request) {
	feeds, err := storage.ListFeeds(s.db, false)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error querying feeds: %v", err), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Blocklist": s.config.Blocklist,
		"Feeds":     feeds,
	}

	if err := s.RenderTemplate(w, "settings.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleUpdateBlocklist handles POST requests to update the blocklist
func (s *Server) HandleUpdateBlocklist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	action := r.FormValue("action")
	phrase := strings.TrimSpace(r.FormValue("phrase"))

	if action == "add" && phrase != "" {
		// Check if already exists
		exists := false
		lowerPhrase := strings.ToLower(phrase)
		for _, p := range s.config.Blocklist {
			if strings.ToLower(p) == lowerPhrase {
				exists = true
				break
			}
		}
		if !exists {
			s.config.Blocklist = append(s.config.Blocklist, phrase)
		}
	} else if action == "remove" && phrase != "" {
		lowerPhrase := strings.ToLower(phrase)
		var newList []string
		for _, p := range s.config.Blocklist {
			if strings.ToLower(p) != lowerPhrase {
				newList = append(newList, p)
			}
		}
		s.config.Blocklist = newList
	}

	// Save config
	if err := config.SaveConfig(s.configPath, s.config); err != nil {
		log.Printf("Error saving config: %v", err)
		http.Error(w, "Error saving config", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// HandleMarkArticleRead handles POST requests to mark an article as read
func (s *Server) HandleMarkArticleRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	articleID := r.FormValue("id")
	if articleID == "" {
		http.Error(w, "Article ID required", http.StatusBadRequest)
		return
	}

	if err := storage.MarkArticleAsRead(s.db, articleID); err != nil {
		log.Printf("Error marking article as read: %v", err)
		http.Error(w, "Error marking article as read", http.StatusInternalServerError)
		return
	}

	// Return JSON response for AJAX calls
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok"}`))
}

// HandleToggleArticleSaved handles POST requests to toggle an article's saved status
func (s *Server) HandleToggleArticleSaved(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	articleID := r.FormValue("id")
	if articleID == "" {
		http.Error(w, "Article ID required", http.StatusBadRequest)
		return
	}

	if err := storage.ToggleArticleSaved(s.db, articleID); err != nil {
		log.Printf("Error toggling article saved status: %v", err)
		http.Error(w, "Error toggling article saved status", http.StatusInternalServerError)
		return
	}

	// Return JSON response for AJAX calls
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok"}`))
}

// HandleUpdateFeeds handles POST requests to update feeds
func (s *Server) HandleUpdateFeeds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	action := r.FormValue("action")

	if action == "toggle" {
		feedID := r.FormValue("feed_id")
		if feedID != "" {
			feed, err := storage.GetFeedByID(s.db, feedID)
			if err == nil {
				feed.Enabled = !feed.Enabled
				if err := storage.UpsertFeed(s.db, feed); err != nil {
					log.Printf("Error updating feed: %v", err)
				} else {
					// Update config
					for i := range s.config.Feeds {
						if s.config.Feeds[i].ID == feedID {
							s.config.Feeds[i].Enabled = feed.Enabled
							break
						}
					}
					config.SaveConfig(s.configPath, s.config)
				}
			}
		}
	} else if action == "add" {
		feedID := strings.TrimSpace(r.FormValue("id"))
		name := strings.TrimSpace(r.FormValue("name"))
		url := strings.TrimSpace(r.FormValue("url"))
		category := strings.TrimSpace(r.FormValue("category"))

		if feedID != "" && name != "" && url != "" && category != "" {
			feed := &storage.Feed{
				ID:       feedID,
				Name:     name,
				URL:      url,
				Category: category,
				Enabled:  true,
			}
			if err := storage.UpsertFeed(s.db, feed); err != nil {
				log.Printf("Error adding feed: %v", err)
			} else {
				// Add to config
				refreshInterval := 10
				s.config.Feeds = append(s.config.Feeds, config.FeedConfig{
					ID:                     feedID,
					Name:                   name,
					URL:                    url,
					Category:               category,
					Enabled:                true,
					RefreshIntervalMinutes: &refreshInterval,
				})
				config.SaveConfig(s.configPath, s.config)
			}
		}
	}

	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// FormatTimeAgo formats a time as "X hours ago" or similar
func FormatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("Jan 2, 2006")
	}
}

// RenderTemplate renders an HTML template
func (s *Server) RenderTemplate(w http.ResponseWriter, name string, data interface{}) error {
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"timeAgo": FormatTimeAgo,
	}).ParseFS(templatesFS, "templates/"+name)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	return tmpl.Execute(w, data)
}

// HandleStatic serves static files (CSS, etc.)
func HandleStatic(w http.ResponseWriter, r *http.Request) {
	// Create a sub filesystem for static files
	staticFS, err := fs.Sub(templatesFS, "static")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Strip the /static/ prefix and serve the file
	http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))).ServeHTTP(w, r)
}

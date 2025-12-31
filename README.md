# CalmNews

A self-contained Go application that fetches articles from RSS/Atom feeds, stores them locally in SQLite, filters them with a blocklist, and serves an HN-style web UI on localhost.

## Features

- Fetches articles from multiple RSS/Atom feeds
- Stores articles locally in SQLite database
- Filters articles based on a configurable blocklist
- Articles expire and are removed after 72 hours, except save ones
- Saved articles remain in the database until manually discarded
- Clean, HN-inspired web interface
- Settings page to manage feeds and blocklist
- Background scheduler for automatic feed updates
- Self-contained binary (no external dependencies at runtime)

## Building

To build the CalmNews binary:

```bash
go build -o bin/calmnews ./cmd/calmnews
```

This will create a `calmnews` binary in the bin directory.

## Running

Simply run the binary:

```bash
./calmnews
```

The application will:
- Create a data directory at `~/.calmnews/` if it doesn't exist
- Generate a default `config.yaml` if one doesn't exist
- Initialize the SQLite database at `~/.calmnews/news.db`
- Start the web server on `http://127.0.0.1:8080`

Open your browser and navigate to `http://localhost:8080` to view the front page.

## Configuration

### Config File Location

The configuration file is located at `~/.calmnews/config.yaml`.

### Default Configuration

On first run, CalmNews creates a default configuration with:
- Example feeds (Hacker News, Lobsters)
- Sample blocklist entries
- Default UI settings

### Config Structure

```yaml
feeds:
  - id: "hackernews"
    name: "Hacker News"
    url: "https://hnrss.org/frontpage"
    category: "tech"
    enabled: true
    refresh_interval_minutes: 10

blocklist:
  - "he who shall not be named"
  - "voldemort"

ui:
  items_per_page: 50
  default_view: "latest"
  show_filtered_count: true
```

### Adding Feeds

You can add feeds in two ways:

1. **Via the Web UI**: Go to Settings → Feeds → Add New Feed
2. **Via config file**: Edit `~/.calmnews/config.yaml` and add a new feed entry, then restart the application

### Managing Blocklist

You can manage the blocklist in two ways:

1. **Via the Web UI**: Go to Settings → Blocklist → Add/Remove phrases
2. **Via config file**: Edit `~/.calmnews/config.yaml` and modify the `blocklist` section, then restart the application

## Data Storage

### Database Location

The SQLite database is stored at `~/.calmnews/news.db`.

### Database Schema

- **feeds**: Stores feed configuration and metadata
- **articles**: Stores all fetched articles with metadata

Articles are deduplicated based on a hash of the feed URL and entry GUID/link.

## Usage

### Front Page Views

- **Latest**: Shows articles from the last 3 days (or latest 300 articles)
- **Today**: Shows articles published today
- **This Week**: Shows articles from the last 7 days

### Feed Filtering

Use the dropdown on the front page to filter articles by specific feed or view all feeds.

### Pagination

Navigate through pages using the Previous/Next links at the bottom of the article list.

### Filtered Articles

If `show_filtered_count` is enabled in the config, you'll see a notice at the top showing how many articles were filtered out by the blocklist.

## Stopping the Application

Press `Ctrl+C` to gracefully shutdown the server. The application will:
- Stop accepting new requests
- Complete any in-flight requests
- Close the database connection
- Exit cleanly

## Troubleshooting

### Port Already in Use

If port 8080 is already in use, you'll need to either:
- Stop the other application using port 8080
- Modify the code to use a different port (edit `cmd/calmnews/main.go`)

### Feed Fetching Errors

If a feed fails to fetch, check:
- The feed URL is correct and accessible
- Your internet connection
- The feed format is valid RSS/Atom

Errors are logged to stdout but don't stop the application.

### Database Issues

If you encounter database issues:
- Check that `~/.calmnews/` directory is writable
- Delete `~/.calmnews/news.db` to start fresh (you'll lose all stored articles)

## Development

### Project Structure

```
calmnews/
├── cmd/calmnews/          # Main entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── storage/           # Database operations
│   ├── feeds/             # Feed fetching and parsing
│   ├── filter/            # Blocklist filtering
│   └── web/               # HTTP handlers and templates
├── go.mod
└── README.md
```

### Dependencies

- `github.com/mmcdole/gofeed` - RSS/Atom parsing
- `github.com/ncruces/go-sqlite3` - SQLite driver (pure Go, no CGO)
- `gopkg.in/yaml.v3` - YAML configuration


### Install Docker on Ubuntu Linux 24
#### Install Docker and Docker Compose on Ubuntu 24.04

To install Docker and Docker Compose on Ubuntu 24.04, follow these steps:

#### 1. Uninstall Old Versions (Optional)
```bash
sudo apt-get remove docker docker-engine docker.io containerd runc
```

#### 2. Update the apt Package Index
```bash
sudo apt-get update
```

#### 3. Install Required Packages
```bash
sudo apt-get install \
    ca-certificates \
    curl \
    gnupg \
    lsb-release
```

#### 4. Add Docker’s Official GPG Key
```bash
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg
```

#### 5. Set up the Docker Repository
```bash
echo \
  "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  "$(lsb_release -cs)" stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
```

#### 6. Update apt and Install Docker Engine
```bash
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

#### 7. Test Your Docker Installation
```bash
sudo docker run hello-world
```

#### 8. (Optional) Manage Docker as a Non-root User
```bash
sudo usermod -aG docker $USER
newgrp docker
```

#### 9. Verify Docker Compose Plugin
```bash
docker compose version
```

You can now use both `docker` and `docker compose` directly from the command line.

**Reference:**  
- [Docker Engine Install on Ubuntu](https://docs.docker.com/engine/install/ubuntu/)
- [Post-installation steps for Linux](https://docs.docker.com/engine/install/linux-postinstall/)




## License

This is a personal project. Use as you wish.


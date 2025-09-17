# linkding-to-opml

A bidirectional command-line tool for working with [Linkding](https://github.com/sissbruecker/linkding) bookmarks and OPML feeds:

- **Export**: Fetch bookmarks from Linkding, discover RSS/Atom feeds, and export as OPML
- **Import**: Convert OPML feeds to HTML bookmarks for importing into Linkding

## Features

### Export (Linkding → OPML)
- 🔗 Fetches bookmarks from Linkding API with optional tag filtering
- 📡 Automatically discovers RSS/Atom feeds using standard autodiscovery methods
- 📄 Generates OPML 2.0 compatible files for feed readers
- 💾 Intelligent caching system to avoid repeated network requests

### Import (OPML → Linkding)
- 📥 Converts OPML feeds to HTML bookmark files
- 🔍 Discovers website URLs from RSS/Atom feeds
- 🏷️ Supports adding tags to all imported bookmarks
- 📋 Generates Netscape HTML format compatible with Linkding import

### Shared Features
- ⚡ Concurrent processing for fast operation (configurable worker pool)
- 🛡️ Comprehensive error handling and logging
- ⚙️ Flexible configuration via YAML files or command-line flags

## Installation

### From Source

```bash
git clone <repository-url>
cd linkding-to-opml
go build -o linkding-to-opml
```

## Quick Start

### Export (Linkding → OPML)

1. **Basic usage with command-line flags:**
   ```bash
   ./linkding-to-opml export \
     --linkding-url https://your-linkding-instance.com \
     --linkding-token your-api-token
   ```

2. **Using a configuration file:**
   ```bash
   # Copy the example configuration
   cp linkding-to-opml.yaml.example linkding-to-opml.yaml

   # Edit with your settings
   nano linkding-to-opml.yaml

   # Run the export
   ./linkding-to-opml export
   ```

### Import (OPML → Linkding)

1. **Convert OPML to HTML bookmarks:**
   ```bash
   ./linkding-to-opml tohtml feeds.opml bookmarks.html
   ```

2. **Add tags to imported bookmarks:**
   ```bash
   ./linkding-to-opml tohtml --tags "imported,rss" feeds.opml bookmarks.html
   ```

3. **Import the HTML file into Linkding:**
   - Open your Linkding web interface
   - Go to Settings → Import
   - Upload the generated `bookmarks.html` file
   - Linkding will import the bookmarks with deduplication

## Configuration

### Configuration File

Create a `linkding-to-opml.yaml` file in your current directory:

```yaml
# Required: Linkding API settings
linkding:
  token: "your-api-token-here"
  url: "https://your-linkding-instance.com"
  timeout: "30s"

# Optional: Cache settings
cache:
  file_path: "./linkding-to-opml.gob"
  max_age: 720  # hours (30 days)

# Optional: HTTP client settings
http:
  timeout: "30s"
  user_agent: "Mozilla/5.0 (compatible; linkding-to-opml/1.0)"
  max_redirects: 3

# Optional: Processing settings
output: "feeds.opml"
concurrency: 16
tags: []  # Empty = all bookmarks

# Optional: Logging
verbose: false
debug: false
quiet: false
```

### Command-Line Flags

All configuration options can be overridden with command-line flags:

```bash
# Required
--linkding-token string     Linkding API token
--linkding-url string       Linkding server URL

# Optional
--tags strings              Filter by tags (comma-separated)
--output string             Output OPML file path (default: feeds.opml)
--cache string              Cache file path
--max-age int               Cache max-age in hours (default: 720)
--concurrency int           Number of concurrent workers (default: 16)
--config string             Configuration file path

# Logging
--verbose                   Enable verbose logging
--debug                     Enable debug logging  
--quiet                     Suppress summary output
```

## Usage Examples

### Export Examples (Linkding → OPML)

#### Export all bookmarks
```bash
./linkding-to-opml export --linkding-url https://linkding.example.com --linkding-token abc123
```

#### Export bookmarks with specific tags (AND operation)
```bash
./linkding-to-opml export --tags "rss,tech" --output tech-feeds.opml
```

#### Use custom cache location and max-age
```bash
./linkding-to-opml export --cache /tmp/my-cache.gob --max-age 168  # 1 week
```

#### Enable verbose logging to see progress
```bash
./linkding-to-opml export --verbose
```

#### Export with custom concurrency
```bash
./linkding-to-opml export --concurrency 8  # Use 8 workers instead of 16
```

### Import Examples (OPML → Linkding)

#### Basic OPML to HTML conversion
```bash
./linkding-to-opml tohtml feeds.opml bookmarks.html
```

#### Preview conversion without creating file
```bash
./linkding-to-opml tohtml --dry-run feeds.opml bookmarks.html
```

#### Add tags to all converted bookmarks
```bash
./linkding-to-opml tohtml --tags "imported,feeds" my-feeds.opml import.html
```

#### Use custom concurrency for feed processing
```bash
./linkding-to-opml tohtml --concurrency 8 feeds.opml bookmarks.html
```

#### Process with verbose logging
```bash
./linkding-to-opml tohtml --verbose feeds.opml bookmarks.html
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
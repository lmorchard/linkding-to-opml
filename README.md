# linkding-to-opml

A command-line tool that fetches bookmarks from your [Linkding](https://github.com/sissbruecker/linkding) instance, discovers RSS/Atom feeds from those URLs, and exports them as an OPML file for use in feed readers.

## Features

- 🔗 Fetches bookmarks from Linkding API with optional tag filtering
- 📡 Automatically discovers RSS/Atom feeds using standard autodiscovery methods
- ⚡ Concurrent processing for fast operation (configurable worker pool)
- 💾 Intelligent caching system to avoid repeated network requests
- 📄 Generates OPML 2.0 compatible files
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

### Export all bookmarks
```bash
./linkding-to-opml export --linkding-url https://linkding.example.com --linkding-token abc123
```

### Export bookmarks with specific tags (AND operation)
```bash
./linkding-to-opml export --tags "rss,tech" --output tech-feeds.opml
```

### Use custom cache location and max-age
```bash
./linkding-to-opml export --cache /tmp/my-cache.gob --max-age 168  # 1 week
```

### Enable verbose logging to see progress
```bash
./linkding-to-opml export --verbose
```

### Export with custom concurrency
```bash
./linkding-to-opml export --concurrency 8  # Use 8 workers instead of 16
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
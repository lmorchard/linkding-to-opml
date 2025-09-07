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

## How It Works

1. **Fetch Bookmarks**: Connects to your Linkding instance and fetches bookmarks (optionally filtered by tags)

2. **Check Cache**: For each bookmark URL, checks if we've already discovered feeds recently (within max-age)

3. **Discover Feeds**: For uncached URLs, fetches the webpage and looks for RSS/Atom feed links using standard autodiscovery methods

4. **Extract Feed Info**: Fetches discovered feed URLs and extracts feed titles

5. **Generate OPML**: Creates an OPML 2.0 document with:
   - `title`: Feed title from feed metadata
   - `xmlUrl`: Discovered feed URL
   - `htmlUrl`: Original bookmark URL

6. **Save Results**: Writes the OPML file and updates the cache

## Error Handling

The tool handles various error conditions gracefully:

- **Network errors**: Logs warnings and continues processing other bookmarks
- **No feeds found**: Logs warnings but doesn't stop the process
- **Malformed feeds**: Skips invalid feeds and continues
- **API errors**: Stops processing with clear error messages
- **File permission errors**: Clear error messages with suggestions

## Performance Tips

- **Concurrent processing**: Adjust `--concurrency` based on your system and network
- **Cache utilization**: Regular runs benefit from cached feed discoveries
- **Tag filtering**: Use specific tags to process fewer bookmarks
- **Network timeout**: Increase HTTP timeout for slow networks

## License

This project is licensed under the MIT License - see the LICENSE file for details.
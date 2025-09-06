# Dev Session Spec: Initial Setup

## Overview
Build a command-line utility in Go that fetches bookmarks from Linkding, discovers RSS/Atom feeds from those URLs, and exports them as an OPML file.

## Technology Stack
- **Language**: Go
- **CLI Framework**: Cobra
- **Configuration**: Viper (YAML config files)
- **Logging**: Logrus
- **Linkding API**: https://github.com/piero-vic/go-linkding
- **Cache Storage**: encoding/gob
- **OPML Generation**: XML library

## Core Functionality

### Bookmark Fetching
- Fetch bookmarks from Linkding API using user's token and server URL
- Support filtering options:
  - No tags: fetch ALL bookmarks
  - Single tag: fetch bookmarks with that tag
  - Multiple tags: fetch bookmarks with ALL specified tags (AND operation)

### Feed Autodiscovery
- For each bookmark URL, discover RSS/Atom feeds using standard autodiscovery process: https://www.rssboard.org/rss-autodiscovery
- Take the first feed URL found per bookmark
- Process concurrently (default: 16 goroutines, configurable)

### Caching System
- Key/value cache: URL (key) → autodiscovery result (value)
- Storage: encoding/gob format
- Default location: `./linkding-to-opml.gob`
- Cache max-age: default 720 hours (30 days), configurable in hours
- Skip autodiscovery for cached entries newer than max-age

### OPML Output
- Flat list structure (no grouping)
- Each entry contains:
  - Feed title (from feed metadata)
  - htmlUrl = original bookmark URL
  - xmlUrl = discovered feed URL
- Default output: `feeds.opml`

## Command Structure

### Main Command: `export`
Primary subcommand for generating OPML files.

#### CLI Flags
- `--tags`: Comma-separated list of tags for filtering (optional)
- `--output`: OPML output file path (default: feeds.opml)
- `--config`: Configuration file override (default: ./linkding-to-opml.yaml)
- `--cache`: Cache file path override (default: ./linkding-to-opml.gob)
- `--max-age`: Cache max-age in hours override
- `--verbose`: Enable verbose logging (INFO level)
- `--debug`: Enable debug logging (DEBUG level)
- `--quiet`: Suppress summary output (errors/warnings still shown)
- `--linkding-token`: API token (required, unless in configuration)
- `--linkding-url`: Server URL (required, unless in configuration)
- `--linkding-timeout`: API timeout (optional)
- `--concurrency`: Number of concurrent workers (default: 16)

## Configuration File

### Default Location
- `./linkding-to-opml.yaml`
- Override with `--config` flag

### Configuration Properties
All CLI flags (except `--config`) have corresponding YAML properties:
- `linkding.token`: API token (required)
- `linkding.url`: Server URL (required)
- `linkding.timeout`: API timeout
- `output`: Default OPML output path
- `cache`: Cache file path
- `max_age`: Cache max-age in hours
- `tags`: Default tags list
- `http.timeout`: HTTP timeout for fetching pages (default: 30s)
- `http.user_agent`: User-Agent string (default: "Mozilla/5.0 (compatible; linkding-to-opml/1.0)")
- `http.max_redirects`: Maximum redirects to follow (default: 3)
- `concurrency`: Number of concurrent workers (default: 16)

## Error Handling
- **Unreachable URLs**: Log warning, continue processing
- **No feeds found**: Log warning, continue processing
- **API errors**: Log error and bail out
- **Failed URLs excluded**: Do not include in OPML output

## Progress Reporting
- **Verbose mode**: Full process narration + progress updates for URL fetching
- **Default mode**: Summary statistics at end (e.g., "Found 23 feeds from 200 bookmarks, 12 cached, 165 newly discovered")
- **Quiet mode**: Suppress summary, show only errors/warnings

## HTTP Client Settings
- Configurable timeout (default: 30 seconds)
- Configurable max redirects (default: 3)
- Browser-like User-Agent with tool identification
- No rate limiting - process as fast as possible with concurrency controls

## Future Considerations
- Multiple feed selection per URL
- Cache management subcommands
- Additional OPML organization options

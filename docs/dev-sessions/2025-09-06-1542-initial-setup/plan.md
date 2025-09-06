# Implementation Plan: linkding-to-opml

## Project Architecture Overview

The tool will be structured as a Go module with the following core components:
- CLI layer (Cobra commands)
- Configuration management (Viper)
- Linkding API client
- Feed autodiscovery engine
- Cache management
- OPML generation
- HTTP client with retry/timeout logic

## Phase 1: Foundation and Project Setup

### Step 1: Initialize Go Module and Basic Structure
**Prompt for LLM:**
```
Create a new Go project for a CLI tool called "linkding-to-opml". Initialize the go.mod file and create the basic directory structure:

- cmd/ (for CLI commands)
- internal/ (for internal packages)
- internal/config/ (configuration management)
- internal/linkding/ (Linkding API client)
- internal/cache/ (cache management)  
- internal/feeds/ (feed autodiscovery)
- internal/opml/ (OPML generation)
- main.go (entry point)

Add dependencies for:
- github.com/spf13/cobra (CLI framework)
- github.com/spf13/viper (configuration)
- github.com/sirupsen/logrus (logging)
- github.com/piero-vic/go-linkding (Linkding API)

Create a basic main.go that initializes a Cobra root command with version info and basic help.
```

### Step 2: Configuration Structure and Loading
**Prompt for LLM:**
```
Implement the configuration management system in internal/config/:

Create a Config struct that matches all the configuration properties from the spec:
- Linkding settings (token, url, timeout)
- Cache settings (file path, max-age)
- HTTP settings (timeout, user-agent, max-redirects)
- Output settings (opml file path)
- Processing settings (concurrency, tags)
- Logging settings (verbose, debug, quiet)

Implement functions to:
1. Load configuration from YAML file (default: ./linkding-to-opml.yaml)
2. Merge configuration with command-line flags using Viper
3. Validate required configuration (linkding token and URL)
4. Set up logrus with appropriate log levels based on config

Include a sample configuration YAML file with comments explaining each option.
```

### Step 3: Basic CLI Structure with Export Command
**Prompt for LLM:**
```
Create the CLI command structure using Cobra in cmd/:

1. Implement the root command with global flags:
   - --config (config file path override)
   - --verbose, --debug, --quiet (logging levels)

2. Implement the export subcommand with flags matching the spec:
   - --tags, --output, --cache, --max-age
   - --linkding-token, --linkding-url, --linkding-timeout
   - --concurrency

3. Wire the export command to:
   - Load and validate configuration
   - Set up logging based on flags
   - Call a placeholder export function that prints configuration summary

4. Update main.go to execute the root command

The export command should validate that required Linkding credentials are available either via flags or config file, and exit with helpful error messages if missing.
```

## Phase 2: Core Components

### Step 4: Cache Management System
**Prompt for LLM:**
```
Implement the cache management system in internal/cache/:

Create a Cache struct that:
1. Stores URL -> FeedDiscoveryResult mappings using encoding/gob
2. Tracks timestamp for each cache entry
3. Supports max-age validation (entries older than max-age hours are considered stale)

Implement:
- CacheEntry struct with URL, FeedURL, FeedTitle, Timestamp fields
- LoadCache() function to read from gob file (handle missing file gracefully)
- SaveCache() function to write to gob file
- Get(url string, maxAgeHours int) function that returns cached result if fresh, nil if stale/missing
- Set(url, feedURL, feedTitle string) function to store new cache entries
- IsStale(entry, maxAgeHours) function to check cache freshness

Include proper error handling and logging. The cache should gracefully handle corrupted cache files by starting fresh.
```

### Step 5: Linkding API Integration
**Prompt for LLM:**
```
Implement the Linkding API client wrapper in internal/linkding/:

Create a Client struct that wraps the go-linkding library with:
1. Configuration for token, URL, and timeout
2. Method to fetch bookmarks with optional tag filtering
3. Support for AND operation when multiple tags provided
4. Proper error handling and logging

Implement:
- NewClient(token, url, timeout) function
- FetchBookmarks(tags []string) function that:
  - Fetches all bookmarks if no tags provided
  - Filters by single tag if one tag provided  
  - Filters by multiple tags using AND logic if multiple tags provided
  - Returns slice of Bookmark structs with URL, Title, and Tags fields
  - Handles API errors with informative messages

Include comprehensive error handling for network issues, authentication failures, and API errors. Log the number of bookmarks fetched at INFO level.
```

### Step 6: HTTP Client for Feed Discovery
**Prompt for LLM:**
```
Create a configurable HTTP client in internal/feeds/ for fetching web pages:

Implement:
- HTTPClient struct with configurable timeout, user-agent, and max redirects
- NewHTTPClient(config) function that creates http.Client with:
  - Custom timeout from configuration
  - Custom user-agent (default: "Mozilla/5.0 (compatible; linkding-to-opml/1.0)")
  - Redirect policy limiting max redirects (default: 3)
  - Proper error handling for redirects, timeouts

- FetchPage(url string) function that:
  - Makes HTTP GET request to URL
  - Returns response body as string and error
  - Handles common HTTP errors (404, 500, timeout, etc.)
  - Logs warnings for failed requests
  - Returns empty string and error for failures

Include retry logic for temporary failures and comprehensive error handling. This will be used by the feed autodiscovery system in the next step.
```

### Step 7: Feed Autodiscovery Engine
**Prompt for LLM:**
```
Implement feed autodiscovery in internal/feeds/ building on the HTTP client:

Create:
- FeedDiscoveryResult struct with FeedURL, FeedTitle, Error fields
- DiscoverFeed(url string, httpClient *HTTPClient) function that:
  1. Fetches the webpage HTML using the HTTP client
  2. Parses HTML for feed autodiscovery links following RSS autodiscovery spec
  3. Looks for <link> tags with rel="alternate" and type="application/rss+xml" or "application/atom+xml"
  4. If autodiscovery links found, fetches the first feed URL found
  5. Parses the feed XML to extract the feed title
  6. Returns FeedDiscoveryResult with feed URL and title
  7. Returns error result if no feeds found or if feed is invalid

Use Go's html package for HTML parsing and xml package for feed parsing. Handle malformed HTML/XML gracefully. Log detailed information at DEBUG level and warnings for failed discoveries.

The function should return the first valid feed found, not try multiple feeds from the same page.
```

## Phase 3: Integration and Processing

### Step 8: Concurrent Processing Engine
**Prompt for LLM:**
```
Create a concurrent bookmark processing system in internal/feeds/:

Implement:
- ProcessBookmarks function that takes:
  - Slice of bookmarks from Linkding
  - Cache instance
  - HTTP client
  - Configuration (concurrency level, max-age, logging settings)

The function should:
1. Create a worker pool with configurable concurrency (default 16)
2. For each bookmark:
   - Check cache first (skip if fresh entry exists)
   - If not cached or stale, perform feed autodiscovery
   - Update cache with results (both successful and failed attempts)
   - Log progress in verbose mode ("Processing bookmark 45/200: example.com")
3. Collect all successful feed discoveries
4. Save updated cache to disk
5. Return slice of successful feed discovery results

Use Go channels and goroutines for concurrent processing. Include proper error handling and graceful shutdown. Track statistics for successful discoveries, cache hits, and failures.
```

### Step 9: OPML Generation
**Prompt for LLM:**
```
Implement OPML file generation in internal/opml/:

Create:
- OPMLOutline struct matching OPML specification with xmlUrl, htmlUrl, title attributes
- OPML struct with head and body containing outlines
- GenerateOPML function that takes feed discovery results and creates OPML structure
- WriteOPML function that marshals OPML to XML file

The OPML should:
1. Have proper OPML 2.0 XML structure with head and body elements
2. Create flat list of outline elements (no grouping/folders)
3. For each successful feed discovery:
   - title = feed title from feed metadata
   - xmlUrl = discovered feed URL  
   - htmlUrl = original bookmark URL from Linkding
4. Include proper XML encoding and formatting
5. Handle file writing errors gracefully

Follow OPML 2.0 specification for proper XML structure and attributes. Include XML declaration and proper encoding.
```

### Step 10: Statistics and Progress Reporting
**Prompt for LLM:**
```
Add comprehensive statistics and progress reporting throughout the application:

Create internal/stats/ package with:
- ProcessingStats struct tracking:
  - Total bookmarks processed
  - Cache hits vs new discoveries  
  - Successful feed discoveries
  - Failed/no-feed URLs
  - Processing time
  
- StatTracker that:
  - Collects stats during processing
  - Formats summary report for end-user
  - Provides progress callbacks for verbose mode

Update the processing engine to:
1. Call progress callbacks during verbose logging
2. Track all statistics during processing
3. Display final summary unless --quiet flag is used

Format example: "Found 23 feeds from 200 bookmarks, 12 cached, 165 newly discovered, 35 failed (Processing time: 45s)"

Include elapsed time tracking and proper formatting for user-friendly output.
```

## Phase 4: Final Integration and Testing

### Step 11: Wire Everything Together in Export Command
**Prompt for LLM:**
```
Complete the export command implementation by integrating all components:

Update cmd/export.go to:
1. Load and validate configuration
2. Initialize all components (cache, Linkding client, HTTP client, etc.)
3. Fetch bookmarks from Linkding API
4. Process bookmarks with concurrent feed discovery
5. Generate and write OPML file
6. Display progress and final statistics
7. Handle all errors gracefully with informative messages

The export flow should be:
1. Load config and validate Linkding credentials
2. Initialize cache from disk
3. Create Linkding API client and fetch bookmarks
4. Create HTTP client and processing components
5. Process bookmarks concurrently with feed discovery
6. Generate OPML from successful discoveries
7. Write OPML to specified output file
8. Save updated cache to disk
9. Display summary statistics (unless --quiet)

Include comprehensive error handling at each step with helpful error messages for users. Log the complete process flow in verbose mode.
```

### Step 12: Error Handling and Edge Cases
**Prompt for LLM:**
```
Add comprehensive error handling and edge case management throughout the application:

1. Configuration validation:
   - Helpful error messages for missing required fields
   - Validation for file paths and URLs
   - Warning for unusual configuration values

2. Network error handling:
   - Retry logic for temporary failures
   - Timeout handling with helpful messages
   - SSL/TLS error handling
   - DNS resolution failures

3. File system operations:
   - Permission errors for cache and output files
   - Disk space issues
   - Directory creation for output paths

4. Data validation:
   - Malformed URLs from Linkding
   - Invalid XML in feeds
   - Character encoding issues

5. Graceful degradation:
   - Continue processing when individual URLs fail
   - Partial success scenarios
   - Cache corruption recovery

Update all components to handle these scenarios gracefully and provide helpful error messages to users. Include suggestions for common fixes in error messages.
```

### Step 13: Documentation and Usage Examples
**Prompt for LLM:**
```
Add comprehensive help text and usage examples to the CLI application:

1. Update root command with detailed description and usage examples
2. Update export command with:
   - Detailed flag descriptions
   - Usage examples for common scenarios
   - Configuration file examples

3. Create example configuration file with comments explaining all options
4. Add version information and build metadata

Examples should cover:
- Basic usage with minimal configuration
- Using configuration files vs command-line flags
- Tag filtering scenarios
- Cache management advice
- Troubleshooting common issues

Ensure all help text is clear, concise, and includes practical examples that users can copy and modify for their needs.
```

## Implementation Notes

### Best Practices Applied
- **Incremental Development**: Each step builds on previous work without orphaned code
- **Error Handling**: Comprehensive error handling at each layer
- **Testability**: Components are designed for easy unit testing
- **Configuration**: Flexible configuration via files and CLI flags
- **Logging**: Appropriate logging levels throughout
- **Concurrency**: Safe concurrent processing with proper synchronization
- **Resource Management**: Proper cleanup of resources and graceful shutdown

### Dependencies Management
- All dependencies are established in Step 1
- No new major dependencies introduced in later steps
- Each step validates that required dependencies are working

### Integration Points
- Configuration flows through all components
- Cache is shared between processing steps
- HTTP client is reused for all network operations
- Statistics are collected consistently throughout

This plan ensures that each step produces working, testable code that integrates with previous steps, leading to a complete, robust application.
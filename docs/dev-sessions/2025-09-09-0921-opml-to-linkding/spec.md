# Dev Session Spec: OPML to Linkding

## Current State
Currently, this tool has one command, "export", which fetches Linkding bookmarks and produces an OPML file as output from feeds detected from those bookmarks.

## Goal
We'd like to implement a second command, "import", which takes an OPML file, tries to find the HTML web pages associated with those feeds (either via the htmlURL attribute or by fetching the feeds and chasing down further links), and then imports those pages into Linkding as new bookmarks.

## Requirements

### Core Functionality
- Parse OPML files and extract feed entries
- Traverse nested folder structures but import as flat list (ignore hierarchy)
- Discover HTML pages from feed entries using:
  1. htmlURL attribute in OPML (if present)
  2. Fetch feed and extract associated website links
  3. Fallback to feed URL itself if HTML page cannot be found
- Create bookmarks in Linkding via API

### Data Extraction Priority
For each bookmark, extract the following metadata in order of priority:
1. From OPML attributes (if available)
2. From feed metadata (if feed is fetched)
3. From HTML page metadata (if page is fetched)

Fields to extract:
- URL (required)
- Title
- Description

### Configuration Options

#### CLI Arguments
- `--dry-run`: Preview import without creating bookmarks
- `--verbose`: Enable INFO level logging
- `--debug`: Enable DEBUG level logging
- `--duplicates [skip|update]`: How to handle existing bookmarks (default: skip)
- `--tags TAG1,TAG2`: Comma-separated tags to apply to all imported bookmarks
- `--concurrency N`: Number of concurrent web fetches (default: 16)

#### Config File Options
Same options available via configuration file as used by export command:
- Linkding API URL
- Linkding API token
- Default duplicate handling strategy
- Default tags
- Default concurrency level

### Processing Behavior
- **Import Strategy**: Progressive - import bookmarks as they are processed
- **Error Handling**: Continue on failures, log errors, skip failed entries
- **Duplicate Detection**: Check if URL already exists in Linkding
- **Rate Limiting**: No delays between requests, no automatic retries
- **Concurrency**: Parallel fetching with configurable limit (default: 16)

### Logging and Output
- **Default**: Show errors and final statistics only
- **Verbose Mode**: Add INFO level messages showing progress
- **Debug Mode**: Add DEBUG level messages with detailed processing info
- **Missing HTML Pages**: Log as INFO when falling back to feed URL
- **Summary Statistics**: Display at end:
  - Total entries processed
  - Bookmarks successfully imported
  - Duplicates skipped/updated
  - Failed imports

### Authentication
- Use same configuration as export command
- Support for API token via:
  - Config file
  - Environment variables
  - CLI arguments

## Success Criteria
- Successfully parse standard OPML files with nested structures
- Import bookmarks with appropriate metadata
- Handle errors gracefully without stopping entire import
- Provide clear feedback via logging and summary statistics
- Dry-run mode accurately previews what would be imported
- Respect duplicate handling preferences
- Apply custom tags when specified
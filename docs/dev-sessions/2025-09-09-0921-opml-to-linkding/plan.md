# Dev Session Plan: OPML to Linkding Import

## Overview
This plan implements a new "import" command that reads OPML files and creates bookmarks in Linkding. The implementation is broken down into small, iterative chunks that build upon each other.

## Implementation Phases

### Phase 1: Foundation - Basic Command Structure
1. Create the import command skeleton
2. Add CLI argument parsing
3. Integrate with existing config system

### Phase 2: OPML Processing
1. Create OPML parser to extract feed entries
2. Add recursive traversal for nested structures
3. Build flat list of feed items

### Phase 3: URL Discovery
1. Extract URLs from OPML attributes (htmlUrl, xmlUrl)
2. Create feed fetcher to get feed content
3. Implement HTML page discovery from feeds
4. Add fallback logic for missing pages

### Phase 4: Linkding Integration  
1. Add bookmark creation API calls
2. Implement duplicate detection
3. Add update vs skip logic for duplicates

### Phase 5: Concurrency & Performance
1. Implement worker pool for concurrent fetching
2. Add configurable concurrency limits
3. Ensure thread-safe operations

### Phase 6: Features & Polish
1. Add dry-run mode
2. Implement summary statistics tracking
3. Add proper logging levels
4. Wire everything together with tests

## Detailed Implementation Steps

### Step 1: Create Import Command Skeleton
- Add new file `cmd/import.go`
- Register command with cobra
- Create basic command structure with help text

### Step 2: Add Import Command CLI Arguments
- Add flags for OPML file input
- Add --dry-run flag
- Add --duplicates flag (skip/update)
- Add --tags flag for custom tags
- Add --concurrency flag
- Bind flags to viper config

### Step 3: Create OPML Reader Module
- Create `internal/opml/reader.go`
- Define structs for OPML parsing
- Implement basic XML unmarshaling
- Add method to extract all feed entries

### Step 4: Implement Recursive OPML Traversal
- Add recursive function to traverse outline elements
- Flatten nested structures into single list
- Extract xmlUrl, htmlUrl, title, and text attributes

### Step 5: Create Feed Entry Data Structure
- Define FeedEntry struct with URL, title, description
- Create ImportItem struct to track processing state
- Add methods for data extraction priority

### Step 6: Implement Feed Fetcher
- Create `internal/feeds/fetcher.go` 
- Add HTTP client for fetching feeds
- Parse RSS/Atom feeds
- Extract website links from feed metadata

### Step 7: Add HTML Discovery Logic
- Implement htmlUrl extraction from OPML
- Add feed parsing to find website links
- Create fallback to use feed URL
- Handle missing pages gracefully

### Step 8: Create Linkding Bookmark Creator
- Add CreateBookmark method to linkding client
- Include title, URL, description fields
- Support custom tags parameter

### Step 9: Implement Duplicate Detection
- Add GetBookmarkByURL to linkding client
- Check for existing bookmarks before creating
- Return existing bookmark info for comparison

### Step 10: Add Duplicate Handling Logic
- Implement skip behavior for duplicates
- Implement update behavior for duplicates
- Make behavior configurable via CLI/config

### Step 11: Create Worker Pool
- Implement concurrent processing with channels
- Add worker pool with configurable size
- Ensure thread-safe bookmark creation

### Step 12: Add Progress Tracking
- Create ImportStats struct
- Track processed, imported, skipped, failed counts
- Update stats atomically during processing

### Step 13: Implement Dry Run Mode
- Add flag checking before API calls
- Log what would be done without doing it
- Show same statistics as real run

### Step 14: Add Logging Levels
- Integrate with existing logging setup
- Add INFO level progress messages
- Add DEBUG level detailed processing info
- Default to errors + summary only

### Step 15: Wire Everything Together
- Connect all components in import command
- Add error handling throughout
- Display final summary statistics
- Run integration tests

## LLM Implementation Prompts

---

### Prompt 1: Create Import Command Skeleton

**Context:** We have a Go CLI tool using Cobra for commands. The existing `export` command is in `cmd/export.go`. We need to add a new `import` command.

**Task:** Create a new file `cmd/import.go` with a basic import command structure. The command should:
- Use cobra.Command with Use: "import"
- Accept a positional argument for the OPML file path
- Have appropriate short and long descriptions
- Include a RunE function that just prints a placeholder message
- Register the command in the init() function

Follow the same pattern as the existing export command but simplified for now.

---

### Prompt 2: Add Import Command CLI Arguments

**Context:** We have a basic import command. Now we need to add all the CLI flags specified in the spec.

**Task:** Update `cmd/import.go` to add these flags:
- `--dry-run` (bool): Preview without creating bookmarks
- `--duplicates` (string): "skip" or "update", default "skip"
- `--tags` (string slice): Comma-separated tags to apply
- `--concurrency` (int): Number of workers, default 16
- Use viper.BindPFlag for each flag
- Add validation in PreRunE to ensure OPML file exists
- Ensure flags work with config file values

---

### Prompt 3: Create OPML Reader Module

**Context:** We need to parse OPML files. OPML is XML with nested outline elements containing feed information.

**Task:** Create `internal/opml/reader.go` with:
- OPML struct matching OPML 2.0 spec
- Outline struct with XMLUrl, HtmlUrl, Title, Text, Type attributes
- Outline should have Children []Outline for nesting
- ReadFile function that takes a path and returns parsed OPML
- Proper XML struct tags for unmarshaling
- Error handling for invalid files

---

### Prompt 4: Implement Recursive OPML Traversal

**Context:** OPML files can have nested folder structures. We need to flatten them into a list of feed entries.

**Task:** Add to `internal/opml/reader.go`:
- GetAllFeeds() method on OPML struct that returns []FeedEntry
- FeedEntry struct with XMLUrl, HtmlUrl, Title, Description fields
- Recursive traversal of outline elements
- Skip outline elements without xmlUrl (folders)
- Collect all feed entries into flat list

---

### Prompt 5: Create Import Item Structure

**Context:** We need a data structure to track each item being imported with its processing state.

**Task:** Create `internal/importer/types.go` with:
- ImportItem struct containing:
  - FeedEntry (embedded)
  - DiscoveredURL string (final URL to bookmark)
  - DiscoveredTitle string
  - DiscoveredDescription string
  - Status (pending/success/skipped/failed)
  - Error (if failed)
- ImportStats struct to track counts
- Helper methods to update item with discovered data

---

### Prompt 6: Implement Feed Fetcher

**Context:** We need to fetch RSS/Atom feeds to discover website URLs when htmlUrl is missing from OPML.

**Task:** Update or create `internal/feeds/fetcher.go` to add:
- FetchFeed(url string) (*Feed, error) function
- Extract website link from feed's Link field
- Support both RSS and Atom feed formats
- Use existing HTTP client configuration
- Return parsed feed with metadata
- Handle timeouts and errors gracefully

---

### Prompt 7: Add HTML Discovery Logic

**Context:** We need a three-tier fallback system for discovering the URL to bookmark.

**Task:** Create `internal/importer/discovery.go` with:
- DiscoverBookmarkURL(item *ImportItem) error function
- First check htmlUrl from OPML
- If missing, fetch feed and look for website link
- If still missing, fall back to xmlUrl
- Update ImportItem with discovered URL, title, description
- Log at INFO level when falling back to feed URL

---

### Prompt 8: Add Linkding Bookmark Creation

**Context:** The existing linkding client fetches bookmarks. We need to add creation capability.

**Task:** Update `internal/linkding/client.go` to add:
- CreateBookmark(url, title, description string, tags []string) (*Bookmark, error)
- Use POST /api/bookmarks/ endpoint
- Include all fields in request body
- Handle API errors appropriately
- Return created bookmark structure

---

### Prompt 9: Implement Duplicate Detection

**Context:** Before creating bookmarks, we need to check if they already exist.

**Task:** Update `internal/linkding/client.go` to add:
- GetBookmarkByURL(url string) (*Bookmark, error)
- Use GET /api/bookmarks/ with url parameter
- Return nil, nil if not found (not an error)
- Return bookmark if found
- Handle pagination if needed

---

### Prompt 10: Add Duplicate Handling Logic

**Context:** We have duplicate detection. Now we need to handle duplicates based on user preference.

**Task:** Update `internal/linkding/client.go` to add:
- UpdateBookmark(id int, url, title, description string, tags []string) error
- Use PUT /api/bookmarks/{id}/ endpoint

Create `internal/importer/processor.go` with:
- ProcessBookmark(item *ImportItem, duplicates string, tags []string) error
- Check for existing bookmark
- Skip if exists and duplicates="skip"
- Update if exists and duplicates="update"
- Create if doesn't exist

---

### Prompt 11: Create Worker Pool

**Context:** We need concurrent processing for performance.

**Task:** Update `internal/importer/processor.go` to add:
- ProcessItems(items []*ImportItem, concurrency int, options ProcessOptions) *ImportStats
- Create worker pool with specified concurrency
- Use channels for work distribution
- Process items concurrently
- Collect results safely
- Update stats atomically

---

### Prompt 12: Add Progress Tracking

**Context:** We need to track and report import statistics.

**Task:** Update `internal/importer/types.go` and `processor.go`:
- Add atomic counters to ImportStats
- Increment counters thread-safely during processing
- Add Summary() method to format statistics
- Track: total, imported, updated, skipped, failed
- Include timing information

---

### Prompt 13: Implement Dry Run Mode

**Context:** Users want to preview what would happen without making changes.

**Task:** Update `internal/importer/processor.go`:
- Add DryRun bool to ProcessOptions
- Skip actual API calls when dry-run is true
- Still perform discovery and duplicate checking
- Log what would be done
- Return same statistics structure
- Clearly indicate dry-run in output

---

### Prompt 14: Add Logging Integration

**Context:** We have verbose and debug flags. We need proper logging throughout the import process.

**Task:** Update all importer files to:
- Use existing logging setup from export command
- Add INFO logs for major steps (when verbose=true)
- Add DEBUG logs for detailed processing (when debug=true)
- Default to only errors and final summary
- Ensure consistent log formatting
- Add progress indicators for long operations

---

### Prompt 15: Wire Everything Together

**Context:** All components are built. Now we need to connect them in the import command.

**Task:** Update `cmd/import.go` RunE function to:
1. Load configuration
2. Read and parse OPML file
3. Extract all feed entries
4. Create ImportItems from entries
5. Initialize linkding client
6. Create processor with options from flags
7. Process all items with worker pool
8. Display summary statistics
9. Handle errors appropriately
10. Return proper exit codes

Add basic integration test to verify the flow works end-to-end.

---

## Testing Strategy

### Unit Tests
- OPML parsing with various file formats
- Feed discovery logic with fallbacks
- Duplicate detection and handling
- Statistics tracking

### Integration Tests
- Full import flow with mock Linkding API
- Dry-run mode verification
- Concurrent processing correctness
- Error handling scenarios

### Manual Testing
- Real OPML files from popular feed readers
- Various Linkding configurations
- Performance with large OPML files
- Network failure scenarios

## Success Metrics
- Successfully imports standard OPML files
- Handles duplicates according to user preference
- Provides clear feedback via logging
- Dry-run accurately previews changes
- Concurrent processing improves performance
- Graceful error handling without data loss
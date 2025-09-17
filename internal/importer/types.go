package importer

import (
	"fmt"
	"sync/atomic"
	"time"

	"linkding-to-opml/internal/opml"
)

// ImportStatus represents the status of an import item
type ImportStatus int

const (
	StatusPending ImportStatus = iota
	StatusSuccess
	StatusSkipped
	StatusFailed
)

func (s ImportStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusSuccess:
		return "success"
	case StatusSkipped:
		return "skipped"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ImportItem represents a single item being imported with its processing state
type ImportItem struct {
	// Original feed entry from OPML
	opml.FeedEntry

	// Discovered data (final values to bookmark)
	DiscoveredURL         string
	DiscoveredTitle       string
	DiscoveredDescription string

	// Processing state
	Status     ImportStatus
	Error      error
	WasUpdated bool // True if this was an update rather than a new import
}

// UpdateWithDiscoveredData updates the item with discovered URL and metadata
func (item *ImportItem) UpdateWithDiscoveredData(url, title, description string) {
	item.DiscoveredURL = url
	
	// Use discovered title if available, otherwise fall back to original
	if title != "" {
		item.DiscoveredTitle = title
	} else {
		item.DiscoveredTitle = item.Title
	}
	
	// Use discovered description if available, otherwise fall back to original
	if description != "" {
		item.DiscoveredDescription = description
	} else {
		item.DiscoveredDescription = item.Description
	}
}

// GetFinalURL returns the URL that should be bookmarked
func (item *ImportItem) GetFinalURL() string {
	if item.DiscoveredURL != "" {
		return item.DiscoveredURL
	}
	return item.XMLURL // fallback to feed URL
}

// GetFinalTitle returns the title that should be used for the bookmark
func (item *ImportItem) GetFinalTitle() string {
	if item.DiscoveredTitle != "" {
		return item.DiscoveredTitle
	}
	if item.Title != "" {
		return item.Title
	}
	return "Untitled"
}

// GetFinalDescription returns the description that should be used for the bookmark
func (item *ImportItem) GetFinalDescription() string {
	if item.DiscoveredDescription != "" {
		return item.DiscoveredDescription
	}
	return item.Description
}

// ImportStats tracks statistics during the import process
type ImportStats struct {
	StartTime time.Time
	EndTime   time.Time
	
	Total     int64
	Processed int64
	Imported  int64
	Updated   int64
	Skipped   int64
	Failed    int64
}

// NewImportStats creates a new ImportStats with start time
func NewImportStats(total int) *ImportStats {
	return &ImportStats{
		StartTime: time.Now(),
		Total:     int64(total),
	}
}

// IncrementProcessed atomically increments the processed counter
func (s *ImportStats) IncrementProcessed() {
	atomic.AddInt64(&s.Processed, 1)
}

// IncrementImported atomically increments the imported counter
func (s *ImportStats) IncrementImported() {
	atomic.AddInt64(&s.Imported, 1)
}

// IncrementUpdated atomically increments the updated counter
func (s *ImportStats) IncrementUpdated() {
	atomic.AddInt64(&s.Updated, 1)
}

// IncrementSkipped atomically increments the skipped counter
func (s *ImportStats) IncrementSkipped() {
	atomic.AddInt64(&s.Skipped, 1)
}

// IncrementFailed atomically increments the failed counter
func (s *ImportStats) IncrementFailed() {
	atomic.AddInt64(&s.Failed, 1)
}

// Finish marks the import as complete and records end time
func (s *ImportStats) Finish() {
	s.EndTime = time.Now()
}

// Duration returns the total duration of the import
func (s *ImportStats) Duration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// Summary returns a formatted summary of the import statistics
func (s *ImportStats) Summary() string {
	duration := s.Duration()
	
	return fmt.Sprintf(`Import Summary:
  Total entries: %d
  Processed: %d
  Imported: %d
  Updated: %d  
  Skipped: %d
  Failed: %d
  Duration: %v`,
		s.Total,
		atomic.LoadInt64(&s.Processed),
		atomic.LoadInt64(&s.Imported),
		atomic.LoadInt64(&s.Updated),
		atomic.LoadInt64(&s.Skipped),
		atomic.LoadInt64(&s.Failed),
		duration)
}
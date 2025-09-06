package stats

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// ProcessingStats tracks statistics during bookmark processing
type ProcessingStats struct {
	// Counters (use atomic operations for thread safety)
	TotalBookmarks    int64 `json:"total_bookmarks"`
	CacheHits         int64 `json:"cache_hits"`
	NewDiscoveries    int64 `json:"new_discoveries"`
	SuccessfulFeeds   int64 `json:"successful_feeds"`
	FailedDiscoveries int64 `json:"failed_discoveries"`
	
	// Timing
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time,omitempty"`
	ProcessingTime time.Duration `json:"processing_time"`
	
	// Progress tracking
	mu               sync.RWMutex
	progressCallback func(processed, total int64, url string, success bool)
}

// StatTracker manages statistics collection during processing
type StatTracker struct {
	stats *ProcessingStats
}

// NewStatTracker creates a new statistics tracker
func NewStatTracker(totalBookmarks int) *StatTracker {
	return &StatTracker{
		stats: &ProcessingStats{
			TotalBookmarks: int64(totalBookmarks),
			StartTime:      time.Now(),
		},
	}
}

// SetProgressCallback sets a callback function for progress reporting
func (st *StatTracker) SetProgressCallback(callback func(processed, total int64, url string, success bool)) {
	st.stats.mu.Lock()
	defer st.stats.mu.Unlock()
	st.stats.progressCallback = callback
}

// RecordCacheHit increments the cache hit counter
func (st *StatTracker) RecordCacheHit(url string) {
	atomic.AddInt64(&st.stats.CacheHits, 1)
	st.reportProgress(url, true)
}

// RecordNewDiscovery increments the new discovery counter
func (st *StatTracker) RecordNewDiscovery(url string, success bool) {
	atomic.AddInt64(&st.stats.NewDiscoveries, 1)
	
	if success {
		atomic.AddInt64(&st.stats.SuccessfulFeeds, 1)
	} else {
		atomic.AddInt64(&st.stats.FailedDiscoveries, 1)
	}
	
	st.reportProgress(url, success)
}

// reportProgress calls the progress callback if set
func (st *StatTracker) reportProgress(url string, success bool) {
	st.stats.mu.RLock()
	callback := st.stats.progressCallback
	st.stats.mu.RUnlock()
	
	if callback != nil {
		processed := atomic.LoadInt64(&st.stats.CacheHits) + atomic.LoadInt64(&st.stats.NewDiscoveries)
		callback(processed, st.stats.TotalBookmarks, url, success)
	}
}

// Finish marks the end of processing and calculates final timing
func (st *StatTracker) Finish() {
	st.stats.EndTime = time.Now()
	st.stats.ProcessingTime = st.stats.EndTime.Sub(st.stats.StartTime)
	
	logrus.WithFields(logrus.Fields{
		"total_bookmarks":    st.stats.TotalBookmarks,
		"cache_hits":         st.stats.CacheHits,
		"new_discoveries":    st.stats.NewDiscoveries,
		"successful_feeds":   st.stats.SuccessfulFeeds,
		"failed_discoveries": st.stats.FailedDiscoveries,
		"processing_time":    st.stats.ProcessingTime,
	}).Info("Processing statistics finalized")
}

// GetStats returns a copy of the current statistics
func (st *StatTracker) GetStats() ProcessingStats {
	return ProcessingStats{
		TotalBookmarks:    atomic.LoadInt64(&st.stats.TotalBookmarks),
		CacheHits:         atomic.LoadInt64(&st.stats.CacheHits),
		NewDiscoveries:    atomic.LoadInt64(&st.stats.NewDiscoveries),
		SuccessfulFeeds:   atomic.LoadInt64(&st.stats.SuccessfulFeeds),
		FailedDiscoveries: atomic.LoadInt64(&st.stats.FailedDiscoveries),
		StartTime:         st.stats.StartTime,
		EndTime:           st.stats.EndTime,
		ProcessingTime:    st.stats.ProcessingTime,
	}
}

// FormatSummary creates a user-friendly summary of processing results
func (st *StatTracker) FormatSummary(quiet bool) string {
	if quiet {
		return ""
	}
	
	stats := st.GetStats()
	
	return fmt.Sprintf("Found %d feeds from %d bookmarks, %d cached, %d newly discovered, %d failed (Processing time: %v)",
		stats.SuccessfulFeeds, 
		stats.TotalBookmarks, 
		stats.CacheHits, 
		stats.NewDiscoveries, 
		stats.FailedDiscoveries, 
		stats.ProcessingTime.Round(time.Second))
}

// FormatProgressUpdate creates a progress update message
func FormatProgressUpdate(processed, total int64, url string, success bool) string {
	status := "✓"
	if !success {
		status = "✗"
	}
	
	return fmt.Sprintf("[%d/%d] %s %s", processed, total, status, url)
}

// LogVerboseProgress logs detailed progress information
func (st *StatTracker) LogVerboseProgress(processed, total int64, url string, success bool, feedURL, feedTitle string) {
	progress := fmt.Sprintf("%d/%d", processed, total)
	
	if success {
		logrus.WithFields(logrus.Fields{
			"progress":   progress,
			"url":        url,
			"feed_url":   feedURL,
			"feed_title": feedTitle,
		}).Info("Successfully discovered feed")
	} else {
		logrus.WithFields(logrus.Fields{
			"progress": progress,
			"url":      url,
		}).Warn("Failed to discover feed")
	}
}

// GetProcessedCount returns the total number of processed bookmarks
func (st *StatTracker) GetProcessedCount() int64 {
	return atomic.LoadInt64(&st.stats.CacheHits) + atomic.LoadInt64(&st.stats.NewDiscoveries)
}

// GetSuccessRate returns the success rate as a percentage
func (st *StatTracker) GetSuccessRate() float64 {
	processed := st.GetProcessedCount()
	if processed == 0 {
		return 0.0
	}
	
	successful := atomic.LoadInt64(&st.stats.SuccessfulFeeds)
	return float64(successful) / float64(processed) * 100.0
}

// IsComplete returns true if all bookmarks have been processed
func (st *StatTracker) IsComplete() bool {
	return st.GetProcessedCount() >= st.stats.TotalBookmarks
}
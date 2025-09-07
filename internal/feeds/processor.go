package feeds

import (
	"fmt"
	"sync"
	"time"

	"linkding-to-opml/internal/cache"
	"linkding-to-opml/internal/linkding"

	"github.com/sirupsen/logrus"
)

// ProcessingConfig holds configuration for bookmark processing
type ProcessingConfig struct {
	Concurrency    int
	MaxAge         int
	UserAgent      string
	HTTPConfig     HTTPConfig
	Verbose        bool
	SaveFailedHTML bool
	DebugOutputDir string
}

// ProcessingStats holds statistics about the processing operation
type ProcessingStats struct {
	TotalBookmarks    int
	CacheHits         int
	NewDiscoveries    int
	SuccessfulFeeds   int
	FailedDiscoveries int
	ProcessingTime    time.Duration
}

// ProcessBookmarks processes bookmarks concurrently to discover feeds
func ProcessBookmarks(bookmarks []*linkding.Bookmark, cache *cache.Cache, config ProcessingConfig) ([]*FeedDiscoveryResult, *ProcessingStats) {
	startTime := time.Now()

	stats := &ProcessingStats{
		TotalBookmarks: len(bookmarks),
	}

	logrus.WithFields(logrus.Fields{
		"total_bookmarks": len(bookmarks),
		"concurrency":     config.Concurrency,
		"max_age_hours":   config.MaxAge,
	}).Info("Starting concurrent bookmark processing")

	// Create HTTP client for feed discovery
	httpClient := NewHTTPClient(config.HTTPConfig)

	// Create channels for work distribution
	bookmarkChan := make(chan *linkding.Bookmark, len(bookmarks))
	resultChan := make(chan *FeedDiscoveryResult, len(bookmarks))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go worker(i+1, bookmarkChan, resultChan, cache, httpClient, config, stats, &wg)
	}

	// Send bookmarks to workers
	for _, bookmark := range bookmarks {
		bookmarkChan <- bookmark
	}
	close(bookmarkChan)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var successful []*FeedDiscoveryResult

	processedCount := 0
	for result := range resultChan {
		processedCount++

		if config.Verbose {
			if result.IsSuccessful() {
				logrus.WithFields(logrus.Fields{
					"progress": fmt.Sprintf("%d/%d", processedCount, len(bookmarks)),
					"url":      result.URL,
					"feed":     result.FeedURL,
					"title":    result.FeedTitle,
				}).Info("Successfully discovered feed")
			} else {
				logrus.WithFields(logrus.Fields{
					"progress": fmt.Sprintf("%d/%d", processedCount, len(bookmarks)),
					"url":      result.URL,
					"error":    result.Error,
				}).Warn("Failed to discover feed")
			}
		}

		if result.IsSuccessful() {
			successful = append(successful, result)
			stats.SuccessfulFeeds++
		} else {
			stats.FailedDiscoveries++
		}
	}

	// Save updated cache
	if err := cache.SaveCache(); err != nil {
		logrus.WithError(err).Error("Failed to save cache after processing")
	} else {
		logrus.Debug("Successfully saved updated cache")
	}

	stats.ProcessingTime = time.Since(startTime)

	logrus.WithFields(logrus.Fields{
		"total_processed":    processedCount,
		"successful_feeds":   stats.SuccessfulFeeds,
		"failed_discoveries": stats.FailedDiscoveries,
		"cache_hits":         stats.CacheHits,
		"new_discoveries":    stats.NewDiscoveries,
		"processing_time":    stats.ProcessingTime,
	}).Info("Completed bookmark processing")

	return successful, stats
}

// worker processes bookmarks in a separate goroutine
func worker(workerID int, bookmarkChan <-chan *linkding.Bookmark, resultChan chan<- *FeedDiscoveryResult,
	cache *cache.Cache, httpClient *HTTPClient, config ProcessingConfig, stats *ProcessingStats, wg *sync.WaitGroup,
) {
	defer wg.Done()

	logrus.WithField("worker_id", workerID).Debug("Worker started")

	for bookmark := range bookmarkChan {
		result := processBookmark(bookmark, cache, httpClient, config, stats)
		resultChan <- result
	}

	logrus.WithField("worker_id", workerID).Debug("Worker finished")
}

// processBookmark processes a single bookmark, checking cache first
func processBookmark(bookmark *linkding.Bookmark, cache *cache.Cache, httpClient *HTTPClient,
	config ProcessingConfig, stats *ProcessingStats,
) *FeedDiscoveryResult {
	// Check cache first
	if cachedEntry := cache.Get(bookmark.URL, config.MaxAge); cachedEntry != nil {
		stats.CacheHits++

		logrus.WithFields(logrus.Fields{
			"url": bookmark.URL,
			"age": time.Since(cachedEntry.Timestamp),
		}).Debug("Using cached feed discovery result")

		result := &FeedDiscoveryResult{
			URL:       bookmark.URL,
			FeedURL:   cachedEntry.FeedURL,
			FeedTitle: cachedEntry.FeedTitle,
		}

		// Set error if this was a failed cache entry
		if !cachedEntry.HasFeed() {
			result.Error = fmt.Errorf("no feed found (cached)")
		}

		return result
	}

	// Perform new discovery
	stats.NewDiscoveries++

	logrus.WithField("url", bookmark.URL).Debug("Performing new feed discovery")

	result := DiscoverFeedWithDebug(bookmark.URL, httpClient, config.UserAgent, config.SaveFailedHTML, config.DebugOutputDir)

	// Update cache with result
	if result.IsSuccessful() {
		cache.Set(bookmark.URL, result.FeedURL, result.FeedTitle)
	} else {
		cache.SetFailed(bookmark.URL)
	}

	return result
}

// FormatProcessingSummary creates a user-friendly summary of processing results
func (s *ProcessingStats) FormatProcessingSummary(quiet bool) string {
	if quiet {
		return ""
	}

	return fmt.Sprintf("Found %d feeds from %d bookmarks, %d cached, %d newly discovered, %d failed (Processing time: %v)",
		s.SuccessfulFeeds, s.TotalBookmarks, s.CacheHits, s.NewDiscoveries, s.FailedDiscoveries, s.ProcessingTime.Round(time.Second))
}

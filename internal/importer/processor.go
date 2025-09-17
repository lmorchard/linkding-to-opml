package importer

import (
	"fmt"
	"sync"
	"time"

	"linkding-to-opml/internal/feeds"
	"linkding-to-opml/internal/netscape"

	"github.com/sirupsen/logrus"
)

// ProcessOptions contains options for processing bookmarks
type ProcessOptions struct {
	Tags          []string // Tags to apply to all bookmarks
	DryRun        bool     // If true, don't actually generate bookmarks
	RetryAttempts int      // Number of retry attempts for failed URL discovery
}

// ConvertToBookmark converts an ImportItem to a Netscape bookmark
func ConvertToBookmark(item *ImportItem, options ProcessOptions) netscape.Bookmark {
	// Build final tags list
	allTags := make([]string, len(options.Tags))
	copy(allTags, options.Tags)

	return netscape.Bookmark{
		URL:         item.GetFinalURL(),
		Title:       item.GetFinalTitle(),
		Description: item.GetFinalDescription(),
		Tags:        allTags,
		AddDate:     time.Now(),
	}
}

// ProcessItems processes multiple import items concurrently, discovering URLs and generating bookmarks
func ProcessItems(items []*ImportItem, httpClient *feeds.HTTPClient, options ProcessOptions, concurrency int) (*ImportStats, []netscape.Bookmark) {
	logrus.WithFields(logrus.Fields{
		"total_items": len(items),
		"concurrency": concurrency,
		"dry_run":     options.DryRun,
		"global_tags": options.Tags,
	}).Info("Starting concurrent processing of import items")

	stats := NewImportStats(len(items))
	bookmarks := make([]netscape.Bookmark, 0, len(items))
	bookmarksMutex := sync.Mutex{}

	// Create channels for work distribution
	workQueue := make(chan *ImportItem, len(items))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			logrus.WithField("worker_id", workerID).Debug("Worker started")

			for item := range workQueue {
				// Step 1: URL Discovery with retry
				logrus.WithFields(logrus.Fields{
					"worker_id": workerID,
					"title":     item.Title,
					"xml_url":   item.XMLURL,
				}).Debug("Processing item in worker")

				err := retryOperation(func() error {
					return DiscoverBookmarkURL(item, httpClient)
				}, options.RetryAttempts, "URL discovery", item)

				if err != nil {
					logrus.WithFields(logrus.Fields{
						"worker_id": workerID,
						"title":     item.Title,
						"error":     err.Error(),
					}).Error("Failed URL discovery after retries")

					item.Status = StatusFailed
					item.Error = err
					stats.IncrementFailed()
					continue
				}

				// Check if we found a valid URL
				if item.GetFinalURL() == "" {
					logrus.WithFields(logrus.Fields{
						"worker_id": workerID,
						"title":     item.Title,
					}).Warn("No valid URL found for item")

					item.Status = StatusFailed
					item.Error = fmt.Errorf("no valid URL found")
					stats.IncrementFailed()
					continue
				}

				// Step 2: Convert to bookmark
				if !options.DryRun {
					bookmark := ConvertToBookmark(item, options)

					bookmarksMutex.Lock()
					bookmarks = append(bookmarks, bookmark)
					bookmarksMutex.Unlock()

					logrus.WithFields(logrus.Fields{
						"worker_id": workerID,
						"url":       bookmark.URL,
						"title":     bookmark.Title,
						"tags":      bookmark.Tags,
					}).Info("Converted item to bookmark")
				} else {
					logrus.WithFields(logrus.Fields{
						"worker_id": workerID,
						"url":       item.GetFinalURL(),
						"title":     item.GetFinalTitle(),
					}).Info("Would create bookmark (dry run)")
				}

				item.Status = StatusSuccess
				stats.IncrementImported()
			}

			logrus.WithField("worker_id", workerID).Debug("Worker finished")
		}(i)
	}

	// Feed work to workers
	for _, item := range items {
		workQueue <- item
	}
	close(workQueue)

	// Wait for all workers to complete
	wg.Wait()

	logrus.WithFields(logrus.Fields{
		"total_processed": len(items),
		"successful":      stats.Imported,
		"failed":          stats.Failed,
		"bookmarks":       len(bookmarks),
	}).Info("Finished processing import items")

	return stats, bookmarks
}

// retryOperation retries an operation with exponential backoff
func retryOperation(operation func() error, maxAttempts int, operationName string, item *ImportItem) error {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		logrus.WithFields(logrus.Fields{
			"title":        item.Title,
			"url":          item.GetFinalURL(),
			"operation":    operationName,
			"attempt":      attempt,
			"max_attempts": maxAttempts,
		}).Debug("Attempting operation")

		err := operation()
		if err == nil {
			if attempt > 1 {
				logrus.WithFields(logrus.Fields{
					"title":     item.Title,
					"url":       item.GetFinalURL(),
					"operation": operationName,
					"attempt":   attempt,
				}).Info("Operation succeeded after retry")
			}
			return nil
		}

		lastErr = err

		if attempt < maxAttempts {
			// Calculate exponential backoff delay
			backoffDelay := time.Duration(attempt) * time.Second

			logrus.WithFields(logrus.Fields{
				"title":      item.Title,
				"url":        item.GetFinalURL(),
				"operation":  operationName,
				"attempt":    attempt,
				"error":      err.Error(),
				"retry_in":   backoffDelay,
			}).Warn("Operation failed, retrying")

			time.Sleep(backoffDelay)
		} else {
			logrus.WithFields(logrus.Fields{
				"title":        item.Title,
				"url":          item.GetFinalURL(),
				"operation":    operationName,
				"attempt":      attempt,
				"max_attempts": maxAttempts,
				"error":        err.Error(),
			}).Error("Operation failed after all retry attempts")
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxAttempts, lastErr)
}
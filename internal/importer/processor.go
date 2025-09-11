package importer

import (
	"fmt"
	"sync"
	"time"

	"linkding-to-opml/internal/feeds"
	"linkding-to-opml/internal/linkding"

	"github.com/sirupsen/logrus"
)

// ProcessOptions contains options for processing bookmarks
type ProcessOptions struct {
	DuplicateAction string   // "skip" or "update"
	Tags            []string // Tags to apply to all bookmarks
	DryRun          bool     // If true, don't actually create/update bookmarks
	RetryAttempts   int      // Number of retry attempts for failed operations
}

// ProcessBookmark processes a single import item, handling duplicates and creating/updating bookmarks
func ProcessBookmark(item *ImportItem, client *linkding.Client, options ProcessOptions) error {
	logrus.WithFields(logrus.Fields{
		"title":      item.GetFinalTitle(),
		"url":        item.GetFinalURL(),
		"dry_run":    options.DryRun,
		"duplicates": options.DuplicateAction,
	}).Debug("Processing bookmark")

	// Validate the final URL
	finalURL := item.GetFinalURL()
	if finalURL == "" {
		return fmt.Errorf("no valid URL found for bookmark")
	}

	// Check if bookmark already exists (skip in dry-run mode or if client is nil)
	var existing *linkding.Bookmark
	var err error
	
	if !options.DryRun && client != nil {
		existing, err = client.GetBookmarkByURL(finalURL)
		if err != nil {
			return fmt.Errorf("failed to check for existing bookmark: %w", err)
		}
	} else {
		// In dry-run mode, simulate no existing bookmarks
		existing = nil
	}

	// Prepare the tags (combine item-specific tags with global tags)
	allTags := make([]string, len(options.Tags))
	copy(allTags, options.Tags)

	if existing != nil {
		// Bookmark already exists - handle according to duplicate action
		logrus.WithFields(logrus.Fields{
			"existing_id":    existing.ID,
			"existing_title": existing.Title,
			"new_title":      item.GetFinalTitle(),
			"action":         options.DuplicateAction,
		}).Debug("Found existing bookmark")

		if options.DuplicateAction == "skip" {
			logrus.WithFields(logrus.Fields{
				"url":   finalURL,
				"title": item.GetFinalTitle(),
			}).Info("Skipping duplicate bookmark")
			
			item.Status = StatusSkipped
			return nil
		}

		if options.DuplicateAction == "update" {
			if options.DryRun {
				logrus.WithFields(logrus.Fields{
					"id":    existing.ID,
					"url":   finalURL,
					"title": item.GetFinalTitle(),
				}).Info("Would update existing bookmark (dry run)")
			} else {
				if client != nil {
					err := client.UpdateBookmark(
						existing.ID,
						finalURL,
						item.GetFinalTitle(),
						item.GetFinalDescription(),
						allTags,
					)
					if err != nil {
						item.Status = StatusFailed
						item.Error = err
						return fmt.Errorf("failed to update bookmark: %w", err)
					}

					logrus.WithFields(logrus.Fields{
						"id":    existing.ID,
						"url":   finalURL,
						"title": item.GetFinalTitle(),
					}).Info("Updated existing bookmark")
				} else {
					logrus.WithFields(logrus.Fields{
						"url":   finalURL,
						"title": item.GetFinalTitle(),
					}).Info("Would update existing bookmark (no client provided)")
				}
			}

			item.Status = StatusSuccess
			item.WasUpdated = true // Flag to distinguish updates from new imports
			return nil
		}

		return fmt.Errorf("unknown duplicate action: %s", options.DuplicateAction)
	}

	// Bookmark doesn't exist - create new one
	if options.DryRun {
		logrus.WithFields(logrus.Fields{
			"url":   finalURL,
			"title": item.GetFinalTitle(),
			"tags":  allTags,
		}).Info("Would create new bookmark (dry run)")
	} else {
		if client != nil {
			_, err := client.CreateBookmark(
				finalURL,
				item.GetFinalTitle(),
				item.GetFinalDescription(),
				allTags,
			)
			if err != nil {
				item.Status = StatusFailed
				item.Error = err
				return fmt.Errorf("failed to create bookmark: %w", err)
			}

			logrus.WithFields(logrus.Fields{
				"url":   finalURL,
				"title": item.GetFinalTitle(),
				"tags":  allTags,
			}).Info("Created new bookmark")
		} else {
			logrus.WithFields(logrus.Fields{
				"url":   finalURL,
				"title": item.GetFinalTitle(),
				"tags":  allTags,
			}).Info("Would create new bookmark (no client provided)")
		}
	}

	item.Status = StatusSuccess
	return nil
}

// retryOperation executes an operation with retry logic and exponential backoff
func retryOperation(operation func() error, maxAttempts int, operationName string, item *ImportItem) error {
	var lastErr error
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := operation()
		if err == nil {
			// Success
			if attempt > 1 {
				logrus.WithFields(logrus.Fields{
					"title":     item.GetFinalTitle(),
					"url":       item.GetFinalURL(),
					"operation": operationName,
					"attempt":   attempt,
					"max_attempts": maxAttempts,
				}).Info("Operation succeeded after retry")
			}
			return nil
		}
		
		lastErr = err
		
		if attempt < maxAttempts {
			// Calculate exponential backoff: 1s, 2s, 4s, 8s, etc.
			backoffDuration := time.Duration(1<<uint(attempt-1)) * time.Second
			
			logrus.WithFields(logrus.Fields{
				"title":     item.GetFinalTitle(),
				"url":       item.GetFinalURL(),
				"operation": operationName,
				"attempt":   attempt,
				"max_attempts": maxAttempts,
				"error":     err.Error(),
				"backoff":   backoffDuration,
			}).Warn("Operation failed, retrying after backoff")
			
			time.Sleep(backoffDuration)
		} else {
			logrus.WithFields(logrus.Fields{
				"title":     item.GetFinalTitle(),
				"url":       item.GetFinalURL(),
				"operation": operationName,
				"attempt":   attempt,
				"max_attempts": maxAttempts,
				"error":     err.Error(),
			}).Error("Operation failed after all retry attempts")
		}
	}
	
	return fmt.Errorf("operation failed after %d attempts: %w", maxAttempts, lastErr)
}

// ProcessItems processes multiple import items concurrently using a worker pool
func ProcessItems(items []*ImportItem, httpClient *feeds.HTTPClient, linkdingClient *linkding.Client, options ProcessOptions, concurrency int) *ImportStats {
	logrus.WithFields(logrus.Fields{
		"total_items":  len(items),
		"concurrency":  concurrency,
		"dry_run":      options.DryRun,
		"duplicates":   options.DuplicateAction,
		"global_tags":  options.Tags,
	}).Info("Starting concurrent processing of import items")

	stats := NewImportStats(len(items))
	
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
					}).Error("URL discovery failed after all retries")
					
					item.Status = StatusFailed
					item.Error = err
					stats.IncrementFailed()
					stats.IncrementProcessed()
					continue
				}
				
				// Step 2: Process bookmark (create/update in Linkding) with retry
				err = retryOperation(func() error {
					return ProcessBookmark(item, linkdingClient, options)
				}, options.RetryAttempts, "bookmark processing", item)
				
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"worker_id": workerID,
						"title":     item.GetFinalTitle(),
						"url":       item.GetFinalURL(),
						"error":     err.Error(),
					}).Error("Bookmark processing failed after all retries")
					
					item.Status = StatusFailed
					item.Error = err
					stats.IncrementFailed()
				} else {
					// Update stats based on item status
					switch item.Status {
					case StatusSuccess:
						if item.WasUpdated {
							stats.IncrementUpdated()
						} else {
							stats.IncrementImported()
						}
					case StatusSkipped:
						stats.IncrementSkipped()
					default:
						// This shouldn't happen if ProcessBookmark works correctly
						logrus.WithFields(logrus.Fields{
							"worker_id": workerID,
							"status":    item.Status,
						}).Warn("Unexpected item status after processing")
					}
				}
				
				stats.IncrementProcessed()
				
				logrus.WithFields(logrus.Fields{
					"worker_id":    workerID,
					"title":        item.GetFinalTitle(),
					"final_url":    item.GetFinalURL(),
					"status":       item.Status,
					"processed":    stats.Processed,
					"total":        stats.Total,
				}).Debug("Completed processing item")
			}
			
			logrus.WithField("worker_id", workerID).Debug("Worker finished")
		}(i)
	}
	
	// Send all items to work queue
	for _, item := range items {
		workQueue <- item
	}
	close(workQueue)
	
	// Wait for all workers to complete
	wg.Wait()
	
	stats.Finish()
	
	logrus.WithFields(logrus.Fields{
		"total":     stats.Total,
		"processed": stats.Processed,
		"imported":  stats.Imported,
		"updated":   stats.Updated,
		"skipped":   stats.Skipped,
		"failed":    stats.Failed,
		"duration":  stats.Duration(),
	}).Info("Completed concurrent processing")
	
	return stats
}
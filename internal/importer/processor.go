package importer

import (
	"fmt"

	"linkding-to-opml/internal/linkding"

	"github.com/sirupsen/logrus"
)

// ProcessOptions contains options for processing bookmarks
type ProcessOptions struct {
	DuplicateAction string   // "skip" or "update"
	Tags            []string // Tags to apply to all bookmarks
	DryRun          bool     // If true, don't actually create/update bookmarks
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

	// Check if bookmark already exists
	existing, err := client.GetBookmarkByURL(finalURL)
	if err != nil {
		return fmt.Errorf("failed to check for existing bookmark: %w", err)
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
			}

			item.Status = StatusSuccess
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
	}

	item.Status = StatusSuccess
	return nil
}
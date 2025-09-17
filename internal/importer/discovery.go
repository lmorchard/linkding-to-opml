package importer

import (
	"fmt"
	"strings"

	"linkding-to-opml/internal/feeds"

	"github.com/sirupsen/logrus"
)

// DiscoverBookmarkURL implements the three-tier fallback system for discovering bookmark URLs
func DiscoverBookmarkURL(item *ImportItem, httpClient *feeds.HTTPClient) error {
	logrus.WithFields(logrus.Fields{
		"title":    item.Title,
		"xml_url":  item.XMLURL,
		"html_url": item.HTMLURL,
	}).Debug("Starting URL discovery")

	// Tier 1: Use htmlUrl from OPML if available
	if item.HTMLURL != "" && item.HTMLURL != item.XMLURL {
		logrus.WithFields(logrus.Fields{
			"title":    item.Title,
			"html_url": item.HTMLURL,
		}).Debug("Using htmlUrl from OPML")
		
		item.UpdateWithDiscoveredData(item.HTMLURL, item.Title, item.Description)
		return nil
	}

	// Tier 2: Fetch feed and extract website link
	if item.XMLURL != "" {
		logrus.WithFields(logrus.Fields{
			"title":   item.Title,
			"xml_url": item.XMLURL,
		}).Debug("Fetching feed to discover website link")

		feed, err := feeds.FetchFeed(item.XMLURL, httpClient)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"title":   item.Title,
				"xml_url": item.XMLURL,
				"error":   err.Error(),
			}).Debug("Failed to fetch feed, will fallback to feed URL")
		} else if feed.Link != "" {
			logrus.WithFields(logrus.Fields{
				"title":        item.Title,
				"website_link": feed.Link,
				"feed_type":    feed.FeedType,
			}).Debug("Discovered website link from feed")
			
			// Use feed metadata if available
			title := item.Title
			if feed.Title != "" {
				title = feed.Title
			}
			
			description := item.Description
			if feed.Description != "" {
				description = feed.Description
			}
			
			item.UpdateWithDiscoveredData(feed.Link, title, description)
			return nil
		}
	}

	// Tier 3: Fallback to feed URL itself
	logrus.WithFields(logrus.Fields{
		"title":   item.Title,
		"xml_url": item.XMLURL,
	}).Info("No website link found, falling back to feed URL")

	if item.XMLURL == "" {
		return fmt.Errorf("no URL available for bookmark")
	}

	item.UpdateWithDiscoveredData(item.XMLURL, item.Title, item.Description)
	return nil
}

// ValidateURL performs basic URL validation
func ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL is empty")
	}
	
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}
	
	return nil
}
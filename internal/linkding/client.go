package linkding

import (
	"fmt"
	"strings"
	"time"

	"github.com/piero-vic/go-linkding"
	"github.com/sirupsen/logrus"
)

// Bookmark represents a bookmark from Linkding
type Bookmark struct {
	URL   string   `json:"url"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

// Client wraps the go-linkding client with additional functionality
type Client struct {
	client  *linkding.Client
	timeout time.Duration
}

// NewClient creates a new Linkding API client
func NewClient(token, url string, timeout time.Duration) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("linkding token cannot be empty")
	}
	if url == "" {
		return nil, fmt.Errorf("linkding URL cannot be empty")
	}

	client := linkding.NewClient(url, token)

	logrus.WithFields(logrus.Fields{
		"url":     url,
		"timeout": timeout,
	}).Debug("Created Linkding API client")

	return &Client{
		client:  client,
		timeout: timeout,
	}, nil
}

// FetchBookmarks fetches bookmarks from Linkding, optionally filtered by tags
func (c *Client) FetchBookmarks(tags []string) ([]*Bookmark, error) {
	logrus.WithField("tags", tags).Info("Fetching bookmarks from Linkding API")

	// Use linkding client to fetch bookmarks
	// For now, we'll get all bookmarks and filter client-side
	// The go-linkding library may support server-side filtering in the future
	bookmarkList, err := c.client.ListBookmarks(linkding.ListBookmarksParams{
		Limit:  1000, // Get lots of bookmarks (adjust as needed)
		Offset: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bookmarks from Linkding: %w", err)
	}

	var filteredBookmarks []*Bookmark

	// Convert and filter bookmarks
	for _, bookmark := range bookmarkList.Results {
		// Convert linkding bookmark to our internal format
		bookmarkTags := make([]string, len(bookmark.TagNames))
		copy(bookmarkTags, bookmark.TagNames)

		internalBookmark := &Bookmark{
			URL:   bookmark.URL,
			Title: bookmark.Title,
			Tags:  bookmarkTags,
		}

		// Apply tag filtering if tags are specified
		if len(tags) > 0 {
			if c.matchesTags(internalBookmark, tags) {
				filteredBookmarks = append(filteredBookmarks, internalBookmark)
			}
		} else {
			// No filtering - include all bookmarks
			filteredBookmarks = append(filteredBookmarks, internalBookmark)
		}
	}

	logrus.WithFields(logrus.Fields{
		"total_fetched": len(bookmarkList.Results),
		"after_filter":  len(filteredBookmarks),
		"filter_tags":   tags,
	}).Info("Successfully fetched and filtered bookmarks")

	return filteredBookmarks, nil
}

// matchesTags checks if a bookmark has ALL the specified tags (AND operation)
func (c *Client) matchesTags(bookmark *Bookmark, requiredTags []string) bool {
	if len(requiredTags) == 0 {
		return true // No filter tags means match all
	}

	// Convert bookmark tags to a map for faster lookup
	bookmarkTags := make(map[string]bool)
	for _, tag := range bookmark.Tags {
		bookmarkTags[strings.ToLower(tag)] = true
	}

	// Check if bookmark has ALL required tags (AND operation)
	for _, requiredTag := range requiredTags {
		if !bookmarkTags[strings.ToLower(requiredTag)] {
			logrus.WithFields(logrus.Fields{
				"url":          bookmark.URL,
				"bookmark_tags": bookmark.Tags,
				"required_tags": requiredTags,
				"missing_tag":   requiredTag,
			}).Debug("Bookmark does not match tag filter")
			return false
		}
	}

	logrus.WithFields(logrus.Fields{
		"url":          bookmark.URL,
		"bookmark_tags": bookmark.Tags,
		"required_tags": requiredTags,
	}).Debug("Bookmark matches tag filter")

	return true
}
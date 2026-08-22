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
	ID    int      `json:"id"`
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

// pageSize is the number of bookmarks requested per API call. Linkding pages
// every list response, so the full set must be walked one page at a time.
const pageSize = 500

// FetchBookmarks fetches bookmarks from Linkding, optionally filtered by tags.
//
// Linkding never returns the whole collection in a single response, so this
// pages until every record has been read. Tag filtering is pushed to the server
// via the search query (which matches tags exactly and ANDs multiple tags), and
// the results are re-checked client-side by matchesTags.
func (c *Client) FetchBookmarks(tags []string) ([]*Bookmark, error) {
	logrus.WithField("tags", tags).Info("Fetching bookmarks from Linkding API")

	query := buildTagQuery(tags)

	var filteredBookmarks []*Bookmark
	totalFetched := 0
	offset := 0

	for {
		bookmarkList, err := c.client.ListBookmarks(linkding.ListBookmarksParams{
			Query:  query,
			Limit:  pageSize,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch bookmarks from Linkding: %w", err)
		}

		// An empty page means the server has nothing further to give, even if
		// its reported count disagrees. Without this the loop could not end.
		if len(bookmarkList.Results) == 0 {
			break
		}

		totalFetched += len(bookmarkList.Results)

		// Convert and filter bookmarks
		for _, bookmark := range bookmarkList.Results {
			// Convert linkding bookmark to our internal format
			bookmarkTags := make([]string, len(bookmark.TagNames))
			copy(bookmarkTags, bookmark.TagNames)

			internalBookmark := &Bookmark{
				ID:    bookmark.ID,
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

		offset += len(bookmarkList.Results)
		if offset >= bookmarkList.Count {
			break
		}

		logrus.WithFields(logrus.Fields{
			"fetched": offset,
			"total":   bookmarkList.Count,
		}).Debug("Fetched a page of bookmarks, continuing")
	}

	logrus.WithFields(logrus.Fields{
		"total_fetched": totalFetched,
		"after_filter":  len(filteredBookmarks),
		"filter_tags":   tags,
	}).Info("Successfully fetched and filtered bookmarks")

	return filteredBookmarks, nil
}

// buildTagQuery renders tags into a Linkding search query. Linkding treats a
// "#tag" token as an exact tag match and ANDs multiple tokens together, which
// matches the semantics of matchesTags. Tags containing whitespace cannot be
// expressed as a query token, so they are left to client-side filtering.
func buildTagQuery(tags []string) string {
	var tokens []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || strings.ContainsAny(tag, " \t") {
			continue
		}
		tokens = append(tokens, "#"+tag)
	}
	return strings.Join(tokens, " ")
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
				"url":           bookmark.URL,
				"bookmark_tags": bookmark.Tags,
				"required_tags": requiredTags,
				"missing_tag":   requiredTag,
			}).Debug("Bookmark does not match tag filter")
			return false
		}
	}

	logrus.WithFields(logrus.Fields{
		"url":           bookmark.URL,
		"bookmark_tags": bookmark.Tags,
		"required_tags": requiredTags,
	}).Debug("Bookmark matches tag filter")

	return true
}

// CreateBookmark creates a new bookmark in Linkding
func (c *Client) CreateBookmark(url, title, description string, tags []string) (*Bookmark, error) {
	logrus.WithFields(logrus.Fields{
		"url":   url,
		"title": title,
		"tags":  tags,
	}).Debug("Creating bookmark in Linkding")

	request := linkding.CreateBookmarkRequest{
		URL:         url,
		Title:       title,
		Description: description,
		TagNames:    tags,
		IsArchived:  false,
		Unread:      false,
		Shared:      false,
	}

	bookmark, err := c.client.CreateBookmark(request)
	if err != nil {
		return nil, fmt.Errorf("failed to create bookmark: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"id":    bookmark.ID,
		"url":   bookmark.URL,
		"title": bookmark.Title,
	}).Info("Successfully created bookmark")

	// Convert to internal format
	return &Bookmark{
		ID:    bookmark.ID,
		URL:   bookmark.URL,
		Title: bookmark.Title,
		Tags:  bookmark.TagNames,
	}, nil
}

// GetBookmarkByURL checks if a bookmark with the given URL already exists
func (c *Client) GetBookmarkByURL(url string) (*Bookmark, error) {
	logrus.WithField("url", url).Debug("Checking if bookmark exists by URL")

	response, err := c.client.CheckBookmark(url)
	if err != nil {
		return nil, fmt.Errorf("failed to check bookmark: %w", err)
	}

	// CheckBookmark returns nil bookmark if not found
	if response.Bookmark == nil {
		logrus.WithField("url", url).Debug("Bookmark not found")
		return nil, nil
	}

	logrus.WithFields(logrus.Fields{
		"id":    response.Bookmark.ID,
		"url":   response.Bookmark.URL,
		"title": response.Bookmark.Title,
	}).Debug("Found existing bookmark")

	// Convert to internal format
	return &Bookmark{
		ID:    response.Bookmark.ID,
		URL:   response.Bookmark.URL,
		Title: response.Bookmark.Title,
		Tags:  response.Bookmark.TagNames,
	}, nil
}

// UpdateBookmark updates an existing bookmark in Linkding
func (c *Client) UpdateBookmark(id int, url, title, description string, tags []string) error {
	logrus.WithFields(logrus.Fields{
		"id":    id,
		"url":   url,
		"title": title,
		"tags":  tags,
	}).Debug("Updating bookmark in Linkding")

	request := linkding.CreateBookmarkRequest{
		URL:         url,
		Title:       title,
		Description: description,
		TagNames:    tags,
		IsArchived:  false,
		Unread:      false,
		Shared:      false,
	}

	_, err := c.client.UpdateBookmark(id, request)
	if err != nil {
		return fmt.Errorf("failed to update bookmark: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"id":    id,
		"url":   url,
		"title": title,
	}).Info("Successfully updated bookmark")

	return nil
}

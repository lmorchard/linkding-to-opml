package feeds

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// Feed represents a parsed RSS or Atom feed with website link
type Feed struct {
	Title       string
	Description string
	Link        string // Website link
	FeedType    string
}

// RSSFeed represents an RSS feed structure with website link
type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel RSSChannel `xml:"channel"`
}

// RSSChannel represents an RSS channel with website link
type RSSChannel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
}

// AtomFeed represents an Atom feed structure with website link
type AtomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Summary string      `xml:"subtitle"`
	Links   []AtomLink  `xml:"link"`
}

// AtomLink represents a link in an Atom feed
type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

// FetchFeed fetches and parses a feed from the given URL
func FetchFeed(feedURL string, httpClient *HTTPClient) (*Feed, error) {
	logrus.WithField("feed_url", feedURL).Debug("Fetching feed")

	// Use the existing HTTP client to fetch the feed content
	// We'll use a default user agent for feed fetching
	userAgent := "linkding-to-opml/1.0 (Feed Fetcher)"
	content, err := httpClient.FetchPage(feedURL, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}

	// Try to parse as RSS first
	feed, err := parseRSSFeed(content)
	if err == nil {
		feed.FeedType = "RSS"
		logrus.WithFields(logrus.Fields{
			"feed_url":   feedURL,
			"feed_type":  "RSS",
			"title":      feed.Title,
			"link":       feed.Link,
		}).Debug("Successfully parsed RSS feed")
		return feed, nil
	}

	// Try to parse as Atom
	feed, err = parseAtomFeed(content)
	if err == nil {
		feed.FeedType = "Atom"
		logrus.WithFields(logrus.Fields{
			"feed_url":   feedURL,
			"feed_type":  "Atom", 
			"title":      feed.Title,
			"link":       feed.Link,
		}).Debug("Successfully parsed Atom feed")
		return feed, nil
	}

	return nil, fmt.Errorf("failed to parse feed as RSS or Atom")
}

// parseRSSFeed parses RSS feed content and extracts metadata
func parseRSSFeed(content string) (*Feed, error) {
	var rss RSSFeed
	if err := xml.Unmarshal([]byte(content), &rss); err != nil {
		return nil, fmt.Errorf("failed to parse RSS: %w", err)
	}

	feed := &Feed{
		Title:       rss.Channel.Title,
		Description: rss.Channel.Description,
		Link:        rss.Channel.Link,
	}

	// Clean up link - remove trailing slashes and whitespace
	feed.Link = strings.TrimSpace(strings.TrimSuffix(feed.Link, "/"))

	return feed, nil
}

// parseAtomFeed parses Atom feed content and extracts metadata
func parseAtomFeed(content string) (*Feed, error) {
	var atom AtomFeed
	if err := xml.Unmarshal([]byte(content), &atom); err != nil {
		return nil, fmt.Errorf("failed to parse Atom: %w", err)
	}

	feed := &Feed{
		Title:       atom.Title,
		Description: atom.Summary,
	}

	// Find the website link (rel="alternate" with type="text/html")
	for _, link := range atom.Links {
		if link.Rel == "alternate" && strings.Contains(link.Type, "text/html") {
			feed.Link = link.Href
			break
		}
	}

	// If no alternate link found, try to find any link without rel="self"
	if feed.Link == "" {
		for _, link := range atom.Links {
			if link.Rel != "self" {
				feed.Link = link.Href
				break
			}
		}
	}

	// Clean up link - remove trailing slashes and whitespace
	feed.Link = strings.TrimSpace(strings.TrimSuffix(feed.Link, "/"))

	return feed, nil
}
package feeds

import (
	"encoding/xml"
	"fmt"
	"io"
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
	Title       string   `xml:"title"`
	Description string   `xml:"description"`
	Link        string   `xml:"link"`
	Image       RSSImage `xml:"image"`
}

// RSSImage represents an RSS image element
type RSSImage struct {
	Title string `xml:"title"`
	URL   string `xml:"url"`
	Link  string `xml:"link"`
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

// RDFFeed represents an RSS 1.0/RDF feed structure
type RDFFeed struct {
	XMLName     xml.Name   `xml:"RDF"`
	Channel     RDFChannel `xml:"channel"`
	Title       string     `xml:"title"`
	Link        string     `xml:"link"`
	Description string     `xml:"description"`
}

// RDFChannel represents the channel element in RSS 1.0/RDF
type RDFChannel struct {
	XMLName     xml.Name `xml:"channel"`
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
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

	// Try to parse as RSS 1.0/RDF
	feed, err = parseRDFFeed(content)
	if err == nil {
		feed.FeedType = "RDF"
		logrus.WithFields(logrus.Fields{
			"feed_url":   feedURL,
			"feed_type":  "RDF",
			"title":      feed.Title,
			"link":       feed.Link,
		}).Debug("Successfully parsed RDF feed")
		return feed, nil
	}

	return nil, fmt.Errorf("failed to parse feed as RSS, Atom, or RDF")
}

// parseRSSFeed parses RSS feed content and extracts metadata
func parseRSSFeed(content string) (*Feed, error) {
	// Create decoder with charset support for feeds with non-UTF-8 encoding
	decoder := xml.NewDecoder(strings.NewReader(content))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		// For simplicity, just return the input reader
		// Go's XML parser will handle most common encodings automatically
		return input, nil
	}
	
	var rss RSSFeed
	if err := decoder.Decode(&rss); err != nil {
		return nil, fmt.Errorf("failed to parse RSS: %w", err)
	}

	feed := &Feed{
		Title:       rss.Channel.Title,
		Description: rss.Channel.Description,
		Link:        rss.Channel.Link,
	}

	// If channel link is empty, try to find the first link manually
	// This handles cases where XML unmarshaling fails due to multiple link elements
	if feed.Link == "" {
		decoder := xml.NewDecoder(strings.NewReader(content))
		inChannel := false
		depth := 0
		
		for {
			token, err := decoder.Token()
			if err != nil {
				break
			}
			
			if se, ok := token.(xml.StartElement); ok {
				switch se.Name.Local {
				case "channel":
					inChannel = true
					depth = 1
				case "image", "item":
					if inChannel {
						// Skip entire image and item elements to avoid their links
						decoder.Skip()
						continue
					}
				case "link":
					// Only process links that are direct children of channel (depth 1)
					if inChannel && depth == 1 {
						var linkText string
						if err := decoder.DecodeElement(&linkText, &se); err == nil && linkText != "" {
							feed.Link = linkText
							break
						}
					}
				default:
					if inChannel {
						depth++
					}
				}
			} else if ee, ok := token.(xml.EndElement); ok {
				if ee.Name.Local == "channel" {
					inChannel = false
					depth = 0
				} else if inChannel {
					depth--
				}
			}
		}
	}

	// Clean up link - remove trailing slashes and whitespace
	feed.Link = strings.TrimSpace(strings.TrimSuffix(feed.Link, "/"))

	return feed, nil
}

// parseAtomFeed parses Atom feed content and extracts metadata
func parseAtomFeed(content string) (*Feed, error) {
	// Create decoder with charset support and entity handling
	decoder := xml.NewDecoder(strings.NewReader(content))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}
	
	var atom AtomFeed
	if err := decoder.Decode(&atom); err != nil {
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

// parseRDFFeed parses RSS 1.0/RDF feed content and extracts metadata
func parseRDFFeed(content string) (*Feed, error) {
	var rdf RDFFeed
	if err := xml.Unmarshal([]byte(content), &rdf); err != nil {
		return nil, fmt.Errorf("failed to parse RDF: %w", err)
	}

	feed := &Feed{
		Title:       rdf.Channel.Title,
		Description: rdf.Channel.Description,
		Link:        rdf.Channel.Link,
	}

	// If channel link is empty, try the top-level link
	if feed.Link == "" {
		feed.Link = rdf.Link
	}

	// Clean up link - remove trailing slashes and whitespace
	feed.Link = strings.TrimSpace(strings.TrimSuffix(feed.Link, "/"))

	return feed, nil
}
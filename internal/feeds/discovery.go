package feeds

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

// FeedDiscoveryResult represents the result of attempting to discover a feed from a URL
type FeedDiscoveryResult struct {
	URL       string `json:"url"`        // Original bookmark URL
	FeedURL   string `json:"feed_url"`   // Discovered feed URL
	FeedTitle string `json:"feed_title"` // Feed title from feed metadata
	Error     error  `json:"error"`      // Error if discovery failed
}

// RSS represents a simplified RSS feed structure for title extraction
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

// Atom represents a simplified Atom feed structure for title extraction
type Atom struct {
	XMLName xml.Name `xml:"feed"`
	Title   string   `xml:"title"`
}

// Channel represents an RSS channel
type Channel struct {
	Title string `xml:"title"`
}

// DiscoverFeed attempts to discover and validate an RSS/Atom feed from a given URL
func DiscoverFeed(pageURL string, httpClient *HTTPClient, userAgent string) *FeedDiscoveryResult {
	result := &FeedDiscoveryResult{
		URL: pageURL,
	}

	logrus.WithField("url", pageURL).Debug("Starting feed autodiscovery")

	// Step 1: Fetch the webpage
	pageContent, err := httpClient.FetchPage(pageURL, userAgent)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch page: %w", err)
		logrus.WithFields(logrus.Fields{
			"url":   pageURL,
			"error": err,
		}).Debug("Feed discovery failed: could not fetch page")
		return result
	}

	// Step 2: Parse HTML and find feed links
	feedURLs := findFeedLinks(pageContent, pageURL)
	if len(feedURLs) == 0 {
		result.Error = fmt.Errorf("no feed links found in page")
		logrus.WithField("url", pageURL).Debug("Feed discovery failed: no feed links found")
		return result
	}

	// Step 3: Try the first feed URL found (as per spec)
	feedURL := feedURLs[0]
	logrus.WithFields(logrus.Fields{
		"page_url": pageURL,
		"feed_url": feedURL,
		"total_found": len(feedURLs),
	}).Debug("Found feed links, trying first one")

	// Step 4: Fetch and validate the feed
	feedContent, err := httpClient.FetchPage(feedURL, userAgent)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch feed: %w", err)
		logrus.WithFields(logrus.Fields{
			"page_url": pageURL,
			"feed_url": feedURL,
			"error":    err,
		}).Debug("Feed discovery failed: could not fetch feed")
		return result
	}

	// Step 5: Parse feed and extract title
	feedTitle, err := extractFeedTitle(feedContent)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse feed: %w", err)
		logrus.WithFields(logrus.Fields{
			"page_url": pageURL,
			"feed_url": feedURL,
			"error":    err,
		}).Debug("Feed discovery failed: could not parse feed")
		return result
	}

	// Success!
	result.FeedURL = feedURL
	result.FeedTitle = feedTitle
	
	logrus.WithFields(logrus.Fields{
		"page_url":   pageURL,
		"feed_url":   feedURL,
		"feed_title": feedTitle,
	}).Debug("Feed discovery successful")

	return result
}

// findFeedLinks parses HTML content and extracts RSS/Atom feed URLs using autodiscovery
func findFeedLinks(htmlContent, baseURL string) []string {
	var feedURLs []string

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		logrus.WithError(err).Debug("Failed to parse HTML, falling back to regex")
		return findFeedLinksRegex(htmlContent, baseURL)
	}

	// Walk the HTML tree looking for link elements
	var walkNode func(*html.Node)
	walkNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, typ, href string
			
			// Extract attributes
			for _, attr := range n.Attr {
				switch attr.Key {
				case "rel":
					rel = attr.Val
				case "type":
					typ = attr.Val
				case "href":
					href = attr.Val
				}
			}

			// Check if this is a feed link
			if strings.Contains(strings.ToLower(rel), "alternate") {
				if strings.Contains(typ, "application/rss+xml") || 
				   strings.Contains(typ, "application/atom+xml") ||
				   strings.Contains(typ, "application/rdf+xml") {
					
					// Convert relative URLs to absolute
					if feedURL := resolveURL(href, baseURL); feedURL != "" {
						feedURLs = append(feedURLs, feedURL)
						logrus.WithFields(logrus.Fields{
							"base_url": baseURL,
							"rel":      rel,
							"type":     typ,
							"href":     href,
							"resolved": feedURL,
						}).Debug("Found feed link")
					}
				}
			}
		}

		// Recursively walk child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkNode(c)
		}
	}

	walkNode(doc)

	// If we didn't find any feeds with HTML parsing, try regex as fallback
	if len(feedURLs) == 0 {
		logrus.WithField("base_url", baseURL).Debug("No feeds found with HTML parser, trying regex fallback")
		feedURLs = findFeedLinksRegex(htmlContent, baseURL)
	}

	return feedURLs
}

// findFeedLinksRegex is a fallback method using regex to find feed links
func findFeedLinksRegex(htmlContent, baseURL string) []string {
	var feedURLs []string

	// Regex to find feed links (simplified version)
	linkRegex := regexp.MustCompile(`(?i)<link[^>]+rel[^>]*alternate[^>]+type[^>]*application/(rss|atom)\+xml[^>]+href[^>]*=["']([^"']+)["'][^>]*>`)
	matches := linkRegex.FindAllStringSubmatch(htmlContent, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			href := match[2]
			if feedURL := resolveURL(href, baseURL); feedURL != "" {
				feedURLs = append(feedURLs, feedURL)
				logrus.WithFields(logrus.Fields{
					"base_url": baseURL,
					"href":     href,
					"resolved": feedURL,
				}).Debug("Found feed link with regex")
			}
		}
	}

	return feedURLs
}

// resolveURL converts relative URLs to absolute URLs
func resolveURL(href, baseURL string) string {
	if href == "" {
		return ""
	}

	// Parse base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"base_url": baseURL,
			"error":    err,
		}).Debug("Failed to parse base URL")
		return ""
	}

	// Parse href URL
	ref, err := url.Parse(href)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"href":  href,
			"error": err,
		}).Debug("Failed to parse href URL")
		return ""
	}

	// Resolve relative URL
	resolved := base.ResolveReference(ref)
	return resolved.String()
}

// extractFeedTitle parses RSS or Atom feed content and extracts the title
func extractFeedTitle(feedContent string) (string, error) {
	// Try parsing as RSS first
	var rss RSS
	if err := xml.Unmarshal([]byte(feedContent), &rss); err == nil && rss.Channel.Title != "" {
		return strings.TrimSpace(rss.Channel.Title), nil
	}

	// Try parsing as Atom
	var atom Atom
	if err := xml.Unmarshal([]byte(feedContent), &atom); err == nil && atom.Title != "" {
		return strings.TrimSpace(atom.Title), nil
	}

	return "", fmt.Errorf("could not extract title from feed (not valid RSS or Atom)")
}

// IsSuccessful returns true if the feed discovery was successful
func (r *FeedDiscoveryResult) IsSuccessful() bool {
	return r.Error == nil && r.FeedURL != "" && r.FeedTitle != ""
}
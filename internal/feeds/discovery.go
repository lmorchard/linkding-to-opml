package feeds

import (
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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
	return DiscoverFeedWithDebug(pageURL, httpClient, userAgent, false, "")
}

// DiscoverFeedWithDebug attempts to discover and validate an RSS/Atom feed from a given URL with debug options
func DiscoverFeedWithDebug(pageURL string, httpClient *HTTPClient, userAgent string, saveFailedHTML bool, debugOutputDir string) *FeedDiscoveryResult {
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
		}).Warn("Feed discovery failed: could not fetch page")
		return result
	}

	logrus.WithFields(logrus.Fields{
		"url": pageURL,
		"page_size": len(pageContent),
		"content_preview": getContentPreview(pageContent, 200),
	}).Debug("Successfully fetched page for feed discovery")

	// Step 2: Parse HTML and find feed links
	feedURLs := findFeedLinks(pageContent, pageURL)
	if len(feedURLs) == 0 {
		result.Error = fmt.Errorf("no feed links found in page")
		
		// Save failed HTML for debugging if requested
		if saveFailedHTML && debugOutputDir != "" {
			savedPath := saveFailedHTMLContent(pageURL, pageContent, debugOutputDir, "no_feeds_found")
			logrus.WithFields(logrus.Fields{
				"url": pageURL,
				"saved_html": savedPath,
			}).Debug("Saved failed HTML for debugging")
		}
		
		logrus.WithFields(logrus.Fields{
			"url": pageURL,
			"page_size": len(pageContent),
			"has_head_tag": strings.Contains(strings.ToLower(pageContent), "<head"),
			"has_link_tag": strings.Contains(strings.ToLower(pageContent), "<link"),
			"has_rss_mention": strings.Contains(strings.ToLower(pageContent), "rss"),
			"has_atom_mention": strings.Contains(strings.ToLower(pageContent), "atom"),
			"has_feed_mention": strings.Contains(strings.ToLower(pageContent), "feed"),
			"content_type_analysis": analyzeContentType(pageContent),
		}).Warn("Feed discovery failed: no feed links found in page")
		return result
	}

	// Step 3: Try each feed URL found until we get one that works
	logrus.WithFields(logrus.Fields{
		"page_url": pageURL,
		"total_found": len(feedURLs),
		"all_feeds": feedURLs,
	}).Info("Found potential feed links, trying each one")

	for i, feedURL := range feedURLs {
		logrus.WithFields(logrus.Fields{
			"page_url": pageURL,
			"feed_url": feedURL,
			"attempt": i + 1,
			"total": len(feedURLs),
		}).Debug("Attempting to fetch feed")

		// Step 4: Fetch and validate the feed
		feedContent, err := httpClient.FetchPage(feedURL, userAgent)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"page_url": pageURL,
				"feed_url": feedURL,
				"error":    err,
				"attempt": i + 1,
			}).Debug("Failed to fetch this feed URL, trying next")
			continue
		}

		logrus.WithFields(logrus.Fields{
			"page_url": pageURL,
			"feed_url": feedURL,
			"feed_size": len(feedContent),
			"feed_preview": getContentPreview(feedContent, 200),
		}).Debug("Successfully fetched feed content")

		// Step 5: Parse feed and extract title
		feedTitle, err := extractFeedTitle(feedContent)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"page_url": pageURL,
				"feed_url": feedURL,
				"error":    err,
				"feed_size": len(feedContent),
				"feed_preview": getContentPreview(feedContent, 500),
				"attempt": i + 1,
			}).Debug("Failed to parse this feed, trying next")
			continue
		}

		// Success!
		result.FeedURL = feedURL
		result.FeedTitle = feedTitle
		
		logrus.WithFields(logrus.Fields{
			"page_url":   pageURL,
			"feed_url":   feedURL,
			"feed_title": feedTitle,
			"attempt":    i + 1,
		}).Info("Feed discovery successful")

		return result
	}

	// If we get here, none of the feed URLs worked
	result.Error = fmt.Errorf("found %d potential feed URLs but none were valid feeds", len(feedURLs))
	
	// Save failed HTML for debugging if requested
	if saveFailedHTML && debugOutputDir != "" {
		savedPath := saveFailedHTMLContent(pageURL, pageContent, debugOutputDir, "feeds_found_but_invalid")
		logrus.WithFields(logrus.Fields{
			"page_url": pageURL,
			"saved_html": savedPath,
			"attempted_feeds": len(feedURLs),
		}).Debug("Saved failed HTML for debugging")
	}
	
	logrus.WithFields(logrus.Fields{
		"page_url": pageURL,
		"attempted_feeds": len(feedURLs),
		"all_feeds": feedURLs,
	}).Warn("Feed discovery failed: no valid feeds found among candidates")

	return result
}

// findFeedLinks parses HTML content and extracts RSS/Atom feed URLs using autodiscovery
func findFeedLinks(htmlContent, baseURL string) []string {
	var feedURLs []string
	
	logrus.WithFields(logrus.Fields{
		"base_url": baseURL,
		"content_size": len(htmlContent),
	}).Debug("Starting feed link discovery")

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		logrus.WithError(err).Debug("Failed to parse HTML, falling back to regex")
		return findFeedLinksRegex(htmlContent, baseURL)
	}

	linkCount := 0
	alternateCount := 0
	
	// Walk the HTML tree looking for link elements
	var walkNode func(*html.Node)
	walkNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			linkCount++
			var rel, typ, href, title string
			
			// Extract attributes
			for _, attr := range n.Attr {
				switch strings.ToLower(attr.Key) {
				case "rel":
					rel = attr.Val
				case "type":
					typ = attr.Val
				case "href":
					href = attr.Val
				case "title":
					title = attr.Val
				}
			}
			
			// Log all link tags for debugging
			logrus.WithFields(logrus.Fields{
				"rel": rel,
				"type": typ,
				"href": href,
				"title": title,
			}).Debug("Found link tag")

			// Check if this is a feed link with more permissive matching
			relLower := strings.ToLower(rel)
			typLower := strings.ToLower(typ)
			
			if strings.Contains(relLower, "alternate") {
				alternateCount++
				
				// Check for feed types (be more permissive)
				isFeedType := strings.Contains(typLower, "application/rss+xml") || 
							  strings.Contains(typLower, "application/atom+xml") ||
							  strings.Contains(typLower, "application/rdf+xml") ||
							  strings.Contains(typLower, "text/xml") ||
							  strings.Contains(typLower, "application/xml")
							  
				// Also check href for common feed patterns
				hrefLower := strings.ToLower(href)
				hasCommonFeedPath := strings.Contains(hrefLower, "rss") ||
									 strings.Contains(hrefLower, "feed") ||
									 strings.Contains(hrefLower, "atom") ||
									 strings.Contains(hrefLower, ".xml")
				
				if isFeedType || (strings.Contains(relLower, "alternate") && hasCommonFeedPath) {
					// Convert relative URLs to absolute
					if feedURL := resolveURL(href, baseURL); feedURL != "" {
						feedURLs = append(feedURLs, feedURL)
						logrus.WithFields(logrus.Fields{
							"base_url": baseURL,
							"rel":      rel,
							"type":     typ,
							"href":     href,
							"title":    title,
							"resolved": feedURL,
							"match_reason": getMatchReason(isFeedType, hasCommonFeedPath),
						}).Info("Found feed link")
					}
				} else {
					logrus.WithFields(logrus.Fields{
						"base_url": baseURL,
						"rel":      rel,
						"type":     typ,
						"href":     href,
						"title":    title,
						"reason":   "type_mismatch_or_no_feed_path",
					}).Debug("Alternate link found but not recognized as feed")
				}
			}
		}

		// Recursively walk child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkNode(c)
		}
	}

	walkNode(doc)
	
	logrus.WithFields(logrus.Fields{
		"base_url": baseURL,
		"total_link_tags": linkCount,
		"alternate_links": alternateCount,
		"feed_links_found": len(feedURLs),
	}).Debug("HTML parsing complete")

	// If we didn't find any feeds with HTML parsing, try regex as fallback
	if len(feedURLs) == 0 {
		logrus.WithField("base_url", baseURL).Debug("No feeds found with HTML parser, trying regex fallback")
		feedURLs = findFeedLinksRegex(htmlContent, baseURL)
		
		if len(feedURLs) > 0 {
			logrus.WithFields(logrus.Fields{
				"base_url": baseURL,
				"regex_found": len(feedURLs),
			}).Info("Regex fallback found feeds that HTML parsing missed")
		}
	}
	
	// Try common feed paths as last resort
	if len(feedURLs) == 0 {
		logrus.WithField("base_url", baseURL).Debug("No feeds found, trying common feed paths")
		feedURLs = tryCommonFeedPaths(baseURL)
		
		if len(feedURLs) > 0 {
			logrus.WithFields(logrus.Fields{
				"base_url": baseURL,
				"common_paths_found": len(feedURLs),
			}).Info("Common feed paths found feeds")
		}
	}

	return feedURLs
}

// getMatchReason returns a human-readable reason why a link was matched as a feed
func getMatchReason(isFeedType, hasCommonFeedPath bool) string {
	if isFeedType && hasCommonFeedPath {
		return "feed_type_and_path"
	}
	if isFeedType {
		return "feed_type"
	}
	if hasCommonFeedPath {
		return "common_path"
	}
	return "unknown"
}

// tryCommonFeedPaths attempts to find feeds at common locations
func tryCommonFeedPaths(baseURL string) []string {
	var feedURLs []string
	
	// Parse the base URL to build common feed paths
	base, err := url.Parse(baseURL)
	if err != nil {
		return feedURLs
	}
	
	// Common feed paths to try
	commonPaths := []string{
		"/feed",
		"/feed.xml", 
		"/rss",
		"/rss.xml",
		"/atom.xml",
		"/feeds/all.atom.xml",
		"/index.xml",
		"/.rss",
	}
	
	for _, path := range commonPaths {
		feedURL := base.Scheme + "://" + base.Host + path
		feedURLs = append(feedURLs, feedURL)
		
		logrus.WithFields(logrus.Fields{
			"base_url": baseURL,
			"feed_url": feedURL,
			"path": path,
		}).Debug("Added common feed path to try")
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

// getContentPreview returns a safe preview of content for debugging
func getContentPreview(content string, maxLength int) string {
	if len(content) == 0 {
		return "(empty)"
	}
	
	// Remove any potentially problematic characters for logging
	preview := strings.ReplaceAll(content, "\n", " ")
	preview = strings.ReplaceAll(preview, "\r", " ")
	preview = strings.ReplaceAll(preview, "\t", " ")
	
	if len(preview) > maxLength {
		preview = preview[:maxLength] + "..."
	}
	
	return preview
}

// analyzeContentType attempts to determine what type of content we received
func analyzeContentType(content string) string {
	if len(content) == 0 {
		return "empty"
	}
	
	lowerContent := strings.ToLower(content)
	
	// Check for common indicators
	if strings.Contains(lowerContent, "<!doctype html") || strings.Contains(lowerContent, "<html") {
		return "html"
	}
	
	if strings.Contains(lowerContent, "<?xml") {
		if strings.Contains(lowerContent, "<rss") {
			return "rss_xml"
		}
		if strings.Contains(lowerContent, "<feed") {
			return "atom_xml"
		}
		return "xml"
	}
	
	if strings.Contains(lowerContent, "<rss") {
		return "rss"
	}
	
	if strings.Contains(lowerContent, "<feed") {
		return "atom"
	}
	
	if strings.Contains(lowerContent, "{") && strings.Contains(lowerContent, "}") {
		return "json"
	}
	
	// Check if it looks like an error page
	if strings.Contains(lowerContent, "404") || strings.Contains(lowerContent, "not found") {
		return "404_error"
	}
	
	if strings.Contains(lowerContent, "403") || strings.Contains(lowerContent, "forbidden") {
		return "403_error"
	}
	
	if strings.Contains(lowerContent, "500") || strings.Contains(lowerContent, "internal server error") {
		return "500_error"
	}
	
	return "unknown"
}

// saveFailedHTMLContent saves HTML content to disk for debugging purposes
func saveFailedHTMLContent(pageURL, htmlContent, debugOutputDir, reason string) string {
	if debugOutputDir == "" {
		return ""
	}
	
	// Create debug directory if it doesn't exist
	if err := os.MkdirAll(debugOutputDir, 0755); err != nil {
		logrus.WithError(err).Warn("Failed to create debug output directory")
		return ""
	}
	
	// Generate a safe filename from the URL
	urlHash := fmt.Sprintf("%x", md5.Sum([]byte(pageURL)))
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s_%s_%s.html", timestamp, reason, urlHash[:8])
	
	filePath := filepath.Join(debugOutputDir, filename)
	
	// Create debug info header
	debugInfo := fmt.Sprintf(`<!-- Debug Info
URL: %s
Reason: %s
Timestamp: %s
Content Size: %d bytes
-->

`, pageURL, reason, time.Now().Format(time.RFC3339), len(htmlContent))
	
	// Write HTML content with debug info
	content := debugInfo + htmlContent
	
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		logrus.WithFields(logrus.Fields{
			"file_path": filePath,
			"error": err,
		}).Warn("Failed to save HTML content for debugging")
		return ""
	}
	
	return filePath
}
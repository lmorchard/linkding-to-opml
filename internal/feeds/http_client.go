package feeds

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// HTTPClient provides a configurable HTTP client for fetching web pages
type HTTPClient struct {
	client *http.Client
}

// HTTPConfig holds configuration for the HTTP client
type HTTPConfig struct {
	Timeout      time.Duration
	UserAgent    string
	MaxRedirects int
}

// NewHTTPClient creates a new HTTP client with the specified configuration
func NewHTTPClient(config HTTPConfig) *HTTPClient {
	// Custom redirect policy to limit the number of redirects
	redirectPolicy := func(req *http.Request, via []*http.Request) error {
		if len(via) >= config.MaxRedirects {
			logrus.WithFields(logrus.Fields{
				"url":           req.URL.String(),
				"redirect_count": len(via),
				"max_redirects":  config.MaxRedirects,
			}).Debug("HTTP request exceeded maximum redirects")
			return fmt.Errorf("stopped after %d redirects", config.MaxRedirects)
		}
		return nil
	}

	client := &http.Client{
		Timeout:       config.Timeout,
		CheckRedirect: redirectPolicy,
	}

	logrus.WithFields(logrus.Fields{
		"timeout":       config.Timeout,
		"user_agent":    config.UserAgent,
		"max_redirects": config.MaxRedirects,
	}).Debug("Created HTTP client for feed discovery")

	return &HTTPClient{
		client: client,
	}
}

// FetchPage fetches a web page and returns its content as a string
func (h *HTTPClient) FetchPage(url, userAgent string) (string, error) {
	logrus.WithField("url", url).Debug("Fetching web page")

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set User-Agent header
	req.Header.Set("User-Agent", userAgent)
	
	// Set additional headers that make us look more like a browser
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// Perform request
	resp, err := h.client.Do(req)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"url":   url,
			"error": err,
		}).Debug("HTTP request failed")
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"url":         url,
			"status_code": resp.StatusCode,
			"status":      resp.Status,
		}).Debug("HTTP request returned non-200 status")
		return "", fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Handle compressed content
	var reader io.Reader = resp.Body
	contentEncoding := resp.Header.Get("Content-Encoding")
	
	if strings.Contains(contentEncoding, "gzip") {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
		
		logrus.WithField("url", url).Debug("Decompressing gzip content")
	}

	// Read response body
	body, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"url":              url,
		"status_code":      resp.StatusCode,
		"body_size":        len(body),
		"content_type":     resp.Header.Get("Content-Type"),
		"content_encoding": contentEncoding,
		"was_compressed":   strings.Contains(contentEncoding, "gzip"),
	}).Debug("Successfully fetched web page")

	return string(body), nil
}

// IsRetryableError determines if an HTTP error is worth retrying
func IsRetryableError(err error) bool {
	// For now, we don't implement retry logic
	// This could be expanded to handle specific error types like temporary network errors
	return false
}

// GetContentType extracts content type from response headers (helper for future use)
func GetContentType(resp *http.Response) string {
	return resp.Header.Get("Content-Type")
}
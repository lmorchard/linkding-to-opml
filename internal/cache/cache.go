package cache

import (
	"encoding/gob"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CacheEntry represents a single cached feed discovery result
type CacheEntry struct {
	URL       string    `json:"url"`
	FeedURL   string    `json:"feed_url"`
	FeedTitle string    `json:"feed_title"`
	Timestamp time.Time `json:"timestamp"`
}

// Cache manages the persistent cache of feed discovery results
type Cache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	filePath string
}

// NewCache creates a new cache instance
func NewCache(filePath string) *Cache {
	return &Cache{
		entries:  make(map[string]*CacheEntry),
		filePath: filePath,
	}
}

// LoadCache loads the cache from disk, creating a new cache if file doesn't exist or is corrupted
func (c *Cache) LoadCache() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if cache file exists
	if _, err := os.Stat(c.filePath); os.IsNotExist(err) {
		logrus.Debug("Cache file does not exist, starting with empty cache")
		return nil
	}

	// Open cache file
	file, err := os.Open(c.filePath)
	if err != nil {
		logrus.WithError(err).Warn("Failed to open cache file, starting with empty cache")
		return nil
	}
	defer file.Close()

	// Decode cache entries
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&c.entries)
	if err != nil {
		logrus.WithError(err).Warn("Failed to decode cache file (possibly corrupted), starting with empty cache")
		c.entries = make(map[string]*CacheEntry)
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"file":    c.filePath,
		"entries": len(c.entries),
	}).Debug("Successfully loaded cache from disk")

	return nil
}

// SaveCache writes the cache to disk
func (c *Cache) SaveCache() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create temporary file for atomic write
	tempFile := c.filePath + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary cache file: %w", err)
	}

	// Encode cache entries
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(c.entries)
	if err != nil {
		file.Close()
		os.Remove(tempFile)
		return fmt.Errorf("failed to encode cache data: %w", err)
	}

	file.Close()

	// Atomically replace the old cache file
	err = os.Rename(tempFile, c.filePath)
	if err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to replace cache file: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"file":    c.filePath,
		"entries": len(c.entries),
	}).Debug("Successfully saved cache to disk")

	return nil
}

// Get retrieves a cached entry if it exists and is not stale
func (c *Cache) Get(url string, maxAgeHours int) *CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[url]
	if !exists {
		logrus.WithField("url", url).Debug("Cache miss: no entry found")
		return nil
	}

	if c.isStale(entry, maxAgeHours) {
		logrus.WithFields(logrus.Fields{
			"url": url,
			"age": time.Since(entry.Timestamp),
		}).Debug("Cache miss: entry is stale")
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"url":        url,
		"feed_url":   entry.FeedURL,
		"feed_title": entry.FeedTitle,
		"age":        time.Since(entry.Timestamp),
	}).Debug("Cache hit: returning fresh entry")

	return entry
}

// Set stores a new cache entry
func (c *Cache) Set(url, feedURL, feedTitle string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := &CacheEntry{
		URL:       url,
		FeedURL:   feedURL,
		FeedTitle: feedTitle,
		Timestamp: time.Now(),
	}

	c.entries[url] = entry

	logrus.WithFields(logrus.Fields{
		"url":        url,
		"feed_url":   feedURL,
		"feed_title": feedTitle,
	}).Debug("Cached new feed discovery result")
}

// SetFailed stores a cache entry for a URL that failed feed discovery
// This prevents repeated attempts for URLs that don't have feeds
func (c *Cache) SetFailed(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := &CacheEntry{
		URL:       url,
		FeedURL:   "", // Empty indicates no feed found
		FeedTitle: "",
		Timestamp: time.Now(),
	}

	c.entries[url] = entry

	logrus.WithField("url", url).Debug("Cached failed feed discovery result")
}

// isStale checks if a cache entry is older than the maximum allowed age
func (c *Cache) isStale(entry *CacheEntry, maxAgeHours int) bool {
	maxAge := time.Duration(maxAgeHours) * time.Hour
	return time.Since(entry.Timestamp) > maxAge
}

// Stats returns cache statistics
func (c *Cache) Stats() (int, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalEntries := len(c.entries)
	successfulEntries := 0

	for _, entry := range c.entries {
		if entry.FeedURL != "" {
			successfulEntries++
		}
	}

	return totalEntries, successfulEntries
}

// HasFeed returns true if the cached entry has a feed URL (successful discovery)
func (entry *CacheEntry) HasFeed() bool {
	return entry != nil && entry.FeedURL != ""
}
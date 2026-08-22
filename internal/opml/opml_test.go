package opml

import (
	"errors"
	"testing"

	"linkding-to-opml/internal/feeds"
)

func result(bookmarkURL, feedURL, feedTitle string) *feeds.FeedDiscoveryResult {
	return &feeds.FeedDiscoveryResult{
		URL:       bookmarkURL,
		FeedURL:   feedURL,
		FeedTitle: feedTitle,
	}
}

func xmlURLs(o *OPML) []string {
	urls := make([]string, 0, len(o.Body.Outlines))
	for _, outline := range o.Body.Outlines {
		urls = append(urls, outline.XMLURL)
	}
	return urls
}

// TestGenerateOPMLDeduplicatesByFeedURL covers the common case where several
// bookmarks on the same site all resolve to that site's single feed. Emitting
// one outline per bookmark would subscribe a reader to the same feed repeatedly.
func TestGenerateOPMLDeduplicatesByFeedURL(t *testing.T) {
	results := []*feeds.FeedDiscoveryResult{
		result("https://news.ycombinator.com/item?id=1", "https://news.ycombinator.com/rss", "Hacker News"),
		result("https://adactio.com/journal/1", "https://adactio.com/journal/rss", "Adactio"),
		result("https://news.ycombinator.com/item?id=2", "https://news.ycombinator.com/rss", "Hacker News"),
		result("https://news.ycombinator.com/item?id=3", "https://news.ycombinator.com/rss", "Hacker News"),
	}

	got := GenerateOPML(results, "Test")

	want := []string{"https://news.ycombinator.com/rss", "https://adactio.com/journal/rss"}
	urls := xmlURLs(got)
	if len(urls) != len(want) {
		t.Fatalf("GenerateOPML() produced %d outlines (%v), want %d", len(urls), urls, len(want))
	}
	for i, w := range want {
		if urls[i] != w {
			t.Errorf("outline %d has xmlUrl %q, want %q", i, urls[i], w)
		}
	}
}

// TestGenerateOPMLKeepsFirstOccurrenceOfDuplicateFeed pins down which duplicate
// survives, so the htmlUrl a reader shows stays stable between runs.
func TestGenerateOPMLKeepsFirstOccurrenceOfDuplicateFeed(t *testing.T) {
	results := []*feeds.FeedDiscoveryResult{
		result("https://example.com/first", "https://example.com/feed", "First"),
		result("https://example.com/second", "https://example.com/feed", "Second"),
	}

	got := GenerateOPML(results, "Test")

	if len(got.Body.Outlines) != 1 {
		t.Fatalf("GenerateOPML() produced %d outlines, want 1", len(got.Body.Outlines))
	}
	outline := got.Body.Outlines[0]
	if outline.HTMLURL != "https://example.com/first" {
		t.Errorf("outline htmlUrl = %q, want the first occurrence %q", outline.HTMLURL, "https://example.com/first")
	}
	if outline.Title != "First" {
		t.Errorf("outline title = %q, want %q", outline.Title, "First")
	}
}

// TestGenerateOPMLKeepsDistinctFeeds guards against the dedup collapsing feeds
// that merely share a site.
func TestGenerateOPMLKeepsDistinctFeeds(t *testing.T) {
	results := []*feeds.FeedDiscoveryResult{
		result("https://example.com/a", "https://example.com/feed/a", "A"),
		result("https://example.com/b", "https://example.com/feed/b", "B"),
		result("https://example.com/c", "https://example.com/feed/c", "C"),
	}

	got := GenerateOPML(results, "Test")

	if len(got.Body.Outlines) != 3 {
		t.Errorf("GenerateOPML() produced %d outlines (%v), want 3", len(got.Body.Outlines), xmlURLs(got))
	}
}

// TestGenerateOPMLSkipsUnsuccessfulResults confirms dedup did not disturb the
// existing filter on failed discoveries.
func TestGenerateOPMLSkipsUnsuccessfulResults(t *testing.T) {
	failed := result("https://example.com/broken", "", "")
	failed.Error = errors.New("discovery failed")

	results := []*feeds.FeedDiscoveryResult{
		failed,
		result("https://example.com/missing-title", "https://example.com/feed", ""),
		result("https://example.com/ok", "https://example.com/good-feed", "Good"),
	}

	got := GenerateOPML(results, "Test")

	want := []string{"https://example.com/good-feed"}
	if urls := xmlURLs(got); len(urls) != 1 || urls[0] != want[0] {
		t.Errorf("GenerateOPML() produced %v, want %v", urls, want)
	}
}

package linkding

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeBookmark is the minimal shape the fake Linkding server serves.
type fakeBookmark struct {
	ID       int      `json:"id"`
	URL      string   `json:"url"`
	Title    string   `json:"title"`
	TagNames []string `json:"tag_names"`
}

// fakeLinkding stands in for a Linkding instance. It reproduces the paging and
// tag-search behaviour observed against a real instance:
//
//   - responses are page-limited; `count` reports the full match total while
//     `results` holds only the requested slice
//   - `limit` defaults to 100 when absent
//   - a `q` of "#tag" matches that tag exactly (it does NOT prefix-match, so
//     "#feeds" will not match a "feeds:subscriptions" tag)
//   - multiple "#tag" tokens are ANDed together
//   - tag matching is case-insensitive
type fakeLinkding struct {
	bookmarks []fakeBookmark
	maxLimit  int

	// requests records the raw query string of every list call, so tests can
	// assert on how the client paged and what it asked the server to filter.
	requests []url.Values
}

func (f *fakeLinkding) matches(b fakeBookmark, q string) bool {
	for _, token := range strings.Fields(q) {
		if !strings.HasPrefix(token, "#") {
			continue
		}
		want := strings.ToLower(strings.TrimPrefix(token, "#"))
		found := false
		for _, tag := range b.TagNames {
			if strings.ToLower(tag) == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (f *fakeLinkding) start(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/bookmarks/", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		f.requests = append(f.requests, query)

		limit := 100
		if v := query.Get("limit"); v != "" {
			parsed, err := strconv.Atoi(v)
			if err != nil {
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
			limit = parsed
		}
		if f.maxLimit > 0 && limit > f.maxLimit {
			limit = f.maxLimit
		}

		offset := 0
		if v := query.Get("offset"); v != "" {
			parsed, err := strconv.Atoi(v)
			if err != nil {
				http.Error(w, "bad offset", http.StatusBadRequest)
				return
			}
			offset = parsed
		}

		matched := make([]fakeBookmark, 0, len(f.bookmarks))
		for _, b := range f.bookmarks {
			if f.matches(b, query.Get("q")) {
				matched = append(matched, b)
			}
		}

		start := offset
		if start > len(matched) {
			start = len(matched)
		}
		end := start + limit
		if end > len(matched) {
			end = len(matched)
		}
		page := matched[start:end]

		next := ""
		if end < len(matched) {
			next = fmt.Sprintf("/api/bookmarks/?limit=%d&offset=%d", limit, end)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"count":    len(matched),
			"next":     next,
			"previous": "",
			"results":  page,
		})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// newTestClient builds a Client pointed at the fake server.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client, err := NewClient("test-token", server.URL, 30*time.Second)
	if err != nil {
		t.Fatalf("NewClient() returned unexpected error: %v", err)
	}
	return client
}

// makeBookmarks builds n bookmarks, tagging every taggedEvery-th one with tag.
func makeBookmarks(n, taggedEvery int, tag string) []fakeBookmark {
	bookmarks := make([]fakeBookmark, 0, n)
	for i := 0; i < n; i++ {
		tags := []string{"misc"}
		if taggedEvery > 0 && i%taggedEvery == 0 {
			tags = append(tags, tag)
		}
		bookmarks = append(bookmarks, fakeBookmark{
			ID:       i,
			URL:      fmt.Sprintf("https://example.com/%d", i),
			Title:    fmt.Sprintf("Bookmark %d", i),
			TagNames: tags,
		})
	}
	return bookmarks
}

// TestFetchBookmarksReturnsEveryTaggedBookmarkAcrossPages is the regression test
// for the truncation bug: with a bookmark collection far larger than one page,
// tagged bookmarks living beyond the first page were silently dropped.
func TestFetchBookmarksReturnsEveryTaggedBookmarkAcrossPages(t *testing.T) {
	const tag = "feeds:subscriptions"
	// 5000 bookmarks, every 10th tagged => 500 expected matches, only 100 of
	// which fall within the first 1000 records.
	fake := &fakeLinkding{bookmarks: makeBookmarks(5000, 10, tag)}
	server := fake.start(t)
	client := newTestClient(t, server)

	got, err := client.FetchBookmarks([]string{tag})
	if err != nil {
		t.Fatalf("FetchBookmarks() returned unexpected error: %v", err)
	}

	if len(got) != 500 {
		t.Errorf("FetchBookmarks() returned %d bookmarks, want 500", len(got))
	}

	for _, b := range got {
		hasTag := false
		for _, candidate := range b.Tags {
			if candidate == tag {
				hasTag = true
				break
			}
		}
		if !hasTag {
			t.Errorf("FetchBookmarks() returned %s without tag %q (tags: %v)", b.URL, tag, b.Tags)
		}
	}
}

// TestFetchBookmarksReturnsAllBookmarksWhenUnfiltered covers the no-tag path,
// which pages through the entire collection.
func TestFetchBookmarksReturnsAllBookmarksWhenUnfiltered(t *testing.T) {
	fake := &fakeLinkding{bookmarks: makeBookmarks(2500, 0, "")}
	server := fake.start(t)
	client := newTestClient(t, server)

	got, err := client.FetchBookmarks(nil)
	if err != nil {
		t.Fatalf("FetchBookmarks() returned unexpected error: %v", err)
	}

	if len(got) != 2500 {
		t.Errorf("FetchBookmarks() returned %d bookmarks, want 2500", len(got))
	}
}

// TestFetchBookmarksFiltersOnTheServer asserts the tag filter is pushed into the
// API query. Filtering only client-side forces a walk of the entire collection,
// which on a large instance means tens of thousands of records transferred to
// find a few hundred matches.
func TestFetchBookmarksFiltersOnTheServer(t *testing.T) {
	const tag = "feeds:subscriptions"
	// 5000 bookmarks, every 10th tagged => 500 matches, one page at pageSize.
	fake := &fakeLinkding{bookmarks: makeBookmarks(5000, 10, tag)}
	server := fake.start(t)
	client := newTestClient(t, server)

	got, err := client.FetchBookmarks([]string{tag})
	if err != nil {
		t.Fatalf("FetchBookmarks() returned unexpected error: %v", err)
	}
	if len(got) != 500 {
		t.Fatalf("FetchBookmarks() returned %d bookmarks, want 500", len(got))
	}

	for i, req := range fake.requests {
		if q := req.Get("q"); q != "#"+tag {
			t.Errorf("request %d sent q=%q, want %q", i, q, "#"+tag)
		}
	}

	// 500 matches at a 500-per-page limit is a single request. Requiring more
	// means unmatched bookmarks were dragged over the wire.
	if len(fake.requests) != 1 {
		t.Errorf("made %d requests to collect 500 matches, want 1", len(fake.requests))
	}
}

// TestFetchBookmarksSendsNoQueryWhenUnfiltered guards against sending a stray
// query that would silently narrow an unfiltered export.
func TestFetchBookmarksSendsNoQueryWhenUnfiltered(t *testing.T) {
	fake := &fakeLinkding{bookmarks: makeBookmarks(10, 0, "")}
	server := fake.start(t)
	client := newTestClient(t, server)

	if _, err := client.FetchBookmarks(nil); err != nil {
		t.Fatalf("FetchBookmarks() returned unexpected error: %v", err)
	}

	for i, req := range fake.requests {
		if q := req.Get("q"); q != "" {
			t.Errorf("request %d sent q=%q, want no query", i, q)
		}
	}
}

func TestBuildTagQuery(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want string
	}{
		{name: "no tags", tags: nil, want: ""},
		{name: "single tag", tags: []string{"feeds:subscriptions"}, want: "#feeds:subscriptions"},
		{name: "multiple tags are ANDed", tags: []string{"feeds:subscriptions", "rss"}, want: "#feeds:subscriptions #rss"},
		{name: "blank tags are dropped", tags: []string{"", "  ", "rss"}, want: "#rss"},
		{name: "surrounding whitespace is trimmed", tags: []string{"  rss  "}, want: "#rss"},
		{name: "tags containing spaces are left to client-side filtering", tags: []string{"two words", "rss"}, want: "#rss"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildTagQuery(tt.tags); got != tt.want {
				t.Errorf("buildTagQuery(%q) = %q, want %q", tt.tags, got, tt.want)
			}
		})
	}
}

// TestFetchBookmarksHandlesServerCappedPageSize covers a server that enforces a
// smaller page size than requested. Paging must follow what the server actually
// returned rather than assuming the requested limit was honoured.
func TestFetchBookmarksHandlesServerCappedPageSize(t *testing.T) {
	fake := &fakeLinkding{bookmarks: makeBookmarks(1200, 0, ""), maxLimit: 100}
	server := fake.start(t)
	client := newTestClient(t, server)

	got, err := client.FetchBookmarks(nil)
	if err != nil {
		t.Fatalf("FetchBookmarks() returned unexpected error: %v", err)
	}

	if len(got) != 1200 {
		t.Errorf("FetchBookmarks() returned %d bookmarks, want 1200", len(got))
	}
}

// TestFetchBookmarksStopsWhenServerOverreportsCount ensures an inflated `count`
// cannot spin the pager forever. The loop must end on an empty page.
func TestFetchBookmarksStopsWhenServerOverreportsCount(t *testing.T) {
	mux := http.NewServeMux()
	calls := 0
	mux.HandleFunc("/api/bookmarks/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls > 20 {
			t.Error("FetchBookmarks() did not stop paging; likely an infinite loop")
			http.Error(w, "too many calls", http.StatusInternalServerError)
			return
		}
		results := []fakeBookmark{}
		if calls == 1 {
			results = append(results, fakeBookmark{ID: 1, URL: "https://example.com/1"})
		}
		w.Header().Set("Content-Type", "application/json")
		// count is wildly larger than anything the server will ever hand back.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"count": 999999, "next": "", "previous": "", "results": results,
		})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := newTestClient(t, server)
	got, err := client.FetchBookmarks(nil)
	if err != nil {
		t.Fatalf("FetchBookmarks() returned unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("FetchBookmarks() returned %d bookmarks, want 1", len(got))
	}
}

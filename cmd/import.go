package cmd

import (
	"fmt"
	"os"
	"time"

	"linkding-to-opml/internal/feeds"
	"linkding-to-opml/internal/importer"
	"linkding-to-opml/internal/opml"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var importCmd = &cobra.Command{
	Use:   "import [OPML file]",
	Short: "Import OPML feeds as bookmarks to Linkding",
	Long: `Import reads an OPML file containing RSS/Atom feeds, discovers the associated
website URLs, and creates bookmarks in your Linkding instance.

The import process:
1. Parses the OPML file to extract feed entries
2. Discovers website URLs from feeds (using htmlUrl or feed content)  
3. Creates bookmarks in Linkding with discovered metadata
4. Handles duplicates according to your preferences

Examples:
  # Import feeds from OPML file
  linkding-to-opml import feeds.opml

  # Preview import without making changes
  linkding-to-opml import --dry-run feeds.opml

  # Import with custom tags
  linkding-to-opml import --tags "imported,rss" feeds.opml

  # Handle duplicates by updating existing bookmarks
  linkding-to-opml import --duplicates update feeds.opml`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	// Add import command to root
	rootCmd.AddCommand(importCmd)

	// Import-specific flags
	importCmd.Flags().Bool("dry-run", false, "Preview import without creating bookmarks")
	importCmd.Flags().String("duplicates", "skip", "How to handle existing bookmarks: skip or update")
	importCmd.Flags().StringSlice("tags", []string{}, "Comma-separated tags to apply to all imported bookmarks")
	importCmd.Flags().Int("concurrency", 16, "Number of concurrent workers for web fetching")

	// Bind flags to viper
	_ = viper.BindPFlag("import.dry_run", importCmd.Flags().Lookup("dry-run"))
	_ = viper.BindPFlag("import.duplicates", importCmd.Flags().Lookup("duplicates"))
	_ = viper.BindPFlag("import.tags", importCmd.Flags().Lookup("tags"))
	_ = viper.BindPFlag("import.concurrency", importCmd.Flags().Lookup("concurrency"))
}

func runImport(cmd *cobra.Command, args []string) error {
	opmlFile := args[0]
	
	// Verify OPML file exists
	if _, err := os.Stat(opmlFile); os.IsNotExist(err) {
		return fmt.Errorf("OPML file does not exist: %s", opmlFile)
	}
	
	// Validate duplicates flag
	duplicates := viper.GetString("import.duplicates")
	if duplicates != "skip" && duplicates != "update" {
		return fmt.Errorf("invalid duplicates value: %s (must be 'skip' or 'update')", duplicates)
	}
	
	// Get other flags
	dryRun := viper.GetBool("import.dry_run")
	tags := viper.GetStringSlice("import.tags")
	concurrency := viper.GetInt("import.concurrency")
	
	// Parse OPML file
	opmlDoc, err := opml.ReadFile(opmlFile)
	if err != nil {
		return fmt.Errorf("failed to read OPML file: %w", err)
	}
	
	// Extract all feeds
	feedEntries := opmlDoc.GetAllFeeds()
	fmt.Printf("Found %d feed entries in OPML file\n", len(feedEntries))
	
	// Create import items
	items := make([]*importer.ImportItem, len(feedEntries))
	for i, feed := range feedEntries {
		items[i] = &importer.ImportItem{
			FeedEntry: feed,
			Status:    importer.StatusPending,
		}
	}
	
	// Create HTTP client for feed fetching
	httpClient := feeds.NewHTTPClient(feeds.HTTPConfig{
		Timeout:      30 * time.Second,
		UserAgent:    "linkding-to-opml/1.0",
		MaxRedirects: 3,
	})
	
	// Test URL discovery on the first few items
	fmt.Printf("Testing URL discovery on first items:\n")
	for i, item := range items {
		if i >= 3 { // Only test first 3 items
			break
		}
		
		fmt.Printf("\nProcessing: %s\n", item.Title)
		fmt.Printf("  XML URL: %s\n", item.XMLURL)
		fmt.Printf("  HTML URL: %s\n", item.HTMLURL)
		
		err := importer.DiscoverBookmarkURL(item, httpClient)
		if err != nil {
			fmt.Printf("  Error: %s\n", err)
			continue
		}
		
		fmt.Printf("  Final URL: %s\n", item.GetFinalURL())
		fmt.Printf("  Final Title: %s\n", item.GetFinalTitle())
		fmt.Printf("  Final Description: %s\n", item.GetFinalDescription())
	}
	
	fmt.Printf("\nDry run: %t, Duplicates: %s, Tags: %v, Concurrency: %d\n", 
		dryRun, duplicates, tags, concurrency)
	return nil
}
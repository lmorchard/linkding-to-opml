package cmd

import (
	"fmt"
	"os"
	"time"

	"linkding-to-opml/internal/feeds"
	"linkding-to-opml/internal/importer"
	"linkding-to-opml/internal/linkding"
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
	// dryRun := viper.GetBool("import.dry_run")  // Will be used in full implementation
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
	
	// Test concurrent processing with worker pool (dry run mode)
	fmt.Printf("Testing concurrent processing with %d workers...\n", concurrency)
	
	processOptions := importer.ProcessOptions{
		DuplicateAction: duplicates,
		Tags:            tags,
		DryRun:          true, // Force dry run for testing (no real Linkding calls)
	}
	
	// Create dummy Linkding client (nil is OK for dry-run mode)
	var linkdingClient *linkding.Client = nil
	
	stats := importer.ProcessItems(items, httpClient, linkdingClient, processOptions, concurrency)
	
	fmt.Printf("\n%s\n", stats.Summary())
	
	fmt.Printf("Configuration: Duplicates=%s, Tags=%v, Concurrency=%d\n", 
		duplicates, tags, concurrency)
	return nil
}
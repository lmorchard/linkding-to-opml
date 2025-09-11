package cmd

import (
	"fmt"
	"os"

	"linkding-to-opml/internal/config"
	"linkding-to-opml/internal/feeds"
	"linkding-to-opml/internal/importer"
	"linkding-to-opml/internal/linkding"
	"linkding-to-opml/internal/opml"

	"github.com/sirupsen/logrus"
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
	importCmd.Flags().Int("retry-attempts", 3, "Number of retry attempts for failed operations")

	// Bind flags to viper
	_ = viper.BindPFlag("import.dry_run", importCmd.Flags().Lookup("dry-run"))
	_ = viper.BindPFlag("import.duplicates", importCmd.Flags().Lookup("duplicates"))
	_ = viper.BindPFlag("import.tags", importCmd.Flags().Lookup("tags"))
	_ = viper.BindPFlag("import.concurrency", importCmd.Flags().Lookup("concurrency"))
	_ = viper.BindPFlag("retry_attempts", importCmd.Flags().Lookup("retry-attempts"))
}

func runImport(cmd *cobra.Command, args []string) error {
	opmlFile := args[0]
	
	// Verify OPML file exists
	if _, err := os.Stat(opmlFile); os.IsNotExist(err) {
		return fmt.Errorf("OPML file does not exist: %s", opmlFile)
	}
	
	// Load configuration
	configFile := viper.GetString("config")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Set up logging
	cfg.SetupLogging()
	
	// Validate configuration (only if not in dry-run mode)
	if !viper.GetBool("import.dry_run") {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}
	
	// Get import-specific flags
	dryRun := viper.GetBool("import.dry_run")
	duplicates := viper.GetString("import.duplicates")
	tags := viper.GetStringSlice("import.tags")
	concurrency := viper.GetInt("import.concurrency")
	retryAttempts := cfg.RetryAttempts
	
	// Validate duplicates flag
	if duplicates != "skip" && duplicates != "update" {
		return fmt.Errorf("invalid duplicates value: %s (must be 'skip' or 'update')", duplicates)
	}
	
	logrus.WithFields(logrus.Fields{
		"opml_file":      opmlFile,
		"dry_run":        dryRun,
		"duplicates":     duplicates,
		"tags":           tags,
		"concurrency":    concurrency,
		"retry_attempts": retryAttempts,
	}).Info("Starting OPML import")
	
	// Parse OPML file
	opmlDoc, err := opml.ReadFile(opmlFile)
	if err != nil {
		return fmt.Errorf("failed to read OPML file: %w", err)
	}
	
	// Extract all feeds
	feedEntries := opmlDoc.GetAllFeeds()
	logrus.WithField("feed_count", len(feedEntries)).Info("Extracted feed entries from OPML")
	
	if len(feedEntries) == 0 {
		logrus.Warn("No feed entries found in OPML file")
		return nil
	}
	
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
		Timeout:      cfg.HTTP.Timeout,
		UserAgent:    cfg.HTTP.UserAgent,
		MaxRedirects: cfg.HTTP.MaxRedirects,
	})
	
	// Create Linkding client (unless dry-run mode)
	var linkdingClient *linkding.Client
	if !dryRun {
		linkdingClient, err = linkding.NewClient(
			cfg.Linkding.Token,
			cfg.Linkding.URL,
			cfg.Linkding.Timeout,
		)
		if err != nil {
			return fmt.Errorf("failed to create Linkding client: %w", err)
		}
	}
	
	// Process items concurrently
	processOptions := importer.ProcessOptions{
		DuplicateAction: duplicates,
		Tags:            tags,
		DryRun:          dryRun,
		RetryAttempts:   retryAttempts,
	}
	
	logrus.WithFields(logrus.Fields{
		"total_items": len(items),
		"concurrency": concurrency,
		"dry_run":     dryRun,
	}).Info("Starting concurrent processing")
	
	stats := importer.ProcessItems(items, httpClient, linkdingClient, processOptions, concurrency)
	
	// Display final summary
	if !cfg.Quiet {
		fmt.Printf("\n%s\n", stats.Summary())
	}
	
	// Log summary to structured logs as well
	logrus.WithFields(logrus.Fields{
		"total":     stats.Total,
		"processed": stats.Processed,
		"imported":  stats.Imported,
		"updated":   stats.Updated,
		"skipped":   stats.Skipped,
		"failed":    stats.Failed,
		"duration":  stats.Duration(),
	}).Info("Import completed")
	
	// Return error if any items failed
	if stats.Failed > 0 {
		logrus.WithField("failed_count", stats.Failed).Error("Some items failed to import")
		return fmt.Errorf("import completed with %d failures", stats.Failed)
	}
	
	return nil
}
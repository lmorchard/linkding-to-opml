package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"linkding-to-opml/internal/config"
	"linkding-to-opml/internal/feeds"
	"linkding-to-opml/internal/importer"
	"linkding-to-opml/internal/netscape"
	"linkding-to-opml/internal/opml"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var toHTMLCmd = &cobra.Command{
	Use:   "tohtml [OPML file] [HTML output file]",
	Short: "Convert OPML feeds to HTML bookmarks for Linkding import",
	Long: `Convert an OPML file containing RSS/Atom feeds to a Netscape HTML bookmark
file that can be imported directly into Linkding through its web interface.

The conversion process:
1. Parses the OPML file to extract feed entries
2. Discovers website URLs from feeds (using htmlUrl or feed content)
3. Generates an HTML bookmark file compatible with Linkding
4. No API calls - safe and reliable import via Linkding's web UI

Examples:
  # Convert OPML to HTML bookmarks
  linkding-to-opml tohtml feeds.opml bookmarks.html

  # Preview conversion without creating file
  linkding-to-opml tohtml --dry-run feeds.opml bookmarks.html

  # Add tags to all converted bookmarks
  linkding-to-opml tohtml --tags "imported,rss" feeds.opml my-bookmarks.html`,
	Args: cobra.ExactArgs(2),
	RunE: runToHTML,
}

func init() {
	// Add tohtml command to root
	rootCmd.AddCommand(toHTMLCmd)

	// Command-specific flags
	toHTMLCmd.Flags().Bool("dry-run", false, "Preview conversion without creating HTML file")
	toHTMLCmd.Flags().StringSlice("tags", []string{}, "Tags to apply to all converted bookmarks")
	toHTMLCmd.Flags().Int("concurrency", 16, "Number of concurrent workers for web fetching")
	toHTMLCmd.Flags().Int("retry-attempts", 3, "Number of retry attempts for failed URL discovery")

	// Bind flags to viper
	_ = viper.BindPFlag("tohtml.dry_run", toHTMLCmd.Flags().Lookup("dry-run"))
	_ = viper.BindPFlag("tohtml.tags", toHTMLCmd.Flags().Lookup("tags"))
	_ = viper.BindPFlag("tohtml.concurrency", toHTMLCmd.Flags().Lookup("concurrency"))
	_ = viper.BindPFlag("retry_attempts", toHTMLCmd.Flags().Lookup("retry-attempts"))
}

func runToHTML(cmd *cobra.Command, args []string) error {
	opmlFile := args[0]
	outputFile := args[1]

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

	// Get command-specific flags
	dryRun := viper.GetBool("tohtml.dry_run")
	tags := viper.GetStringSlice("tohtml.tags")
	concurrency := viper.GetInt("tohtml.concurrency")
	retryAttempts := cfg.RetryAttempts

	logrus.WithFields(logrus.Fields{
		"opml_file":      opmlFile,
		"output_file":    outputFile,
		"dry_run":        dryRun,
		"tags":           tags,
		"concurrency":    concurrency,
		"retry_attempts": retryAttempts,
	}).Info("Starting OPML to HTML conversion")

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

	// Process items concurrently to discover URLs and metadata
	processOptions := importer.ProcessOptions{
		Tags:          tags,
		DryRun:        dryRun,
		RetryAttempts: retryAttempts,
	}

	logrus.WithFields(logrus.Fields{
		"total_items": len(items),
		"concurrency": concurrency,
		"dry_run":     dryRun,
	}).Info("Starting concurrent processing")

	stats, bookmarks := importer.ProcessItems(items, httpClient, processOptions, concurrency)

	// Create output file if not in dry-run mode
	if !dryRun && len(bookmarks) > 0 {
		outputPath, err := filepath.Abs(outputFile)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		// Export bookmarks to HTML
		exporter := netscape.NewExporter(file)
		if err := exporter.Export(bookmarks); err != nil {
			return fmt.Errorf("failed to export bookmarks: %w", err)
		}

		logrus.WithFields(logrus.Fields{
			"file":      outputPath,
			"bookmarks": len(bookmarks),
		}).Info("Successfully created HTML bookmark file")

		// Display completion message
		if !cfg.Quiet {
			fmt.Printf("\n✅ Created %s with %d bookmarks\n", outputPath, len(bookmarks))
			fmt.Println("\nTo import into Linkding:")
			fmt.Println("1. Log into your Linkding instance")
			fmt.Println("2. Go to Settings → Import")
			fmt.Println("3. Upload the generated HTML file")
			fmt.Println("4. Linkding will handle the import with proper deduplication")
		}
	}

	// Display final summary
	if !cfg.Quiet {
		fmt.Printf("\n%s\n", stats.Summary())
	}

	// Log summary to structured logs as well
	logrus.WithFields(logrus.Fields{
		"total":     stats.Total,
		"processed": stats.Processed,
		"imported":  stats.Imported,
		"failed":    stats.Failed,
		"duration":  stats.Duration(),
		"bookmarks": len(bookmarks),
	}).Info("Conversion completed")

	// Return error if any items failed
	if stats.Failed > 0 {
		logrus.WithField("failed_count", stats.Failed).Error("Some items failed to process")
		return fmt.Errorf("conversion completed with %d failures", stats.Failed)
	}

	return nil
}
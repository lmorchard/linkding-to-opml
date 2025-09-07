package cmd

import (
	"fmt"

	"linkding-to-opml/internal/cache"
	"linkding-to-opml/internal/config"
	"linkding-to-opml/internal/feeds"
	"linkding-to-opml/internal/linkding"
	"linkding-to-opml/internal/opml"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export bookmarks from Linkding to OPML",
	Long: `Export fetches bookmarks from your Linkding instance, discovers RSS/Atom feeds 
from those URLs, and exports them as an OPML file that can be imported into feed readers.

The export process:
1. Fetches bookmarks from Linkding (optionally filtered by tags)
2. Discovers RSS/Atom feeds from bookmark URLs (with caching)
3. Generates an OPML file with discovered feeds

Examples:
  # Export all bookmarks to default feeds.opml
  linkding-to-opml export

  # Export bookmarks with specific tags
  linkding-to-opml export --tags "rss,tech"

  # Export to custom output file
  linkding-to-opml export --output my-feeds.opml

  # Use custom configuration file
  linkding-to-opml export --config /path/to/config.yaml`,
	RunE: runExport,
}

func init() {
	// Add export command to root
	rootCmd.AddCommand(exportCmd)

	// Export-specific flags
	exportCmd.Flags().StringSlice("tags", []string{}, "Comma-separated list of tags to filter bookmarks (empty = all bookmarks)")
	exportCmd.Flags().StringP("output", "o", "", "OPML output file path (default: feeds.opml)")
	exportCmd.Flags().String("cache", "", "Cache file path (default: ./linkding-to-opml.gob)")
	exportCmd.Flags().Int("max-age", 0, "Cache max-age in hours (default: 720)")
	exportCmd.Flags().String("linkding-token", "", "Linkding API token (required)")
	exportCmd.Flags().String("linkding-url", "", "Linkding server URL (required)")
	exportCmd.Flags().String("linkding-timeout", "", "Linkding API timeout (default: 30s)")
	exportCmd.Flags().IntP("concurrency", "c", 0, "Number of concurrent workers (default: 16)")
	exportCmd.Flags().Bool("save-failed-html", false, "Save HTML content of failed feed discoveries for debugging")
	exportCmd.Flags().String("debug-output-dir", "", "Directory to save debug output (default: ./debug)")

	// Bind flags to viper
	viper.BindPFlag("tags", exportCmd.Flags().Lookup("tags"))
	viper.BindPFlag("output", exportCmd.Flags().Lookup("output"))
	viper.BindPFlag("cache.file_path", exportCmd.Flags().Lookup("cache"))
	viper.BindPFlag("cache.max_age", exportCmd.Flags().Lookup("max-age"))
	viper.BindPFlag("linkding.token", exportCmd.Flags().Lookup("linkding-token"))
	viper.BindPFlag("linkding.url", exportCmd.Flags().Lookup("linkding-url"))
	viper.BindPFlag("linkding.timeout", exportCmd.Flags().Lookup("linkding-timeout"))
	viper.BindPFlag("concurrency", exportCmd.Flags().Lookup("concurrency"))
	viper.BindPFlag("save_failed_html", exportCmd.Flags().Lookup("save-failed-html"))
	viper.BindPFlag("debug_output_dir", exportCmd.Flags().Lookup("debug-output-dir"))
}

func runExport(cmd *cobra.Command, args []string) error {
	// Load configuration
	configFile := viper.GetString("config")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set up logging
	cfg.SetupLogging()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	logrus.Info("Starting linkding-to-opml export process")

	// Step 1: Initialize cache
	logrus.Debug("Initializing cache")
	cache := cache.NewCache(cfg.Cache.FilePath)
	if err := cache.LoadCache(); err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	// Step 2: Create Linkding API client
	logrus.Debug("Creating Linkding API client")
	linkdingClient, err := linkding.NewClient(cfg.Linkding.Token, cfg.Linkding.URL, cfg.Linkding.Timeout)
	if err != nil {
		return fmt.Errorf("failed to create Linkding client: %w", err)
	}

	// Step 3: Fetch bookmarks from Linkding
	logrus.Info("Fetching bookmarks from Linkding API")
	bookmarks, err := linkdingClient.FetchBookmarks(cfg.Tags)
	if err != nil {
		return fmt.Errorf("failed to fetch bookmarks: %w", err)
	}

	if len(bookmarks) == 0 {
		logrus.Warn("No bookmarks found matching the specified criteria")
		if !cfg.Quiet {
			fmt.Println("No bookmarks found. Nothing to export.")
		}
		return nil
	}

	// Step 4: Process bookmarks with concurrent feed discovery
	logrus.WithField("bookmark_count", len(bookmarks)).Info("Starting feed discovery")

	processingConfig := feeds.ProcessingConfig{
		Concurrency: cfg.Concurrency,
		MaxAge:      cfg.Cache.MaxAge,
		UserAgent:   cfg.HTTP.UserAgent,
		HTTPConfig: feeds.HTTPConfig{
			Timeout:      cfg.HTTP.Timeout,
			UserAgent:    cfg.HTTP.UserAgent,
			MaxRedirects: cfg.HTTP.MaxRedirects,
		},
		Verbose:        cfg.Verbose,
		SaveFailedHTML: cfg.SaveFailedHTML,
		DebugOutputDir: cfg.DebugOutputDir,
	}

	results, stats := feeds.ProcessBookmarks(bookmarks, cache, processingConfig)

	if len(results) == 0 {
		logrus.Warn("No feeds discovered from bookmarks")
		if !cfg.Quiet {
			fmt.Println("No feeds were discovered from the bookmarks. No OPML file will be created.")
		}
		return nil
	}

	// Step 5: Generate OPML
	logrus.WithField("feed_count", len(results)).Info("Generating OPML document")
	opmlDoc := opml.GenerateOPML(results, "Feeds exported from Linkding")

	// Step 6: Validate OPML
	if err := opml.ValidateOPML(opmlDoc); err != nil {
		return fmt.Errorf("generated OPML is invalid: %w", err)
	}

	// Step 7: Write OPML file
	logrus.WithField("output_file", cfg.Output).Info("Writing OPML file")
	if err := opml.WriteOPML(opmlDoc, cfg.Output); err != nil {
		return fmt.Errorf("failed to write OPML file: %w", err)
	}

	// Step 8: Display summary statistics
	if !cfg.Quiet {
		summary := stats.FormatProcessingSummary(false)
		fmt.Println(summary)
		fmt.Printf("OPML file written to: %s\n", cfg.Output)
	}

	logrus.Info("Export process completed successfully")
	return nil
}

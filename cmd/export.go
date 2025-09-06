package cmd

import (
	"fmt"
	"strings"

	"linkding-to-opml/internal/config"

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

	// Bind flags to viper
	viper.BindPFlag("tags", exportCmd.Flags().Lookup("tags"))
	viper.BindPFlag("output", exportCmd.Flags().Lookup("output"))
	viper.BindPFlag("cache.file_path", exportCmd.Flags().Lookup("cache"))
	viper.BindPFlag("cache.max_age", exportCmd.Flags().Lookup("max-age"))
	viper.BindPFlag("linkding.token", exportCmd.Flags().Lookup("linkding-token"))
	viper.BindPFlag("linkding.url", exportCmd.Flags().Lookup("linkding-url"))
	viper.BindPFlag("linkding.timeout", exportCmd.Flags().Lookup("linkding-timeout"))
	viper.BindPFlag("concurrency", exportCmd.Flags().Lookup("concurrency"))
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

	// Print configuration summary for now
	fmt.Printf("Linkding-to-OPML Export Configuration:\n")
	fmt.Printf("  Linkding URL: %s\n", cfg.Linkding.URL)
	fmt.Printf("  API Token: %s...\n", maskToken(cfg.Linkding.Token))
	fmt.Printf("  API Timeout: %s\n", cfg.Linkding.Timeout)
	fmt.Printf("  Output File: %s\n", cfg.Output)
	fmt.Printf("  Cache File: %s\n", cfg.Cache.FilePath)
	fmt.Printf("  Cache Max-Age: %d hours\n", cfg.Cache.MaxAge)
	fmt.Printf("  Concurrency: %d\n", cfg.Concurrency)
	fmt.Printf("  HTTP Timeout: %s\n", cfg.HTTP.Timeout)
	fmt.Printf("  User-Agent: %s\n", cfg.HTTP.UserAgent)
	fmt.Printf("  Max Redirects: %d\n", cfg.HTTP.MaxRedirects)
	
	if len(cfg.Tags) > 0 {
		fmt.Printf("  Filter Tags: %s\n", strings.Join(cfg.Tags, ", "))
	} else {
		fmt.Printf("  Filter Tags: (all bookmarks)\n")
	}

	fmt.Printf("  Logging: ")
	if cfg.Debug {
		fmt.Printf("DEBUG\n")
	} else if cfg.Verbose {
		fmt.Printf("VERBOSE\n")
	} else if cfg.Quiet {
		fmt.Printf("QUIET\n")
	} else {
		fmt.Printf("NORMAL\n")
	}

	// TODO: Implement actual export logic in subsequent steps
	fmt.Printf("\n[TODO] Export logic will be implemented in subsequent steps\n")
	
	return nil
}

// maskToken shows only the first 8 characters of the token for security
func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:8] + "****"
}
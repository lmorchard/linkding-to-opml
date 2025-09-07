package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "linkding-to-opml",
	Short: "Convert Linkding bookmarks to OPML feed list",
	Long: `linkding-to-opml fetches bookmarks from your Linkding instance, 
discovers RSS/Atom feeds from those URLs, and exports them as an OPML file.

This tool helps you convert your bookmark collection into a feed reader-friendly format
by automatically discovering RSS/Atom feeds from your bookmarks and creating a standard
OPML file that can be imported into any feed reader.

Features:
• Fetches bookmarks from Linkding API with optional tag filtering
• Discovers RSS/Atom feeds using standard autodiscovery methods
• Caches feed discovery results to avoid repeated network requests
• Concurrent processing for fast operation
• Generates OPML 2.0 compatible files
• Comprehensive error handling and logging

Examples:
  # Export all bookmarks to feeds.opml
  linkding-to-opml export --linkding-url https://your-linkding.com --linkding-token your-token

  # Export only bookmarks tagged with "tech" and "rss"
  linkding-to-opml export --tags "tech,rss" --output tech-feeds.opml

  # Use configuration file
  linkding-to-opml export --config /path/to/config.yaml

  # Enable verbose logging
  linkding-to-opml export --verbose

Configuration:
  Create a linkding-to-opml.yaml file in your current directory with:
  
  linkding:
    token: "your-api-token"
    url: "https://your-linkding-instance.com"
  
  See the example configuration file for all available options.`,
	Version: "1.0.0",
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().String("config", "", "Configuration file path (default: ./linkding-to-opml.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress summary output (errors/warnings still shown)")

	// Bind global flags to viper
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

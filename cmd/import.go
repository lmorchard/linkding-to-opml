package cmd

import (
	"fmt"
	"os"

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
	
	fmt.Printf("Import command placeholder - would import from: %s\n", opmlFile)
	fmt.Printf("Dry run: %t, Duplicates: %s, Tags: %v, Concurrency: %d\n", 
		dryRun, duplicates, tags, concurrency)
	return nil
}
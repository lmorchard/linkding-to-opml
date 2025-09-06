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

This tool helps you convert your bookmark collection into a feed reader-friendly format.`,
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
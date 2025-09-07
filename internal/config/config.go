package config

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Linkding API settings
	Linkding struct {
		Token   string        `mapstructure:"token"`
		URL     string        `mapstructure:"url"`
		Timeout time.Duration `mapstructure:"timeout"`
	} `mapstructure:"linkding"`

	// Cache settings
	Cache struct {
		FilePath string `mapstructure:"file_path"`
		MaxAge   int    `mapstructure:"max_age"` // in hours
	} `mapstructure:"cache"`

	// HTTP client settings
	HTTP struct {
		Timeout      time.Duration `mapstructure:"timeout"`
		UserAgent    string        `mapstructure:"user_agent"`
		MaxRedirects int           `mapstructure:"max_redirects"`
	} `mapstructure:"http"`

	// Output settings
	Output string `mapstructure:"output"`

	// Processing settings
	Tags        []string `mapstructure:"tags"`
	Concurrency int      `mapstructure:"concurrency"`

	// Logging settings
	Verbose bool `mapstructure:"verbose"`
	Debug   bool `mapstructure:"debug"`
	Quiet   bool `mapstructure:"quiet"`
	
	// Debug settings
	SaveFailedHTML bool   `mapstructure:"save_failed_html"`
	DebugOutputDir string `mapstructure:"debug_output_dir"`
}

// LoadConfig loads configuration from file and merges with command-line flags
func LoadConfig(configFile string) (*Config, error) {
	// Set defaults
	viper.SetDefault("cache.file_path", "./linkding-to-opml.gob")
	viper.SetDefault("cache.max_age", 720) // 30 days in hours
	viper.SetDefault("output", "feeds.opml")
	viper.SetDefault("concurrency", 16)
	viper.SetDefault("http.timeout", "30s")
	viper.SetDefault("http.user_agent", "Mozilla/5.0 (compatible; linkding-to-opml/1.0)")
	viper.SetDefault("http.max_redirects", 3)
	viper.SetDefault("linkding.timeout", "30s")
	viper.SetDefault("save_failed_html", false)
	viper.SetDefault("debug_output_dir", "./debug")

	// Set config file
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("linkding-to-opml")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logrus.Debug("No config file found, using defaults and command-line flags")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Bind environment variables with prefix
	viper.SetEnvPrefix("LINKDING_TO_OPML")
	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return &config, nil
}

// Validate checks that required configuration is present
func (c *Config) Validate() error {
	if c.Linkding.Token == "" {
		return fmt.Errorf("linkding token is required (set via --linkding-token flag or linkding.token in config)")
	}

	if c.Linkding.URL == "" {
		return fmt.Errorf("linkding URL is required (set via --linkding-url flag or linkding.url in config)")
	}

	return nil
}

// SetupLogging configures logrus based on the logging settings
func (c *Config) SetupLogging() {
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	if c.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else if c.Verbose {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}
}
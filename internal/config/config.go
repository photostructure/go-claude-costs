package config

import (
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	ClaudeDir string
	Days      int
	Verbose   bool
	ShowCache bool
}

// NewDefault creates a new Config with default values
func NewDefault() *Config {
	return &Config{
		Days:      30,
		Verbose:   false,
		ShowCache: false,
		ClaudeDir: getDefaultClaudeDir(),
	}
}

// Validate ensures the configuration is valid
func (c *Config) Validate() error {
	if c.Days <= 0 {
		c.Days = 30
	}

	// Ensure ClaudeDir exists
	if _, err := os.Stat(c.ClaudeDir); os.IsNotExist(err) {
		return err
	}

	return nil
}

// getDefaultClaudeDir returns the default Claude directory path
func getDefaultClaudeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude")
}

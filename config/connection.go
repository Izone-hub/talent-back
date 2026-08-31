package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	// Database
	PostgresHost     string `mapstructure:"HOST_ADDRESS"`
	PostgresPort     string `mapstructure:"HOST_PORT"`
	PostgresUser     string `mapstructure:"HOST_USERNAME"`
	PostgresPassword string `mapstructure:"HOST_PASSWORD"`
	PostgresDb       string `mapstructure:"DATABASE"`

	// GitHub OAuth
	GitHubClientID     string `mapstructure:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `mapstructure:"GITHUB_CLIENT_SECRET"`
	GitHubRedirectURL  string `mapstructure:"GITHUB_REDIRECT_URL"`

	// Admin Configuration
	AdminGitHubUsernames string `mapstructure:"ADMIN_GITHUB_USERNAMES"`

	// Security
	JWTSecret string `mapstructure:"JWT_SECRET"`

	// Server
	Port string `mapstructure:"PORT"`

	// Talent Analyzer (AI service)
	AnalyzerURL         string `mapstructure:"ANALYZER_URL"`
	InternalServiceToken string `mapstructure:"INTERNAL_SERVICE_TOKEN"`
}

// Helper method to get admin usernames as slice
func (c *Config) GetAdminUsernames() []string {
	if c.AdminGitHubUsernames == "" {
		return []string{}
	}
	// Split by comma and trim spaces
	usernames := strings.Split(c.AdminGitHubUsernames, ",")
	for i, username := range usernames {
		usernames[i] = strings.TrimSpace(username)
	}
	return usernames
}

// Helper to get database connection string
func (c *Config) GetDatabaseURL() string {
	return "postgresql://" + c.PostgresUser + ":" + c.PostgresPassword +
		"@" + c.PostgresHost + ":" + c.PostgresPort + "/" + c.PostgresDb +
		"?sslmode=disable"
}

func LoadConfig(path string) (config Config, err error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.AddConfigPath(".") // Also look in current directory

	viper.AutomaticEnv() // Read environment variables

	// Set defaults
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("HOST_ADDRESS", "localhost")
	viper.SetDefault("HOST_PORT", "5432")

	err = viper.ReadInConfig()
	if err != nil {
		// It's okay if .env doesn't exist, we'll use env vars or defaults
		log.Printf("Warning: .env file not found: %v", err)
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return config, err
	}

	// Validate required fields
	if config.GitHubClientID == "" {
		log.Fatal("GITHUB_CLIENT_ID is required")
	}
	if config.GitHubClientSecret == "" {
		log.Fatal("GITHUB_CLIENT_SECRET is required")
	}
	if config.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	return config, nil
}

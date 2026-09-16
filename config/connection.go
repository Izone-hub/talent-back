package config

import (
	"fmt"
	"log"
	"net/url"
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
	PostgresSSLMode  string `mapstructure:"DB_SSLMODE"`

	// Frontend
	FrontendURL string `mapstructure:"FRONTEND_URL"`

	// CORS
	CORSAllowedOrigins string `mapstructure:"CORS_ALLOWED_ORIGINS"`

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
	AnalyzerURL          string `mapstructure:"ANALYZER_URL"`
	InternalServiceToken string `mapstructure:"INTERNAL_SERVICE_TOKEN"`

	// Environment
	Environment string `mapstructure:"ENVIRONMENT"`
	AppEnv      string `mapstructure:"APP_ENV"`
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	env := strings.ToLower(strings.TrimSpace(c.Environment))
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(c.AppEnv))
	}
	return env == "production" || env == "prod"
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
	sslMode := c.PostgresSSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	escapedUser := url.QueryEscape(c.PostgresUser)
	escapedPass := url.QueryEscape(c.PostgresPassword)
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		escapedUser, escapedPass, c.PostgresHost, c.PostgresPort, c.PostgresDb, sslMode)
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
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("ANALYZER_URL", "http://analyzer:8000")
	viper.SetDefault("FRONTEND_URL", "http://localhost:5173")
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000")
	viper.SetDefault("ENVIRONMENT", "development")

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

	// Fail-closed enforcement in production mode
	if config.IsProduction() {
		if config.JWTSecret == "test" || config.JWTSecret == "secret" || len(config.JWTSecret) < 32 {
			log.Fatal("FATAL [Production]: JWT_SECRET must be set to a secure random string of at least 32 characters (cannot be 'test' or default).")
		}
		if config.PostgresSSLMode == "disable" || config.PostgresSSLMode == "" {
			log.Fatal("FATAL [Production]: DB_SSLMODE cannot be 'disable'. Set DB_SSLMODE=require or DB_SSLMODE=verify-full in production.")
		}
	} else {
		if config.JWTSecret == "test" || config.JWTSecret == "secret" || len(config.JWTSecret) < 16 {
			log.Println("SECURITY WARNING: JWT_SECRET is weak, too short, or using default test value! For production, configure a strong random secret.")
		}
	}

	return config, nil
}

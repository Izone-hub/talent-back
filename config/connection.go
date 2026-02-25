package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

type Config struct {
	PostgresHost     string `mapstructure:"HOST_ADDRESS"`
	PostgresPort     string `mapstructure:"HOST_PORT"`
	PostgresUser     string `mapstructure:"HOST_USERNAME"`
	PostgresPassword string `mapstructure:"HOST_PASSWORD"`
	PostgresDb       string `mapstructure:"DATABASE"`
	JWTSecret        string `mapstructure:"JWT_SECRET"`
	
	// GitHub OAuth
	GitHubClientID     string `mapstructure:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `mapstructure:"GITHUB_CLIENT_SECRET"`
	GitHubRedirectURL  string `mapstructure:"GITHUB_REDIRECT_URL"`
	FrontendURL        string `mapstructure:"FRONTEND_URL"`
}

func LoadConfig(path string) (config Config, err error) {
	// Load .env file using gotenv (this sets environment variables)
	envPath := filepath.Join(path, ".env")
	if _, err := os.Stat(envPath); err == nil {
		gotenv.Load(envPath)
	}

	// Now viper will read from environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("")

	// Explicitly bind environment variable names
	viper.BindEnv("HOST_ADDRESS")
	viper.BindEnv("HOST_PORT")
	viper.BindEnv("HOST_USERNAME")
	viper.BindEnv("HOST_PASSWORD")
	viper.BindEnv("DATABASE")
	viper.BindEnv("JWT_SECRET")
	viper.BindEnv("GITHUB_CLIENT_ID")
	viper.BindEnv("GITHUB_CLIENT_SECRET")
	viper.BindEnv("GITHUB_REDIRECT_URL")
	viper.BindEnv("FRONTEND_URL")

	err = viper.Unmarshal(&config)
	if err != nil {
		return config, err
	}
	return config, nil
}

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

	err = viper.Unmarshal(&config)
	if err != nil {
		return config, err
	}
	return config, nil
}

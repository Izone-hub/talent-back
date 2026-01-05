package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	PostgresHost     string `mapstructure:"HOST_ADDRESS"`
	PostgresPort     string `mapstructure:"HOST_PORT"`
	PostgresUser     string `mapstructure:"HOST_USERNAME"`
	PostgresPassword string `mapstructure:"HOST_PASSWORD"`
	PostgresDb       string `mapstructure:"DATABASE"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return config, err
	}
	err = viper.Unmarshal(&config)
	if err != nil {
		return config, err
	}
	return config, nil
}

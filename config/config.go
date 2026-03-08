package config

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	configDir := filepath.Join(homeDir, ".config", "universal-music-library")
	configFile := filepath.Join(configDir, "config.yaml")

	viper.SetConfigFile(configFile)

	if err := viper.ReadInConfig(); err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) || os.IsNotExist(err) {
			if err := os.MkdirAll(configDir, 0o755); err != nil {
				panic(err)
			}
			if err := viper.SafeWriteConfigAs(configFile); err != nil {
				panic(err)
			}
			slog.Info("Created config file", "path", configFile)
		} else {
			panic(err)
		}
	}
}

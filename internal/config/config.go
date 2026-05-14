package config

import (
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server  ServerConfig  `toml:"server"`
	Storage StorageConfig `toml:"storage"`
	Buffer  BufferConfig  `toml:"buffer"`
}

type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type StorageConfig struct {
	LogDir      string `toml:"log_dir"`
	MaxFileSize int64 `toml:"max_file_size"`
	MaxAgeDays  int   `toml:"max_age_days"`
}

type BufferConfig struct {
	QueueSize      int           `toml:"queue_size"`
	FlushInterval  time.Duration `toml:"flush_interval"`
}

var AppConfig *Config

func LoadConfig(path string) (*Config, error) {
	config := &Config{}

	if _, err := toml.DecodeFile(path, config); err != nil {
		return nil, err
	}

	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 29121
	}
	if config.Storage.LogDir == "" {
		config.Storage.LogDir = "./logs"
	}
	if config.Storage.MaxFileSize == 0 {
		config.Storage.MaxFileSize = 10 * 1024 * 1024
	}
	if config.Storage.MaxAgeDays == 0 {
		config.Storage.MaxAgeDays = 7
	}
	if config.Buffer.QueueSize == 0 {
		config.Buffer.QueueSize = 10000
	}
	if config.Buffer.FlushInterval == 0 {
		config.Buffer.FlushInterval = time.Second
	}

	AppConfig = config
	return config, nil
}

func GetConfig() *Config {
	return AppConfig
}

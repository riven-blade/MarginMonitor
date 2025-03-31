package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Exchange 配置结构体
type Exchange struct {
	ExchangeKey    string `yaml:"exchange_key"`
	ExchangeSecret string `yaml:"exchange_secret"`
}

// Monitor 配置结构体
type Monitor struct {
	CheckInterval   int64   `yaml:"checkInterval"`
	DangerThreshold float64 `yaml:"dangerThreshold"`
}

// Config 整体配置
type Config struct {
	Exchange Exchange `yaml:"exchange"`
	Monitor  Monitor  `yaml:"monitor"`
}

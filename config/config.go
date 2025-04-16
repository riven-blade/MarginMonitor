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
	ExchangeKey    string `yaml:"exchangeKey"`
	ExchangeSecret string `yaml:"exchangeSecret"`
	Name           string `yaml:"name"`
}

// Monitor 配置结构体
type Monitor struct {
	CheckInterval   int64   `yaml:"checkInterval"`
	DangerThreshold float64 `yaml:"dangerThreshold"`
}

// Config 整体配置
type Config struct {
	Exchange     []Exchange   `yaml:"exchange"`
	Monitor      Monitor      `yaml:"monitor"`
	Telegram     Telegram     `yaml:"telegram"`
	Proxy        string       `yaml:"proxy"`
	AddMultiple  float64      `yaml:"add_multiple"`
	RefreshPairs RefreshPairs `yaml:"refreshPairs"`
}

type Telegram struct {
	BotToken string `yaml:"bot_token"`
	ChatID   int64  `yaml:"chat_id"`
}

type RefreshPairs struct {
	Interval int         `yaml:"interval"`
	Redis    RedisConfig `yaml:"redis"`
	Bot      []Bot       `yaml:"bot"`
}

type Bot struct {
	Name          string `yaml:"name"`
	TopNum        int    `yaml:"top_num"`
	PairNum       int    `yaml:"pair_num"`
	ConfigPath    string `yaml:"config_path"`
	TopConfigPath string `yaml:"top_config_path"`
	Passwd        string `yaml:"passwd"`
	Username      string `yaml:"username"`
	ReloadAPI     string `yaml:"reload_api"`
	ReloadTopApi  string `yaml:"reload_top_api"`
}

type RedisConfig struct {
	URL          string `yaml:"url"`
	Password     string `yaml:"password"`
	DB           int    `yaml:"db"`
	PoolSize     int    `yaml:"pool_size"`
	MinIdleConns int    `yaml:"min_idle_conns"`
	MaxRetries   int    `yaml:"max_retries"`
}

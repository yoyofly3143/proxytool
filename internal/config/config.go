package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Subscription struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Config struct {
	Subscriptions []Subscription `json:"subscriptions"`
	SelectedNode  string         `json:"selected_node"`
	HTTPPort      int            `json:"http_port"`
	SocksPort     int            `json:"socks_port"`
	AllowLAN      bool           `json:"allow_lan"`
	LastUpdate    time.Time      `json:"last_update,omitempty"`
}


func DefaultConfig() *Config {
	return &Config{
		Subscriptions: []Subscription{},
		HTTPPort:      7890,
		SocksPort:     7891,
	}
}

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".proxytool")
}

func Path() string {
	return filepath.Join(Dir(), "config.json")
}

func Load() (*Config, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = 7890
	}
	if cfg.SocksPort == 0 {
		cfg.SocksPort = 7891
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	if err := os.MkdirAll(Dir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0644)
}

func (c *Config) AddSubscription(name, url string) {
	for i, s := range c.Subscriptions {
		if s.Name == name {
			c.Subscriptions[i].URL = url
			return
		}
	}
	c.Subscriptions = append(c.Subscriptions, Subscription{Name: name, URL: url})
}

func (c *Config) RemoveSubscription(name string) bool {
	for i, s := range c.Subscriptions {
		if s.Name == name {
			c.Subscriptions = append(c.Subscriptions[:i], c.Subscriptions[i+1:]...)
			return true
		}
	}
	return false
}

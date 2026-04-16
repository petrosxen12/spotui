package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	AppName            = "spotui"
	DefaultRedirectURI = "http://127.0.0.1:8888/callback"
)

type Config struct {
	ClientID          string     `json:"client_id"`
	RedirectURI       string     `json:"redirect_uri"`
	PreferredDeviceID string     `json:"preferred_device_id"`
	LastSearch        LastSearch `json:"last_search"`
}

type LastSearch struct {
	Query      string       `json:"query"`
	Tracks     []SearchItem `json:"tracks"`
	Playlists  []SearchItem `json:"playlists"`
	SearchedAt time.Time    `json:"searched_at"`
}

type SearchItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"uri"`
}

func Load() (*Config, error) {
	cfg := &Config{
		ClientID:    strings.TrimSpace(os.Getenv("SPOTUI_CLIENT_ID")),
		RedirectURI: strings.TrimSpace(os.Getenv("SPOTUI_REDIRECT_URI")),
	}
	if cfg.RedirectURI == "" {
		cfg.RedirectURI = DefaultRedirectURI
	}

	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if envClientID := strings.TrimSpace(os.Getenv("SPOTUI_CLIENT_ID")); envClientID != "" {
		cfg.ClientID = envClientID
	}
	if envRedirectURI := strings.TrimSpace(os.Getenv("SPOTUI_REDIRECT_URI")); envRedirectURI != "" {
		cfg.RedirectURI = envRedirectURI
	}
	if cfg.RedirectURI == "" {
		cfg.RedirectURI = DefaultRedirectURI
	}
	return cfg, nil
}

func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

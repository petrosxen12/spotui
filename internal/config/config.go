package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	AppName            = "spotui"
	DefaultRedirectURI = "http://127.0.0.1:8888/callback"
)

type Config struct {
	ClientID          string            `json:"client_id"`
	RedirectURI       string            `json:"redirect_uri"`
	PreferredDeviceID string            `json:"preferred_device_id"`
	LastUsedDevice    LastDevice        `json:"last_used_device"`
	LastSearch        LastSearch        `json:"last_search"`
	LocalPlayer       LocalPlayerConfig `json:"local_player"`
}

type LastDevice struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	SeenAt   time.Time `json:"seen_at"`
	Selected bool      `json:"selected"`
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

type LocalPlayerConfig struct {
	Enabled                bool   `json:"enabled"`
	DeviceName             string `json:"device_name"`
	Backend                string `json:"backend"`
	AudioDevice            string `json:"audio_device"`
	Bitrate                int    `json:"bitrate"`
	InitialVolume          int    `json:"initial_volume"`
	SpotifydPath           string `json:"spotifyd_path"`
	AutostartPromptEnabled bool   `json:"autostart_prompt_enabled"`
}

func (cfg *LocalPlayerConfig) UnmarshalJSON(data []byte) error {
	type rawLocalPlayerConfig struct {
		Enabled                *bool  `json:"enabled"`
		DeviceName             string `json:"device_name"`
		Backend                string `json:"backend"`
		AudioDevice            string `json:"audio_device"`
		Bitrate                int    `json:"bitrate"`
		InitialVolume          int    `json:"initial_volume"`
		SpotifydPath           string `json:"spotifyd_path"`
		AutostartPromptEnabled *bool  `json:"autostart_prompt_enabled"`
	}

	defaults := DefaultLocalPlayerConfig()
	raw := rawLocalPlayerConfig{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*cfg = defaults
	if raw.Enabled != nil {
		cfg.Enabled = *raw.Enabled
	}
	if raw.DeviceName != "" {
		cfg.DeviceName = raw.DeviceName
	}
	if raw.Backend != "" {
		cfg.Backend = raw.Backend
	}
	cfg.AudioDevice = raw.AudioDevice
	cfg.Bitrate = raw.Bitrate
	cfg.InitialVolume = raw.InitialVolume
	cfg.SpotifydPath = raw.SpotifydPath
	if raw.AutostartPromptEnabled != nil {
		cfg.AutostartPromptEnabled = *raw.AutostartPromptEnabled
	}
	cfg.ApplyDefaults()
	return nil
}

func DefaultLocalPlayerConfig() LocalPlayerConfig {
	return LocalPlayerConfig{
		Enabled:                false,
		DeviceName:             "spotui",
		Backend:                "portaudio",
		Bitrate:                320,
		InitialVolume:          100,
		AutostartPromptEnabled: true,
	}
}

func (cfg *LocalPlayerConfig) ApplyDefaults() {
	defaults := DefaultLocalPlayerConfig()
	if cfg.DeviceName == "" {
		cfg.DeviceName = defaults.DeviceName
	}
	if cfg.Backend == "" {
		cfg.Backend = defaults.Backend
	}
	switch cfg.Bitrate {
	case 96, 160, 320:
	default:
		cfg.Bitrate = defaults.Bitrate
	}
	if cfg.InitialVolume < 0 || cfg.InitialVolume > 100 {
		cfg.InitialVolume = defaults.InitialVolume
	}
}

func Load() (*Config, error) {
	cfg := &Config{
		ClientID:    strings.TrimSpace(os.Getenv("SPOTUI_CLIENT_ID")),
		RedirectURI: strings.TrimSpace(os.Getenv("SPOTUI_REDIRECT_URI")),
		LocalPlayer: DefaultLocalPlayerConfig(),
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
	cfg.LocalPlayer.ApplyDefaults()
	return cfg, nil
}

func Save(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is required")
	}
	cfg.LocalPlayer.ApplyDefaults()
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := WriteSecureFile(path, append(data, '\n')); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

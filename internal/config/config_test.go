package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesLocalPlayerDefaultsWhenAbsent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.LocalPlayer.DeviceName != "spotui" {
		t.Fatalf("DeviceName = %q, want %q", cfg.LocalPlayer.DeviceName, "spotui")
	}
	if cfg.LocalPlayer.Backend != "portaudio" {
		t.Fatalf("Backend = %q, want %q", cfg.LocalPlayer.Backend, "portaudio")
	}
	if cfg.LocalPlayer.Bitrate != 320 {
		t.Fatalf("Bitrate = %d, want 320", cfg.LocalPlayer.Bitrate)
	}
	if !cfg.LocalPlayer.AutostartPromptEnabled {
		t.Fatal("AutostartPromptEnabled = false, want true")
	}
}

func TestLoadPreservesExplicitFalseLocalPlayerFlags(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	data := []byte("{\"local_player\":{\"enabled\":false,\"autostart_prompt_enabled\":false}}\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.LocalPlayer.Enabled {
		t.Fatal("Enabled = true, want false")
	}
	if cfg.LocalPlayer.AutostartPromptEnabled {
		t.Fatal("AutostartPromptEnabled = true, want false")
	}
}

func TestSaveRoundTripsLocalPlayerConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg := &Config{
		ClientID:    "client",
		RedirectURI: DefaultRedirectURI,
		LocalPlayer: LocalPlayerConfig{
			Enabled:                true,
			DeviceName:             "desk-speakers",
			Backend:                "alsa",
			AudioDevice:            "hw:0,0",
			Bitrate:                160,
			InitialVolume:          55,
			SpotifydPath:           "/usr/local/bin/spotifyd",
			AutostartPromptEnabled: false,
		},
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !loaded.LocalPlayer.Enabled {
		t.Fatal("Enabled = false, want true")
	}
	if loaded.LocalPlayer.DeviceName != "desk-speakers" {
		t.Fatalf("DeviceName = %q, want %q", loaded.LocalPlayer.DeviceName, "desk-speakers")
	}
	if loaded.LocalPlayer.Backend != "alsa" {
		t.Fatalf("Backend = %q, want %q", loaded.LocalPlayer.Backend, "alsa")
	}
	if loaded.LocalPlayer.AudioDevice != "hw:0,0" {
		t.Fatalf("AudioDevice = %q, want %q", loaded.LocalPlayer.AudioDevice, "hw:0,0")
	}
	if loaded.LocalPlayer.Bitrate != 160 {
		t.Fatalf("Bitrate = %d, want 160", loaded.LocalPlayer.Bitrate)
	}
	if loaded.LocalPlayer.InitialVolume != 55 {
		t.Fatalf("InitialVolume = %d, want 55", loaded.LocalPlayer.InitialVolume)
	}
	if loaded.LocalPlayer.SpotifydPath != "/usr/local/bin/spotifyd" {
		t.Fatalf("SpotifydPath = %q, want %q", loaded.LocalPlayer.SpotifydPath, "/usr/local/bin/spotifyd")
	}
	if loaded.LocalPlayer.AutostartPromptEnabled {
		t.Fatal("AutostartPromptEnabled = true, want false")
	}
}

package spotifyd

import (
	"strings"
	"testing"

	"github.com/petrosxen/spotui/internal/config"
)

func TestGenerateConfigIncludesManagedFields(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	manager, err := NewManager(config.LocalPlayerConfig{
		Enabled:       true,
		DeviceName:    "office",
		Backend:       "alsa",
		AudioDevice:   "hw:0,0",
		Bitrate:       160,
		InitialVolume: 75,
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	data, err := manager.GenerateConfig()
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}
	text := string(data)

	for _, want := range []string{
		"[global]",
		`device_name = "office"`,
		`backend = "alsa"`,
		`device = "hw:0,0"`,
		"bitrate = 160",
		"initial_volume = 75",
		`cache_path = "`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated config missing %q:\n%s", want, text)
		}
	}
}

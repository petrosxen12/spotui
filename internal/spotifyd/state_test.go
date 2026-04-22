package spotifyd

import (
	"strings"
	"testing"
	"time"

	"github.com/petrosxen/spotui/internal/config"
)

func TestLoadRuntimeStateRejectsPIDMismatch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	manager, err := NewManager(config.LocalPlayerConfig{})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	state := State{
		PID:        41,
		BinaryPath: "/usr/bin/spotifyd",
		ConfigPath: manager.Files().ConfigPath,
		LogPath:    manager.Files().LogPath,
		DeviceName: "spotui",
		Backend:    "pulseaudio",
		StartedAt:  time.Now().UTC(),
	}
	if err := manager.writeRuntimeState(state); err != nil {
		t.Fatalf("writeRuntimeState() error = %v", err)
	}
	if err := config.WriteSecureFile(manager.Files().PIDPath, []byte("42\n")); err != nil {
		t.Fatalf("WriteSecureFile() error = %v", err)
	}

	_, _, err = manager.loadRuntimeState()
	if err == nil {
		t.Fatal("loadRuntimeState() error = nil, want mismatch error")
	}
	if !strings.Contains(err.Error(), "pid mismatch") {
		t.Fatalf("loadRuntimeState() error = %q, want mismatch error", err)
	}
}

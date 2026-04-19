package spotifyd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestStatusCleansStaleRuntimeMetadata(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	manager, err := NewManager(config.LocalPlayerConfig{})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	state := State{
		PID:        999999,
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

	status, err := manager.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.ProcessState != ProcessStateStopped {
		t.Fatalf("ProcessState = %q, want %q", status.ProcessState, ProcessStateStopped)
	}
	if _, err := os.Stat(manager.Files().PIDPath); !os.IsNotExist(err) {
		t.Fatalf("PIDPath still exists, err = %v", err)
	}
	if _, err := os.Stat(manager.Files().StatePath); !os.IsNotExist(err) {
		t.Fatalf("StatePath still exists, err = %v", err)
	}
}

func TestStartAndStopManagedSpotifydProcess(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	binary := writeFakeSpotifyd(t)
	manager, err := NewManager(config.LocalPlayerConfig{
		Enabled:      true,
		DeviceName:   "desk",
		Backend:      "pulseaudio",
		SpotifydPath: binary,
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	status, err := manager.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if status.PID == 0 {
		t.Fatal("Start() returned pid 0")
	}
	if status.ProcessState != ProcessStateRunning && status.ProcessState != ProcessStateStarting {
		t.Fatalf("ProcessState = %q, want running or starting", status.ProcessState)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, err = manager.Status(context.Background())
		if err != nil {
			t.Fatalf("Status() error = %v", err)
		}
		if status.ProcessState == ProcessStateRunning {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if status.ProcessState != ProcessStateRunning {
		t.Fatalf("final ProcessState = %q, want %q", status.ProcessState, ProcessStateRunning)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := manager.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	stopped, err := manager.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() after stop error = %v", err)
	}
	if stopped.ProcessState != ProcessStateStopped {
		t.Fatalf("ProcessState after stop = %q, want %q", stopped.ProcessState, ProcessStateStopped)
	}
}

func writeFakeSpotifyd(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "spotifyd")
	script := `#!/bin/sh
exec -a spotifyd /bin/sh -c 'trap "exit 0" TERM INT; while :; do sleep 1; done' spotifyd "$@"
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

package spotifyd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/petrosxen/spotui/internal/config"
)

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

func TestResetStopsMismatchedSpotifydProcessAndCleansMetadata(t *testing.T) {
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

	cmd := exec.Command(binary, "--no-daemon", "--config-path", filepath.Join(t.TempDir(), "other.conf"))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start manual spotifyd error = %v", err)
	}
	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	state := State{
		PID:        pid,
		BinaryPath: binary,
		ConfigPath: manager.Files().ConfigPath,
		LogPath:    manager.Files().LogPath,
		DeviceName: "desk",
		Backend:    "pulseaudio",
		StartedAt:  time.Now().UTC(),
	}
	if err := manager.writeRuntimeState(state); err != nil {
		t.Fatalf("writeRuntimeState() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := manager.Reset(ctx); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	alive, err := processAlive(pid)
	if err != nil {
		t.Fatalf("processAlive() error = %v", err)
	}
	if alive {
		t.Fatal("expected reset to terminate mismatched spotifyd process")
	}
	if _, err := os.Stat(manager.Files().PIDPath); !os.IsNotExist(err) {
		t.Fatalf("PIDPath still exists, err = %v", err)
	}
	if _, err := os.Stat(manager.Files().StatePath); !os.IsNotExist(err) {
		t.Fatalf("StatePath still exists, err = %v", err)
	}
}

func TestStartReturnsLogExcerptWhenSpotifydExitsImmediately(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	binary := filepath.Join(t.TempDir(), "spotifyd")
	script := `#!/bin/sh
echo "Error: invalid backend" >&2
exit 1
`
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manager, err := NewManager(config.LocalPlayerConfig{
		Enabled:      true,
		DeviceName:   "desk",
		Backend:      "pulseaudio",
		SpotifydPath: binary,
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	_, err = manager.Start(context.Background())
	if err == nil {
		t.Fatal("Start() error = nil, want startup failure")
	}
	if !strings.Contains(err.Error(), "spotifyd exited during startup") {
		t.Fatalf("Start() error = %q, want startup failure message", err)
	}
	if !strings.Contains(err.Error(), "Error: invalid backend") {
		t.Fatalf("Start() error = %q, want log excerpt", err)
	}
}

func writeFakeSpotifyd(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "spotifyd")
	script := `#!/bin/sh
trap "exit 0" TERM INT
while :; do
  sleep 1
done
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

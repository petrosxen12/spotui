package spotifyd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/petrosxen/spotui/internal/config"
)

type Manager struct {
	cfg   config.LocalPlayerConfig
	files RuntimeFiles
}

func NewManager(cfg config.LocalPlayerConfig) (*Manager, error) {
	files, err := runtimeFiles()
	if err != nil {
		return nil, err
	}
	cfg.ApplyDefaults()
	return &Manager{
		cfg:   cfg,
		files: files,
	}, nil
}

func (m *Manager) Files() RuntimeFiles {
	return m.files
}

func (m *Manager) Start(ctx context.Context) (Status, error) {
	if err := m.cleanupStaleIfStopped(); err != nil {
		return Status{}, err
	}
	current, err := m.Status(ctx)
	if err != nil {
		return Status{}, err
	}
	if current.ProcessState == ProcessStateRunning {
		return current, nil
	}
	if current.ProcessState == ProcessStateUnhealthy {
		return current, errors.New(current.Message)
	}

	binaryPath, err := m.ResolveBinary()
	if err != nil {
		return Status{}, err
	}
	if err := m.WriteConfig(); err != nil {
		return Status{}, err
	}

	logFile, err := os.OpenFile(m.files.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Status{}, fmt.Errorf("open spotifyd log: %w", err)
	}
	defer logFile.Close()

	cmd := exec.CommandContext(ctx, binaryPath, "--no-daemon", "--config-path", m.files.ConfigPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return Status{}, fmt.Errorf("start spotifyd: %w", err)
	}

	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		return Status{}, fmt.Errorf("detach spotifyd process: %w", err)
	}

	state := State{
		PID:         pid,
		BinaryPath:  binaryPath,
		ConfigPath:  m.files.ConfigPath,
		LogPath:     m.files.LogPath,
		DeviceName:  m.cfg.DeviceName,
		Backend:     m.cfg.Backend,
		AudioDevice: m.cfg.AudioDevice,
		StartedAt:   time.Now().UTC(),
	}
	if err := m.writeRuntimeState(state); err != nil {
		return Status{}, err
	}

	return m.verifyStartup(ctx, pid, binaryPath, state.StartedAt)
}

func (m *Manager) Stop(ctx context.Context) error {
	status, err := m.Status(ctx)
	if err != nil {
		return err
	}
	if status.PID == 0 {
		return nil
	}
	if status.ProcessState == ProcessStateUnhealthy {
		return errors.New("refusing to stop unmanaged process referenced by spotifyd runtime metadata")
	}
	if err := terminateProcess(ctx, status.PID); err != nil {
		return err
	}
	return m.cleanupStaleRuntimeState()
}

func (m *Manager) Reset(ctx context.Context) error {
	state, _, err := m.loadRuntimeState()
	if err != nil {
		return err
	}
	if state == nil || state.PID == 0 {
		return m.cleanupStaleRuntimeState()
	}

	alive, err := processAlive(state.PID)
	if err != nil {
		return err
	}
	if !alive {
		return m.cleanupStaleRuntimeState()
	}

	matches, err := matchesManagedCommand(state.PID, *state)
	if err != nil {
		return err
	}
	if matches {
		return m.Stop(ctx)
	}

	isSpotifyd, err := processLooksLikeSpotifyd(state.PID, state.BinaryPath)
	if err != nil {
		return err
	}
	if !isSpotifyd {
		return errors.New("refusing to reset runtime metadata that points to a non-spotifyd live process")
	}

	if err := terminateProcess(ctx, state.PID); err != nil {
		return err
	}
	return m.cleanupStaleRuntimeState()
}

func (m *Manager) cleanupStaleIfStopped() error {
	state, _, err := m.loadRuntimeState()
	if err != nil {
		return err
	}
	if state == nil {
		return nil
	}
	alive, matches, err := m.inspectProcess(*state)
	if err != nil {
		return err
	}
	if alive && !matches {
		return errors.New("spotifyd runtime metadata points to a different live process")
	}
	if alive {
		return nil
	}
	return m.cleanupStaleRuntimeState()
}

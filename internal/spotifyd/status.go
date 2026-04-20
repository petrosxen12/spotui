package spotifyd

import (
	"context"
	"errors"
	"time"
)

func (m *Manager) Status(_ context.Context) (Status, error) {
	status := Status{
		ProcessState: ProcessStateStopped,
		DeviceName:   m.cfg.DeviceName,
		Backend:      m.cfg.Backend,
		AudioDevice:  m.cfg.AudioDevice,
		RuntimeFiles: m.files,
	}

	binaryPath, err := m.ResolveBinary()
	if err == nil {
		status.BinaryAvailable = true
		status.BinaryPath = binaryPath
	} else if !errors.Is(err, ErrBinaryNotFound) {
		return status, err
	} else {
		status.Message = err.Error()
	}

	state, pid, err := m.loadRuntimeState()
	if err != nil {
		return status, err
	}
	if pid == 0 && state == nil {
		return status, nil
	}
	if state == nil {
		if err := m.cleanupStaleRuntimeState(); err != nil {
			return status, err
		}
		status.Message = "removed stale spotifyd pid metadata"
		return status, nil
	}

	status.PID = state.PID
	status.StartedAt = state.StartedAt
	if state.BinaryPath != "" {
		status.BinaryPath = state.BinaryPath
	}
	if !status.BinaryAvailable && status.BinaryPath != "" {
		status.BinaryAvailable = isExecutableFile(status.BinaryPath)
	}

	alive, matches, err := m.inspectProcess(*state)
	if err != nil {
		return status, err
	}
	switch {
	case alive && matches:
		status.ProcessState = ProcessStateRunning
	case alive && !matches:
		status.ProcessState = ProcessStateUnhealthy
		status.Message = "spotifyd pid belongs to a different process"
	case !alive:
		if err := m.cleanupStaleRuntimeState(); err != nil {
			return status, err
		}
		status.PID = 0
		status.StartedAt = time.Time{}
		status.Message = "removed stale spotifyd runtime metadata"
	}
	return status, nil
}

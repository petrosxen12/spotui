package spotifyd

import (
	"errors"
	"time"
)

var ErrBinaryNotFound = errors.New("spotifyd binary not found")

type ProcessState string

const (
	ProcessStateStopped   ProcessState = "stopped"
	ProcessStateStarting  ProcessState = "starting"
	ProcessStateRunning   ProcessState = "running"
	ProcessStateUnhealthy ProcessState = "unhealthy"
)

type RuntimeFiles struct {
	ConfigPath string
	PIDPath    string
	LogPath    string
	StatePath  string
	CacheDir   string
}

type Status struct {
	BinaryPath      string
	BinaryAvailable bool
	ProcessState    ProcessState
	PID             int
	DeviceName      string
	Backend         string
	AudioDevice     string
	StartedAt       time.Time
	RuntimeFiles    RuntimeFiles
	Message         string
}

type State struct {
	PID         int       `json:"pid"`
	BinaryPath  string    `json:"binary_path"`
	ConfigPath  string    `json:"config_path"`
	LogPath     string    `json:"log_path"`
	DeviceName  string    `json:"device_name"`
	Backend     string    `json:"backend"`
	AudioDevice string    `json:"audio_device"`
	StartedAt   time.Time `json:"started_at"`
}

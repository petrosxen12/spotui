package spotifyd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/petrosxen/spotui/internal/config"
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

func runtimeFiles() (RuntimeFiles, error) {
	configPath, err := config.SpotifydConfigPath()
	if err != nil {
		return RuntimeFiles{}, err
	}
	pidPath, err := config.SpotifydPIDPath()
	if err != nil {
		return RuntimeFiles{}, err
	}
	logPath, err := config.SpotifydLogPath()
	if err != nil {
		return RuntimeFiles{}, err
	}
	statePath, err := config.SpotifydStatePath()
	if err != nil {
		return RuntimeFiles{}, err
	}
	cacheDir, err := config.SpotifydCacheDir()
	if err != nil {
		return RuntimeFiles{}, err
	}
	return RuntimeFiles{
		ConfigPath: configPath,
		PIDPath:    pidPath,
		LogPath:    logPath,
		StatePath:  statePath,
		CacheDir:   cacheDir,
	}, nil
}

func (m *Manager) Files() RuntimeFiles {
	return m.files
}

func (m *Manager) ResolveBinary() (string, error) {
	if override := strings.TrimSpace(m.cfg.SpotifydPath); override != "" {
		if isExecutableFile(override) {
			return override, nil
		}
		return "", fmt.Errorf("%w: %s", ErrBinaryNotFound, override)
	}
	path, err := exec.LookPath("spotifyd")
	if err != nil {
		return "", ErrBinaryNotFound
	}
	return path, nil
}

func (m *Manager) GenerateConfig() ([]byte, error) {
	lines := []string{
		"[global]",
		"device_name = " + strconv.Quote(m.cfg.DeviceName),
		"backend = " + strconv.Quote(m.cfg.Backend),
		"bitrate = " + strconv.Itoa(m.cfg.Bitrate),
		"initial_volume = " + strconv.Itoa(m.cfg.InitialVolume),
		"cache_path = " + strconv.Quote(m.files.CacheDir),
	}
	if strings.TrimSpace(m.cfg.AudioDevice) != "" {
		lines = append(lines, "device = "+strconv.Quote(strings.TrimSpace(m.cfg.AudioDevice)))
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

func (m *Manager) WriteConfig() error {
	if err := os.MkdirAll(filepath.Dir(m.files.ConfigPath), 0o700); err != nil {
		return fmt.Errorf("create spotifyd runtime dir: %w", err)
	}
	if err := os.MkdirAll(m.files.CacheDir, 0o700); err != nil {
		return fmt.Errorf("create spotifyd cache dir: %w", err)
	}
	data, err := m.GenerateConfig()
	if err != nil {
		return err
	}
	if err := config.WriteSecureFile(m.files.ConfigPath, data); err != nil {
		return fmt.Errorf("write spotifyd config: %w", err)
	}
	return nil
}

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

	// Give spotifyd a brief window to reject an invalid config before we report success.
	time.Sleep(250 * time.Millisecond)
	status, err := m.Status(ctx)
	if err != nil {
		return Status{}, err
	}
	if status.ProcessState == ProcessStateStopped {
		return Status{}, m.startupFailure()
	}
	if status.ProcessState == ProcessStateStarting {
		status.PID = pid
		status.StartedAt = state.StartedAt
		status.BinaryPath = binaryPath
		status.BinaryAvailable = true
		status.RuntimeFiles = m.files
	}
	return status, nil
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

func (m *Manager) loadRuntimeState() (*State, int, error) {
	pid, err := readPIDFile(m.files.PIDPath)
	if err != nil {
		return nil, 0, err
	}
	data, err := os.ReadFile(m.files.StatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, pid, nil
		}
		return nil, 0, fmt.Errorf("read spotifyd state: %w", err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, 0, fmt.Errorf("parse spotifyd state: %w", err)
	}
	if state.PID == 0 {
		state.PID = pid
	}
	if pid != 0 && state.PID != pid {
		return nil, 0, fmt.Errorf("spotifyd runtime metadata pid mismatch: pid file=%d state=%d", pid, state.PID)
	}
	return &state, state.PID, nil
}

func (m *Manager) writeRuntimeState(state State) error {
	if err := config.WriteSecureFile(m.files.PIDPath, []byte(strconv.Itoa(state.PID)+"\n")); err != nil {
		return fmt.Errorf("write spotifyd pid: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode spotifyd state: %w", err)
	}
	if err := config.WriteSecureFile(m.files.StatePath, append(data, '\n')); err != nil {
		return fmt.Errorf("write spotifyd state: %w", err)
	}
	return nil
}

func (m *Manager) cleanupStaleRuntimeState() error {
	for _, path := range []string{m.files.PIDPath, m.files.StatePath} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove stale runtime file %s: %w", path, err)
		}
	}
	return nil
}

func (m *Manager) inspectProcess(state State) (alive bool, matches bool, err error) {
	alive, err = processAlive(state.PID)
	if err != nil || !alive {
		return alive, false, err
	}
	matches, err = matchesManagedCommand(state.PID, state)
	return alive, matches, err
}

func matchesManagedCommand(pid int, state State) (bool, error) {
	cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read spotifyd cmdline: %w", err)
	}
	args := parseCmdline(data)
	if len(args) == 0 {
		return false, nil
	}

	binaryBase := filepath.Base(state.BinaryPath)
	if binaryBase == "" {
		binaryBase = "spotifyd"
	}
	if !hasExpectedBinaryArg(args, binaryBase, state.BinaryPath) {
		return false, nil
	}

	for i := 0; i < len(args); i++ {
		if args[i] == "--config-path" && i+1 < len(args) && args[i+1] == state.ConfigPath {
			return true, nil
		}
		if strings.HasPrefix(args[i], "--config-path=") && strings.TrimPrefix(args[i], "--config-path=") == state.ConfigPath {
			return true, nil
		}
	}
	return false, nil
}

func hasExpectedBinaryArg(args []string, binaryBase, binaryPath string) bool {
	for _, arg := range args {
		if filepath.Base(arg) == binaryBase {
			return true
		}
		if binaryPath != "" && arg == binaryPath {
			return true
		}
	}
	return false
}

func parseCmdline(data []byte) []string {
	parts := strings.Split(string(data), "\x00")
	args := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		args = append(args, part)
	}
	return args
}

func readPIDFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("read spotifyd pid: %w", err)
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return 0, nil
	}
	pid, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse spotifyd pid: %w", err)
	}
	return pid, nil
}

func processAlive(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return !processZombie(pid), nil
	}
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}
	return false, fmt.Errorf("inspect process %d: %w", pid, err)
}

func processZombie(pid int) bool {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return false
	}
	parts := strings.Fields(string(data))
	return len(parts) > 2 && parts[2] == "Z"
}

func processLooksLikeSpotifyd(pid int, binaryPath string) (bool, error) {
	args, err := readCmdlineArgs(pid)
	if err != nil {
		return false, err
	}
	if len(args) == 0 {
		return false, nil
	}

	binaryBase := filepath.Base(binaryPath)
	if binaryBase == "" {
		binaryBase = "spotifyd"
	}
	return hasExpectedBinaryArg(args, binaryBase, binaryPath), nil
}

func terminateProcess(ctx context.Context, pid int) error {
	if err := signalProcessGroup(pid, syscall.SIGTERM); err != nil {
		return err
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		alive, err := processAlive(pid)
		if err != nil {
			return err
		}
		if !alive {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err := signalProcessGroup(pid, syscall.SIGKILL); err != nil {
		return err
	}
	return nil
}

func signalProcessGroup(pid int, signal syscall.Signal) error {
	if pid <= 0 {
		return nil
	}
	err := syscall.Kill(-pid, signal)
	if err == nil || errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return fmt.Errorf("%s spotifyd: %w", signalLabel(signal), err)
}

func signalLabel(signal syscall.Signal) string {
	switch signal {
	case syscall.SIGTERM:
		return "terminate"
	case syscall.SIGKILL:
		return "kill"
	default:
		return "signal"
	}
}

func (m *Manager) startupFailure() error {
	excerpt := tailLog(m.files.LogPath, 12)
	if excerpt == "" {
		return fmt.Errorf("spotifyd exited during startup; check %s", m.files.LogPath)
	}
	return fmt.Errorf("spotifyd exited during startup; check %s\n%s", m.files.LogPath, excerpt)
}

func tailLog(path string, maxLines int) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	lines := make([]string, 0, maxLines)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > maxLines {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func readCmdlineArgs(pid int) ([]string, error) {
	cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read spotifyd cmdline: %w", err)
	}
	return parseCmdline(data), nil
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode().Perm()&0o111 != 0
}

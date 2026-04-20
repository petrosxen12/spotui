package spotifyd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func (m *Manager) inspectProcess(state State) (alive bool, matches bool, err error) {
	alive, err = processAlive(state.PID)
	if err != nil || !alive {
		return alive, false, err
	}
	matches, err = matchesManagedCommand(state.PID, state)
	return alive, matches, err
}

func matchesManagedCommand(pid int, state State) (bool, error) {
	args, err := readCmdlineArgs(pid)
	if err != nil {
		return false, err
	}
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

package spotifyd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/petrosxen/spotui/internal/config"
)

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

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode().Perm()&0o111 != 0
}

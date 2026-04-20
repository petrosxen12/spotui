package spotifyd

import "github.com/petrosxen/spotui/internal/config"

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

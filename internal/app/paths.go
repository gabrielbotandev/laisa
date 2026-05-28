package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DataDir returns the application data directory.
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "shai"), nil
	case "linux":
		if v := os.Getenv("XDG_DATA_HOME"); v != "" {
			return filepath.Join(v, "shai"), nil
		}
		return filepath.Join(home, ".local", "share", "shai"), nil
	default:
		return "", fmt.Errorf("unsupported OS: %s (Linux and macOS only)", runtime.GOOS)
	}
}

// ConfigDir returns the configuration directory.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "shai"), nil
	case "linux":
		if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
			return filepath.Join(v, "shai"), nil
		}
		return filepath.Join(home, ".config", "shai"), nil
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// CacheDir returns the cache directory.
func CacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Caches", "shai"), nil
	case "linux":
		if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
			return filepath.Join(v, "shai"), nil
		}
		return filepath.Join(home, ".cache", "shai"), nil
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// ModelsDir returns the local models directory.
func ModelsDir() (string, error) {
	data, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(data, "models"), nil
}

// ConfigPath returns the YAML config file path.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// VenvDir returns the Python virtual environment directory.
func VenvDir() (string, error) {
	data, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(data, ".venv"), nil
}

// VenvPython returns the path to the venv Python interpreter.
func VenvPython() (string, error) {
	venv, err := VenvDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(venv, "Scripts", "python.exe"), nil
	}
	return filepath.Join(venv, "bin", "python"), nil
}

// BackendDir returns the backend scripts directory.
func BackendDir() (string, error) {
	data, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(data, "backend"), nil
}

// BackendRunner returns the path to runner.py on disk.
func BackendRunner() (string, error) {
	dir, err := BackendDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "runner.py"), nil
}

// BackendRequirements returns the path to requirements.txt on disk.
func BackendRequirements() (string, error) {
	dir, err := BackendDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "requirements.txt"), nil
}

// EnsureDirs creates data, config, cache, and models directories.
func EnsureDirs() error {
	data, err := DataDir()
	if err != nil {
		return err
	}
	cfg, err := ConfigDir()
	if err != nil {
		return err
	}
	cache, err := CacheDir()
	if err != nil {
		return err
	}
	models, err := ModelsDir()
	if err != nil {
		return err
	}
	backend, err := BackendDir()
	if err != nil {
		return err
	}

	for _, d := range []string{data, cfg, cache, models, backend} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", d, err)
		}
	}
	return nil
}

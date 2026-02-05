package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	configDirName  = ".cli-notes"
	configFileName = "config.json"
)

var ErrNotConfigured = errors.New("cli-notes is not configured")

// Config stores user-defined CLI Notes settings.
type Config struct {
	NotesDir string `json:"notes_dir"`
}

// DefaultNotesDir returns the default notes directory used by the configurator.
func DefaultNotesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "notes"), nil
}

// ConfigPath returns the configuration file path.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName, configFileName), nil
}

// Exists reports whether the config file exists.
func Exists() (bool, error) {
	path, err := ConfigPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// Load reads and validates the saved configuration.
func Load() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, ErrNotConfigured
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	notesDir, err := NormalizeNotesDir(cfg.NotesDir)
	if err != nil {
		return Config{}, fmt.Errorf("invalid notes_dir: %w", err)
	}
	cfg.NotesDir = notesDir

	return cfg, nil
}

// Save writes configuration to disk.
func Save(cfg Config) error {
	notesDir, err := NormalizeNotesDir(cfg.NotesDir)
	if err != nil {
		return fmt.Errorf("invalid notes_dir: %w", err)
	}
	cfg.NotesDir = notesDir

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(path, data, 0o600)
}

// NormalizeNotesDir expands and normalizes a notes directory path.
func NormalizeNotesDir(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("path is required")
	}

	expanded, err := expandHome(trimmed)
	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}

	return filepath.Clean(abs), nil
}

func expandHome(path string) (string, error) {
	if path == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

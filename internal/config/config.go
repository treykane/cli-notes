package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/treykane/cli-notes/internal/logging"
)

const (
	configDirName  = ".cli-notes"
	configFileName = "config.json"
)

var ErrNotConfigured = errors.New("cli-notes is not configured")
var log = logging.New("config")

// Config stores user-defined CLI Notes settings.
type Config struct {
	NotesDir        string            `json:"notes_dir,omitempty"`
	TreeSort        string            `json:"tree_sort,omitempty"`
	TemplatesDir    string            `json:"templates_dir,omitempty"`
	Workspaces      []WorkspaceConfig `json:"workspaces,omitempty"`
	ActiveWorkspace string            `json:"active_workspace,omitempty"`
	Keybindings     map[string]string `json:"keybindings,omitempty"`
	KeymapFile      string            `json:"keymap_file,omitempty"`
}

type WorkspaceConfig struct {
	Name     string `json:"name"`
	NotesDir string `json:"notes_dir"`
}

// DefaultNotesDir returns the default notes directory used by the configurator.
func DefaultNotesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, "notes"), nil
}

// DefaultTemplatesDir returns the default templates directory.
func DefaultTemplatesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, configDirName, "templates"), nil
}

// DefaultKeymapPath returns the default keymap file path.
func DefaultKeymapPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, configDirName, "keymap.json"), nil
}

// ConfigPath returns the configuration file path.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
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
	return false, fmt.Errorf("stat config path %q: %w", path, err)
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
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	legacyNotesDir := strings.TrimSpace(cfg.NotesDir)
	if legacyNotesDir != "" {
		notesDir, normErr := NormalizeNotesDir(legacyNotesDir)
		if normErr != nil {
			return Config{}, fmt.Errorf("invalid notes_dir: %w", normErr)
		}
		cfg.NotesDir = notesDir
	}
	cfg.TreeSort = strings.TrimSpace(strings.ToLower(cfg.TreeSort))
	if cfg.TreeSort == "" {
		cfg.TreeSort = "name"
	}

	templatesDir := strings.TrimSpace(cfg.TemplatesDir)
	if templatesDir == "" {
		templatesDir, err = DefaultTemplatesDir()
		if err != nil {
			return Config{}, err
		}
	}
	templatesDir, err = NormalizeNotesDir(templatesDir)
	if err != nil {
		return Config{}, fmt.Errorf("invalid templates_dir: %w", err)
	}
	cfg.TemplatesDir = templatesDir
	keymapPath := strings.TrimSpace(cfg.KeymapFile)
	if keymapPath == "" {
		keymapPath, err = DefaultKeymapPath()
		if err != nil {
			return Config{}, err
		}
	}
	keymapPath, err = NormalizeNotesDir(keymapPath)
	if err != nil {
		return Config{}, fmt.Errorf("invalid keymap_file: %w", err)
	}
	cfg.KeymapFile = keymapPath
	if cfg.Keybindings == nil {
		cfg.Keybindings = map[string]string{}
	}
	if len(cfg.Workspaces) == 0 && strings.TrimSpace(cfg.NotesDir) == "" {
		return Config{}, fmt.Errorf("invalid notes_dir: %w", errors.New("path is required"))
	}

	normalizedWorkspaces, normalizedActive, err := normalizeWorkspaces(cfg.Workspaces, cfg.ActiveWorkspace, cfg.NotesDir)
	if err != nil {
		return Config{}, err
	}
	cfg.Workspaces = normalizedWorkspaces
	cfg.ActiveWorkspace = normalizedActive
	for _, ws := range cfg.Workspaces {
		if ws.Name == cfg.ActiveWorkspace {
			cfg.NotesDir = ws.NotesDir
			break
		}
	}

	return cfg, nil
}

// Save writes configuration to disk.
func Save(cfg Config) error {
	var err error
	legacyNotesDir := strings.TrimSpace(cfg.NotesDir)
	if legacyNotesDir != "" {
		notesDir, normErr := NormalizeNotesDir(legacyNotesDir)
		if normErr != nil {
			return fmt.Errorf("invalid notes_dir: %w", normErr)
		}
		cfg.NotesDir = notesDir
	}
	cfg.TreeSort = strings.TrimSpace(strings.ToLower(cfg.TreeSort))
	if cfg.TreeSort == "" {
		cfg.TreeSort = "name"
	}

	templatesDir := strings.TrimSpace(cfg.TemplatesDir)
	if templatesDir == "" {
		templatesDir, err = DefaultTemplatesDir()
		if err != nil {
			return err
		}
	}
	templatesDir, err = NormalizeNotesDir(templatesDir)
	if err != nil {
		return fmt.Errorf("invalid templates_dir: %w", err)
	}
	cfg.TemplatesDir = templatesDir
	keymapPath := strings.TrimSpace(cfg.KeymapFile)
	if keymapPath == "" {
		keymapPath, err = DefaultKeymapPath()
		if err != nil {
			return err
		}
	}
	keymapPath, err = NormalizeNotesDir(keymapPath)
	if err != nil {
		return fmt.Errorf("invalid keymap_file: %w", err)
	}
	cfg.KeymapFile = keymapPath
	if len(cfg.Workspaces) == 0 && strings.TrimSpace(cfg.NotesDir) == "" {
		return fmt.Errorf("invalid notes_dir: %w", errors.New("path is required"))
	}

	normalizedWorkspaces, normalizedActive, err := normalizeWorkspaces(cfg.Workspaces, cfg.ActiveWorkspace, cfg.NotesDir)
	if err != nil {
		return err
	}
	cfg.Workspaces = normalizedWorkspaces
	cfg.ActiveWorkspace = normalizedActive
	for _, ws := range cfg.Workspaces {
		if ws.Name == cfg.ActiveWorkspace {
			cfg.NotesDir = ws.NotesDir
			break
		}
	}
	if cfg.Keybindings == nil {
		cfg.Keybindings = map[string]string{}
	}
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir %q: %w", filepath.Dir(path), err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	log.Info("saved config", "path", path, "notes_dir", cfg.NotesDir)
	return nil
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
		return "", fmt.Errorf("resolve absolute path for %q: %w", expanded, err)
	}

	return filepath.Clean(abs), nil
}

func expandHome(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

func normalizeWorkspaces(workspaces []WorkspaceConfig, activeWorkspace string, fallbackNotesDir string) ([]WorkspaceConfig, string, error) {
	normalized := make([]WorkspaceConfig, 0, len(workspaces)+1)
	seenNames := map[string]bool{}
	seenDirs := map[string]bool{}
	addWorkspace := func(name, notesDir string) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return errors.New("workspace name is required")
		}
		notesDir, err := NormalizeNotesDir(notesDir)
		if err != nil {
			return fmt.Errorf("workspace %q invalid notes_dir: %w", name, err)
		}
		lower := strings.ToLower(name)
		if seenNames[lower] {
			return fmt.Errorf("duplicate workspace name %q", name)
		}
		if seenDirs[notesDir] {
			return fmt.Errorf("duplicate workspace notes_dir %q", notesDir)
		}
		seenNames[lower] = true
		seenDirs[notesDir] = true
		normalized = append(normalized, WorkspaceConfig{Name: name, NotesDir: notesDir})
		return nil
	}

	for _, ws := range workspaces {
		if err := addWorkspace(ws.Name, ws.NotesDir); err != nil {
			return nil, "", err
		}
	}

	fallback := strings.TrimSpace(fallbackNotesDir)
	if len(normalized) == 0 {
		if fallback == "" {
			return nil, "", errors.New("at least one workspace is required")
		}
		if err := addWorkspace("default", fallback); err != nil {
			return nil, "", err
		}
	}

	active := strings.TrimSpace(activeWorkspace)
	if active == "" {
		active = normalized[0].Name
	}
	found := false
	for _, ws := range normalized {
		if strings.EqualFold(ws.Name, active) {
			active = ws.Name
			found = true
			break
		}
	}
	if !found {
		active = normalized[0].Name
	}
	return normalized, active, nil
}

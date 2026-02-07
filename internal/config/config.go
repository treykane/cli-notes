// Package config manages the persistent user configuration for cli-notes.
//
// Configuration is stored in a JSON file at ~/.cli-notes/config.json. The file
// is created by the first-run configurator (see cmd/notes/main.go) and can be
// re-generated at any time with `notes --configure`.
//
// # Configuration Fields
//
//   - notes_dir:         Legacy single-workspace notes directory (migrated to workspaces).
//   - tree_sort:         Persisted tree sort mode (name, modified, size, created).
//   - templates_dir:     Directory containing note templates (default: ~/.cli-notes/templates).
//   - workspaces:        Named workspace list, each with its own notes_dir.
//   - active_workspace:  Name of the currently active workspace.
//   - keybindings:       Inline action→key overrides (merged with keymap_file).
//   - keymap_file:       Path to an external keymap JSON file (default: ~/.cli-notes/keymap.json).
//   - theme_preset:      UI color preset (ocean_citrus, sunset, neon_slate).
//
// # Workspace Migration
//
// Older config files that have only a top-level notes_dir (no workspaces array)
// are automatically migrated: a single workspace named "default" is created
// pointing at the legacy notes_dir.
//
// # Path Normalization
//
// All directory paths stored in config are expanded (~ → home dir) and made
// absolute before use, so relative or tilde-prefixed paths in the JSON are
// handled transparently. See NormalizeNotesDir for details.
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
	// configDirName is the hidden directory under the user's home where app-level
	// configuration (config.json, keymap.json, templates/) is stored.
	configDirName = ".cli-notes"

	// configFileName is the name of the JSON configuration file inside configDirName.
	configFileName = "config.json"

	// ThemePresetOceanCitrus is the default Ocean + Citrus UI palette.
	ThemePresetOceanCitrus = "ocean_citrus"
	// ThemePresetSunset is the warm amber/salmon UI palette.
	ThemePresetSunset = "sunset"
	// ThemePresetNeonSlate is the cool cyan/lime UI palette.
	ThemePresetNeonSlate = "neon_slate"
)

// ErrNotConfigured is returned by Load when no config file exists, signaling
// the caller to run the interactive configurator before starting the app.
var ErrNotConfigured = errors.New("cli-notes is not configured")

// log is the structured logger for the config package, tagged with component="config".
var log = logging.New("config")

// Config stores user-defined CLI Notes settings.
//
// The struct is serialized to and deserialized from ~/.cli-notes/config.json.
// Fields tagged with omitempty are excluded from the JSON output when empty,
// keeping the config file concise for simple single-workspace setups.
type Config struct {
	// NotesDir is the active workspace's notes directory (absolute path).
	// In multi-workspace configs this is derived from the active workspace
	// entry and may differ from what is stored on disk.
	NotesDir string `json:"notes_dir,omitempty"`

	// TreeSort is the persisted tree sort mode (name, modified, size, created).
	TreeSort string `json:"tree_sort,omitempty"`
	// TreeSortByWorkspace stores per-workspace sort mode keyed by workspace notes_dir.
	TreeSortByWorkspace map[string]string `json:"tree_sort_by_workspace,omitempty"`

	// TemplatesDir is the directory scanned for note templates when creating
	// new notes. Defaults to ~/.cli-notes/templates if unset.
	TemplatesDir string `json:"templates_dir,omitempty"`

	// Workspaces lists all configured named workspaces. If empty, a default
	// workspace is synthesized from NotesDir during Load.
	Workspaces []WorkspaceConfig `json:"workspaces,omitempty"`

	// ActiveWorkspace is the name of the workspace to activate on startup.
	// Must match one of the entries in Workspaces.
	ActiveWorkspace string `json:"active_workspace,omitempty"`

	// Keybindings holds inline action→key overrides from config.json. These
	// are merged with (and take priority over) any keymap_file bindings.
	Keybindings map[string]string `json:"keybindings,omitempty"`

	// KeymapFile is the path to an external keymap JSON file with additional
	// keybinding overrides. Defaults to ~/.cli-notes/keymap.json if unset.
	KeymapFile string `json:"keymap_file,omitempty"`

	// ThemePreset selects the app UI color palette. Supported values:
	// ocean_citrus, sunset, neon_slate.
	ThemePreset string `json:"theme_preset,omitempty"`
}

// WorkspaceConfig pairs a human-readable workspace name with the absolute path
// to its notes directory. Names must be unique (case-insensitive) and
// directories must not overlap between workspaces.
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

// Exists reports whether the config file exists on disk. This is used at
// startup to decide whether to run the first-run configurator.
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

// Load reads, parses, and validates the saved configuration from disk.
//
// Validation steps performed during load:
//  1. All directory paths are normalized (~ expanded, made absolute).
//  2. TreeSort defaults to "name" if empty.
//  3. TemplatesDir defaults to ~/.cli-notes/templates if empty.
//  4. KeymapFile defaults to ~/.cli-notes/keymap.json if empty.
//  5. ThemePreset defaults to ocean_citrus when missing or invalid.
//  6. Workspaces are normalized: names are validated for uniqueness, directories
//     are expanded and checked for duplicates. If no workspaces are configured,
//     a "default" workspace is created from the legacy notes_dir field.
//  7. ActiveWorkspace is resolved to an existing workspace name (falls back to
//     the first workspace if the configured name doesn't match).
//  8. NotesDir is set to the active workspace's directory.
//
// Returns ErrNotConfigured if the config file does not exist.
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
	cfg.TreeSortByWorkspace = normalizeTreeSortByWorkspace(cfg.TreeSortByWorkspace)

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
	cfg.ThemePreset = NormalizeThemePreset(cfg.ThemePreset)
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

// Save writes configuration to disk at ~/.cli-notes/config.json.
//
// Before writing, the configuration is normalized using the same rules as Load
// (path expansion, workspace deduplication, sort mode defaulting) so the
// persisted file is always in canonical form. The config directory is created
// if it doesn't exist. The file is written with restrictive permissions (0600)
// since it may contain filesystem paths the user considers private.
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
	cfg.TreeSortByWorkspace = normalizeTreeSortByWorkspace(cfg.TreeSortByWorkspace)

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
	cfg.ThemePreset = NormalizeThemePreset(cfg.ThemePreset)
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

// NormalizeNotesDir expands and normalizes a filesystem path for use as a
// notes directory (or templates directory, or keymap file path).
//
// Processing steps:
//  1. Trim whitespace.
//  2. Expand leading ~ or ~/ to the user's home directory.
//  3. Resolve to an absolute path.
//  4. Clean redundant separators and . / .. components.
//
// Returns an error if the path is empty or home directory resolution fails.
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

// expandHome replaces a leading ~ or ~/ with the current user's home directory.
// Paths that don't start with ~ are returned unchanged. This allows users to
// write portable paths like "~/notes" in their config file.
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

// normalizeWorkspaces validates and normalizes the workspace list.
//
// It enforces the following invariants:
//   - Every workspace has a non-empty, unique name (case-insensitive).
//   - Every workspace's notes_dir is a valid, unique absolute path.
//   - At least one workspace exists (if the list is empty, a "default" workspace
//     is created from fallbackNotesDir).
//   - activeWorkspace resolves to an existing workspace name; if it doesn't
//     match any workspace, the first workspace is selected as the default.
//
// Returns the normalized workspace list and the resolved active workspace name.
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

func normalizeTreeSortByWorkspace(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}
	normalized := make(map[string]string, len(raw))
	for notesDir, mode := range raw {
		dir, err := NormalizeNotesDir(notesDir)
		if err != nil {
			continue
		}
		switch value := strings.TrimSpace(strings.ToLower(mode)); value {
		case "name", "modified", "size", "created":
			normalized[dir] = value
		}
	}
	return normalized
}

// NormalizeThemePreset canonicalizes theme preset names and falls back to the
// default preset when the value is empty or unknown.
func NormalizeThemePreset(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.NewReplacer("-", "_", " ", "_", "/", "_").Replace(normalized)
	switch normalized {
	case "", ThemePresetOceanCitrus, "oceancitrus":
		return ThemePresetOceanCitrus
	case ThemePresetSunset:
		return ThemePresetSunset
	case ThemePresetNeonSlate, "neonslate":
		return ThemePresetNeonSlate
	default:
		return ThemePresetOceanCitrus
	}
}

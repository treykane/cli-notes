package config

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureLogOutput captures logs written during a test function.
func captureLogOutput(t *testing.T, fn func()) []byte {
	t.Helper()
	var buf bytes.Buffer
	oldLogger := log
	log = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	defer func() { log = oldLogger }()
	fn()
	return buf.Bytes()
}

func TestSaveConfigDirCreationError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create a file where the config dir should be
	configDir := filepath.Join(home, configDirName)
	if err := os.WriteFile(configDir, []byte("blocking file"), 0o644); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	cfg := Config{NotesDir: "~/notes"}
	err := Save(cfg)
	if err == nil {
		t.Fatal("expected error when config dir path is blocked by a file")
	}

	if !strings.Contains(err.Error(), "create config dir") {
		t.Errorf("error should mention config dir creation, got: %v", err)
	}
}

func TestSaveConfigFileWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create config dir but make it read-only
	configDir := filepath.Join(home, configDirName)
	if err := os.Mkdir(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(configDir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(configDir, 0o755) // cleanup

	cfg := Config{NotesDir: "~/notes"}
	err := Save(cfg)
	if err == nil {
		t.Fatal("expected error when config file cannot be written")
	}

	if !strings.Contains(err.Error(), "write config") {
		t.Errorf("error should mention config write, got: %v", err)
	}
}

func TestSaveConfigSuccessLogsInfo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := Config{NotesDir: "~/my-notes"}

	logs := captureLogOutput(t, func() {
		if err := Save(cfg); err != nil {
			t.Fatalf("save config: %v", err)
		}
	})

	logStr := string(logs)
	if !strings.Contains(logStr, "level=INFO") {
		t.Error("successful save should log at INFO level")
	}
	if !strings.Contains(logStr, "saved config") {
		t.Error("log should contain 'saved config'")
	}
	expectedPath := filepath.Join(home, configDirName, configFileName)
	if !strings.Contains(logStr, expectedPath) {
		t.Errorf("log should contain config path %q", expectedPath)
	}
}

func TestLoadConfigReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create config file with no read permissions
	configPath := filepath.Join(home, configDirName, configFileName)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"notes_dir":"~/notes"}`), 0o000); err != nil {
		t.Fatalf("write config: %v", err)
	}
	defer os.Chmod(configPath, 0o644) // cleanup

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when config file cannot be read")
	}

	if !strings.Contains(err.Error(), "read config") {
		t.Errorf("error should mention read config, got: %v", err)
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, configDirName, configFileName)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{invalid json`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when config contains invalid JSON")
	}

	if !strings.Contains(err.Error(), "parse config") {
		t.Errorf("error should mention parse config, got: %v", err)
	}
}

func TestExistsStatError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create config dir with no permissions
	configDir := filepath.Join(home, configDirName)
	if err := os.Mkdir(configDir, 0o000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.Chmod(configDir, 0o755) // cleanup

	// This should fail to stat the config file due to directory permissions
	exists, err := Exists()
	if err == nil {
		t.Fatal("expected error when config directory cannot be accessed")
	}

	if exists {
		t.Error("should return false when stat fails")
	}

	if !strings.Contains(err.Error(), "stat config path") {
		t.Errorf("error should mention stat, got: %v", err)
	}
}

func TestLoadConfigEmptyNotesDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, configDirName, configFileName)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"notes_dir":"   "}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when notes_dir is empty")
	}

	if !strings.Contains(err.Error(), "invalid notes_dir") {
		t.Errorf("error should mention invalid notes_dir, got: %v", err)
	}
}

func TestSaveConfigInvalidNotesDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := Config{NotesDir: "   "}
	err := Save(cfg)
	if err == nil {
		t.Fatal("expected error when notes_dir is empty")
	}

	if !strings.Contains(err.Error(), "invalid notes_dir") {
		t.Errorf("error should mention invalid notes_dir, got: %v", err)
	}
}

func TestConfigPathUserHomeDirError(t *testing.T) {
	// Save original HOME
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Unset HOME to cause UserHomeDir to fail
	os.Unsetenv("HOME")

	_, err := ConfigPath()
	if err == nil {
		t.Fatal("expected error when HOME is not set")
	}

	if !strings.Contains(err.Error(), "resolve home dir") {
		t.Errorf("error should mention home dir resolution, got: %v", err)
	}
}

func TestDefaultNotesDirUserHomeDirError(t *testing.T) {
	// Save original HOME
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Unset HOME to cause UserHomeDir to fail
	os.Unsetenv("HOME")

	_, err := DefaultNotesDir()
	if err == nil {
		t.Fatal("expected error when HOME is not set")
	}

	if !strings.Contains(err.Error(), "resolve home dir") {
		t.Errorf("error should mention home dir resolution, got: %v", err)
	}
}

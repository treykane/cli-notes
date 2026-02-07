package app

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// captureLogOutput captures logs written during a test function.
func captureLogOutput(t *testing.T, fn func()) []byte {
	t.Helper()
	var buf bytes.Buffer
	oldLogger := appLog
	appLog = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	defer func() { appLog = oldLogger }()
	fn()
	return buf.Bytes()
}

func TestEnsureNotesDirPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	readOnlyDir := filepath.Join(root, "readonly")
	if err := os.Mkdir(readOnlyDir, 0o555); err != nil {
		t.Fatalf("create readonly dir: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0o755) // cleanup

	subdir := filepath.Join(readOnlyDir, "notes")
	err := ensureNotesDir(subdir)
	if err == nil {
		t.Fatal("expected error when creating directory in read-only parent")
	}

	if !strings.Contains(err.Error(), "create notes directory") {
		t.Errorf("error message should mention directory creation, got: %v", err)
	}
}

func TestEnsureNotesDirWelcomeFileWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	// Create directory but make it read-only after creation
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(root, 0o755) // cleanup

	logs := captureLogOutput(t, func() {
		err := ensureNotesDir(root)
		if err == nil {
			t.Error("expected error when writing welcome file to read-only directory")
		}
		if !strings.Contains(err.Error(), "seed welcome note") {
			t.Errorf("error should mention welcome note, got: %v", err)
		}
	})

	// Logs should be empty since this is a returned error, not logged
	_ = logs
}

func TestStartEditNoteFileReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	noReadFile := filepath.Join(root, "secret.md")
	if err := os.WriteFile(noReadFile, []byte("secret"), 0o000); err != nil {
		t.Fatalf("write file: %v", err)
	}
	defer os.Chmod(noReadFile, 0o644) // cleanup

	m := &Model{
		notesDir:    root,
		currentFile: noReadFile,
	}

	logs := captureLogOutput(t, func() {
		result, _ := m.startEditNote()
		resultModel := result.(*Model)

		if resultModel.status != "Error reading note" {
			t.Errorf("expected status 'Error reading note', got: %q", resultModel.status)
		}

		if resultModel.mode != modeBrowse {
			t.Errorf("mode should remain browse on error, got: %v", resultModel.mode)
		}
	})

	logStr := string(logs)
	if !strings.Contains(logStr, "Error reading note") {
		t.Error("log should contain 'Error reading note'")
	}
	if !strings.Contains(logStr, "level=ERROR") {
		t.Error("log should be at ERROR level")
	}
	if !strings.Contains(logStr, noReadFile) {
		t.Errorf("log should contain file path %q", noReadFile)
	}
}

func TestSaveNewNoteWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	readOnlyDir := filepath.Join(root, "readonly")
	if err := os.Mkdir(readOnlyDir, 0o555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0o755) // cleanup

	input := textinput.New()
	input.SetValue("test-note")

	m := &Model{
		notesDir:  root,
		newParent: readOnlyDir,
		mode:      modeNewNote,
		expanded:  make(map[string]bool),
		input:     input,
	}

	logs := captureLogOutput(t, func() {
		result, _ := m.saveNewNote()
		resultModel := result.(*Model)

		if resultModel.status != "Error creating note" {
			t.Errorf("expected status 'Error creating note', got: %q", resultModel.status)
		}

		if resultModel.mode != modeNewNote {
			t.Errorf("mode should remain modeNewNote on error, got: %v", resultModel.mode)
		}
	})

	logStr := string(logs)
	if !strings.Contains(logStr, "Error creating note") {
		t.Error("log should contain 'Error creating note'")
	}
	if !strings.Contains(logStr, "level=ERROR") {
		t.Error("log should be at ERROR level")
	}
	if !strings.Contains(logStr, "path=") {
		t.Error("log should contain path attribute")
	}
}

func TestSaveNewFolderPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	readOnlyDir := filepath.Join(root, "readonly")
	if err := os.Mkdir(readOnlyDir, 0o555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0o755) // cleanup

	input := textinput.New()
	input.SetValue("subfolder")

	m := &Model{
		notesDir:  root,
		newParent: readOnlyDir,
		mode:      modeNewFolder,
		expanded:  make(map[string]bool),
		input:     input,
	}

	logs := captureLogOutput(t, func() {
		result, _ := m.saveNewFolder()
		resultModel := result.(*Model)

		if resultModel.status != "Error creating folder" {
			t.Errorf("expected status 'Error creating folder', got: %q", resultModel.status)
		}

		if resultModel.mode != modeNewFolder {
			t.Errorf("mode should remain modeNewFolder on error, got: %v", resultModel.mode)
		}
	})

	logStr := string(logs)
	if !strings.Contains(logStr, "Error creating folder") {
		t.Error("log should contain 'Error creating folder'")
	}
	if !strings.Contains(logStr, "level=ERROR") {
		t.Error("log should be at ERROR level")
	}
}

func TestSaveEditWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	readOnlyFile := filepath.Join(root, "readonly.md")
	if err := os.WriteFile(readOnlyFile, []byte("original"), 0o444); err != nil {
		t.Fatalf("write file: %v", err)
	}
	defer os.Chmod(readOnlyFile, 0o644) // cleanup

	m := &Model{
		notesDir:    root,
		currentFile: readOnlyFile,
		mode:        modeEditNote,
		editor:      textarea.New(),
	}
	m.editor.SetValue("modified content")

	logs := captureLogOutput(t, func() {
		result, _ := m.saveEdit()
		resultModel := result.(*Model)

		if resultModel.status != "Error saving note" {
			t.Errorf("expected status 'Error saving note', got: %q", resultModel.status)
		}

		if resultModel.mode != modeEditNote {
			t.Errorf("mode should remain modeEditNote on error, got: %v", resultModel.mode)
		}
	})

	logStr := string(logs)
	if !strings.Contains(logStr, "Error saving note") {
		t.Error("log should contain 'Error saving note'")
	}
	if !strings.Contains(logStr, "level=ERROR") {
		t.Error("log should be at ERROR level")
	}
	if !strings.Contains(logStr, readOnlyFile) {
		t.Errorf("log should contain file path %q", readOnlyFile)
	}
}

func TestPerformDeleteFilePermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	readOnlyDir := filepath.Join(root, "readonly")
	protectedFile := filepath.Join(readOnlyDir, "protected.md")

	if err := os.Mkdir(readOnlyDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(protectedFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Chmod(readOnlyDir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0o755) // cleanup

	m := &Model{
		notesDir: root,
		expanded: make(map[string]bool),
	}
	item := &treeItem{
		path:  protectedFile,
		name:  "protected.md",
		isDir: false,
	}

	logs := captureLogOutput(t, func() {
		m.performDelete(item)

		if m.status != "Error deleting file" {
			t.Errorf("expected status 'Error deleting file', got: %q", m.status)
		}
	})

	logStr := string(logs)
	if !strings.Contains(logStr, "Error deleting file") {
		t.Error("log should contain 'Error deleting file'")
	}
	if !strings.Contains(logStr, "level=ERROR") {
		t.Error("log should be at ERROR level")
	}
	if !strings.Contains(logStr, protectedFile) {
		t.Errorf("log should contain file path %q", protectedFile)
	}
}

func TestPerformDeleteFolderPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	parentDir := filepath.Join(root, "parent")
	emptyDir := filepath.Join(parentDir, "empty")

	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(parentDir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(parentDir, 0o755) // cleanup

	m := &Model{
		notesDir: root,
		expanded: make(map[string]bool),
	}
	item := &treeItem{
		path:  emptyDir,
		name:  "empty",
		isDir: true,
	}

	logs := captureLogOutput(t, func() {
		m.performDelete(item)

		if m.status != "Error deleting folder" {
			t.Errorf("expected status 'Error deleting folder', got: %q", m.status)
		}
	})

	logStr := string(logs)
	if !strings.Contains(logStr, "Error deleting folder") {
		t.Error("log should contain 'Error deleting folder'")
	}
	if !strings.Contains(logStr, "level=ERROR") {
		t.Error("log should be at ERROR level")
	}
}

func TestIsDirEmptyHandlesReadError(t *testing.T) {
	// Test with non-existent directory
	isEmpty := isDirEmpty("/nonexistent/path/that/does/not/exist")
	if isEmpty {
		t.Error("isDirEmpty should return false for non-existent directory")
	}

	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	// Test with permission error
	root := t.TempDir()
	noReadDir := filepath.Join(root, "noread")
	if err := os.Mkdir(noReadDir, 0o000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.Chmod(noReadDir, 0o755) // cleanup

	isEmpty = isDirEmpty(noReadDir)
	if isEmpty {
		t.Error("isDirEmpty should return false when directory cannot be read")
	}
}

func TestSelectedParentDirHandlesStatError(t *testing.T) {
	root := t.TempDir()
	nonExistentPath := filepath.Join(root, "does-not-exist.md")

	m := &Model{
		notesDir: root,
		cursor:   0,
		items: []treeItem{
			{path: nonExistentPath, name: "does-not-exist.md", isDir: false},
		},
	}

	// Should fall back to root directory when stat fails
	parent := m.selectedParentDir()
	if parent != root {
		t.Errorf("expected parent to be root %q, got %q", root, parent)
	}
}

func TestStartEditNoteWithEmptyCurrentFile(t *testing.T) {
	m := &Model{
		currentFile: "",
	}

	result, cmd := m.startEditNote()
	resultModel := result.(*Model)

	if resultModel.status != "No note selected" {
		t.Errorf("expected status 'No note selected', got: %q", resultModel.status)
	}
	if cmd != nil {
		t.Error("expected nil command when no file selected")
	}
}

func TestSaveEditWithEmptyCurrentFile(t *testing.T) {
	m := &Model{
		currentFile: "",
	}

	result, cmd := m.saveEdit()
	resultModel := result.(*Model)

	if resultModel.status != "No note selected" {
		t.Errorf("expected status 'No note selected', got: %q", resultModel.status)
	}
	if cmd != nil {
		t.Error("expected nil command when no file selected")
	}
}

func TestSaveNewNoteSuccessLogsNoErrors(t *testing.T) {
	root := t.TempDir()

	input := textinput.New()
	input.SetValue("success-note")

	m := &Model{
		notesDir:  root,
		newParent: root,
		mode:      modeNewNote,
		expanded:  make(map[string]bool),
		items:     []treeItem{},
		input:     input,
	}

	logs := captureLogOutput(t, func() {
		result, _ := m.saveNewNote()
		resultModel := result.(*Model)

		if resultModel.mode != modeBrowse {
			t.Errorf("expected mode to be modeBrowse after success, got: %v", resultModel.mode)
		}
		if !strings.Contains(resultModel.status, "Created note:") {
			t.Errorf("expected success status, got: %q", resultModel.status)
		}
	})

	logStr := string(logs)
	if strings.Contains(logStr, "level=ERROR") {
		t.Errorf("should not log errors on success, got: %s", logStr)
	}
}

func TestSetStatusErrorLogsWithAttributes(t *testing.T) {
	m := &Model{}

	logs := captureLogOutput(t, func() {
		m.setStatusError("Test error message", os.ErrPermission, "file", "/test/path.md", "operation", "write")
	})

	logStr := string(logs)
	if !strings.Contains(logStr, "level=ERROR") {
		t.Error("log should be at ERROR level")
	}
	if !strings.Contains(logStr, "Test error message") {
		t.Error("log should contain error message")
	}
	if !strings.Contains(logStr, "file=/test/path.md") {
		t.Error("log should contain file attribute")
	}
	if !strings.Contains(logStr, "operation=write") {
		t.Error("log should contain operation attribute")
	}
	if !strings.Contains(logStr, "permission denied") {
		t.Error("log should contain the actual error")
	}

	if m.status != "Test error message" {
		t.Errorf("status should be set to error message, got: %q", m.status)
	}
}

func TestDeleteSelectedWithNoSelection(t *testing.T) {
	m := &Model{
		notesDir: t.TempDir(),
		items:    []treeItem{},
		cursor:   0,
	}

	m.deleteSelected()

	if m.status != "No item selected" {
		t.Errorf("expected 'No item selected', got: %q", m.status)
	}
}

func TestDeleteSelectedRootDirectory(t *testing.T) {
	root := t.TempDir()
	m := &Model{
		notesDir: root,
		items: []treeItem{
			{path: root, name: "root", isDir: true},
		},
		cursor: 0,
	}

	m.deleteSelected()

	if m.status != "Cannot delete the root notes directory" {
		t.Errorf("expected root deletion error, got: %q", m.status)
	}
}

func TestDeleteSelectedNonEmptyFolder(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "nonempty")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(folder, "file.md"), []byte("content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m := &Model{
		notesDir: root,
		items: []treeItem{
			{path: folder, name: "nonempty", isDir: true},
		},
		cursor: 0,
	}

	m.deleteSelected()

	if !strings.Contains(m.status, "not empty") {
		t.Errorf("expected non-empty folder error, got: %q", m.status)
	}
}

func TestSaveRenameItemUpdatesCurrentFile(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "old.md")
	newPath := filepath.Join(root, "new.md")
	if err := os.WriteFile(oldPath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write old file: %v", err)
	}

	input := textinput.New()
	input.SetValue("new.md")
	m := &Model{
		notesDir:    root,
		mode:        modeRenameItem,
		actionPath:  oldPath,
		currentFile: oldPath,
		expanded:    map[string]bool{root: true},
		searchIndex: newSearchIndex(root),
		input:       input,
	}

	result, _ := m.saveRenameItem()
	got := result.(*Model)

	if got.mode != modeBrowse {
		t.Fatalf("expected browse mode, got %v", got.mode)
	}
	if got.currentFile != newPath {
		t.Fatalf("expected currentFile %q, got %q", newPath, got.currentFile)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected renamed file at %q: %v", newPath, err)
	}
}

func TestSaveMoveItemUpdatesCurrentFileForFolderMove(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	dstDir := filepath.Join(root, "dst")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("mkdir dst: %v", err)
	}
	notePath := filepath.Join(srcDir, "note.md")
	if err := os.WriteFile(notePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}

	input := textinput.New()
	input.SetValue("dst")
	m := &Model{
		notesDir:    root,
		mode:        modeMoveItem,
		actionPath:  srcDir,
		currentFile: notePath,
		expanded:    map[string]bool{root: true, srcDir: true, dstDir: true},
		searchIndex: newSearchIndex(root),
		input:       input,
	}

	result, _ := m.saveMoveItem()
	got := result.(*Model)
	newFolder := filepath.Join(dstDir, "src")
	newNote := filepath.Join(newFolder, "note.md")

	if got.currentFile != newNote {
		t.Fatalf("expected moved currentFile %q, got %q", newNote, got.currentFile)
	}
	if _, err := os.Stat(newFolder); err != nil {
		t.Fatalf("expected moved folder at %q: %v", newFolder, err)
	}
}

func TestDeleteSelectedStartsConfirmationMode(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.md")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	m := &Model{
		notesDir: root,
		items:    []treeItem{{path: path, name: "a.md", isDir: false}},
		cursor:   0,
	}

	m.deleteSelected()

	if m.mode != modeConfirmDelete {
		t.Fatalf("expected modeConfirmDelete, got %v", m.mode)
	}
	if m.pendingDelete.path != path {
		t.Fatalf("expected pending delete path %q, got %q", path, m.pendingDelete.path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should not be deleted until confirmation: %v", err)
	}
}

// Verify Model has required fields for testing
var _ = tea.Model(&Model{})

func TestNoteCRUDLifecycleIntegration(t *testing.T) {
	cases := []struct {
		name       string
		parentRel  string
		noteName   string
		initial    string
		editedBody string
		queryToken string
	}{
		{
			name:       "root note lifecycle",
			parentRel:  ".",
			noteName:   "lifecycle.md",
			initial:    "# lifecycle\n\nalpha token\n",
			editedBody: "# lifecycle\n\nbeta token\n",
			queryToken: "beta",
		},
		{
			name:       "nested note lifecycle",
			parentRel:  "projects/sub",
			noteName:   "nested.md",
			initial:    "# nested\n\nhello world\n",
			editedBody: "# nested\n\nupdated phrase\n",
			queryToken: "updated",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			parent := root
			if tc.parentRel != "." {
				parent = filepath.Join(root, tc.parentRel)
				if err := os.MkdirAll(parent, 0o755); err != nil {
					t.Fatalf("mkdir parent: %v", err)
				}
			}

			m := newTestCRUDModel(root)
			m.newParent = parent
			expandAncestors(m.expanded, root, parent)
			m.input.SetValue(tc.noteName)
			m.selectedTemplate = &noteTemplate{content: tc.initial}

			model, _ := m.saveNewNote()
			m = model.(*Model)
			notePath := filepath.Join(parent, tc.noteName)
			if got := m.currentFile; got != notePath {
				t.Fatalf("expected current file %q, got %q", notePath, got)
			}

			gotContent, err := os.ReadFile(notePath)
			if err != nil {
				t.Fatalf("read created note: %v", err)
			}
			if want := normalizeNoteContent(tc.initial); string(gotContent) != want {
				t.Fatalf("unexpected created content\nwant: %q\ngot:  %q", want, string(gotContent))
			}
			assertTreeHasPath(t, m.items, notePath)
			assertSearchHasQuery(t, m.searchIndex, tc.initial, true)

			m.editor.SetValue(tc.editedBody)
			model, _ = m.saveEdit()
			m = model.(*Model)
			gotContent, err = os.ReadFile(notePath)
			if err != nil {
				t.Fatalf("read edited note: %v", err)
			}
			if want := normalizeNoteContent(tc.editedBody); string(gotContent) != want {
				t.Fatalf("unexpected edited content\nwant: %q\ngot:  %q", want, string(gotContent))
			}
			assertTreeHasPath(t, m.items, notePath)
			assertSearchHasQuery(t, m.searchIndex, tc.queryToken, true)
			assertSearchHasQuery(t, m.searchIndex, "alpha", false)

			reselectTreeItem(t, m, notePath)
			m.deleteSelected()
			if m.mode != modeConfirmDelete {
				t.Fatalf("expected confirm delete mode, got %v", m.mode)
			}
			model, _ = m.handleConfirmDeleteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			m = model.(*Model)
			if _, err := os.Stat(notePath); !os.IsNotExist(err) {
				t.Fatalf("expected deleted note, stat err: %v", err)
			}
			assertTreeHasPath(t, m.items, notePath, false)
			assertSearchHasQuery(t, m.searchIndex, tc.queryToken, false)
		})
	}
}

func TestMovePathWithFallbackUsesCopyDeleteOnEXDEVFile(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src.md")
	dst := filepath.Join(root, "dst.md")
	if err := os.WriteFile(src, []byte("hello"), 0o640); err != nil {
		t.Fatalf("write src: %v", err)
	}

	oldRename := renameForMove
	defer func() { renameForMove = oldRename }()
	renameForMove = func(oldPath, newPath string) error {
		return &os.LinkError{Op: "rename", Old: oldPath, New: newPath, Err: syscall.EXDEV}
	}

	if err := movePathWithFallback(src, dst); err != nil {
		t.Fatalf("move with fallback: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected source removed, stat err: %v", err)
	}
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("expected dst content copied, got %q", string(content))
	}
}

func TestMovePathWithFallbackUsesCopyDeleteOnEXDEVFolder(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "note.md"), []byte("nested"), 0o644); err != nil {
		t.Fatalf("write nested: %v", err)
	}

	oldRename := renameForMove
	defer func() { renameForMove = oldRename }()
	renameForMove = func(oldPath, newPath string) error {
		return &os.LinkError{Op: "rename", Old: oldPath, New: newPath, Err: syscall.EXDEV}
	}

	if err := movePathWithFallback(src, dst); err != nil {
		t.Fatalf("move folder with fallback: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected source folder removed, stat err: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dst, "nested", "note.md"))
	if err != nil {
		t.Fatalf("read moved nested file: %v", err)
	}
	if string(content) != "nested" {
		t.Fatalf("unexpected moved content: %q", string(content))
	}
}

func TestMovePathWithFallbackCleansDestinationOnCopyFailure(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src.md")
	dst := filepath.Join(root, "dst.md")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	oldRename := renameForMove
	oldCopy := copyPathForMove
	defer func() {
		renameForMove = oldRename
		copyPathForMove = oldCopy
	}()

	renameForMove = func(oldPath, newPath string) error {
		return &os.LinkError{Op: "rename", Old: oldPath, New: newPath, Err: syscall.EXDEV}
	}
	copyPathForMove = func(oldPath, newPath string) error {
		if err := os.WriteFile(newPath, []byte("partial"), 0o644); err != nil {
			t.Fatalf("seed partial destination: %v", err)
		}
		return os.ErrPermission
	}

	if err := movePathWithFallback(src, dst); err == nil {
		t.Fatal("expected move failure")
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("expected destination cleanup, stat err: %v", err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source should remain after failed fallback: %v", err)
	}
}

func newTestCRUDModel(root string) *Model {
	idx := newSearchIndex(root)
	if err := idx.ensureBuilt(); err != nil {
		panic(err)
	}
	m := &Model{
		notesDir:       root,
		mode:           modeNewNote,
		expanded:       map[string]bool{root: true},
		items:          buildTree(root, map[string]bool{root: true}, sortModeName, nil),
		input:          textinput.New(),
		editor:         textarea.New(),
		searchIndex:    idx,
		notePositions:  map[string]notePosition{},
		noteOpenCounts: map[string]int{},
		pinnedPaths:    map[string]bool{},
		renderCache:    map[string]renderCacheEntry{},
	}
	m.editor.Focus()
	return m
}

func assertTreeHasPath(t *testing.T, items []treeItem, path string, expected ...bool) {
	t.Helper()
	want := true
	if len(expected) > 0 {
		want = expected[0]
	}
	found := false
	for _, item := range items {
		if item.path == path {
			found = true
			break
		}
	}
	if found != want {
		t.Fatalf("tree path presence mismatch for %q: found=%v want=%v", path, found, want)
	}
}

func assertSearchHasQuery(t *testing.T, idx *searchIndex, query string, want bool) {
	t.Helper()
	if err := idx.ensureBuilt(); err != nil {
		t.Fatalf("ensure search index built: %v", err)
	}
	got := len(idx.search(query)) > 0
	if got != want {
		t.Fatalf("search presence mismatch for query %q: got=%v want=%v", query, got, want)
	}
}

func reselectTreeItem(t *testing.T, m *Model, targetPath string) {
	t.Helper()
	for i, item := range m.items {
		if item.path == targetPath {
			m.cursor = i
			return
		}
	}
	t.Fatalf("target path not found in tree: %q", targetPath)
}

func expandAncestors(expanded map[string]bool, root, targetDir string) {
	if expanded == nil {
		return
	}
	dir := filepath.Clean(targetDir)
	root = filepath.Clean(root)
	for {
		expanded[dir] = true
		if dir == root {
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

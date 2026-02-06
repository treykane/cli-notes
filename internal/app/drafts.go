package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const draftAutoSaveInterval = 5 * time.Second

type draftRecord struct {
	SourcePath string    `json:"source_path"`
	Content    string    `json:"content"`
	UpdatedAt  time.Time `json:"updated_at"`
	DraftPath  string    `json:"-"`
}

type draftAutoSaveTickMsg struct{}

func (m *Model) scheduleDraftAutosave() tea.Cmd {
	return tea.Tick(draftAutoSaveInterval, func(time.Time) tea.Msg {
		return draftAutoSaveTickMsg{}
	})
}

func (m *Model) handleDraftAutoSaveTick(_ draftAutoSaveTickMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeEditNote && m.currentFile != "" {
		if err := m.saveDraftForCurrentFile(); err != nil {
			appLog.Warn("auto-save draft", "path", m.currentFile, "error", err)
		}
	}
	return m, m.scheduleDraftAutosave()
}

func (m *Model) saveDraftForCurrentFile() error {
	if m.currentFile == "" {
		return nil
	}
	content := m.editor.Value()
	if onDisk, err := os.ReadFile(m.currentFile); err == nil && string(onDisk) == content {
		m.clearDraftForPath(m.currentFile)
		return nil
	}

	record := draftRecord{
		SourcePath: m.currentFile,
		Content:    content,
		UpdatedAt:  time.Now(),
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	draftPath := m.draftPathForSource(m.currentFile)
	if err := os.MkdirAll(filepath.Dir(draftPath), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(draftPath, data, 0o600); err != nil {
		return err
	}
	m.lastDraftAutosaveAt = record.UpdatedAt
	return nil
}

func (m *Model) draftsDir() string {
	return filepath.Join(m.notesDir, managedNotesDirName, ".drafts")
}

func (m *Model) draftPathForSource(sourcePath string) string {
	hash := sha256.Sum256([]byte(sourcePath))
	name := hex.EncodeToString(hash[:]) + ".json"
	return filepath.Join(m.draftsDir(), name)
}

func (m *Model) clearDraftForPath(path string) {
	if path == "" {
		return
	}
	draftPath := m.draftPathForSource(path)
	if err := os.Remove(draftPath); err != nil && !os.IsNotExist(err) {
		appLog.Warn("remove draft", "path", draftPath, "error", err)
	}
}

func (m *Model) loadPendingDrafts() {
	entries, err := os.ReadDir(m.draftsDir())
	if err != nil {
		if !os.IsNotExist(err) {
			appLog.Warn("list draft files", "dir", m.draftsDir(), "error", err)
		}
		return
	}

	recoveries := make([]draftRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(m.draftsDir(), entry.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			appLog.Warn("read draft", "path", path, "error", readErr)
			continue
		}
		var record draftRecord
		if err := json.Unmarshal(data, &record); err != nil {
			appLog.Warn("parse draft", "path", path, "error", err)
			continue
		}
		if record.SourcePath == "" || !isWithinRoot(m.notesDir, record.SourcePath) {
			_ = os.Remove(path)
			continue
		}
		if onDisk, statErr := os.ReadFile(record.SourcePath); statErr == nil && string(onDisk) == record.Content {
			_ = os.Remove(path)
			continue
		}
		record.DraftPath = path
		recoveries = append(recoveries, record)
	}

	if len(recoveries) == 0 {
		return
	}

	sort.Slice(recoveries, func(i, j int) bool {
		return recoveries[i].UpdatedAt.After(recoveries[j].UpdatedAt)
	})
	m.pendingDrafts = recoveries
	m.advanceDraftRecoveryPrompt()
}

func (m *Model) advanceDraftRecoveryPrompt() {
	if len(m.pendingDrafts) == 0 {
		m.activeDraft = nil
		if m.mode == modeDraftRecovery {
			m.mode = modeBrowse
			m.status = "Draft recovery complete"
		}
		return
	}

	next := m.pendingDrafts[0]
	m.pendingDrafts = m.pendingDrafts[1:]
	m.activeDraft = &next
	m.mode = modeDraftRecovery
	m.status = "Unsaved draft found"
}

func (m *Model) handleDraftRecoveryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shouldIgnoreInput(msg) {
		return m, nil
	}
	if m.activeDraft == nil {
		m.mode = modeBrowse
		return m, nil
	}

	switch msg.String() {
	case "y", "Y":
		record := *m.activeDraft
		if err := os.MkdirAll(filepath.Dir(record.SourcePath), DirPermission); err != nil {
			m.setStatusError("Draft recovery failed", err)
			return m, nil
		}
		if err := os.WriteFile(record.SourcePath, []byte(record.Content), FilePermission); err != nil {
			m.setStatusError("Draft recovery failed", err)
			return m, nil
		}
		_ = os.Remove(record.DraftPath)
		m.currentFile = record.SourcePath
		m.currentNoteContent = record.Content
		m.status = "Recovered draft: " + filepath.Base(record.SourcePath)
		m.advanceDraftRecoveryPrompt()
		m.refreshTree()
		return m, m.setCurrentFile(record.SourcePath)
	case "n", "N":
		record := *m.activeDraft
		_ = os.Remove(record.DraftPath)
		m.status = "Discarded draft: " + filepath.Base(record.SourcePath)
		m.advanceDraftRecoveryPrompt()
		return m, nil
	case "esc":
		m.activeDraft = nil
		m.pendingDrafts = nil
		m.mode = modeBrowse
		m.status = "Skipped draft recovery"
		return m, nil
	default:
		return m, nil
	}
}

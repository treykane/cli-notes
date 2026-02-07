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

// draftAutoSaveInterval controls how frequently the editor content is
// automatically saved to a draft file while in edit mode. This provides
// crash-recovery protection: if the application exits unexpectedly, the
// user's unsaved work can be recovered on the next launch.
const draftAutoSaveInterval = 5 * time.Second

// draftRecord represents a single auto-saved draft stored on disk.
//
// Each draft is serialized as a JSON file inside the managed drafts directory
// (<notes_dir>/.cli-notes/.drafts/). The filename is a SHA-256 hash of the
// source note's absolute path, ensuring a stable 1:1 mapping between notes
// and their draft files regardless of filename characters.
type draftRecord struct {
	// SourcePath is the absolute path of the original note file that was
	// being edited when this draft was created.
	SourcePath string `json:"source_path"`

	// Content holds the full editor buffer text at the time of the auto-save.
	Content string `json:"content"`

	// UpdatedAt records when this draft was last written, used for sorting
	// recovery candidates (most recent first).
	UpdatedAt time.Time `json:"updated_at"`

	// DraftPath is the filesystem path of the draft JSON file itself.
	// This field is not serialized; it is populated at load time so the
	// recovery logic can delete the draft file after recovery or discard.
	DraftPath string `json:"-"`
}

// draftAutoSaveTickMsg is the Bubble Tea message emitted by the periodic
// auto-save timer. When received by the Update loop, it triggers a draft
// save (if currently editing) and reschedules the next tick.
type draftAutoSaveTickMsg struct{}

// scheduleDraftAutosave returns a Bubble Tea command that emits a
// draftAutoSaveTickMsg after draftAutoSaveInterval elapses.
//
// This is called once at Init() and then again after each tick is handled,
// creating a continuous auto-save loop that runs for the lifetime of the app.
func (m *Model) scheduleDraftAutosave() tea.Cmd {
	return tea.Tick(draftAutoSaveInterval, func(time.Time) tea.Msg {
		return draftAutoSaveTickMsg{}
	})
}

// handleDraftAutoSaveTick processes the periodic auto-save timer tick.
//
// If the user is currently editing a note, the editor buffer is saved to a
// draft file. Any errors during save are logged but do not interrupt the
// editing session. The next auto-save tick is always rescheduled regardless
// of whether a save was attempted or succeeded.
func (m *Model) handleDraftAutoSaveTick(_ draftAutoSaveTickMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeEditNote && m.currentFile != "" {
		if err := m.saveDraftForCurrentFile(); err != nil {
			appLog.Warn("auto-save draft", "path", m.currentFile, "error", err)
		}
	}
	return m, m.scheduleDraftAutosave()
}

// saveDraftForCurrentFile writes the current editor buffer to a draft file.
//
// The draft is only written when the editor content differs from the on-disk
// file content. If the content matches the saved file exactly, any existing
// draft is removed instead (the user has manually synced or the content was
// reverted).
//
// Draft files are stored as JSON in <notes_dir>/.cli-notes/.drafts/ using a
// SHA-256 hash of the source path as the filename. This avoids conflicts
// with special characters in note names and ensures each note has at most
// one draft file.
func (m *Model) saveDraftForCurrentFile() error {
	if m.currentFile == "" {
		return nil
	}
	content := m.editor.Value()

	// If the editor content matches the on-disk file, there is nothing
	// unsaved — clean up any stale draft and return early.
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

// draftsDir returns the absolute path to the directory where draft files
// are stored: <notes_dir>/.cli-notes/.drafts/
func (m *Model) draftsDir() string {
	return filepath.Join(m.notesDir, managedNotesDirName, ".drafts")
}

// draftPathForSource computes the draft file path for a given source note.
//
// It uses a SHA-256 hash of the source path to generate a deterministic,
// collision-resistant filename. This means:
//   - Each source note always maps to the same draft file.
//   - No issues with special characters, long paths, or path separators.
//   - Looking up or deleting a draft only requires the source path.
func (m *Model) draftPathForSource(sourcePath string) string {
	hash := sha256.Sum256([]byte(sourcePath))
	name := hex.EncodeToString(hash[:]) + ".json"
	return filepath.Join(m.draftsDir(), name)
}

// clearDraftForPath removes the draft file associated with the given source
// note path. This is called after a successful save or when the user cancels
// editing, since the draft is no longer needed.
//
// If no draft exists or the removal fails for a non-critical reason (e.g.
// file already deleted), the error is logged but not propagated.
func (m *Model) clearDraftForPath(path string) {
	if path == "" {
		return
	}
	draftPath := m.draftPathForSource(path)
	if err := os.Remove(draftPath); err != nil && !os.IsNotExist(err) {
		appLog.Warn("remove draft", "path", draftPath, "error", err)
	}
}

// loadPendingDrafts scans the drafts directory for recoverable unsaved work.
//
// This is called once during app initialization (in New()). For each draft
// file found, it:
//  1. Reads and parses the draft JSON.
//  2. Validates that the source path is within the current notes directory
//     (drafts from other workspaces or deleted notes are cleaned up).
//  3. Compares the draft content against the current on-disk file content.
//     If they match, the draft is stale and is silently removed.
//  4. Collects remaining drafts as recovery candidates, sorted by UpdatedAt
//     (most recent first).
//
// If any valid drafts are found, the app enters modeDraftRecovery to prompt
// the user to recover or discard each one before normal use begins.
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

		// Discard drafts that reference files outside the current notes root
		// (e.g. leftovers from a different workspace configuration).
		if record.SourcePath == "" || !isWithinRoot(m.notesDir, record.SourcePath) {
			_ = os.Remove(path)
			continue
		}

		// Discard drafts whose content already matches the on-disk file
		// (the note was saved normally after the draft was created).
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

	// Present the most recently modified drafts first so the user sees
	// the most relevant recovery candidates at the top.
	sort.Slice(recoveries, func(i, j int) bool {
		return recoveries[i].UpdatedAt.After(recoveries[j].UpdatedAt)
	})
	m.pendingDrafts = recoveries
	m.advanceDraftRecoveryPrompt()
}

// advanceDraftRecoveryPrompt moves to the next pending draft recovery
// candidate. If no more candidates remain, it exits recovery mode and
// returns to normal browse mode.
//
// This is called after the user accepts or discards a draft, and also
// during initial setup after loadPendingDrafts populates the queue.
func (m *Model) advanceDraftRecoveryPrompt() {
	if len(m.pendingDrafts) == 0 {
		m.activeDraft = nil
		if m.mode == modeDraftRecovery {
			m.mode = modeBrowse
			m.status = "Draft recovery complete"
		}
		return
	}

	// Pop the first candidate from the queue and present it.
	next := m.pendingDrafts[0]
	m.pendingDrafts = m.pendingDrafts[1:]
	m.activeDraft = &next
	m.mode = modeDraftRecovery
	m.status = "Unsaved draft found"
}

// handleDraftRecoveryKey processes user input during the draft recovery
// prompt shown at startup.
//
// The user has three choices for each draft:
//   - 'y' / 'Y': Recover the draft by overwriting the source note file
//     with the draft content, then open the recovered note.
//   - 'n' / 'N': Discard the draft permanently and move to the next one.
//   - 'Esc': Skip all remaining drafts and enter normal browse mode.
//
// After each decision, advanceDraftRecoveryPrompt is called to present the
// next candidate or exit recovery mode.
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
		// Recover: write the draft content back to the source note file.
		record := *m.activeDraft
		if err := os.MkdirAll(filepath.Dir(record.SourcePath), DirPermission); err != nil {
			m.setStatusError("Draft recovery failed", err)
			return m, nil
		}
		if err := os.WriteFile(record.SourcePath, []byte(record.Content), FilePermission); err != nil {
			m.setStatusError("Draft recovery failed", err)
			return m, nil
		}
		// Clean up the draft file now that the content has been restored.
		_ = os.Remove(record.DraftPath)
		m.currentFile = record.SourcePath
		m.currentNoteContent = record.Content
		m.status = "Recovered draft: " + filepath.Base(record.SourcePath)
		m.advanceDraftRecoveryPrompt()
		m.refreshTree()
		return m, m.setCurrentFile(record.SourcePath)
	case "n", "N":
		// Discard: permanently delete the draft file and move on.
		record := *m.activeDraft
		_ = os.Remove(record.DraftPath)
		m.status = "Discarded draft: " + filepath.Base(record.SourcePath)
		m.advanceDraftRecoveryPrompt()
		return m, nil
	case "esc":
		// Skip all: abandon remaining recovery prompts and enter browse mode.
		m.activeDraft = nil
		m.pendingDrafts = nil
		m.mode = modeBrowse
		m.status = "Skipped draft recovery"
		return m, nil
	default:
		return m, nil
	}
}

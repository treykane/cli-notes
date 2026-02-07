// templates.go implements the note template picker flow.
//
// When the user presses `n` to create a new note, the app checks the
// configured templates directory (default: ~/.cli-notes/templates/) for
// template files. If templates exist, a picker popup is shown before the
// note-name input so the user can choose a starting template. If no
// templates are found, the flow falls through directly to the name input
// with the built-in default template.
//
// Templates are plain files (any format, though typically .md) stored in
// the templates directory. Each file's content is read at picker-open time
// and used verbatim as the initial content of the new note. A synthetic
// "Default (no template)" entry is always prepended so the user can opt out
// of templating.
package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// noteTemplate represents a single entry in the template picker.
//
// The first entry in the picker list is always the built-in default
// (name="Default (no template)", path="", content=""), which signals
// that the standard auto-generated heading template should be used.
// All other entries correspond to files in the templates directory.
type noteTemplate struct {
	name    string // display name shown in the picker (filename for real templates)
	path    string // absolute path to the template file (empty for the default entry)
	content string // raw file content to seed the new note with
}

// loadTemplates reads template files from the configured templates directory
// and returns a slice of noteTemplate entries for the picker popup.
//
// Behavior:
//   - A synthetic "Default (no template)" entry is always the first item.
//   - Sub-directories inside the templates directory are ignored.
//   - Template files that cannot be read are logged and skipped.
//   - The returned slice is sorted alphabetically (case-insensitive) after
//     the default entry.
//   - Returns nil if the templates directory is missing, empty, or contains
//     only directories — this signals the caller to skip the picker entirely
//     and go straight to the note-name input.
func (m *Model) loadTemplates() []noteTemplate {
	templates := []noteTemplate{{name: "Default (no template)"}}
	if m.templatesDir == "" {
		return templates
	}

	entries, err := os.ReadDir(m.templatesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		m.setStatusError("Error loading templates", err, "dir", m.templatesDir)
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(m.templatesDir, entry.Name())
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			appLog.Warn("read template", "path", path, "error", readErr)
			continue
		}
		templates = append(templates, noteTemplate{
			name:    entry.Name(),
			path:    path,
			content: string(content),
		})
	}

	// If only the default entry remains, no real templates were found.
	if len(templates) <= 1 {
		return nil
	}

	// Sort real templates alphabetically, leaving the default entry at index 0.
	sort.Slice(templates[1:], func(i, j int) bool {
		return strings.ToLower(templates[1+i].name) < strings.ToLower(templates[1+j].name)
	})
	return templates
}

// handleTemplatePickerKey processes key events while the template picker popup
// is active. Navigation uses j/k or arrow keys. Enter/Ctrl+S confirms the
// selection and transitions to the note-name input (modeNewNote). Esc cancels
// the entire new-note flow and returns to browse mode.
//
// When the user selects the default entry (path == ""), selectedTemplate is set
// to nil so saveNewNote uses the auto-generated heading template. Otherwise,
// selectedTemplate is set to a copy of the chosen template's metadata and
// content so it can be written when the note name is confirmed.
func (m *Model) handleTemplatePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shouldIgnoreInput(msg) {
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if len(m.templates) > 0 {
			m.templateCursor = clamp(m.templateCursor-1, 0, len(m.templates)-1)
		}
		return m, nil
	case "down", "j":
		if len(m.templates) > 0 {
			m.templateCursor = clamp(m.templateCursor+1, 0, len(m.templates)-1)
		}
		return m, nil
	case "enter", "ctrl+s":
		if len(m.templates) == 0 {
			m.mode = modeBrowse
			m.status = "No templates available"
			return m, nil
		}
		chosen := m.templates[m.templateCursor]
		if chosen.path == "" {
			m.selectedTemplate = nil
			m.status = "Using default note template"
		} else {
			m.selectedTemplate = &noteTemplate{name: chosen.name, path: chosen.path, content: chosen.content}
			m.status = "Using template: " + chosen.name
		}
		m.configureInputForMode(modeNewNote, "Note name (without .md extension)")
		return m, nil
	case "esc":
		m.mode = modeBrowse
		m.templates = nil
		m.selectedTemplate = nil
		m.status = "New note cancelled"
		return m, nil
	default:
		return m, nil
	}
}

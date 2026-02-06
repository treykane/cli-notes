package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type noteTemplate struct {
	name    string
	path    string
	content string
}

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

	if len(templates) <= 1 {
		return nil
	}

	sort.Slice(templates[1:], func(i, j int) bool {
		return strings.ToLower(templates[1+i].name) < strings.ToLower(templates[1+j].name)
	})
	return templates
}

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

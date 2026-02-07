package app

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/treykane/cli-notes/internal/config"
)

const (
	actionSearch            = "search.open"
	actionRecent            = "recent.open"
	actionOutline           = "outline.open"
	actionWorkspace         = "workspace.open"
	actionNewNote           = "note.new"
	actionNewFolder         = "folder.new"
	actionEditNote          = "note.edit"
	actionSort              = "tree.sort.cycle"
	actionPin               = "tree.pin.toggle"
	actionDelete            = "item.delete"
	actionCopyContent       = "note.copy_content"
	actionCopyPath          = "note.copy_path"
	actionRename            = "item.rename"
	actionRefresh           = "tree.refresh"
	actionMove              = "item.move"
	actionGitCommit         = "git.commit"
	actionGitPull           = "git.pull"
	actionGitPush           = "git.push"
	actionExport            = "note.export"
	actionWikiLinks         = "wiki.links.open"
	actionSplitToggle       = "split.toggle"
	actionSplitFocus        = "split.focus.toggle"
	actionHelp              = "help.toggle"
	actionQuit              = "app.quit"
	actionPreviewLinkFollow = "preview.link.follow"
)

var defaultActionKeys = map[string]string{
	actionSearch:      "ctrl+p",
	actionRecent:      "ctrl+o",
	actionOutline:     "o",
	actionWorkspace:   "ctrl+w",
	actionNewNote:     "n",
	actionNewFolder:   "f",
	actionEditNote:    "e",
	actionSort:        "s",
	actionPin:         "t",
	actionDelete:      "d",
	actionCopyContent: "y",
	actionCopyPath:    "shift+y",
	actionRename:      "r",
	actionRefresh:     "ctrl+r",
	actionMove:        "m",
	actionGitCommit:   "c",
	actionGitPull:     "p",
	actionGitPush:     "shift+p",
	actionExport:      "x",
	actionWikiLinks:   "shift+l",
	actionSplitToggle: "z",
	actionSplitFocus:  "tab",
	actionHelp:        "?",
	actionQuit:        "q",
}

func (m *Model) loadKeybindings(cfg config.Config) {
	m.keyForAction = map[string]string{}
	for action, key := range defaultActionKeys {
		m.keyForAction[action] = key
	}
	for action, key := range cfg.Keybindings {
		m.applyKeybindingOverride(action, key)
	}

	fileOverrides := loadKeymapFile(cfg.KeymapFile)
	for action, key := range fileOverrides {
		m.applyKeybindingOverride(action, key)
	}
	m.rebuildActionKeyIndex()
}

func loadKeymapFile(path string) map[string]string {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			appLog.Warn("read keymap file", "path", path, "error", err)
		}
		return nil
	}
	overrides := map[string]string{}
	if err := json.Unmarshal(data, &overrides); err != nil {
		appLog.Warn("parse keymap file", "path", path, "error", err)
		return nil
	}
	return overrides
}

func (m *Model) applyKeybindingOverride(action, key string) {
	action = strings.TrimSpace(action)
	key = normalizeKeyString(key)
	if action == "" || key == "" {
		return
	}
	if _, ok := defaultActionKeys[action]; !ok {
		appLog.Warn("ignore unknown keybinding action", "action", action)
		return
	}
	m.keyForAction[action] = key
}

func (m *Model) rebuildActionKeyIndex() {
	m.keyToAction = map[string]string{}
	for action, key := range m.keyForAction {
		if key == "" {
			continue
		}
		if existing, ok := m.keyToAction[key]; ok && existing != action {
			appLog.Warn("keybinding conflict ignored", "key", key, "action", action, "existing_action", existing)
			continue
		}
		m.keyToAction[key] = action
	}
}

func normalizeKeyString(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	// Bubble Tea may report uppercase single rune keys for shifted letters.
	if len([]rune(key)) == 1 && strings.ToUpper(key) == key && strings.ToLower(key) != key {
		return "shift+" + strings.ToLower(key)
	}
	switch strings.ToLower(key) {
	case "y":
		if key == "Y" {
			return "shift+y"
		}
	}
	return strings.ToLower(key)
}

func (m *Model) actionForKey(key string) string {
	if m.keyToAction == nil {
		return ""
	}
	return m.keyToAction[normalizeKeyString(key)]
}

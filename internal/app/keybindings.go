package app

import (
	"encoding/json"
	"os"
	"slices"
	"strings"

	"github.com/treykane/cli-notes/internal/config"
)

// ---------------------------------------------------------------------------
// Action constants
// ---------------------------------------------------------------------------
//
// Each constant below identifies a user-triggerable action in browse mode.
// Actions are the abstraction layer between physical key presses and
// application behavior: the user presses a key, the key is looked up in the
// keyToAction map, and the resulting action string is dispatched in
// handleBrowseKey.
//
// Default key assignments are declared in defaultActionKeys. Users can
// override any assignment via the "keybindings" map in config.json or via an
// external keymap file (default: ~/.cli-notes/keymap.json).
// ---------------------------------------------------------------------------

const (
	// actionSearchHint shows a browse-mode hint to use Ctrl+P search.
	actionSearchHint = "search.hint"

	// actionCursorUp moves the tree selection up by one item.
	actionCursorUp = "tree.cursor.up"

	// actionCursorDown moves the tree selection down by one item.
	actionCursorDown = "tree.cursor.down"

	// actionJumpTop moves selection to the first visible tree item.
	actionJumpTop = "tree.jump.top"

	// actionJumpBottom moves selection to the last visible tree item.
	actionJumpBottom = "tree.jump.bottom"

	// actionExpandToggle toggles expansion for the selected directory.
	actionExpandToggle = "tree.expand.toggle"

	// actionCollapse collapses the selected directory.
	actionCollapse = "tree.collapse"

	// actionSearch opens the Ctrl+P full-text search popup.
	actionSearch = "search.open"

	// actionRecent opens the recent-files quick-jump popup (Ctrl+O).
	actionRecent = "recent.open"

	// actionOutline opens the heading outline popup for the current note.
	actionOutline = "outline.open"

	// actionWorkspace opens the workspace switcher popup (Ctrl+W).
	actionWorkspace = "workspace.open"

	// actionNewNote starts the new-note creation flow (template picker →
	// name input → file creation).
	actionNewNote = "note.new"

	// actionNewFolder starts the new-folder creation flow.
	actionNewFolder = "folder.new"

	// actionEditNote enters edit mode for the currently selected note.
	actionEditNote = "note.edit"

	// actionSort cycles through the tree sort modes (name → modified →
	// size → created → name …).
	actionSort = "tree.sort.cycle"

	// actionPreviewScrollPageUp scrolls the active preview pane up by one
	// viewport page.
	actionPreviewScrollPageUp = "preview.scroll.page_up"

	// actionPreviewScrollPageDown scrolls the active preview pane down by one
	// viewport page.
	actionPreviewScrollPageDown = "preview.scroll.page_down"

	// actionPreviewScrollHalfUp scrolls the active preview pane up by half
	// a viewport page.
	actionPreviewScrollHalfUp = "preview.scroll.half_up"

	// actionPreviewScrollHalfDown scrolls the active preview pane down by half
	// a viewport page.
	actionPreviewScrollHalfDown = "preview.scroll.half_down"

	// actionPin toggles the pinned state of the currently selected tree item.
	// Pinned items float to the top of their parent folder regardless of sort.
	actionPin = "tree.pin.toggle"

	// actionDelete initiates deletion of the selected item (prompts for
	// confirmation before actually removing).
	actionDelete = "item.delete"

	// actionCopyContent copies the raw text content of the current note to
	// the system clipboard.
	actionCopyContent = "note.copy_content"

	// actionCopyPath copies the absolute filesystem path of the current note
	// to the system clipboard.
	actionCopyPath = "note.copy_path"

	// actionRename enters rename mode for the selected tree item.
	actionRename = "item.rename"

	// actionRefresh forces a full rebuild of the tree, search index, render
	// cache, and git status.
	actionRefresh = "tree.refresh"

	// actionMove enters move mode for the selected tree item, prompting for
	// a destination folder path.
	actionMove = "item.move"

	// actionGitCommit starts the git commit flow (git add -A && git commit).
	// Only available when the notes directory is inside a git repository.
	actionGitCommit = "git.commit"

	// actionGitPull runs git pull --ff-only in the notes directory.
	actionGitPull = "git.pull"

	// actionGitPush runs git push in the notes directory.
	actionGitPush = "git.push"

	// actionExport opens the export popup for the current note (HTML / PDF).
	actionExport = "note.export"

	// actionWikiLinks opens the wiki-links popup showing all [[...]] links
	// found in the current note and their resolution status.
	actionWikiLinks = "wiki.links.open"

	// actionSplitToggle enables or disables split-pane mode, which shows two
	// notes side by side.
	actionSplitToggle = "split.toggle"

	// actionSplitFocus toggles keyboard focus between the primary and
	// secondary split panes.
	actionSplitFocus = "split.focus.toggle"

	// actionHelp toggles the in-app keyboard shortcut reference panel.
	actionHelp = "help.toggle"

	// actionQuit exits the application, saving state and cleaning up.
	actionQuit = "app.quit"

	// actionPreviewLinkFollow follows a hyperlink under the cursor in preview
	// mode (reserved for future use).
	actionPreviewLinkFollow = "preview.link.follow"
)

// defaultActionKeys maps each action to its factory-default key bindings.
//
// These defaults are designed to be intuitive for users familiar with Vim-
// style terminal applications. They can be overridden per-user via config.json
// ("keybindings" object) or an external keymap file ("keymap_file" path).
//
// Key strings use the Bubble Tea notation:
//   - Modifier keys: "ctrl+", "alt+", "shift+"
//   - Special keys: "enter", "esc", "tab", "up", "down", "left", "right"
//   - Single characters: "n", "f", "e", "?", etc.
var defaultActionKeys = map[string][]string{
	actionSearchHint:            {"/"},
	actionCursorUp:              {"up", "k"},
	actionCursorDown:            {"down", "j", "ctrl+n"},
	actionJumpTop:               {"g"},
	actionJumpBottom:            {"shift+g"},
	actionExpandToggle:          {"enter", "right", "l"},
	actionCollapse:              {"left", "h"},
	actionSearch:                {"ctrl+p"},
	actionRecent:                {"ctrl+o"},
	actionOutline:               {"o"},
	actionWorkspace:             {"ctrl+w"},
	actionNewNote:               {"n"},
	actionNewFolder:             {"f"},
	actionEditNote:              {"e"},
	actionSort:                  {"s"},
	actionPreviewScrollPageUp:   {"pgup"},
	actionPreviewScrollPageDown: {"pgdown"},
	actionPreviewScrollHalfUp:   {"ctrl+u"},
	actionPreviewScrollHalfDown: {"ctrl+d"},
	actionPin:                   {"t"},
	actionDelete:                {"d"},
	actionCopyContent:           {"y"},
	actionCopyPath:              {"shift+y"},
	actionRename:                {"r"},
	actionRefresh:               {"ctrl+r", "shift+r"},
	actionMove:                  {"m"},
	actionGitCommit:             {"c"},
	actionGitPull:               {"p"},
	actionGitPush:               {"shift+p"},
	actionExport:                {"x"},
	actionWikiLinks:             {"shift+l"},
	actionSplitToggle:           {"z"},
	actionSplitFocus:            {"tab"},
	actionHelp:                  {"?"},
	actionQuit:                  {"q", "ctrl+c"},
}

// ---------------------------------------------------------------------------
// Keybinding initialization
// ---------------------------------------------------------------------------

// loadKeybindings initializes the bidirectional key↔action maps from three
// sources, applied in order of increasing priority:
//
//  1. defaultActionKeys — built-in factory defaults (always applied first).
//  2. cfg.Keybindings — inline overrides from the "keybindings" object in
//     ~/.cli-notes/config.json.
//  3. External keymap file — overrides from the JSON file at cfg.KeymapFile
//     (default ~/.cli-notes/keymap.json), if it exists.
//
// After all sources are merged, rebuildActionKeyIndex is called to build the
// reverse lookup map (key string → action) used at runtime for fast dispatch.
//
// Any unknown action names in user overrides are logged as warnings and
// ignored. Overrides replace an action's full default key set with the
// configured key. Key conflicts (two actions mapped to the same key) are also
// logged as warnings; the first action to claim a key wins.
func (m *Model) loadKeybindings(cfg config.Config) {
	// Start with a fresh copy of the factory defaults.
	m.keyForAction = map[string][]string{}
	for action, keys := range defaultActionKeys {
		m.keyForAction[action] = append([]string(nil), keys...)
	}

	// Layer on inline config overrides (lower priority than keymap file).
	for action, key := range cfg.Keybindings {
		m.applyKeybindingOverride(action, key)
	}

	// Layer on external keymap file overrides (highest priority).
	fileOverrides := loadKeymapFile(cfg.KeymapFile)
	for action, key := range fileOverrides {
		m.applyKeybindingOverride(action, key)
	}

	// Build the reverse index for runtime key → action lookups.
	m.rebuildActionKeyIndex()
}

// loadKeymapFile reads and parses an external JSON keymap file.
//
// The file is expected to contain a flat JSON object mapping action strings
// to key strings, for example:
//
//	{
//	    "note.new": "ctrl+n",
//	    "tree.sort.cycle": "S"
//	}
//
// If the file does not exist, nil is returned silently (the keymap file is
// entirely optional). Parse errors or read errors for existing files are
// logged as warnings.
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

// applyKeybindingOverride updates a single action's key binding, replacing the
// action's full default key set.
//
// Both the action and key are trimmed and normalized. If the action string
// is not recognized (i.e. it does not exist in defaultActionKeys), the
// override is ignored and a warning is logged. This prevents typos in
// config files from silently failing.
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
	m.keyForAction[action] = []string{key}
}

// rebuildActionKeyIndex constructs the reverse lookup map (keyToAction) from
// the current keyForAction map.
//
// The reverse map is used at runtime by actionForKey to translate incoming
// key press strings into action identifiers without iterating over the full
// action map on every keystroke.
//
// If two actions are mapped to the same key, a warning is logged and the
// first action encountered keeps the binding. The conflicting action's key
// is effectively unbound. This is a deliberate safety measure to prevent
// ambiguous key presses.
func (m *Model) rebuildActionKeyIndex() {
	m.keyToAction = map[string]string{}
	for action, keys := range m.keyForAction {
		for _, key := range keys {
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
}

// ---------------------------------------------------------------------------
// Key string normalization
// ---------------------------------------------------------------------------

// normalizeKeyString converts a user-provided key string into the canonical
// lowercase form used internally by Bubble Tea and the keybinding maps.
//
// Normalization rules:
//   - Whitespace is trimmed.
//   - The entire string is lowercased (Bubble Tea reports keys in lowercase).
//   - A single uppercase letter (e.g. "Y") is converted to "shift+y" because
//     Bubble Tea may report shifted letter keys as uppercase runes. This
//     ensures that both "Y" and "shift+y" in config files produce the same
//     internal representation.
//
// Examples:
//
//	normalizeKeyString("Ctrl+P")  → "ctrl+p"
//	normalizeKeyString(" Y ")     → "shift+y"
//	normalizeKeyString("shift+l") → "shift+l"
//	normalizeKeyString("")        → ""
func normalizeKeyString(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	// Bubble Tea may report uppercase single rune keys for shifted letters.
	// Normalize "Y" → "shift+y" so config files can use either form.
	if len([]rune(key)) == 1 && strings.ToUpper(key) == key && strings.ToLower(key) != key {
		return "shift+" + strings.ToLower(key)
	}
	return strings.ToLower(key)
}

// actionForKey looks up the action bound to the given key string.
//
// The key is normalized before lookup to ensure consistent matching
// regardless of how the terminal reports the key event. Returns an empty
// string if no action is bound to the key.
//
// This function is called on every key press in browse mode (from
// handleBrowseKey) to determine which action, if any, should be triggered.
func (m *Model) actionForKey(key string) string {
	if m.keyToAction == nil {
		return ""
	}
	return m.keyToAction[normalizeKeyString(key)]
}

func (m *Model) actionKeyLabels(action string) []string {
	keys, ok := m.keyForAction[action]
	if !ok || len(keys) == 0 {
		return nil
	}
	labels := make([]string, 0, len(keys))
	for _, key := range keys {
		label := humanizeKeyLabel(key)
		if label == "" {
			continue
		}
		if slices.Contains(labels, label) {
			continue
		}
		labels = append(labels, label)
	}
	return labels
}

func (m *Model) primaryActionKey(action, fallback string) string {
	keys := m.actionKeyLabels(action)
	if len(keys) == 0 {
		return fallback
	}
	return keys[0]
}

func (m *Model) allActionKeys(action, fallback string) string {
	keys := m.actionKeyLabels(action)
	if len(keys) == 0 {
		return fallback
	}
	return strings.Join(keys, ", ")
}

func humanizeKeyLabel(key string) string {
	normalized := normalizeKeyString(key)
	if normalized == "" {
		return ""
	}
	special := map[string]string{
		"up":        "↑",
		"down":      "↓",
		"left":      "←",
		"right":     "→",
		"enter":     "Enter",
		"esc":       "Esc",
		"tab":       "Tab",
		"home":      "Home",
		"end":       "End",
		"pgup":      "PgUp",
		"pgdown":    "PgDn",
		"space":     "Space",
		"backspace": "Backspace",
	}
	parts := strings.Split(normalized, "+")
	for i, part := range parts {
		switch part {
		case "ctrl":
			parts[i] = "Ctrl"
		case "alt":
			parts[i] = "Alt"
		case "shift":
			parts[i] = "Shift"
		default:
			if label, ok := special[part]; ok {
				parts[i] = label
				continue
			}
			runes := []rune(part)
			if len(runes) == 1 && runes[0] >= 'a' && runes[0] <= 'z' {
				parts[i] = strings.ToUpper(part)
			} else {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
	}
	return strings.Join(parts, "+")
}

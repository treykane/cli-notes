# Tasks

## Features
[ ] Allow Syncing Via GIT repository

## UI
[x] Add notes to the UI for all keybinds.
[ ] Render the markdown notes unless in Edit mode. 
[x] Add a search to the UI.
[x] Move search to a popup opened with Ctrl+P.
[x] Look into using VIM friendly keybinds.
[ ] First note always gets stuck "Rendering..." until you view the next next note in the tree.
[ ] Clear stale pane text when switching help or screens (ensure panes pad to full width/height).
[x] Bring the colors and styles into the Note Edit UI.
[x] Avoid OSC background color query strings (ex: `1;rgb:...`) being inserted into notes when editing.
[x] Filter stray OSC background response sequences from editor input on first edit.
[x] Capture and classify remaining "random" injected input on first edit (enable `CLI_NOTES_DEBUG_INPUT`).
[x] Add mode-aware preview/edit color accents so it is obvious when edit mode is active.
[ ] Search currently matches note and folder names only (no note body/content search).

## Performance
[ ] Improve performance on a fresh build.

## Developer Experiqence
[x] Improve developer experience by adding more detailed documentation.
[ ] Add more tests to ensure the application works as expected.
[ ] Improve error handling and logging.
[ ] Improve code readability and maintainability.
[x] Improve code organization and modularity.
[x] Add inline documentation to `cmd/notes/main.go`.
[x] Add CONTRIBUTING.md with PR and code style guidelines.

## Testing
[ ] Implement testing framework
[ ] Add unit tests for tree building and render cache behavior in `internal/app`.

# Tasks

## Features
[ ] Allow Syncing Via GIT repository
[x] Add a first-run configurator and `--configure` command for selecting notes storage directory.

## UI
[x] Add notes to the UI for all keybinds.
[x] Add a search to the UI.
[x] Move search to a popup opened with Ctrl+P.
[x] Look into using VIM friendly keybinds.
[x] First note always gets stuck "Rendering..." until you view the next next note in the tree.
[x] Clear stale pane text when switching help or screens (ensure panes pad to full width/height).
[x] Bring the colors and styles into the Note Edit UI.
[x] Avoid OSC background color query strings (ex: `1;rgb:...`) being inserted into notes when editing.
[x] Filter stray OSC background response sequences from editor input on first edit.
[x] Capture and classify remaining "random" injected input on first edit (enable `CLI_NOTES_DEBUG_INPUT`).
[x] Add mode-aware preview/edit color accents so it is obvious when edit mode is active.
[x] Turn the right-pane filename row into a solid status bar and add more visual distinction in the file tree.
[x] Swap tree colors to blue MD files + green folders and make tree selection highlight full-width.
[x] Ensure selected tree rows highlight both the full row and row text clearly.
[x] Make the bottom info/status footer more visible in split panes (high-contrast sticky-style bar).
[x] Prevent footer clipping by reserving the bottom line explicitly after pane rendering.
[x] Search currently matches note and folder names only (no note body/content search).

## Performance
[x] Improve performance on a fresh build.
[x] Consider adding an incremental/cached content index for search to keep `Ctrl+P` fast with large note collections.
[ ] Add a search-index benchmark suite (cold build vs warm query; small vs large note sets) to guard future regressions.

## Developer Experiqence
[x] Improve developer experience by adding more detailed documentation.
[x] Add more tests to ensure the application works as expected.
[x] Improve error handling and logging.
[ ] Improve code readability and maintainability.
[x] Improve code organization and modularity.
[x] Add inline documentation to `cmd/notes/main.go`.
[x] Add CONTRIBUTING.md with PR and code style guidelines.

## Testing
[x] Implement testing framework
[x] Add unit tests for tree building and render cache behavior in `internal/app`.
[x] Add unit tests for config load/save and first-run configurator behavior.
[ ] Add tests for filesystem failure paths (permission/read/write errors) to verify user status messages and log emission.

# Tasks

## Open Follow-up Tasks

- [ ] Allow vertical selection when highlighting text
- [ ] **Dynamic keybinding legends**: Help/footer key legends are still static strings and do not reflect user-customized keymaps. Render these hints from the active action->keys map.

## New Feature Ideas

- [ ] **`--version` CLI flag**: Add a `--version` flag that prints the build version/commit hash, useful for issue reporting and debugging. Wire it into the CI build with `-ldflags`.
- [ ] **Search result count indicator**: Show "N matches" or "M of N" position info in the Ctrl+P search popup so the user knows how many results matched and where they are in the list.
- [ ] **Scrollable help panel**: The `?` help screen is truncated at terminal height. Make it scrollable (viewport-based) so all keybindings are accessible on small terminals.
- [ ] **Configurable file watcher interval**: Expose `FileWatchInterval` (currently hardcoded at 2s) as a config option for users on slow network mounts or with very large directories who want to tune poll frequency.
- [ ] **Note word count goals / progress indicator**: Allow users to set a target word count per note (via frontmatter, e.g. `word_goal: 500`) and display a progress bar or percentage in the footer during editing.
- [ ] **Fuzzy search matching**: Upgrade Ctrl+P search from substring matching to fuzzy matching (e.g. "ntmd" matching "notes-metadata.md") for faster navigation in large workspaces.
- [ ] **Trash / soft delete**: Instead of permanent deletion, move deleted notes to a `.cli-notes/.trash/` directory with a configurable retention period, giving users a recovery window.
- [ ] **Bulk operations**: Add multi-select (e.g. via space bar or visual range) for batch delete, move, or tag operations across multiple notes.
- [ ] **Note backlinks panel**: Show a "referenced by" list for the current note — all other notes that contain a `[[wiki link]]` pointing to it — useful for building a personal knowledge graph.

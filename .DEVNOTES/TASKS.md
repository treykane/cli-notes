# Tasks

## Open Follow-up Tasks

- [ ] Allow vertical selection when highlighting text
- [ ] **Selection highlight precision**: Switch highlight rendering from substring matching to offset-aware spans to avoid incorrect highlights when selected text repeats.

## New Feature Ideas

- [ ] **Note word count goals / progress indicator**: Allow users to set a target word count per note (via frontmatter, e.g. `word_goal: 500`) and display a progress bar or percentage in the footer during editing.
- [ ] **Fuzzy search matching**: Upgrade Ctrl+P search from substring matching to fuzzy matching (e.g. "ntmd" matching "notes-metadata.md") for faster navigation in large workspaces.
- [ ] **Trash / soft delete**: Instead of permanent deletion, move deleted notes to a `.cli-notes/.trash/` directory with a configurable retention period, giving users a recovery window.
- [ ] **Bulk operations**: Add multi-select (e.g. via space bar or visual range) for batch delete, move, or tag operations across multiple notes.
- [ ] **Note backlinks panel**: Show a "referenced by" list for the current note — all other notes that contain a `[[wiki link]]` pointing to it — useful for building a personal knowledge graph.

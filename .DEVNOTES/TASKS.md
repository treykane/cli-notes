# Tasks

## Open Follow-up Tasks

- [ ] Improve mouse-selection precision for soft-wrapped lines in the textarea (current mapping uses a simplified visual-row projection).
- [ ] Add a dedicated `overlay_test.go` covering every overlay transition and ensuring `openOverlay`/`closeOverlay` cleanup stays complete as new overlays are added.
- [ ] Add mutation-pipeline unit tests for `applyMutationEffects` covering path upsert/remove/invalidate ordering and `setCurrentFile` command behavior.
- [ ] Add targeted benchmarks for tree rebuilds with metadata cache enabled vs. cold cache to track large-workspace performance over time.

## Code Quality & Optimization

- [ ] **Consolidate `handleBrowseKey` key dispatch**: Many keys (arrows, g/G, enter, h/l, ctrl+p, ctrl+o, o, etc.) are matched by direct string comparison before the action-based keybinding system is consulted, which means custom keybindings cannot override them. Route all browse-mode keys through the `actionForKey` dispatch so the configurable keybinding system is authoritative for every action.
- [ ] **Cap or evict glamour `rendererCache`**: The global `rendererCache` map (render.go) grows one entry per unique width bucket and is never pruned. Add a bounded LRU or cap to prevent memory growth when the terminal is resized frequently.
- [ ] **Avoid double `readMarkdownContentAndMetadata` during uncached tree builds**: When `buildTree` is called without the metadata cache callback (e.g. in test helpers), every markdown file is read from disk inline. Ensure all production call paths use `buildTreeWithMetadataCache` with the `cachedTagsForPath` callback.
- [ ] **Reduce map iteration in `removeDescendants`**: `searchIndex.removeDescendants` iterates the entire `docs` map with `strings.HasPrefix`. For large workspaces (10k+ files), consider a trie or sorted-slice approach to speed up prefix removal.
- [ ] **Consolidate duplicate search popup key handling**: `handleSearchKey` has separate cases for `"up"/"k"` and `"ctrl+p"` (both call `moveSearchCursor(-1)`) and similar for down. Merge these into combined match arms for clarity.

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
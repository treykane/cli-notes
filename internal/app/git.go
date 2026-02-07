package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// gitRepoStatus holds the current state of the git repository that contains
// the notes directory (if any).
//
// This information is refreshed after every git operation (commit, pull, push)
// and after tree refreshes. It is displayed in the footer status bar to give
// the user a quick at-a-glance view of their repository state without leaving
// the app.
//
// If the notes directory is not inside a git repository, isRepo will be false
// and all other fields are zero-valued.
type gitRepoStatus struct {
	// isRepo is true when the notes directory is inside a git work tree.
	// All other fields are only meaningful when this is true.
	isRepo bool

	// branch is the current branch name (e.g. "main", "feature/xyz"), or
	// "(detached)" when HEAD is not on a named branch.
	branch string

	// hasUpstream is true when the current branch has a configured remote
	// tracking branch. When false, ahead/behind counts are not available.
	hasUpstream bool

	// ahead is the number of local commits not yet pushed to the upstream
	// tracking branch. Only meaningful when hasUpstream is true.
	ahead int

	// behind is the number of remote commits not yet pulled into the local
	// branch. Only meaningful when hasUpstream is true.
	behind int

	// dirty is true when the working tree has uncommitted changes (tracked
	// modifications, staged changes, or untracked files).
	dirty bool

	// lastError holds the most recent error message from a git status
	// command, if any. It is displayed as a "status-error" indicator in
	// the footer.
	lastError string
}

// refreshGitStatus queries the git repository state for the current notes
// directory and updates m.git with the results.
//
// The function performs two git commands:
//  1. "git rev-parse --is-inside-work-tree" — to determine if the notes dir
//     is inside a git repo at all. If not, m.git is reset and the function
//     returns early.
//  2. "git status --porcelain=1 --branch" — to extract the branch name,
//     upstream tracking info (ahead/behind counts), and dirty state.
//
// Any errors from the status command are stored in m.git.lastError rather
// than surfaced to the user, since git integration is optional and
// non-critical.
func (m *Model) refreshGitStatus() {
	m.git = gitRepoStatus{}

	out, err := m.runGit("rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(out) != "true" {
		return
	}

	m.git.isRepo = true
	branch, branchErr := m.runGit("rev-parse", "--abbrev-ref", "HEAD")
	if branchErr == nil {
		m.git.branch = strings.TrimSpace(branch)
	}

	statusOut, statusErr := m.runGit("status", "--porcelain=1", "--branch")
	if statusErr != nil {
		m.git.lastError = firstLine(statusOut)
		if m.git.lastError == "" {
			m.git.lastError = statusErr.Error()
		}
		appLog.Warn("read git status", "error", statusErr, "output", statusOut)
		return
	}

	lines := strings.Split(strings.TrimSpace(statusOut), "\n")
	if len(lines) > 0 {
		m.git.hasUpstream, m.git.ahead, m.git.behind = parseGitPorcelainBranchLine(lines[0])
	}
	// If there are any lines beyond the branch header, the working tree has
	// uncommitted changes (dirty state).
	if len(lines) > 1 {
		m.git.dirty = true
	}
}

// parseGitPorcelainBranchLine extracts upstream tracking information from
// the first line of "git status --porcelain=1 --branch" output.
//
// The branch line has the format:
//
//	## main...origin/main [ahead 2, behind 1]
//
// The function parses:
//   - hasUpstream: true if "..." is present (indicates a tracking branch).
//   - ahead: the number after "ahead " in the bracketed section, or 0.
//   - behind: the number after "behind " in the bracketed section, or 0.
func parseGitPorcelainBranchLine(line string) (bool, int, int) {
	line = strings.TrimSpace(strings.TrimPrefix(line, "##"))
	if line == "" {
		return false, 0, 0
	}

	hasUpstream := strings.Contains(line, "...")
	ahead := parseGitCount(line, "ahead ")
	behind := parseGitCount(line, "behind ")
	return hasUpstream, ahead, behind
}

// parseGitCount extracts a numeric count that follows the given token string
// within a line. For example, parseGitCount("ahead 3, behind 1", "ahead ")
// returns 3.
//
// This is used to extract ahead/behind counts from the git status branch
// line. Returns 0 if the token is not found or the following text is not a
// valid integer.
func parseGitCount(line, token string) int {
	idx := strings.Index(line, token)
	if idx < 0 {
		return 0
	}
	start := idx + len(token)
	end := start
	for end < len(line) && line[end] >= '0' && line[end] <= '9' {
		end++
	}
	if end == start {
		return 0
	}
	n, err := strconv.Atoi(line[start:end])
	if err != nil {
		return 0
	}
	return n
}

// gitFooterSummary produces a compact, human-readable summary string of the
// current git repository state for display in the footer status bar.
//
// The output format is:
//
//	"git main ↑2 ↓0 clean"       (on branch "main", 2 ahead, 0 behind, clean)
//	"git (detached) no-upstream dirty"  (detached HEAD, no tracking, dirty)
//
// Returns an empty string if the notes directory is not inside a git
// repository, which causes the footer to omit the git section entirely.
func (m *Model) gitFooterSummary() string {
	if !m.git.isRepo {
		return ""
	}
	parts := []string{"git"}
	branch := m.git.branch
	if branch == "" {
		branch = "(detached)"
	}
	parts = append(parts, branch)

	if m.git.hasUpstream {
		parts = append(parts, fmt.Sprintf("↑%d", m.git.ahead))
		parts = append(parts, fmt.Sprintf("↓%d", m.git.behind))
	} else {
		parts = append(parts, "no-upstream")
	}
	if m.git.dirty {
		parts = append(parts, "dirty")
	} else {
		parts = append(parts, "clean")
	}
	if m.git.lastError != "" {
		parts = append(parts, "status-error")
	}
	return strings.Join(parts, " ")
}

// ---------------------------------------------------------------------------
// Git action handlers
// ---------------------------------------------------------------------------

// handleGitCommitStart enters the git commit input mode, prompting the user
// to type a commit message.
//
// The text input is pre-populated with a default message that includes the
// current date and time (e.g. "Update notes (2025-02-07 14:30)"). The user
// can accept the default by pressing Enter/Ctrl+S or type a custom message.
//
// If the notes directory is not a git repository, a status message is shown
// and no mode change occurs.
func (m *Model) handleGitCommitStart() (tea.Model, tea.Cmd) {
	if !m.git.isRepo {
		m.status = "Git is unavailable for this notes directory"
		return m, nil
	}
	m.mode = modeGitCommit
	m.showHelp = false
	m.input.Reset()
	m.input.Placeholder = "Commit message"
	m.input.SetValue(m.defaultCommitMessage())
	m.input.CursorEnd()
	m.input.Focus()
	m.status = "Git commit: Enter or Ctrl+S to commit, Esc to cancel"
	return m, nil
}

// handleGitPull runs "git pull --ff-only" in the notes directory.
//
// The --ff-only flag ensures that only fast-forward merges are performed,
// which avoids creating merge commits or triggering conflict resolution.
// If the pull introduces changes, the tree view, search index, render cache,
// and git status are all refreshed to reflect the new state.
//
// If the pull fails (e.g. due to divergent histories, network errors, or
// authentication failures), the error is shown in the status bar and logged.
func (m *Model) handleGitPull() (tea.Model, tea.Cmd) {
	if !m.git.isRepo {
		m.status = "Git is unavailable for this notes directory"
		return m, nil
	}

	out, err := m.runGit("pull", "--ff-only")
	if err != nil {
		m.status = "Git pull failed: " + firstLine(out)
		if strings.TrimSpace(firstLine(out)) == "" {
			m.status = "Git pull failed: " + err.Error()
		}
		appLog.Warn("git pull failed", "error", err, "output", out)
		m.refreshGitStatus()
		return m, nil
	}

	m.status = "Git pull complete"
	if line := firstLine(out); line != "" {
		m.status = "Git pull: " + line
	}

	// After a successful pull, external files may have changed. Invalidate
	// all caches and rebuild the tree to pick up any new, modified, or
	// deleted notes.
	m.searchIndex.invalidate()
	m.refreshTree()
	m.reconcileCurrentFileAfterFilesystemChange()
	m.refreshGitStatus()
	if m.currentFile != "" {
		return m, m.setCurrentFile(m.currentFile)
	}
	return m, nil
}

// handleGitPush runs "git push" in the notes directory to push local commits
// to the configured remote.
//
// This is a straightforward push with no flags. If the push fails (e.g. due
// to rejected updates, network errors, or authentication issues), the error
// is shown in the status bar. On success, the git status is refreshed to
// update the ahead/behind counts.
func (m *Model) handleGitPush() (tea.Model, tea.Cmd) {
	if !m.git.isRepo {
		m.status = "Git is unavailable for this notes directory"
		return m, nil
	}

	out, err := m.runGit("push")
	if err != nil {
		m.status = "Git push failed: " + firstLine(out)
		if strings.TrimSpace(firstLine(out)) == "" {
			m.status = "Git push failed: " + err.Error()
		}
		appLog.Warn("git push failed", "error", err, "output", out)
		m.refreshGitStatus()
		return m, nil
	}

	m.status = "Git push complete"
	if line := firstLine(out); line != "" {
		m.status = "Git push: " + line
	}
	m.refreshGitStatus()
	return m, nil
}

// runGitCommit executes a two-step commit: "git add -A" (stage everything)
// followed by "git commit -m <message>".
//
// If the provided message is empty or whitespace-only, a default commit
// message with the current timestamp is used.
//
// The function handles common outcomes:
//   - Successful commit: status bar shows the commit message.
//   - "nothing to commit": recognized as a non-error condition and reported
//     calmly in the status bar.
//   - Add or commit failure: error details are shown in the status bar and
//     logged.
//
// After the commit (whether successful or not), the app returns to browse
// mode and the git status is refreshed.
func (m *Model) runGitCommit(message string) (tea.Model, tea.Cmd) {
	if !m.git.isRepo {
		m.mode = modeBrowse
		m.status = "Git is unavailable for this notes directory"
		return m, nil
	}

	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = m.defaultCommitMessage()
	}

	// Stage all changes (new files, modifications, deletions).
	if out, err := m.runGit("add", "-A"); err != nil {
		m.status = "Git add failed: " + firstLine(out)
		if strings.TrimSpace(firstLine(out)) == "" {
			m.status = "Git add failed: " + err.Error()
		}
		appLog.Warn("git add failed", "error", err, "output", out)
		m.refreshGitStatus()
		return m, nil
	}

	// Create the commit with the user's (or default) message.
	out, err := m.runGit("commit", "-m", msg)
	m.mode = modeBrowse
	if err != nil {
		line := strings.ToLower(firstLine(out))
		switch {
		case strings.Contains(line, "nothing to commit"):
			// Not a real error — just nothing staged to commit.
			m.status = "Nothing to commit"
		default:
			m.status = "Git commit failed: " + firstLine(out)
			if strings.TrimSpace(firstLine(out)) == "" {
				m.status = "Git commit failed: " + err.Error()
			}
			appLog.Warn("git commit failed", "error", err, "output", out)
		}
		m.refreshGitStatus()
		return m, nil
	}

	m.status = "Committed: " + msg
	m.refreshGitStatus()
	return m, nil
}

// defaultCommitMessage generates a timestamped commit message used as the
// default value in the commit message input and as a fallback when the user
// submits an empty message.
//
// Format: "Update notes (2025-02-07 14:30)"
func (m *Model) defaultCommitMessage() string {
	return fmt.Sprintf("Update notes (%s)", time.Now().Format("2006-01-02 15:04"))
}

// ---------------------------------------------------------------------------
// Git execution helpers
// ---------------------------------------------------------------------------

// runGit executes a git command with the notes directory as the working
// directory (via "git -C <notesDir> <args...>").
//
// It captures both stdout and stderr, merging them into a single output
// string. This merged output is used for error reporting in the status bar,
// since git sometimes writes important information to stderr (e.g. progress
// messages, error details).
//
// Returns the combined output and any execution error. The caller should
// inspect both — a non-nil error with informative output is common for git
// commands that fail with explanatory messages.
func (m *Model) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", m.notesDir}, args...)...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	errOut := strings.TrimSpace(stderr.String())

	// Merge stdout and stderr into a single output string for the caller.
	switch {
	case out != "" && errOut != "":
		out = out + "\n" + errOut
	case out == "":
		out = errOut
	}
	return out, err
}

// ---------------------------------------------------------------------------
// Filesystem reconciliation
// ---------------------------------------------------------------------------

// reconcileCurrentFileAfterFilesystemChange checks whether the currently
// displayed note file still exists on disk after an external change (e.g.
// git pull, manual file deletion).
//
// If the file has been deleted or is no longer a regular file, the viewport
// is reset to the "Select a note to view" placeholder and the current file
// reference is cleared. If the file still exists, no action is taken.
//
// This prevents the app from showing stale content or crashing when the
// underlying file disappears.
func (m *Model) reconcileCurrentFileAfterFilesystemChange() {
	if m.currentFile == "" {
		return
	}
	info, err := os.Stat(m.currentFile)
	if err == nil && !info.IsDir() {
		return
	}
	m.currentFile = ""
	m.viewport.SetContent("Select a note to view")
}

// firstLine extracts the first non-empty line from a (possibly multi-line)
// text string. This is used to produce concise status bar messages from
// verbose git command output.
//
// Returns an empty string if the input is empty or whitespace-only.
func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	return strings.TrimSpace(lines[0])
}

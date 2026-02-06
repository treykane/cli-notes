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

type gitRepoStatus struct {
	isRepo      bool
	branch      string
	hasUpstream bool
	ahead       int
	behind      int
	dirty       bool
	lastError   string
}

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
	if len(lines) > 1 {
		m.git.dirty = true
	}
}

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
	m.searchIndex.invalidate()
	m.refreshTree()
	m.reconcileCurrentFileAfterFilesystemChange()
	m.refreshGitStatus()
	if m.currentFile != "" {
		return m, m.setCurrentFile(m.currentFile)
	}
	return m, nil
}

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

	if out, err := m.runGit("add", "-A"); err != nil {
		m.status = "Git add failed: " + firstLine(out)
		if strings.TrimSpace(firstLine(out)) == "" {
			m.status = "Git add failed: " + err.Error()
		}
		appLog.Warn("git add failed", "error", err, "output", out)
		m.refreshGitStatus()
		return m, nil
	}

	out, err := m.runGit("commit", "-m", msg)
	m.mode = modeBrowse
	if err != nil {
		line := strings.ToLower(firstLine(out))
		switch {
		case strings.Contains(line, "nothing to commit"):
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

func (m *Model) defaultCommitMessage() string {
	return fmt.Sprintf("Update notes (%s)", time.Now().Format("2006-01-02 15:04"))
}

func (m *Model) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", m.notesDir}, args...)...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	errOut := strings.TrimSpace(stderr.String())
	switch {
	case out != "" && errOut != "":
		out = out + "\n" + errOut
	case out == "":
		out = errOut
	}
	return out, err
}

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

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	return strings.TrimSpace(lines[0])
}

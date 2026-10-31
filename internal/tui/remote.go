package tui

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type remoteMsg struct {
	host, path, text string
	err              error
}

// shq single-quotes s for a POSIX shell.
func shq(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// fetchRemote cats $HOME/<path> on host (key-based ssh only; never prompts).
func fetchRemote(host, path string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "--", host, `cat -- "$HOME"/`+shq(path)).CombinedOutput()
		return remoteMsg{host, path, string(out), err}
	}
}

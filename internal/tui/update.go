package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) Init() tea.Cmd { return nil }

func (m model) listW() int { return max(24, m.w/3) }
func (m model) bodyH() int { return m.h - 4 }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch t := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = t.Width, t.Height
		m.vp.Width, m.vp.Height = t.Width-m.listW()-5, t.Height-10
		m.showDiff()
	case opMsg:
		m.msg = string(t)
		m.load()
	case editDoneMsg:
		m.msg = "edited"
		if t.err != nil {
			m.msg = "edit: " + t.err.Error()
		}
		m.load()
	case remoteMsg:
		if m.remote[t.path] == nil {
			m.remote[t.path] = map[string]remoteMsg{}
		}
		m.remote[t.path][t.host] = t
		m.showDiff()
	case tea.MouseMsg:
		return m.mouse(t)
	case tea.KeyMsg:
		return m.key(t)
	}
	return m, nil
}

func (m model) mouse(t tea.MouseMsg) (tea.Model, tea.Cmd) {
	onList := t.X < m.listW()
	wheel := t.Button == tea.MouseButtonWheelUp || t.Button == tea.MouseButtonWheelDown
	switch {
	case wheel && onList:
		d := 1
		if t.Button == tea.MouseButtonWheelUp {
			d = -1
		}
		m.cur = max(0, min(m.cur+d, len(m.items)-1))
		m.showDiff()
	case wheel:
		var c tea.Cmd
		m.vp, c = m.vp.Update(t)
		return m, c
	case t.Button == tea.MouseButtonLeft && t.Action == tea.MouseActionPress:
		if t.Y == 0 {
			x := lipgloss.Width(badge.Render("DOTGO"))
			for i, n := range tabNames {
				l := len(fmt.Sprintf("  [%d] %s", i+1, n))
				if t.X >= x && t.X < x+l {
					m.setTab(i)
					break
				}
				x += l
			}
		} else if t.Y >= 2 && onList {
			rs, cr := m.rows()
			if ri := m.listStart(cr) + t.Y - 2; ri >= 0 && ri < len(rs) && rs[ri].idx >= 0 {
				m.cur = rs[ri].idx
				m.showDiff()
			}
		}
	}
	return m, nil
}

func (m model) key(t tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := t.String()
	if m.confirm != nil {
		c := m.confirm
		m.confirm = nil
		if k == "y" {
			m.msg = c.fn()
			m.load()
		} else {
			m.msg = "cancelled"
			m.showDiff()
		}
		return m, nil
	}
	m.msg = ""
	if m.prompt != "" {
		return m.promptKey(t)
	}
	if m.help {
		m.help = false
		return m, nil
	}
	var it item
	if len(m.items) > 0 {
		it = m.items[m.cur]
	}
	has := len(m.items) > 0
	switch k {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.help = true
	case "1", "2", "3", "4", "5", "6":
		m.setTab(int(k[0] - '1'))
	case "j", "down":
		m.cur = min(m.cur+1, max(0, len(m.items)-1))
		m.showDiff()
	case "k", "up":
		m.cur = max(m.cur-1, 0)
		m.showDiff()
	case "g":
		m.cur = 0
		m.showDiff()
	case "G":
		m.cur = max(0, len(m.items)-1)
		m.showDiff()
	case "/":
		m.prompt = "search"
		m.in.SetValue(m.filter)
		m.in.Focus()
		return m, textinput.Blink
	case "esc":
		m.filter, m.mark = "", ""
		m.applyFilter()
	case "r":
		m.load()
	case "s":
		m.sbs = !m.sbs
		m.showDiff()
	case "n":
		m.pair++
		m.showDiff()
	case "]":
		m.moveHunk(1)
	case "[":
		m.moveHunk(-1)
	case "c", "a":
		m.prompt = map[string]string{"c": "commit", "a": "add"}[k]
		m.in.Focus()
		return m, textinput.Blink
	case "e":
		if has && m.tab != tLog {
			return m, m.edit(m.editTarget(it.key))
		}
	case " ":
		if m.tab == tChanges && has {
			if it.y == ' ' { // fully staged -> unstage
				m.msg = m.do("restore", "--staged", "--", it.key)
			} else {
				m.tryAdd(it.key)
			}
			m.load()
		}
	case "S": // stage/unstage the selected hunk
		if m.tab == tChanges && m.hc < len(m.hunks) {
			h := m.hunks[m.hc]
			args := []string{"apply", "--cached"}
			if h.Staged {
				args = append(args, "-R")
			}
			out, err := m.Git.RunIn(h.Patch(), args...)
			m.msg = "hunk ok"
			if err != nil {
				m.msg = "apply: " + oneLine(out)
			}
			m.load()
		}
	case "x":
		if m.tab == tChanges && has {
			p, g := it.key, m.Git
			m.confirm = &confirmT{"discard ALL changes (staged+unstaged) to " + p + "?", func() string {
				if out, err := g.Run("restore", "--source=HEAD", "--staged", "--worktree", "--", p); err != nil {
					return "git restore: " + oneLine(out)
				}
				return "discarded " + p
			}}
		}
	case "A":
		if has && m.tab != tLog {
			base, _, _ := strings.Cut(it.key, "##")
			m.startApply(base)
		}
	case "l":
		if has && m.tab != tLog {
			m.logPath = it.key
			m.tab, m.cur = tLog, 0
			m.load()
		}
	case "L":
		m.logPath, m.tab, m.cur = "", tLog, 0
		m.load()
	case "m":
		if has && m.tab == tLog {
			if m.mark == it.key {
				m.mark = ""
			} else {
				m.mark = it.key
			}
			m.showDiff()
		}
	case "o":
		if has && m.tab == tRemote && len(m.Remotes) > 0 {
			m.remote[it.key] = map[string]remoteMsg{}
			var cs []tea.Cmd
			for _, h := range m.Remotes {
				cs = append(cs, fetchRemote(h, it.key))
			}
			m.showDiff()
			return m, tea.Batch(cs...)
		}
	case "p":
		m.msg = "pushing…"
		g := m.Git
		return m, func() tea.Msg { o, _ := g.Push(); return opMsg(oneLine(o)) }
	case "P":
		m.msg = "pulling…"
		g := m.Git
		return m, func() tea.Msg {
			o, err := g.Pull()
			if err != nil {
				o = "pull failed, rebase aborted: " + o
			}
			return opMsg(oneLine(o))
		}
	default:
		var c tea.Cmd
		m.vp, c = m.vp.Update(t)
		return m, c
	}
	return m, nil
}

func (m model) promptKey(t tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch t.String() {
	case "enter":
		v := strings.TrimSpace(m.in.Value())
		pr := m.prompt
		m.prompt = ""
		m.in.Reset()
		switch {
		case pr == "search":
			m.filter, m.cur = v, 0
			m.applyFilter()
			return m, nil
		case v == "":
			m.msg = "empty input"
		case pr == "commit":
			m.msg = m.do("commit", "-m", v)
		case pr == "add":
			m.tryAdd(v)
		}
		m.load()
		return m, nil
	case "esc":
		m.prompt = ""
		m.in.Reset()
		return m, nil
	}
	var c tea.Cmd
	m.in, c = m.in.Update(t)
	return m, c
}

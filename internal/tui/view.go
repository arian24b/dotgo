package tui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const helpText = `
  Navigation   j/k move · g/G top/bottom · 1-6 tabs (or click) · / search · esc clear · r reload · mouse wheel/click
  Edit         e open the file (or this machine's active variant) in $VISUAL/$EDITOR/vi
  Changes      space stage/unstage file · ] [ pick hunk · S stage/unstage hunk · x discard (confirm)
               c commit · a add new file by path · p push · P pull --rebase --autostash
  Compare      s side-by-side ⇄ unified · n next variant pair (Variants tab)
  Variants     A apply this machine's ##host.<name> / ##os.<os> variant (preview + confirm, backup kept)
  History      l log of selected file · L log of all · m mark a commit, then move to diff against it
  Remote       o fetch selected file over ssh from ssh_hosts and diff against local
  Safety       secret-looking files ask before staging; destructive actions always preview + y/n
  Other        ? this help · q quit                                  (press any key to close)
`

var tabHints = [...]string{
	"space stage · S hunk · ]/[ · x discard · c commit · a add · e edit · s side-by-side",
	"s side-by-side · e edit · p push · P pull",
	"A apply variant · e edit · n next pair · s side-by-side",
	"l history · e edit · A apply variant",
	"m mark · l/L",
	"o fetch via ssh · e edit",
}

func (m model) View() string {
	if m.w == 0 {
		return ""
	}
	bodyH := m.bodyH()
	tabs := ""
	for i, n := range tabNames {
		s := fmt.Sprintf("  [%d] %s", i+1, n)
		if i == m.tab {
			s = accent.Render(s)
		} else {
			s = dim.Render(s)
		}
		tabs += s
	}
	left := badge.Render("DOTGO") + tabs
	info := fmt.Sprintf("%s · %s · %s", m.Host, runtime.GOOS, m.br)
	if m.filter != "" {
		info = "/" + m.filter + " · " + info
	}
	info = dim.Render(info)
	head := ansiTrunc(left+strings.Repeat(" ", max(1, m.w-lipgloss.Width(left)-lipgloss.Width(info)-1))+info, m.w)
	rule := dim.Render(strings.Repeat("─", m.w))

	var body string
	if m.help {
		body = lipgloss.NewStyle().Height(bodyH).Render(accent.Render("\n  dotgo help") + "\n" + helpText)
	} else {
		body = m.body(bodyH)
	}

	foot := dim.Render(tabHints[m.tab] + " · / search · ? help · q quit")
	switch {
	case m.confirm != nil:
		foot = yellow.Render(m.confirm.text) + dim.Render("  [y/n]")
	case m.prompt != "":
		foot = accent.Render(m.prompt+" › ") + m.in.View()
	case m.msg != "":
		foot = oneLine(m.msg)
	}
	return head + "\n" + rule + "\n" + body + "\n" + rule + "\n" + ansiTrunc(foot, m.w-1)
}

func (m model) rows() (rs []row, curRow int) {
	if m.tab == tLog {
		for i := range m.items {
			rs = append(rs, row{idx: i})
		}
		return rs, m.cur
	}
	last, n := "\x00", map[string]int{}
	for _, it := range m.items {
		n[group(it.key)]++
	}
	for i, it := range m.items {
		g := group(it.key)
		if g != last {
			rs = append(rs, row{head: fmt.Sprintf("▾ %s (%d)", g, n[g]), idx: -1})
			last = g
		}
		if i == m.cur {
			curRow = len(rs)
		}
		rs = append(rs, row{idx: i})
	}
	return
}

func (m model) listStart(cr int) int { return max(0, cr-m.bodyH()+2) }

func (m model) body(bodyH int) string {
	rs, cr := m.rows()
	var lines []string
	for i := m.listStart(cr); i < len(rs) && len(lines) < bodyH; i++ {
		r := rs[i]
		var l string
		if r.idx < 0 {
			l = accent.Render(r.head)
		} else {
			it := m.items[r.idx]
			mark := " "
			switch {
			case m.tab == tLog && it.key == m.mark:
				mark = accent.Render("◆")
			case m.tab != tChanges:
			case it.x != ' ' && it.x != '?':
				mark = green.Render("●")
			case it.y != ' ':
				mark = yellow.Render("●")
			}
			name := it.label
			if m.tab != tLog {
				if _, rest, ok := strings.Cut(it.label, "/"); ok {
					name = rest
				}
			}
			l = "  " + mark + " " + ansiTrunc(name, m.listW()-6)
		}
		if r.idx == m.cur && r.idx >= 0 {
			l = selRow.Width(m.listW()).Render(l)
		}
		lines = append(lines, l)
	}
	list := lipgloss.NewStyle().Width(m.listW()).Height(bodyH).Render(strings.Join(lines, "\n"))

	detail := dim.Render("nothing to show")
	if len(m.items) > 0 {
		it := m.items[m.cur]
		title := it.key
		if m.tab == tLog {
			title = it.label
		}
		kv := func(s string) string {
			k, v, _ := strings.Cut(s, "  ")
			return dim.Render(fmt.Sprintf("%-9s", k)) + strings.TrimSpace(v)
		}
		detail = accent.Render(ansiTrunc(title, m.vp.Width)) + "\n" +
			dim.Render(m.Host+" · "+tabNames[m.tab]) + "\n\n" +
			kv(m.kv[0]) + "\n" + kv(m.kv[1]) + "\n" +
			m.vp.View()
	}
	right := lipgloss.NewStyle().Width(m.w-m.listW()-3).Height(bodyH).
		Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#2a2f45")).
		PaddingLeft(1).Render(detail)
	return lipgloss.JoinHorizontal(lipgloss.Top, list, right)
}

// Package tui is the bubbletea interface: model (state), update (keys), view (layout).
package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/arian24b/dotgo/internal/diff"
	"github.com/arian24b/dotgo/internal/gitx"
	"github.com/arian24b/dotgo/internal/secrets"
	"github.com/arian24b/dotgo/internal/variant"
)

const (
	tChanges = iota
	tUpstream
	tVariants
	tAll
	tLog
	tRemote
)

var tabNames = []string{"Changes", "Upstream", "Variants", "All", "Log", "Remote"}

// Options configures the TUI.
type Options struct {
	Git      *gitx.Client
	WorkTree string
	Host     string
	Editor   []string // command + args, e.g. ["vi"]
	Link     bool     // apply variants as symlinks
	Remotes  []string // ssh hosts for the Remote tab
}

type item struct {
	key, label string
	x, y       byte
}

type row struct {
	head string
	idx  int
}

type confirmT struct {
	text string
	fn   func() string
}

type opMsg string
type editDoneMsg struct{ err error }

type model struct {
	Options
	br string

	tab        int
	all, items []item
	cur        int
	filter     string
	pair       int
	sbs        bool
	hunks      []diff.Hunk
	hc         int
	hLine      []int
	mark       string // Log: commit marked for a two-commit diff
	logPath    string // Log: restrict to one file
	kv         [2]string
	remote     map[string]map[string]remoteMsg // path -> host -> result

	vp      viewport.Model
	in      textinput.Model
	prompt  string // "", "commit", "add", "search"
	confirm *confirmT
	help    bool
	msg     string
	w, h    int
}

// Run starts the TUI and blocks until the user quits.
func Run(o Options) error {
	m := model{Options: o, in: textinput.New(), vp: viewport.New(0, 0), remote: map[string]map[string]remoteMsg{}}
	m.load()
	_, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

// ---- loading ----

func (m *model) load() {
	var all []item
	switch m.tab {
	case tChanges:
		for _, e := range m.Git.Status() {
			all = append(all, item{e.Path, e.Path, e.X, e.Y})
		}
	case tUpstream:
		out, err := m.Git.Run("diff", "--name-only", "-z", "@{u}")
		if err != nil {
			m.msg = "no upstream: " + oneLine(out)
			break
		}
		for _, p := range strings.Split(out, "\x00") {
			if p != "" {
				all = append(all, item{p, p, ' ', ' '})
			}
		}
	case tVariants:
		for _, p := range m.Git.VariantBases() {
			all = append(all, item{key: p, label: p})
		}
	case tAll, tRemote:
		for _, p := range m.Git.LsFiles() {
			all = append(all, item{key: p, label: p})
		}
	case tLog:
		args := []string{"log", "--pretty=format:%h\t%s\t%ar", "-n", "500"}
		if m.logPath != "" {
			args = append(args, "--", m.logPath)
		}
		out, _ := m.Git.Run(args...)
		for _, l := range strings.Split(out, "\n") {
			if f := strings.SplitN(l, "\t", 3); len(f) == 3 {
				all = append(all, item{key: f[0], label: f[0] + " " + f[1] + dim.Render("  "+f[2])})
			}
		}
	}
	if m.tab != tLog {
		sort.SliceStable(all, func(i, j int) bool { // root files first, then by path (dirs contiguous)
			a, b := all[i].key, all[j].key
			if ai, bi := strings.Contains(a, "/"), strings.Contains(b, "/"); ai != bi {
				return !ai
			}
			return a < b
		})
	}
	m.all = all
	m.applyFilter()
	m.br = m.Git.Branch()
}

func (m *model) applyFilter() {
	m.items = nil
	f := strings.ToLower(m.filter)
	for _, it := range m.all {
		if f == "" || strings.Contains(strings.ToLower(it.label), f) {
			m.items = append(m.items, it)
		}
	}
	m.cur = max(0, min(m.cur, len(m.items)-1))
	m.showDiff()
}

func (m *model) setTab(t int) {
	if t < 0 || t >= len(tabNames) || t == m.tab {
		return
	}
	m.tab, m.cur, m.pair, m.filter = t, 0, 0, ""
	m.load()
}

func (m model) readLocal(p string) string {
	b, _ := os.ReadFile(filepath.Join(m.WorkTree, p))
	return string(b)
}

// ---- detail pane ----

func (m *model) showDiff() {
	m.hunks, m.hc, m.kv = nil, 0, [2]string{}
	if len(m.items) == 0 {
		m.vp.SetContent(dim.Render("nothing to show"))
		return
	}
	it := m.items[m.cur]
	p := it.key
	w := m.vp.Width
	var out string
	switch m.tab {
	case tChanges:
		m.kv[0] = "Status  " + strings.TrimSpace(fmt.Sprintf("%c%c", it.x, it.y))
		m.kv[1] = fmt.Sprintf("Variants  %d", max(0, len(m.Git.Variants(p))-1))
		if m.sbs {
			out = sideBySide(m.Git.Show("HEAD:"+p), m.readLocal(p), w)
			break
		}
		u, _ := m.Git.Run("diff", "--no-color", "--", p)
		s, _ := m.Git.Run("diff", "--no-color", "--cached", "--", p)
		m.hunks = append(diff.ParseHunks(u, false), diff.ParseHunks(s, true)...)
		if len(m.hunks) == 0 {
			out = colorize(strings.TrimSpace(u + "\n" + s))
		} else {
			out = m.renderHunks()
		}
	case tUpstream:
		m.kv[0] = "Against  @{u}"
		if m.sbs {
			out = sideBySide(m.Git.Show("@{u}:"+p), m.readLocal(p), w)
		} else {
			d, _ := m.Git.Run("diff", "--no-color", "@{u}", "--", p)
			out = colorize(d)
		}
	case tVariants:
		vs := m.Git.Variants(p)
		m.kv[0] = fmt.Sprintf("Variants  %d", max(0, len(vs)-1))
		m.kv[1] = "Active  " + orDash(variant.Pick(vs, m.Host, runtime.GOOS))
		if len(vs) < 2 {
			out = "no ##variants tracked (name them e.g. " + p + "##os.Darwin)"
			break
		}
		a, b := vs[m.pair%len(vs)], vs[(m.pair+1)%len(vs)]
		hdr := fmt.Sprintf("%s  ↔  %s   (n: next pair)\n\n", a, b)
		if m.sbs {
			out = hdr + sideBySide(m.Git.Show(":"+a), m.Git.Show(":"+b), w)
		} else {
			d, _ := m.Git.Run("diff", "--no-color", ":"+a, ":"+b)
			out = hdr + colorize(d)
		}
	case tAll:
		b := m.readLocal(p)
		ls := strings.Split(b, "\n")
		if len(ls) > 400 {
			ls = append(ls[:400], "…")
		}
		m.kv[0] = fmt.Sprintf("Size  %d B", len(b))
		out = strings.Join(ls, "\n")
	case tLog:
		m.kv[0] = "Marked  " + orDash(m.mark)
		m.kv[1] = "File  " + orDash(m.logPath)
		var d string
		if m.mark != "" && m.mark != p {
			d, _ = m.Git.Run("diff", "--no-color", m.mark, p, "--", orDot(m.logPath))
			d = fmt.Sprintf("diff %s..%s\n%s", m.mark, p, d)
		} else {
			d, _ = m.Git.Run("show", "--no-color", "--stat", "-p", p, "--", orDot(m.logPath))
		}
		out = colorize(d)
	case tRemote:
		out = m.remoteView(p)
	}
	if strings.TrimSpace(out) == "" {
		out = dim.Render("no differences")
	}
	m.vp.SetContent(clip(out, w))
	m.vp.GotoTop()
}

func (m model) remoteView(p string) string {
	if len(m.Remotes) == 0 {
		return dim.Render("no remotes: set ssh_hosts in the config or pass --ssh user@host[,user@host2]")
	}
	res := m.remote[p]
	if res == nil {
		return dim.Render("press o to fetch " + p + " from " + strings.Join(m.Remotes, ", "))
	}
	local := m.readLocal(p)
	var b []string
	for _, h := range m.Remotes {
		r, ok := res[h]
		b = append(b, accent.Render("── "+h+" ──"))
		switch {
		case !ok:
			b = append(b, dim.Render("fetching…"))
		case r.err != nil:
			b = append(b, red.Render("unavailable: "+oneLine(r.text)))
		default:
			d := diff.Unified(local, r.text, "local", h)
			if d == "" {
				d = dim.Render("identical")
			}
			b = append(b, colorize(d))
		}
	}
	return strings.Join(b, "\n")
}

func (m *model) renderHunks() string {
	var b []string
	m.hLine = m.hLine[:0]
	for i, h := range m.hunks {
		if i == 0 || h.Staged != m.hunks[i-1].Staged {
			t := "── unstaged ──"
			if h.Staged {
				t = "── staged ──"
			}
			b = append(b, accent.Render(t))
		}
		m.hLine = append(m.hLine, len(b))
		first, rest, _ := strings.Cut(h.Body, "\n")
		mk := "  "
		if i == m.hc {
			mk = accent.Render("▶ ")
		}
		b = append(b, mk+cyan.Render(first))
		for _, l := range strings.Split(colorize(rest), "\n") {
			b = append(b, "  "+l)
		}
	}
	return clip(strings.Join(b, "\n"), m.vp.Width)
}

func (m *model) moveHunk(d int) {
	if len(m.hunks) == 0 {
		return
	}
	m.hc = max(0, min(m.hc+d, len(m.hunks)-1))
	m.vp.SetContent(m.renderHunks())
	m.vp.SetYOffset(max(0, m.hLine[m.hc]-1))
}

// ---- actions ----

func (m model) do(a ...string) string {
	out, err := m.Git.Run(a...)
	if err != nil {
		return "git " + a[0] + ": " + oneLine(out)
	}
	return a[0] + " ok"
}

func (m *model) tryAdd(p string) {
	if secrets.Looks(p) {
		g := m.Git
		m.confirm = &confirmT{"looks like a secret: " + p + " — stage anyway?", func() string {
			if out, err := g.Run("add", "--", p); err != nil {
				return "git add: " + oneLine(out)
			}
			return "add ok"
		}}
		return
	}
	m.msg = m.do("add", "--", p)
}

func (m *model) startApply(base string) {
	vs := m.Git.Variants(base)
	src := variant.Pick(vs, m.Host, runtime.GOOS)
	if src == "" || src == base {
		m.msg = "no ##variant matches host " + m.Host + " / os " + runtime.GOOS
		return
	}
	d := diff.Unified(m.readLocal(base), m.readLocal(src), base+" (now)", src)
	if d == "" {
		d = "identical — nothing would change"
	}
	m.vp.SetContent(clip(colorize(d), m.vp.Width))
	verb := "copy"
	if m.Link {
		verb = "symlink"
	}
	dst, sp, link := filepath.Join(m.WorkTree, base), filepath.Join(m.WorkTree, src), m.Link
	m.confirm = &confirmT{fmt.Sprintf("%s %s → %s (backup kept as .dotgo-bak)?", verb, src, base), func() string {
		res, err := variant.Apply(sp, dst, link)
		if err != nil {
			return "apply: " + err.Error()
		}
		return res
	}}
}

// editTarget is the file `e` opens: the active variant on the Variants tab, else the item itself.
func (m model) editTarget(p string) string {
	if m.tab == tVariants {
		if v := variant.Pick(m.Git.Variants(p), m.Host, runtime.GOOS); v != "" {
			return v
		}
	}
	return p
}

func (m model) edit(p string) tea.Cmd {
	full := filepath.Join(m.WorkTree, p)
	if _, err := os.Stat(full); err != nil {
		return func() tea.Msg { return editDoneMsg{fmt.Errorf("%s is not in the work tree", p)} }
	}
	c := exec.Command(m.Editor[0], append(append([]string{}, m.Editor[1:]...), full)...)
	return tea.ExecProcess(c, func(err error) tea.Msg { return editDoneMsg{err} })
}

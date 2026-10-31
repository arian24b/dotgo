package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/arian24b/dotgo/internal/diff"
)

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func ansiTrunc(s string, w int) string { return lipgloss.NewStyle().MaxWidth(max(1, w)).Render(s) }

// clip truncates every line (ANSI-aware) so wide diffs can't break the layout.
func clip(s string, w int) string {
	ls := strings.Split(s, "\n")
	for i, l := range ls {
		ls[i] = ansiTrunc(l, w)
	}
	return strings.Join(ls, "\n")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func orDot(s string) string {
	if s == "" {
		return "."
	}
	return s
}

func group(p string) string {
	if g, _, ok := strings.Cut(p, "/"); ok {
		return g
	}
	return "~"
}

// colorize colors a plain unified diff.
func colorize(s string) string {
	ls := strings.Split(s, "\n")
	for i, l := range ls {
		switch {
		case strings.HasPrefix(l, "+++"), strings.HasPrefix(l, "---"), strings.HasPrefix(l, "diff "), strings.HasPrefix(l, "index "):
			ls[i] = dim.Render(l)
		case strings.HasPrefix(l, "@@"):
			ls[i] = cyan.Render(l)
		case strings.HasPrefix(l, "+"):
			ls[i] = green.Render(l)
		case strings.HasPrefix(l, "-"):
			ls[i] = red.Render(l)
		}
	}
	return strings.Join(ls, "\n")
}

// sideBySide renders a vs b in two columns of total width w, collapsing long unchanged runs.
func sideBySide(a, b string, w int) string {
	ops, ok := diff.Ops(diff.Lines(a), diff.Lines(b))
	if !ok {
		return dim.Render("too large for side-by-side (press s for unified)")
	}
	cw := max(10, (w-3)/2)
	none := lipgloss.NewStyle()
	var out []string
	emit := func(l string, ls lipgloss.Style, r string, rs lipgloss.Style) {
		cell := func(s string, st lipgloss.Style) string {
			return st.Width(cw).Render(ansiTrunc(strings.ReplaceAll(s, "\t", "    "), cw))
		}
		out = append(out, cell(l, ls)+dim.Render(" │ ")+cell(r, rs))
	}
	for i := 0; i < len(ops); {
		if ops[i].Kind == ' ' {
			j := i
			for j < len(ops) && ops[j].Kind == ' ' {
				j++
			}
			run, kh, kt := ops[i:j], 2, 2
			if i == 0 {
				kh = 0
			}
			if j == len(ops) {
				kt = 0
			}
			same := func(os []diff.Op) {
				for _, o := range os {
					emit(o.S, none, o.S, none)
				}
			}
			if len(run) > kh+kt+1 {
				same(run[:kh])
				emit(fmt.Sprintf("⋯ %d unchanged ⋯", len(run)-kh-kt), dim, "", none)
				same(run[len(run)-kt:])
			} else {
				same(run)
			}
			i = j
			continue
		}
		var dels, adds []string
		for ; i < len(ops) && ops[i].Kind != ' '; i++ {
			if ops[i].Kind == '-' {
				dels = append(dels, ops[i].S)
			} else {
				adds = append(adds, ops[i].S)
			}
		}
		for k := 0; k < max(len(dels), len(adds)); k++ {
			var l, r string
			ls, rs := none, none
			if k < len(dels) {
				l, ls = dels[k], red
			}
			if k < len(adds) {
				r, rs = adds[k], green
			}
			emit(l, ls, r, rs)
		}
	}
	if len(out) == 0 {
		return dim.Render("no differences")
	}
	return strings.Join(out, "\n")
}

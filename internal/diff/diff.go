// Package diff has the pure text-diff helpers: line ops, hunks and unified diffs.
package diff

import (
	"os"
	"os/exec"
	"strings"
)

// Op is one line of a diff: Kind is ' ' (same), '-' (only in a) or '+' (only in b).
type Op struct {
	Kind byte
	S    string
}

// Lines splits text into lines without a trailing empty element.
func Lines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// MaxCells bounds len(a)*len(b) accepted by Ops (it needs that much memory).
const MaxCells = 4_000_000

// Ops computes a line diff via LCS. It returns nil, false when the inputs are too large.
func Ops(a, b []string) ([]Op, bool) {
	n, m := len(a), len(b)
	if n*m > MaxCells {
		return nil, false
	}
	dp := make([][]int32, n+1)
	for i := range dp {
		dp[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}
	var ops []Op
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, Op{' ', a[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, Op{'-', a[i]})
			i++
		default:
			ops = append(ops, Op{'+', b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, Op{'-', a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, Op{'+', b[j]})
	}
	return ops, true
}

// Hunk is one @@ block of a plain (uncolored) git diff, plus the file header needed to apply it.
type Hunk struct {
	Staged     bool
	Head, Body string
}

// ParseHunks splits a plain `git diff` of one file into hunks.
func ParseHunks(d string, staged bool) []Hunk {
	var head []string
	var hs []Hunk
	for _, l := range strings.Split(d, "\n") {
		switch {
		case strings.HasPrefix(l, "@@"):
			hs = append(hs, Hunk{Staged: staged, Body: l})
		case len(hs) == 0:
			head = append(head, l)
		default:
			hs[len(hs)-1].Body += "\n" + l
		}
	}
	h := strings.Join(head, "\n")
	for i := range hs {
		hs[i].Head = h
	}
	return hs
}

// Patch is the single-hunk patch suitable for `git apply`.
func (h Hunk) Patch() string { return h.Head + "\n" + h.Body + "\n" }

// Unified is a unified diff of two strings via the system `diff -u` ("" when equal).
func Unified(a, b, la, lb string) string {
	fa, fb := tmp(a), tmp(b)
	defer os.Remove(fa)
	defer os.Remove(fb)
	out, _ := exec.Command("diff", "-u", "--label", la, "--label", lb, fa, fb).Output() // exit 1 = differences
	return strings.TrimRight(string(out), "\n")
}

func tmp(s string) string {
	f, err := os.CreateTemp("", "dotgo-*")
	if err != nil {
		return os.DevNull
	}
	defer f.Close()
	if s != "" {
		s = strings.TrimSuffix(s, "\n") + "\n"
	}
	f.WriteString(s)
	return f.Name()
}

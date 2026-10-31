// Package gitx is a thin wrapper around the git CLI for a (bare) dotfiles repo.
package gitx

import (
	"os/exec"
	"strings"
)

// Client runs git with fixed leading arguments (-C, --git-dir, --work-tree).
type Client struct{ args []string }

// New targets a bare repo with a separate work tree (the usual dotfiles setup).
func New(gitDir, workTree string) *Client {
	return &Client{[]string{"-C", workTree, "--git-dir=" + gitDir, "--work-tree=" + workTree}}
}

// Run executes git and returns combined output without the trailing newline.
func (c *Client) Run(a ...string) (string, error) { return c.RunIn("", a...) }

// RunIn is Run with stdin.
func (c *Client) RunIn(stdin string, a ...string) (string, error) {
	cmd := exec.Command("git", append(append([]string{}, c.args...), a...)...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	return strings.TrimRight(string(out), "\n"), err
}

// Entry is one line of `git status --porcelain`.
type Entry struct {
	X, Y byte
	Path string
}

// ParseStatus parses `status --porcelain -z` output.
func ParseStatus(out string) []Entry {
	var es []Entry
	parts := strings.Split(out, "\x00")
	for i := 0; i < len(parts); i++ {
		p := parts[i]
		if len(p) < 4 {
			continue
		}
		es = append(es, Entry{p[0], p[1], p[3:]})
		if p[0] == 'R' || p[0] == 'C' {
			i++ // skip original path
		}
	}
	return es
}

func (c *Client) Status() []Entry {
	out, _ := c.Run("status", "--porcelain", "-z")
	return ParseStatus(out)
}

// LsFiles lists tracked paths matching the optional pathspecs.
func (c *Client) LsFiles(spec ...string) []string {
	out, _ := c.Run(append([]string{"ls-files", "-z", "--"}, spec...)...)
	var fs []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			fs = append(fs, p)
		}
	}
	return fs
}

// Variants returns the base file plus its ##alternates tracked in the index.
func (c *Client) Variants(path string) []string {
	base, _, _ := strings.Cut(path, "##")
	var vs []string
	for _, l := range c.LsFiles(base, base+"##*") {
		if l == base || strings.HasPrefix(l, base+"##") {
			vs = append(vs, l)
		}
	}
	return vs
}

// VariantBases lists every base path that has at least one ##variant.
func (c *Client) VariantBases() []string {
	seen, bs := map[string]bool{}, []string(nil)
	for _, f := range c.LsFiles() {
		if b, _, ok := strings.Cut(f, "##"); ok && !seen[b] {
			seen[b] = true
			bs = append(bs, b)
		}
	}
	return bs
}

// Show returns a blob's text, or "" if it does not exist.
func (c *Client) Show(rev string) string {
	out, err := c.Run("show", rev)
	if err != nil {
		return ""
	}
	return out
}

// Branch is the current branch name.
func (c *Client) Branch() string {
	b, _ := c.Run("branch", "--show-current")
	return b
}

// HasUpstream reports whether the current branch tracks a remote branch.
func (c *Client) HasUpstream() bool {
	_, err := c.Run("rev-parse", "--abbrev-ref", "@{u}")
	return err == nil
}

// Push pushes, setting the upstream on first push.
func (c *Client) Push() (string, error) {
	if !c.HasUpstream() {
		return c.Run("push", "-u", "origin", "HEAD")
	}
	return c.Run("push")
}

// Pull rebases onto upstream; on failure it aborts the rebase so the repo is never left half-done.
func (c *Client) Pull() (string, error) {
	out, err := c.Run("pull", "--rebase", "--autostash")
	if err != nil {
		c.Run("rebase", "--abort") // no-op when no rebase is in progress
	}
	return out, err
}

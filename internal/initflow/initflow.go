// Package initflow sets up dotgo on a machine: either the first one (new bare repo)
// or an additional one (clone the existing repo, back up conflicting files, apply variants).
package initflow

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/arian24b/dotgo/internal/config"
	"github.com/arian24b/dotgo/internal/gitx"
	"github.com/arian24b/dotgo/internal/variant"
)

// Options for Run. Empty GitDir/WorkTree/Host fall back to the defaults.
type Options struct {
	GitDir, WorkTree, Remote, Host, ConfigPath string
	First                                      bool // brand-new repo (first machine)
	Yes                                        bool // accept defaults, never prompt
	In                                         io.Reader
	Out                                        io.Writer
}

type prompter struct {
	in  *bufio.Reader
	out io.Writer
	yes bool
}

func (p *prompter) ask(label, def string) string {
	if p.yes {
		return def
	}
	if def != "" {
		fmt.Fprintf(p.out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(p.out, "%s: ", label)
	}
	line, _ := p.in.ReadString('\n')
	if line = strings.TrimSpace(line); line != "" {
		return line
	}
	return def
}

func (p *prompter) confirm(label string, def bool) bool {
	if p.yes {
		return def
	}
	d := "y/N"
	if def {
		d = "Y/n"
	}
	switch strings.ToLower(p.ask(label+" ["+d+"]", "")) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	}
	return def
}

// Run performs the init flow and writes the resulting config.
func Run(o Options) (config.Config, error) {
	def := config.Default()
	p := &prompter{bufio.NewReader(o.In), o.Out, o.Yes}
	host, _ := os.Hostname()
	cfg := config.Config{
		GitDir:   config.Expand(p.ask("Bare repo location", firstNonEmpty(o.GitDir, def.GitDir))),
		WorkTree: config.Expand(p.ask("Work tree (where dotfiles live)", firstNonEmpty(o.WorkTree, def.WorkTree))),
		Host:     p.ask("Name of this machine (for ##host.<name> variants)", firstNonEmpty(o.Host, host)),
		Remote:   o.Remote,
	}
	if entries, err := os.ReadDir(cfg.GitDir); err == nil && len(entries) > 0 {
		return cfg, fmt.Errorf("%s already exists and is not empty; pick another --git-dir or remove it", cfg.GitDir)
	}
	first := o.First
	if cfg.Remote == "" && !first {
		if o.Yes {
			return cfg, errors.New("pass --first (new repo) or --remote URL (join existing)")
		}
		first = p.confirm("Is this your FIRST machine (create a new dotfiles repo)?", false)
		if !first {
			if cfg.Remote = p.ask("Git URL of your existing dotfiles repo", ""); cfg.Remote == "" {
				return cfg, errors.New("a remote URL is required to join an existing repo")
			}
		}
	}
	if err := os.MkdirAll(cfg.WorkTree, 0o755); err != nil {
		return cfg, err
	}
	g := gitx.New(cfg.GitDir, cfg.WorkTree)
	var err error
	if first {
		if cfg.Remote == "" {
			cfg.Remote = p.ask("Remote URL to push to (empty = add later)", "")
		}
		err = initFirst(g, cfg, o.Out)
	} else {
		err = joinExisting(g, cfg, p)
	}
	if err != nil {
		return cfg, err
	}

	path := o.ConfigPath
	if path == "" {
		if path, err = config.Path(); err != nil {
			return cfg, err
		}
	}
	if err := cfg.Save(path); err != nil {
		return cfg, err
	}
	fmt.Fprintf(o.Out, "\nconfig saved to %s\nDone. Run `dotgo` to manage your dotfiles", path)
	if first {
		fmt.Fprint(o.Out, " (press a to add files, c to commit, p to push)")
	}
	fmt.Fprintln(o.Out, ".")
	return cfg, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func initFirst(g *gitx.Client, cfg config.Config, out io.Writer) error {
	if b, err := exec.Command("git", "init", "--bare", "-b", "main", cfg.GitDir).CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %s", strings.TrimSpace(string(b)))
	}
	if b, err := g.Run("config", "status.showUntrackedFiles", "no"); err != nil {
		return fmt.Errorf("git config: %s", b)
	}
	if cfg.Remote != "" {
		if b, err := g.Run("remote", "add", "origin", cfg.Remote); err != nil {
			return fmt.Errorf("git remote add: %s", b)
		}
	}
	fmt.Fprintf(out, "created bare repo at %s\n", cfg.GitDir)
	return nil
}

func joinExisting(g *gitx.Client, cfg config.Config, p *prompter) error {
	out := p.out
	if b, err := exec.Command("git", "clone", "--bare", cfg.Remote, cfg.GitDir).CombinedOutput(); err != nil {
		return fmt.Errorf("git clone: %s", strings.TrimSpace(string(b)))
	}
	// bare clones don't track remote branches by default
	for _, a := range [][]string{
		{"config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*"},
		{"config", "status.showUntrackedFiles", "no"},
		{"fetch", "origin"},
	} {
		if b, err := g.Run(a...); err != nil {
			return fmt.Errorf("git %s: %s", a[0], b)
		}
	}
	backup := filepath.Join(cfg.WorkTree, ".dotgo-backup", time.Now().Format("20060102-150405"))
	moved, err := checkout(g, cfg.WorkTree, backup)
	if err != nil {
		return err
	}
	if len(moved) > 0 {
		fmt.Fprintf(out, "backed up %d conflicting file(s) to %s\n", len(moved), backup)
	}
	if br := g.Branch(); br != "" {
		g.Run("branch", "--set-upstream-to=origin/"+br, br) // best effort
	}
	fmt.Fprintf(out, "checked out %s into %s\n", cfg.Remote, cfg.WorkTree)

	// pick this machine's variants
	type pair struct{ base, src string }
	var todo []pair
	for _, base := range g.VariantBases() {
		if src := variant.Pick(g.Variants(base), cfg.Host, runtime.GOOS); src != "" && src != base {
			todo = append(todo, pair{base, src})
		}
	}
	if len(todo) == 0 {
		return nil
	}
	for _, t := range todo {
		fmt.Fprintf(out, "  %s ← %s\n", t.base, t.src)
	}
	if !p.confirm(fmt.Sprintf("Apply these %d variant(s) for %s/%s?", len(todo), cfg.Host, runtime.GOOS), true) {
		return nil
	}
	for _, t := range todo {
		if _, err := variant.Apply(filepath.Join(cfg.WorkTree, t.src), filepath.Join(cfg.WorkTree, t.base), false); err != nil {
			return err
		}
	}
	return nil
}

// checkout checks out HEAD into the work tree, moving files that would be
// overwritten into backup (preserving their relative paths). It returns the moved files.
func checkout(g *gitx.Client, workTree, backup string) ([]string, error) {
	var moved []string
	for range 3 {
		msg, err := g.Run("checkout")
		if err == nil {
			return moved, nil
		}
		files := conflicts(msg)
		if len(files) == 0 {
			return moved, fmt.Errorf("git checkout: %s", msg)
		}
		for _, f := range files {
			dst := filepath.Join(backup, f)
			if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
				return moved, err
			}
			if err := os.Rename(filepath.Join(workTree, f), dst); err != nil {
				return moved, err
			}
			moved = append(moved, f)
		}
	}
	return moved, errors.New("git checkout kept failing after backing up conflicts")
}

// conflicts parses the tab-indented path list of git's "would be overwritten" error.
func conflicts(msg string) []string {
	var fs []string
	for _, l := range strings.Split(msg, "\n") {
		if strings.HasPrefix(l, "\t") {
			fs = append(fs, strings.TrimSpace(l))
		}
	}
	return fs
}

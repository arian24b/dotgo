package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// newRepo makes a bare repo with a separate work tree, like the dotfiles setup.
func newRepo(t *testing.T) (*Client, string) {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	d := t.TempDir()
	gd, wt := filepath.Join(d, "bare"), filepath.Join(d, "wt")
	os.MkdirAll(wt, 0o755)
	if out, err := exec.Command("git", "init", "--bare", gd).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	return New(gd, wt), wt
}

func write(t *testing.T, wt, name, s string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(wt, name), []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStatusAndVariants(t *testing.T) {
	g, wt := newRepo(t)
	write(t, wt, "rc", "a\n")
	write(t, wt, "rc##os.Darwin", "b\n")
	write(t, wt, "rc##os.Linux", "c\n")
	write(t, wt, "other", "x\n")
	g.Run("add", "rc", "rc##os.Darwin", "rc##os.Linux", "other")
	g.Run("commit", "-qm", "x")
	write(t, wt, "rc", "changed\n")

	es := g.Status()
	if len(es) != 1 || es[0].Path != "rc" || es[0].Y != 'M' {
		t.Fatalf("status: %+v", es)
	}
	if vs := g.Variants("rc##os.Linux"); len(vs) != 3 {
		t.Fatalf("variants: %v", vs)
	}
	if bs := g.VariantBases(); len(bs) != 1 || bs[0] != "rc" {
		t.Fatalf("bases: %v", bs)
	}
}

func TestStagedVariantsReadFromIndex(t *testing.T) {
	g, wt := newRepo(t)
	write(t, wt, "rc##os.Darwin", "a\n")
	write(t, wt, "rc##os.Linux", "b\n")
	g.Run("add", ".") // staged, never committed
	if out, err := g.Run("diff", ":rc##os.Darwin", ":rc##os.Linux"); err != nil || out == "" {
		t.Fatalf("diff: %q %v", out, err)
	}
}

func TestPushSetsUpstream(t *testing.T) {
	g, wt := newRepo(t)
	remote := filepath.Join(t.TempDir(), "r.git")
	exec.Command("git", "init", "--bare", "-b", "main", remote).Run()
	g.Run("symbolic-ref", "HEAD", "refs/heads/main")
	g.Run("remote", "add", "origin", remote)
	write(t, wt, "f", "1\n")
	g.Run("add", "f")
	g.Run("commit", "-qm", "x")
	if g.HasUpstream() {
		t.Fatal("no upstream yet")
	}
	if out, err := g.Push(); err != nil {
		t.Fatal(out)
	}
	if !g.HasUpstream() {
		t.Fatal("upstream should be set after first push")
	}
}

func TestDiffSplitsIntoTwoHunks(t *testing.T) {
	g, wt := newRepo(t)
	var s string
	for i := 1; i <= 30; i++ {
		s += "l" + string(rune('a'+i%26)) + string(rune('0'+i%10)) + "\n"
	}
	write(t, wt, "f", s)
	g.Run("add", "f")
	g.Run("commit", "-qm", "x")
	lines := []byte(s)
	lines[1] = 'Z'                   // first line change
	lines[len(lines)-3] = 'Z'        // last line change
	write(t, wt, "f", string(lines)) // two far-apart hunks
	u, _ := g.Run("diff", "--no-color", "--", "f")
	if n := countHunks(u); n != 2 {
		t.Fatalf("want 2 hunks, got %d:\n%s", n, u)
	}
}

func countHunks(d string) (n int) {
	for _, l := range splitLines(d) {
		if len(l) > 1 && l[:2] == "@@" {
			n++
		}
	}
	return
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	return append(out, cur)
}

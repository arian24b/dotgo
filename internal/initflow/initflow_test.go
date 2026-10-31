package initflow

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arian24b/dotgo/internal/config"
	"github.com/arian24b/dotgo/internal/gitx"
)

func gitIn(t *testing.T, dir string, a ...string) {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, a...)...)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %s", a, out)
	}
}

func TestFirstThenJoinSecondMachine(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	root := t.TempDir()

	// shared remote
	remote := filepath.Join(root, "remote.git")
	if out, err := exec.Command("git", "init", "--bare", "-b", "main", remote).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}

	// machine 1: first-time init, add files (incl. a variant for this OS), push
	home1 := filepath.Join(root, "home1")
	var buf bytes.Buffer
	cfg1, err := Run(Options{GitDir: filepath.Join(root, "dot1"), WorkTree: home1, Host: "m1", First: true, Remote: remote,
		Yes: true, ConfigPath: filepath.Join(root, "c1.json"), In: strings.NewReader(""), Out: &buf})
	if err != nil {
		t.Fatalf("first: %v\n%s", err, buf.String())
	}
	os.WriteFile(filepath.Join(home1, ".bashrc"), []byte("shared\n"), 0o644)
	os.WriteFile(filepath.Join(home1, ".bashrc##os."+runtime.GOOS), []byte("os-specific\n"), 0o644)
	g1 := gitx.New(cfg1.GitDir, cfg1.WorkTree)
	if out, err := g1.Run("add", ".bashrc", ".bashrc##os."+runtime.GOOS); err != nil {
		t.Fatal(out)
	}
	if out, err := g1.Run("commit", "-m", "init"); err != nil {
		t.Fatal(out)
	}
	if out, err := g1.Push(); err != nil {
		t.Fatalf("push: %s", out)
	}

	// machine 2: has its own conflicting .bashrc
	home2 := filepath.Join(root, "home2")
	os.MkdirAll(home2, 0o755)
	os.WriteFile(filepath.Join(home2, ".bashrc"), []byte("distro default\n"), 0o644)
	buf.Reset()
	cfg2, err := Run(Options{GitDir: filepath.Join(root, "dot2"), WorkTree: home2, Host: "m2", Remote: remote,
		Yes: true, ConfigPath: filepath.Join(root, "c2.json"), In: strings.NewReader(""), Out: &buf})
	if err != nil {
		t.Fatalf("join: %v\n%s", err, buf.String())
	}
	if b, _ := os.ReadFile(filepath.Join(home2, ".bashrc")); string(b) != "os-specific\n" {
		t.Fatalf(".bashrc should be this OS's variant, got %q\n%s", b, buf.String())
	}
	bak, _ := filepath.Glob(filepath.Join(home2, ".dotgo-backup", "*", ".bashrc"))
	if len(bak) != 1 {
		t.Fatalf("expected conflicting file backed up: %v", bak)
	}
	if b, _ := os.ReadFile(bak[0]); string(b) != "distro default\n" {
		t.Fatalf("backup content %q", b)
	}
	if !gitx.New(cfg2.GitDir, cfg2.WorkTree).HasUpstream() {
		t.Fatal("upstream not set on joined machine")
	}
	if c, found, err := config.Load(filepath.Join(root, "c2.json")); err != nil || !found || c.Host != "m2" {
		t.Fatalf("config: %+v %v %v", c, found, err)
	}
	// refuse to clobber an existing repo
	if _, err := Run(Options{GitDir: cfg2.GitDir, WorkTree: home2, Remote: remote, Yes: true, In: strings.NewReader(""), Out: &buf}); err == nil {
		t.Fatal("expected refusal on existing git dir")
	}
}

func TestConflicts(t *testing.T) {
	msg := "error: The following untracked working tree files would be overwritten by checkout:\n\t.bashrc\n\t.config/x\nPlease move or remove them before you switch branches.\nAborting"
	if got := conflicts(msg); len(got) != 2 || got[1] != ".config/x" {
		t.Fatalf("%v", got)
	}
}

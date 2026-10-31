package variant

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPick(t *testing.T) {
	vs := []string{"rc", "rc##os.Darwin", "rc##os.Linux", "rc##host.Fpc"}
	if got := Pick(vs, "fpc.lan", "linux"); got != "rc##host.Fpc" {
		t.Fatalf("host should win: %s", got)
	}
	if got := Pick(vs, "mac", "darwin"); got != "rc##os.Darwin" {
		t.Fatalf("os: %s", got)
	}
	if Pick([]string{"rc"}, "x", "linux") != "" {
		t.Fatal("no variant expected")
	}
}

func TestApply(t *testing.T) {
	d := t.TempDir()
	src, dst := filepath.Join(d, "rc##os.Linux"), filepath.Join(d, "rc")
	os.WriteFile(src, []byte("new\n"), 0o644)
	os.WriteFile(dst, []byte("old\n"), 0o644)
	if _, err := Apply(src, dst, false); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(dst); string(b) != "new\n" {
		t.Fatalf("copy: %q", b)
	}
	if b, _ := os.ReadFile(dst + ".dotgo-bak"); string(b) != "old\n" {
		t.Fatalf("backup: %q", b)
	}
	if _, err := Apply(src, dst, true); err != nil {
		t.Fatal(err)
	}
	if l, err := os.Readlink(dst); err != nil || l != "rc##os.Linux" {
		t.Fatalf("link: %q %v", l, err)
	}
	// copying over a symlink must not write through into the variant
	if _, err := Apply(src, dst, false); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Lstat(dst); fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("still a symlink")
	}
}

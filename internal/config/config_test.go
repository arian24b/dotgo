package config

import (
	"path/filepath"
	"testing"
)

func TestRoundTripAndDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.json")
	if _, found, err := Load(p); found || err != nil {
		t.Fatalf("missing file: found=%v err=%v", found, err)
	}
	in := Config{GitDir: "~/.dotfiles", WorkTree: "~", Host: "fpc", SSH: []string{"a@b"}}
	if err := in.Save(p); err != nil {
		t.Fatal(err)
	}
	got, found, err := Load(p)
	if err != nil || !found || got.Host != "fpc" || got.SSH[0] != "a@b" {
		t.Fatalf("%+v %v %v", got, found, err)
	}
	if got.GitDir == "~/.dotfiles" {
		t.Fatal("~ not expanded")
	}
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "nvim -u NONE")
	if e := got.EditorCmd(); len(e) != 3 || e[0] != "nvim" {
		t.Fatalf("editor: %v", e)
	}
}

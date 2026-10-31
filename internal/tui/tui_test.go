package tui

import (
	"strings"
	"testing"
)

func TestClipAndShq(t *testing.T) {
	if got := clip("abcdefghij\nxy", 4); got != "abcd\nxy" {
		t.Fatalf("clip: %q", got)
	}
	if got := shq("a'b"); got != `'a'\''b'` {
		t.Fatalf("shq: %s", got)
	}
}

func TestSideBySide(t *testing.T) {
	s := sideBySide("a\nsame\n", "b\nsame\n", 40)
	if !strings.Contains(s, "a") || !strings.Contains(s, "b") || strings.Count(s, "\n") != 1 {
		t.Fatalf("sbs: %q", s)
	}
}

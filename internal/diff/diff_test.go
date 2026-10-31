package diff

import (
	"strings"
	"testing"
)

func TestHunks(t *testing.T) {
	d := "diff --git a/f b/f\n--- a/f\n+++ b/f\n@@ -1 +1 @@\n-a\n+b\n@@ -9 +9 @@\n-x\n+y"
	hs := ParseHunks(d, false)
	if len(hs) != 2 || !strings.HasPrefix(hs[1].Patch(), "diff --git") || !strings.HasSuffix(hs[1].Patch(), "+y\n") {
		t.Fatalf("hunks: %+v", hs)
	}
}

func TestOpsAndUnified(t *testing.T) {
	ops, ok := Ops(Lines("a\nsame\n"), Lines("b\nsame\n"))
	if !ok || len(ops) != 3 || ops[0].Kind != '-' || ops[1].Kind != '+' || ops[2].Kind != ' ' {
		t.Fatalf("ops: %+v", ops)
	}
	if Unified("x\n", "x\n", "a", "b") != "" {
		t.Fatal("equal texts should give empty diff")
	}
	if !strings.Contains(Unified("x\n", "y\n", "a", "b"), "+y") {
		t.Fatal("expected +y")
	}
}

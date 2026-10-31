package secrets

import "testing"

func TestLooks(t *testing.T) {
	for p, want := range map[string]bool{
		".ssh/id_ed25519": true, ".ssh/id_ed25519.pub": false, ".env": true,
		".env.example": false, "a/token.json": true, ".zshrc": false, ".gnupg/x/y": true,
	} {
		if Looks(p) != want {
			t.Errorf("Looks(%q) != %v", p, want)
		}
	}
}

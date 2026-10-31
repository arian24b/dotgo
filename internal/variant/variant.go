// Package variant resolves and applies per-host / per-OS file variants.
// A variant is a sibling file named <base>##host.<name> or <base>##os.<goos>.
package variant

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// Pick chooses the variant for this machine from vs (base + alternates):
// host.<name> beats os.<goos>. It returns "" when none matches.
func Pick(vs []string, host, goos string) string {
	h := strings.ToLower(host)
	short, _, _ := strings.Cut(h, ".")
	osv := ""
	for _, v := range vs {
		_, tag, ok := strings.Cut(v, "##")
		if !ok {
			continue
		}
		t := strings.ToLower(tag)
		if t == "host."+h || t == "host."+short {
			return v
		}
		if t == "os."+goos {
			osv = v
		}
	}
	return osv
}

// Base strips the ##suffix from a path.
func Base(p string) string {
	b, _, _ := strings.Cut(p, "##")
	return b
}

// Apply copies (or symlinks) src over dst. A differing dst is first saved as dst.dotgo-bak.
func Apply(src, dst string, link bool) (string, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	if fi, err := os.Lstat(dst); err == nil {
		if old, err := os.ReadFile(dst); err == nil && (link || !bytes.Equal(old, data)) {
			if err := os.WriteFile(dst+".dotgo-bak", old, 0o600); err != nil {
				return "", err
			}
		}
		if fi.Mode()&os.ModeSymlink != 0 || link {
			if err := os.Remove(dst); err != nil { // never write through a symlink into a variant
				return "", err
			}
		}
	}
	if link {
		if err := os.Symlink(filepath.Base(src), dst); err != nil {
			return "", err
		}
		return "linked " + filepath.Base(src), nil
	}
	mode := os.FileMode(0o644)
	if s, err := os.Stat(src); err == nil {
		mode = s.Mode().Perm()
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		return "", err
	}
	return "copied " + filepath.Base(src), nil
}

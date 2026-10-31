// Package config loads and saves the per-machine dotgo configuration.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is stored per machine (it is not part of the dotfiles repo).
type Config struct {
	GitDir   string   `json:"git_dir"`                 // bare repository
	WorkTree string   `json:"work_tree"`               // usually $HOME
	Host     string   `json:"host,omitempty"`          // stable name used by ##host.<name> variants
	Remote   string   `json:"remote,omitempty"`        // origin URL
	SSH      []string `json:"ssh_hosts,omitempty"`     // other machines for the Remote tab
	Link     bool     `json:"link_variants,omitempty"` // symlink instead of copy when applying variants
	Editor   string   `json:"editor,omitempty"`        // default: $VISUAL, $EDITOR, vi
}

// Default is used when no config file exists.
func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{GitDir: filepath.Join(home, ".dotfiles"), WorkTree: home}
}

// Path returns the config file location ($DOTGO_CONFIG or <user config dir>/dotgo/config.json).
func Path() (string, error) {
	if p := os.Getenv("DOTGO_CONFIG"); p != "" {
		return p, nil
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "dotgo", "config.json"), nil
}

// Load reads path over the defaults. found is false if the file does not exist.
func Load(path string) (c Config, found bool, err error) {
	c = Default()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, false, nil
	}
	if err != nil {
		return c, false, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, false, fmt.Errorf("parse %s: %w", path, err)
	}
	c.GitDir, c.WorkTree = Expand(c.GitDir), Expand(c.WorkTree)
	return c, true, nil
}

// Save writes the config (0600) creating its directory.
func (c Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

// Expand resolves a leading ~.
func Expand(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
}

// Hostname is the configured host name, or the OS hostname.
func (c Config) Hostname() string {
	if c.Host != "" {
		return c.Host
	}
	h, _ := os.Hostname()
	return h
}

// EditorCmd returns the editor command and its arguments.
func (c Config) EditorCmd() []string {
	for _, e := range []string{c.Editor, os.Getenv("VISUAL"), os.Getenv("EDITOR")} {
		if f := strings.Fields(e); len(f) > 0 {
			return f
		}
	}
	return []string{"vi"}
}

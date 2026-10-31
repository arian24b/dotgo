// Command dotgo manages dotfiles across machines in a git repo, with a terminal UI.
//
//	dotgo               open the TUI
//	dotgo init          set up this machine (first machine, or join an existing repo)
//	dotgo config        print the config path and effective settings
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/arian24b/dotgo/internal/config"
	"github.com/arian24b/dotgo/internal/gitx"
	"github.com/arian24b/dotgo/internal/initflow"
	"github.com/arian24b/dotgo/internal/tui"
)

// version is set at release time: -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	args := os.Args[1:]
	var err error
	switch {
	case len(args) > 0 && args[0] == "init":
		err = runInit(args[1:])
	case len(args) > 0 && args[0] == "config":
		err = runConfig(args[1:])
	case len(args) > 0 && (args[0] == "version" || args[0] == "--version"):
		fmt.Println("dotgo", version)
	case len(args) > 0 && (args[0] == "help" || args[0] == "-h" || args[0] == "--help"):
		fmt.Print(usage)
	default:
		err = runTUI(args)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "dotgo:", err)
		os.Exit(1)
	}
}

const usage = `dotgo — manage dotfiles across machines

  dotgo                 open the TUI (flags: --config --git-dir --work-tree --ssh --link)
  dotgo init            set up this machine
      --first             create a brand-new dotfiles repo (first machine)
      --remote URL        join the existing repo at URL (additional machine)
      --git-dir --work-tree --host --yes
  dotgo config          show config path and effective settings
  dotgo version         print the version
`

func configPath(flagVal string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	return config.Path()
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	var o initflow.Options
	var cfgPath string
	fs.BoolVar(&o.First, "first", false, "first machine: create a new bare repo")
	fs.StringVar(&o.Remote, "remote", "", "git URL of the dotfiles repo (join, or origin for --first)")
	fs.StringVar(&o.GitDir, "git-dir", "", "bare repo location (default ~/.dotfiles)")
	fs.StringVar(&o.WorkTree, "work-tree", "", "work tree (default ~)")
	fs.StringVar(&o.Host, "host", "", "name of this machine (default: hostname)")
	fs.BoolVar(&o.Yes, "yes", false, "accept defaults, don't prompt")
	fs.StringVar(&cfgPath, "config", "", "config file path")
	fs.Parse(args)
	var err error
	if o.ConfigPath, err = configPath(cfgPath); err != nil {
		return err
	}
	o.In, o.Out = os.Stdin, os.Stdout
	_, err = initflow.Run(o)
	return err
}

func load(cfgPath string) (config.Config, string, bool, error) {
	path, err := configPath(cfgPath)
	if err != nil {
		return config.Config{}, "", false, err
	}
	cfg, found, err := config.Load(path)
	return cfg, path, found, err
}

func runConfig(args []string) error {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	cfgPath := fs.String("config", "", "config file path")
	fs.Parse(args)
	cfg, path, found, err := load(*cfgPath)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Printf("%s (exists: %v)\n%s\n", path, found, b)
	return nil
}

func runTUI(args []string) error {
	fs := flag.NewFlagSet("dotgo", flag.ExitOnError)
	cfgPath := fs.String("config", "", "config file path")
	fs.Parse(args) // first pass only to find --config
	cfg, _, found, err := load(*cfgPath)
	if err != nil {
		return err
	}
	gd := fs.String("git-dir", cfg.GitDir, "bare repo")
	wt := fs.String("work-tree", cfg.WorkTree, "work tree")
	ssh := fs.String("ssh", strings.Join(cfg.SSH, ","), "comma-separated ssh hosts for the Remote tab")
	link := fs.Bool("link", cfg.Link, "apply variants as symlinks instead of copies")
	fs.Parse(args)

	gitDir, workTree := config.Expand(*gd), config.Expand(*wt)
	if _, err := os.Stat(gitDir); err != nil {
		hint := "run `dotgo init`"
		if found {
			hint = "check git_dir in the config"
		}
		return fmt.Errorf("no dotfiles repo at %s — %s", gitDir, hint)
	}
	var remotes []string
	for _, h := range strings.Split(*ssh, ",") {
		if h = strings.TrimSpace(h); h != "" {
			remotes = append(remotes, h)
		}
	}
	return tui.Run(tui.Options{
		Git: gitx.New(gitDir, workTree), WorkTree: workTree, Host: cfg.Hostname(),
		Editor: cfg.EditorCmd(), Link: *link, Remotes: remotes,
	})
}

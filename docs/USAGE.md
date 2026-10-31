# dotgo usage guide

[English](USAGE.md) · [فارسی](USAGE.fa.md)

## 1. Concepts
- **Repo** — a *bare* git repo (default `~/.dotfiles`) whose work tree is your home directory. Untracked files are hidden, so only what you `add` is managed.
- **Shared file** — a normal tracked file, e.g. `.gitconfig`. Same content on every machine.
- **Variant** — a sibling file with a `##` suffix. dotgo picks one per machine:
  | Name | Used when |
  |---|---|
  | `.zshrc##host.<name>` | this machine's name is `<name>` (wins) |
  | `.zshrc##os.Darwin` / `.zshrc##os.Linux` | the OS matches |
  | `.zshrc` | base file; fallback and the apply target |

  `A` copies the matching variant over the base file (or symlinks with `--link` / `link_variants`). The old file is saved as `.zshrc.dotgo-bak`.
- **Machine name** — `host` in the config, otherwise the OS hostname.

## 2. Set up
**First machine**
```sh
dotgo init --first --remote git@github.com:YOU/dotfiles.git   # remote is optional
dotgo                       # a → type a path (e.g. .zshrc) to start tracking; c → commit; p → push
```
**Another machine**
```sh
dotgo init --remote git@github.com:YOU/dotfiles.git --host ubuntu
```
This clones the repo, moves files that would be overwritten into `~/.dotgo-backup/<time>/`, checks out, sets the upstream, and offers to apply this machine's variants.
Without flags `dotgo init` asks questions; `--yes` accepts defaults.

**Already have `~/.dotfiles`?** Run `dotgo` — no init needed.

## 3. Daily workflow
1. `dotgo` → **Changes** tab shows modified files grouped by folder (green dot = staged, yellow = not).
2. `j`/`k` select, `e` edit, `s` toggle side-by-side.
3. `space` stages a file; `]`/`[` pick a hunk and `S` stages just that hunk.
4. `c` commit, `p` push. On the other machine: `P` pull, then `A` to re-apply variants.

### Tabs
| Tab | Shows |
|---|---|
| 1 Changes | working-tree changes, hunks, staging |
| 2 Upstream | files that differ from the remote branch |
| 3 Variants | files with `##` variants; diff between pairs (`n` = next pair); the active one is shown |
| 4 All | every tracked file with a preview |
| 5 Log | history (`l` = this file, `L` = all); `m` marks a commit, then select another to diff the two |
| 6 Remote | `o` fetches the file from `ssh_hosts` and diffs it with the local copy |

### Keys
Press `?` in the app for the full list. Mouse: wheel scrolls, click selects rows and tabs.

## 4. Configuration
`<user config dir>/dotgo/config.json` — Linux: `~/.config/dotgo/`, macOS: `~/Library/Application Support/dotgo/` — or the path in `$DOTGO_CONFIG`. Per machine; not stored in the repo.
```json
{
  "git_dir": "/home/me/.dotfiles",
  "work_tree": "/home/me",
  "host": "ubuntu",
  "remote": "git@github.com:YOU/dotfiles.git",
  "ssh_hosts": ["me@mac.local"],
  "link_variants": false,
  "editor": "nvim"
}
```
Flags override the config: `--config --git-dir --work-tree --ssh --link`. `dotgo config` prints the effective values.

## 5. Safety
- Discard, apply-variant and "stage a secret" always show a preview and wait for `y`.
- Names such as `id_*`, `.env`, `*.pem`, `*token*`, `.gnupg/*` trigger a warning before staging. Keep private keys out of the repo.
- A failed pull aborts its rebase and reports the error; nothing is left half-done.
- The Remote tab uses key-based ssh only (no password prompts) and only reads files.

## 6. Troubleshooting
| Problem | Fix |
|---|---|
| `no dotfiles repo at …` | run `dotgo init`, or check `git_dir` in the config |
| New file doesn't show in Changes | untracked files are hidden by design; press `a` and type its path |
| Upstream tab: "no upstream" | push once (`p`) — the first push sets the upstream |
| Variant not applied | check the name (`##os.Darwin`, `##host.<name>`) and that it is committed; the Variants tab shows the active one |
| Remote tab: "unavailable" | `ssh user@host` must work without a password and use the same `$HOME` layout |

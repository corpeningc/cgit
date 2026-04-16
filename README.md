# cgit

A fun CLI tool to make git operations interactive and enjoyable.

Built in Go with Cobra CLI and Charm TUI libraries for learning purposes.

## Features

### Interactive TUIs
- **Log viewer** — browse commit history with `cgit log`; press `enter` to view a diff, `p` to cherry-pick
- **Status viewer** — tabbed staged/unstaged file list with `cgit status` (or `cgit st`); press `m` to launch file manager
- **Branch manager** — navigate, switch, delete, and rename branches with `cgit branches` (or `cgit br`)
- **Stash picker** — browse stashes with a split-pane diff preview using `cgit pop`; `enter` pops, `a` applies, `d` drops
- **Conflict resolver** — step through merge conflicts interactively with `cgit conflicts` (or `cgit cf`)
- **File manager** — stage and restore files with fuzzy search using `cgit manage` (or `cgit m`)

### Commits
- Commit staged changes: `cgit commit <message>`
- Amend the last commit: `cgit amend`
- Commit and push in one step: `cgit commit-and-push <message>` (or `cgit cap`)
- Undo the last commit (keeps changes staged): `cgit undo`

### Branches
- Create and switch to a new branch: `cgit new-branch <name>` (or `cgit nb`)
- Switch branches interactively: `cgit switch` (or `cgit sw`); use `-r` to include remotes
- Feature branch workflow: `cgit feature` (or `cgit feat`)
  - Create: `cgit feat -n <name> -o <origin>`
  - Close: `cgit feat -c -o <origin>`

### Rebase
- Interactively rebase the last N commits: `cgit rebase` (or `cgit rebase -n 20`)
- Set the default limit in config

### Remote Operations
- Push: `cgit push`
- Pull: `cgit pull [branch]`
- Merge remote changes: `cgit merge <branch>`

### Stash
- Stash changes: `cgit store [name]`
- Pop/apply/drop stashes interactively: `cgit pop`

### Utilities
- Hard reset and clean working directory: `cgit full-clean` (or `cgit fc`)
- Show/edit config: `cgit config`
- Shell completions: `cgit completion --help`

### Persistent Status Bar
All TUI views show a top-line status bar with the current branch, ahead/behind counts, and a clean/dirty indicator.

### Config
cgit reads `~/.config/cgit/config.json` (or `$CGIT_CONFIG`). Defaults:

```json
{
  "log_limit": 100,
  "rebase_limit": 15,
  "split_pane": true,
  "editor": ""
}
```

Run `cgit config` to see the active config path and values.

## Installation

### Prerequisites
- Go 1.21 or later

```bash
go install "github.com/corpeningc/cgit@latest"
```

## Shell Completions

```bash
# bash
cgit completion bash > /etc/bash_completion.d/cgit

# zsh
cgit completion zsh > "${fpath[1]}/_cgit"

# fish
cgit completion fish > ~/.config/fish/completions/cgit.fish
```

## Usage

```bash
cgit          # see all available commands
cgit --help   # usage details
```

# cgit

A fun CLI tool to make git operations interactive and enjoyable.

Built in Go with Cobra CLI and Charm TUI libraries for learning purposes.

## Features

### Interactive Shell
- Launch interactive mode with `cgit` or `cgit shell`

### File Management
- Interactive file management with `cgit manage` (or `cgit m`)
  - Supports staging and restoring files with fuzzy search
  - Use `-s` flag to manage staged files

### Branch Operations
- Create and switch to new branches with `cgit new-branch <name>` (or `cgit nb`)
- Interactive branch selection with `cgit switch` and no arguments
- Switch between existing branches via `cgit switch [branch]` (or `cgit sw`)
  - Use `-r` flag to include remote branches
- Feature branch workflow with `cgit feature` (or `cgit feat`)
  - Create feature branches: `cgit feat -n <name> -o <origin>`
  - Close feature branches: `cgit feat -c -o <origin>`

### Commits and Pushes
- Commit changes with `cgit commit <message>`
- Push to remote with `cgit push`
- Commit and push in one step with `cgit commit-and-push <message>` (or `cgit cap`)

### Stash Operations
- Interactive stash selection with `cgit store` (WIP)
- Stash changes with a message via `cgit store [name]`
- Pop most recent stash with `cgit pop`

### Repository Operations
- Pull latest changes with `cgit pull [branch]`
- Merge remote changes with `cgit merge <branch>`
- View repository status with `cgit status` (or `cgit st`)
- Clean working directory with `cgit full-clean` (or `cgit fc`)


## Installation

### Prerequisites
- Go 1.25 or later
  
``` bash
go install "github.com/corpeningc/cgit@latest"
```


## Usage

```bash
cgit          # See all available commands
```

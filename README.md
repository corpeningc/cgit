# cgit

A fun CLI tool to make git operations interactive and enjoyable.

Built in Go with Cobra CLI and Charm TUI libraries for learning purposes.

## Features

- Interactive file staging with `cgit add`
- Terminal UI git status with `cgit status` (or `cgit st`)
- Quick commit and push with `cgit commit-and-push` (or `cgit cap`)
- Easy branch creation with `cgit new-branch` (or `cgit nb`)
- Simple branch switching with `cgit switch` (or `cgit sw`)


## Installation
``` bash
go install "github.com/corpeningc/cgit@latest"
```


## Usage

```bash
cgit status      # Interactive git status
cgit add         # Interactive file picker for staging
cgit cap "msg"   # Commit and push in one command
cgit nb feature  # Create and switch to new branch
```

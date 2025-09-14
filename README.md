# cgit

A fun CLI tool to make git operations interactive and enjoyable.

Built in Go with Cobra CLI and Charm TUI libraries for learning purposes.

## Features

- Interactive file staging with `cgit add`
- Quick commit and push with `cgit commit-and-push` (or `cgit cap`)
- Easy branch creation with `cgit new-branch` (or `cgit nb`)
- Simple branch switching with `cgit switch` (or `cgit sw`)


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
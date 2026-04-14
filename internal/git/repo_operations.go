package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type RepoStatus struct {
	CurrentBranch string
	StagedFiles   []FileStatus
	UnstagedFiles []FileStatus
}

type GitRepo struct {
	WorkDir string
}

func formatCommandError(operation string, err error, stdout, stderr bytes.Buffer) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s failed: %v\nStdout: %s\nStderr: %s",
		operation, err, stdout.String(), stderr.String())
}

func New(workDir string) *GitRepo {
	return &GitRepo{WorkDir: workDir}
}

func (repo *GitRepo) Fetch() error {
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return formatCommandError("fetch", err, stdout, stderr)
}

func (repo *GitRepo) PullLatestRemote(branch string) error {
	cmd := exec.Command("git", "pull", "origin", branch)
	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return formatCommandError("pull", err, stdout, stderr)
}

func (repo *GitRepo) Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	os.Environ()
	cmd.Dir = repo.WorkDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return formatCommandError("commit", err, stdout, stderr)
}

type PushOptions struct {
	ForceWithLease bool
	SetUpstream    bool
}

func (repo *GitRepo) Push() error {
	return repo.PushWithOptions(PushOptions{})
}

func (repo *GitRepo) PushWithOptions(opts PushOptions) error {
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		return err
	}

	statusCmd := exec.Command("git", "status")
	statusCmd.Env = os.Environ()
	statusCmd.Dir = repo.WorkDir

	err = statusCmd.Run()
	if err != nil {
		return err
	}

	args := []string{"push", "origin", currentBranch}
	if opts.ForceWithLease {
		args = append(args, "--force-with-lease")
	}
	if opts.SetUpstream {
		args = append(args, "--set-upstream")
	}

	pushCmd := exec.Command("git", args...)
	pushCmd.Env = os.Environ()
	pushCmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	pushCmd.Stdout = &stdout
	pushCmd.Stderr = &stderr

	err = pushCmd.Run()
	return formatCommandError("push", err, stdout, stderr)
}

func (repo *GitRepo) IsClean() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repo.WorkDir

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return len(output) == 0, nil
}

func (repo *GitRepo) GetRepositoryStatus() (*RepoStatus, error) {
	status := &RepoStatus{}

	// Get current branch
	branch, err := repo.GetCurrentBranch()
	if err != nil {
		return nil, err
	}
	status.CurrentBranch = branch

	// Get file status
	stagedFiles, unstagedFiles, err := repo.GetFileStatuses()
	if err != nil {
		return nil, err
	}

	status.StagedFiles = stagedFiles
	status.UnstagedFiles = unstagedFiles

	return status, nil
}

func (repo *GitRepo) Stash(message string) error {
	var cmd *exec.Cmd

	if message != "" {
		cmd = exec.Command("git", "stash", "push", "-m", message)
	} else {
		cmd = exec.Command("git", "stash")
	}

	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return formatCommandError("stash changes", err, stdout, stderr)
}

type StashEntry struct {
	Ref         string
	Description string
}

func (repo *GitRepo) StashList() ([]StashEntry, error) {
	cmd := exec.Command("git", "stash", "list", "--format=%gd|%s")
	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, formatCommandError("list stashes", err, stdout, stderr)
	}

	var entries []StashEntry
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		entries = append(entries, StashEntry{Ref: parts[0], Description: parts[1]})
	}
	return entries, nil
}

func (repo *GitRepo) StashPopRef(ref string) error {
	cmd := exec.Command("git", "stash", "pop", ref)
	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return formatCommandError("pop stash", err, stdout, stderr)
}

func (repo *GitRepo) StashPop() error {
	cmd := exec.Command("git", "stash", "pop")
	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return formatCommandError("pop stash", err, stdout, stderr)
}

func (repo *GitRepo) GetLog(limit int) (string, error) {
	args := []string{"log", "--oneline", "--graph", "--decorate", fmt.Sprintf("-n%d", limit)}
	cmd := exec.Command("git", args...)
	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", formatCommandError("get log", err, stdout, stderr)
	}
	return stdout.String(), nil
}

func (repo *GitRepo) FullClean() error {
	cmd := exec.Command("git", "reset", "--hard")
	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return formatCommandError("reset --hard", err, stdout, stderr)
	}

	cleanCmd := exec.Command("git", "clean", "-fd")
	cleanCmd.Dir = repo.WorkDir

	var cleanStdout, cleanStderr bytes.Buffer
	cleanCmd.Stdout = &cleanStdout
	cleanCmd.Stderr = &cleanStderr

	err = cleanCmd.Run()
	return formatCommandError("clean -fd", err, cleanStdout, cleanStderr)
}

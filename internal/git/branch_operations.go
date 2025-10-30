package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (repo *GitRepo) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Env = os.Environ()
	cmd.Dir = repo.WorkDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (repo *GitRepo) MergeLatest(branch string) error {
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		return err
	}

	// Probably dont want to merge into main or master directly so just pull
	if currentBranch == "main" || currentBranch == "master" {
		cmd := exec.Command("git", "pull")
		cmd.Dir = repo.WorkDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		return formatCommandError("pull", err, stdout, stderr)
	}

	// Get latest from remote
	err = repo.PullLatestRemote(branch)

	if err != nil {
		return err
	}

	cmd := exec.Command("git", "merge", "origin/"+branch)
	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	return formatCommandError("merge", err, stdout, stderr)
}

func (repo *GitRepo) MergeLocalBranch(branchName string) error {
	cmd := exec.Command("git", "merge", branchName)
	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return formatCommandError("merge local branch", err, stdout, stderr)
}

func (repo *GitRepo) CreateBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return formatCommandError("create branch", err, stdout, stderr)
}

func (repo *GitRepo) SwitchBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return formatCommandError("switch branch", err, stdout, stderr)
}

func (repo *GitRepo) GetAllBranches(remote bool) ([]string, error) {
	getBranchCmd := exec.Command("git", "branch", "-a")
	getBranchCmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	getBranchCmd.Stdout = &stdout
	getBranchCmd.Stderr = &stderr

	err := getBranchCmd.Run()
	if err != nil {
		return nil, formatCommandError("get branches", err, stdout, stderr)
	}

	var branches []string
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "* ") {
			line = strings.TrimSpace(line[2:])
		}

		if strings.Contains(line, "remotes/origin/HEAD") {
			continue
		}

		if strings.HasPrefix(line, "remotes/") {
			if remote {
				branch := strings.TrimPrefix(line, "remotes/origin/")
				branches = append(branches, branch)
			} else {
				continue
			}
		}

		if line != "" {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

func (repo *GitRepo) DeleteBranch(branchName string) error {
	cmd := exec.Command("git", "branch", "-d", branchName)
	cmd.Dir = repo.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return formatCommandError("delete branch", err, stdout, stderr)
}


package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func formatCommandError(operation string, err error, stdout, stderr bytes.Buffer) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s failed: %v\nStdout: %s\nStderr: %s", 
		operation, err, stdout.String(), stderr.String())
}

type GitRepo struct {
	WorkDir string
}

func New(workDir string) *GitRepo {
	return &GitRepo{WorkDir: workDir}
}

func (repo *GitRepo) GetModifiedFiles() ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repo.WorkDir

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[2:]))
		}
	}

	return files, nil
}

func (repo *GitRepo) AddFiles(files []string) error {
	if len(files) == 0 {
		return nil
	}

	args := append([]string{"add"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = repo.WorkDir
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return formatCommandError("add files", err, stdout, stderr)
}

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

	// Merge remote into current
	cmd := exec.Command("git", "merge", "origin/"+branch)
	cmd.Dir = repo.WorkDir
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err = cmd.Run()
	return formatCommandError("merge", err, stdout, stderr)
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

func (repo *GitRepo) Push() error {
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
			return err
	}
	
	statusCmd := exec.Command("git", "status")
	statusCmd.Env = os.Environ() 
	statusCmd.Dir = repo.WorkDir
	statusCmd.Run() 
	
	pushCmd := exec.Command("git", "push", "origin", currentBranch)
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
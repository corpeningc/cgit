package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

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

	return cmd.Run()
}

func (repo *GitRepo) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch")
	os.Environ()
	cmd.Dir = repo.WorkDir

	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error executing git branch:", err)
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func (repo *GitRepo) Fetch() error {
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = repo.WorkDir
	return cmd.Run()
}

func (repo *GitRepo) PullLatestRemote(branch string) error {
	cmd := exec.Command("git", "pull", "origin", branch)
	cmd.Dir = repo.WorkDir
	return cmd.Run()
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
		return cmd.Run()
	}

	// Get latest from remote
	err = repo.PullLatestRemote(branch)

	if err != nil {
		return err
	}

	// Merge remote into current
	cmd := exec.Command("git", "merge", "origin/"+branch)
	cmd.Dir = repo.WorkDir
	return cmd.Run()
}

func (repo *GitRepo) Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	os.Environ()
	cmd.Dir = repo.WorkDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
			fmt.Printf("Commit failed: %v\n", err)
			fmt.Printf("Stdout: %s\n", stdout.String())
			fmt.Printf("Stderr: %s\n", stderr.String())
	}

	return err
}

func (repo *GitRepo) Push() error {
    currentBranch, err := repo.GetCurrentBranch()
    if err != nil {
        return err
    }
    
    // Warmup with git status
    statusCmd := exec.Command("git", "status")
    statusCmd.Env = os.Environ() // Actually assign the environment
    statusCmd.Dir = repo.WorkDir
    statusCmd.Run() // Just run it, ignore output for warmup
    
    // Now push
    pushCmd := exec.Command("git", "push", "origin", currentBranch)
    pushCmd.Env = os.Environ() // Assign environment
    pushCmd.Dir = repo.WorkDir
    
    var stdout, stderr bytes.Buffer
    pushCmd.Stdout = &stdout
    pushCmd.Stderr = &stderr
    
    err = pushCmd.Run() // Only run once
    if err != nil {
        return fmt.Errorf("push failed: %v\nStdout: %s\nStderr: %s", 
            err, stdout.String(), stderr.String())
    }
    
    return nil // Don't call cmd.Run() again
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
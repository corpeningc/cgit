package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type FileStatus struct {
	Path      string
	Status    string // M(odified), A(dded), D(eleted), R(enamed), ?(untracked)
	Staged    bool
	WorkTree  bool
}

type RepoStatus struct {
	CurrentBranch string
	Ahead         int
	Behind        int
	StagedFiles   []FileStatus
	UnstagedFiles []FileStatus
	LastCommit    CommitInfo
}

type CommitInfo struct {
	Hash    string
	Message string
	Author  string
	Date    string
}

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

func (repo *GitRepo) GetRepositoryStatus() (*RepoStatus, error) {
	status := &RepoStatus{}
	
	// Get current branch
	branch, err := repo.GetCurrentBranch()
	if err != nil {
		return nil, err
	}
	status.CurrentBranch = branch
	
	// Get ahead/behind counts
	ahead, behind, err := repo.GetBranchTracking()
	if err == nil {
		status.Ahead = ahead
		status.Behind = behind
	}
	
	// Get file status
	stagedFiles, unstagedFiles, err := repo.GetFileStatuses()
	if err != nil {
		return nil, err
	}
	status.StagedFiles = stagedFiles
	status.UnstagedFiles = unstagedFiles
	
	// Get last commit
	commitInfo, err := repo.GetLastCommit()
	if err == nil {
		status.LastCommit = commitInfo
	}
	
	return status, nil
}

func (repo *GitRepo) GetBranchTracking() (int, int, error) {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	cmd.Dir = repo.WorkDir
	
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, nil // No upstream or other issue, return 0s
	}
	
	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return 0, 0, nil
	}
	
	var ahead, behind int
	fmt.Sscanf(parts[0], "%d", &ahead)
	fmt.Sscanf(parts[1], "%d", &behind)
	
	return ahead, behind, nil
}

func (repo *GitRepo) GetLastCommit() (CommitInfo, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%H|%s|%an|%ar")
	cmd.Dir = repo.WorkDir
	
	output, err := cmd.Output()
	if err != nil {
		return CommitInfo{}, err
	}
	
	parts := strings.Split(strings.TrimSpace(string(output)), "|")
	if len(parts) != 4 {
		return CommitInfo{}, fmt.Errorf("unexpected git log format")
	}
	
	return CommitInfo{
		Hash:    parts[0][:8], // Short hash
		Message: parts[1],
		Author:  parts[2],
		Date:    parts[3],
	}, nil
}

func (repo *GitRepo) GetFileStatuses() ([]FileStatus, []FileStatus, error) {
	cmd := exec.Command("git", "status", "--porcelain=v1")
	cmd.Dir = repo.WorkDir
	
	output, err := cmd.Output()
	if err != nil {
		return nil, nil, err
	}
	
	var stagedFiles, unstagedFiles []FileStatus
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 3 {
			continue
		}
		
		stageStatus := string(line[0])
		workTreeStatus := string(line[1])
		filePath := strings.TrimSpace(line[3:])
		
		// Staged files
		if stageStatus != " " && stageStatus != "?" {
			stagedFiles = append(stagedFiles, FileStatus{
				Path:     filePath,
				Status:   stageStatus,
				Staged:   true,
				WorkTree: false,
			})
		}
		
		// Unstaged files
		if workTreeStatus != " " {
			unstagedFiles = append(unstagedFiles, FileStatus{
				Path:     filePath,
				Status:   workTreeStatus,
				Staged:   false,
				WorkTree: true,
			})
		}
	}
	
	return stagedFiles, unstagedFiles, nil
}

func (repo *GitRepo) StageFile(filePath string) error {
	cmd := exec.Command("git", "add", filePath)
	cmd.Dir = repo.WorkDir
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	return formatCommandError("stage file", err, stdout, stderr)
}

func (repo *GitRepo) UnstageFile(filePath string) error {
	cmd := exec.Command("git", "restore", "--staged", filePath)
	cmd.Dir = repo.WorkDir
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	return formatCommandError("unstage file", err, stdout, stderr)
}

func (repo *GitRepo) DiscardChanges(filePath string) error {
	cmd := exec.Command("git", "restore", filePath)
	cmd.Dir = repo.WorkDir
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	return formatCommandError("discard changes", err, stdout, stderr)
}

func (repo *GitRepo) GetFileDiff(filePath string, staged bool) (string, error) {
	var cmd *exec.Cmd
	if staged {
		cmd = exec.Command("git", "diff", "--staged", filePath)
	} else {
		cmd = exec.Command("git", "diff", filePath)
	}
	cmd.Dir = repo.WorkDir
	
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	return string(output), nil
}

func (repo *GitRepo) StageAllFiles() error {
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = repo.WorkDir
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	return formatCommandError("stage all files", err, stdout, stderr)
}
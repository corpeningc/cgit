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
	StagedFiles   []FileStatus
	UnstagedFiles []FileStatus
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
	
	
	// Get file status
	stagedFiles, unstagedFiles, err := repo.GetFileStatuses()
	if err != nil {
		return nil, err
	}
	status.StagedFiles = stagedFiles
	status.UnstagedFiles = unstagedFiles
	
	
	return status, nil
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
		
		// Git quotes filenames with special characters - remove the quotes
		if strings.HasPrefix(filePath, "\"") && strings.HasSuffix(filePath, "\"") {
			filePath = filePath[1 : len(filePath)-1]
		}
		
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


func (repo *GitRepo) Stash(message string) error {
	cmd := exec.Command("git", "stash", "push", "-m", message)
	cmd.Dir = repo.WorkDir
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	return formatCommandError("stash changes", err, stdout, stderr)
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

package git

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
)

type FileStatus struct {
	Path      string
	Status    string // M(odified), A(dded), D(eleted), R(enamed), ?(untracked)
	Staged    bool
	WorkTree  bool
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
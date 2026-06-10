package cli

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type gitFileChange struct {
	Path       string
	ChangeType string
}

func gitChangedFiles(root string, mode string, sinceRef string) ([]gitFileChange, error) {
	args := []string{"diff", "--name-status"}
	if mode == "staged" {
		args = []string{"diff", "--cached", "--name-status"}
	} else if mode == "since" {
		args = []string{"diff", "--name-status", sinceRef + "..HEAD"}
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git diff failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}
	changes := parseGitNameStatus(string(output))
	if mode == "changed" {
		staged, err := gitStagedFiles(root)
		if err != nil {
			return nil, err
		}
		untracked, err := gitUntrackedFiles(root)
		if err != nil {
			return nil, err
		}
		changes = dedupeGitFileChanges(append(append(changes, staged...), untracked...))
	}
	return changes, nil
}

func gitStagedFiles(root string) ([]gitFileChange, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-status")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git diff --cached failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}
	return parseGitNameStatus(string(output)), nil
}

func gitUntrackedFiles(root string) ([]gitFileChange, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git ls-files failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}
	var changes []gitFileChange
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		changes = append(changes, gitFileChange{Path: filepath.ToSlash(line), ChangeType: "added"})
	}
	return changes, nil
}

func dedupeGitFileChanges(changes []gitFileChange) []gitFileChange {
	seen := map[string]bool{}
	var deduped []gitFileChange
	for _, change := range changes {
		if seen[change.Path] {
			continue
		}
		seen[change.Path] = true
		deduped = append(deduped, change)
	}
	return deduped
}

func parseGitNameStatus(output string) []gitFileChange {
	var changes []gitFileChange
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		status := fields[0]
		path := fields[len(fields)-1]
		changes = append(changes, gitFileChange{Path: filepath.ToSlash(path), ChangeType: gitStatusToChangeType(status)})
	}
	return changes
}

func gitStatusToChangeType(status string) string {
	switch {
	case strings.HasPrefix(status, "M"):
		return "modified"
	case strings.HasPrefix(status, "A"):
		return "added"
	case strings.HasPrefix(status, "D"):
		return "deleted"
	case strings.HasPrefix(status, "R"):
		return "renamed"
	default:
		return strings.ToLower(status)
	}
}

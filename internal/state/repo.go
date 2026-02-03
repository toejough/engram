package state

import (
	"fmt"
	"os/exec"
	"strings"
)

// FindRepoRoot returns the git repository root for the given directory.
// Returns an error if the directory is not inside a git repository.
func FindRepoRoot(startDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = startDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

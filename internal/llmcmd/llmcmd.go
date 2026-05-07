// Package llmcmd spawns shell commands, pipes prompts to stdin, and captures
// stdout for LLM processing.
package llmcmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

const (
	defaultShell = "/bin/sh"
)

// Runner spawns a shell command, pipes the prompt to stdin, returns stdout.
type Runner struct {
	cmdString string
}

// New returns a Runner that invokes cmdString via /bin/sh -c when called.
func New(cmdString string) *Runner {
	return &Runner{cmdString: cmdString}
}

// Run pipes prompt to the command's stdin and returns trimmed stdout.
// The cmdString was provided at construction time and is treated as trusted.
//
//nolint:gosec // cmdString is set at construction, not from user input
func (r *Runner) Run(ctx context.Context, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, defaultShell, "-c", r.cmdString)
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("llm-cmd exited: %w (stderr: %s)",
			err, stderr.String())
	}

	return strings.TrimRight(stdout.String(), "\n"), nil
}

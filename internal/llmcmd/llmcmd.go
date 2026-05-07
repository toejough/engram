// Package llmcmd spawns shell commands, pipes prompts to stdin, and captures
// stdout for LLM processing.
package llmcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Runner spawns a shell command, pipes the prompt to stdin, returns stdout.
type Runner struct {
	cmdString string
	timeout   time.Duration
}

// New returns a Runner with the default 60s timeout.
func New(cmdString string) *Runner {
	return NewWithTimeout(cmdString, defaultTimeout)
}

// NewWithTimeout sets a custom wall-clock timeout.
func NewWithTimeout(cmdString string, timeout time.Duration) *Runner {
	return &Runner{cmdString: cmdString, timeout: timeout}
}

// Run pipes prompt to the command's stdin and returns trimmed stdout.
// The cmdString was provided at construction time and is treated as trusted.
//
//nolint:gosec // cmdString is set at construction, not from user input
func (r *Runner) Run(ctx context.Context, prompt string) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(timeoutCtx, defaultShell, "-c", r.cmdString)
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Env = append(os.Environ(), "ENGRAM_COMPANION_MODE=1")

	err := cmd.Run()
	if err != nil {
		if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("llm-cmd timeout after %s: %w", r.timeout, timeoutCtx.Err())
		}

		return "", fmt.Errorf("llm-cmd exited: %w (stderr: %s)", err, stderr.String())
	}

	return strings.TrimRight(stdout.String(), "\n"), nil
}

// unexported constants.
const (
	defaultShell   = "/bin/sh"
	defaultTimeout = 60 * time.Second
)

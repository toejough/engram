package llmcmd

import (
	"context"
	"fmt"
)

// CallerFunc returns a function with the signature expected by
// internal/cli/learn.go's llmCaller — model is ignored, system+user are
// concatenated into a single prompt and run through the shell command.
func CallerFunc(runner *Runner) func(context.Context, string, string, string) (string, error) {
	return func(ctx context.Context, _model, system, user string) (string, error) {
		prompt := system + "\n\n" + user

		out, err := runner.Run(ctx, prompt)
		if err != nil {
			return "", fmt.Errorf("calling llm-cmd: %w", err)
		}

		return out, nil
	}
}

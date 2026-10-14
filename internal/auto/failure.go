package auto

import (
	"errors"
	"regexp"
	"time"
)

// StepError is a failure with its class; any other error a step fails with is an `error`.
type StepError struct{ Message, Kind string }

// Error is the message of the failure.
func (e *StepError) Error() string { return e.Message }

// unexported constants.
const (
	deadlineCommand = 30 * time.Second
	deadlineIngest  = 300 * time.Second
	deadlineLookup  = 25 * time.Second
	deadlineMoment  = 5 * time.Second
	errorLimit      = 200
	kindError       = "error"
	kindTimeout     = "timeout"
	kindUnavailable = "unavailable"
)

// unexported variables.
var (
	timedOut    = regexp.MustCompile(`(?i)still running after|timed out|^aborted$`)
	unavailable = regexp.MustCompile(`(?i)failed to start|ENOENT|permission denied|no such file or directory|` +
		`read-only file system|input/output error`)
)

// exec runs one engram (or system) command with its deadline; a command that fails, cannot start or passes its
// deadline fails the step with its class (793b205).
func (e *engine) exec(argv []string, deadline time.Duration) string {
	return must(e.step(func() (string, error) {
		out, err := e.deps.Runner.Run(argv, deadline)
		if err != nil {
			text := cutEnd(out.Stderr+err.Error(), errorLimit)

			return "", &StepError{Message: argv[1] + " failed: " + text, Kind: failureKind(text)}
		}

		return out.Stdout, nil
	}))
}

// failureKind is the class of a failure from the text it left: the stderr of a command, the reason a model call gave.
func failureKind(text string) string {
	switch {
	case timedOut.MatchString(text):
		return kindTimeout
	case unavailable.MatchString(text):
		return kindUnavailable
	}

	return kindError
}

// kindOf is the class of an error: the one it carries, else the one its message shows.
func kindOf(err error) string {
	if step, found := errors.AsType[*StepError](err); found {
		return step.Kind
	}

	return failureKind(err.Error())
}

// warning is the one line the user is shown, on the status line and nowhere else: what was skipped and why.
func warning(step, kind string) string {
	reason := map[string]string{kindUnavailable: "the vault is unavailable", kindTimeout: "the System timed out"}[kind]
	if reason == "" {
		reason = "of an error"
	}

	return "Automatic memory was skipped because " + reason + " (" + step + ")"
}

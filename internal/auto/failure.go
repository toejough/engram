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
	// the query of a moment's lookup, which a prompt waits for: 1.5 times the 95th percentile of the query on the notes
	// of a real vault, six at once, 1.42 s (793b408)
	deadlineMomentQuery = 2200 * time.Millisecond
	// the query of the nearest notes of a capture or an offer, which nothing waits for
	deadlineNearest = 25 * time.Second
	errorLimit      = 200
	kindBehind      = "behind"
	kindBusy        = "busy"
	kindError       = "error"
	kindTimeout     = "timeout"
	kindUnavailable = "unavailable"
)

// unexported variables.
var (
	busy        = regexp.MustCompile(`(?i)holds the manifest lock`)
	timedOut    = regexp.MustCompile(`(?i)still running after|timed out|^aborted$`)
	unavailable = regexp.MustCompile(`(?i)failed to start|ENOENT|permission denied|no such file or directory|` +
		`read-only file system|input/output error`)
)

// amend amends a note, within the deadline of a command.
func (e *engine) amend(request AmendRequest) {
	e.perform("amend", func() (string, error) {
		return "", e.deps.Vault.Amend(request, deadlineCommand)
	})
}

// perform runs one operation of the vault; an operation that fails, cannot start or passes its deadline fails the step
// with its class (793b205). operation names the operation in the failure.
func (e *engine) perform(operation string, effect func() (string, error)) string {
	return must(e.step(func() (string, error) {
		out, err := effect()
		if err != nil {
			text := cutEnd(err.Error(), errorLimit)

			return "", &StepError{Message: operation + " failed: " + text, Kind: failureKind(text)}
		}

		return out, nil
	}))
}

// query searches the vault and returns its payload.
func (e *engine) query(request QueryRequest, deadline time.Duration) string {
	return e.perform("query", func() (string, error) { return e.deps.Vault.Query(request, deadline) })
}

// show reads a note.
func (e *engine) show(name string) string {
	return e.perform("show", func() (string, error) { return e.deps.Vault.Show(name, deadlineCommand) })
}

// failureKind is the class of a failure from the text it left: the stderr of a command, the reason a model call gave.
func failureKind(text string) string {
	switch {
	case busy.MatchString(text):
		return kindBusy
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
	reason := map[string]string{
		kindUnavailable: "the vault is unavailable", kindTimeout: "the System timed out",
		kindBusy: "another indexing is running", kindBehind: "the index is not caught up",
	}[kind]
	if reason == "" {
		reason = "of an error"
	}

	return "Automatic memory was skipped because " + reason + " (" + step + ")"
}

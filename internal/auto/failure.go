package auto

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
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
	// the longest reason a warning carries, in characters, before it is cut (793b451)
	reasonLimit = 120
	// the start of every warning
	skippedPrefix = "Automatic memory skipped: "
	// the operations a warning names, in the terms of the user (793b451)
	stepBacklog  = "backlog capture"
	stepCuration = "curation"
	stepIndexing = "indexing"
)

// unexported variables.
var (
	busy = regexp.MustCompile(`(?i)holds the manifest lock`)
	// the verb an operation of the vault adds to its text ("query failed: "), which the warning does not repeat
	stepVerb    = regexp.MustCompile(`^\S+ failed: `)
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

// consequence is what happens next to what the operation was to do, a statement the code makes true (793b451).
func consequence(step string) string {
	switch {
	case step == "prompt lookup":
		return "no notes delivered for this prompt"
	case strings.HasSuffix(step, " lookup"):
		return "no notes delivered this time"
	case step == "capture":
		return "nothing lost, the backlog capture covers it later"
	case step == "summary":
		return "no activity summary this turn"
	}

	return "nothing lost, retried next turn"
}

// cutToLine is the text on one line, without control characters (a terminal would act on them), cut to the limit with
// an ellipsis when it was longer.
func cutToLine(text string, limit int) string {
	printable := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return -1
		}

		return r
	}, text)
	line := strings.Join(strings.Fields(printable), " ")
	if runes := []rune(line); len(runes) > limit {
		return string(runes[:limit]) + "..."
	}

	return line
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

// predicateOf is what happened to an operation by the class of its failure; it follows the name of the operation.
func predicateOf(kind string) string {
	switch kind {
	case kindUnavailable:
		return "could not reach the vault"
	case kindTimeout:
		return "timed out"
	}

	return "failed"
}

// reasonOf is the text of a failure as the warning gives it: redacted, without the verb of its operation, on one line.
func reasonOf(text string) string {
	return cutToLine(stepVerb.ReplaceAllString(redact(text), ""), reasonLimit)
}

// warning is the one line the user is shown, on the status line and nowhere else: what was skipped, why, what happens
// next and where the details are.
func warning(step, kind, text, log string) string {
	return skippedPrefix + whatHappened(step, kind, reasonOf(text)) + "; " + consequence(step) + " (details: " + log + ")"
}

// whatHappened is the operation and what became of it, with the reason of the classes that have one.
func whatHappened(step, kind, reason string) string {
	switch kind {
	case kindBusy:
		if step == stepIndexing {
			return step + " was already running in another session"
		}

		return step + " was blocked because indexing was already running in another session"
	case kindBehind:
		return step + " was held back: the index is not caught up"
	}

	if reason == "" {
		return step + " " + predicateOf(kind)
	}

	return step + " " + predicateOf(kind) + ": " + reason
}

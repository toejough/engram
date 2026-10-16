package auto

import (
	"fmt"
	"strings"
	"time"
)

// unexported constants.
const (
	// the failure record is kept under a mebibyte: past it the oldest lines are dropped, down to three quarters of it so
	// that the file is not rewritten whole by every failure after it fills
	logKeep  = 768 * 1024
	logLimit = 1024 * 1024
)

// failureJob words and records a failure the mod hands over: one of its own steps, which the engine cannot see
// (793b451).
func (e *engine) failureJob() Done {
	panic(&StepError{Message: e.job.Text, Kind: failureKind(e.job.Text)})
}

// record appends a failure to the failure record (793d36b4). A record that cannot be written is a line of the debug log
// and nothing else: the warning and the job go on as they were.
func (e *engine) record(step, kind, text string) {
	if e.deps.LogPath == "" {
		return
	}

	line := fmt.Sprintf("%s session=%q operation=%q class=%s reason=%q\n",
		e.deps.Clock.Now().UTC().Format(time.RFC3339), orElse(e.job.SessionID, "-"), step, kind,
		cutToLine(cutEnd(redact(text), errorLimit), errorLimit))

	err := e.deps.Files.Write(e.deps.LogPath, cutOldest(e.quietly(e.deps.LogPath)+line))
	if err != nil {
		e.logf("auto.log could not be written: %s", err)
	}
}

// shownLog is the path of the failure record as the warning shows it: under the home directory, a tilde.
func (e *engine) shownLog() string {
	if e.deps.Home != "" && strings.HasPrefix(e.deps.LogPath, e.deps.Home+"/") {
		return "~" + strings.TrimPrefix(e.deps.LogPath, e.deps.Home)
	}

	return e.deps.LogPath
}

// cutOldest drops the oldest lines of a record that has passed its limit, down to what is kept; the newest line stays.
func cutOldest(record string) string {
	if len(record) <= logLimit {
		return record
	}

	rest := record[len(record)-logKeep:]
	if newline := strings.IndexByte(rest, '\n'); newline >= 0 && newline+1 < len(rest) {
		return rest[newline+1:]
	}

	return rest
}

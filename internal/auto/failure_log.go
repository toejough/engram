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
	// no limit at all: the size of text that is not cut
	noLimit = 1 << 30
	// how many characters of the start and of the end of a reply the mod could not read the failure shows (793b469)
	replyEndChars = 80
	// how many characters at each end of such a reply are redacted before the ends are cut from them
	replyWindow = 8192
)

// failureJob words and records a failure the mod hands over: one of its own steps, which the engine cannot see
// (793b451). A reply the mod could not read comes with its size and its ends (793b469).
func (e *engine) failureJob() Done {
	// the mod cut the text before it reached the engine, so the end of it may be the start of a credential
	message := maskTail(redact(e.job.Text))

	panic(&StepError{Message: message, Kind: failureKind(e.job.Text), Detail: replyDetail(e.job)})
}

// record appends a failure to the failure record (793d36b4). A record that cannot be written is a line of the debug log
// and nothing else: the warning and the job go on as they were.
func (e *engine) record(step, kind, text, detail string) {
	if e.deps.LogPath == "" {
		return
	}

	line := fmt.Sprintf("%s session=%q operation=%q class=%s reason=%q\n",
		e.deps.Clock.Now().UTC().Format(time.RFC3339), orElse(e.job.SessionID, "-"), step, kind,
		cutToLine(cutRedacted(tidy(redact(text)), errorLimit), errorLimit)+detail)

	err := e.deps.Files.Write(e.deps.LogPath, cutOldest(e.quietly(e.deps.LogPath)+line))
	if err != nil {
		e.logf("auto.log could not be written: %s", err)
	}
}

// shownLog is the path of the failure record as the warning shows it: under the home directory, a tilde.
func (e *engine) shownLog() string { return e.shown(e.deps.LogPath) }

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

// replyDetail says how large a reply the mod could not read was, in bytes, and how it began and ended: what shows a
// reply that was cut short. The reply may be megabytes and a redaction of all of it takes minutes, so the ends are
// taken from a window of replyWindow characters at each end, where the window's own edges and the end of the reply
// (which the process cut) are hidden when a run of token-like characters touches them; the window is redacted whole,
// and the 80 characters shown are cut from the redacted text and hidden again at their own edges, so that no
// credential is split by a cut before it is recognized (793b471).
func replyDetail(job Job) string {
	if job.Unreadable == "" {
		return ""
	}

	head := redact(maskTail(maskHead(cutEnd(job.Unreadable, replyWindow))))
	tail := redact(maskTail(maskHead(cutStart(job.Unreadable, replyWindow))))
	head = maskTail(maskHead(cutEnd(cutToLine(head, noLimit), replyEndChars)))
	tail = maskTail(maskHead(cutStart(cutToLine(tail, noLimit), replyEndChars)))

	return fmt.Sprintf(" (reply of %d bytes, starts \"%s\", ends \"%s\")", len(job.Unreadable), head, tail)
}

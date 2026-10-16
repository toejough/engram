package auto

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// unexported constants.
const (
	// the file the log lines a reply does not carry are appended to, beside the failure record (793d36b1)
	debugLogName = "auto-debug.log"
	// the most `engram auto run` prints for one round, in bytes (793d36b1). The mod reads what a program prints through
	// `$.process.run`, which resolves no more than 4 MiB of standard output and cuts the rest without an error (SP18,
	// 793b467), so a reply over that limit is a document cut short. The limit is a sixteenth of it.
	replyLimit = 256 * 1024
	// the longest log line and warning a reply carries, in characters
	replyLineLimit = 300
	// the most log lines a reply carries; the lines after them go to the debug file
	replyLogLines     = 20
	replyWarningLimit = 1000
	// the most warnings a reply carries
	replyWarnings = 10
)

// boundLog cuts the log lines of the round to the lines a reply carries, and appends the others to the debug file.
func (e *engine) boundLog() bool {
	cut := false

	for index, line := range e.resp.Log {
		if shortened := cutEnd(line, replyLineLimit); shortened != line {
			e.resp.Log[index], cut = shortened+"...", true
		}
	}

	if len(e.resp.Log) <= replyLogLines {
		return cut
	}

	kept, spilled := e.resp.Log[:replyLogLines-1], e.resp.Log[replyLogLines-1:]
	where := "left out"

	if path := e.debugLogPath(); path != "" {
		err := e.deps.Files.Write(path, cutOldest(e.quietly(path)+strings.Join(spilled, "\n")+"\n"))
		if err == nil {
			where = "in " + e.shown(path)
		}
	}

	e.resp.Log = append(slices.Clone(kept), fmt.Sprintf("engram-memory: %d more log line(s) %s", len(spilled), where))

	return true
}

// boundWarnings keeps the first warnings of the round, each cut: every one of them is in the failure record.
func (e *engine) boundWarnings() bool {
	cut := len(e.resp.Warnings) > replyWarnings
	e.resp.Warnings = e.resp.Warnings[:min(len(e.resp.Warnings), replyWarnings)]

	for index, text := range e.resp.Warnings {
		if shortened := cutEnd(text, replyWarningLimit); shortened != text {
			e.resp.Warnings[index], cut = shortened+"...", true
		}
	}

	return cut
}

// debugLogPath is the file of the log lines a reply does not carry: beside the failure record.
func (e *engine) debugLogPath() string {
	if slash := strings.LastIndex(e.deps.LogPath, "/"); slash >= 0 {
		return e.deps.LogPath[:slash+1] + debugLogName
	}

	return ""
}

// encode is the reply as JSON, under replyLimit (793d36b1): the log lines past the first few go to the debug file, a
// line and a warning are cut, and a reply that is over the limit all the same, because of the record of the job or the
// request it relays, is replaced by the end of the job with a warning. A reply that was bounded says `truncated`.
func (e *engine) encode() []byte {
	cut := e.boundLog()
	cut = e.boundWarnings() || cut
	e.resp.Truncated = e.resp.Truncated || cut

	out := must(json.Marshal(e.resp)) // a response of plain values cannot fail to encode
	if len(out) <= replyLimit {
		return out
	}

	e.overflowed(len(out))

	out = must(json.Marshal(e.resp))
	if len(out) > replyLimit {
		e.resp.Done.Ledger, e.resp.Done.Work = nil, nil
		out = must(json.Marshal(e.resp))
	}

	return out
}

// overflowed ends a job whose reply passed the limit: what the job relayed or delivered cannot be sent, so the job
// ends with a warning that names the step, and the session state the mod handed over stays as it was.
func (e *engine) overflowed(size int) {
	e.resp = Response{
		Log: e.resp.Log, Warnings: e.resp.Warnings, Counts: e.resp.Counts, Truncated: true,
		Done: &Done{Ledger: e.job.Ledger, Work: e.job.Work},
	}
	e.warn(e.stepName(), &StepError{
		Message: fmt.Sprintf("the reply of %d bytes passes the limit of %d bytes", size, replyLimit), Kind: kindError,
	})
}

// shown is a path as the warning shows it: under the home directory, a tilde.
func (e *engine) shown(path string) string {
	if e.deps.Home != "" && strings.HasPrefix(path, e.deps.Home+"/") {
		return "~" + strings.TrimPrefix(path, e.deps.Home)
	}

	return path
}

// stepName is the step of the job, in the terms of the user.
func (e *engine) stepName() string {
	switch {
	case e.job.Step != "":
		return e.job.Step
	case e.job.Kind == "housekeeping":
		return orElse(e.job.Part, stepHousekeeping)
	case e.job.Kind == "summary":
		return stepSummary
	}

	return stepCapture
}

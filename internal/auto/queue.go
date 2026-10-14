package auto

import (
	"encoding/json"
	"slices"
)

// QueueEntry is a turn capture that was queued and has not finished (793b398): the window and the source it will be
// written with, the transcript of the session it came from (its turns are marked captured when it ends), and how many
// times a later session has taken it up.
type QueueEntry struct {
	Window     string `json:"window"`
	Source     string `json:"source"`
	Transcript string `json:"transcript,omitempty"`
	Tries      int    `json:"tries,omitempty"`
}

// unexported constants.
const (
	queueFile  = "/capture-queue.json"
	queueTries = 3
)

// dequeue takes the entry out of the queue file once its capture has ended; a file that does not hold it is left as it
// is.
func (e *engine) dequeue(entry QueueEntry) {
	e.updateQueue(func(entries []QueueEntry) []QueueEntry {
		return slices.DeleteFunc(entries, func(queued QueueEntry) bool {
			return queued.Window == entry.Window && queued.Source == entry.Source
		})
	})
}

// enqueueJob writes the turn to the queue file and hands the entry back, so that the turn is on disk before the event
// returns. A turn that cannot be written is left to the mod to capture from memory, as before the queue: the failure
// is the System's own bookkeeping and goes to the debug log only.
func (e *engine) enqueueJob() Done {
	entry := e.turnEntry()
	if entry.Window == "" {
		return Done{}
	}

	e.updateQueue(func(entries []QueueEntry) []QueueEntry { return append(entries, entry) })

	return Done{Queued: &entry}
}

// resumeJob hands a session that starts the entries its predecessors left in the file: a new process, `startup` or
// `resume`, and not a clear or a compaction of the process that holds the queue's own jobs. An entry that has been
// taken up three times is dropped, so that a window the judge cannot read is not tried for ever.
func (e *engine) resumeJob() Done {
	if e.job.Start != "startup" && e.job.Start != "resume" {
		return Done{}
	}

	var handed []QueueEntry

	e.updateQueue(func(entries []QueueEntry) []QueueEntry {
		handed = slices.DeleteFunc(entries, func(entry QueueEntry) bool { return entry.Tries >= queueTries })
		if dropped := len(entries) - len(handed); dropped > 0 {
			e.logf("capture queue: dropped %d entries taken up %d times", dropped, queueTries)
		}

		for index := range handed {
			handed[index].Tries++
		}

		return slices.Clone(handed)
	})

	return Done{Resumed: handed}
}

// turnEntry is the turn of the job as the queue holds it: the window of the turn, redacted before it is written.
func (e *engine) turnEntry() QueueEntry {
	return QueueEntry{
		Window:     redact(windowFromMessages(e.job.Messages, e.job.Answer)),
		Source:     sourceFor(e.job.SessionID, e.job.Turn, ""),
		Transcript: e.job.Transcript,
	}
}

// updateQueue changes the queue file of the chunks directory, beside the backlog marks. A file that is absent or is not
// a queue is an empty queue; a file is written only when the change makes it differ.
func (e *engine) updateQueue(change func(entries []QueueEntry) []QueueEntry) {
	file := e.chunksDirectory() + queueFile
	text, _ := e.read(file)

	var entries []QueueEntry
	if json.Unmarshal([]byte(text), &entries) != nil {
		entries = nil
	}

	before := must(json.Marshal(append([]QueueEntry{}, entries...)))
	changed := must(json.Marshal(append([]QueueEntry{}, change(entries)...)))

	if string(before) != string(changed) {
		e.write(file, string(changed))
	}
}

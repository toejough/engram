package auto

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Marks is the mark file: `Sources` maps a source path to the hash of the file when its backlog capture finished;
// `Turns` maps a session's file name to the time its turns were last captured by the System itself; `Partial` maps a
// source path to its progress (the windows are counted from the end of the conversation); `Installed` and `Cutoff` are
// the time of the first run and the fixed cut-off (that time minus 7 days), absent in a file that has not had its
// first run yet.
type Marks struct {
	Sources   map[string]string      `json:"sources"`
	Turns     map[string]float64     `json:"turns"`
	Partial   map[string]partialMark `json:"partial"`
	Installed *float64               `json:"installed,omitempty"`
	Cutoff    *float64               `json:"cutoff,omitempty"`
}

// firstRun records the first run: the install time and the cut-off, once. A mark file that has a cut-off keeps it.
func (m *Marks) firstRun(now float64) {
	if m.Cutoff == nil {
		cutoff := now - cutoffMs
		m.Installed, m.Cutoff = &now, &cutoff
	}
}

// progress records that the first `done` windows of a source are judged, or that the source is captured: the mark is
// the hash it has now, and the progress is dropped.
func (m *Marks) progress(source Source, done int, finished bool) {
	if finished {
		m.Sources[source.Path] = source.Hash
		delete(m.Partial, source.Path)
	} else {
		m.Partial[source.Path] = partialMark{Hash: source.Hash, Done: done}
	}
}

// Source is a transcript the manifest knows, with the progress the marks hold for it once a run has been planned.
type Source struct {
	Path    string
	MtimeMs float64
	Hash    string
	Partial partialMark
}

// unexported constants.
const (
	cutoffMs   = 7 * 24 * 60 * 60 * 1000
	marksFile  = "/backlog-marks.json"
	nanosPerMs = 1e6
	// A source is left alone while it may still be running (quiet). History before the cut-off is not mined: the
	// cut-off is one fixed point, install time minus 7 days, recorded in the mark file at the first run (793b323). It
	// is not a window that moves: a source that ended after it stays eligible until it is captured, however long the
	// sessions stopped.
	quietMs = 15 * 60 * 1000
	// A file the platform will not read whole (over 4 MiB) is read from its tail; the first line is then cut and is
	// dropped.
	readLimit      = 4 << 20
	sourcesPerTurn = 20
	tailBytes      = 3000000
	// The rows a harness writes just after the last turn the System captured (a closing row, a flush) are not a turn:
	// a source changed by no more than this after that turn is covered by it. A turn that ends later is captured
	// (793b317).
	turnSlackMs = 10 * 1000
	// What one turn's backlog work does: the windows judged, and the sources looked at (a source with nothing to
	// capture is marked, but each looked at costs a read).
	windowsPerTurn = 6
)

// backlogPlan is what the backlog takes from the manifest and the marks: whether this is the first run, whether the
// index is behind, else the sources it may look at in this run, the most recently changed first, each with the
// progress its marks hold.
type backlogPlan struct {
	FirstRun bool     `json:"firstRun,omitempty"`
	Behind   bool     `json:"behind,omitempty"`
	Todo     []Source `json:"todo,omitempty"`
}

// batch is what one source asks of a turn: the windows to judge, the end of the conversation first.
type batch struct {
	Unreadable string
	Windows    []string
	Messages   int
	Done       int
	Finished   bool
	Empty      bool
}

// backlog is the job after a turn's housekeeping (793e8g), whose ingest has brought the manifest up to date: take the
// one most recently changed source nobody has captured yet and judge up to windowsPerTurn of its windows, the end of
// the conversation first, with the same judgement as a turn. The source is marked only when every window has been
// judged without a failure; a source with nothing to judge is marked at once, one that cannot be read is left alone.
// The manifest is read once and what it holds is not kept in the record of the job: the record is sent with every
// round, and the manifest of a real vault is tens of megabytes (793b467, 793b468).
func (e *engine) backlog() {
	dir := e.deps.ChunksDir
	now := e.now()
	plan := recorded(e, func() backlogPlan { return e.planBacklog(dir, now) })

	if plan.FirstRun {
		e.updateMarks(dir, func(stored *Marks) { stored.firstRun(now) })
	}

	if plan.Behind {
		e.warn(stepBacklog, &StepError{
			Message: "backlog: index not caught up: its newest session changed before the cut-off", Kind: kindBehind,
		})

		return
	}

	for _, source := range plan.Todo {
		next := recorded(e, func() batch { return e.prepare(source) })
		if next.Unreadable != "" {
			e.logf("backlog: %s: %s", source.Path, next.Unreadable)
			e.work.Unreadable = append(e.work.Unreadable, source.Path)

			continue
		}

		for index, conversation := range next.Windows {
			e.capture(conversation, backlogSourceFor(source.Path, fmt.Sprintf("message %d", next.Messages)))
			e.updateMarks(dir, func(stored *Marks) {
				stored.progress(source, next.Done+index+1, next.Finished && index == len(next.Windows)-1)
			})
		}

		// a source with no window left to judge is marked; one with no message in it is no work, and the next is looked at
		if len(next.Windows) > 0 {
			return
		}

		e.updateMarks(dir, func(stored *Marks) { stored.progress(source, next.Done, next.Finished) })

		if !next.Empty {
			return
		}
	}
}

// indexBehind says whether the index has not caught up: its newest session changed before the cut-off, so nothing can
// be eligible, and the user is told instead of a backlog that silently finds nothing (793d28b1).
func (e *engine) indexBehind(sources []Source, marks Marks) bool {
	return len(sources) > 0 && marks.Cutoff != nil && sources[0].MtimeMs < *marks.Cutoff
}

// markTurns records that the System captured the turns of this session up to now, so that the backlog does not read
// the session again once it is quiet (793e8g).
func (e *engine) markTurns(transcript string) {
	now := e.now()
	if transcript != "" {
		e.updateMarks(e.deps.ChunksDir, func(marks *Marks) {
			marks.firstRun(now)
			marks.Turns[baseName(transcript)] = now
		})
	}
}

// now is the time in milliseconds since the epoch.
func (e *engine) now() float64 {
	return recorded(e, func() float64 { return float64(e.deps.Clock.Now().UnixMilli()) })
}

// planBacklog reads the manifest, which may be tens of megabytes, and the marks, which grow with every source
// captured, and keeps of them what a run can use: a run looks at sourcesPerTurn sources at most, so a source that
// cannot be read, which is passed over, and the sources a run can look at are all that is kept. An absent manifest is
// an empty one. The files are read with `quietly` inside the one recorded effect, so the record of the job holds the
// plan and never a file (793b468).
func (e *engine) planBacklog(dir string, now float64) backlogPlan {
	marks := parseMarks(e.quietly(dir + marksFile))
	first := marks.Cutoff == nil

	marks.firstRun(now)

	sources := parseManifest(e.quietly(dir + "/manifest.json"))
	if e.indexBehind(sources, marks) {
		return backlogPlan{FirstRun: first, Behind: true}
	}

	left := slices.DeleteFunc(pending(sources, marks, now, e.job.Transcript), func(source Source) bool {
		return slices.Contains(e.work.Unreadable, source.Path)
	})
	todo := left[:min(len(left), sourcesPerTurn)]
	for index := range todo {
		todo[index].Partial = marks.Partial[todo[index].Path]
	}

	return backlogPlan{FirstRun: first, Todo: todo}
}

// prepare reads a source and cuts its next windows: from the windows (oldest first), the last `done` are left and up
// to windowsPerTurn before them are taken. The file is read from its tail when the platform would not read it whole.
func (e *engine) prepare(source Source) batch {
	text, err := e.deps.Files.Read(source.Path)
	if err != nil {
		return batch{Unreadable: err.Error()}
	}

	if len(text) > readLimit {
		text = text[len(text)-tailBytes:]
		text = text[strings.Index(text, "\n")+1:]
	}

	messages := messagesFromTranscript(text)
	windows := windowsFromMessages(messages)
	done := source.Partial.Done

	if source.Partial.Hash != source.Hash {
		done = 0
	}

	end := max(len(windows)-done, 0)
	next := slices.Clone(windows[max(end-windowsPerTurn, 0):end])
	slices.Reverse(next)

	finished := end-len(next) <= 0

	return batch{Windows: next, Empty: len(windows) == 0, Messages: len(messages), Done: done, Finished: finished}
}

// updateMarks changes the mark file of the chunks directory: it is read, changed and written in one effect, so that a
// replay of the job neither reads it again nor records it.
func (e *engine) updateMarks(dir string, change func(marks *Marks)) {
	must(e.step(func() (string, error) {
		marks := parseMarks(e.quietly(dir + marksFile))
		change(&marks)

		return "", e.deps.Files.Write(dir+marksFile, string(must(json.Marshal(marks))))
	}))
}

// partialMark records that the last `Done` windows of a source, at its hash, are judged.
type partialMark struct {
	Hash string `json:"hash"`
	Done int    `json:"done"`
}

// baseName is the last element of a path.
func baseName(file string) string { return file[strings.LastIndex(file, "/")+1:] }

// isFalsy is JavaScript's falsiness of a decoded JSON value.
func isFalsy(value any) bool { return slices.Contains([]any{nil, "", false, 0.0}, value) }

// orEmpty is a map, or an empty one.
func orEmpty[M ~map[K]V, K comparable, V any](values M) M {
	if values == nil {
		return M{}
	}

	return values
}

// parseManifest lists the session transcripts the manifest knows, the most recently changed first: a `.jsonl` source
// that is not a subagent's (its own stop captured it) and not a duplicate of another source (the canonical one is the
// one indexed). An absent manifest is an empty one.
func parseManifest(text string) []Source {
	var manifest map[string]map[string]any

	check(strings.TrimSpace(text) == "" || (json.Unmarshal([]byte(text), &manifest) == nil && manifest != nil),
		"the ingest manifest is not an object")

	var sources []Source

	for name, entry := range manifest {
		nanos, _ := entry["mtime_unix_nano"].(float64)
		hash, _ := entry["file_hash"].(string)

		if strings.HasSuffix(name, ".jsonl") && !strings.Contains(name, "/subagents/") && isFalsy(entry["duplicate_of"]) {
			sources = append(sources, Source{Path: name, MtimeMs: nanos / nanosPerMs, Hash: hash})
		}
	}

	sort.Slice(sources, func(i, j int) bool {
		left, right := sources[i], sources[j]

		return left.MtimeMs > right.MtimeMs || (left.MtimeMs == right.MtimeMs && left.Path < right.Path)
	})

	return sources
}

// parseMarks reads the mark file; an absent one is an empty one.
func parseMarks(text string) Marks {
	var marks Marks

	read := json.Unmarshal([]byte(text), &marks) == nil && marks.Sources != nil && marks.Turns != nil

	check(strings.TrimSpace(text) == "" || read, "the backlog marks are not {sources, turns}")

	marks.Sources, marks.Turns, marks.Partial = orEmpty(marks.Sources), orEmpty(marks.Turns), orEmpty(marks.Partial)

	return marks
}

// pending are the sources to capture: not the live session (by its file name), quiet for a while, changed after the
// cut-off (none, while the marks hold no cut-off), and neither marked with their present hash nor covered by the
// System's own turn captures.
func pending(sources []Source, marks Marks, now float64, live string) []Source {
	return slices.DeleteFunc(slices.Clone(sources), func(source Source) bool {
		turn, covered := marks.Turns[baseName(source.Path)]
		mark, marked := marks.Sources[source.Path]

		return (live != "" && baseName(source.Path) == baseName(live)) || marks.Cutoff == nil ||
			now-source.MtimeMs < quietMs || source.MtimeMs < *marks.Cutoff || (marked && mark == source.Hash) ||
			(covered && source.MtimeMs <= turn+turnSlackMs)
	})
}

package auto_test

// The persisted queue of turn captures (793b362, 793b398): a turn is written to the queue file before its hook
// returns, taken out when its capture ends, kept when the capture fails, and handed to the next session that starts.

import (
	"encoding/json"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/auto"
)

func Test793d27b1_ACaptureWithNoEntryIsTheTurnItself(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()

	world.Do(t, captureOf(turnMessages()))

	g.Expect(world.Requests).To(HaveLen(1))
	g.Expect(world.Requests[0].Prompt).To(ContainSubstring("User: No, use targ, not go test directly."))
	g.Expect(world.Written).NotTo(HaveKey(queueFile))
}

func Test793d28b3_ACaptureOfAnEntryMarksTheSessionTheEntryCameFromNotTheOneRunning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Files[queueFile] = queueText(leftBehind(0))

	world.Do(t, map[string]any{
		"kind": "capture", "entry": leftBehind(0), "transcript": transcriptPath, "step": "the capture",
	})

	turns, written := marksOf(t, world, defaultChunks)
	g.Expect(written).To(BeTrue())
	g.Expect(turns).To(HaveKey("old-session.jsonl"))
	g.Expect(turns).NotTo(HaveKey("9f1e2d3c.jsonl"))
}

func Test793d37b1_AFileThatCannotBeWrittenQueuesNothingAndWarnsNoOneSoTheTurnIsCapturedFromMemory(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Unwritable[queueFile] = true
	job := captureOf(turnMessages())
	job["kind"] = "enqueue"
	delete(job, "step") // the mod names no step for the queue: its failure is not a warning

	done := world.Do(t, job)

	g.Expect(done.Queued).To(BeNil())
	g.Expect(world.Warnings).To(BeEmpty())
	g.Expect(strings.Join(world.Logs, "\n")).To(ContainSubstring("permission denied"))
}

func Test793d37b1_ATurnIsAddedAfterTheOnesAlreadyInTheFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Files[queueFile] = queueText(leftBehind(2))
	job := captureOf(turnMessages())
	job["kind"] = "enqueue"

	world.Do(t, job)

	entries := queueOf(t, world)
	g.Expect(entries).To(HaveLen(2))
	g.Expect(entries[0]).To(Equal(leftBehind(2)))
	g.Expect(entries[1].Window).To(ContainSubstring("User: No, use targ"))
}

func Test793d37b1_ATurnIsRedactedBeforeItIsWrittenToDisk(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	// the second secret sits in the name of a tool, a line the window writes without redacting the message: only the
	// redaction of the window itself hides it
	toolNamed := map[string]any{
		"role": "assistant", "text": "", "toolUses": []map[string]any{{"tool": "api_key=sk-" + queueSecretTail}},
	}
	job := captureOf([]any{said("user", "No. password: hunter2hunter2 is wrong"), toolNamed})
	job["kind"] = "enqueue"

	world.Do(t, job)

	g.Expect(world.Written[queueFile]).To(ContainSubstring("[redacted]"))
	g.Expect(world.Written[queueFile]).NotTo(ContainSubstring("hunter2hunter2"))
	g.Expect(world.Written[queueFile]).NotTo(ContainSubstring(queueSecretTail))
}

func Test793d37b1_ATurnIsWrittenToTheFileWithItsWindowAndSourceAndNothingIsJudged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	job := captureOf(turnMessages())
	job["kind"], job["transcript"] = "enqueue", transcriptPath

	done := world.Do(t, job)

	entries := queueOf(t, world)
	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Window).To(Equal(
		"Agent: I will run go test ./...\nUser: No, use targ, not go test directly.\nAgent: Understood, I will use targ."))
	g.Expect(entries[0].Source).To(Equal(auto.SourceFor("sess-1", 3, "")))
	g.Expect(entries[0].Transcript).To(Equal(transcriptPath))
	g.Expect(done.Queued).NotTo(BeNil())
	g.Expect(*done.Queued).To(Equal(auto.QueueEntry{
		Window: entries[0].Window, Source: entries[0].Source, Transcript: transcriptPath,
	}))
	g.Expect(world.Requests).To(BeEmpty())
	g.Expect(world.Commands).To(BeEmpty())
}

func Test793d37b1_ATurnWithNoWordsOfTheUserIsNotQueued(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	job := captureOf([]any{said("assistant", "Done.")})
	job["kind"], job["answer"] = "enqueue", "Done."

	done := world.Do(t, job)

	g.Expect(done.Queued).To(BeNil())
	g.Expect(world.Writes).To(BeEmpty())
}

func Test793d37b1_NoFileAndAFileThatIsNotAQueueAreAnEmptyQueue(t *testing.T) {
	t.Parallel()

	for name, text := range map[string]string{
		"no file": "", "not json": "{{", "an object": `{"a":1}`, "entries of nothing": "[1]",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			world := NewWorld()

			if text != "" {
				world.Files[queueFile] = text
			}

			done := world.Do(t, map[string]any{"kind": "resume", "start": "startup"})

			g.Expect(done.Resumed).To(BeEmpty())
			g.Expect(world.Warnings).To(BeEmpty())
		})
	}
}

func Test793d37b2_ASessionThatIsNotNewTakesNothingAndWritesNothing(t *testing.T) {
	t.Parallel()

	// a clear or a compaction happens in the process that holds the queue's own jobs: taking them would capture twice
	for _, start := range []string{"clear", "compact", ""} {
		t.Run(start, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			world := NewWorld()
			world.Files[queueFile] = queueText(leftBehind(0))

			done := world.Do(t, map[string]any{"kind": "resume", "start": start})

			g.Expect(done.Resumed).To(BeEmpty())
			g.Expect(world.Writes).To(BeEmpty())
		})
	}
}

func Test793d37b2_ASessionThatStartsTakesTheEntriesLeftBehindAndCountsTheTry(t *testing.T) {
	t.Parallel()

	for _, start := range []string{"startup", "resume"} {
		t.Run(start, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			world := NewWorld()
			world.Files[queueFile] = queueText(leftBehind(0), leftBehind(2))

			done := world.Do(t, map[string]any{"kind": "resume", "start": start})

			g.Expect(done.Resumed).To(Equal([]auto.QueueEntry{entryOf(leftBehind(1)), entryOf(leftBehind(3))}))
			g.Expect(queueOf(t, world)).To(Equal([]queued{leftBehind(1), leftBehind(3)}))
		})
	}
}

func Test793d37b2_AnEntryThatHasBeenTakenThreeTimesIsDroppedWithoutBeingTakenAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Files[queueFile] = queueText(leftBehind(3), leftBehind(0))

	done := world.Do(t, map[string]any{"kind": "resume", "start": "startup"})

	g.Expect(done.Resumed).To(Equal([]auto.QueueEntry{entryOf(leftBehind(1))}))
	g.Expect(queueOf(t, world)).To(Equal([]queued{leftBehind(1)}))
	g.Expect(strings.Join(world.Logs, "\n")).To(ContainSubstring("dropped"))
}

func Test793d37b3_ACaptureOfAnEntryTakesItOutOfTheFileWhenItEnds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Files[queueFile] = queueText(leftBehind(0), queued{Window: "User: other", Source: "s2"})

	done := world.Do(t, map[string]any{
		"kind": "capture", "entry": leftBehind(0), "transcript": transcriptPath, "step": "the capture",
	})

	g.Expect(done.Queued).To(BeNil())
	g.Expect(world.Requests).To(HaveLen(1))
	g.Expect(world.Requests[0].Prompt).To(ContainSubstring("User: No, use targ, not go test directly."))
	g.Expect(queueOf(t, world)).To(Equal([]queued{{Window: "User: other", Source: "s2"}}))
}

func Test793d37b3_ACaptureThatFailsKeepsItsEntryInTheFileForTheNextSession(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Model = func(string, auto.Request) map[string]any { return Unanswered("api-error") }
	world.Files[queueFile] = queueText(leftBehind(1))

	world.Do(t, map[string]any{"kind": "capture", "entry": leftBehind(1), "step": "the capture"})

	g.Expect(world.Warnings).To(HaveLen(1))
	g.Expect(queueOf(t, world)).To(Equal([]queued{leftBehind(1)}))
	g.Expect(world.Writes).To(BeEmpty())
}

func Test793d37b3_TheFileHoldsExactlyTheTurnsNotYetCapturedWhateverTheOrder(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		world := NewWorld()
		count := rapid.IntRange(1, len(turnShapes)).Draw(rt, "count")
		entries := make([]auto.QueueEntry, 0, count)

		// turns that share a window, turns that share a source, and no two the same in both
		for index := range count {
			shape := turnShapes[index]
			job := captureOf([]any{said("user", "No, use targ "+strings.Repeat("x", shape.words))})
			job["kind"], job["turn"] = "enqueue", shape.turn

			entries = append(entries, *world.Do(&testing.T{}, job).Queued)
		}

		order := rapid.Permutation(entries).Draw(rt, "order")
		for done, entry := range order {
			world.Do(&testing.T{}, map[string]any{"kind": "capture", "entry": entry, "step": "the capture"})

			if left := len(queueOf(&testing.T{}, world)); left != count-done-1 {
				rt.Fatalf("%d entries left after %d captures of %d", left, done+1, count)
			}
		}

		if text := strings.TrimSpace(world.Written[queueFile]); text != "[]" {
			rt.Fatalf("the file of a finished queue is %q", text)
		}
	})
}

// unexported constants.
const (
	leftBehindSource = "captured from the conversation; written automatically (session left-behind, turn 9)"
	leftBehindWindow = "Agent: I will run go test ./...\nUser: No, use targ, not go test directly.\nAgent: Understood."
	oldTranscript    = "/private/home/.claude/projects/p/old-session.jsonl"
	queueFile        = defaultChunks + "/capture-queue.json"
	queueSecretTail  = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// unexported variables.
var (
	turnShapes = []struct{ words, turn int }{{1, 0}, {1, 1}, {1, 2}, {2, 0}, {2, 1}, {2, 2}}
)

// queued is an entry of the queue file as the tests read it.
type queued struct {
	Window     string `json:"window"`
	Source     string `json:"source"`
	Transcript string `json:"transcript"`
	Tries      int    `json:"tries"`
}

// entryOf is an entry of the file as the job hands it.
func entryOf(entry queued) auto.QueueEntry {
	return auto.QueueEntry{
		Window: entry.Window, Source: entry.Source, Transcript: entry.Transcript, Tries: entry.Tries,
	}
}

// leftBehind is the turn a session that ended left in the file, taken `tries` times already.
func leftBehind(tries int) queued {
	return queued{Window: leftBehindWindow, Source: leftBehindSource, Transcript: oldTranscript, Tries: tries}
}

// queueOf is the entries of the queue file the job wrote last.
func queueOf(t *testing.T, world *World) []queued {
	t.Helper()

	var entries []queued

	err := json.Unmarshal([]byte(world.Files[queueFile]), &entries)
	if err != nil {
		t.Fatalf("the queue file cannot be read: %v", err)
	}

	return entries
}

// queueText is the queue file holding these entries.
func queueText(entries ...queued) string {
	text, err := json.Marshal(entries)
	if err != nil {
		panic(err)
	}

	return string(text)
}

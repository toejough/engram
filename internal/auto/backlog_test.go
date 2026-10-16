package auto_test

import (
	"encoding/json"
	"fmt"
	stdmaps "maps"
	"regexp"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test793d28b1_ABrokenManifestOrMarkFileIsReportedAndNothingIsCaptured(t *testing.T) {
	t.Parallel()

	for name, files := range map[string]map[string]string{
		"manifest": {blManifestFile: `["not","an","object"]`},
		"marks":    {blManifestFile: `{}`, blMarksFile: `{"sources":3}`},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			world := NewWorld()
			world.Clock = time.UnixMilli(blNowMs)
			world.ChunksDir = "/data/engram/chunks"

			stdmaps.Copy(world.Files, files)

			blHousekeep(t, world, "")

			g.Expect(world.Names()).To(Equal([]string{"ingest"}))
			g.Expect(world.Warnings).To(HaveLen(1))
			g.Expect(world.Warnings[0]).To(ContainSubstring("backlog capture"))
		})
	}
}

func Test793d28b1_AMarkFileWithNoCutoffGetsOneAndKeepsItsMarks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blOneQuiet()
	world.Files[blMarksFile] = blMarksJSON(map[string]string{blPi: blHashA}, map[string]float64{})

	blHousekeep(t, world, "")

	marks := blMarksOf(t, world)
	g.Expect(marks.Cutoff).To(HaveValue(BeEquivalentTo(blNowMs - 7*blDayMs)))
	g.Expect(marks.Sources[blPi]).To(Equal(blHashA))
	g.Expect(world.Names()).To(Equal([]string{"ingest"}))
}

func Test793d28b1_ASourceMayStillBeRunningIsNotCaptured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	manifest := blManifest{
		blCcSource: blOf(3*blHour, "sha256:cc"), blPi: blOf(5*blMinuteMs, "sha256:pi"),
	}
	world := blWorld(manifest, map[string]string{blCcSource: blSaid, blPi: blSaid})

	blHousekeep(t, world, blCcLive)

	// the live session is left by its file name, the other one is still inside the quiet period
	g.Expect(world.Names()).To(Equal([]string{"ingest"}))
	g.Expect(world.Requests).To(BeEmpty())
}

func Test793d28b1_ASourceThatGrewSinceItWasMarkedIsCapturedAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blOneQuiet()
	world.Files[blMarksFile] = blMarksJSON(map[string]string{blPi: "sha256:old"}, map[string]float64{})

	blHousekeep(t, world, "")

	g.Expect(world.Names()).To(Equal([]string{"ingest", "query"}))
	g.Expect(world.Requests).To(HaveLen(1))
	g.Expect(blMarksOf(t, world).Sources[blPi]).To(Equal(blHashA))
}

func Test793d28b1_ASourceWhoseHashTheMarkHoldsIsNotCapturedAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blOneQuiet()
	world.Files[blMarksFile] = blMarksJSON(map[string]string{blPi: blHashA}, map[string]float64{})

	blHousekeep(t, world, "")

	g.Expect(world.Names()).To(Equal([]string{"ingest"}))
	g.Expect(world.Requests).To(BeEmpty())
}

func Test793d28b1_AnIndexThatHoldsASessionAfterTheCutoffIsNotReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	manifest := blManifest{blPi: blOf(10*blDayMs, "sha256:stale"), blPiTwo: blOf(2*blHour, blHashA)}
	world := blWorld(manifest, map[string]string{blPi: blSaid, blPiTwo: blSaid})

	blHousekeep(t, world, "")

	g.Expect(world.Warnings).To(BeEmpty())
}

func Test793d28b1_AnIndexWhoseNewestSessionIsOlderThanTheCutoffIsReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	manifest := blManifest{blPi: blOf(10*blDayMs, "sha256:stale"), blPiTwo: blOf(9*blDayMs, "sha256:staler")}
	world := blWorld(manifest, map[string]string{blPi: blSaid, blPiTwo: blSaid})

	blHousekeep(t, world, "")

	// nothing is captured, and the user is told why instead of a backlog that silently finds nothing
	g.Expect(world.Requests).To(BeEmpty())
	g.Expect(world.Warnings).To(Equal([]string{
		skipped("backlog capture was held back: the index is not caught up", retried),
	}))
	g.Expect(strings.Join(world.Logs, "\n")).To(ContainSubstring("backlog: index not caught up"))
}

func Test793d28b1_AtMostOneSourceIsCapturedPerTurnTheMostRecentlyChangedFirst(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	manifest := blManifest{
		blPi: blOf(4*blHour, "sha256:older"), blPiTwo: blOf(2*blHour, "sha256:newer"),
	}
	world := blWorld(manifest, map[string]string{
		blPi:    blPiRows([2]string{"user", "older source says: use go vet"}, [2]string{"assistant", "ok"}),
		blPiTwo: blPiRows([2]string{"user", "newer source says: use targ"}, [2]string{"assistant", "ok"}),
	})

	blHousekeep(t, world, "")

	g.Expect(world.Requests).To(HaveLen(1))
	g.Expect(world.Requests[0].Prompt).To(ContainSubstring("newer source says"))
	g.Expect(blMarksOf(t, world).Sources).To(HaveLen(1))
	g.Expect(blMarksOf(t, world).Sources).To(HaveKey(blPiTwo))

	blHousekeep(t, world, "")

	g.Expect(world.Requests).To(HaveLen(2))
	g.Expect(world.Requests[1].Prompt).To(ContainSubstring("older source says"))
	g.Expect(blMarksOf(t, world).Sources).To(HaveLen(2))
}

func Test793d28b1_AtMostTwentySourcesAreLookedAtInOneTurn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	manifest := make(blManifest, blSourcesInTurn+1)

	for index := range blSourcesInTurn + 1 {
		manifest[fmt.Sprintf("/home/.pi/agent/sessions/p/s%02d.jsonl", index)] = blOf(int64(2*blHour+index), blHashA)
	}

	world := blWorld(manifest, nil)

	blHousekeep(t, world, "")

	logged := 0

	for _, line := range world.Logs {
		if strings.Contains(line, "backlog: ") {
			logged++
		}
	}

	g.Expect(logged).To(Equal(blSourcesInTurn))
}

func Test793d28b1_NoManifestIsAnEmptyBacklogNotAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Clock = time.UnixMilli(blNowMs)
	world.ChunksDir = "/data/engram/chunks"

	blHousekeep(t, world, "")

	g.Expect(world.Names()).To(Equal([]string{"ingest"}))
	g.Expect(world.Warnings).To(BeEmpty())
}

func Test793d28b1_OneQuietSourceIsCapturedThenMarked(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blOneQuiet()
	world.Model = Replies(map[string]string{"capture": LessonJSON})

	blHousekeep(t, world, "")

	g.Expect(world.Names()).To(Equal([]string{"ingest", "query", "learn"}))
	g.Expect(world.Commands[0].Argv).To(Equal([]string{"engram", "ingest", "--auto"}))
	g.Expect(world.Requests).To(HaveLen(1))
	g.Expect(world.Requests[0].Model).To(Equal("haiku"))
	g.Expect(world.Requests[0].Prompt).To(ContainSubstring("User: no, use targ, not go test directly"))
	g.Expect(world.Requests[0].Prompt).To(ContainSubstring("Agent: Running go test"))
	g.Expect(blMarksOf(t, world).Sources[blPi]).To(Equal(blHashA))
}

func Test793d28b1_TheCutoffDoesNotRollWhateverTheIdleTime(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	manifest := blManifest{
		blPi: blOf(3*blDayMs, "sha256:three-days"), blPiTwo: blOf(8*blDayMs, "sha256:eight-days"),
	}
	history := blPiRows([2]string{"user", "history says: use go vet"}, [2]string{"assistant", "ok"})
	world := blWorld(manifest, map[string]string{blPi: blSaid, blPiTwo: history})
	world.Model = func(string, auto.Request) map[string]any { return Unanswered("api-error") }

	blHousekeep(t, world, "")

	// the capture failed, so the source holds no mark; 30 days pass with no session
	g.Expect(blMarksOf(t, world).Sources).NotTo(HaveKey(blPi))

	before := len(world.Requests)
	world.Clock = world.Clock.Add(30 * blDayMs * time.Millisecond)

	blHousekeep(t, world, "")

	g.Expect(len(world.Requests)).To(BeNumerically(">", before))
	g.Expect(blMarksOf(t, world).Cutoff).To(HaveValue(BeEquivalentTo(blNowMs - 7*blDayMs)))
	g.Expect(strings.Join(blPromptsOf(world), "\n")).NotTo(ContainSubstring("history says"))
}

func Test793d28b1_TheCutoffIsAFixedPointARunRecordsAndNotAWindow(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	manifest := blManifest{
		blPi: blOf(8*blDayMs, "sha256:before"), blPiTwo: blOf(6*blDayMs, "sha256:after"),
	}
	after := blPiRows([2]string{"user", "after the cut-off says: use targ"}, [2]string{"assistant", "ok"})
	world := blWorld(manifest, map[string]string{blPi: blSaid, blPiTwo: after})

	blHousekeep(t, world, "")

	marks := blMarksOf(t, world)
	g.Expect(marks.Cutoff).NotTo(BeNil())
	g.Expect(marks.Installed).NotTo(BeNil())

	if marks.Cutoff == nil || marks.Installed == nil {
		return
	}

	g.Expect(*marks.Cutoff).To(BeEquivalentTo(blNowMs - 7*blDayMs))
	g.Expect(*marks.Installed).To(BeEquivalentTo(blNowMs))
	g.Expect(world.Requests).To(HaveLen(1))
	g.Expect(world.Requests[0].Prompt).To(ContainSubstring("after the cut-off says"))

	blHousekeep(t, world, "")

	g.Expect(world.Requests).To(HaveLen(1))
	g.Expect(blMarksOf(t, world).Sources).To(HaveKey(blPiTwo))
	g.Expect(blMarksOf(t, world).Sources).NotTo(HaveKey(blPi))
}

func Test793d28b1_TheCutoffIsRecordedOnceEvenByAFirstRunWithNothingToCapture(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blWorld(blManifest{}, nil)

	blHousekeep(t, world, "")

	g.Expect(blMarksOf(t, world).Cutoff).To(HaveValue(BeEquivalentTo(blNowMs - 7*blDayMs)))

	world.Files[blMarksFile] = `{"sources":{},"turns":{},"cutoff":123,"installed":456}`
	world.Clock = world.Clock.Add(blDayMs * time.Millisecond)

	blHousekeep(t, world, "")

	marks := blMarksOf(t, world)
	g.Expect(marks.Cutoff).To(HaveValue(BeEquivalentTo(123)))
	g.Expect(marks.Installed).To(HaveValue(BeEquivalentTo(456)))
}

func Test793d28b1_WhatIsNotASessionTranscriptIsNotASource(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	manifest := blManifest{
		"/home/.claude/projects/p/9f1e2d3c/subagents/agent-a1.jsonl": blOf(2*blHour, "sha256:1"),
		"/home/work/NOTES.md": blOf(2*blHour, "sha256:2"),
		blPi:                  blOf(2*blHour, "sha256:3", "/elsewhere.jsonl"),
	}
	world := blWorld(manifest, nil)

	blHousekeep(t, world, "")

	g.Expect(world.Names()).To(Equal([]string{"ingest"}))
}

func Test793d28b2_ALessonBeyondTheWindowsOfOneTurnIsNotLostTheSourceIsMarkedWhenAllAreJudged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blWorld(blManifest{blPi: blOf(2*blHour, blHashA)}, map[string]string{blPi: blLongRows()})

	blHousekeep(t, world, "")

	g.Expect(blMarksOf(t, world).Sources).NotTo(HaveKey(blPi))

	first := len(world.Requests)

	blHousekeep(t, world, "")

	g.Expect(blMarksOf(t, world).Sources[blPi]).To(Equal(blHashA))

	prompts := blPromptsOf(world)
	g.Expect(len(prompts)).To(BeNumerically(">", first))
	g.Expect(blUniqueCount(prompts)).To(Equal(len(prompts)))
	g.Expect(strings.Join(prompts, "\n")).To(ContainSubstring("message 0 "))

	// and a third turn finds nothing left
	blHousekeep(t, world, "")

	g.Expect(world.Requests).To(HaveLen(len(prompts)))
}

func Test793d28b2_ALongTranscriptIsJudgedInOverlappingWindowsTheEndFirst(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blWorld(blManifest{blPi: blOf(2*blHour, blHashA)}, map[string]string{blPi: blLongRows()})

	blHousekeep(t, world, "")

	g.Expect(world.Requests).To(HaveLen(blWindowsInTurn))

	if len(world.Requests) != blWindowsInTurn {
		return
	}

	g.Expect(world.Requests[0].Prompt).To(ContainSubstring("reply 39 "))

	oldest := blRegexpFirst(`(?m)^(?:User|Agent): (message|reply) (\d+)`, world.Requests[0].Prompt)
	g.Expect(oldest).To(HaveLen(3))

	if len(oldest) == 3 {
		g.Expect(world.Requests[1].Prompt).To(ContainSubstring(oldest[1] + " " + oldest[2] + " "))
	}
}

func Test793d28b2_ASourceOverThePlatformReadLimitIsReadFromItsTail(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	head := blPiRows([2]string{"user", "HEAD-ONLY says: use go vet"}, [2]string{"assistant", "ok"})
	big := head + "\n" + strings.Repeat(`{"type":"noise"}`+"\n", blBigPadding) + blSaid
	world := blWorld(blManifest{blPi: blOf(2*blHour, blHashA)}, map[string]string{blPi: big})

	blHousekeep(t, world, "")

	g.Expect(len(big)).To(BeNumerically(">", 4<<20))
	g.Expect(world.Requests).To(HaveLen(1))
	g.Expect(strings.Join(blPromptsOf(world), "\n")).NotTo(ContainSubstring("HEAD-ONLY"))
	g.Expect(strings.Join(blPromptsOf(world), "\n")).To(ContainSubstring("no, use targ, not go test directly"))
	g.Expect(blMarksOf(t, world).Sources[blPi]).To(Equal(blHashA))
}

func Test793d28b2_ASourceThatGrewWhileItsWindowsWereConsumedStartsAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blWorld(blManifest{blPi: blOf(2*blHour, blHashA)}, map[string]string{blPi: blLongRows()})

	blHousekeep(t, world, "")

	world.Files[blManifestFile] = string(asJSON(blManifest{blPi: blOf(2*blHour, "sha256:bbb")}))
	before := len(world.Requests)

	blHousekeep(t, world, "")

	g.Expect(len(world.Requests)).To(BeNumerically(">", before))

	if len(world.Requests) > before {
		g.Expect(world.Requests[before].Prompt).To(ContainSubstring("reply 39 "))
	}

	g.Expect(blMarksOf(t, world).Sources).NotTo(HaveKey(blPi))
}

func Test793d28b2_ASourceWithNoMessageIsMarkedAndNotReadAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blWorld(blManifest{blPi: blOf(2*blHour, blHashA)}, map[string]string{blPi: blPiRows()})

	blHousekeep(t, world, "")

	g.Expect(world.Names()).To(Equal([]string{"ingest"}))
	g.Expect(blMarksOf(t, world).Sources[blPi]).To(Equal(blHashA))

	// were it read again it would now fail to read, and the failure would be reported
	world.Missing[blPi] = true
	world.Logs = nil

	blHousekeep(t, world, "")

	g.Expect(world.Logs).To(BeEmpty())
	g.Expect(world.Warnings).To(BeEmpty())
}

func Test793d28b3_ACaptureThatEndsInAWarningLeavesNoMark(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blOneQuiet()
	world.Model = func(string, auto.Request) map[string]any { return Unanswered("api-error") }

	blHousekeep(t, world, "")

	g.Expect(world.Names()).To(Equal([]string{"ingest", "query"}))
	g.Expect(world.Warnings).To(HaveLen(1))
	g.Expect(blMarksOf(t, world).Sources).To(BeEmpty())
}

func Test793d28b3_ALessonANearestNoteStatesReinforcesItAndTheSourceIsMarked(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blOneQuiet()
	found := "version: 1\nphrases: []\nitems:\n  - path: 7.2026-10-12.use-targ.md\n    kind: feedback\n    content: |+\n" +
		"      ---\n      situation: running Go tests in this repo\n      ---\n\n      Lesson learned: use targ.\nclusters:\n"
	world.Process["query"] = func([]string) (auto.Output, error) { return auto.Output{Stdout: found}, nil }
	covered := strings.Replace(LessonJSON, `"covered":null`, `"covered":1`, 1)
	world.Model = Replies(map[string]string{"capture": covered})

	blHousekeep(t, world, "")

	g.Expect(world.Find("learn")).To(BeEmpty())
	amends := world.Find("amend")
	g.Expect(amends).To(HaveLen(1))

	if len(amends) != 1 {
		return
	}

	g.Expect(amends[0].Argv).To(Equal([]string{"engram", "amend", "--target", "7.2026-10-12.use-targ.md", "--activate"}))
	g.Expect(blMarksOf(t, world).Sources[blPi]).To(Equal(blHashA))
}

func Test793d28b3_ASessionWhoseTurnsTheSystemCapturedIsNotCapturedAgainAsBacklog(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blWorld(blManifest{blCcSource: blOf(3*blHour, "sha256:cc")}, map[string]string{blCcSource: blSaid})

	// the turn capture of the session leaves a mark by the file name
	world.Do(t, map[string]any{
		"kind": "capture", "sessionId": "9f1e2d3c", "turn": 3, "transcript": blCcLive, "answer": "Four.",
		"messages": blTurnMessages(),
	})

	g.Expect(blMarksOf(t, world).Turns["9f1e2d3c.jsonl"]).To(BeEquivalentTo(blNowMs))

	// another live session scans the manifest later: the ended session's turns are marked captured
	world.Clock = world.Clock.Add(4 * blHour * time.Millisecond)
	world.Commands = nil
	world.Requests = nil

	blHousekeep(t, world, "/private/home/.claude/projects/p/7a7a7a7a.jsonl")

	g.Expect(world.Names()).To(Equal([]string{"ingest"}))
	g.Expect(world.Requests).To(BeEmpty())
}

func Test793d28b3_ASourceThatCannotBeReadDoesNotUseUpTheTurnAndIsNotReadAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	manifest := blManifest{blPi: blOf(2*blHour, blHashA), blPiTwo: blOf(4*blHour, "sha256:bbb")}
	world := blWorld(manifest, map[string]string{blPiTwo: blSaid})

	blHousekeep(t, world, "")

	g.Expect(world.Requests).To(HaveLen(1))
	g.Expect(blMarksOf(t, world).Sources).To(HaveLen(1))
	g.Expect(blMarksOf(t, world).Sources).To(HaveKey(blPiTwo))
	g.Expect(strings.Join(world.Logs, "\n")).To(ContainSubstring("backlog: " + blPi))

	// the file is there now, but the source that failed to read is not tried again by this session
	world.Files[blPi] = blSaid

	blHousekeep(t, world, "")

	g.Expect(world.Requests).To(HaveLen(1))
}

func Test793d28b3_ASourceThatCannotBeReadIsSkippedWithoutAWrite(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blWorld(blManifest{blPi: blOf(2*blHour, blHashA)}, nil)

	blHousekeep(t, world, "")

	g.Expect(world.Names()).To(Equal([]string{"ingest"}))
	// the only write is the first run's cut-off: no source is marked
	g.Expect(world.Writes).To(Equal([]string{blMarksFile}))
	g.Expect(blMarksOf(t, world).Sources).To(BeEmpty())
}

func Test793d28b3_ATurnCaptureOfASessionWithNoTranscriptMarksNoTurn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blWorld(blManifest{}, nil)

	world.Do(t, map[string]any{
		"kind": "capture", "sessionId": "s1", "turn": 1, "answer": "Four.",
		"messages": blTurnMessages(),
	})

	g.Expect(world.Writes).To(BeEmpty())
}

func Test793d28b3_ATurnKilledSoonAfterTheLastCapturedTurnIsStillCaptured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	turns := map[string]float64{"9f1e2d3c.jsonl": blNowMs - blHour}
	world := blWorld(blManifest{blCcSource: blOf(blHour-blMinuteMs, "sha256:cc")}, map[string]string{
		blCcSource: blSaid, blMarksFile: blMarksJSON(map[string]string{}, turns),
	})

	blHousekeep(t, world, "")

	g.Expect(world.Names()).To(Equal([]string{"ingest", "query"}))
}

func Test793d28b3_TheClosingRowsAHarnessWritesAfterTheLastCapturedTurnAreNotATurn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	turns := map[string]float64{"9f1e2d3c.jsonl": blNowMs - blHour}
	world := blWorld(blManifest{blCcSource: blOf(blHour-3000, "sha256:cc")}, map[string]string{
		blCcSource: blSaid, blMarksFile: blMarksJSON(map[string]string{}, turns),
	})

	blHousekeep(t, world, "")

	g.Expect(world.Names()).To(Equal([]string{"ingest"}))
}

func Test793d28b3_TheNoteIsSourcedFromTheTranscriptAndThePositionInIt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := blOneQuiet()
	world.Model = Replies(map[string]string{"capture": LessonJSON})

	blHousekeep(t, world, "")

	learns := world.Find("learn")
	g.Expect(learns).To(HaveLen(1))

	if len(learns) != 1 {
		return
	}

	g.Expect(learns[0].Argv).To(ContainElement(
		"--source=captured from the conversation; written automatically (source " + blPi + ", position message 4)"))
	g.Expect(learns[0].Argv).To(ContainElement("--action=use targ test"))
}

// unexported constants.
const (
	blBigPadding    = 270_000
	blCcLive        = "/private/home/.claude/projects/p/9f1e2d3c.jsonl"
	blCcSource      = "/home/.claude/projects/p/9f1e2d3c.jsonl"
	blChunksDir     = "/data/engram/chunks"
	blDayMs         = 24 * blHour
	blHashA         = "sha256:aaa"
	blHour          = 3_600_000
	blLongPairs     = 40
	blManifestFile  = blChunksDir + "/manifest.json"
	blMarksFile     = blChunksDir + "/backlog-marks.json"
	blMinuteMs      = 60_000
	blNanosPerMilli = 1_000_000
	blNowMs         = 1_800_000_000_000
	blPi            = "/home/.pi/agent/sessions/p/2026-10-12_pi1.jsonl"
	blPiTwo         = "/home/.pi/agent/sessions/p/2026-10-11_pi2.jsonl"
	blSourcesInTurn = 20
	blWindowsInTurn = 6
)

// unexported variables.
var (
	blSaid = blPiRows(
		[2]string{"user", "Run the tests."}, [2]string{"assistant", "Running go test ./..."},
		[2]string{"user", "no, use targ, not go test directly"}, [2]string{"assistant", "Understood, I will use targ."},
	)
)

type blEntry map[string]any

type blManifest map[string]blEntry

type blMarksView struct {
	Sources   map[string]string  `json:"sources"`
	Turns     map[string]float64 `json:"turns"`
	Cutoff    *float64           `json:"cutoff"`
	Installed *float64           `json:"installed"`
}

func blHousekeep(t *testing.T, world *World, live string) {
	t.Helper()

	job := map[string]any{"kind": "housekeeping", "turnEnded": true}
	if live != "" {
		job["transcript"] = live
	}

	world.Do(t, job)
}

func blLongRows() string {
	turns := make([][2]string, 0, 2*blLongPairs)
	for index := range blLongPairs {
		filler := strings.Repeat("word ", 150)
		turns = append(turns,
			[2]string{"user", fmt.Sprintf("message %d %s", index, filler)},
			[2]string{"assistant", fmt.Sprintf("reply %d %s", index, filler)})
	}

	return blPiRows(turns...)
}

func blMarksJSON(sources map[string]string, turns map[string]float64) string {
	return string(asJSON(map[string]any{"sources": sources, "turns": turns}))
}

func blMarksOf(t *testing.T, world *World) blMarksView {
	t.Helper()

	var view blMarksView
	err := json.Unmarshal([]byte(world.Files[blMarksFile]), &view)
	if err != nil {
		t.Fatalf("the marks cannot be read: %v", err)
	}

	return view
}

// blOf is a manifest entry of a source that changed ageMs before the test's now.
func blOf(ageMs int64, hash string, extra ...string) blEntry {
	entry := blEntry{"mtime_unix_nano": (blNowMs - ageMs) * blNanosPerMilli, "size": 100, "file_hash": hash}
	for _, duplicate := range extra {
		entry["duplicate_of"] = duplicate
	}

	return entry
}

func blOneQuiet() *World {
	return blWorld(blManifest{blPi: blOf(2*blHour, blHashA)}, map[string]string{blPi: blSaid})
}

// blPiRows is a Pi session file: rows of type message with a role.
func blPiRows(turns ...[2]string) string {
	rows := make([]string, 0, len(turns)+1)
	rows = append(rows, `{"type":"session","version":3,"id":"p1"}`)

	for _, turn := range turns {
		rows = append(rows, string(asJSON(map[string]any{"type": "message", "message": map[string]any{
			"role": turn[0], "content": []map[string]string{{"type": "text", "text": turn[1]}},
		}})))
	}

	return strings.Join(rows, "\n")
}

func blPromptsOf(world *World) []string {
	prompts := make([]string, 0, len(world.Requests))
	for _, request := range world.Requests {
		prompts = append(prompts, request.Prompt)
	}

	return prompts
}

func blRegexpFirst(pattern, text string) []string {
	return regexp.MustCompile(pattern).FindStringSubmatch(text)
}

func blTurnMessages() []map[string]any {
	return []map[string]any{
		{"role": "user", "text": "What is two plus two?"}, {"role": "assistant", "text": "Four."},
	}
}

func blUniqueCount(values []string) int {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		seen[value] = true
	}

	return len(seen)
}

// blWorld is a world whose chunks directory holds the manifest.
func blWorld(manifest blManifest, files map[string]string) *World {
	world := NewWorld()
	world.Clock = time.UnixMilli(blNowMs)
	world.ChunksDir = "/data/engram/chunks"
	world.Files[blManifestFile] = string(asJSON(manifest))

	stdmaps.Copy(world.Files, files)

	return world
}

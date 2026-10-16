package auto_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

// The dogfood defect of 2026-10-16: the housekeeping of a vault whose manifest holds 229k entries printed a reply of
// tens of megabytes, because the record of its effects (the `state` it sends back) held the manifest as it read it;
// the mod parsed a document cut short. A job's reply no longer holds what a file it read held.
func Test793d36b1_AHousekeepingOverALargeManifestPrintsRepliesUnderTheCap(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	manifest := blManifest{blPi: blOf(2*blHour, blHashA)}

	for index := range 20000 {
		name := fmt.Sprintf("/home/u/.claude/projects/p/doc-%d.md", index)
		manifest[name] = blOf(30*blDayMs, "sha256:"+strings.Repeat("c", 64))
	}

	world := blWorld(manifest, map[string]string{blPi: blSaid})

	blHousekeep(t, world, "")

	g.Expect(len(world.Files[blManifestFile])).To(BeNumerically(">", 2*replyCap))
	g.Expect(world.Rounds).To(BeNumerically(">=", 2), "the backlog source was not judged")
	g.Expect(world.Requests).To(HaveLen(1), "the backlog source was not judged")

	for round, size := range world.ReplySizes {
		g.Expect(size).To(BeNumerically("<=", replyCap), "the reply of round %d", round)
	}
}

func Test793d36b1_AJobWhoseRecordPassesTheCapEndsWithAWarningAndSendsNoState(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Clock = time.UnixMilli(1_700_000_000_000)
	big := strings.Repeat("a note body that is far too long. ", 20000)
	world.Process["query"] = func([]string) (auto.Output, error) {
		return auto.Output{Stdout: strings.Replace(NoteQuery, "run go build first (SENTINEL-N)", big, 1)}, nil
	}

	first := world.round(t, map[string]any{
		"kind": "lookup", "moment": "prompt", "step": "prompt lookup", "text": "is it done?",
	})

	g.Expect(world.ReplySizes[0]).To(BeNumerically("<=", replyCap))
	g.Expect(first.Request).To(BeNil())
	g.Expect(first.State).To(BeEmpty())
	g.Expect(first.Done).NotTo(BeNil())
	g.Expect(first.Truncated).To(BeTrue())
	g.Expect(first.Warnings).To(HaveLen(1))
	g.Expect(first.Warnings[0]).To(And(
		HavePrefix("Automatic memory skipped: prompt lookup failed: the reply of "),
		ContainSubstring(" bytes passes the limit of 262144 bytes; "),
	))
	g.Expect(world.Written[world.LogPath]).To(ContainSubstring(`operation="prompt lookup" class=error`))
}

func Test793d36b1_ALogLineThatIsHugeIsCutAndFewLinesAreNotTruncated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()

	few := auto.EncodeWithLogs(world.Deps(), 3, "short")

	var reply map[string]any

	g.Expect(json.Unmarshal(few, &reply)).To(Succeed())
	g.Expect(reply).NotTo(HaveKey("truncated"))
	g.Expect(reply["log"]).To(HaveLen(3))
	g.Expect(world.Written).NotTo(HaveKey(DebugLog))

	huge := auto.EncodeWithLogs(world.Deps(), 3, strings.Repeat("x", 3*replyCap))

	g.Expect(len(huge)).To(BeNumerically("<=", replyCap))
	g.Expect(json.Unmarshal(huge, &reply)).To(Succeed())
}

func Test793d36b1_AResultThatPassesTheCapIsReplacedByTheEndOfTheJobWithTheStateItWasGiven(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	input := asJSON(map[string]any{
		"kind": "summary", "ledger": map[string][]string{"main": {"1.n.md"}},
		"work": map[string]any{"skipped": []string{"x"}},
	})

	output := auto.EncodeWithBlock(world.Deps(), input, strings.Repeat("b", 2*replyCap))

	var reply auto.Response

	g.Expect(json.Unmarshal(output, &reply)).To(Succeed())
	g.Expect(len(output)).To(BeNumerically("<=", replyCap))
	g.Expect(reply.Truncated).To(BeTrue())
	g.Expect(reply.Done).NotTo(BeNil())

	if reply.Done == nil {
		return
	}

	g.Expect(reply.Done.Block).To(BeEmpty())
	g.Expect(string(reply.Done.Ledger)).To(Equal(`{"main":["1.n.md"]}`))
	g.Expect(string(reply.Done.Work)).To(Equal(`{"skipped":["x"]}`))
	g.Expect(reply.Warnings).To(HaveLen(1))
	g.Expect(reply.Warnings[0]).To(HavePrefix("Automatic memory skipped: summary failed: the reply of "))
}

func Test793d36b1_AnEngineWithNoFailureRecordLeavesTheLinesItCannotCarryOut(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.LogPath = "auto.log"

	output := auto.EncodeWithLogs(world.Deps(), 100, "a line")

	var reply map[string]any

	g.Expect(json.Unmarshal(output, &reply)).To(Succeed())

	lines, _ := reply["log"].([]any)
	g.Expect(lines).To(HaveLen(replyLogMax))
	g.Expect(lines[replyLogMax-1]).To(Equal("engram-memory: 81 more log line(s) left out"))
	g.Expect(world.Written).To(BeEmpty())
}

func Test793d36b1_TenThousandLogLinesMakeAReplyUnderTheCapThatIsStillJSON(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()

	output := auto.EncodeWithLogs(world.Deps(), 10000, "ingest: /home/u/.claude/projects/p/some-session-file.jsonl")

	var reply map[string]any

	g.Expect(json.Unmarshal(output, &reply)).To(Succeed())
	g.Expect(len(output)).To(BeNumerically("<=", replyCap))
	g.Expect(reply["truncated"]).To(BeTrue())

	lines, _ := reply["log"].([]any)
	g.Expect(len(lines)).To(BeNumerically("<=", replyLogMax))
	g.Expect(reply).To(HaveKey("counts"))

	// the lines the reply does not carry are in the debug file, whole
	g.Expect(strings.Count(world.Written[DebugLog], "\n")).To(Equal(10000 - replyLogMax + 1))
	g.Expect(lines[len(lines)-1]).To(ContainSubstring("~/.local/share/engram/auto-debug.log"))
}

func Test793d36b1_TheStepAnOverflowIsBlamedOnIsTheJobsStepInTheTermsOfTheUser(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ name, job, step string }{
		{"a step the job names", `{"kind":"lookup","step":"prompt lookup"}`, "prompt lookup"},
		{"a part of the housekeeping", `{"kind":"housekeeping","part":"curation"}`, "curation"},
		{"the whole housekeeping", `{"kind":"housekeeping"}`, "indexing, curation and backlog capture"},
		{"a summary", `{"kind":"summary"}`, "summary"},
		{"a capture", `{"kind":"capture"}`, "capture"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(auto.StepOf([]byte(test.job))).To(Equal(test.step))
		})
	}
}

// unexported constants.
const (
	replyCap    = 256 * 1024
	replyLogMax = 20
)

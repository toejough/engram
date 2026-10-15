package auto_test

import (
	"errors"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test793d36b3_AFailedStepIsContainedAndTheUserIsWarnedOfItsClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		process func([]string) (auto.Output, error)
		reason  string
	}{
		{
			"the vault cannot be read: unavailable",
			func([]string) (auto.Output, error) {
				return auto.Output{Stderr: "query: scan: open /vault: permission denied"}, errors.New("exit status 1")
			},
			"the vault is unavailable",
		},
		{
			"the engram program cannot start: unavailable",
			func([]string) (auto.Output, error) {
				return auto.Output{}, errors.New("engram failed to start: ENOENT: no such file or directory")
			},
			"the vault is unavailable",
		},
		{
			"a command that passes its deadline: timed out",
			func([]string) (auto.Output, error) {
				return auto.Output{}, errors.New("engram aborted: still running after 25000ms")
			},
			"the System timed out",
		},
		{
			"a command that fails: of an error",
			func([]string) (auto.Output, error) {
				return auto.Output{Stderr: "engram: injected failure"}, errNonZero
			},
			"of an error",
		},
		{
			"output that cannot be parsed is an error, not an empty result",
			func([]string) (auto.Output, error) { return auto.Output{Stdout: "this is not a query result"}, nil },
			"of an error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			world := NewWorld()
			world.Process["query"] = test.process

			done := failureLookup(t, world)

			g.Expect(done.Block).To(BeEmpty())
			g.Expect(done.Notes).To(BeEmpty())
			g.Expect(world.Requests).To(BeEmpty())
			g.Expect(world.Warnings).To(Equal([]string{failureWarning(test.reason, failureStep)}))
		})
	}
}

func Test793d36b3_AFailingBacklogStepIsWarnedAboutNamingTheStep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.ChunksDir = "/chunks"
	world.Files["/chunks/manifest.json"] = "[1]"

	world.Do(t, map[string]any{"kind": "housekeeping", "turnEnded": true})

	g.Expect(world.Warnings).To(Equal([]string{failureWarning("of an error", "the backlog capture")}))
}

func Test793d36b3_AFailingIndexAndAFailingScanEachWarnNamingTheStepAndTheQueueGoesOn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Process["ingest"] = func([]string) (auto.Output, error) {
		return auto.Output{}, errors.New("ingest aborted: still running after 300000ms")
	}
	world.Files["/vault/"+curateOffer] = curateOpenFile
	world.Missing["/vault/walk"] = true

	world.Do(t, map[string]any{"kind": "housekeeping"})

	g.Expect(world.Warnings).To(Equal([]string{
		failureWarning("the System timed out", "indexing"),
		failureWarning("the vault is unavailable", "curation"),
	}))
	g.Expect(world.Names()).To(Equal([]string{"ingest"}))
}

func Test793d36b3_AFailingJobDoesNotStopTheNextJob(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	calls := 0
	world.Process["query"] = func([]string) (auto.Output, error) {
		calls++
		if calls == 1 {
			return auto.Output{Stderr: "boom"}, errNonZero
		}

		return auto.Output{Stdout: NoteQuery}, nil
	}
	world.Model = Replies(map[string]string{"relevance": "[1]"})

	first := failureLookup(t, world)
	second := failureLookup(t, world)

	g.Expect(first.Block).To(BeEmpty())
	g.Expect(second.Block).To(ContainSubstring(failureNotesIn[:1]))
	g.Expect(world.Warnings).To(HaveLen(1))
	g.Expect(world.Find("query")).To(HaveLen(2))
}

func Test793d36b3_ALookupThatFindsNothingIsNotAFailure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()

	done := failureLookup(t, world)

	g.Expect(done.Block).To(BeEmpty())
	g.Expect(world.Warnings).To(BeEmpty())
}

func Test793d36b3_ARelevanceJudgementThatIsCutIsATimeoutAndOneTheProviderRefusedIsAnErrorNeitherDelivers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reply  string
		reason string
	}{
		{"aborted", "the System timed out"},
		{"the provider refused the request", "of an error"},
	}

	for _, test := range tests {
		t.Run(test.reply, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			world := NewWorld()
			world.Process["query"] = func([]string) (auto.Output, error) { return auto.Output{Stdout: NoteQuery}, nil }
			world.Model = func(string, auto.Request) map[string]any { return Unanswered(test.reply) }

			done := failureLookup(t, world)

			g.Expect(world.Requests).To(HaveLen(1))
			g.Expect(done.Block).To(BeEmpty())
			g.Expect(world.Warnings).To(Equal([]string{failureWarning(test.reason, failureStep)}))
			g.Expect(world.Counts).To(Equal(auto.Counts{}))
		})
	}
}

func Test793d36b3_AnIndexAnotherRunHoldsIsSkippedWithItsOwnWarningAndTheQueueGoesOn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Process["ingest"] = func([]string) (auto.Output, error) {
		return auto.Output{}, errors.New("ingest: acquiring manifest lock: another ingest holds the manifest lock; " +
			"skipped")
	}
	world.Files["/vault/"+curateOffer] = curateOpenFile
	world.Missing["/vault/walk"] = true

	world.Do(t, map[string]any{"kind": "housekeeping"})

	g.Expect(world.Warnings).To(Equal([]string{
		failureWarning("another indexing is running", "indexing"),
		failureWarning("the vault is unavailable", "curation"),
	}))
}

func Test793d36b3_AnUnparseableRelevanceJudgementIsAnErrorAndDeliversNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Process["query"] = func([]string) (auto.Output, error) { return auto.Output{Stdout: NoteQuery}, nil }
	world.Model = Replies(map[string]string{"relevance": "the first note is relevant"})

	done := failureLookup(t, world)

	g.Expect(done.Block).To(BeEmpty())
	g.Expect(world.Warnings).To(Equal([]string{failureWarning("of an error", failureStep)}))
}

func Test793d36b3_TheClassOfAFailureIsWhatTheTextOfTheCommandSays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		stderr string
		err    string
		reason string
	}{
		{"read-only file system", "write: read-only file system", "exit status 1", "the vault is unavailable"},
		{"input/output error", "read: input/output error", "exit status 1", "the vault is unavailable"},
		{"timed out", "context timed out", "exit status 1", "the System timed out"},
		{"anything else", "usage: engram query", "exit status 2", "of an error"},
		{
			"a long text is cut from its end before it is classified",
			strings.Repeat("x", 500) + "permission denied", "exit status 1", "of an error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			world := NewWorld()
			world.Process["query"] = func([]string) (auto.Output, error) {
				return auto.Output{Stderr: test.stderr}, errors.New(test.err)
			}

			failureLookup(t, world)

			g.Expect(world.Warnings).To(Equal([]string{failureWarning(test.reason, failureStep)}))
			g.Expect(strings.Join(world.Logs, "\n")).To(ContainSubstring("query failed: "))
		})
	}
}

func Test793d36b3_TheWarningIsInTheWarningsAndNeverInWhatTheJobDelivers(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Process["query"] = func([]string) (auto.Output, error) {
		return auto.Output{Stderr: "boom"}, errNonZero
	}

	done := failureLookup(t, world)

	g.Expect(world.Warnings).To(HaveLen(1))
	g.Expect(done.Block).NotTo(ContainSubstring("skipped"))
	g.Expect(done.Line).To(BeEmpty())
	g.Expect(world.Logs).NotTo(BeEmpty())
}

// unexported constants.
const (
	failureNotesIn = "1.2026-10-12.note-n.md"
	failurePrompt  = "I just finished renaming the package. Is it done?"
	failureStep    = "the prompt lookup"
)

// unexported variables.
var (
	errNonZero = errors.New("exit status 70")
)

// failureLookup runs the lookup of a prompt in a world.
func failureLookup(t *testing.T, world *World) auto.Done {
	t.Helper()

	return world.Do(t, map[string]any{"kind": "lookup", "moment": "prompt", "text": failurePrompt, "step": failureStep})
}

func failureWarning(reason, step string) string {
	return "Automatic memory was skipped because " + reason + " (" + step + ")"
}

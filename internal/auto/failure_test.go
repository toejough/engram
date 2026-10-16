package auto_test

import (
	"errors"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	"github.com/toejough/engram/internal/auto"
)

func Test793d36b3_AFailedStepIsContainedAndTheUserIsWarnedOfItsClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		process func([]string) (auto.Output, error)
		what    string
	}{
		{
			"the vault cannot be read: unavailable",
			func([]string) (auto.Output, error) {
				return auto.Output{Stderr: "query: scan: open /vault: permission denied; "}, errors.New("exit status 1")
			},
			"prompt lookup could not reach the vault: query: scan: open /vault: permission denied; exit status 1",
		},
		{
			"the engram program cannot start: unavailable",
			func([]string) (auto.Output, error) {
				return auto.Output{}, errors.New("engram failed to start: ENOENT: no such file or directory")
			},
			"prompt lookup could not reach the vault: engram failed to start: ENOENT: no such file or directory",
		},
		{
			"a command that passes its deadline: timed out",
			func([]string) (auto.Output, error) {
				return auto.Output{}, errors.New("engram aborted: still running after 25000ms")
			},
			"prompt lookup timed out: engram aborted: still running after 25000ms",
		},
		{
			"a command that fails: failed, with its own text",
			func([]string) (auto.Output, error) {
				return auto.Output{Stderr: "engram: injected failure; "}, errNonZero
			},
			"prompt lookup failed: engram: injected failure; exit status 70",
		},
		{
			"output that cannot be parsed is an error, not an empty result",
			func([]string) (auto.Output, error) { return auto.Output{Stdout: "this is not a query result"}, nil },
			"prompt lookup failed: ",
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
			g.Expect(world.Warnings).To(HaveLen(1))
			g.Expect(world.Warnings[0]).To(HavePrefix("Automatic memory skipped: " + test.what))
			g.Expect(world.Warnings[0]).To(HaveSuffix("; " + failureNext + " (details: " + failureLogShown + ")"))
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

	g.Expect(world.Warnings).To(HaveLen(1))
	g.Expect(world.Warnings[0]).To(HavePrefix("Automatic memory skipped: backlog capture failed: "))
	g.Expect(world.Warnings[0]).To(HaveSuffix("; " + retried + " (details: " + failureLogShown + ")"))
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

	g.Expect(world.Warnings).To(HaveLen(2))
	g.Expect(world.Warnings[0]).To(Equal(skipped(
		"indexing timed out: ingest aborted: still running after 300000ms", retried)))
	g.Expect(world.Warnings[1]).To(HavePrefix("Automatic memory skipped: curation could not reach the vault: "))
	g.Expect(world.Warnings[1]).To(HaveSuffix("; " + retried + " (details: " + failureLogShown + ")"))
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
		reply string
		what  string
	}{
		{"aborted", "prompt lookup timed out: "},
		{"the provider refused the request", "prompt lookup failed: "},
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
			g.Expect(world.Warnings).To(HaveLen(1))
			g.Expect(world.Warnings[0]).To(HavePrefix("Automatic memory skipped: " + test.what))
			g.Expect(world.Warnings[0]).To(ContainSubstring(test.reply))
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

	g.Expect(world.Warnings).To(HaveLen(2))
	g.Expect(world.Warnings[0]).To(Equal(skipped("indexing was already running in another session", retried)))
	g.Expect(world.Warnings[1]).To(HavePrefix("Automatic memory skipped: curation could not reach the vault: "))
}

func Test793d36b3_AnUnparseableRelevanceJudgementIsAnErrorAndDeliversNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Process["query"] = func([]string) (auto.Output, error) { return auto.Output{Stdout: NoteQuery}, nil }
	world.Model = Replies(map[string]string{"relevance": "the first note is relevant"})

	done := failureLookup(t, world)

	g.Expect(done.Block).To(BeEmpty())
	g.Expect(world.Warnings).To(HaveLen(1))
	g.Expect(world.Warnings[0]).To(HavePrefix("Automatic memory skipped: prompt lookup failed: "))
}

func Test793d36b3_TheClassOfAFailureIsWhatTheTextOfTheCommandSays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		stderr string
		err    string
		what   string
	}{
		{"read-only file system", "write: read-only file system", "exit status 1", "could not reach the vault"},
		{"input/output error", "read: input/output error", "exit status 1", "could not reach the vault"},
		{"timed out", "context timed out", "exit status 1", "timed out"},
		{"anything else", "usage: engram query", "exit status 2", "failed"},
		{
			"a long text is cut from its end before it is classified",
			strings.Repeat("x", 500) + "permission denied", "exit status 1", "failed",
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

			g.Expect(world.Warnings).To(HaveLen(1))
			g.Expect(world.Warnings[0]).To(HavePrefix("Automatic memory skipped: " + failureStep + " " + test.what))
			g.Expect(strings.Join(world.Logs, "\n")).To(ContainSubstring("query failed: "))
		})
	}
}

func Test793d36b3_EveryWarningNamesTheOperationTheReasonWhatHappensNextAndTheLog(t *testing.T) {
	t.Parallel()

	busyText := "ingest: acquiring manifest lock: another ingest holds the manifest lock; skipped"
	tests := []struct {
		name    string
		process string
		fail    string
		job     map[string]any
		want    string
	}{
		{
			"indexing, busy", "ingest", busyText, map[string]any{"kind": "housekeeping"},
			skipped("indexing was already running in another session", retried),
		},
		{
			"indexing, timeout", "ingest", "ingest aborted: still running after 300000ms",
			map[string]any{"kind": "housekeeping"},
			skipped("indexing timed out: ingest aborted: still running after 300000ms", retried),
		},
		{
			"indexing, unavailable", "ingest", "write: read-only file system", map[string]any{"kind": "housekeeping"},
			skipped("indexing could not reach the vault: write: read-only file system", retried),
		},
		{
			"indexing, error", "ingest", "exit status 70", map[string]any{"kind": "housekeeping"},
			skipped("indexing failed: exit status 70", retried),
		},
		{
			"a route lookup, error", "query", "exit status 70",
			map[string]any{"kind": "tool", "route": true, "args": "write Go tests", "step": "route lookup"},
			skipped("route lookup failed: exit status 70", "no notes delivered this time"),
		},
		{
			"a history lookup, timeout", "query", "engram aborted: still running after 25000ms",
			map[string]any{"kind": "deep", "step": "history lookup", "text": failurePrompt, "lastPrompt": failurePrompt},
			skipped("history lookup timed out: engram aborted: still running after 25000ms", "no notes delivered this time"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			world := NewWorld()
			world.Process[test.process] = func([]string) (auto.Output, error) {
				return auto.Output{}, errors.New(test.fail)
			}

			world.Do(t, test.job)

			g.Expect(world.Warnings).To(ContainElement(test.want))
			g.Expect(world.Warnings).NotTo(ContainElement(ContainSubstring("housekeeping")))
		})
	}
}

func Test793d36b3_TheReasonOfAWarningIsOneLineRedactedAndCutAt120Characters(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	secret := "ghp_" + strings.Repeat("a1B2", 8)
	world.Process["query"] = func([]string) (auto.Output, error) {
		return auto.Output{}, errors.New("first line with " + secret + "\nsecond line " + strings.Repeat("y", 300))
	}

	failureLookup(t, world)

	g.Expect(world.Warnings).To(HaveLen(1))
	g.Expect(world.Warnings[0]).NotTo(ContainSubstring(secret))
	g.Expect(world.Warnings[0]).NotTo(ContainSubstring("\n"))
	g.Expect(world.Warnings[0]).To(ContainSubstring("first line with [redacted] second line yyy"))
	g.Expect(world.Warnings[0]).To(ContainSubstring("yyy...; " + failureNext))

	reason := strings.TrimPrefix(strings.Split(world.Warnings[0], "; ")[0], "Automatic memory skipped: prompt lookup failed: ")
	g.Expect(len([]rune(reason))).To(BeNumerically("<=", 123))
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
	failureStep    = "prompt lookup"
	// the failure record as the warning shows it: the home directory is a tilde
	failureLogShown = "~/.local/share/engram/auto.log"
	failureNext     = "no notes delivered for this prompt"
	retried         = "nothing lost, retried next turn"
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

// warnedOf is a warning of an operation, with what happened starting as given, and the pointer to the log at its end.
func warnedOf(operation, what string) types.GomegaMatcher {
	return And(
		HavePrefix("Automatic memory skipped: "+operation+" "+what),
		HaveSuffix(" (details: "+failureLogShown+")"),
	)
}

// skipped is a warning as the user reads it: what happened, what happens next, and where the details are.
func skipped(what, next string) string {
	return "Automatic memory skipped: " + what + "; " + next + " (details: " + failureLogShown + ")"
}

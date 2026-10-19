package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

//go:generate impgen cli.LearnRun --dependency --import-path github.com/toejough/engram/internal/cli

func Test855d4i1a_Write_ARunbookLessonBecomesARunbookNote(t *testing.T) {
	t.Parallel()

	written := writeLearning(t, writeInput{reply: runbookLessonReply}, []cli.LearnArgs{{
		Type: "runbook", Slug: "release-the-project", Vault: "/vault", VaultName: "personal", Position: "top",
		Source:    "written automatically (session " + sessionID + ")",
		Situation: "releasing the project", DoneWhen: "the tag is pushed",
		Body: "1. run make ci 2. tag with git tag vX.Y.Z 3. push the tag with git push origin vX.Y.Z",
	}})

	NewWithT(t).Expect(written).To(ContainSubstring("/vault/release-the-project.md"),
		"the path of the note, as the learn path prints it")
}

func Test855d4i1b_Write_ALessonIsWrittenAsAContinuationOrASiblingOfASeenNote(t *testing.T) {
	t.Parallel()

	t.Run("a continuation and a sibling", func(t *testing.T) {
		t.Parallel()

		writeLearning(t, writeInput{
			reply: twoLessons,
			seen:  []string{"1.2026-10-17.zorblax-build.md", "2a.2026-10-17.zorblax-format.md"},
			placement: `[{"slug":"format-before-commit","position":"continuation","target":"1.2026-10-17.zorblax-build.md",` +
				`"supersedes":[]},{"slug":"staging-reset","position":"sibling","target":"2a.2026-10-17.zorblax-format.md",` +
				`"supersedes":[]}]`,
		}, []cli.LearnArgs{
			feedbackLearn("continuation", "1"),
			factLearn("sibling", "2a"),
		})
	})

	t.Run("a placement that follows the model's reasoning", func(t *testing.T) {
		t.Parallel()

		writeLearning(t, writeInput{
			reply: factLessonReply, seen: []string{"2a.2026-10-17.zorblax-format.md"},
			placement: "The lesson extends the thought of the note at the same level.\n\n" +
				`[{"slug":"staging-reset","position":"sibling","target":"2a.2026-10-17.zorblax-format.md",` +
				`"supersedes":[]}]`,
		}, []cli.LearnArgs{factLearn("sibling", "2a")})
	})

	t.Run("a sibling of a note at the top is written at the top", func(t *testing.T) {
		t.Parallel()

		writeLearning(t, writeInput{
			reply: factLessonReply, seen: []string{"1.2026-10-17.zorblax-build.md"},
			placement: `[{"slug":"staging-reset","position":"sibling","target":"1.2026-10-17.zorblax-build.md",` +
				`"supersedes":[]}]`,
		}, []cli.LearnArgs{factLearn("top", "")})
	})
}

func Test855d4i1c_Write_ALessonIsWrittenAtTheTopWithoutAPlacement(t *testing.T) {
	t.Parallel()

	writeLearning(t, writeInput{reply: factLessonReply}, []cli.LearnArgs{factLearn("top", "")})
}

func Test855d4i1d_Write_ATargetOrASupersededNoteTheCarryDoesNotNameIsIgnored(t *testing.T) {
	t.Parallel()

	writeLearning(t, writeInput{
		reply: factLessonReply, seen: []string{"1.2026-10-17.zorblax-build.md"},
		placement: `[{"slug":"staging-reset","position":"continuation","target":"2.2026-10-17.zorblax-other.md",` +
			`"supersedes":[{"note":"2.2026-10-17.zorblax-other.md","type":"updates","claim":"the old value"}]}]`,
	}, []cli.LearnArgs{factLearn("top", "")})
}

func Test855d4i1e_Write_ASupersessionTiesTheNotes(t *testing.T) {
	t.Parallel()

	args := factLearn("top", "")
	args.Supersedes = []string{"1.2026-10-17.zorblax-staging-reset|updates|the staging database is reset every Sunday"}

	writeLearning(t, writeInput{
		reply: factLessonReply, seen: []string{"1.2026-10-17.zorblax-staging-reset.md"},
		placement: `[{"slug":"staging-reset","position":"top","target":"","supersedes":[{` +
			`"note":"1.2026-10-17.zorblax-staging-reset.md","type":"updates",` +
			`"claim":"the staging database is reset every Sunday"}]}]`,
	}, []cli.LearnArgs{args})
}

func Test855d4i1f_Write_APlacementReplyThatIsNotAListIsIgnoredAndRecorded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := &recordedFiles{}

	writeLearning(t, writeInput{
		reply: factLessonReply, seen: []string{"1.2026-10-17.zorblax-staging-reset.md"},
		placement: "I would put it under the first note.", records: records,
	}, []cli.LearnArgs{factLearn("top", "")})

	g.Expect(records.kinds()).To(ConsistOf("written", "skipped"), "the note, and that the placement was not used")

	for _, record := range records.all() {
		if record["kind"] == "skipped" {
			g.Expect(record["reason"]).To(And(ContainSubstring("placement was not used"),
				ContainSubstring("not a list")), "the reason")
			g.Expect(record).NotTo(HaveKey("slug"), "the placement is no lesson")
		}
	}
}

func Test855d4i1g_Write_APlacementThatFailedIsRecordedAndTheLessonsAreWrittenAtTheTop(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	records := &recordedFiles{}

	writeLearning(t, writeInput{
		reply: factLessonReply, records: records,
		placementFailure: "engram auto place exited with 1: boom",
	}, []cli.LearnArgs{factLearn("top", "")})

	g.Expect(records.kinds()).To(ConsistOf("written", "skipped"), "the note, and that the placement was not used")

	for _, record := range records.all() {
		if record["kind"] == "skipped" {
			g.Expect(record["reason"]).To(And(ContainSubstring("placement was not used"),
				ContainSubstring("engram auto place exited with 1")), "the reason")
			g.Expect(record).NotTo(HaveKey("slug"), "the placement is no lesson")
		}
	}
}

// unexported constants.
const (
	factLessonReply = `[{"kind":"fact","slug":"staging-reset","situation":"testing on the staging database",` +
		`"covered":null,"subject":"the staging database","predicate":"is reset every","object":"Sunday night"}]`
	runbookLessonReply = `[{"kind":"runbook","slug":"release-the-project","situation":"releasing the project",` +
		`"covered":null,"done_when":"the tag is pushed","body":"1. run make ci 2. tag with git tag vX.Y.Z ` +
		`3. push the tag with git push origin vX.Y.Z"}]`
)

// writeInput is what the memory mod hands to the writing after a capture: the reply to the lessons request, the seen
// notes the placement's carry names, the reply to the placement request, and where the records go.
type writeInput struct {
	reply     string
	seen      []string
	placement string
	// placementFailure is why the placement could not be had, when it could not (855d2b).
	placementFailure string
	records          *recordedFiles
}

// factLearn is the write of the fact lesson of factLessonReply through the learn path, at a position of the order.
func factLearn(position, target string) cli.LearnArgs {
	return cli.LearnArgs{
		Type: "fact", Slug: "staging-reset", Vault: "/vault", VaultName: "personal", Position: position, Target: target,
		Source:    "written automatically (session " + sessionID + ")",
		Situation: "testing on the staging database", Subject: "the staging database", Predicate: "is reset every",
		Object: "Sunday night",
	}
}

// feedbackLearn is the write of the feedback lesson of twoLessons through the learn path, at a position of the order.
func feedbackLearn(position, target string) cli.LearnArgs {
	return cli.LearnArgs{
		Type: "feedback", Slug: "format-before-commit", Vault: "/vault", VaultName: "personal", Position: position,
		Target: target, Source: "written automatically (session " + sessionID + ")",
		Situation: "committing a Python file", Behavior: "left trailing whitespace", Impact: "the lint check failed",
		Action: "run the formatter before committing",
	}
}

// writeLearning runs the writing for an input against a mock learn path, expects the learn path to be given the
// arguments in order, and returns what the command printed.
func writeLearning(t *testing.T, input writeInput, expected []cli.LearnArgs) string {
	t.Helper()

	learn, learnCalls := MockLearnRun(t)
	records := input.records

	if records == nil {
		records = &recordedFiles{}
	}

	fields := map[string]any{"session": sessionID, "turn": turnID, "reply": input.reply}
	if input.placement != "" {
		fields["placement"] = input.placement
	}

	if input.placementFailure != "" {
		fields["placementFailure"] = input.placementFailure
	}

	if input.seen != nil {
		fields["carry"] = map[string]any{"seen": input.seen}
	}

	stdin, marshalErr := json.Marshal(fields)
	if marshalErr != nil {
		t.Fatalf("encoding the input: %v", marshalErr)
	}

	done := make(chan error, 1)

	var out bytes.Buffer

	go func() {
		args := cli.AutoWriteArgs{VaultPath: "/vault", VaultName: "personal", RecordsDir: "/data/auto"}

		done <- cli.RunAutoWrite(context.Background(), args,
			cli.AutoWriteDeps{
				Stdin: strings.NewReader(string(stdin)), Learn: learn, Now: func() time.Time { return recordedAt },
				WriteRecord: records.write,
			}, &out)
	}()

	go func() {
		for _, args := range expected {
			learnCalls.ArgsShould(Not(BeNil()), Equal(args)).Return("/vault/"+args.Slug+".md", nil)
		}
	}()

	select {
	case err := <-done:
		NewWithT(t).Expect(err).NotTo(HaveOccurred())
	case <-time.After(mockTimeout):
		t.Fatalf("the writing did not finish: the learn path was not given the arguments %+v in order", expected)
	}

	return out.String()
}

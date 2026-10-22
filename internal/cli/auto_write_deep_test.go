package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/auto"
	"github.com/toejough/engram/internal/cli"
)

func Test855d4j1b_WriteDeep_ALessonThatCorrectsANoteOfTheCarryTiesIt(t *testing.T) {
	t.Parallel()

	deep := deepWriting{carry: deepCarry(freezeNote, passageA), reply: deepReply(`[]`,
		`[`+freezeLesson(`["`+passageA.ID+`"]`, freezeNote.ID)+`]`), records: &recordedFiles{}}

	deep.run(t, []cli.LearnArgs{freezeLearn([]string{passageA.ID}, []string{"1.2026-10-01.production-cutover|narrows|c"})},
		nil)
}

//go:generate impgen cli.AmendRun --dependency --import-path github.com/toejough/engram/internal/cli

func Test855d4j1b_WriteDeep_ALessonTheVaultLacksIsCrystallizedWithItsEvidence(t *testing.T) {
	t.Parallel()

	records := &recordedFiles{}
	deep := deepWriting{carry: deepCarry(passageA), reply: deepReply(
		`[{"id":"`+passageA.ID+`","verdict":"evidence"}]`, `[`+freezeLesson(`["`+passageA.ID+`"]`, "")+`]`), records: records}

	deep.run(t, []cli.LearnArgs{freezeLearn([]string{passageA.ID}, nil)}, nil)

	g := NewWithT(t)
	all := records.all()

	g.Expect(records.kinds()).To(Equal([]string{"written"}))

	for _, record := range all {
		g.Expect(record["note"]).To(Equal("production-cutover-freeze.md"))
		g.Expect(record["session"]).To(Equal(sessionID))
		g.Expect(record["turn"]).To(BeEmpty(), "the run has no turn")
	}
}

func Test855d4j1c_WriteDeep_ACoveredNoteIsAmendedForProvenanceOnly(t *testing.T) {
	t.Parallel()

	records := &recordedFiles{}
	deep := deepWriting{carry: deepCarry(freezeNote, passageA), reply: deepReply(
		`[{"id":"`+freezeNote.ID+`","verdict":"covered","chunks":["`+passageA.ID+`"],"fields":{"object":"ignored"}}]`, `[]`),
		records: records}

	deep.run(t, nil, []cli.AmendArgs{{
		Vault: "/vault", VaultName: "personal", Target: freezeNote.ID, ChunkSources: []string{passageA.ID},
		ChunksDir: "/chunks",
	}})

	NewWithT(t).Expect(records.kinds()).To(Equal([]string{"written"}))
}

func Test855d4j1c_WriteDeep_AnAmendedNoteIsRecordedAsAmended(t *testing.T) {
	t.Parallel()

	records := &recordedFiles{}
	deep := deepWriting{carry: deepCarry(freezeNote, passageA), reply: deepReply(
		`[{"id":"`+freezeNote.ID+`","verdict":"covered","chunks":["`+passageA.ID+`"]}]`, `[]`), records: records}

	deep.run(t, nil, []cli.AmendArgs{{
		Vault: "/vault", VaultName: "personal", Target: freezeNote.ID, ChunkSources: []string{passageA.ID},
		ChunksDir: "/chunks",
	}})

	for _, record := range records.all() {
		NewWithT(t).Expect(record["reason"]).To(Equal("amended"))
	}
}

func Test855d4j1d_WriteDeep_ANearNoteIsRevisedWithTheFieldsTheJudgementNames(t *testing.T) {
	t.Parallel()

	records := &recordedFiles{}
	deep := deepWriting{carry: deepCarry(freezeNote, passageA), reply: deepReply(
		`[{"id":"`+freezeNote.ID+`","verdict":"near","chunks":["`+passageA.ID+`"],`+
			`"fields":{"object":"the last Friday of every month","behavior":"not a field of a fact"}}]`, `[]`),
		records: records}

	deep.run(t, nil, []cli.AmendArgs{{
		Vault: "/vault", VaultName: "personal", Target: freezeNote.ID, ChunkSources: []string{passageA.ID},
		ChunksDir: "/chunks", Object: "the last Friday of every month",
	}})

	g := NewWithT(t)
	kinds := records.kinds()

	g.Expect(kinds).To(ConsistOf("written", "skipped"))

	for _, record := range records.all() {
		if record["kind"] == "skipped" {
			g.Expect(record["reason"]).To(And(ContainSubstring("behavior"), ContainSubstring(freezeNote.ID)))
		}
	}
}

func Test855d4j1e_WriteDeep_ANoteOfTheParentVaultAndAPendingCopyAreNotAmended(t *testing.T) {
	t.Parallel()

	parent := freezeNote
	parent.ID, parent.Parent = "1.2026-10-01.parent-note.md", true
	pending := freezeNote
	pending.ID, pending.Pending = "2.2026-10-01.pending-note.md", true
	records := &recordedFiles{}
	deep := deepWriting{carry: deepCarry(parent, pending, passageA), reply: deepReply(
		`[{"id":"`+parent.ID+`","verdict":"near","chunks":["`+passageA.ID+`"],"fields":{"object":"x"}},`+
			`{"id":"`+pending.ID+`","verdict":"near","chunks":["`+passageA.ID+`"],"fields":{"object":"x"}}]`, `[]`),
		records: records}

	deep.run(t, nil, nil)

	g := NewWithT(t)

	g.Expect(records.kinds()).To(Equal([]string{"skipped", "skipped"}))

	var reasons strings.Builder

	for _, record := range records.all() {
		reasons.WriteString(fmt.Sprint(record["reason"]) + "\n")
	}

	g.Expect(reasons.String()).To(And(ContainSubstring(parent.ID), ContainSubstring(pending.ID)))
}

func Test855d4j1f_WriteDeep_AnIDTheCarryDoesNotHoldIsIgnored(t *testing.T) {
	t.Parallel()

	records := &recordedFiles{}
	deep := deepWriting{carry: deepCarry(freezeNote, passageA), reply: deepReply(
		`[{"id":"`+freezeNote.ID+`","verdict":"covered","chunks":["`+passageA.ID+`","other.md#nowhere"]},`+
			`{"id":"2.2026-10-01.other-file.md","verdict":"covered","chunks":["`+passageA.ID+`"]}]`,
		`[`+freezeLesson(`["other.md#nowhere"]`, "2.2026-10-01.other-file.md")+`]`),
		records: records}

	deep.run(t, []cli.LearnArgs{freezeLearn(nil, nil)}, []cli.AmendArgs{{
		Vault: "/vault", VaultName: "personal", Target: freezeNote.ID, ChunkSources: []string{passageA.ID},
		ChunksDir: "/chunks",
	}})
}

func Test855d4j1h_WriteDeep_AGlanceCarryWritesNothingAndSaysWhy(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	records := &recordedFiles{}
	carry := deepCarry(freezeNote, passageA)
	carry.Depth = auto.DepthGlance

	err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", RecordsDir: "/data/auto"},
		cli.AutoWriteDeps{
			Stdin: deepStdin(carry, deepReply(`[{"id":"`+freezeNote.ID+`","verdict":"covered","chunks":["`+passageA.ID+`"]}]`,
				`[`+freezeLesson(`["`+passageA.ID+`"]`, "")+`]`)),
			Learn: func(context.Context, cli.LearnArgs) (string, error) { return "", errRefused },
			Amend: func(context.Context, cli.AmendArgs) error { return errRefused },
			Now:   func() time.Time { return recordedAt }, WriteRecord: records.write,
			NoteExists: func(string) bool { return false },
		}, &bytes.Buffer{})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(records.kinds()).To(Equal([]string{"skipped"}), "no write was tried, so none failed")
}

func Test861b4zzzf1a_WriteDeep_AFailedAmendAndAFailedLearnAreRecordedAndALessonTheVaultStatesIsSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	records := &recordedFiles{}
	reply := deepReply(`[{"id":"`+freezeNote.ID+`","verdict":"covered","chunks":["`+passageA.ID+`"]}]`,
		`[`+strings.Replace(freezeLesson(`[]`, ""), `"covered":null`, `"covered":"`+freezeNote.ID+`"`, 1)+
			`,`+strings.Replace(freezeLesson(`[]`, ""), `"production-cutover-freeze"`, `"another"`, 1)+`]`)

	err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", RecordsDir: "/data/auto"},
		cli.AutoWriteDeps{
			Stdin: deepStdin(deepCarry(freezeNote, passageA), reply),
			Learn: func(context.Context, cli.LearnArgs) (string, error) { return "", errRefused },
			Amend: func(context.Context, cli.AmendArgs) error { return errRefused },
			Now:   func() time.Time { return recordedAt }, WriteRecord: records.write,
			NoteExists: func(path string) bool { return path == "/vault/"+freezeNote.ID },
		}, &bytes.Buffer{})

	g.Expect(err).NotTo(HaveOccurred(), "the failures are recorded")
	g.Expect(records.kinds()).To(ConsistOf("skipped", "failed", "failed"))
}

func Test861b4zzzf1b_WriteDeep_NothingOutsideTheCarryIsWritten(t *testing.T) {
	t.Parallel()

	pool := []string{passageA.ID, freezeNote.ID, "x.md#1", "2.2026-10-01.other.md", "3.2026-10-01.third.md"}

	rapid.Check(t, func(rt *rapid.T) {
		pick := func(label string) []string {
			return rapid.SliceOfN(rapid.SampledFrom(pool), 0, 4).Draw(rt, label)
		}
		chunks, noteChunks := pick("lesson chunks"), pick("note chunks")
		id := rapid.SampledFrom(pool).Draw(rt, "candidate")
		verdict := rapid.SampledFrom([]string{"covered", "near", "evidence", "unrelated"}).Draw(rt, "verdict")
		lesson := freezeLesson(quotedList(chunks), rapid.SampledFrom(pool).Draw(rt, "superseded"))
		reply := deepReply(`[{"id":"`+id+`","verdict":"`+verdict+`","chunks":`+quotedList(noteChunks)+`}]`, `[`+lesson+`]`)

		learned, amended := recordedDeepWrite(rt, reply)

		checkLearned(rt, learned, passagesAmong(chunks))

		judgedToAmend := verdict == "covered" || verdict == "near"
		wantedAmend := id == freezeNote.ID && judgedToAmend && len(passagesAmong(noteChunks)) > 0

		if wantedAmend != (len(amended) == 1) {
			rt.Fatalf("%d amends for %s judged %s with the chunks %v", len(amended), id, verdict, noteChunks)
		}

		for _, args := range amended {
			if args.Target != freezeNote.ID || !sameSet(args.ChunkSources, passagesAmong(noteChunks)) {
				rt.Fatalf("the amend %+v is not of the note and the passages of the carry", args)
			}
		}
	})
}

// unexported variables.
var (
	freezeNote = auto.Candidate{
		ID: "1.2026-10-01.production-cutover.md", Kind: auto.KindNote, Type: "fact", Situation: "planning a cutover",
		Text: "the production cutover is frozen on Fridays",
	}
	passageA = auto.Candidate{
		ID: "earlier.md#a1", Kind: auto.KindPassage, Text: "the cutover is frozen on the last Friday",
	}
)

// deepWriting is a deep write to run against mocks of the learn path and the amend.
type deepWriting struct {
	carry   auto.Carry
	reply   string
	records *recordedFiles
}

// run runs the writing of a deep delivery and expects the learn path and the amend to be given the arguments, in order.
func (d deepWriting) run(t *testing.T, learns []cli.LearnArgs, amends []cli.AmendArgs) {
	t.Helper()

	learn, learnCalls := MockLearnRun(t)
	amend, amendCalls := MockAmendRun(t)
	done := make(chan error, 1)

	var out bytes.Buffer

	go func() {
		done <- cli.RunAutoWrite(context.Background(),
			cli.AutoWriteArgs{VaultPath: "/vault", VaultName: "personal", RecordsDir: "/data/auto", ChunksDir: "/chunks"},
			cli.AutoWriteDeps{
				Stdin: deepStdin(d.carry, d.reply), Learn: learn, Amend: amend, Now: func() time.Time { return recordedAt },
				WriteRecord: d.records.write, NoteExists: func(string) bool { return false },
			}, &out)
	}()

	given := make(chan struct{})

	go func() {
		for _, args := range learns {
			learnCalls.ArgsShould(Not(BeNil()), Equal(args)).Return("/vault/"+args.Slug+".md", nil)
		}

		for _, args := range amends {
			amendCalls.ArgsShould(Not(BeNil()), Equal(args)).Return(nil)
		}

		close(given)
	}()

	select {
	case err := <-done:
		NewWithT(t).Expect(err).NotTo(HaveOccurred())
	case <-time.After(mockTimeout):
		t.Fatalf("the writing did not finish: expected the learn path %+v and the amend %+v", learns, amends)
	}

	select {
	case <-given:
	case <-time.After(mockTimeout):
		t.Fatalf("the writing finished without the learn path %+v and the amend %+v being given those arguments",
			learns, amends)
	}
}

// checkLearned fails the property when the one note written for a whole lesson has a chunk source or a supersession
// that the carry does not hold.
func checkLearned(rt *rapid.T, learned []cli.LearnArgs, wanted []string) {
	rt.Helper()

	if len(learned) != 1 {
		rt.Fatalf("%d notes were written for one whole lesson", len(learned))
	}

	if !sameSet(learned[0].ChunkSources, wanted) {
		rt.Fatalf("the lesson has the chunk sources %v, not %v", learned[0].ChunkSources, wanted)
	}

	for _, tie := range learned[0].Supersedes {
		note, _, _ := strings.Cut(tie, "|")
		if note+".md" != freezeNote.ID {
			rt.Fatalf("the lesson supersedes %q, which the carry does not hold", tie)
		}
	}
}

// deepCarry is the carry of a deep recall that holds the candidates.
func deepCarry(candidates ...auto.Candidate) auto.Carry {
	return auto.Carry{Cue: "prompt", Depth: auto.DepthDeep, Candidates: candidates}
}

// deepReply is the reply of the candidate judgement at deep depth.
func deepReply(candidates, lessons string) string {
	return `{"candidates":` + candidates + `,"lessons":` + lessons + `}`
}

// deepStdin is the input of the write of a deep delivery.
func deepStdin(carry auto.Carry, reply string) *strings.Reader {
	encoded, err := json.Marshal(map[string]any{"session": sessionID, "cue": "prompt", "reply": reply, "carry": carry})
	if err != nil {
		panic(err)
	}

	return strings.NewReader(string(encoded))
}

// freezeLearn is the write of freezeLesson through the learn path.
func freezeLearn(chunks, supersedes []string) cli.LearnArgs {
	return cli.LearnArgs{
		Type: "fact", Slug: "production-cutover-freeze", Vault: "/vault", VaultName: "personal", Position: "top",
		Source: "written automatically (session " + sessionID + ")", ChunkSources: chunks, Supersedes: supersedes,
		Situation: "planning a production cutover", Subject: "the production cutover", Predicate: "is frozen on",
		Object: "the last Friday of every month",
	}
}

// freezeLesson is a fact lesson that lists chunks and may supersede a note.
func freezeLesson(chunks, supersedes string) string {
	tie := ""
	if supersedes != "" {
		tie = `,"supersedes":"` + supersedes + `","type":"narrows","claim":"c"`
	}

	return `{"kind":"fact","slug":"production-cutover-freeze","situation":"planning a production cutover",` +
		`"covered":null,"chunks":` + chunks + tie + `,"subject":"the production cutover","predicate":"is frozen on",` +
		`"object":"the last Friday of every month"}`
}

// passagesAmong returns the IDs among those given that are the carry's passage.
func passagesAmong(ids []string) []string {
	var kept []string

	for _, one := range ids {
		if one == passageA.ID {
			kept = append(kept, one)
		}
	}

	return kept
}

// quotedList is a list of strings as JSON writes it.
func quotedList(items []string) string {
	encoded, err := json.Marshal(items)
	if err != nil {
		panic(err)
	}

	return string(encoded)
}

// recordedDeepWrite runs the writing of a deep delivery for the reply, over a carry of freezeNote and passageA, and
// returns what the learn path and the amend were given.
func recordedDeepWrite(rt *rapid.T, reply string) ([]cli.LearnArgs, []cli.AmendArgs) {
	var (
		learned []cli.LearnArgs
		amended []cli.AmendArgs
	)

	err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", VaultName: "personal"},
		cli.AutoWriteDeps{
			Stdin: deepStdin(deepCarry(freezeNote, passageA), reply),
			Learn: func(_ context.Context, args cli.LearnArgs) (string, error) {
				learned = append(learned, args)

				return "/vault/x.md", nil
			},
			Amend: func(_ context.Context, args cli.AmendArgs) error {
				amended = append(amended, args)

				return nil
			},
			Now: func() time.Time { return recordedAt }, WriteRecord: (&recordedFiles{}).write,
			NoteExists: func(string) bool { return false },
		}, &bytes.Buffer{})
	if err != nil {
		rt.Fatalf("writing: %v", err)
	}

	return learned, amended
}

// sameSet reports whether two lists hold the same items, whatever the order and the repeats.
func sameSet(left, right []string) bool {
	return len(left) == 0 && len(right) == 0 || slices.Equal(slices.Compact(slices.Sorted(slices.Values(left))),
		slices.Compact(slices.Sorted(slices.Values(right))))
}

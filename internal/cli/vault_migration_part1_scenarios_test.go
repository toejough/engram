package cli_test

// Scenario tests of the vault-migration capability, part 1 (docs/specs/product/vault-migration.md, 792a10a to
// 792a10e): the flat-vault notice, re-parenting only on request, the candidates, the refused answers, and the renames
// with their link rewrites. Helpers carry the prefix vm1.

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/update"
	"github.com/toejough/engram/internal/vaultgraph"
)

// Test792a10a1a_Update_OnlyTopLevelNotesGetTheBranchingNotice verifies 792a10a1a: a vault whose every note has a
// depth-1 Luhmann ID gets one notice naming engram update --reparent-luhmann, and no note file is modified.
func Test792a10a1a_Update_OnlyTopLevelNotesGetTheBranchingNotice(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := vm1MachineWithVault(
		vm1NoteSpec{id: "1", date: "2026-07-01", slug: "first-note", body: "first"},
		vm1NoteSpec{id: "2", date: "2026-07-02", slug: "second-note", body: "second"},
		vm1NoteSpec{id: "13", date: "2026-07-03", slug: "third-note", body: "third"},
	)
	before := machine.fileSystem.snapshot(iu3Vault)

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	notices := vm1ReparentLines(outcome.stdout)
	g.Expect(notices).To(HaveLen(1), "one line names the command")
	g.Expect(notices).To(ContainElement(ContainSubstring("`engram update --reparent-luhmann`")))
	g.Expect(machine.fileSystem.snapshot(iu3Vault)).To(Equal(before), "no note file is modified")
}

// Test792a10a1b_Update_ABranchedNoteSilencesTheBranchingNotice verifies 792a10a1b: one note with a Luhmann ID deeper
// than depth 1 silences the notice, whether the other notes are top-level or the branched notes are all there is.
func Test792a10a1b_Update_ABranchedNoteSilencesTheBranchingNotice(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		notes []vm1NoteSpec
	}{
		{
			name: "one branched note among top-level notes",
			notes: []vm1NoteSpec{
				{id: "1", date: "2026-07-01", slug: "first-note", body: "first"},
				{id: "2", date: "2026-07-02", slug: "second-note", body: "second"},
				{id: "1a", date: "2026-07-03", slug: "branch-note", body: "branched"},
			},
		},
		{
			name: "a note three segments deep alone",
			notes: []vm1NoteSpec{
				{id: "4a1", date: "2026-07-01", slug: "deep-note", body: "deep"},
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			machine := vm1MachineWithVault(testCase.notes...)

			outcome := machine.update(t, cli.UpdateArgs{})
			g.Expect(outcome.err).NotTo(HaveOccurred())

			if outcome.err != nil {
				return
			}

			g.Expect(vm1ReparentLines(outcome.stdout)).To(BeEmpty())
			g.Expect(outcome.stdout).NotTo(ContainSubstring("vault holds only top-level notes"))
		})
	}
}

// Test792a10a1c_Update_AnEmptyVaultHasNoBranchingNotice verifies 792a10a1c: a vault with no notes gets no notice.
func Test792a10a1c_Update_AnEmptyVaultHasNoBranchingNotice(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := vm1MachineWithVault()

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(vm1ReparentLines(outcome.stdout)).To(BeEmpty())
	g.Expect(outcome.stdout).NotTo(ContainSubstring("vault holds only top-level notes"))
}

// Test792a10a1d_Update_TheBranchingNoticeNamesTheAnsweringProcedure verifies 792a10a1d: the notice for a flat vault
// names both engram update --reparent-luhmann and the learn skill's batch mode.
func Test792a10a1d_Update_TheBranchingNoticeNamesTheAnsweringProcedure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := vm1MachineWithVault(
		vm1NoteSpec{id: "1", date: "2026-07-01", slug: "first-note", body: "first"},
		vm1NoteSpec{id: "2", date: "2026-07-02", slug: "second-note", body: "second"},
	)

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	notices := vm1ReparentLines(outcome.stdout)
	g.Expect(notices).To(HaveLen(1))

	if len(notices) != 1 {
		return
	}

	g.Expect(notices[0]).To(ContainSubstring("engram update --reparent-luhmann"))
	g.Expect(notices[0]).To(ContainSubstring("the learn skill's batch mode"))
}

// Test792a10b1a_Update_APlainUpdateNeverReParents verifies 792a10b1a: a vault of top-level notes with similar content
// and links between them is left alone by engram update without --reparent-luhmann, for a dry run, a real run and a
// run given an answers file. The control runs the same dependencies with the flag and sees them renamed.
func Test792a10b1a_Update_APlainUpdateNeverReParents(t *testing.T) {
	t.Parallel()

	plain := []struct {
		name string
		args cli.UpdateArgs
	}{
		{name: "a dry run", args: cli.UpdateArgs{DryRun: true}},
		{name: "a real run", args: cli.UpdateArgs{}},
		{name: "a run given an answers file", args: cli.UpdateArgs{Answers: vm1AnswersPath}},
	}

	for _, testCase := range plain {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			machine, vault := vm1MachineOverSimilarNotes(t)
			reparent, effects := vault.deps()
			vault.files[vm1AnswersPath] = vm1AnswersFile(t, reparent, vm1TwelveUnderSeven())
			before := maps.Clone(vault.files)
			machineBefore := machine.fileSystem.snapshot(vm1VaultDir)

			deps := cli.ExportNewUpdateDepsWithSpawnAndReparent(
				machine.fileSystem, machine.cmd, machine.env, machine.spawn, func(int) {}, reparent,
			)

			var stdout bytes.Buffer

			err := cli.ExportRunUpdate(t.Context(), testCase.args, deps, &stdout)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(effects.renames).To(BeEmpty(), "no note is renamed")
			g.Expect(effects.writes).To(BeEmpty(), "no link or supersedes reference is rewritten")
			g.Expect(vault.files).To(Equal(before))
			g.Expect(machine.fileSystem.snapshot(vm1VaultDir)).To(Equal(machineBefore))
			g.Expect(stdout.String()).NotTo(ContainSubstring("candidates"))
			g.Expect(stdout.String()).NotTo(ContainSubstring("renamed"))
		})
	}

	t.Run("the same dependencies re-parent when the flag is given", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		_, vault := vm1MachineOverSimilarNotes(t)
		outcome := vault.apply(t, vm1TwelveUnderSeven())
		g.Expect(outcome.err).NotTo(HaveOccurred())
		g.Expect(outcome.effects.renames).NotTo(BeEmpty(), "the probes see a re-parenting when it runs")
		g.Expect(outcome.effects.writes).NotTo(BeEmpty())
	})
}

// Test792a10b1b_Update_TheFlagRunsNothingElse verifies 792a10b1b: on a machine with no harness and no Go toolchain,
// engram update --reparent-luhmann derives candidates, and applies answers, without failing for lack of either, and
// runs no command and starts no binary. The control is the plain update on the same machine, which fails for
// lack of a harness.
func Test792a10b1b_Update_TheFlagRunsNothingElse(t *testing.T) {
	t.Parallel()

	t.Run("deriving candidates", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		machine, vault := vm1MachineWithoutHarness(t)
		reparent, _ := vault.deps()
		exits := []int{}
		deps := cli.ExportNewUpdateDepsWithSpawnAndReparent(
			machine.fileSystem, machine.cmd, machine.env, machine.spawn, func(code int) { exits = append(exits, code) },
			reparent,
		)

		var stdout bytes.Buffer

		err := cli.ExportRunUpdate(t.Context(), cli.UpdateArgs{ReparentLuhmann: true}, deps, &stdout)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(vm1CandidatePairs(t, stdout.String())).To(Equal([]string{"12>7", "7>12"}))
		g.Expect(machine.cmd.calls).To(BeEmpty(), "no go install, no git")
		g.Expect(machine.spawn.spawned()).To(BeEmpty())
		g.Expect(exits).To(BeEmpty())
	})

	t.Run("applying answers", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		machine, vault := vm1MachineWithoutHarness(t)
		reparent, effects := vault.deps()
		vault.files[vm1AnswersPath] = vm1AnswersFile(t, reparent, vm1TwelveUnderSeven())
		deps := cli.ExportNewUpdateDepsWithSpawnAndReparent(
			machine.fileSystem, machine.cmd, machine.env, machine.spawn, func(int) {}, reparent,
		)

		var stdout bytes.Buffer

		err := cli.ExportRunUpdate(
			t.Context(), cli.UpdateArgs{ReparentLuhmann: true, Answers: vm1AnswersPath}, deps, &stdout,
		)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(effects.renames).To(HaveKeyWithValue(
			vm1VaultDir+"/12.2026-01-02.second-note.md", vm1VaultDir+"/7a.2026-01-02.second-note.md"))
		g.Expect(machine.cmd.calls).To(BeEmpty())
		g.Expect(machine.spawn.spawned()).To(BeEmpty())
	})

	t.Run("the plain update on the same machine needs a harness", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		machine, vault := vm1MachineWithoutHarness(t)
		reparent, _ := vault.deps()
		deps := cli.ExportNewUpdateDepsWithSpawnAndReparent(
			machine.fileSystem, machine.cmd, machine.env, machine.spawn, func(int) {}, reparent,
		)

		err := cli.ExportRunUpdate(t.Context(), cli.UpdateArgs{DryRun: true}, deps, &bytes.Buffer{})
		g.Expect(err).To(MatchError(update.ErrNoHarness))
	})
}

// Test792a10c1a_Update_CandidatesAreEmittedForReview verifies 792a10c1a: a vault of top-level notes with sidecars,
// some pairs of which are similar, prints a JSON payload of the candidate pairs with a fingerprint and a next_command
// field, and renames and rewrites nothing.
func Test792a10c1a_Update_CandidatesAreEmittedForReview(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := vm1NewVault(t, vm1SimilarPairNotes()...)
	reparent, effects := vault.deps()
	before := maps.Clone(vault.files)

	stdout, err := vm1RunUpdate(t, reparent, cli.UpdateArgs{ReparentLuhmann: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	payload := vm1DecodePayload(t, stdout)
	g.Expect(vm1CandidatePairs(t, stdout)).To(Equal([]string{"12>7", "7>12"}))

	for _, candidate := range vm1Candidates(payload) {
		similarity, _ := candidate["similarity"].(float64)
		g.Expect(similarity).To(BeNumerically(">=", vm1SimilarityFloor))
		g.Expect(candidate["note_excerpt"]).NotTo(BeEmpty())
		g.Expect(candidate["target_excerpt"]).NotTo(BeEmpty())
	}

	g.Expect(payload["fingerprint"]).NotTo(BeEmpty())
	g.Expect(payload["next_command"]).To(Equal("engram update --reparent-luhmann --answers <path-to-answers-file>"))
	g.Expect(effects.renames).To(BeEmpty())
	g.Expect(effects.writes).To(BeEmpty())
	g.Expect(vault.files).To(Equal(before), "nothing is renamed, rewritten or created")
}

// Test792a10c1b_Update_NoPairIsSimilarEnough verifies 792a10c1b: top-level notes none of whose pairs reaches the
// similarity floor print "update --reparent-luhmann: no candidates found" and write nothing, for orthogonal vectors
// and for a pair just under the floor.
func Test792a10c1b_Update_NoPairIsSimilarEnough(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		vectors [][]float32
	}{
		{name: "orthogonal notes", vectors: [][]float32{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}},
		{name: "a pair just under the floor", vectors: [][]float32{{1, 0, 0}, {vm1JustUnderFloorX, vm1JustUnderFloorY, 0}}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			notes := make([]vm1NoteSpec, 0, len(testCase.vectors))

			for index, vector := range testCase.vectors {
				notes = append(notes, vm1NoteSpec{
					id: strconv.Itoa(index + 1), date: "2026-01-01", slug: "note-" + strconv.Itoa(index+1),
					body: "body " + strconv.Itoa(index+1), vector: vector,
				})
			}

			vault := vm1NewVault(t, notes...)
			reparent, effects := vault.deps()
			before := maps.Clone(vault.files)

			stdout, err := vm1RunUpdate(t, reparent, cli.UpdateArgs{ReparentLuhmann: true})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(stdout).To(Equal("update --reparent-luhmann: no candidates found\n"))
			g.Expect(effects.writes).To(BeEmpty())
			g.Expect(effects.renames).To(BeEmpty())
			g.Expect(vault.files).To(Equal(before))
		})
	}
}

// Test792a10c1c_Update_TheInstructionRoutesToTheLearnSkill verifies 792a10c1c: the instruction of a payload with
// candidates names the learn skill's batch mode, requires one entry per distinct candidate note, names --dry-run as
// the preview before applying, and does not tell the answering pass to run apply.
func Test792a10c1c_Update_TheInstructionRoutesToTheLearnSkill(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := vm1NewVault(t, vm1SimilarPairNotes()...)
	reparent, _ := vault.deps()

	stdout, err := vm1RunUpdate(t, reparent, cli.UpdateArgs{ReparentLuhmann: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	instruction, _ := vm1DecodePayload(t, stdout)["instruction"].(string)
	g.Expect(instruction).To(ContainSubstring("the learn skill's batch mode"))
	g.Expect(instruction).To(ContainSubstring("exactly one entry per distinct candidate"))
	g.Expect(instruction).To(ContainSubstring("preview it first with `next_command` plus --dry-run"))
	g.Expect(instruction).To(ContainSubstring("The answering pass itself does not run apply"))
	g.Expect(instruction).NotTo(ContainSubstring("then re-run"), "the answering pass is not told to apply")
}

// Test792a10c1d_Update_OnlyTopLevelNotesWithSidecarsAreConsidered verifies 792a10c1d: a branched note and a
// top-level note without a sidecar, each as similar as can be to a top-level note with a sidecar, appear neither as
// a candidate nor as a target.
func Test792a10c1d_Update_OnlyTopLevelNotesWithSidecarsAreConsidered(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	anchor := []float32{1, 0, 0}
	vault := vm1NewVault(t,
		vm1NoteSpec{id: "7", date: "2026-01-01", slug: "anchor-note", body: "the same text", vector: anchor},
		vm1NoteSpec{id: "12", date: "2026-01-02", slug: "near-note", body: "the same text", vector: anchor},
		vm1NoteSpec{id: "7a", date: "2026-01-03", slug: "branched-note", body: "the same text", vector: anchor},
		vm1NoteSpec{id: "20", date: "2026-01-04", slug: "no-sidecar-note", body: "the same text", noSidecar: true},
	)
	reparent, _ := vault.deps()

	stdout, err := vm1RunUpdate(t, reparent, cli.UpdateArgs{ReparentLuhmann: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(vm1CandidatePairs(t, stdout)).To(Equal([]string{"12>7", "7>12"}), "only 7 and 12 are considered")
}

// Test792a10d1a_Update_AMissingAnswersFileIsRefused verifies 792a10d1a: with no file at the --answers path, the
// update fails with an error that the answers file cannot be read, and the vault is unchanged.
func Test792a10d1a_Update_AMissingAnswersFileIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := vm1NewVault(t, vm1SimilarPairNotes()...)
	reparent, effects := vault.deps()
	before := maps.Clone(vault.files)

	_, err := vm1RunUpdate(t, reparent, cli.UpdateArgs{ReparentLuhmann: true, Answers: "/no/such/answers.json"})
	g.Expect(err).To(MatchError(ContainSubstring("reading --answers")))
	g.Expect(effects.writes).To(BeEmpty())
	g.Expect(effects.renames).To(BeEmpty())
	g.Expect(vault.files).To(Equal(before))
}

// Test792a10d1b_Update_MalformedAnswersAreRefused verifies 792a10d1b: an answers file that is not valid JSON fails
// with "cannot parse --answers JSON" and leaves the vault unchanged.
func Test792a10d1b_Update_MalformedAnswersAreRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"prose":          "these are not answers",
		"truncated JSON": `{"reparenting":[{"note":"12","position":"continuation"`,
		"an empty file":  "",
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			vault := vm1NewVault(t, vm1SimilarPairNotes()...)
			vault.files[vm1AnswersPath] = []byte(content)
			reparent, effects := vault.deps()
			before := maps.Clone(vault.files)

			_, err := vm1RunUpdate(t, reparent, cli.UpdateArgs{ReparentLuhmann: true, Answers: vm1AnswersPath})
			g.Expect(err).To(MatchError(ContainSubstring("cannot parse --answers JSON")))
			g.Expect(effects.writes).To(BeEmpty())
			g.Expect(effects.renames).To(BeEmpty())
			g.Expect(vault.files).To(Equal(before))
		})
	}
}

// Test792a10d1c_Update_StaleAnswersAreRefused verifies 792a10d1c: answers derived before the vault changed are
// refused with a stale-answers error telling the user to re-run engram update --reparent-luhmann, and no file is
// written and the chunk index is unchanged.
func Test792a10d1c_Update_StaleAnswersAreRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := vm1NewVault(t, vm1SimilarPairNotes()...)
	derivedWith, _ := vault.deps()
	answers := vm1AnswersFile(t, derivedWith, vm1TwelveUnderSeven())

	// The vault changes after the candidates were derived: a new top-level note arrives.
	vault.addNote(t, vm1NoteSpec{
		id: "30", date: "2026-01-04", slug: "late-note", body: "a late note", vector: []float32{0, 1, 0},
	})
	vault.files[vm1AnswersPath] = answers
	vault.files[vm1ChunksDir+"/manifest.json"] = []byte(`{"` + vm1VaultDir + `/12.2026-01-02.second-note.md":` +
		`{"mtime_unix_nano":1,"size":10,"file_hash":"sha256:old"}}`)
	vault.files[vm1ChunksDir+"/vault-12.jsonl"] = []byte("{\"chunk\":1}\n")

	reparent, effects := vault.deps()
	before := maps.Clone(vault.files)

	_, err := vm1RunUpdate(t, reparent, cli.UpdateArgs{ReparentLuhmann: true, Answers: vm1AnswersPath})
	g.Expect(err).To(MatchError(ContainSubstring("the vault changed since the candidates were derived")))
	g.Expect(err).To(MatchError(ContainSubstring("re-run `engram update --reparent-luhmann`")))
	g.Expect(effects.writes).To(BeEmpty())
	g.Expect(effects.renames).To(BeEmpty())
	g.Expect(vault.files).To(Equal(before), "no file is written and the chunk index is as it was")
}

// Test792a10d1d_Update_AnUnknownNoteIsRefused verifies 792a10d1d: an answers file naming a note ID that no note holds
// fails with "--answers references an unknown note" and leaves the vault unchanged, whichever position the unknown
// note is answered with.
func Test792a10d1d_Update_AnUnknownNoteIsRefused(t *testing.T) {
	t.Parallel()

	for _, position := range []string{"continuation", "sibling", "top"} {
		t.Run("answered "+position, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			vault := vm1NewVault(t, vm1SimilarPairNotes()...)
			reparent, effects := vault.deps()
			vault.files[vm1AnswersPath] = vm1AnswersFile(t, reparent,
				vm1TwelveUnderSeven(),
				vm1Answer{Note: "99", Position: position, Target: "7"},
			)
			before := maps.Clone(vault.files)

			_, err := vm1RunUpdate(t, reparent, cli.UpdateArgs{ReparentLuhmann: true, Answers: vm1AnswersPath})
			g.Expect(err).To(MatchError(ContainSubstring("--answers references an unknown note")))
			g.Expect(effects.writes).To(BeEmpty())
			g.Expect(effects.renames).To(BeEmpty())
			g.Expect(vault.files).To(Equal(before))
		})
	}
}

// Test792a10e1a_Update_AContinuationAnswerRenamesTheNoteAndRewritesLinks verifies 792a10e1a: note 12 answered
// continuation of note 7, and note 20 linking [[12.2026-05-01.old-slug]], give note 12's file and sidecar the new
// child ID under 7, its luhmann: field the new ID, its aliases the old basename, and note 20's link the new basename.
func Test792a10e1a_Update_AContinuationAnswerRenamesTheNoteAndRewritesLinks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		oldName = "12.2026-05-01.old-slug"
		newName = "7a.2026-05-01.old-slug"
	)

	vault := vm1NewVault(t,
		vm1NoteSpec{id: "7", date: "2026-05-01", slug: "parent-slug", body: "the parent"},
		vm1NoteSpec{id: "12", date: "2026-05-01", slug: "old-slug", body: "the child"},
		vm1NoteSpec{id: "20", date: "2026-05-02", slug: "linker", body: "see [[" + oldName + "]] for more"},
	)
	oldSidecar := slices.Clone(vault.files[vm1VaultDir+"/"+oldName+".vec.json"])

	outcome := vault.apply(t, vm1TwelveUnderSeven())
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(vault.files).NotTo(HaveKey(vm1VaultDir+"/"+oldName+".md"), "the old file is gone")
	g.Expect(vault.files).NotTo(HaveKey(vm1VaultDir+"/"+oldName+".vec.json"), "the old sidecar is gone")
	g.Expect(vault.files).To(HaveKey(vm1VaultDir + "/" + newName + ".md"))
	g.Expect(vault.files[vm1VaultDir+"/"+newName+".vec.json"]).To(Equal(oldSidecar), "the sidecar moved with the note")

	renamed := vm1ParseFrontmatter(t, vault.text(newName+".md"))
	g.Expect(renamed.Luhmann).To(Equal("7a"))
	g.Expect(renamed.Aliases).To(ContainElement(oldName))

	linker := vault.text("20.2026-05-02.linker.md")
	g.Expect(linker).To(ContainSubstring("[[" + newName + "]]"))
	g.Expect(linker).NotTo(ContainSubstring("[[" + oldName))
}

// Test792a10e1b_Update_RenamesAreOrderedByOriginalID verifies 792a10e1b: notes 2 and 3 answered continuation of
// note 1 become 1a and 1b, whatever the order of the answers file, and numerically rather than as text (9 before 10).
func Test792a10e1b_Update_RenamesAreOrderedByOriginalID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		answers []string          // the order of the answers file
		wantNew map[string]string // the new ID of each answered note
	}{
		{
			name: "notes 2 and 3", answers: []string{"3", "2"},
			wantNew: map[string]string{"2": "1a", "3": "1b"},
		},
		{
			name: "notes 9 and 10, where text order and number order disagree", answers: []string{"10", "9"},
			wantNew: map[string]string{"9": "1a", "10": "1b"},
		},
		{
			name: "notes 2, 10 and 30", answers: []string{"30", "10", "2"},
			wantNew: map[string]string{"2": "1a", "10": "1b", "30": "1c"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			notes := []vm1NoteSpec{{id: "1", date: "2026-01-01", slug: "parent-note", body: "the parent"}}
			answers := make([]vm1Answer, 0, len(testCase.answers))

			for _, id := range testCase.answers {
				notes = append(notes, vm1NoteSpec{id: id, date: "2026-01-02", slug: "note-" + id, body: "child " + id})
				answers = append(answers, vm1Answer{Note: id, Position: "continuation", Target: "1"})
			}

			vault := vm1NewVault(t, notes...)

			outcome := vault.apply(t, answers...)
			g.Expect(outcome.err).NotTo(HaveOccurred())

			for id, newID := range testCase.wantNew {
				g.Expect(outcome.effects.renames).To(HaveKeyWithValue(
					vm1VaultDir+"/"+id+".2026-01-02.note-"+id+".md",
					vm1VaultDir+"/"+newID+".2026-01-02.note-"+id+".md",
				))
			}

			g.Expect(outcome.effects.renames).To(HaveLen(2*len(testCase.wantNew)), "each note and its sidecar")
		})
	}
}

// Test792a10e1c_Update_CascadingRenamesResolveToTheFinalNames verifies 792a10e1c: two notes renamed in one apply that
// link each other each end up linking the other's new basename.
func Test792a10e1c_Update_CascadingRenamesResolveToTheFinalNames(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		firstOld  = "12.2026-01-02.first-slug"
		secondOld = "20.2026-01-03.second-slug"
		firstNew  = "7a.2026-01-02.first-slug"
		secondNew = "7b.2026-01-03.second-slug"
	)

	vault := vm1NewVault(t,
		vm1NoteSpec{id: "7", date: "2026-01-01", slug: "parent-slug", body: "the parent"},
		vm1NoteSpec{id: "12", date: "2026-01-02", slug: "first-slug", body: "points at [[" + secondOld + "]]"},
		vm1NoteSpec{id: "20", date: "2026-01-03", slug: "second-slug", body: "points at [[" + firstOld + "]]"},
	)

	outcome := vault.apply(t,
		vm1TwelveUnderSeven(),
		vm1Answer{Note: "20", Position: "continuation", Target: "7"},
	)
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	first := vault.text(firstNew + ".md")
	second := vault.text(secondNew + ".md")

	g.Expect(first).To(ContainSubstring("points at [[" + secondNew + "]]"))
	g.Expect(first).NotTo(ContainSubstring("[[" + secondOld))
	g.Expect(second).To(ContainSubstring("points at [[" + firstNew + "]]"))
	g.Expect(second).NotTo(ContainSubstring("[[" + firstOld))
}

// Test792a10e1d_Update_ATopAnswerChangesNothing verifies 792a10e1d: an answers file marking a candidate note top
// renames nothing, rewrites no link to the note, and leaves the chunk index as it was for the note.
func Test792a10e1d_Update_ATopAnswerChangesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const topName = "12.2026-01-02.second-note"

	vault := vm1NewVault(t,
		vm1NoteSpec{id: "7", date: "2026-01-01", slug: "first-note", body: "first", vector: []float32{1, 0, 0}},
		vm1NoteSpec{id: "12", date: "2026-01-02", slug: "second-note", body: "second", vector: []float32{0.99, 0.01, 0}},
		vm1NoteSpec{
			id: "20", date: "2026-01-03", slug: "linker", body: "see [[" + topName + "]]", vector: []float32{0, 0, 1},
		},
	)
	vault.files[vm1ChunksDir+"/manifest.json"] = []byte(`{"` + vm1VaultDir + "/" + topName + `.md":` +
		`{"mtime_unix_nano":1,"size":10,"file_hash":"sha256:old"}}`)
	vault.files[vm1ChunksDir+"/vault-12.jsonl"] = []byte("{\"chunk\":1}\n")
	before := maps.Clone(vault.files)

	outcome := vault.apply(t, vm1Answer{Note: "12", Position: "top", Target: "7"})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.effects.renames).To(BeEmpty(), "the note is not renamed")
	g.Expect(outcome.effects.writes).To(BeEmpty(), "no link to it is rewritten")

	delete(before, vm1AnswersPath)
	delete(vault.files, vm1AnswersPath)
	g.Expect(vault.files).To(Equal(before), "the notes, their sidecars and the chunk index are as they were")
}

// Test792a10e1e_Update_OnlySupersedesEntriesAreRewrittenInAFrontmatterList verifies 792a10e1e: a note whose
// supersedes: entry and whose parent link both name the same note keeps the parent link and gets the supersedes:
// entry rewritten when the named note is renamed.
func Test792a10e1e_Update_OnlySupersedesEntriesAreRewrittenInAFrontmatterList(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		oldName = "12.2026-01-02.named-slug"
		newName = "7a.2026-01-02.named-slug"
	)

	referrerFrontmatter := "supersedes:\n    - note: " + oldName + ".md\n      type: narrows\n      claim: older\n" +
		"parent:\n    vault: parent-vault-id\n    links:\n        - note: " + oldName +
		"\n          via: pulled\n          hash: xh1:parent\n"

	vault := vm1NewVault(t,
		vm1NoteSpec{id: "7", date: "2026-01-01", slug: "parent-slug", body: "the parent"},
		vm1NoteSpec{id: "12", date: "2026-01-02", slug: "named-slug", body: "the named note"},
		vm1NoteSpec{id: "20", date: "2026-01-03", slug: "referrer", body: "the referrer", frontmatter: referrerFrontmatter},
	)

	outcome := vault.apply(t, vm1TwelveUnderSeven())
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	referrer := vm1ParseFrontmatter(t, vault.text("20.2026-01-03.referrer.md"))
	g.Expect(referrer.Supersedes).To(HaveLen(1))
	g.Expect(referrer.Parent.Links).To(HaveLen(1))

	if len(referrer.Supersedes) != 1 || len(referrer.Parent.Links) != 1 {
		return
	}

	g.Expect(referrer.Supersedes[0].Note).To(Equal(newName+".md"), "the supersedes: entry names the new basename")
	g.Expect(referrer.Parent.Links[0].Note).To(Equal(oldName), "the parent link is not rewritten")
}

// Test792a10e1f_Update_RewrittenNotesAreNotLeftStale verifies 792a10e1f: a note that links a renamed note is reported
// with-embeddings by engram embed status after the apply, as is every other note.
func Test792a10e1f_Update_RewrittenNotesAreNotLeftStale(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const oldName = "12.2026-01-02.second-note"

	vault := vm1NewVault(t,
		vm1NoteSpec{id: "7", date: "2026-01-01", slug: "first-note", body: "first"},
		vm1NoteSpec{id: "12", date: "2026-01-02", slug: "second-note", body: "second"},
		vm1NoteSpec{id: "20", date: "2026-01-03", slug: "linker", body: "a note linking [[" + oldName + "]]"},
	)
	g.Expect(vm1EmbedStatus(t, vault)).To(ContainSubstring("with-embeddings: 3\n"), "every note starts current")

	outcome := vault.apply(t, vm1TwelveUnderSeven())
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(vault.text("20.2026-01-03.linker.md")).To(ContainSubstring("[[7a.2026-01-02.second-note]]"))
	g.Expect(vm1EmbedStatus(t, vault)).To(Equal("total:           3\n" +
		"with-embeddings: 3\n" +
		"without:         0\n" +
		"stale:           0\n" +
		"incompatible:    0\n" +
		"broken:          0\n"))
}

// Test792a10e1g_Update_AnUntouchedNoteIsNotWritten verifies 792a10e1g: a note that neither is renamed nor links a
// renamed note, here one with CRLF line endings that a write would convert, is byte-identical afterwards with its
// sidecar, and no write was made to it.
func Test792a10e1g_Update_AnUntouchedNoteIsNotWritten(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const untouched = "30.2026-01-04.untouched.md"

	vault := vm1NewVault(t,
		vm1NoteSpec{id: "7", date: "2026-01-01", slug: "first-note", body: "first"},
		vm1NoteSpec{id: "12", date: "2026-01-02", slug: "second-note", body: "second"},
		vm1NoteSpec{id: "20", date: "2026-01-03", slug: "linker", body: "see [[12.2026-01-02.second-note]]"},
		vm1NoteSpec{
			id: "30", date: "2026-01-04", slug: "untouched", body: "no link here [[7.2026-01-01.first-note]]", crlf: true,
		},
	)
	notePath := vm1VaultDir + "/" + untouched
	sidecarPath := embed.SidecarPath(notePath)
	noteBefore := slices.Clone(vault.files[notePath])
	sidecarBefore := slices.Clone(vault.files[sidecarPath])

	outcome := vault.apply(t, vm1TwelveUnderSeven())
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(outcome.effects.writes).NotTo(BeEmpty(), "the rename wrote the notes it touched")
	g.Expect(outcome.effects.writes).NotTo(ContainElement(notePath))
	g.Expect(outcome.effects.writes).NotTo(ContainElement(sidecarPath))
	g.Expect(vault.files[notePath]).To(Equal(noteBefore))
	g.Expect(vault.files[sidecarPath]).To(Equal(sidecarBefore))
}

// unexported constants.
const (
	vm1AnswersPath = "/answers.json"
	vm1ChunksDir   = "/chunks"
	// vm1JustUnderFloorX and vm1JustUnderFloorY make a unit vector whose cosine with (1, 0, 0) is 0.74, under the
	// 0.75 floor of a candidate.
	vm1JustUnderFloorX = 0.74
	vm1JustUnderFloorY = 0.6726
	vm1SimilarityFloor = 0.75
	vm1VaultDir        = "/vault"
)

// vm1Answer is one entry of an answers file.
type vm1Answer struct {
	Note     string `json:"note"`
	Position string `json:"position"`
	Target   string `json:"target"`
}

// vm1Effects records what a run did to the vault through the re-parenting's write capabilities.
type vm1Effects struct {
	writes  []string
	renames map[string]string
}

// vm1Frontmatter is the part of a note's frontmatter the scenarios read.
type vm1Frontmatter struct {
	Luhmann    string   `yaml:"luhmann"`
	Aliases    []string `yaml:"aliases"`
	Supersedes []struct {
		Note string `yaml:"note"`
	} `yaml:"supersedes"`
	Parent struct {
		Links []struct {
			Note string `yaml:"note"`
		} `yaml:"links"`
	} `yaml:"parent"`
}

// vm1NoteSpec describes one note of a vault fixture. A nil vector gives the note a sidecar that the test embedder
// built from the note, so its content hash and model are current; noSidecar leaves the note without one.
type vm1NoteSpec struct {
	id          string
	date        string
	slug        string
	body        string
	frontmatter string
	vector      []float32
	noSidecar   bool
	crlf        bool
}

func (n vm1NoteSpec) filename() string { return n.id + "." + n.date + "." + n.slug + ".md" }

// text is the note as the vault holds it.
func (n vm1NoteSpec) text() string {
	content := "---\ntype: fact\nluhmann: \"" + n.id + "\"\ncreated: " + n.date + "\n" + n.frontmatter +
		"---\n\n" + n.body + "\n"
	if n.crlf {
		return strings.ReplaceAll(content, "\n", "\r\n")
	}

	return content
}

// vm1Outcome is what applying an answers file returned and did.
type vm1Outcome struct {
	err     error
	effects *vm1Effects
}

// vm1Vault is an in-memory vault under /vault with a chunk index under /chunks.
type vm1Vault struct {
	files map[string][]byte
	names []string
}

// addNote puts a note, and its sidecar unless the spec has none, into the vault.
func (v *vm1Vault) addNote(t *testing.T, spec vm1NoteSpec) {
	t.Helper()

	g := NewWithT(t)

	path := vm1VaultDir + "/" + spec.filename()
	content := []byte(spec.text())
	v.files[path] = content
	v.names = append(v.names, spec.filename())

	switch {
	case spec.noSidecar:
		return
	case spec.vector != nil:
		v.files[embed.SidecarPath(path)] = reparentSidecar(spec.vector)
	default:
		sidecar, err := embed.BuildSidecar(t.Context(), reparentFakeEmbedder{}, content)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		v.files[embed.SidecarPath(path)] = embed.MarshalSidecar(sidecar)
	}
}

// apply derives the fingerprint of the vault, writes the answers file with it and runs engram update
// --reparent-luhmann --answers.
func (v *vm1Vault) apply(t *testing.T, entries ...vm1Answer) vm1Outcome {
	t.Helper()

	reparent, effects := v.deps()
	v.files[vm1AnswersPath] = vm1AnswersFile(t, reparent, entries...)

	_, err := vm1RunUpdate(t, reparent, cli.UpdateArgs{ReparentLuhmann: true, Answers: vm1AnswersPath})

	return vm1Outcome{err: err, effects: effects}
}

// deps builds the re-parenting's dependencies over the vault, recording what is written and renamed.
func (v *vm1Vault) deps() (cli.ReparentDeps, *vm1Effects) {
	reparent, renames := newReparentDeps(v.files, slices.Clone(v.names))
	effects := &vm1Effects{renames: renames}
	write := reparent.Rename.WriteFile
	reparent.Rename.WriteFile = func(path string, data []byte) error {
		effects.writes = append(effects.writes, path)

		return write(path, data)
	}

	return reparent, effects
}

// text is the content of a vault file, "" when there is none.
func (v *vm1Vault) text(filename string) string { return string(v.files[vm1VaultDir+"/"+filename]) }

// vm1AnswersFile derives the fingerprint of the vault behind reparent through engram update --reparent-luhmann and
// returns the answers file for entries with it.
func vm1AnswersFile(t *testing.T, reparent cli.ReparentDeps, entries ...vm1Answer) []byte {
	t.Helper()

	g := NewWithT(t)

	stdout, err := vm1RunUpdate(t, reparent, cli.UpdateArgs{ReparentLuhmann: true})
	g.Expect(err).NotTo(HaveOccurred())

	fingerprint, _ := vm1DecodePayload(t, stdout)["fingerprint"].(string)
	g.Expect(fingerprint).NotTo(BeEmpty())

	content, marshalErr := json.Marshal(struct {
		Reparenting []vm1Answer `json:"reparenting"`
		Fingerprint string      `json:"fingerprint"`
	}{Reparenting: entries, Fingerprint: fingerprint})
	g.Expect(marshalErr).NotTo(HaveOccurred())

	return content
}

// vm1CandidatePairs returns the sorted "note>target" pairs of the candidates in a derive payload.
func vm1CandidatePairs(t *testing.T, stdout string) []string {
	t.Helper()

	candidates := vm1Candidates(vm1DecodePayload(t, stdout))
	pairs := make([]string, 0, len(candidates))

	for _, candidate := range candidates {
		note, _ := candidate["note"].(string)
		target, _ := candidate["target"].(string)
		pairs = append(pairs, note+">"+target)
	}

	slices.Sort(pairs)

	return pairs
}

// vm1Candidates returns the candidate entries of a derive payload.
func vm1Candidates(payload map[string]any) []map[string]any {
	entries, _ := payload["candidates"].([]any)
	candidates := make([]map[string]any, 0, len(entries))

	for _, entry := range entries {
		if candidate, ok := entry.(map[string]any); ok {
			candidates = append(candidates, candidate)
		}
	}

	return candidates
}

// vm1DecodePayload decodes the JSON payload engram update --reparent-luhmann printed.
func vm1DecodePayload(t *testing.T, stdout string) map[string]any {
	t.Helper()

	g := NewWithT(t)

	payload := map[string]any{}

	err := json.Unmarshal([]byte(stdout), &payload)
	g.Expect(err).NotTo(HaveOccurred(), "the output is a JSON payload: %s", stdout)

	return payload
}

// vm1EmbedStatus is the report of engram embed status over the vault.
func vm1EmbedStatus(t *testing.T, vault *vm1Vault) string {
	t.Helper()

	g := NewWithT(t)

	deps := cli.EmbedDeps{
		Scan: func(dir string) ([]vaultgraph.Note, error) {
			notes := []vaultgraph.Note{}

			for path := range vault.files {
				if filepath.Dir(path) == dir && strings.HasSuffix(path, ".md") {
					notes = append(notes, vaultgraph.Note{Basename: strings.TrimSuffix(filepath.Base(path), ".md")})
				}
			}

			return notes, nil
		},
		Read: func(path string) ([]byte, error) {
			data, found := vault.files[path]
			if !found {
				return nil, fs.ErrNotExist
			}

			return data, nil
		},
		Embedder: reparentFakeEmbedder{},
	}

	var out bytes.Buffer

	err := cli.RunEmbedStatus(t.Context(), cli.EmbedStatusArgs{VaultPath: vm1VaultDir}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	return out.String()
}

// vm1MachineOverSimilarNotes is a machine with Claude Code whose vault, at /vault, holds the similar pair of notes
// with a note that links and supersedes one of them. The vault behind the re-parenting's capabilities is returned.
func vm1MachineOverSimilarNotes(t *testing.T) (*iu2Machine, *vm1Vault) {
	t.Helper()

	machine := iu3NewMachine()
	vault := vm1NewVault(t, vm1SimilarPairNotes()...)

	vm1SeedMachine(machine, vault)

	return machine, vault
}

// vm1MachineWithVault is a machine with Claude Code whose default vault holds notes, each with a sidecar.
func vm1MachineWithVault(notes ...vm1NoteSpec) *iu2Machine {
	machine := iu3NewMachine()
	machine.fileSystem.seedDir(iu3Vault)

	for _, note := range notes {
		machine.fileSystem.seedFile(iu3Vault+"/"+note.filename(), note.text())
		machine.fileSystem.seedFile(
			embed.SidecarPath(iu3Vault+"/"+note.filename()), string(reparentSidecar([]float32{1, 0, 0})))
	}

	return machine
}

// vm1MachineWithoutHarness is a machine with no Claude Code, no Pi and no Go toolchain, whose vault, at /vault, holds
// the similar pair of notes.
func vm1MachineWithoutHarness(t *testing.T) (*iu2Machine, *vm1Vault) {
	t.Helper()

	machine, vault := vm1MachineOverSimilarNotes(t)
	machine.fileSystem.removeAll(iu2ClaudeDir)
	machine.fileSystem.removeAll("/repo")
	machine.cmd.answers = append(machine.cmd.answers,
		iu1Answer{prefix: "go ", err: update.ErrCommandNotFound},
		iu1Answer{prefix: "git ", err: update.ErrCommandNotFound},
	)

	return machine, vault
}

// vm1NewVault is a vault holding notes.
func vm1NewVault(t *testing.T, notes ...vm1NoteSpec) *vm1Vault {
	t.Helper()

	vault := &vm1Vault{files: map[string][]byte{}, names: make([]string, 0, len(notes))}

	for _, note := range notes {
		vault.addNote(t, note)
	}

	return vault
}

// vm1ParseFrontmatter reads the frontmatter of a note.
func vm1ParseFrontmatter(t *testing.T, content string) vm1Frontmatter {
	t.Helper()

	g := NewWithT(t)

	rest, opened := strings.CutPrefix(content, "---\n")
	g.Expect(opened).To(BeTrue(), "the note opens its frontmatter")

	block, _, closed := strings.Cut(rest, "\n---\n")
	g.Expect(closed).To(BeTrue(), "the note closes its frontmatter")

	var parsed vm1Frontmatter

	err := yaml.Unmarshal([]byte(block), &parsed)
	g.Expect(err).NotTo(HaveOccurred())

	return parsed
}

// vm1ReparentLines returns the lines of a report that name --reparent-luhmann.
func vm1ReparentLines(stdout string) []string {
	return slices.DeleteFunc(iu2Lines(stdout), func(line string) bool {
		return !strings.Contains(line, "--reparent-luhmann")
	})
}

// vm1RunUpdate runs engram update with args over the re-parenting's capabilities, with the vault at /vault and the
// chunk index at /chunks. The machine has no harness, so an update that does not re-parent fails for lack of one.
func vm1RunUpdate(t *testing.T, reparent cli.ReparentDeps, args cli.UpdateArgs) (string, error) {
	t.Helper()

	env := iu1Env{
		home: iu2Home, cwd: "/repo",
		vars: map[string]string{"ENGRAM_VAULT_PATH": vm1VaultDir, "ENGRAM_CHUNKS_DIR": vm1ChunksDir},
	}
	deps := cli.ExportNewUpdateDepsFromWithReparent(newFakeUpdateFS(), stubCommander{}, env, reparent)

	var stdout bytes.Buffer

	err := cli.ExportRunUpdate(t.Context(), args, deps, &stdout)

	return stdout.String(), err
}

// vm1SeedMachine puts the notes and sidecars of vault into the vault of the machine, and points the machine at it.
func vm1SeedMachine(machine *iu2Machine, vault *vm1Vault) {
	machine.env.vars["ENGRAM_VAULT_PATH"] = vm1VaultDir
	machine.env.vars["ENGRAM_CHUNKS_DIR"] = vm1ChunksDir

	for path, data := range vault.files {
		machine.fileSystem.seedFile(path, string(data))
	}
}

// vm1SimilarPairNotes are three top-level notes: 7 and 12 near-identical in vector space, 20 far from both.
func vm1SimilarPairNotes() []vm1NoteSpec {
	return []vm1NoteSpec{
		{id: "7", date: "2026-01-01", slug: "first-note", body: "first note body text.", vector: []float32{1, 0, 0}},
		{
			id: "12", date: "2026-01-02", slug: "second-note", body: "second note body text.",
			vector: []float32{0.99, 0.01, 0},
		},
		{
			id: "20", date: "2026-01-03", slug: "unrelated-note", body: "unrelated, linking [[12.2026-01-02.second-note]].",
			vector: []float32{0, 0, 1},
		},
	}
}

// vm1TwelveUnderSeven answers note 12 as a continuation of note 7.
func vm1TwelveUnderSeven() vm1Answer {
	return vm1Answer{Note: "12", Position: "continuation", Target: "7"}
}

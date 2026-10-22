package cli_test

// Scenarios of Vault Migration, part 3 (861c2zzl): what `engram update --backfill-identity` stamps (792a10k) and what
// it keeps (792a10l). Each test drives the command through runUpdate over the in-memory machine of the install and
// update scenarios (iu3NewMachine), with the backfill's file access and detection wired to that machine: the
// detection is the one learn uses (detectRepo and detectUser over the machine's commander), the vault is a node of
// the machine's filesystem, and the update starts the installed binary as the real one does. The tests of the
// technical siblings (792b11c5, 792b11c7) keep their IDs and call the backfill directly (861d152).

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a10k1a_Update_TheCurrentEnvironmentIsStamped verifies Scenario 792a10k1a: a fact note and a feedback note
// missing repo:, user: and vault: gain all three, user: and vault: from the detection learn uses in the current
// environment and repo: from the working directory's git remote.
func Test792a10k1a_Update_TheCurrentEnvironmentIsStamped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	harness := vm3NewHarness()
	harness.machine.env.vars["ENGRAM_VAULT_NAME"] = "work"
	harness.seed("1.2026-01-01.a.md", vm2FactNote(""))
	harness.seed("2.2026-01-01.b.md", vm2FeedbackNote(""))

	outcome := harness.run(t, cli.UpdateArgs{BackfillIdentity: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.childErrs).To(BeEmpty())
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(vm3Stamped(2)))

	for _, name := range []string{"1.2026-01-01.a.md", "2.2026-01-01.b.md"} {
		fields := vm2Frontmatter(g, harness.note(name))
		g.Expect(fields).To(HaveKeyWithValue("repo", vm3Origin), name)
		g.Expect(fields).To(HaveKeyWithValue("user", vm3User), name)
		g.Expect(fields).To(HaveKeyWithValue("vault", "work"), name)
	}
}

// Test792a10k1b_Update_TheProjectFieldIsPreferred verifies Scenario 792a10k1b: a fact note and a feedback note whose
// project: is non-empty are stamped with that project: as repo:, each its own, not with the git remote of the
// working directory.
func Test792a10k1b_Update_TheProjectFieldIsPreferred(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	harness := vm3NewHarness()
	harness.seed("1.2026-01-01.a.md", vm2FactNote("project: github.com/acme/widgets\n"))
	harness.seed("2.2026-01-01.b.md", vm2FeedbackNote("project: github.com/acme/gadgets\n"))

	outcome := harness.run(t, cli.UpdateArgs{BackfillIdentity: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.childErrs).To(BeEmpty())
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(vm3Stamped(2)))

	fact := vm2Frontmatter(g, harness.note("1.2026-01-01.a.md"))
	g.Expect(fact).To(HaveKeyWithValue("repo", "github.com/acme/widgets"))
	g.Expect(fact).To(HaveKeyWithValue("project", "github.com/acme/widgets"))

	feedback := vm2Frontmatter(g, harness.note("2.2026-01-01.b.md"))
	g.Expect(feedback).To(HaveKeyWithValue("repo", "github.com/acme/gadgets"))

	g.Expect(harness.note("1.2026-01-01.a.md")).NotTo(ContainSubstring(vm3Origin))
	g.Expect(harness.note("2.2026-01-01.b.md")).NotTo(ContainSubstring(vm3Origin))
}

// Test792a10k1c_Update_WithoutAProjectFieldTheRepoIsDetected verifies Scenario 792a10k1c: a note whose project: is
// absent and a note whose project: is empty are stamped with the repository detected from the working directory, as
// learn detects it: the origin remote, else the name of the top-level directory.
func Test792a10k1c_Update_WithoutAProjectFieldTheRepoIsDetected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		answers []iu1Answer
		want    string
	}{
		{"the origin remote", nil, vm3Origin},
		{
			"the top-level directory name",
			[]iu1Answer{
				{prefix: "git remote get-url origin", err: errVM3Git},
				{prefix: "git rev-parse --show-toplevel", stdout: "/work/checkout\n"},
			},
			"checkout",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			harness := vm3NewHarness()
			harness.machine.cmd.answers = slices.Concat(tc.answers, harness.machine.cmd.answers)
			harness.seed("1.2026-01-01.a.md", vm2FactNote(""))
			harness.seed("2.2026-01-01.b.md", vm2FeedbackNote("project: \"\"\n"))

			outcome := harness.run(t, cli.UpdateArgs{BackfillIdentity: true})
			g.Expect(outcome.err).NotTo(HaveOccurred())
			g.Expect(outcome.childErrs).To(BeEmpty())
			g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(vm3Stamped(2)))

			for _, name := range []string{"1.2026-01-01.a.md", "2.2026-01-01.b.md"} {
				g.Expect(vm2Frontmatter(g, harness.note(name))).To(HaveKeyWithValue("repo", tc.want), name)
			}

			detections := harness.machine.cmd.callsOf("git remote get-url origin")
			g.Expect(detections).NotTo(BeEmpty(), "the repository is detected afresh")

			for _, call := range detections {
				g.Expect(call.dir).To(Equal(harness.machine.env.cwd), "in the working directory")
			}
		})
	}
}

// Test792a10k1d_Update_ABackfillTwiceStampsOnce verifies Scenario 792a10k1d: after a backfill has stamped the vault,
// a second backfill writes no note and says there is nothing to backfill.
func Test792a10k1d_Update_ABackfillTwiceStampsOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	harness := vm3NewHarness()
	harness.seed("1.2026-01-01.a.md", vm2FactNote(""))
	harness.seed("2.2026-01-01.b.md", vm2FeedbackNote(""))

	first := harness.run(t, cli.UpdateArgs{BackfillIdentity: true})
	g.Expect(first.err).NotTo(HaveOccurred())
	g.Expect(iu2Lines(first.stdout)).To(ContainElement(vm3Stamped(2)))

	stamped := harness.machine.fileSystem.snapshot(iu3Vault)
	writtenByFirst := len(harness.writes)
	g.Expect(writtenByFirst).To(Equal(2))

	second := harness.run(t, cli.UpdateArgs{BackfillIdentity: true})
	g.Expect(second.err).NotTo(HaveOccurred())
	g.Expect(second.childErrs).To(BeEmpty())
	g.Expect(iu2Lines(second.stdout)).To(ContainElement("update --backfill-identity: nothing to backfill"))
	g.Expect(second.stdout).NotTo(ContainSubstring("stamped"))
	g.Expect(harness.writes).To(HaveLen(writtenByFirst), "the second run writes no note")
	g.Expect(harness.machine.fileSystem.snapshot(iu3Vault)).To(Equal(stamped), "and no note changes")
}

// Test792a10k1e_Update_AnOmittedUserIsFilledIn verifies Scenario 792a10k1e: a fact note and a feedback note with
// repo: and vault: work but no user: gain the detected user: and keep their repo: and vault:; where user detection
// still resolves empty the notes are not written.
func Test792a10k1e_Update_AnOmittedUserIsFilledIn(t *testing.T) {
	t.Parallel()

	userless := func(harness *vm3Harness) {
		harness.seed("1.2026-01-01.a.md", vm2FactNote("repo: github.com/acme/widgets\nvault: work\n"))
		harness.seed("2.2026-01-01.b.md", vm2FeedbackNote("repo: github.com/acme/widgets\nvault: work\n"))
	}

	t.Run("the user resolves", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		harness := vm3NewHarness()
		userless(harness)

		outcome := harness.run(t, cli.UpdateArgs{BackfillIdentity: true})
		g.Expect(outcome.err).NotTo(HaveOccurred())
		g.Expect(outcome.childErrs).To(BeEmpty())
		g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(vm3Stamped(2)))

		for _, name := range []string{"1.2026-01-01.a.md", "2.2026-01-01.b.md"} {
			fields := vm2Frontmatter(g, harness.note(name))
			g.Expect(fields).To(HaveKeyWithValue("user", vm3User), name)
			g.Expect(fields).To(HaveKeyWithValue("repo", "github.com/acme/widgets"), name)
			g.Expect(fields).To(HaveKeyWithValue("vault", "work"), name)
		}
	})

	t.Run("the user still resolves empty", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		harness := vm3NewHarness()
		harness.machine.cmd.answers = vm3Answers("", vm3Origin)
		userless(harness)

		before := harness.machine.fileSystem.snapshot(iu3Vault)

		outcome := harness.run(t, cli.UpdateArgs{BackfillIdentity: true})
		g.Expect(outcome.err).NotTo(HaveOccurred())
		g.Expect(outcome.childErrs).To(BeEmpty())
		g.Expect(iu2Lines(outcome.stdout)).To(ContainElement("update --backfill-identity: nothing to backfill"))
		g.Expect(harness.writes).To(BeEmpty(), "no note is written")
		g.Expect(harness.machine.fileSystem.snapshot(iu3Vault)).To(Equal(before))
	})
}

// Test792a10k1f_Update_ADryRunWritesNothing verifies Scenario 792a10k1f: with --dry-run the output carries
// "[dry-run]" and "would stamp" with the number of notes missing identity, and no file is written or changed.
func Test792a10k1f_Update_ADryRunWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	harness := vm3NewHarness()
	harness.seed("1.2026-01-01.a.md", vm2FactNote(""))
	harness.seed("2.2026-01-01.b.md", vm2FeedbackNote(""))
	harness.seed("3.2026-01-01.c.md", vm2FactNote("project: github.com/acme/widgets\n"))

	before := harness.machine.fileSystem.snapshot(iu3Vault)

	outcome := harness.run(t, cli.UpdateArgs{BackfillIdentity: true, DryRun: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(
		"[dry-run] update --backfill-identity: would stamp 3 note(s) with repo:/user:/vault: provenance"))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("update --backfill-identity: stamped"))
	g.Expect(harness.writes).To(BeEmpty(), "no file is written")
	g.Expect(harness.machine.fileSystem.snapshot(iu3Vault)).To(Equal(before), "and no file is modified")
}

// Test792a10k1g_Update_OtherNoteTypesAreLeftAlone verifies Scenario 792a10k1g as modified in 861a4zzl1: a vocab
// term-note (retired, type: vocab) and a runbook note without user: are neither written nor changed, while a fact
// note beside them is stamped. A current vocab definition note is a fact and is stamped like any other (861d153).
func Test792a10k1g_Update_OtherNoteTypesAreLeftAlone(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		termName    = "vocab.recall.md"
		termNote    = "---\ntype: vocab\nterm: recall\ndescription: recall the vault\n---\n\nbody\n"
		runbookName = "3.2026-01-01.c-runbook.md"
		runbookNote = "---\ntype: runbook\nsituation: s\ndone_when: d\nluhmann: \"3\"\ncreated: 2026-01-01\n---\n\n" +
			"1. step\n"
		factName = "1.2026-01-01.a.md"
	)

	harness := vm3NewHarness()
	harness.seed(termName, termNote)
	harness.seed(runbookName, runbookNote)
	harness.seed(factName, vm2FactNote(""))

	outcome := harness.run(t, cli.UpdateArgs{BackfillIdentity: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.childErrs).To(BeEmpty())
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(vm3Stamped(1)), "only the fact note counts")
	g.Expect(harness.writes).To(ConsistOf(iu3Vault+"/"+factName), "and only the fact note is written")
	g.Expect(harness.note(termName)).To(Equal(termNote))
	g.Expect(harness.note(runbookName)).To(Equal(runbookNote))
}

// Test792a10l1a_Update_AnUnmodeledKeyIsKept verifies Scenario 792a10l1a: a fact note and a feedback note that carry a
// key the typed note model does not define, a string and a nested map, are stamped and still carry those keys with
// their values.
func Test792a10l1a_Update_AnUnmodeledKeyIsKept(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const extra = "luhmann_old: \"12\"\nprovenance:\n    origin: import\n    steps:\n        - a\n        - b\n"

	harness := vm3NewHarness()
	harness.seed("1.2026-01-01.a.md", vm2FactNote(extra))
	harness.seed("2.2026-01-01.b.md", vm2FeedbackNote(extra))

	outcome := harness.run(t, cli.UpdateArgs{BackfillIdentity: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.childErrs).To(BeEmpty())
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(vm3Stamped(2)))

	for _, name := range []string{"1.2026-01-01.a.md", "2.2026-01-01.b.md"} {
		fields := vm2Frontmatter(g, harness.note(name))
		g.Expect(fields).To(HaveKeyWithValue("repo", vm3Origin), name)
		g.Expect(fields).To(HaveKeyWithValue("user", vm3User), name)
		g.Expect(fields).To(HaveKeyWithValue("vault", "personal"), name)
		g.Expect(fields).To(HaveKeyWithValue("luhmann_old", "12"), name)
		g.Expect(fields).To(HaveKeyWithValue("provenance", map[string]any{
			"origin": "import", "steps": []any{"a", "b"},
		}), name)
	}
}

// Test792a10l1b_Update_AnAnchorOnAnUneditedKeyIsKept verifies Scenario 792a10l1b: a note whose source: &src test is
// aliased by another key is stamped, and source and the aliasing key both still decode to test.
func Test792a10l1b_Update_AnAnchorOnAnUneditedKeyIsKept(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		name = "1.2026-01-01.a.md"
		note = "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\nluhmann: \"1\"\n" +
			"created: 2026-01-01\nsource: &src test\nx_src: *src\n---\n\nbody\n"
	)

	harness := vm3NewHarness()
	harness.seed(name, note)

	outcome := harness.run(t, cli.UpdateArgs{BackfillIdentity: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.childErrs).To(BeEmpty())
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(vm3Stamped(1)))

	fields := vm2Frontmatter(g, harness.note(name))
	g.Expect(fields).To(HaveKeyWithValue("user", vm3User), "the note is stamped")
	g.Expect(fields).To(HaveKeyWithValue("source", "test"))
	g.Expect(fields).To(HaveKeyWithValue("x_src", "test"))
	g.Expect(harness.note(name)).To(ContainSubstring("source: &src test\n"))
	g.Expect(harness.note(name)).To(ContainSubstring("x_src: *src\n"))
}

// Test792a10l1c_Update_AnAnchoredIdentityKeyIsRefused verifies Scenario 792a10l1c: of two flagged notes, the one whose
// user: key carries an anchor that another key aliases is left byte-identical, the other is stamped, and the command
// fails naming the first note and user.
func Test792a10l1c_Update_AnAnchoredIdentityKeyIsRefused(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		anchoredName = "1.2026-01-01.anchored.md"
		plainName    = "2.2026-01-01.plain.md"
	)

	anchored := vm2FactNote("user: &u \"\"\nx_user: *u\n")

	harness := vm3NewHarness()
	harness.seed(anchoredName, anchored)
	harness.seed(plainName, vm2FeedbackNote(""))

	outcome := harness.run(t, cli.UpdateArgs{BackfillIdentity: true})
	g.Expect(outcome.err).NotTo(HaveOccurred(), "the parent only relays the exit status of the binary it started")
	g.Expect(outcome.exits).To(Equal([]int{1}), "the command fails")
	g.Expect(outcome.childErrs).To(HaveLen(1))

	if len(outcome.childErrs) != 1 {
		return
	}

	g.Expect(outcome.childErrs[0]).To(MatchError(ContainSubstring(anchoredName)))
	g.Expect(outcome.childErrs[0]).To(MatchError(ContainSubstring("YAML anchor: user")))
	g.Expect(outcome.childErrs[0]).NotTo(MatchError(ContainSubstring(plainName)))
	g.Expect(harness.note(anchoredName)).To(Equal(anchored), "the refused note is byte-identical")
	g.Expect(vm2Frontmatter(g, harness.note(plainName))).To(HaveKeyWithValue("user", vm3User), "the other is stamped")
	g.Expect(harness.writes).To(ConsistOf(iu3Vault + "/" + plainName))
}

// Test792a10l1d_Update_ACRLFNoteIsConverted verifies Scenario 792a10l1d: a fact note missing identity whose lines end
// in \r\n is stamped and written with no \r\n, byte-identical to the backfill of its LF form, and its sidecar is
// rebuilt for the new content.
func Test792a10l1d_Update_ACRLFNoteIsConverted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		name    = "1.2026-01-01.a.md"
		sidecar = "1.2026-01-01.a.vec.json"
	)

	lfNote := vm2FactNote("luhmann_old: \"12\"\n")

	lfHarness := vm3NewHarness()
	lfHarness.seed(name, lfNote)

	lfOutcome := lfHarness.run(t, cli.UpdateArgs{BackfillIdentity: true})
	g.Expect(lfOutcome.err).NotTo(HaveOccurred())

	crlfHarness := vm3NewHarness()
	crlfHarness.seed(name, toCRLF(lfNote))

	outcome := crlfHarness.run(t, cli.UpdateArgs{BackfillIdentity: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.childErrs).To(BeEmpty())
	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(vm3Stamped(1)), "the note is stamped, not skipped")

	written := crlfHarness.note(name)
	g.Expect(written).NotTo(ContainSubstring("\r\n"))
	g.Expect(written).To(Equal(lfHarness.note(name)), "byte-identical to the backfill of its LF form")
	g.Expect(vm2Frontmatter(g, written)).To(HaveKeyWithValue("user", vm3User))

	rebuilt, sidecarErr := embed.UnmarshalSidecar([]byte(crlfHarness.note(sidecar)))
	g.Expect(sidecarErr).NotTo(HaveOccurred())

	if sidecarErr != nil {
		return
	}

	g.Expect(rebuilt.ContentHash).To(Equal(embed.ContentHash([]byte(written))), "its sidecar is fresh")
}

// Test792a10l1e_Update_AnOrdinaryNoteIsWrittenAsBefore verifies Scenario 792a10l1e: every fact and feedback note
// holding only keys the typed note model defines and no anchors is written byte-for-byte as the backfill wrote it
// before node edits were used (the goldens of testdata/backfill_identity_parity, written by that backfill).
func Test792a10l1e_Update_AnOrdinaryNoteIsWrittenAsBefore(t *testing.T) {
	t.Parallel()

	for _, tc := range backfillParityCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			const name = "1aa.2026-01-01.note.md"

			golden, readErr := os.ReadFile(filepath.Join("testdata", "backfill_identity_parity", tc.name+".md"))
			g.Expect(readErr).NotTo(HaveOccurred())

			harness := vm3NewHarness()
			harness.machine.cmd.answers = vm3Answers(vm3User, tc.repo)
			harness.seed(name, tc.input)

			outcome := harness.run(t, cli.UpdateArgs{BackfillIdentity: true})
			g.Expect(outcome.err).NotTo(HaveOccurred())
			g.Expect(outcome.childErrs).To(BeEmpty())
			g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(vm3Stamped(1)))
			g.Expect(harness.note(name)).To(Equal(string(golden)),
				"a note without unknown keys or anchors is written byte-for-byte as before")
		})
	}
}

// unexported constants.
const (
	// vm3AnswerCount is the most answers a machine gives: user.email, the origin remote and the top level.
	vm3AnswerCount = 3
	// vm3NotePerm is the mode the backfill's note writes ask the filesystem for.
	vm3NotePerm fs.FileMode = 0o644
	// vm3Origin is the origin remote of the working directory of the machine.
	vm3Origin = "git@github.com:example/detected.git"
	// vm3User is the user.email the machine's git config answers.
	vm3User = "bob@example.com"
)

// unexported variables.
var (
	errVM3Git    = errors.New("git: exit status 128")
	errVM3NoUser = errors.New("no OS user")
)

// vm3Harness is the machine of the update scenarios with the identity-backfill dependencies wired to its in-memory
// filesystem and commander, and the paths the backfill wrote.
type vm3Harness struct {
	machine *iu2Machine
	writes  []string
}

// identityDeps builds the dependencies of the backfill over the machine, for the environment env: the detection of
// learn over the machine's commander, and the vault listed, read and written as nodes of the machine's filesystem.
func (h *vm3Harness) identityDeps(env iu1Env) cli.IdentityDeps {
	fileSystem := h.machine.fileSystem

	return cli.IdentityDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(vault string) ([]string, error) {
			entries, err := fileSystem.ReadDir(vault)
			if err != nil {
				return nil, fmt.Errorf("listing %s: %w", vault, err)
			}

			names := make([]string, 0, len(entries))

			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
					names = append(names, entry.Name())
				}
			}

			slices.Sort(names)

			return names, nil
		},
		ReadFile: fileSystem.ReadFile,
		WriteFile: func(path string, data []byte) error {
			h.writes = append(h.writes, path)

			return fileSystem.WriteFile(path, data, vm3NotePerm)
		},
		DetectRepo: func(ctx context.Context) string {
			return cli.ExportDetectRepo(ctx, env.Getwd, h.machine.cmd)
		},
		DetectUser: func(ctx context.Context) string {
			return cli.ExportDetectUser(ctx, h.machine.cmd, func() (string, error) { return "", errVM3NoUser })
		},
		Getenv:   env.Getenv,
		Embedder: stubEmbedder{modelID: "stub@4", dims: 4},
	}
}

// note is the text of the vault file name as it is now.
func (h *vm3Harness) note(name string) string {
	return h.machine.fileSystem.text(iu3Vault + "/" + name)
}

// run runs `engram update` with args as the installed binary runs it: the parent starts the binary, which runs the
// rest of the update with the environment it was handed and the flags the parent passed on, and a dry run runs
// in the parent. Both carry the identity dependencies of the harness.
func (h *vm3Harness) run(t *testing.T, args cli.UpdateArgs) iu2Outcome {
	t.Helper()

	var stdout bytes.Buffer

	exits := []int{}
	childErrs := []error{}
	machine := h.machine

	machine.spawn.run = func(_ string, spawnArgs, extraEnv []string) (int, error) {
		vars := maps.Clone(machine.env.vars)

		for _, entry := range extraEnv {
			key, value, _ := strings.Cut(entry, "=")
			vars[key] = value
		}

		childEnv := iu1Env{home: machine.env.home, cwd: machine.env.cwd, vars: vars}
		childArgs := cli.UpdateArgs{BackfillIdentity: slices.Contains(spawnArgs, "--backfill-identity")}
		childDeps := cli.ExportNewUpdateDepsWithSpawnAndIdentity(
			machine.fileSystem, machine.cmd, childEnv, &iu1Spawner{}, func(int) {}, h.identityDeps(childEnv))

		childErr := cli.ExportRunUpdate(t.Context(), childArgs, childDeps, &stdout)
		if childErr != nil {
			childErrs = append(childErrs, childErr)

			return 1, nil
		}

		return 0, nil
	}

	deps := cli.ExportNewUpdateDepsWithSpawnAndIdentity(
		machine.fileSystem, machine.cmd, machine.env, machine.spawn, func(code int) { exits = append(exits, code) },
		h.identityDeps(machine.env))

	err := cli.ExportRunUpdate(t.Context(), args, deps, &stdout)

	return iu2Outcome{err: err, stdout: stdout.String(), exits: exits, childErrs: childErrs}
}

// seed puts the note name into the vault of the machine.
func (h *vm3Harness) seed(name, content string) {
	h.machine.fileSystem.seedFile(iu3Vault+"/"+name, content)
}

// vm3Answers is the commander answers of a machine whose git config user.email is user and whose working directory
// has the origin remote origin; an empty user or origin is one git cannot answer, and with no origin the working
// directory is not inside a repository either.
func vm3Answers(user, origin string) []iu1Answer {
	answers := make([]iu1Answer, 0, vm3AnswerCount)

	if user == "" {
		answers = append(answers, iu1Answer{prefix: "git config user.email", err: errVM3Git})
	} else {
		answers = append(answers, iu1Answer{prefix: "git config user.email", stdout: user + "\n"})
	}

	if origin == "" {
		return append(answers,
			iu1Answer{prefix: "git remote get-url origin", err: errVM3Git},
			iu1Answer{prefix: "git rev-parse --show-toplevel", err: errVM3Git})
	}

	return append(answers, iu1Answer{prefix: "git remote get-url origin", stdout: origin + "\n"})
}

// vm3NewHarness is a harness over a machine whose git config user.email is vm3User and whose working directory has
// the origin remote vm3Origin.
func vm3NewHarness() *vm3Harness {
	machine := iu3NewMachine()
	machine.cmd.answers = vm3Answers(vm3User, vm3Origin)

	return &vm3Harness{machine: machine}
}

// vm3Stamped is the line the update prints when the backfill stamped count notes.
func vm3Stamped(count int) string {
	return "update --backfill-identity: stamped " + strconv.Itoa(count) + " note(s) with repo:/user:/vault: provenance"
}

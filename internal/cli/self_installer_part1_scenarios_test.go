package cli_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// Test792b28c10a2_ReexecArgsFrom_ForwardsOnlyTheThreeReexecableFlags verifies 792b28c10a2: of --with-guidance,
// --regen-vocab, --backfill-identity, --allow-downgrade and --dry-run, the hand-off gets the first three, in order
// (docs/specs/technical/self-installer.md, 792b28c10a).
func Test792b28c10a2_ReexecArgsFrom_ForwardsOnlyTheThreeReexecableFlags(t *testing.T) {
	t.Parallel()

	allFlags := cli.UpdateArgs{
		WithGuidance:     true,
		RegenVocab:       true,
		BackfillIdentity: true,
		AllowDowngrade:   true,
		DryRun:           true,
	}
	wantForwarded := []string{"--with-guidance", "--regen-vocab", "--backfill-identity"}

	t.Run("the arguments built from all five flags", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		g.Expect(cli.ExportReexecArgsFrom(allFlags)).To(Equal(wantForwarded))
	})

	t.Run("the spawned child receives them after the update subcommand", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		spawner := &si1ArgsSpawner{}
		deps := cli.ExportNewUpdateDepsWithSpawn(
			singleClaudeHarnessFS(), stubCommander{}, reexecEnv{home: reexecHome, cwd: "/repo"}, spawner,
			func(int) {},
		)

		handedOff := allFlags
		handedOff.DryRun = false // a dry run never spawns, so the hand-off is observed on a real run

		err := cli.ExportRunUpdate(context.Background(), handedOff, deps, &bytes.Buffer{})
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(spawner.spawned()).To(Equal([][]string{append([]string{"update"}, wantForwarded...)}))
	})
}

// Test792b28c10a8_RunUpdate_PlainUpdate_NeverCallsTheReparenting verifies 792b28c10a8: with a vault of similar
// top-level notes and no --reparent-luhmann, the update only prints the branching notice and calls none of the
// re-parenting's capabilities. The same deps do get called when the flag is given, so the probes are live. The
// probes are counting fakes: impgen cannot generate a mock of embed.Embedder (it imports the standard library
// "embed" for the package) and the other capabilities are func fields.
func Test792b28c10a8_RunUpdate_PlainUpdate_NeverCallsTheReparenting(t *testing.T) {
	t.Parallel()

	t.Run("a plain update leaves the re-parenting alone", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		calls := &si1CallCounter{}
		deps := cli.ExportNewUpdateDepsFromWithReparent(
			si1VaultOfSimilarNotes(), stubCommander{}, reexecEnv{home: reexecHome, cwd: "/repo"},
			si1ReparentDeps(calls),
		)

		stdout := &bytes.Buffer{}
		err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{DryRun: true}, deps, stdout)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(calls.count()).To(BeZero(), "no re-parenting capability may be called")
		g.Expect(stdout.String()).To(ContainSubstring("vault holds only top-level notes"),
			"the vault is flat, so the update only names the command")
		g.Expect(stdout.String()).NotTo(ContainSubstring("update --reparent-luhmann: no candidates found"))
		g.Expect(stdout.String()).NotTo(ContainSubstring("next_command"))
	})

	t.Run("the same deps are called when the flag is given", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		calls := &si1CallCounter{}
		deps := cli.ExportNewUpdateDepsFromWithReparent(
			si1VaultOfSimilarNotes(), stubCommander{}, reexecEnv{home: reexecHome, cwd: "/repo"},
			si1ReparentDeps(calls),
		)

		err := cli.ExportRunUpdate(
			context.Background(), cli.UpdateArgs{ReparentLuhmann: true}, deps, &bytes.Buffer{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(calls.count()).To(BeNumerically(">", 0), "the probe must see the re-parenting when it runs")
	})
}

// Test792b28c10a9_Targets_UpdateHelp_ListsEveryFlag verifies 792b28c10a9: engram update --help describes
// refreshing the binary and harness skills and lists the seven flags.
func Test792b28c10a9_Targets_UpdateHelp_ListsEveryFlag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	deps := newTestDeps(&stdout, &stderr)
	home := t.TempDir()
	deps.Getenv = func(string) string { return "" }
	deps.UserHomeDir = func() (string, error) { return home, nil }

	result, err := targ.Execute([]string{"engram", "update", "--help"}, cli.Targets(deps)...)
	g.Expect(err).NotTo(HaveOccurred())

	help := result.Output

	g.Expect(help).To(ContainSubstring("Refresh engram binary and harness skills"))

	for _, flag := range []string{
		"--dry-run", "--with-guidance", "--allow-downgrade", "--regen-vocab", "--reparent-luhmann", "--answers",
		"--backfill-identity",
	} {
		g.Expect(help).To(ContainSubstring(flag), "the help must list "+flag)
	}
}

// Test792b28c11a4_RunUpdate_FailedRun_RunsNoChecksAndWrapsTheError verifies 792b28c11a4: when Updater.Run returned
// an error, no check read the vault or the chunk index, no report was written, and the error comes back wrapped
// as "update: ...". The positive control proves the path spy sees the checks of a run that did not fail.
func Test792b28c11a4_RunUpdate_FailedRun_RunsNoChecksAndWrapsTheError(t *testing.T) {
	t.Parallel()

	vaultDataDir := reexecHome + "/.local/share/engram"

	t.Run("a run that failed reads neither the vault nor the chunk index", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		noHarness := newFakeUpdateFS() // no ~/.claude or ~/.pi: Updater.Run fails with update.ErrNoHarness
		spy := &pathSpyUpdateFS{Filesystem: noHarness}
		deps := cli.ExportNewUpdateDepsFrom(spy, stubCommander{}, reexecEnv{home: reexecHome, cwd: "/repo"})

		stdout := &bytes.Buffer{}
		err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{DryRun: true}, deps, stdout)

		g.Expect(err).To(MatchError(update.ErrNoHarness))
		g.Expect(err).To(MatchError(HavePrefix("update: ")))
		g.Expect(stdout.String()).To(BeEmpty(), "a failed run writes no report")

		for _, touched := range spy.snapshot() {
			g.Expect(touched).NotTo(HavePrefix(vaultDataDir), "a check read the vault or chunk index: %s", touched)
		}
	})

	t.Run("a run that succeeded does read the vault and the chunk index", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		spy := &pathSpyUpdateFS{Filesystem: singleClaudeHarnessFS()}
		deps := cli.ExportNewUpdateDepsFrom(spy, stubCommander{}, reexecEnv{home: reexecHome, cwd: "/repo"})

		err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{DryRun: true}, deps, &bytes.Buffer{})
		g.Expect(err).NotTo(HaveOccurred())

		touchedVault := false

		for _, touched := range spy.snapshot() {
			touchedVault = touchedVault || strings.HasPrefix(touched, vaultDataDir)
		}

		g.Expect(touchedVault).To(BeTrue(), "the checks must read the vault, or the spy proves nothing")
	})
}

// Test792b28c13a7_WriteUpdateReport_NoticesCoexist verifies 792b28c13a7: a report flagged for both chunk-index
// backlogs and old-format vocabulary files holds the empty-files, duplicates and old-format notices.
func Test792b28c13a7_WriteUpdateReport_NoticesCoexist(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{
		ChunkIndexHasEmptyFiles:         true,
		ChunkIndexHasPrunableDuplicates: true,
		VaultHasOldVocabFiles:           true,
	})
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("empty chunk-index files found"))
	g.Expect(out).To(ContainSubstring("duplicate chunk-index files found"))
	g.Expect(out).To(ContainSubstring("old-format vocab files found"))
}

// Test792b28c13a8_WriteUpdateReport_NoticesFollowAFixedOrder verifies 792b28c13a8: with every notice flagged, the
// notices appear as vocabulary, self-tag, Luhmann, empty files, duplicates, identity, pending offers, vault ID,
// outbox, registration error.
func Test792b28c13a8_WriteUpdateReport_NoticesFollowAFixedOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{
		VaultHasOldVocabFiles:            true,
		VaultHasUntaggedVocabDefinitions: true,
		VaultHasOnlyTopLevelNotes:        true,
		ChunkIndexHasEmptyFiles:          true,
		ChunkIndexHasPrunableDuplicates:  true,
		VaultHasNotesMissingIdentity:     true,
		VaultHasPendingOffers:            true,
		VaultIDUncommitted:               true,
		OutboxNotice:                     "parent outbox: 2 offer(s) queued\n",
		SkillRegistrationErr:             "registration failed",
	})
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	inOrder := []string{
		"old-format vocab files found",
		"missing their vocab/<term> self-tag",
		"vault holds only top-level notes",
		"empty chunk-index files found",
		"duplicate chunk-index files found",
		"notes missing repo:/user:/vault: provenance found",
		"vault holds pending offer(s) awaiting curation",
		"vault ID file .engram-vault-id is not committed",
		"parent outbox: 2 offer(s) queued",
		"skill registration error: registration failed",
	}

	previousIndex := -1

	for _, notice := range inOrder {
		index := strings.Index(out, notice)
		g.Expect(index).To(BeNumerically(">=", 0), "notice missing: %s", notice)
		g.Expect(index).To(BeNumerically(">", previousIndex), "notice out of order: %s", notice)

		previousIndex = index
	}
}

// Test792b28c14a2_WriteUpdateReport_DryRunRegenerationIsAPreview verifies 792b28c14a2: the counts of a
// --regen-vocab run, from a dry run, are phrased as "[dry-run] update --regen-vocab: would remove ...".
func Test792b28c14a2_WriteUpdateReport_DryRunRegenerationIsAPreview(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{
		DryRun:                    true,
		VocabRegenRan:             true,
		VocabRegenOldFilesRemoved: 2,
		VocabRegenMembersCleaned:  3,
		VocabRegenTermsSeeded:     1,
		VocabRegenNotesAssigned:   4,
	})
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("[dry-run] update --regen-vocab: would remove 2 old-format file(s), " +
		"cleaned 3 member note(s), seeded 1 term(s), reassigned 4 note(s)"))
	g.Expect(out).NotTo(ContainSubstring("update --regen-vocab: removed"))
}

// Test792b28c14a3_WriteUpdateReport_RegenerationWithNothingToDo verifies 792b28c14a3: a --regen-vocab run that
// removed and cleaned nothing prints "update --regen-vocab: nothing to regenerate".
func Test792b28c14a3_WriteUpdateReport_RegenerationWithNothingToDo(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{VocabRegenRan: true})
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("update --regen-vocab: nothing to regenerate"))
	g.Expect(out).NotTo(ContainSubstring("old-format file(s)"))
}

// Test792b28c14a5_WriteUpdateReport_DryRunBackfillIsAPreview verifies 792b28c14a5: a --backfill-identity dry run
// that would stamp 3 notes prints "[dry-run] update --backfill-identity: would stamp 3 note(s)".
func Test792b28c14a5_WriteUpdateReport_DryRunBackfillIsAPreview(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{
		DryRun: true, IdentityBackfillRan: true, IdentityBackfillNotesStamped: 3,
	})
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("[dry-run] update --backfill-identity: would stamp 3 note(s)"))
	g.Expect(out).NotTo(ContainSubstring("update --backfill-identity: stamped"))
}

// Test792b28c14a6_WriteUpdateReport_BackfillWithNothingToDo verifies 792b28c14a6: a --backfill-identity run that
// stamped nothing prints "update --backfill-identity: nothing to backfill" and not the identity notice it acted on.
func Test792b28c14a6_WriteUpdateReport_BackfillWithNothingToDo(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{
		IdentityBackfillRan: true, VaultHasNotesMissingIdentity: true,
	})
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("update --backfill-identity: nothing to backfill"))
	g.Expect(out).NotTo(ContainSubstring("provenance found"))
}

// Test792b28c15a7_FinishUpdate_AReportThatCannotBeWrittenIsAnError verifies 792b28c15a7: an output that fails on
// write makes the update return an error that begins "update: writing report:" and keeps the write error's chain.
func Test792b28c15a7_FinishUpdate_AReportThatCannotBeWrittenIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	errDiskFull := errors.New("disk full")
	report := update.Report{
		Source:    update.SourceInfo{Mode: update.SourceLocal, Root: "/r"},
		GoInstall: "go install ./cmd/engram/",
		Harnesses: []update.HarnessReport{{Name: "X"}},
	}

	err := cli.ExportFinishUpdate(si1FailingWriter{err: errDiskFull}, report, nil)

	g.Expect(err).To(MatchError(HavePrefix("update: writing report:")))
	g.Expect(err).To(MatchError(errDiskFull))
}

// unexported constants.
const (
	si1NoteBody = "---\ntype: fact\nluhmann: \"1\"\ncreated: 2026-01-01\n---\n\nthe same point about caching\n"
	si1NoteOne  = "1.2026-01-01.caching-one.md"
	si1NoteTwo  = "2.2026-01-01.caching-two.md"
)

// unexported variables.
var (
	errSi1EmbedderCalled = errors.New("the embedder was called")
)

// si1ArgsSpawner is an update.Spawner that records the arguments of every spawn and exits 0.
type si1ArgsSpawner struct {
	lock  sync.Mutex
	calls [][]string
}

func (s *si1ArgsSpawner) Run(_ string, args []string, _ []string) (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.calls = append(s.calls, append([]string(nil), args...))

	return 0, nil
}

func (s *si1ArgsSpawner) spawned() [][]string {
	s.lock.Lock()
	defer s.lock.Unlock()

	return append([][]string{}, s.calls...)
}

// si1CallCounter counts the calls the re-parenting's capabilities receive.
type si1CallCounter struct{ total atomic.Int32 }

func (c *si1CallCounter) count() int { return int(c.total.Load()) }

func (c *si1CallCounter) record() { c.total.Add(1) }

// si1CountingEmbedder is an embed.Embedder that counts every call as a call of the re-parenting.
type si1CountingEmbedder struct{ calls *si1CallCounter }

func (e si1CountingEmbedder) Dims() int {
	e.calls.record()

	return 0
}

func (e si1CountingEmbedder) Embed(context.Context, string) ([]float32, error) {
	e.calls.record()

	return nil, errSi1EmbedderCalled
}

func (e si1CountingEmbedder) ModelID() string {
	e.calls.record()

	return ""
}

// si1FailingWriter is an io.Writer whose every write fails with err.
type si1FailingWriter struct{ err error }

func (w si1FailingWriter) Write([]byte) (int, error) { return 0, w.err }

// si1ReparentDeps builds ReparentDeps whose every capability counts its call on calls.
func si1ReparentDeps(calls *si1CallCounter) cli.ReparentDeps {
	return cli.ReparentDeps{
		Rename: cli.RenameRewriteDeps{
			ListMD: func(string) ([]string, error) {
				calls.record()

				return []string{si1NoteOne, si1NoteTwo}, nil
			},
			ReadFile: func(string) ([]byte, error) {
				calls.record()

				return []byte(si1NoteBody), nil
			},
			WriteFile: func(string, []byte) error {
				calls.record()

				return nil
			},
			Rename: func(string, string) error {
				calls.record()

				return nil
			},
		},
		Embedder: si1CountingEmbedder{calls: calls},
	}
}

// si1VaultOfSimilarNotes is a local-mode update fixture whose vault holds two similar top-level notes.
func si1VaultOfSimilarNotes() *fakeUpdateFS {
	fileSystem := singleClaudeHarnessFS()
	vaultDir := reexecHome + "/.local/share/engram/vault/"
	fileSystem.files[vaultDir+si1NoteOne] = []byte(si1NoteBody)
	fileSystem.files[vaultDir+si1NoteTwo] = []byte(si1NoteBody)

	return fileSystem
}

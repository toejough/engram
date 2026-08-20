package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

func TestAnyHarnessFailed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportAnyHarnessFailed(update.Report{})).To(BeFalse())
	g.Expect(cli.ExportAnyHarnessFailed(update.Report{
		Harnesses: []update.HarnessReport{{}},
	})).To(BeFalse())
	g.Expect(cli.ExportAnyHarnessFailed(update.Report{
		Harnesses: []update.HarnessReport{{Err: errors.New("boom")}},
	})).To(BeTrue())
	g.Expect(cli.ExportAnyHarnessFailed(update.Report{
		Harnesses: []update.HarnessReport{{Err: errors.New("boom")}, {}},
	})).To(BeTrue())
}

func TestChunkIndexHasEmptyFiles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		files map[string][]byte
		want  bool
	}{
		{"one empty jsonl", map[string][]byte{"/chunks/a.jsonl": {}}, true},
		{"empty among nonempty", map[string][]byte{
			"/chunks/a.jsonl": []byte("x\n"), "/chunks/b.jsonl": {}}, true},
		{"all nonempty", map[string][]byte{"/chunks/a.jsonl": []byte("x\n")}, false},
		{"empty non-jsonl ignored", map[string][]byte{"/chunks/manifest.json": {}}, false},
		{"missing dir", map[string][]byte{}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t) // update_test.go dot-imports gomega — unqualified

			fileSystem := newU1FS()
			maps.Copy(fileSystem.files, tc.files)
			// Do NOT set dirs["/chunks"]: u1FS.hasChildren synthesizes the dir
			// from seeded /chunks/*.jsonl paths (mirror TestOldVocabFilesPresent),
			// so the "missing dir" case (empty maps) genuinely hits the
			// ReadDir-error (self-silencing) branch.

			g.Expect(cli.ExportChunkIndexHasEmptyFiles("/chunks", fileSystem)).To(Equal(tc.want))
		})
	}
}

// TestChunkIndexHasPrunableDuplicates pins the Unit 5 detect-and-notify gate
// directly: `engram update` must NOT run `prune --duplicates` itself (the
// user's explicit reversal of the earlier auto-run design), only detect
// whether running that command would actually remove anything, so the
// notice fires exactly when the command works (#713). Detection re-runs
// prune's own reconciliation (reconcileDuplicateGroups) in dry-run mode —
// the same (FileHash, chunkingClass) grouping, canonical selection, and
// record-level coverage gate — so a refusal-only backlog (every group's
// canonical missing or not verifiably covering its siblings' records)
// stays silent.
func TestChunkIndexHasPrunableDuplicates(t *testing.T) {
	t.Parallel()

	entry := func(hash string, dup ...string) map[string]any {
		e := map[string]any{"mtime_unix_nano": 1, "size": 10, "file_hash": hash}
		if len(dup) > 0 {
			e["duplicate_of"] = dup[0]
		}

		return e
	}

	rec := func(source, hash string) string {
		return `{"source":"` + source + `","anchor":"turn-1","content_hash":"` + hash + `","text":"t","vector":[1]}` + "\n"
	}

	cases := []struct {
		name     string
		manifest map[string]map[string]any // nil = no manifest.json file at all
		// indexFiles maps a source path to its per-source index file's
		// content, installed at ExportIndexPathFor("/chunks", source).
		indexFiles map[string]string
		want       bool
	}{
		{
			name:     "no manifest at all",
			manifest: nil,
			want:     false,
		},
		{
			name: "distinct hashes: nothing to clean up",
			manifest: map[string]map[string]any{
				"/repo/a.md": entry("sha256:x"),
				"/repo/b.md": entry("sha256:y"),
			},
			want: false,
		},
		{
			name: "shared hash but different chunking classes: never a duplicate group",
			manifest: map[string]map[string]any{
				"/repo/a.md":    entry("sha256:x"),
				"/repo/a.jsonl": entry("sha256:x"),
			},
			want: false,
		},
		{
			name: "already tagged duplicate_of by Unit 3's forward pass: nothing for the backlog to clean up",
			manifest: map[string]map[string]any{
				"/repo/a.md": entry("sha256:x"),
				"/repo/b.md": entry("sha256:x", "/repo/a.md"),
			},
			want: false,
		},
		{
			name: "group would be removed: duplicate never indexed (vacuously covered)",
			manifest: map[string]map[string]any{
				"/repo/a.md": entry("sha256:x"),
				"/repo/b.md": entry("sha256:x"),
			},
			indexFiles: map[string]string{
				"/repo/a.md": rec("/repo/a.md", "sha256:aaa"),
			},
			want: true,
		},
		{
			name: "group would be removed: duplicate's records covered by canonical",
			manifest: map[string]map[string]any{
				"/repo/a.md": entry("sha256:x"),
				"/repo/b.md": entry("sha256:x"),
			},
			indexFiles: map[string]string{
				"/repo/a.md": rec("/repo/a.md", "sha256:aaa") + rec("/repo/a.md", "sha256:bbb"),
				"/repo/b.md": rec("/repo/b.md", "sha256:aaa"),
			},
			want: true,
		},
		{
			name: "refusal-only, anomalous: canonical unindexed, sibling index survives",
			manifest: map[string]map[string]any{
				"/repo/a.md": entry("sha256:x"),
				"/repo/b.md": entry("sha256:x"),
			},
			indexFiles: map[string]string{
				"/repo/b.md": rec("/repo/b.md", "sha256:aaa"),
			},
			want: false,
		},
		{
			name: "refusal-only, structural: no member indexed",
			manifest: map[string]map[string]any{
				"/repo/a.md": entry("sha256:x"),
				"/repo/b.md": entry("sha256:x"),
			},
			want: false,
		},
		{
			name: "refusal-only: duplicate holds a record the canonical lacks",
			manifest: map[string]map[string]any{
				"/repo/a.md": entry("sha256:x"),
				"/repo/b.md": entry("sha256:x"),
			},
			indexFiles: map[string]string{
				"/repo/a.md": rec("/repo/a.md", "sha256:aaa"),
				"/repo/b.md": rec("/repo/b.md", "sha256:ccc"),
			},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fileSystem := newU1FS()

			if tc.manifest != nil {
				data, err := json.Marshal(tc.manifest)
				g.Expect(err).NotTo(HaveOccurred())

				fileSystem.files["/chunks/manifest.json"] = data
			}

			for source, content := range tc.indexFiles {
				fileSystem.files[cli.ExportIndexPathFor("/chunks", source)] = []byte(content)
			}

			g.Expect(cli.ExportChunkIndexHasPrunableDuplicates("/chunks", fileSystem)).To(Equal(tc.want))
		})
	}

	t.Run("malformed manifest: detection failure must not panic or propagate an error", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		fileSystem := newU1FS()
		fileSystem.files["/chunks/manifest.json"] = []byte("{not valid json")

		g.Expect(cli.ExportChunkIndexHasPrunableDuplicates("/chunks", fileSystem)).To(BeFalse())
	})
}

func TestDescribeSource_UnknownMode(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{})
	g.Expect(writeErr).NotTo(HaveOccurred())
	g.Expect(buffer.String()).To(ContainSubstring("source: unknown"))
}

func TestFinishUpdate_AnyHarnessFailureIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	report := update.Report{
		Harnesses: []update.HarnessReport{{Name: "X", Err: errors.New("disk")}},
	}

	err := cli.ExportFinishUpdate(&buffer, report, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("harness"))
}

func TestFinishUpdate_ExitStatusProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		count := rapid.IntRange(1, 5).Draw(rt, "count")
		fail := rapid.SliceOfN(rapid.Bool(), count, count).Draw(rt, "fail")

		harnesses := make([]update.HarnessReport, 0, count)
		anyFailed := false

		for i, failed := range fail {
			rep := update.HarnessReport{Name: update.Harness(rune('A' + i))}
			if failed {
				rep.Err = errors.New("boom")
				anyFailed = true
			}

			harnesses = append(harnesses, rep)
		}

		report := update.Report{
			Source:    update.SourceInfo{Mode: update.SourceLocal, Root: "/r"},
			GoInstall: "go install ./cmd/engram/",
			Harnesses: harnesses,
		}

		var buffer bytes.Buffer

		err := cli.ExportFinishUpdate(&buffer, report, nil)
		if anyFailed {
			if err == nil {
				rt.Fatalf("expected error when any harness failed, got nil")
			}
		} else if err != nil {
			rt.Fatalf("expected no error when no harness failed, got %v", err)
		}
	})
}

func TestFinishUpdate_HappyPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	report := update.Report{
		Source:    update.SourceInfo{Mode: update.SourceLocal, Root: "/r"},
		GoInstall: "go install ./cmd/engram/",
		Harnesses: []update.HarnessReport{{Name: "X"}},
	}

	err := cli.ExportFinishUpdate(&buffer, report, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestFinishUpdate_PartialFailureIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	report := update.Report{
		Source:    update.SourceInfo{Mode: update.SourceLocal, Root: "/r"},
		GoInstall: "go install ./cmd/engram/",
		Harnesses: []update.HarnessReport{
			{Name: "A", Err: errors.New("disk")},
			{Name: "B"},
		},
	}

	err := cli.ExportFinishUpdate(&buffer, report, nil)
	g.Expect(err).To(HaveOccurred())
	// Report still written so the user sees per-harness detail.
	g.Expect(buffer.String()).To(ContainSubstring("error: disk"))
}

func TestFinishUpdate_PropagatesRunError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	runErr := errors.New("boom")
	err := cli.ExportFinishUpdate(&buffer, update.Report{}, runErr)
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, runErr)).To(BeTrue())
}

func TestOldVocabFilesPresent(t *testing.T) {
	t.Parallel()

	table := []struct {
		name  string
		files map[string][]byte
		want  bool
	}{
		{
			name:  "vocab-index-present",
			files: map[string][]byte{"/vault/vocab.index.md": []byte("index")},
			want:  true,
		},
		{
			name:  "vocab-term-note-present",
			files: map[string][]byte{"/vault/vocab.recall.md": []byte("term note")},
			want:  true,
		},
		{
			name:  "no-old-vocab-files",
			files: map[string][]byte{"/vault/1.2026-07-01.some-note.md": []byte("note")},
			want:  false,
		},
		{
			name:  "missing-vault-dir",
			files: map[string][]byte{},
			want:  false,
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fileSystem := newU1FS()
			maps.Copy(fileSystem.files, tc.files)

			got := cli.ExportOldVocabFilesPresent("/vault", fileSystem)
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

func TestPluralFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportPluralFile(0)).To(Equal("files"))
	g.Expect(cli.ExportPluralFile(1)).To(Equal("file"))
	g.Expect(cli.ExportPluralFile(2)).To(Equal("files"))
}

func TestRunUpdate_DryRunFromCwd(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stdout := &bytes.Buffer{}
	deps := cli.ExportNewUpdateDepsFrom(liveUpdateFS{}, stubCommander{}, liveUpdateEnv{})

	// Dry-run against the live filesystem: cwd is inside the engram
	// worktree, so source resolution picks local mode without `go install`.
	err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{DryRun: true}, deps, stdout)
	out := stdout.String()

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("update"))

		return
	}

	g.Expect(out).To(ContainSubstring("[dry-run] engram update"))
	g.Expect(out).To(ContainSubstring("source: local clone at "))
}

// TestRunUpdate_RegenVocabFlag_RunsRegenAndUpdatesReport verifies runUpdate
// threads args.RegenVocab through to deps.Vocab (the updateDeps field
// newUpdateDeps composes from newVocabDeps, #712), reporting a --dry-run
// regen summary rather than the plain migration notice.
func TestRunUpdate_RegenVocabFlag_RunsRegenAndUpdatesReport(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files := map[string][]byte{
		"vocab.recall.md": []byte("---\ntype: fact\nterm: recall\ndescription: recall the vault\n---\n\nbody\n"),
	}

	vocabDeps := cli.VocabDeps{
		Lock:       func(string) (func(), error) { return func() {}, nil },
		ListMD:     func(string) ([]string, error) { return []string{"vocab.recall.md"}, nil },
		ReadFile:   func(path string) ([]byte, error) { return files[filepath.Base(path)], nil },
		WriteFile:  func(string, []byte) error { return nil },
		DeleteFile: func(string) error { return nil },
		LogWarning: func(string, ...any) {},
		Now:        time.Now,
	}

	deps := cli.ExportNewUpdateDepsFromWithVocab(liveUpdateFS{}, stubCommander{}, liveUpdateEnv{}, vocabDeps)

	stdout := &bytes.Buffer{}
	err := cli.ExportRunUpdate(
		context.Background(), cli.UpdateArgs{DryRun: true, RegenVocab: true}, deps, stdout)
	g.Expect(err).NotTo(HaveOccurred())

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("[dry-run] update --regen-vocab: would remove 1 old-format file(s)"))
	g.Expect(out).NotTo(ContainSubstring("old-format vocab files found"))
}

// TestRunUpdate_ReparentLuhmannFlag_ShortCircuitsToDeriveOnly verifies
// runUpdate routes --reparent-luhmann to RunReparentLuhmann BEFORE running
// Updater.Run (a nil update.Filesystem/Commander here would panic if the
// normal install/sync flow ran) and that derive-only output reaches stdout.
func TestRunUpdate_ReparentLuhmannFlag_ShortCircuitsToDeriveOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files := map[string][]byte{
		"/vault/1.2026-01-01.alpha.md": []byte("---\ntype: fact\nluhmann: \"1\"\ncreated: 2026-01-01\n---\n\nbody\n"),
	}

	reparentDeps := cli.ReparentDeps{
		Rename: cli.RenameRewriteDeps{
			ListMD:    func(string) ([]string, error) { return []string{"1.2026-01-01.alpha.md"}, nil },
			ReadFile:  func(path string) ([]byte, error) { return files[path], nil },
			WriteFile: func(string, []byte) error { return nil },
			Rename:    func(string, string) error { return nil },
		},
	}

	deps := cli.ExportNewUpdateDepsFromWithReparent(nil, nil, fixedHomeUpdateEnv{vault: "/vault"}, reparentDeps)

	stdout := &bytes.Buffer{}
	err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{ReparentLuhmann: true}, deps, stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("no candidates found"))
}

func TestRunUpdate_WithGuidanceFlagMapsToOptions(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	stdout := &bytes.Buffer{}
	deps := cli.ExportNewUpdateDepsFrom(liveUpdateFS{}, stubCommander{}, liveUpdateEnv{})

	// Dry-run with --with-guidance; only verifies the flag maps to Options.
	err := cli.ExportRunUpdate(
		context.Background(), cli.UpdateArgs{DryRun: true, WithGuidance: true}, deps, stdout)
	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("update"))
	}
}

func TestTildify(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportTildify("/home/joe/x", "/home/joe")).To(Equal("~/x"))
	g.Expect(cli.ExportTildify("/other/x", "/home/joe")).To(Equal("/other/x"))
	g.Expect(cli.ExportTildify("/home/joe/x", "")).To(Equal("/home/joe/x"))
}

// TestVaultHasOnlyTopLevelNotes pins vaultHasOnlyTopLevelNotes (task 1.1):
// true only when the vault holds at least one note and every note's Luhmann
// ID is depth 1 (top-level, no letter/digit branch segments); false for an
// empty or unreadable vault (self-silencing, same convention as
// oldVocabFilesPresent).
func TestVaultHasOnlyTopLevelNotes(t *testing.T) {
	t.Parallel()

	table := []struct {
		name  string
		files map[string][]byte
		want  bool
	}{
		{
			name: "all-top-level",
			files: map[string][]byte{
				"/vault/1.2026-07-01.first-note.md":  []byte("note one"),
				"/vault/2.2026-07-02.second-note.md": []byte("note two"),
			},
			want: true,
		},
		{
			name: "mixed-one-branched-note",
			files: map[string][]byte{
				"/vault/1.2026-07-01.first-note.md":   []byte("note one"),
				"/vault/1a.2026-07-02.branch-note.md": []byte("branched note"),
			},
			want: false,
		},
		{
			name:  "empty-vault",
			files: map[string][]byte{},
			want:  false,
		},
		{
			name: "non-note-files-skipped",
			files: map[string][]byte{
				"/vault/1.2026-07-01.first-note.md":       []byte("note one"),
				"/vault/1.2026-07-01.first-note.vec.json": []byte(`{}`),
			},
			want: true,
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fileSystem := newU1FS()
			maps.Copy(fileSystem.files, tc.files)

			got := cli.ExportVaultHasOnlyTopLevelNotes("/vault", fileSystem)
			g.Expect(got).To(Equal(tc.want))
		})
	}

	t.Run("unreadable-vault-dir", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		fileSystem := newU1FS()

		got := cli.ExportVaultHasOnlyTopLevelNotes("/missing-vault", fileSystem)
		g.Expect(got).To(BeFalse())
	})
}

// TestVocabDefinitionsMissingSelfTags pins the self-tag backfill detection
// for the vocab-definition-self-tags upgrade (4f68fada): a vault holding a
// definition note (bare `vocab` tag + "vocab-<term>-definition" slug) whose
// tags lack the vocab/<term> self-tag triggers the notice; the FAMILY note
// (slug "vocab-definition") is deliberately bare-vocab-only and must never
// trigger it; fully-tagged and fresh vaults stay silent (idempotent-clean).
func TestVocabDefinitionsMissingSelfTags(t *testing.T) {
	t.Parallel()

	untaggedDefinition := []byte("---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines recall-design.\n")
	taggedDefinition := []byte(
		"---\ntype: fact\ntags:\n    - vocab\n    - vocab/recall-design\n---\n\nDefines recall-design.\n")
	familyNote := []byte("---\ntype: fact\ntags:\n    - vocab\n---\n\nFamily root.\n")
	memberNote := []byte("---\ntype: fact\ntags:\n    - vocab/recall-design\n---\n\nMember mentions recall-design.\n")

	table := []struct {
		name  string
		files map[string][]byte
		want  bool
	}{
		{
			name: "untagged-definition-present",
			files: map[string][]byte{
				"/vault/12.2026-07-27.vocab-recall-design-definition.md": untaggedDefinition,
			},
			want: true,
		},
		{
			name: "fully-tagged-definition",
			files: map[string][]byte{
				"/vault/12.2026-07-27.vocab-recall-design-definition.md": taggedDefinition,
			},
			want: false,
		},
		{
			name: "family-note-is-exempt",
			files: map[string][]byte{
				"/vault/1.2026-07-10.vocab-definition.md": familyNote,
			},
			want: false,
		},
		{
			name: "member-note-is-not-a-definition",
			files: map[string][]byte{
				"/vault/3.2026-07-11.some-member.md": memberNote,
			},
			want: false,
		},
		{
			name: "definition-slug-without-bare-vocab-tag",
			files: map[string][]byte{
				"/vault/12.2026-07-27.vocab-recall-design-definition.md": memberNote,
			},
			want: false,
		},
		{
			name:  "missing-vault-dir",
			files: map[string][]byte{},
			want:  false,
		},
		{
			name: "mixed-vault-with-one-untagged",
			files: map[string][]byte{
				"/vault/1.2026-07-10.vocab-definition.md":                familyNote,
				"/vault/12.2026-07-27.vocab-recall-design-definition.md": taggedDefinition,
				"/vault/13.2026-07-27.vocab-eval-design-definition.md": []byte(
					"---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines eval-design.\n"),
			},
			want: true,
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fileSystem := newU1FS()
			maps.Copy(fileSystem.files, tc.files)

			got := cli.ExportVocabDefinitionsMissingSelfTags("/vault", fileSystem)
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

// TestWriteUpdateReport_DryRunContractSweep pins the unified D9 dry-run
// output contract (openspec/changes/update-deploy-as-sync tasks 7.1/7.2) in
// one pass over a fixture exercising every op class the sync/materialization
// engine reports: a sync deletion, a first-sync migration adoption, a
// dangling-link cleanup deletion, an already-imported guidance refresh, and
// a not-yet-imported guidance deploy hint. Every ACTION line (something that
// was or would be done) must carry the "[dry-run] " prefix; every NOTICE
// line (informational: refusal, unattributable, stray, or the static
// skill-row mapping) must NOT — pinning both #709's fix and its generalized
// sibling (the "guidance deployed to..." hint) without regressing the
// existing unprefixed-notice/unprefixed-mapping-row conventions.
func TestWriteUpdateReport_DryRunContractSweep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	report := update.Report{
		DryRun:           true,
		WithGuidance:     true,
		GuidanceImported: true,
		Home:             "/home/joe",
		Source:           update.SourceInfo{Mode: update.SourceLocal, Root: "/r"},
		GoInstall:        "go install ./cmd/engram/",
		GuidanceImports: map[update.Harness]map[string]bool{
			update.HarnessClaude: {"recall.md": true},
		},
		Harnesses: []update.HarnessReport{
			{
				Name:                  update.HarnessClaude,
				ProbeRoot:             ".claude",
				SkillsRoot:            "/home/joe/.claude/skills",
				GuidanceTargetRel:     ".claude/engram",
				ImportsFileRel:        ".claude/CLAUDE.md",
				EngramRoot:            "/home/joe/.claude/engram",
				SkillDirs:             []update.SkillDirCount{{Name: "learn", Files: 2}},
				EngramSyncDeleted:     []string{filepath.Join("guidance", "stale.md")},
				EngramDeletionRefused: true,
				EngramUnattributable:  []string{"stray.md"},
				EngramAdopted:         []string{"/home/joe/.claude/skills/recall"},
				SurfaceUnmanaged:      []string{"/home/joe/.claude/skills/legacy"},
				DanglingLinksRemoved:  []string{"/home/joe/.claude/skills/old-dangling"},
				GuidanceFiles:         []string{"recall.md", "learn.md"},
			},
		},
	}

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()

	actionLines := []string{
		"engram update",
		"installed: Claude Code",
		"engram root: deleted guidance/stale.md",
		"adopted: ~/.claude/skills/recall",
		"cleanup: removed dangling link ~/.claude/skills/old-dangling",
		"guidance refreshed: ~/.claude/engram/recall.md",
		"guidance deployed to ~/.claude/engram/learn.md",
	}
	for _, line := range actionLines {
		g.Expect(out).To(ContainSubstring("[dry-run] "+line), "action line must carry the dry-run prefix: %s", line)
	}

	noticeLines := []string{
		"exists without the .engram-owned marker",
		"unattributable: stray.md",
		"unmanaged (left alone): ~/.claude/skills/legacy",
		"agent-instructions/skills/learn/ → ~/.claude/skills/learn/  (2 files)",
	}
	for _, line := range noticeLines {
		g.Expect(out).To(ContainSubstring(line), "notice line must be present: %s", line)
		g.Expect(out).NotTo(ContainSubstring("[dry-run] "+line), "notice line must NOT carry the dry-run prefix: %s", line)
	}
}

// TestWriteUpdateReport_DuplicatesHint asserts Unit 5's detect-and-notify
// surface: when a prunable duplicate backlog was detected — one `engram
// prune --duplicates` would actually remove something from — the report
// names that command inline with no README pointer (#713), and update
// NEVER performs the removal itself (there is no removed/retained count
// to report; this is purely a notice).
func TestWriteUpdateReport_DuplicatesHint(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{ChunkIndexHasPrunableDuplicates: true})
	g.Expect(writeErr).NotTo(HaveOccurred())
	g.Expect(buffer.String()).To(ContainSubstring("run `engram prune --duplicates`"))
	g.Expect(buffer.String()).NotTo(ContainSubstring("Upgrading"))
	g.Expect(buffer.String()).NotTo(ContainSubstring("README.md"))

	var clean bytes.Buffer

	cleanErr := cli.ExportWriteUpdateReport(&clean, update.Report{ChunkIndexHasPrunableDuplicates: false})
	g.Expect(cleanErr).NotTo(HaveOccurred())
	g.Expect(clean.String()).NotTo(ContainSubstring("duplicate"))

	// Coexists with the other upgrade notices.
	var both bytes.Buffer

	bothErr := cli.ExportWriteUpdateReport(&both, update.Report{
		ChunkIndexHasEmptyFiles:         true,
		ChunkIndexHasPrunableDuplicates: true,
	})
	g.Expect(bothErr).NotTo(HaveOccurred())
	g.Expect(both.String()).To(ContainSubstring("empty chunk-index"))
	g.Expect(both.String()).To(ContainSubstring("prune --duplicates"))
}

func TestWriteUpdateReport_EmptyChunkHint(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{ChunkIndexHasEmptyFiles: true})
	g.Expect(writeErr).NotTo(HaveOccurred())
	g.Expect(buffer.String()).To(ContainSubstring("run `engram prune --empty`"))
	g.Expect(buffer.String()).NotTo(ContainSubstring("Upgrading"))
	g.Expect(buffer.String()).NotTo(ContainSubstring("README.md"))

	var clean bytes.Buffer

	cleanErr := cli.ExportWriteUpdateReport(&clean, update.Report{ChunkIndexHasEmptyFiles: false})
	g.Expect(cleanErr).NotTo(HaveOccurred())
	g.Expect(clean.String()).NotTo(ContainSubstring("empty chunk-index"))

	// Both notices coexist when both conditions hold.
	var both bytes.Buffer

	bothErr := cli.ExportWriteUpdateReport(&both, update.Report{
		VaultHasOldVocabFiles:   true,
		ChunkIndexHasEmptyFiles: true,
	})
	g.Expect(bothErr).NotTo(HaveOccurred())
	g.Expect(both.String()).To(ContainSubstring("empty chunk-index"))
	g.Expect(both.String()).To(ContainSubstring("old-format vocab"))
}

func TestWriteUpdateReport_EngramRootNotice(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	report := update.Report{
		Home: "/home/joe",
		Harnesses: []update.HarnessReport{
			{
				Name:                  update.HarnessClaude,
				ProbeRoot:             ".claude",
				SkillsRoot:            "/home/joe/.claude/skills",
				EngramRoot:            "/home/joe/.claude/engram",
				EngramSyncDeleted:     []string{filepath.Join("skills", "oldskill", "SKILL.md")},
				EngramUnattributable:  []string{"mystery.md"},
				EngramDeletionRefused: true,
			},
		},
	}

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("engram root: deleted skills/oldskill/SKILL.md"))
	g.Expect(out).To(ContainSubstring("engram root ~/.claude/engram exists without the .engram-owned marker"))
	g.Expect(out).To(ContainSubstring("unattributable: mystery.md"))
}

// TestWriteUpdateReport_EngramRootNotice_AdoptedAndSurfaceStray covers task
// 5.1's EngramAdopted field and task 5.2's SurfaceUnmanaged
// listing — both must render (D6's spec requirement that a stray is
// "listed in the update report", not just tracked internally).
func TestWriteUpdateReport_EngramRootNotice_AdoptedAndSurfaceStray(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	report := update.Report{
		Home: "/home/joe",
		Harnesses: []update.HarnessReport{
			{
				Name:             update.HarnessClaude,
				ProbeRoot:        ".claude",
				SkillsRoot:       "/home/joe/.claude/skills",
				EngramRoot:       "/home/joe/.claude/engram",
				EngramAdopted:    []string{"/home/joe/.claude/skills/recall"},
				SurfaceUnmanaged: []string{"/home/joe/.claude/skills/user-own-tool"},
			},
		},
	}

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("adopted: ~/.claude/skills/recall"))
	g.Expect(out).To(ContainSubstring("unmanaged (left alone): ~/.claude/skills/user-own-tool"))
}

func TestWriteUpdateReport_EngramRootNotice_AdoptedAndSurfaceStray_DryRunPrefixesAdopted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	report := update.Report{
		DryRun: true,
		Home:   "/home/joe",
		Harnesses: []update.HarnessReport{
			{
				Name:          update.HarnessClaude,
				ProbeRoot:     ".claude",
				SkillsRoot:    "/home/joe/.claude/skills",
				EngramRoot:    "/home/joe/.claude/engram",
				EngramAdopted: []string{"/home/joe/.claude/skills/recall"},
			},
		},
	}

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())
	g.Expect(buffer.String()).To(ContainSubstring("[dry-run] adopted: ~/.claude/skills/recall"))
}

func TestWriteUpdateReport_EngramRootNotice_DryRunPrefixesDeletions(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	report := update.Report{
		DryRun: true,
		Home:   "/home/joe",
		Harnesses: []update.HarnessReport{
			{
				Name:              update.HarnessClaude,
				ProbeRoot:         ".claude",
				SkillsRoot:        "/home/joe/.claude/skills",
				EngramRoot:        "/home/joe/.claude/engram",
				EngramSyncDeleted: []string{filepath.Join("guidance", "stale.md")},
			},
		},
	}

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())
	g.Expect(buffer.String()).To(ContainSubstring("[dry-run] engram root: deleted guidance/stale.md"))
}

func TestWriteUpdateReport_EngramRootNotice_SilentWhenNothingToReport(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	report := update.Report{
		Home:      "/home/joe",
		Harnesses: []update.HarnessReport{claudeHarnessReport()},
	}

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())
	g.Expect(buffer.String()).NotTo(ContainSubstring("engram root"))
	g.Expect(buffer.String()).NotTo(ContainSubstring("unattributable"))
}

func TestWriteUpdateReport_GuidanceActivationHint(t *testing.T) {
	t.Parallel()

	table := []struct {
		name             string
		harnesses        []update.HarnessReport
		guidanceImports  map[update.Harness]map[string]bool
		guidanceImported bool
		withGuidance     bool
		wantContains     []string
		wantNotContains  []string
	}{
		{
			name:         "claude-deployed-not-imported",
			harnesses:    []update.HarnessReport{claudeHarnessReport("recall.md")},
			withGuidance: true,
			wantContains: []string{
				"guidance deployed to ~/.claude/engram/recall.md — add" +
					" '@~/.claude/engram/recall.md' to ~/.claude/CLAUDE.md to activate it" +
					" (Claude Code will ask you to approve the import once)",
			},
		},
		{
			name:      "claude-deployed-and-imported",
			harnesses: []update.HarnessReport{claudeHarnessReport("recall.md")},
			guidanceImports: map[update.Harness]map[string]bool{
				update.HarnessClaude: {"recall.md": true},
			},
			guidanceImported: true,
			withGuidance:     true,
			wantContains:     []string{"guidance refreshed: ~/.claude/engram/recall.md"},
			wantNotContains:  []string{"add '@~/.claude/engram/recall.md'"},
		},
		{
			name:         "pi-deployed-not-imported",
			harnesses:    []update.HarnessReport{piHarnessReport("recall.md")},
			withGuidance: true,
			wantContains: []string{
				"guidance deployed to ~/.pi/agent/guidance/recall.md — add" +
					" '@~/.pi/agent/guidance/recall.md' to ~/.pi/agent/AGENTS.md to activate it" +
					" (Pi will ask you to approve the import once)",
			},
		},
		{
			name:      "pi-deployed-and-imported",
			harnesses: []update.HarnessReport{piHarnessReport("recall.md")},
			guidanceImports: map[update.Harness]map[string]bool{
				update.HarnessPi: {"recall.md": true},
			},
			guidanceImported: true,
			wantContains:     []string{"guidance refreshed: ~/.pi/agent/guidance/recall.md"},
			wantNotContains:  []string{"add '@~/.pi/agent/guidance/recall.md'"},
		},
		{
			name: "per-harness-independent-wiring",
			harnesses: []update.HarnessReport{
				claudeHarnessReport("recall.md"),
				piHarnessReport("recall.md"),
			},
			guidanceImports: map[update.Harness]map[string]bool{
				update.HarnessClaude: {"recall.md": true},
			},
			guidanceImported: true,
			wantContains: []string{
				"guidance refreshed: ~/.claude/engram/recall.md",
				"add '@~/.pi/agent/guidance/recall.md' to ~/.pi/agent/AGENTS.md",
			},
			wantNotContains: []string{"add '@~/.claude/engram/recall.md'"},
		},
		{
			name:         "plain-update-not-imported",
			harnesses:    []update.HarnessReport{claudeHarnessReport()},
			wantContains: []string{"engram ships recall- and delegation-firing guidance"},
		},
		{
			name:             "plain-update-already-imported",
			harnesses:        []update.HarnessReport{claudeHarnessReport()},
			guidanceImported: true,
			wantNotContains:  []string{"engram ships", "activate it"},
		},
		{
			name:      "mixed-recall-imported-delegate-not",
			harnesses: []update.HarnessReport{claudeHarnessReport("recall.md", "delegate.md")},
			guidanceImports: map[update.Harness]map[string]bool{
				update.HarnessClaude: {"recall.md": true},
			},
			guidanceImported: true,
			wantContains: []string{
				"guidance refreshed: ~/.claude/engram/recall.md",
				"@~/.claude/engram/delegate.md",
			},
			wantNotContains: []string{"add '@~/.claude/engram/recall.md'"},
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			report := update.Report{
				DryRun:           false,
				WithGuidance:     tc.withGuidance,
				GuidanceImported: tc.guidanceImported,
				GuidanceImports:  tc.guidanceImports,
				Home:             "/home/joe",
				Source:           update.SourceInfo{Mode: update.SourceLocal, Root: "/r"},
				GoInstall:        "go install ./cmd/engram/",
				Harnesses:        tc.harnesses,
			}

			var buffer bytes.Buffer

			writeErr := cli.ExportWriteUpdateReport(&buffer, report)
			g.Expect(writeErr).NotTo(HaveOccurred())

			out := buffer.String()

			for _, s := range tc.wantContains {
				g.Expect(out).To(ContainSubstring(s))
			}

			for _, s := range tc.wantNotContains {
				g.Expect(out).NotTo(ContainSubstring(s))
			}
		})
	}
}

// TestWriteUpdateReport_GuidanceCanonicalPathHint covers task 5.3: when a
// harness's guidance surface IS its engram root (Claude today — D1's
// guidance caveat), the report states the canonical guidance/ path
// alongside the flat @import path, since the flat path is now a compat
// symlink rather than where the real content lives.
func TestWriteUpdateReport_GuidanceCanonicalPathHint(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	report := update.Report{
		Home:         "/home/joe",
		WithGuidance: true,
		Harnesses: []update.HarnessReport{
			{
				Name:              update.HarnessClaude,
				ProbeRoot:         ".claude",
				SkillsRoot:        "/home/joe/.claude/skills",
				GuidanceTargetRel: ".claude/engram",
				ImportsFileRel:    ".claude/CLAUDE.md",
				EngramRoot:        "/home/joe/.claude/engram",
				GuidanceFiles:     []string{"recall.md"},
			},
		},
	}

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())
	g.Expect(buffer.String()).To(ContainSubstring("(canonical: ~/.claude/engram/guidance/recall.md)"))
}

// TestWriteUpdateReport_GuidanceCanonicalPathHint_PiOmitted covers the
// negative: Pi's guidance dir is a surface distinct from its engram root, so
// no canonical-path suffix is rendered — the flat/deployed path already IS
// the canonical one for Pi.
func TestWriteUpdateReport_GuidanceCanonicalPathHint_PiOmitted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	report := update.Report{
		Home:      "/home/joe",
		Harnesses: []update.HarnessReport{piHarnessReport("recall.md")},
	}

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())
	g.Expect(buffer.String()).NotTo(ContainSubstring("canonical:"))
}

// TestWriteUpdateReport_LuhmannBranchingHint pins the detect-and-notify
// surface for a flat (all-top-level) vault: when VaultHasOnlyTopLevelNotes
// is true, the report names `engram update --reparent-luhmann` inline; a
// vault that already has branched notes prints nothing. This is a read-only
// notice — nothing here ever mutates a vault note.
func TestWriteUpdateReport_IdentityBackfillHint(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{VaultHasNotesMissingIdentity: true})
	g.Expect(writeErr).NotTo(HaveOccurred())
	g.Expect(buffer.String()).To(ContainSubstring("engram update --backfill-identity"))

	var clean bytes.Buffer

	cleanErr := cli.ExportWriteUpdateReport(&clean, update.Report{VaultHasNotesMissingIdentity: false})
	g.Expect(cleanErr).NotTo(HaveOccurred())
	g.Expect(clean.String()).NotTo(ContainSubstring("backfill-identity"))
}

// TestWriteUpdateReport_IdentityBackfillReport verifies the --backfill-identity
// summary (run) supersedes the plain notice, mirroring
// TestWriteUpdateReport_VocabRegenReport's shape.
func TestWriteUpdateReport_IdentityBackfillReport(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var nothing bytes.Buffer

	nothingErr := cli.ExportWriteUpdateReport(&nothing, update.Report{
		IdentityBackfillRan: true, VaultHasNotesMissingIdentity: true,
	})
	g.Expect(nothingErr).NotTo(HaveOccurred())
	g.Expect(nothing.String()).To(ContainSubstring("update --backfill-identity: nothing to backfill"))
	g.Expect(nothing.String()).NotTo(ContainSubstring("provenance found"))

	var stamped bytes.Buffer

	stampedErr := cli.ExportWriteUpdateReport(&stamped, update.Report{
		IdentityBackfillRan: true, IdentityBackfillNotesStamped: 3,
	})
	g.Expect(stampedErr).NotTo(HaveOccurred())
	g.Expect(stamped.String()).To(ContainSubstring("update --backfill-identity: stamped 3 note(s)"))

	var dryRun bytes.Buffer

	dryRunErr := cli.ExportWriteUpdateReport(&dryRun, update.Report{
		DryRun: true, IdentityBackfillRan: true, IdentityBackfillNotesStamped: 3,
	})
	g.Expect(dryRunErr).NotTo(HaveOccurred())
	g.Expect(dryRun.String()).To(ContainSubstring("[dry-run] update --backfill-identity: would stamp 3 note(s)"))
}

func TestWriteUpdateReport_LocalDryRunWithBothHarnesses(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	report := update.Report{
		DryRun:    true,
		Home:      "/home/joe",
		Source:    update.SourceInfo{Mode: update.SourceLocal, Root: "/home/joe/src/engram"},
		GoInstall: "go install ./cmd/engram/",
		Harnesses: []update.HarnessReport{
			{
				Name:       update.HarnessClaude,
				ProbeRoot:  ".claude",
				SkillsRoot: "/home/joe/.claude/skills",
				SkillDirs: []update.SkillDirCount{
					{Name: "learn", Files: 3},
					{Name: "recall", Files: 1},
				},
			},
			{
				Name:       update.HarnessPi,
				ProbeRoot:  ".pi",
				SkillsRoot: "/home/joe/.pi/agent/skills",
				SkillDirs: []update.SkillDirCount{
					{Name: "learn", Files: 3},
					{Name: "recall", Files: 1},
				},
			},
		},
	}

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("[dry-run] engram update"))
	g.Expect(out).To(ContainSubstring("source: local clone at ~/src/engram"))
	g.Expect(out).To(ContainSubstring("binary: go install ./cmd/engram/"))
	g.Expect(out).To(ContainSubstring("Claude Code (~/.claude/):"))
	g.Expect(out).To(ContainSubstring("agent-instructions/skills/learn/ → ~/.claude/skills/learn/  (3 files)"))
	g.Expect(out).To(ContainSubstring("agent-instructions/skills/recall/ → ~/.claude/skills/recall/  (1 file)"))
	g.Expect(out).To(ContainSubstring("Pi (~/.pi/):"))
	g.Expect(out).To(ContainSubstring("[dry-run] installed: Claude Code, Pi"))
}

// TestWriteUpdateReport_LocalSourceIncludesRevision covers
// update-local-install-safety task 1: local mode's resolved revision must
// show up in the report's source description alongside the clone path.
func TestWriteUpdateReport_LocalSourceIncludesRevision(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	report := update.Report{
		Home:       "/home/joe",
		Source:     update.SourceInfo{Mode: update.SourceLocal, Root: "/home/joe/src/engram", Version: "abc1234"},
		GoInstall:  "go install ./cmd/engram/",
		BinaryPath: "/home/joe/go/bin/engram",
		Harnesses: []update.HarnessReport{
			{
				Name:       update.HarnessClaude,
				ProbeRoot:  ".claude",
				SkillsRoot: "/home/joe/.claude/skills",
			},
		},
	}

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("source: local clone at ~/src/engram (rev abc1234)"))
}

func TestWriteUpdateReport_LuhmannBranchingHint(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{VaultHasOnlyTopLevelNotes: true})
	g.Expect(writeErr).NotTo(HaveOccurred())
	g.Expect(buffer.String()).To(ContainSubstring("engram update --reparent-luhmann"))

	var clean bytes.Buffer

	cleanErr := cli.ExportWriteUpdateReport(&clean, update.Report{VaultHasOnlyTopLevelNotes: false})
	g.Expect(cleanErr).NotTo(HaveOccurred())
	g.Expect(clean.String()).NotTo(ContainSubstring("reparent-luhmann"))
}

func TestWriteUpdateReport_RealRunLocalNoVersion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	report := update.Report{
		Home:       "/home/joe",
		Source:     update.SourceInfo{Mode: update.SourceLocal, Root: "/home/joe/src/engram"},
		GoInstall:  "go install ./cmd/engram/",
		BinaryPath: "/home/joe/go/bin/engram",
		Harnesses: []update.HarnessReport{
			{
				Name:       update.HarnessClaude,
				ProbeRoot:  ".claude",
				SkillsRoot: "/home/joe/.claude/skills",
				SkillDirs:  []update.SkillDirCount{{Name: "learn", Files: 3}},
			},
		},
	}

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).
		To(ContainSubstring("binary: go install ./cmd/engram/ ... ok (engram → ~/go/bin/engram)"))
}

func TestWriteUpdateReport_RealRunRemoteVersionAndBinary(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	report := update.Report{
		Home:          "/home/joe",
		Source:        update.SourceInfo{Mode: update.SourceRemote, Version: "v0.2.0"},
		GoInstall:     "go install github.com/toejough/engram/cmd/engram@latest",
		BinaryPath:    "/home/joe/go/bin/engram",
		BinaryVersion: "v0.2.0",
		Harnesses: []update.HarnessReport{
			{
				Name:       update.HarnessClaude,
				ProbeRoot:  ".claude",
				SkillsRoot: "/home/joe/.claude/skills",
			},
		},
	}

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("ok (engram v0.2.0 → ~/go/bin/engram)"))
	g.Expect(out).To(ContainSubstring("source: remote module github.com/toejough/engram v0.2.0"))
}

func TestWriteUpdateReport_RemoteHarnessFailure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	report := update.Report{
		Source:    update.SourceInfo{Mode: update.SourceRemote, Version: "v0.2.0"},
		GoInstall: "go install github.com/toejough/engram/cmd/engram@latest",
		Harnesses: []update.HarnessReport{
			{
				Name:       update.HarnessClaude,
				SkillsRoot: "/home/joe/.claude/skills",
				Err:        errors.New("disk full"),
			},
		},
	}

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, report)
	g.Expect(writeErr).NotTo(HaveOccurred())

	out := buffer.String()
	g.Expect(out).To(ContainSubstring("source: remote module github.com/toejough/engram v0.2.0"))
	g.Expect(out).To(ContainSubstring("error: disk full"))
	g.Expect(out).NotTo(ContainSubstring("installed:"))
}

func TestWriteUpdateReport_VocabMigrationHint(t *testing.T) {
	t.Parallel()

	table := []struct {
		name            string
		hasOldVocab     bool
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:            "old-vocab-present",
			hasOldVocab:     true,
			wantContains:    []string{"run `engram update --regen-vocab`"},
			wantNotContains: []string{"Upgrading", "README.md"},
		},
		{
			name:            "no-old-vocab",
			hasOldVocab:     false,
			wantNotContains: []string{"old-format vocab"},
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			report := update.Report{
				Home:                  "/home/joe",
				Source:                update.SourceInfo{Mode: update.SourceLocal, Root: "/r"},
				GoInstall:             "go install ./cmd/engram/",
				VaultHasOldVocabFiles: tc.hasOldVocab,
				Harnesses: []update.HarnessReport{
					{
						Name:       update.HarnessClaude,
						ProbeRoot:  ".claude",
						SkillsRoot: "/home/joe/.claude/skills",
					},
				},
			}

			var buffer bytes.Buffer

			writeErr := cli.ExportWriteUpdateReport(&buffer, report)
			g.Expect(writeErr).NotTo(HaveOccurred())

			if writeErr != nil {
				return
			}

			out := buffer.String()

			for _, s := range tc.wantContains {
				g.Expect(out).To(ContainSubstring(s))
			}

			for _, s := range tc.wantNotContains {
				g.Expect(out).NotTo(ContainSubstring(s))
			}
		})
	}
}

// TestWriteUpdateReport_VocabRegenReport pins #712's --regen-vocab report
// rendering: VocabRegenRan renders the regen summary INSTEAD of the plain
// migration notice (a run that just regenerated never repeats the notice it
// acted on), a --dry-run regen reports "would remove", a real regen reports
// "removed", and a regen that found nothing to do prints a cheap no-op line.
func TestWriteUpdateReport_VocabRegenReport(t *testing.T) {
	t.Parallel()

	table := []struct {
		name            string
		report          update.Report
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "real-run-removed",
			report: update.Report{
				VocabRegenRan:             true,
				VocabRegenOldFilesRemoved: 2,
				VocabRegenMembersCleaned:  1,
				VocabRegenTermsSeeded:     1,
				VocabRegenNotesAssigned:   3,
			},
			wantContains: []string{
				"update --regen-vocab: removed 2 old-format file(s)",
				"cleaned 1 member note(s)",
				"seeded 1 term(s)",
				"reassigned 3 note(s)",
			},
			wantNotContains: []string{"old-format vocab files found", "would remove"},
		},
		{
			name: "dry-run-would-remove",
			report: update.Report{
				DryRun:                    true,
				VocabRegenRan:             true,
				VocabRegenOldFilesRemoved: 2,
				VocabRegenMembersCleaned:  1,
				VocabRegenTermsSeeded:     1,
			},
			wantContains:    []string{"[dry-run] update --regen-vocab: would remove 2 old-format file(s)"},
			wantNotContains: []string{"old-format vocab files found", "removed 2"},
		},
		{
			name:            "nothing-to-regenerate",
			report:          update.Report{VocabRegenRan: true},
			wantContains:    []string{"update --regen-vocab: nothing to regenerate"},
			wantNotContains: []string{"old-format vocab files found"},
		},
		{
			name:            "regen-ran-never-repeats-plain-notice",
			report:          update.Report{VocabRegenRan: true, VaultHasOldVocabFiles: true},
			wantNotContains: []string{"old-format vocab files found"},
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			var buffer bytes.Buffer

			writeErr := cli.ExportWriteUpdateReport(&buffer, tc.report)
			g.Expect(writeErr).NotTo(HaveOccurred())

			out := buffer.String()
			for _, want := range tc.wantContains {
				g.Expect(out).To(ContainSubstring(want))
			}

			for _, notWant := range tc.wantNotContains {
				g.Expect(out).NotTo(ContainSubstring(notWant))
			}
		})
	}
}

// TestWriteUpdateReport_VocabSelfTagHint pins the detect-and-notify surface
// for the vocab-definition-self-tags upgrade (4f68fada): when the vault
// holds definition notes missing their vocab/<term> self-tag, the report
// names `engram vocab tag-definitions` inline; a fully-tagged vault prints
// nothing (idempotent-clean); the notice coexists with the other hints.
func TestWriteUpdateReport_VocabSelfTagHint(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{VaultHasUntaggedVocabDefinitions: true})
	g.Expect(writeErr).NotTo(HaveOccurred())
	g.Expect(buffer.String()).To(ContainSubstring("run `engram vocab tag-definitions`"))
	g.Expect(buffer.String()).NotTo(ContainSubstring("Upgrading"))
	g.Expect(buffer.String()).NotTo(ContainSubstring("README.md"))

	var clean bytes.Buffer

	cleanErr := cli.ExportWriteUpdateReport(&clean, update.Report{VaultHasUntaggedVocabDefinitions: false})
	g.Expect(cleanErr).NotTo(HaveOccurred())
	g.Expect(clean.String()).NotTo(ContainSubstring("self-tag"))

	// Coexists with the other upgrade notices.
	var both bytes.Buffer

	bothErr := cli.ExportWriteUpdateReport(&both, update.Report{
		VaultHasOldVocabFiles:            true,
		VaultHasUntaggedVocabDefinitions: true,
	})
	g.Expect(bothErr).NotTo(HaveOccurred())
	g.Expect(both.String()).To(ContainSubstring("old-format vocab"))
	g.Expect(both.String()).To(ContainSubstring("vocab tag-definitions"))
}

// fixedHomeUpdateEnv is a fake update.Env with a fixed home dir and
// ENGRAM_VAULT_PATH override, so --reparent-luhmann wiring tests don't touch
// the real filesystem or home directory.
type fixedHomeUpdateEnv struct{ vault string }

func (e fixedHomeUpdateEnv) Getenv(key string) string {
	if key == "ENGRAM_VAULT_PATH" {
		return e.vault
	}

	return ""
}

func (fixedHomeUpdateEnv) Getwd() (string, error) { return "/cwd", nil }

func (fixedHomeUpdateEnv) UserHomeDir() (string, error) { return "/home/test", nil }

// liveUpdateEnv adapts the real process environment to update.Env for the
// dry-run smoke tests (production Env is composed from cli.Deps).
type liveUpdateEnv struct{}

func (liveUpdateEnv) Getenv(key string) string { return os.Getenv(key) }

func (liveUpdateEnv) Getwd() (string, error) {
	return os.Getwd() // test adapter
}

func (liveUpdateEnv) UserHomeDir() (string, error) {
	return os.UserHomeDir() // test adapter
}

// liveUpdateFS is an os-backed update.Filesystem for the dry-run smoke
// tests (dry-run never writes; write methods exist to satisfy the interface).
type liveUpdateFS struct{}

func (liveUpdateFS) Lstat(path string) (update.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err // errors.Is(fs.ErrNotExist) must survive
	}

	return info, nil
}

func (liveUpdateFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm) // test adapter
}

func (liveUpdateFS) ReadDir(path string) ([]update.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err // errors.Is(fs.ErrNotExist) must survive
	}

	out := make([]update.DirEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry)
	}

	return out, nil
}

func (liveUpdateFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path) // test adapter; test-chosen paths
}

func (liveUpdateFS) ReadLink(path string) (string, error) {
	return os.Readlink(path) // test adapter
}

func (liveUpdateFS) RemoveAll(path string) error {
	return os.RemoveAll(path) // test adapter
}

func (liveUpdateFS) Stat(path string) (update.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err // errors.Is(fs.ErrNotExist) must survive
	}

	return info, nil
}

func (liveUpdateFS) Symlink(target, link string) error {
	return os.Symlink(target, link) // test adapter
}

func (liveUpdateFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm) // test adapter
}

// stubCommander satisfies update.Commander; dry-run local mode only uses it
// for revision resolution (git rev-parse), never go install.
type stubCommander struct{}

func (stubCommander) Run(context.Context, string, string, ...string) ([]byte, []byte, error) {
	return nil, nil, nil
}

// claudeHarnessReport builds a Claude Code HarnessReport with spec-derived
// guidance paths, deploying the given guidance basenames.
func claudeHarnessReport(guidanceFiles ...string) update.HarnessReport {
	return update.HarnessReport{
		Name:              update.HarnessClaude,
		ProbeRoot:         ".claude",
		SkillsRoot:        "/home/joe/.claude/skills",
		GuidanceTargetRel: ".claude/engram",
		ImportsFileRel:    ".claude/CLAUDE.md",
		GuidanceFiles:     guidanceFiles,
	}
}

// piHarnessReport builds a Pi HarnessReport with spec-derived guidance paths,
// deploying the given guidance basenames.
func piHarnessReport(guidanceFiles ...string) update.HarnessReport {
	return update.HarnessReport{
		Name:              update.HarnessPi,
		ProbeRoot:         ".pi",
		SkillsRoot:        "/home/joe/.pi/agent/skills",
		GuidanceTargetRel: ".pi/agent/guidance",
		ImportsFileRel:    ".pi/agent/AGENTS.md",
		GuidanceFiles:     guidanceFiles,
	}
}

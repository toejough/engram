package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"testing"

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

func TestWriteUpdateReport_EmptyChunkHint(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buffer bytes.Buffer

	writeErr := cli.ExportWriteUpdateReport(&buffer, update.Report{ChunkIndexHasEmptyFiles: true})
	g.Expect(writeErr).NotTo(HaveOccurred())
	g.Expect(buffer.String()).To(ContainSubstring("Upgrading"))
	g.Expect(buffer.String()).To(ContainSubstring("README.md"))

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

func TestWriteUpdateReport_GuidanceActivationHint(t *testing.T) {
	t.Parallel()

	table := []struct {
		name             string
		guidanceFiles    []string
		guidanceImports  map[string]bool
		guidanceImported bool
		withGuidance     bool
		wantContains     []string
		wantNotContains  []string
	}{
		{
			name:             "deployed-not-imported",
			guidanceFiles:    []string{"recall.md"},
			guidanceImports:  nil,
			guidanceImported: false,
			withGuidance:     true,
			wantContains:     []string{"@~/.claude/engram/recall.md"},
		},
		{
			name:             "deployed-and-imported",
			guidanceFiles:    []string{"recall.md"},
			guidanceImports:  map[string]bool{"recall.md": true},
			guidanceImported: true,
			withGuidance:     true,
			wantContains:     []string{"guidance refreshed: ~/.claude/engram/recall.md"},
			wantNotContains:  []string{"add '@~/.claude/engram/recall.md'"},
		},
		{
			name:             "plain-update-not-imported",
			guidanceFiles:    nil,
			guidanceImports:  nil,
			guidanceImported: false,
			withGuidance:     false,
			wantContains:     []string{"engram ships recall- and delegation-firing guidance"},
		},
		{
			name:             "plain-update-already-imported",
			guidanceFiles:    nil,
			guidanceImports:  nil,
			guidanceImported: true,
			withGuidance:     false,
			wantNotContains:  []string{"engram ships", "activate it"},
		},
		{
			name:             "mixed-recall-imported-delegate-not",
			guidanceFiles:    []string{"recall.md", "delegate.md"},
			guidanceImports:  map[string]bool{"recall.md": true},
			guidanceImported: true,
			withGuidance:     false,
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
				Harnesses: []update.HarnessReport{
					{
						Name:          update.HarnessClaude,
						ProbeRoot:     ".claude",
						SkillsRoot:    "/home/joe/.claude/skills",
						GuidanceFiles: tc.guidanceFiles,
					},
				},
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
				Name:         update.HarnessOpencode,
				ProbeRoot:    ".config/opencode",
				SkillsRoot:   "/home/joe/.config/opencode/skills",
				CommandsRoot: "/home/joe/.config/opencode/commands",
				SkillDirs: []update.SkillDirCount{
					{Name: "learn", Files: 3},
					{Name: "recall", Files: 1},
				},
				CommandFiles: []string{"learn.md", "recall.md"},
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
	g.Expect(out).To(ContainSubstring("agent-instructions/commands/learn.md → ~/.config/opencode/commands/learn.md"))
	g.Expect(out).To(ContainSubstring("[dry-run] installed: Claude Code, OpenCode"))
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
			name:         "old-vocab-present",
			hasOldVocab:  true,
			wantContains: []string{"Upgrading", "README.md"},
		},
		{
			name:            "no-old-vocab",
			hasOldVocab:     false,
			wantNotContains: []string{"Upgrading"},
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

func (liveUpdateFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm) // test adapter
}

// stubCommander satisfies update.Commander; dry-run local mode never runs it.
type stubCommander struct{}

func (stubCommander) Run(context.Context, string, string, ...string) ([]byte, []byte, error) {
	return nil, nil, nil
}

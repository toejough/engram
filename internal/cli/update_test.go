package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestOsCommander_ReportsFailure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cmd := cli.ExportNewOsCommander()

	_, _, err := cmd.Run(context.Background(), "", "false")
	g.Expect(err).To(HaveOccurred())
}

func TestOsCommander_RunsCommand(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cmd := cli.ExportNewOsCommander()

	stdout, _, err := cmd.Run(context.Background(), "", "echo", "hello world")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strings.TrimSpace(string(stdout))).To(Equal("hello world"))
}

func TestOsUpdateEnv_ReturnsValues(t *testing.T) {
	g := NewWithT(t)

	env := cli.ExportNewOsUpdateEnv()

	home, homeErr := env.UserHomeDir()
	g.Expect(homeErr).NotTo(HaveOccurred())
	g.Expect(home).NotTo(BeEmpty())

	cwd, cwdErr := env.Getwd()
	g.Expect(cwdErr).NotTo(HaveOccurred())
	g.Expect(cwd).NotTo(BeEmpty())

	t.Setenv("ENGRAM_UPDATE_TEST", "1")
	g.Expect(env.Getenv("ENGRAM_UPDATE_TEST")).To(Equal("1"))
}

func TestOsUpdateFS_MkdirAllOnFileErrors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	osFS := cli.ExportNewOsUpdateFS()

	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "blocking")

	writeErr := os.WriteFile(filePath, []byte("x"), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	// MkdirAll fails because filePath exists and is not a directory.
	err := osFS.MkdirAll(filepath.Join(filePath, "sub"), 0o755)
	g.Expect(err).To(HaveOccurred())
}

// osUpdateFS round-trip tests: exercise the production adapters against
// a tmp dir so coverage credits them.

func TestOsUpdateFS_ReadDirEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	tmp := t.TempDir()
	osFS := cli.ExportNewOsUpdateFS()

	entries, err := osFS.ReadDir(tmp)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}

func TestOsUpdateFS_ReadDirMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	osFS := cli.ExportNewOsUpdateFS()

	_, err := osFS.ReadDir(filepath.Join(t.TempDir(), "nope"))
	g.Expect(err).To(HaveOccurred())
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}

func TestOsUpdateFS_ReadFileMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	osFS := cli.ExportNewOsUpdateFS()

	_, err := osFS.ReadFile(filepath.Join(t.TempDir(), "nope"))
	g.Expect(err).To(HaveOccurred())
}

func TestOsUpdateFS_RemoveAllClearsDirAndIsIdempotent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	osFS := cli.ExportNewOsUpdateFS()
	tmp := t.TempDir()
	target := filepath.Join(tmp, "skill")

	mkErr := osFS.MkdirAll(filepath.Join(target, "nested"), 0o755)
	g.Expect(mkErr).NotTo(HaveOccurred())

	writeErr := osFS.WriteFile(filepath.Join(target, "x.md"), []byte("x"), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	removeErr := osFS.RemoveAll(target)
	g.Expect(removeErr).NotTo(HaveOccurred())

	_, statErr := osFS.Stat(target)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())

	// Idempotent: removing a non-existent path returns nil.
	g.Expect(osFS.RemoveAll(target)).NotTo(HaveOccurred())
}

func TestOsUpdateFS_RemoveAllErrorsOnReadOnlyParent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	osFS := cli.ExportNewOsUpdateFS()
	tmp := t.TempDir()
	parent := filepath.Join(tmp, "ro")
	child := filepath.Join(parent, "leaf")

	g.Expect(os.MkdirAll(child, 0o755)).NotTo(HaveOccurred())
	g.Expect(os.WriteFile(filepath.Join(child, "x"), []byte("x"), 0o644)).NotTo(HaveOccurred())

	// Read-only parent prevents removing the child entry.
	g.Expect(os.Chmod(parent, 0o500)).NotTo(HaveOccurred())
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

	err := osFS.RemoveAll(child)
	g.Expect(err).To(MatchError(ContainSubstring("remove")))
}

func TestOsUpdateFS_StatMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	osFS := cli.ExportNewOsUpdateFS()

	_, err := osFS.Stat(filepath.Join(t.TempDir(), "nope"))
	g.Expect(err).To(HaveOccurred())
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}

func TestOsUpdateFS_WriteAndRead(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	osFS := cli.ExportNewOsUpdateFS()
	tmp := t.TempDir()
	target := filepath.Join(tmp, "sub", "file.txt")
	body := []byte("hello")

	mkErr := osFS.MkdirAll(filepath.Dir(target), 0o755)
	g.Expect(mkErr).NotTo(HaveOccurred())

	writeErr := osFS.WriteFile(target, body, 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	got, readErr := osFS.ReadFile(target)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(got).To(Equal(body))

	info, statErr := osFS.Stat(filepath.Dir(target))
	g.Expect(statErr).NotTo(HaveOccurred())

	if statErr != nil || info == nil {
		return
	}

	g.Expect(info.IsDir()).To(BeTrue())

	entries, listErr := osFS.ReadDir(filepath.Dir(target))
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil || entries == nil {
		return
	}

	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Name()).To(Equal("file.txt"))
	g.Expect(entries[0].IsDir()).To(BeFalse())
}

func TestOsUpdateFS_WriteToBadPathErrors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	osFS := cli.ExportNewOsUpdateFS()

	// Writing to a directory path returns an error.
	err := osFS.WriteFile(t.TempDir(), []byte("x"), 0o644)
	g.Expect(err).To(HaveOccurred())
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

	// We run a dry-run against the live filesystem. The current cwd is
	// inside the engram worktree, so source resolution will pick local
	// mode without invoking `go install` (DryRun=true).
	err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{DryRun: true}, stdout)
	// Result depends on the local environment: at least one of
	// ~/.claude or ~/.config/opencode must be present, else
	// ErrNoHarness surfaces. Accept either outcome but verify output
	// when successful.
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

	// Dry-run with --with-guidance; cwd inside engram repo → local mode.
	// We only verify the flag is ACCEPTED and maps to Options (no unknown-field
	// error). The hint OUTPUT is asserted separately in
	// TestWriteUpdateReport_GuidanceActivationHint.
	err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{DryRun: true, WithGuidance: true}, stdout)

	// With dry-run the guidance files are not written; accept either
	// success or ErrNoHarness (env-dependent) but NOT an unknown-field error.
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

func TestWriteUpdateReport_GuidanceActivationHint(t *testing.T) {
	t.Parallel()

	table := []struct {
		name             string
		guidanceFiles    []string
		guidanceImported bool
		withGuidance     bool
		wantActivation   bool
		wantPlainHint    bool
	}{
		{
			name:             "deployed-not-imported-shows-activation-hint",
			guidanceFiles:    []string{"recall.md"},
			guidanceImported: false,
			withGuidance:     true,
			wantActivation:   true,
			wantPlainHint:    false,
		},
		{
			name:             "deployed-and-imported-no-hints",
			guidanceFiles:    []string{"recall.md"},
			guidanceImported: true,
			withGuidance:     true,
			wantActivation:   false,
			wantPlainHint:    false,
		},
		{
			name:             "plain-update-not-imported-shows-plain-hint",
			guidanceFiles:    nil,
			guidanceImported: false,
			withGuidance:     false,
			wantActivation:   false,
			wantPlainHint:    true,
		},
		{
			name:             "plain-update-already-imported-no-hint",
			guidanceFiles:    nil,
			guidanceImported: true,
			withGuidance:     false,
			wantActivation:   false,
			wantPlainHint:    false,
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

			const activationHint = "@~/.claude/engram/recall.md"

			const plainHint = "engram ships recall-firing guidance"

			if tc.wantActivation {
				g.Expect(out).To(ContainSubstring(activationHint))
			} else {
				g.Expect(out).NotTo(ContainSubstring(activationHint))
			}

			if tc.wantPlainHint {
				g.Expect(out).To(ContainSubstring(plainHint))
			} else {
				g.Expect(out).NotTo(ContainSubstring(plainHint))
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
	g.Expect(out).To(ContainSubstring("skills/learn/ → ~/.claude/skills/learn/  (3 files)"))
	g.Expect(out).To(ContainSubstring("skills/recall/ → ~/.claude/skills/recall/  (1 file)"))
	g.Expect(out).To(ContainSubstring("commands/learn.md → ~/.config/opencode/commands/learn.md"))
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

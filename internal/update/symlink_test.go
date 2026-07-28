package update_test

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strconv"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/update"
)

// TestApplyOps_ManifestMode_CommandCopyErrorIsHarnessError covers
// applyCmdOps's error branch, only reachable via the manifest-mode
// fallthrough now that symlink-mode harnesses materialize commands through
// applyCmdLinks instead.
func TestApplyOps_ManifestMode_CommandCopyErrorIsHarnessError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	// Source command file deliberately missing — applyCmdOne's ReadFile fails.

	spec := update.HarnessSpec{
		Name:              update.HarnessClaude,
		ProbeRel:          ".claude",
		CommandsTargetRel: filepath.Join(".claude", "commands"),
		EngramRootRel:     filepath.Join(".claude", "engram"),
		DeployMode:        update.DeployModeManifest,
	}

	cmdOps := []update.CopyOp{
		{
			Harness:     update.HarnessClaude,
			Src:         "/repo/agent-instructions/commands/recall.md",
			Dst:         home + "/.claude/commands/recall.md",
			CommandFile: "recall.md",
		},
	}

	updater := &update.Updater{FS: base, Cmd: &fakeCmd{}, Env: &fakeEnv{home: home, cwd: "/repo"}, Spawn: noopSpawner{}}

	reports := update.ExportApplyOps(updater, []update.HarnessSpec{spec}, home, nil, cmdOps, nil, false, false)
	g.Expect(reports).To(HaveLen(1))
	g.Expect(reports[0].Err).To(HaveOccurred())
	g.Expect(reports[0].CommandFiles).To(BeEmpty())
}

// TestApplyOps_ManifestMode_DryRun_NoWrites covers applyOne's dryRun branch,
// only reachable via the manifest-mode fallthrough now that symlink mode
// gates its own writes through materializeSymlink's separate dryRun checks.
func TestApplyOps_ManifestMode_DryRun_NoWrites(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")

	spec := update.HarnessSpec{
		Name:            update.HarnessClaude,
		ProbeRel:        ".claude",
		SkillsTargetRel: filepath.Join(".claude", "skills"),
		EngramRootRel:   filepath.Join(".claude", "engram"),
		DeployMode:      update.DeployModeManifest,
	}

	skillOps := []update.CopyOp{
		{
			Harness:  update.HarnessClaude,
			Src:      "/repo/agent-instructions/skills/learn/SKILL.md",
			Dst:      home + "/.claude/skills/learn/SKILL.md",
			SkillDir: "learn",
		},
	}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	reports := update.ExportApplyOps(updater, []update.HarnessSpec{spec}, home, skillOps, nil, nil, false, true)
	g.Expect(reports).To(HaveLen(1))
	g.Expect(reports[0].Err).NotTo(HaveOccurred())
	g.Expect(fileSystem.written).To(BeEmpty())
}

// --- 1.4 / D7: manifest-mode fallthrough equals today's copy behavior -------

func TestApplyOps_ManifestMode_FallsThroughToCopyBehavior(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")

	spec := update.HarnessSpec{
		Name:              update.HarnessClaude,
		ProbeRel:          ".claude",
		SkillsTargetRel:   filepath.Join(".claude", "skills"),
		CommandsTargetRel: filepath.Join(".claude", "commands"),
		GuidanceTargetRel: filepath.Join(".claude", "engram"),
		EngramRootRel:     filepath.Join(".claude", "engram"),
		DeployMode:        update.DeployModeManifest,
	}

	fileSystem.files["/repo/agent-instructions/skills/learn/tests/baseline.md"] = []byte("baseline")

	skillOps := []update.CopyOp{
		{
			Harness:  update.HarnessClaude,
			Src:      "/repo/agent-instructions/skills/learn/SKILL.md",
			Dst:      home + "/.claude/skills/learn/SKILL.md",
			SkillDir: "learn",
		},
		// A second file in the SAME skill dir exercises clearSkillDirOnce's
		// "already cleared" skip branch.
		{
			Harness:  update.HarnessClaude,
			Src:      "/repo/agent-instructions/skills/learn/tests/baseline.md",
			Dst:      home + "/.claude/skills/learn/tests/baseline.md",
			SkillDir: "learn",
		},
		// An op for a DIFFERENT harness exercises applySkillOps'
		// copyOp.Harness != name skip branch.
		{
			Harness:  update.HarnessOpencode,
			Src:      "/repo/agent-instructions/skills/learn/SKILL.md",
			Dst:      "/home/joe/.config/opencode/skills/learn/SKILL.md",
			SkillDir: "learn",
		},
	}
	cmdOps := []update.CopyOp{
		{
			Harness:     update.HarnessClaude,
			Src:         "/repo/agent-instructions/commands/recall.md",
			Dst:         home + "/.claude/commands/recall.md",
			CommandFile: "recall.md",
		},
	}
	guidanceOps := []update.CopyOp{
		{
			Harness:      update.HarnessClaude,
			Src:          "/repo/agent-instructions/guidance/recall.md",
			Dst:          home + "/.claude/engram/recall.md",
			GuidanceFile: "recall.md",
		},
	}
	fileSystem.files["/repo/agent-instructions/commands/recall.md"] = []byte("recall cmd")
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("recall guidance")

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	reports := update.ExportApplyOps(
		updater, []update.HarnessSpec{spec}, home, skillOps, cmdOps, guidanceOps, true, false)
	g.Expect(reports).To(HaveLen(1))
	g.Expect(reports[0].Err).NotTo(HaveOccurred())

	// Manifest-mode fallthrough = today's COPY behavior: real files land
	// directly at surface paths, never symlinks.
	g.Expect(fileSystem.written[home+"/.claude/skills/learn/SKILL.md"]).To(Equal([]byte("learn skill")))
	g.Expect(fileSystem.written[home+"/.claude/skills/learn/tests/baseline.md"]).To(Equal([]byte("baseline")))
	g.Expect(fileSystem.written[home+"/.claude/commands/recall.md"]).To(Equal([]byte("recall cmd")))
	g.Expect(fileSystem.written[home+"/.claude/engram/recall.md"]).To(Equal([]byte("recall guidance")))
	g.Expect(reports[0].CommandFiles).To(ConsistOf("recall.md"))
	g.Expect(reports[0].GuidanceFiles).To(ConsistOf("recall.md"))
	g.Expect(skillFileCount(reports[0])).To(Equal(2))

	// The OpenCode-targeted skill op was skipped for this (Claude-only) run.
	_, opencodeWritten := fileSystem.written["/home/joe/.config/opencode/skills/learn/SKILL.md"]
	g.Expect(opencodeWritten).To(BeFalse())

	g.Expect(fileSystem.symlinks).To(BeEmpty())
}

// TestApplyOps_ManifestMode_GuidanceCopyErrorIsHarnessError covers
// applyGuidanceOps's error branch, only reachable via the manifest-mode
// fallthrough now that Claude (GuidanceTargetRel == EngramRootRel) uses
// applyGuidanceCompatLinks and Pi (a separate guidance surface) uses
// applyGuidanceLinks instead (task 5.3 retired the flat-copy path for
// symlink-mode harnesses).
func TestApplyOps_ManifestMode_GuidanceCopyErrorIsHarnessError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	// Source guidance file deliberately missing — applyCmdOne's ReadFile fails.

	spec := update.HarnessSpec{
		Name:              update.HarnessClaude,
		ProbeRel:          ".claude",
		GuidanceTargetRel: filepath.Join(".claude", "guidance"),
		ImportsFileRel:    filepath.Join(".claude", "CLAUDE.md"),
		EngramRootRel:     filepath.Join(".claude", "engram"),
		DeployMode:        update.DeployModeManifest,
	}

	guidanceOps := []update.CopyOp{
		{
			Harness:      update.HarnessClaude,
			Src:          "/repo/agent-instructions/guidance/recall.md",
			Dst:          home + "/.claude/guidance/recall.md",
			GuidanceFile: "recall.md",
		},
	}

	updater := &update.Updater{FS: base, Cmd: &fakeCmd{}, Env: &fakeEnv{home: home, cwd: "/repo"}, Spawn: noopSpawner{}}

	reports := update.ExportApplyOps(updater, []update.HarnessSpec{spec}, home, nil, nil, guidanceOps, true, false)
	g.Expect(reports).To(HaveLen(1))
	g.Expect(reports[0].Err).To(HaveOccurred())
	g.Expect(reports[0].GuidanceFiles).To(BeEmpty())
}

// TestApplyOps_ManifestMode_SkillClearFailureIsHarnessError covers
// clearSkillDirOnce's RemoveAll-error branch, only reachable via the
// manifest-mode fallthrough now that symlink-mode harnesses never RemoveAll
// a fresh skill surface.
func TestApplyOps_ManifestMode_SkillClearFailureIsHarnessError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	base.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")

	spec := update.HarnessSpec{
		Name:            update.HarnessClaude,
		ProbeRel:        ".claude",
		SkillsTargetRel: filepath.Join(".claude", "skills"),
		EngramRootRel:   filepath.Join(".claude", "engram"),
		DeployMode:      update.DeployModeManifest,
	}

	skillOps := []update.CopyOp{
		{
			Harness:  update.HarnessClaude,
			Src:      "/repo/agent-instructions/skills/learn/SKILL.md",
			Dst:      home + "/.claude/skills/learn/SKILL.md",
			SkillDir: "learn",
		},
	}

	fileSystem := &failRemoveAllFS{memFS: base, failOn: ".claude/skills/learn"}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	reports := update.ExportApplyOps(updater, []update.HarnessSpec{spec}, home, skillOps, nil, nil, false, false)
	g.Expect(reports).To(HaveLen(1))
	g.Expect(reports[0].Err).To(HaveOccurred())
	g.Expect(reports[0].Err.Error()).To(ContainSubstring("clear"))
}

// TestApplyOps_ManifestMode_SkillCopyErrorIsHarnessError covers
// applySkillOps' applyOne-error branch (distinct from the clear-error
// branch above): clearSkillDirOnce succeeds but the subsequent read fails.
func TestApplyOps_ManifestMode_SkillCopyErrorIsHarnessError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	// Source file deliberately missing — applyOne's ReadFile fails.

	spec := update.HarnessSpec{
		Name:            update.HarnessClaude,
		ProbeRel:        ".claude",
		SkillsTargetRel: filepath.Join(".claude", "skills"),
		EngramRootRel:   filepath.Join(".claude", "engram"),
		DeployMode:      update.DeployModeManifest,
	}

	skillOps := []update.CopyOp{
		{
			Harness:  update.HarnessClaude,
			Src:      "/repo/agent-instructions/skills/learn/SKILL.md",
			Dst:      home + "/.claude/skills/learn/SKILL.md",
			SkillDir: "learn",
		},
	}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	reports := update.ExportApplyOps(updater, []update.HarnessSpec{spec}, home, skillOps, nil, nil, false, false)
	g.Expect(reports).To(HaveLen(1))
	g.Expect(reports[0].Err).To(HaveOccurred())
}

// TestApplyOps_ManifestMode_SkillMkdirErrorIsHarnessError covers applyOne's
// MkdirAll-error branch, only reachable via the manifest-mode fallthrough.
func TestApplyOps_ManifestMode_SkillMkdirErrorIsHarnessError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	base.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")

	spec := update.HarnessSpec{
		Name:            update.HarnessClaude,
		ProbeRel:        ".claude",
		SkillsTargetRel: filepath.Join(".claude", "skills"),
		EngramRootRel:   filepath.Join(".claude", "engram"),
		DeployMode:      update.DeployModeManifest,
	}

	skillOps := []update.CopyOp{
		{
			Harness:  update.HarnessClaude,
			Src:      "/repo/agent-instructions/skills/learn/SKILL.md",
			Dst:      home + "/.claude/skills/learn/SKILL.md",
			SkillDir: "learn",
		},
	}

	mkdirErr := errors.New("mkdir boom")
	fileSystem := &symlinkFaultFS{memFS: base, mkdirErr: map[string]error{home + "/.claude/skills/learn": mkdirErr}}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	reports := update.ExportApplyOps(updater, []update.HarnessSpec{spec}, home, skillOps, nil, nil, false, false)
	g.Expect(reports).To(HaveLen(1))
	g.Expect(reports[0].Err).To(MatchError(ContainSubstring("mkdir boom")))
}

// TestApplyOps_ManifestMode_SkillWriteErrorIsHarnessError covers applyOne's
// WriteFile-error branch, only reachable via the manifest-mode fallthrough.
func TestApplyOps_ManifestMode_SkillWriteErrorIsHarnessError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	base.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")

	spec := update.HarnessSpec{
		Name:            update.HarnessClaude,
		ProbeRel:        ".claude",
		SkillsTargetRel: filepath.Join(".claude", "skills"),
		EngramRootRel:   filepath.Join(".claude", "engram"),
		DeployMode:      update.DeployModeManifest,
	}

	skillOps := []update.CopyOp{
		{
			Harness:  update.HarnessClaude,
			Src:      "/repo/agent-instructions/skills/learn/SKILL.md",
			Dst:      home + "/.claude/skills/learn/SKILL.md",
			SkillDir: "learn",
		},
	}

	fileSystem := &failWriteFS{memFS: base, failOn: ".claude/skills/learn/SKILL.md"}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	reports := update.ExportApplyOps(updater, []update.HarnessSpec{spec}, home, skillOps, nil, nil, false, false)
	g.Expect(reports).To(HaveLen(1))
	g.Expect(reports[0].Err).To(MatchError(ContainSubstring("write boom")))
}

func TestCleanupDanglingLinksInDir_DryRun_RecordsWithoutRemoving(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	symlinkErr := fileSystem.Symlink("/home/engram/skills/gone", "/home/skills/gone")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	removed, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", true)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(removed).To(ConsistOf("/home/skills/gone"))
	g.Expect(fileSystem.symlinks["/home/skills/gone"]).To(Equal("/home/engram/skills/gone"))
}

func TestCleanupDanglingLinksInDir_EmptyDirParam_NoOp(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	removed, err := update.ExportCleanupDanglingLinksInDir(newMemFS(), "", "/home/engram", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(removed).To(BeEmpty())
}

// --- 4.2: dangling-link cleanup (cleanupDanglingLinksInDir) ------------------

func TestCleanupDanglingLinksInDir_EngramLinkTargetMissing_Removed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	symlinkErr := fileSystem.Symlink("/home/engram/skills/gone", "/home/skills/gone")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	removed, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(removed).To(ConsistOf("/home/skills/gone"))

	_, stillSymlink := fileSystem.symlinks["/home/skills/gone"]
	g.Expect(stillSymlink).To(BeFalse())
}

func TestCleanupDanglingLinksInDir_EngramLinkTargetPresent_Kept(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	fileSystem.dirs["/home/engram/skills/recall"] = true
	symlinkErr := fileSystem.Symlink("/home/engram/skills/recall", "/home/skills/recall")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	removed, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(removed).To(BeEmpty())
	g.Expect(fileSystem.symlinks["/home/skills/recall"]).To(Equal("/home/engram/skills/recall"))
}

func TestCleanupDanglingLinksInDir_ForeignSymlinkTargetMissing_NeverTouched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	// Points OUTSIDE root and its own target is gone too — still foreign:
	// root-membership is decided before target existence is even checked.
	symlinkErr := fileSystem.Symlink("/elsewhere/gone", "/home/skills/user-skill")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	removed, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(removed).To(BeEmpty())
	g.Expect(fileSystem.symlinks["/home/skills/user-skill"]).To(Equal("/elsewhere/gone"))
}

func TestCleanupDanglingLinksInDir_ForeignSymlinkTargetPresent_NeverTouched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	fileSystem.dirs["/elsewhere/user-skill"] = true
	symlinkErr := fileSystem.Symlink("/elsewhere/user-skill", "/home/skills/user-skill")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	removed, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(removed).To(BeEmpty())
	g.Expect(fileSystem.symlinks["/home/skills/user-skill"]).To(Equal("/elsewhere/user-skill"))
}

func TestCleanupDanglingLinksInDir_LstatError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/skills"] = true
	symlinkErr := base.Symlink("/home/engram/skills/gone", "/home/skills/gone")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	lstatErr := errors.New("lstat boom")
	fileSystem := &symlinkFaultFS{memFS: base, lstatErr: map[string]error{"/home/skills/gone": lstatErr}}

	_, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", false)
	g.Expect(err).To(MatchError(ContainSubstring("lstat boom")))
}

func TestCleanupDanglingLinksInDir_MissingDir_NoOp(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	removed, err := update.ExportCleanupDanglingLinksInDir(newMemFS(), "/home/skills", "/home/engram", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(removed).To(BeEmpty())
}

// TestCleanupDanglingLinksInDir_Property_OnlyRootMatchedDanglingLinksRemoved
// draws an arbitrary surface dir mixing real files, real dirs, healthy
// engram links, dangling engram links, and foreign links (some of which are
// also individually dangling), then asserts cleanup removes EXACTLY the
// root-matched dangling-link set and leaves every other entry untouched.
func TestCleanupDanglingLinksInDir_Property_OnlyRootMatchedDanglingLinksRemoved(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		const (
			dir  = "/home/skills"
			root = "/home/engram"
		)

		fileSystem := newMemFS()
		fileSystem.dirs[dir] = true

		entryCount := rapid.IntRange(0, 6).Draw(rt, "entryCount")

		wantRemoved := map[string]bool{}
		wantKept := map[string]bool{}

		for i := range entryCount {
			seedCleanupPropertyEntry(rt, fileSystem, dir, root, i, wantRemoved, wantKept)
		}

		removed, err := update.ExportCleanupDanglingLinksInDir(fileSystem, dir, root, false)
		if err != nil {
			rt.Fatalf("cleanup: %v", err)
		}

		assertCleanupPropertyOutcome(rt, fileSystem, removed, wantRemoved, wantKept)
	})
}

func TestCleanupDanglingLinksInDir_ReadDirError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/skills"] = true

	readDirErr := errors.New("readdir boom")
	fileSystem := &symlinkFaultFS{memFS: base, readDirErr: map[string]error{"/home/skills": readDirErr}}

	_, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", false)
	g.Expect(err).To(MatchError(ContainSubstring("readdir boom")))
}

func TestCleanupDanglingLinksInDir_ReadLinkError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/skills"] = true
	symlinkErr := base.Symlink("/home/engram/skills/gone", "/home/skills/gone")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	readLinkErr := errors.New("readlink boom")
	fileSystem := &symlinkFaultFS{memFS: base, readLinkErr: map[string]error{"/home/skills/gone": readLinkErr}}

	_, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", false)
	g.Expect(err).To(MatchError(ContainSubstring("readlink boom")))
}

func TestCleanupDanglingLinksInDir_RealDir_NeverTouched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	fileSystem.dirs["/home/skills/user-installed"] = true
	fileSystem.files["/home/skills/user-installed/SKILL.md"] = []byte("real dir")

	removed, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(removed).To(BeEmpty())
	g.Expect(fileSystem.dirs["/home/skills/user-installed"]).To(BeTrue())
}

func TestCleanupDanglingLinksInDir_RealFile_NeverTouched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/commands"] = true
	fileSystem.files["/home/commands/recall.md"] = []byte("real file")

	removed, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/commands", "/home/engram", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(removed).To(BeEmpty())
	g.Expect(fileSystem.files["/home/commands/recall.md"]).To(Equal([]byte("real file")))
}

func TestCleanupDanglingLinksInDir_RelativeTargetResolvesLexically_Dangling(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	// Relative target "../engram/skills/gone", resolved lexically against
	// parent dir "/home/skills", is "/home/engram/skills/gone" — inside
	// root, nothing seeded there.
	symlinkErr := fileSystem.Symlink("../engram/skills/gone", "/home/skills/gone")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	removed, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(removed).To(ConsistOf("/home/skills/gone"))
}

func TestCleanupDanglingLinksInDir_RelativeTargetResolvesLexically_Healthy(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	fileSystem.dirs["/home/engram/skills/recall"] = true
	symlinkErr := fileSystem.Symlink("../engram/skills/recall", "/home/skills/recall")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	removed, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(removed).To(BeEmpty())
}

func TestCleanupDanglingLinksInDir_RemoveAllError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/skills"] = true
	symlinkErr := base.Symlink("/home/engram/skills/gone", "/home/skills/gone")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	removeErr := errors.New("remove boom")
	fileSystem := &symlinkFaultFS{memFS: base, removeAllErr: map[string]error{"/home/skills/gone": removeErr}}

	_, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", false)
	g.Expect(err).To(MatchError(ContainSubstring("remove boom")))
}

func TestCleanupDanglingLinksInDir_StatError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/skills"] = true
	symlinkErr := base.Symlink("/home/engram/skills/gone", "/home/skills/gone")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	statErr := errors.New("stat boom")
	fileSystem := &symlinkFaultFS{memFS: base, statErr: map[string]error{"/home/engram/skills/gone": statErr}}

	_, err := update.ExportCleanupDanglingLinksInDir(fileSystem, "/home/skills", "/home/engram", false)
	g.Expect(err).To(MatchError(ContainSubstring("stat boom")))
}

// --- pure lexical-resolution helpers -----------------------------------------

func TestLexicallyResolveSymlinkTarget_Absolute(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := update.ExportLexicallyResolveSymlinkTarget("/home/engram/skills/recall", "/home/skills")
	g.Expect(got).To(Equal("/home/engram/skills/recall"))
}

func TestLexicallyResolveSymlinkTarget_RelativeJoinsParent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := update.ExportLexicallyResolveSymlinkTarget("../engram/skills/recall", "/home/skills")
	g.Expect(got).To(Equal("/home/engram/skills/recall"))
}

func TestLexicallyResolveSymlinkTarget_RelativeSameDir(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := update.ExportLexicallyResolveSymlinkTarget("recall-real", "/home/engram/skills")
	g.Expect(got).To(Equal("/home/engram/skills/recall-real"))
}

// --- 4.1: symlink materialization (materializeSymlink) ----------------------

func TestMaterializeSymlink_AbsentCreatesLink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()

	blocked, err := update.ExportMaterializeSymlink(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(blocked).To(BeFalse())

	target, ok := fileSystem.symlinks["/home/skills/recall"]
	g.Expect(ok).To(BeTrue())
	g.Expect(target).To(Equal("/home/engram/skills/recall"))
	g.Expect(fileSystem.dirs["/home/skills"]).To(BeTrue())
}

func TestMaterializeSymlink_AbsentDryRun_NoWrite(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()

	blocked, err := update.ExportMaterializeSymlink(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", true)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(blocked).To(BeFalse())

	_, ok := fileSystem.symlinks["/home/skills/recall"]
	g.Expect(ok).To(BeFalse())
	g.Expect(fileSystem.dirs["/home/skills"]).To(BeFalse())
}

func TestMaterializeSymlink_ExistingCorrectSymlink_NoOp(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	symlinkErr := fileSystem.Symlink("/home/engram/skills/recall", "/home/skills/recall")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	blocked, err := update.ExportMaterializeSymlink(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(blocked).To(BeFalse())
	g.Expect(fileSystem.removed).To(BeEmpty()) // healthy — never touched
	g.Expect(fileSystem.symlinks["/home/skills/recall"]).To(Equal("/home/engram/skills/recall"))
}

func TestMaterializeSymlink_ExistingWrongSymlink_DryRun_NoWrite(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	symlinkErr := fileSystem.Symlink("/home/engram/skills/old-name", "/home/skills/recall")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	blocked, err := update.ExportMaterializeSymlink(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", true)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(blocked).To(BeFalse())
	g.Expect(fileSystem.removed).To(BeEmpty())
	g.Expect(fileSystem.symlinks["/home/skills/recall"]).To(Equal("/home/engram/skills/old-name"))
}

func TestMaterializeSymlink_ExistingWrongSymlink_Repoints(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	symlinkErr := fileSystem.Symlink("/home/engram/skills/old-name", "/home/skills/recall")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	blocked, err := update.ExportMaterializeSymlink(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(blocked).To(BeFalse())
	g.Expect(fileSystem.removed).To(ContainElement("/home/skills/recall"))
	g.Expect(fileSystem.symlinks["/home/skills/recall"]).To(Equal("/home/engram/skills/recall"))
}

func TestMaterializeSymlink_LstatError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lstatErr := errors.New("lstat boom")
	fileSystem := &symlinkFaultFS{memFS: newMemFS(), lstatErr: map[string]error{"/home/skills/recall": lstatErr}}

	_, err := update.ExportMaterializeSymlink(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).To(MatchError(ContainSubstring("lstat boom")))
}

func TestMaterializeSymlink_MkdirError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mkdirErr := errors.New("mkdir boom")
	fileSystem := &symlinkFaultFS{memFS: newMemFS(), mkdirErr: map[string]error{"/home/skills": mkdirErr}}

	_, err := update.ExportMaterializeSymlink(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).To(MatchError(ContainSubstring("mkdir boom")))
}

func TestMaterializeSymlink_ReadLinkError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/skills"] = true
	symlinkErr := base.Symlink("/home/engram/skills/old-name", "/home/skills/recall")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	readLinkErr := errors.New("readlink boom")
	fileSystem := &symlinkFaultFS{memFS: base, readLinkErr: map[string]error{"/home/skills/recall": readLinkErr}}

	_, err := update.ExportMaterializeSymlink(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).To(MatchError(ContainSubstring("readlink boom")))
}

func TestMaterializeSymlink_RealDir_LeftUntouchedAndBlocked(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills/recall"] = true
	fileSystem.files["/home/skills/recall/SKILL.md"] = []byte("real pre-migration copy")

	blocked, err := update.ExportMaterializeSymlink(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(blocked).To(BeTrue())
	g.Expect(fileSystem.dirs["/home/skills/recall"]).To(BeTrue())
	g.Expect(fileSystem.files["/home/skills/recall/SKILL.md"]).To(Equal([]byte("real pre-migration copy")))
	g.Expect(fileSystem.removed).To(BeEmpty())
}

func TestMaterializeSymlink_RealFile_LeftUntouchedAndBlocked(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/home/commands/recall.md"] = []byte("real pre-migration copy")

	blocked, err := update.ExportMaterializeSymlink(
		fileSystem, "/home/commands/recall.md", "/home/engram/commands/recall.md", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(blocked).To(BeTrue())
	g.Expect(fileSystem.files["/home/commands/recall.md"]).To(Equal([]byte("real pre-migration copy")))
	g.Expect(fileSystem.removed).To(BeEmpty())

	_, isSymlink := fileSystem.symlinks["/home/commands/recall.md"]
	g.Expect(isSymlink).To(BeFalse())
}

func TestMaterializeSymlink_RepointRemoveError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/skills"] = true
	symlinkErr := base.Symlink("/home/engram/skills/old-name", "/home/skills/recall")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	removeErr := errors.New("remove boom")
	fileSystem := &symlinkFaultFS{memFS: base, removeAllErr: map[string]error{"/home/skills/recall": removeErr}}

	_, err := update.ExportMaterializeSymlink(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).To(MatchError(ContainSubstring("remove boom")))
}

func TestMaterializeSymlink_RepointSymlinkCreateError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/skills"] = true
	symlinkErr := base.Symlink("/home/engram/skills/old-name", "/home/skills/recall")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	createErr := errors.New("symlink recreate boom")
	fileSystem := &symlinkFaultFS{memFS: base, symlinkErr: map[string]error{"/home/skills/recall": createErr}}

	_, err := update.ExportMaterializeSymlink(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).To(MatchError(ContainSubstring("symlink recreate boom")))
}

func TestMaterializeSymlink_SymlinkCreateError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	symlinkErr := errors.New("symlink boom")
	fileSystem := &symlinkFaultFS{memFS: newMemFS(), symlinkErr: map[string]error{"/home/skills/recall": symlinkErr}}

	_, err := update.ExportMaterializeSymlink(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).To(MatchError(ContainSubstring("symlink boom")))
}

func TestPathWithinRoot_ExactMatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(update.ExportPathWithinRoot("/home/engram", "/home/engram")).To(BeTrue())
}

func TestPathWithinRoot_NestedMatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(update.ExportPathWithinRoot("/home/engram/skills/recall", "/home/engram")).To(BeTrue())
}

func TestPathWithinRoot_OutsideEntirely(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(update.ExportPathWithinRoot("/elsewhere/x", "/home/engram")).To(BeFalse())
}

func TestPathWithinRoot_SiblingPrefixNotMatched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// "/home/engram2/x" textually starts with "/home/engram" but is NOT
	// inside it — the separator-boundary check must reject this.
	g.Expect(update.ExportPathWithinRoot("/home/engram2/x", "/home/engram")).To(BeFalse())
}

// TestRun_SymlinkFreeTree_CleanupAndMaterializationRecordNothingUnexpected
// is the note-475 regression check (design.md D5, vault note 475): a
// harness surface holding zero symlinks — only real, pre-migration content
// — must survive materialization+cleanup with every recorded path an exact
// match of its logical (Join-and-Clean) spelling. An EvalSymlinks-style
// resolution anywhere in this path would leak a macOS /var → /private/var
// rewrite into every recorded path even though nothing here is a symlink;
// this fails loudly if that ever creeps in.
func TestRun_SymlinkFreeTree_CleanupAndMaterializationRecordNothingUnexpected(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/learn"] = true
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")

	// The surface holds the SAME skill as a REAL directory (the
	// pre-migration shape task 5.1 adopts) — NO symlinks anywhere.
	fileSystem.dirs[home+"/.claude/skills"] = true
	fileSystem.dirs[home+"/.claude/skills/learn"] = true
	fileSystem.files[home+"/.claude/skills/learn/SKILL.md"] = []byte("real copy, pre-migration")

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())

	// Nothing dangling — there were no symlinks to begin with.
	g.Expect(harness.DanglingLinksRemoved).To(BeEmpty())

	// The real pre-migration copy is an intended-set member (skill "learn"
	// matches the source) — it is ADOPTED (task 5.1), not left in place:
	// removed and replaced by a symlink into the engram root, at its EXACT
	// logical (unresolved) path — never rewritten by any resolution step.
	g.Expect(fileSystem.removed).To(ContainElement(home + "/.claude/skills/learn"))

	target, isSymlink := fileSystem.symlinks[home+"/.claude/skills/learn"]
	g.Expect(isSymlink).To(BeTrue())
	g.Expect(target).To(Equal(home + "/.claude/engram/skills/learn"))
	g.Expect(harness.EngramAdopted).To(ConsistOf(home + "/.claude/skills/learn"))
	g.Expect(harness.SurfaceUnattributable).To(BeEmpty())

	// EngramRoot and SkillsRoot are the exact logical Join(home, *Rel)
	// strings, never a resolved/rewritten variant.
	g.Expect(harness.EngramRoot).To(Equal(home + "/.claude/engram"))
	g.Expect(harness.SkillsRoot).To(Equal(home + "/.claude/skills"))
}

// TestRun_SymlinkMode_ClaudeGuidanceCreatesCompatSymlink covers task 5.3:
// the old flat-copy path (applyGuidanceOps) for Claude's D1 guidance
// exclusion is RETIRED — the flat path is now a compat symlink into the
// canonical guidance/ subtree, not a real copy.
func TestRun_SymlinkMode_ClaudeGuidanceCreatesCompatSymlink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("claude recall guidance")
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{WithGuidance: true})
	g.Expect(err).NotTo(HaveOccurred())

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())
	g.Expect(harness.GuidanceFiles).To(ConsistOf("recall.md"))

	engramRoot := home + "/.claude/engram"

	// The canonical copy lives under guidance/, synced by the root sync engine.
	g.Expect(fileSystem.written[engramRoot+"/guidance/recall.md"]).To(Equal([]byte("claude recall guidance")))

	// The flat @import path is a COMPAT SYMLINK into it — no more real copy.
	target, isSymlink := fileSystem.symlinks[engramRoot+"/recall.md"]
	g.Expect(isSymlink).To(BeTrue())
	g.Expect(target).To(Equal(engramRoot + "/guidance/recall.md"))
	g.Expect(fileSystem.files[engramRoot+"/recall.md"]).To(BeNil())
}

func TestRun_SymlinkMode_DanglingLinkRemovedWhenSourceArtifactGone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	// The source skill "gone" no longer exists this run.

	root := home + "/.claude/engram"
	fileSystem.dirs[root] = true
	fileSystem.files[root+"/.engram-owned"] = []byte{}
	fileSystem.dirs[home+"/.claude/skills"] = true
	symlinkErr := fileSystem.Symlink(root+"/skills/gone", home+"/.claude/skills/gone")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())
	g.Expect(harness.DanglingLinksRemoved).To(ConsistOf(home + "/.claude/skills/gone"))

	_, stillSymlink := fileSystem.symlinks[home+"/.claude/skills/gone"]
	g.Expect(stillSymlink).To(BeFalse())
}

func TestRun_SymlinkMode_DryRun_NoLinksCreated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/recall"] = true
	fileSystem.files["/repo/agent-instructions/skills/recall/SKILL.md"] = []byte("recall skill")

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())

	g.Expect(fileSystem.symlinks).To(BeEmpty())
	g.Expect(fileSystem.written).To(BeEmpty())
	g.Expect(fileSystem.dirs[home+"/.claude/skills"]).To(BeFalse())
}

func TestRun_SymlinkMode_ForeignSymlinkNeverTouched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/recall"] = true
	fileSystem.files["/repo/agent-instructions/skills/recall/SKILL.md"] = []byte("recall skill")

	fileSystem.dirs[home+"/.claude/skills"] = true
	fileSystem.dirs["/opt/my-tools/some-skill"] = true
	symlinkErr := fileSystem.Symlink("/opt/my-tools/some-skill", home+"/.claude/skills/my-own-skill")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())

	g.Expect(fileSystem.symlinks[home+"/.claude/skills/my-own-skill"]).To(Equal("/opt/my-tools/some-skill"))
	g.Expect(harness.DanglingLinksRemoved).To(BeEmpty())
	g.Expect(fileSystem.removed).NotTo(ContainElement(home + "/.claude/skills/my-own-skill"))
}

// --- Run()-level integration: symlink materialization + cleanup wired in ----

func TestRun_SymlinkMode_FreshSkillAndCommandLinksCreated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.config/opencode"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/recall"] = true
	fileSystem.files["/repo/agent-instructions/skills/recall/SKILL.md"] = []byte("recall skill")
	fileSystem.dirs["/repo/agent-instructions/commands"] = true
	fileSystem.files["/repo/agent-instructions/commands/recall.md"] = []byte("recall cmd")

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())

	engramRoot := home + "/.config/opencode/engram"
	g.Expect(harness.EngramRoot).To(Equal(engramRoot))

	skillTarget, ok := fileSystem.symlinks[home+"/.config/opencode/skills/recall"]
	g.Expect(ok).To(BeTrue())
	g.Expect(skillTarget).To(Equal(engramRoot + "/skills/recall"))

	cmdTarget, ok := fileSystem.symlinks[home+"/.config/opencode/commands/recall.md"]
	g.Expect(ok).To(BeTrue())
	g.Expect(cmdTarget).To(Equal(engramRoot + "/commands/recall.md"))

	g.Expect(fileSystem.written[engramRoot+"/skills/recall/SKILL.md"]).To(Equal([]byte("recall skill")))
	g.Expect(fileSystem.written[engramRoot+"/commands/recall.md"]).To(Equal([]byte("recall cmd")))

	_, surfaceHasRealFile := fileSystem.files[home+"/.config/opencode/skills/recall/SKILL.md"]
	g.Expect(surfaceHasRealFile).To(BeFalse())

	g.Expect(skillFileCount(harness)).To(Equal(1))
	g.Expect(harness.CommandFiles).To(ConsistOf("recall.md"))
	g.Expect(harness.SurfaceUnattributable).To(BeEmpty())
}

// TestRun_SymlinkMode_PiGuidanceLinkError_IsHarnessError covers
// applyGuidanceLinks' materializeSymlink-error branch via a full Run(),
// injecting an Lstat failure at the Pi guidance surface path.
func TestRun_SymlinkMode_PiGuidanceLinkError_IsHarnessError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.pi"] = true
	base.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	base.dirs["/repo/agent-instructions/skills"] = true
	base.files["/repo/agent-instructions/guidance/recall.md"] = []byte("pi recall guidance")
	base.dirs["/repo/agent-instructions/guidance"] = true

	lstatErr := errors.New("lstat boom")
	fileSystem := &symlinkFaultFS{
		memFS:    base,
		lstatErr: map[string]error{home + "/.pi/agent/guidance/recall.md": lstatErr},
	}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{WithGuidance: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(MatchError(ContainSubstring("lstat boom")))
}

// TestRun_SymlinkMode_PiGuidanceLinksGuidanceFiles also seeds Claude Code
// alongside Pi (both opted into --with-guidance) so Pi's applyGuidanceLinks
// loop processes Claude's guidanceOps entries too, exercising the
// copyOp.Harness != spec.Name skip branch.
func TestRun_SymlinkMode_PiGuidanceLinksGuidanceFiles(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.pi"] = true
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("pi recall guidance")
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{WithGuidance: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(2))

	var piHarness update.HarnessReport

	for _, harness := range report.Harnesses {
		if harness.Name == update.HarnessPi {
			piHarness = harness
		}

		g.Expect(harness.Err).NotTo(HaveOccurred())
	}

	g.Expect(piHarness.GuidanceFiles).To(ConsistOf("recall.md"))

	engramRoot := home + "/.pi/agent/engram"

	target, ok := fileSystem.symlinks[home+"/.pi/agent/guidance/recall.md"]
	g.Expect(ok).To(BeTrue())
	g.Expect(target).To(Equal(engramRoot + "/guidance/recall.md"))

	g.Expect(fileSystem.written[engramRoot+"/guidance/recall.md"]).To(Equal([]byte("pi recall guidance")))

	// Claude's guidance op (processed and skipped by Pi's loop) never wrote
	// or linked anything into Pi's engram root.
	_, piGotClaudeCopy := fileSystem.written[engramRoot+"/recall.md"]
	g.Expect(piGotClaudeCopy).To(BeFalse())
}

// TestRun_SymlinkMode_PiRealGuidanceFile_AdoptsToSymlink covers task 5.1: a
// real pre-existing guidance file at Pi's SEPARATE guidance surface (an
// intended-set member) is adopted — replaced by a symlink into the
// engram root — never merely left in place and reported.
func TestRun_SymlinkMode_PiRealGuidanceFile_AdoptsToSymlink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.pi"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("fresh pi guidance")
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true

	// A pre-existing REAL guidance file already sits at the Pi surface path.
	fileSystem.dirs[home+"/.pi/agent/guidance"] = true
	fileSystem.files[home+"/.pi/agent/guidance/recall.md"] = []byte("stale real copy")

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{WithGuidance: true})
	g.Expect(err).NotTo(HaveOccurred())

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())

	surfacePath := home + "/.pi/agent/guidance/recall.md"
	engramPath := home + "/.pi/agent/engram/guidance/recall.md"

	g.Expect(fileSystem.files[surfacePath]).To(BeNil())
	g.Expect(fileSystem.removed).To(ContainElement(surfacePath))
	target, isSymlink := fileSystem.symlinks[surfacePath]
	g.Expect(isSymlink).To(BeTrue())
	g.Expect(target).To(Equal(engramPath))

	g.Expect(harness.EngramAdopted).To(ConsistOf(surfacePath))
	g.Expect(harness.SurfaceUnattributable).To(BeEmpty())
	g.Expect(harness.GuidanceFiles).To(ConsistOf("recall.md"))

	g.Expect(fileSystem.written[engramPath]).To(Equal([]byte("fresh pi guidance")))
}

// TestRun_SymlinkMode_RealSkillDirAdoptsToSymlink covers task 5.1: a real
// skill dir already occupying a symlink-mode harness's skills surface (an
// intended-set member) is adopted — replaced by a symlink into the engram
// root — never merely left in place and reported.
func TestRun_SymlinkMode_RealSkillDirAdoptsToSymlink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/recall"] = true
	fileSystem.files["/repo/agent-instructions/skills/recall/SKILL.md"] = []byte("fresh")

	fileSystem.dirs[home+"/.claude/skills"] = true
	fileSystem.dirs[home+"/.claude/skills/recall"] = true
	fileSystem.files[home+"/.claude/skills/recall/SKILL.md"] = []byte("stale real copy")

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())

	g.Expect(fileSystem.files[home+"/.claude/skills/recall/SKILL.md"]).To(BeNil())

	target, isSymlink := fileSystem.symlinks[home+"/.claude/skills/recall"]
	g.Expect(isSymlink).To(BeTrue())
	g.Expect(target).To(Equal(home + "/.claude/engram/skills/recall"))
	g.Expect(fileSystem.removed).To(ContainElement(home + "/.claude/skills/recall"))

	g.Expect(harness.EngramAdopted).To(ConsistOf(home + "/.claude/skills/recall"))
	g.Expect(harness.SurfaceUnattributable).To(BeEmpty())
	g.Expect(fileSystem.written[home+"/.claude/engram/skills/recall/SKILL.md"]).To(Equal([]byte("fresh")))
}

func TestRun_SymlinkMode_WrongTargetLinkRepointed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/recall"] = true
	fileSystem.files["/repo/agent-instructions/skills/recall/SKILL.md"] = []byte("recall skill")

	fileSystem.dirs[home+"/.claude/skills"] = true
	symlinkErr := fileSystem.Symlink(home+"/.claude/engram/skills/old-recall", home+"/.claude/skills/recall")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())

	g.Expect(fileSystem.removed).To(ContainElement(home + "/.claude/skills/recall"))
	g.Expect(fileSystem.symlinks[home+"/.claude/skills/recall"]).To(Equal(home + "/.claude/engram/skills/recall"))
}

// --- error-path coverage: materialization/cleanup failures memFS alone -----
// --- cannot reach -----------------------------------------------------------

// symlinkFaultFS wraps memFS and injects a configurable real I/O error for
// specific operations at specific paths — exercises materializeSymlink's
// and cleanupDanglingLinksInDir's error branches. Distinct from
// engramFaultFS (sync_test.go), which covers the engram-root sync engine's
// own I/O surface.
type symlinkFaultFS struct {
	*memFS

	lstatErr     map[string]error
	readDirErr   map[string]error
	readLinkErr  map[string]error
	mkdirErr     map[string]error
	symlinkErr   map[string]error
	removeAllErr map[string]error
	statErr      map[string]error
}

func (f *symlinkFaultFS) Lstat(path string) (update.FileInfo, error) {
	if err, ok := f.lstatErr[path]; ok {
		return nil, err
	}

	return f.memFS.Lstat(path)
}

func (f *symlinkFaultFS) MkdirAll(path string, perm fs.FileMode) error {
	if err, ok := f.mkdirErr[path]; ok {
		return err
	}

	return f.memFS.MkdirAll(path, perm)
}

func (f *symlinkFaultFS) ReadDir(path string) ([]update.DirEntry, error) {
	if err, ok := f.readDirErr[path]; ok {
		return nil, err
	}

	return f.memFS.ReadDir(path)
}

func (f *symlinkFaultFS) ReadLink(path string) (string, error) {
	if err, ok := f.readLinkErr[path]; ok {
		return "", err
	}

	return f.memFS.ReadLink(path)
}

func (f *symlinkFaultFS) RemoveAll(path string) error {
	if err, ok := f.removeAllErr[path]; ok {
		return err
	}

	return f.memFS.RemoveAll(path)
}

func (f *symlinkFaultFS) Stat(path string) (update.FileInfo, error) {
	if err, ok := f.statErr[path]; ok {
		return nil, err
	}

	return f.memFS.Stat(path)
}

func (f *symlinkFaultFS) Symlink(target, link string) error {
	if err, ok := f.symlinkErr[link]; ok {
		return err
	}

	return f.memFS.Symlink(target, link)
}

// assertCleanupPropertyOutcome checks cleanup removed EXACTLY wantRemoved
// and left everything in wantKept alone.
func assertCleanupPropertyOutcome(
	rt *rapid.T,
	fileSystem *memFS,
	removed []string,
	wantRemoved, wantKept map[string]bool,
) {
	gotRemoved := map[string]bool{}
	for _, r := range removed {
		gotRemoved[r] = true
	}

	if len(gotRemoved) != len(wantRemoved) {
		rt.Fatalf("removed count mismatch: got %v want %v", gotRemoved, wantRemoved)
	}

	for path := range wantRemoved {
		if !gotRemoved[path] {
			rt.Fatalf("expected %s to be removed, was not", path)
		}

		if _, stillSymlink := fileSystem.symlinks[path]; stillSymlink {
			rt.Fatalf("%s still a symlink after removal", path)
		}
	}

	for path := range wantKept {
		if gotRemoved[path] {
			rt.Fatalf("%s was removed but should have been kept", path)
		}
	}
}

// mustSymlink creates a symlink in the property test's fixture, failing the
// rapid run on the (never-expected) memFS.Symlink error.
func mustSymlink(rt *rapid.T, fileSystem *memFS, target, link string) {
	symlinkErr := fileSystem.Symlink(target, link)
	if symlinkErr != nil {
		rt.Fatalf("symlink: %v", symlinkErr)
	}
}

// seedCleanupPropertyEntry draws one surface-dir entry kind and seeds it
// into fileSystem, recording its path into wantRemoved or wantKept.
func seedCleanupPropertyEntry(
	rt *rapid.T,
	fileSystem *memFS,
	dir, root string,
	index int,
	wantRemoved, wantKept map[string]bool,
) {
	path := dir + "/e" + strconv.Itoa(index)

	kind := rapid.SampledFrom(
		[]string{"real-file", "real-dir", "healthy-link", "dangling-link", "foreign-link"},
	).Draw(rt, "kind"+strconv.Itoa(index))

	switch kind {
	case "real-file":
		fileSystem.files[path] = []byte("content" + strconv.Itoa(index))
		wantKept[path] = true
	case "real-dir":
		fileSystem.dirs[path] = true
		wantKept[path] = true
	case "healthy-link":
		target := root + "/skills/target" + strconv.Itoa(index)
		fileSystem.dirs[target] = true
		mustSymlink(rt, fileSystem, target, path)
		wantKept[path] = true
	case "dangling-link":
		target := root + "/skills/gone" + strconv.Itoa(index) // never seeded
		mustSymlink(rt, fileSystem, target, path)
		wantRemoved[path] = true
	case "foreign-link":
		target := "/elsewhere/thing" + strconv.Itoa(index) // outside root
		mustSymlink(rt, fileSystem, target, path)
		wantKept[path] = true
	}
}

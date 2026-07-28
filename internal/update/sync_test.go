package update_test

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/update"
)

func TestApplyEngramSyncOps_CreateMkdirError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.files["/src/a.md"] = []byte("content")

	mkdirErr := errors.New("mkdir boom")
	fileSystem := &engramFaultFS{memFS: base, mkdirErr: map[string]error{"/root/skills/a": mkdirErr}}

	ops := []update.EngramSyncOp{
		{Kind: update.EngramSyncCreate, RelPath: "skills/a/SKILL.md", AbsPath: "/root/skills/a/SKILL.md", Src: "/src/a.md"},
	}

	applyErr := update.ExportApplyEngramSyncOps(fileSystem, ops)
	g.Expect(applyErr).To(HaveOccurred())
	g.Expect(applyErr.Error()).To(ContainSubstring("mkdir boom"))
}

func TestApplyEngramSyncOps_CreateReadFileError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()

	readErr := errors.New("disk read boom")
	fileSystem := &engramFaultFS{memFS: base, readFileErr: map[string]error{"/src/a.md": readErr}}

	ops := []update.EngramSyncOp{
		{Kind: update.EngramSyncCreate, RelPath: "skills/a/SKILL.md", AbsPath: "/root/skills/a/SKILL.md", Src: "/src/a.md"},
	}

	applyErr := update.ExportApplyEngramSyncOps(fileSystem, ops)
	g.Expect(applyErr).To(HaveOccurred())
	g.Expect(applyErr.Error()).To(ContainSubstring("disk read boom"))
}

func TestApplyEngramSyncOps_CreateWriteFileError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.files["/src/a.md"] = []byte("content")

	writeErr := errors.New("write boom")
	fileSystem := &engramFaultFS{
		memFS:        base,
		writeFileErr: map[string]error{"/root/skills/a/SKILL.md": writeErr},
	}

	ops := []update.EngramSyncOp{
		{Kind: update.EngramSyncCreate, RelPath: "skills/a/SKILL.md", AbsPath: "/root/skills/a/SKILL.md", Src: "/src/a.md"},
	}

	applyErr := update.ExportApplyEngramSyncOps(fileSystem, ops)
	g.Expect(applyErr).To(HaveOccurred())
	g.Expect(applyErr.Error()).To(ContainSubstring("write boom"))
}

func TestApplyEngramSyncOps_DeleteRemoveAllError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()

	removeErr := errors.New("remove boom")
	fileSystem := &engramFaultFS{
		memFS:        base,
		removeAllErr: map[string]error{"/root/skills/old/SKILL.md": removeErr},
	}

	ops := []update.EngramSyncOp{
		{Kind: update.EngramSyncDelete, RelPath: "skills/old/SKILL.md", AbsPath: "/root/skills/old/SKILL.md"},
	}

	applyErr := update.ExportApplyEngramSyncOps(fileSystem, ops)
	g.Expect(applyErr).To(HaveOccurred())
	g.Expect(applyErr.Error()).To(ContainSubstring("remove boom"))
}

func TestPlanAndApplyEngramRootSync_Property_RootMatchesIntendedSet(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		intended, preExisting := drawEngramRootFixture(rt)

		fileSystem := newMemFS()
		fileSystem.files[engramRootSyncFixtureOutside] = []byte("must-not-touch")

		rootIntended := seedIntendedRootFiles(fileSystem, intended)
		seedPreExistingRootFiles(fileSystem, preExisting)

		ops, planErr := update.ExportPlanEngramRootSync(engramRootSyncFixtureRoot, rootIntended, true, true, fileSystem)
		if planErr != nil {
			rt.Fatalf("plan: %v", planErr)
		}

		applyErr := update.ExportApplyEngramSyncOps(fileSystem, ops)
		if applyErr != nil {
			rt.Fatalf("apply: %v", applyErr)
		}

		pruned, pruneErr := update.ExportPruneEmptyDirs(engramRootSyncFixtureRoot, rootIntended, true, true, fileSystem)
		if pruneErr != nil {
			rt.Fatalf("prune: %v", pruneErr)
		}

		pruneApplyErr := update.ExportApplyPrunedDirs(fileSystem, engramRootSyncFixtureRoot, pruned)
		if pruneApplyErr != nil {
			rt.Fatalf("apply prune: %v", pruneApplyErr)
		}

		assertRootMatchesIntended(rt, fileSystem, intended)
		assertNothingTouchedOutsideRoot(rt, fileSystem)
		assertNoEmptyDirsInManagedSubtrees(rt, fileSystem)
	})
}

// --- 3.1: engram-owned root + ownership marker -----------------------------

func TestRun_EngramRoot_AbsentCreatesRootAndMarker(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/learn"] = true
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())
	g.Expect(harness.EngramRoot).To(Equal(home + "/.claude/engram"))
	g.Expect(harness.EngramDeletionRefused).To(BeFalse())
	g.Expect(harness.EngramUnattributable).To(BeEmpty())

	_, markerWritten := fileSystem.written[home+"/.claude/engram/.engram-owned"]
	g.Expect(markerWritten).To(BeTrue())

	written, ok := fileSystem.written[home+"/.claude/engram/skills/learn/SKILL.md"]
	g.Expect(ok).To(BeTrue())
	g.Expect(written).To(Equal([]byte("learn skill")))
}

func TestRun_EngramRoot_AllHarnessesGetOwnRoots(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.dirs[home+"/.config/opencode"] = true
	fileSystem.dirs[home+"/.pi"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(3))

	want := map[update.Harness]string{
		update.HarnessClaude:   home + "/.claude/engram",
		update.HarnessOpencode: home + "/.config/opencode/engram",
		update.HarnessPi:       home + "/.pi/agent/engram",
	}

	for _, harness := range report.Harnesses {
		g.Expect(harness.Err).NotTo(HaveOccurred())
		g.Expect(harness.EngramRoot).To(Equal(want[harness.Name]), string(harness.Name))
	}
}

func TestRun_EngramRoot_ApplyMarkerWriteError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	base.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	base.dirs["/repo/agent-instructions/skills"] = true

	root := home + "/.claude/engram"

	writeErr := errors.New("marker write boom")
	fileSystem := &engramFaultFS{memFS: base, writeFileErr: map[string]error{root + "/.engram-owned": writeErr}}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(HaveOccurred())
	g.Expect(report.Harnesses[0].Err.Error()).To(ContainSubstring("marker write boom"))
}

func TestRun_EngramRoot_ApplyMkdirRootError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	base.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	base.dirs["/repo/agent-instructions/skills"] = true

	root := home + "/.claude/engram"

	mkdirErr := errors.New("mkdir root boom")
	fileSystem := &engramFaultFS{memFS: base, mkdirErr: map[string]error{root: mkdirErr}}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(HaveOccurred())
	g.Expect(report.Harnesses[0].Err.Error()).To(ContainSubstring("mkdir root boom"))
}

func TestRun_EngramRoot_ApplySyncOpWriteError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	base.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	base.dirs["/repo/agent-instructions/skills"] = true
	base.dirs["/repo/agent-instructions/skills/learn"] = true
	base.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")

	root := home + "/.claude/engram"

	writeErr := errors.New("skill write boom")
	fileSystem := &engramFaultFS{
		memFS:        base,
		writeFileErr: map[string]error{root + "/skills/learn/SKILL.md": writeErr},
	}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(HaveOccurred())
	g.Expect(report.Harnesses[0].Err.Error()).To(ContainSubstring("skill write boom"))
}

func TestRun_EngramRoot_DryRun_NoWrites(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/learn"] = true
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())
	g.Expect(harness.EngramRoot).To(Equal(home + "/.claude/engram"))

	// No writes at all under dry-run, including the engram root and marker.
	g.Expect(fileSystem.written).To(BeEmpty())

	_, rootNowExists := fileSystem.dirs[home+"/.claude/engram"]
	g.Expect(rootNowExists).To(BeFalse())
}

// TestRun_EngramRoot_DryRun_PrunePreviewedNotApplied checks the prune half of
// D4 follows the same dry-run contract as sync-deletion itself: the pruned
// directory is PREVIEWED in the report (same field, dry-run-prefixed by the
// CLI render layer) but nothing is actually removed from disk.
func TestRun_EngramRoot_DryRun_PrunePreviewedNotApplied(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true

	root := home + "/.claude/engram"
	fileSystem.dirs[root] = true
	fileSystem.files[root+"/.engram-owned"] = []byte{}
	fileSystem.dirs[root+"/skills/oldskill"] = true
	fileSystem.files[root+"/skills/oldskill/SKILL.md"] = []byte("stale content")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())

	// Previewed: both the file deletion and the resulting empty-dir prune.
	g.Expect(harness.EngramSyncDeleted).To(ConsistOf(
		filepath.Join("skills", "oldskill", "SKILL.md"),
		filepath.Join("skills", "oldskill"),
	))

	// Nothing actually removed under dry-run.
	g.Expect(fileSystem.removed).To(BeEmpty())
	g.Expect(fileSystem.files[root+"/skills/oldskill/SKILL.md"]).To(Equal([]byte("stale content")))
	g.Expect(fileSystem.dirs[root+"/skills/oldskill"]).To(BeTrue())
}

// --- 3.3: guidance opt-in gates management, not removal ---------------------

func TestRun_EngramRoot_GuidanceNotOptedIn_SubtreeUntouched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/learn"] = true
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")
	// Source guidance exists, but this harness's CLAUDE.md does not import
	// it and --with-guidance is not passed — guidance stays unmanaged.
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("recall guidance")
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true

	root := home + "/.claude/engram"
	fileSystem.dirs[root] = true
	fileSystem.files[root+"/.engram-owned"] = []byte{}
	fileSystem.files[root+"/guidance/stale.md"] = []byte("leftover from a prior opted-in sync")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())
	g.Expect(harness.EngramSyncDeleted).To(BeEmpty())

	g.Expect(fileSystem.files[root+"/guidance/stale.md"]).To(Equal([]byte("leftover from a prior opted-in sync")))
}

func TestRun_EngramRoot_GuidanceOptedIn_SyncsIncludingDeletion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("fresh recall guidance")
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true

	root := home + "/.claude/engram"
	fileSystem.dirs[root] = true
	fileSystem.files[root+"/.engram-owned"] = []byte{}
	fileSystem.files[root+"/guidance/stale.md"] = []byte("no longer in source")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{WithGuidance: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())
	g.Expect(harness.EngramSyncDeleted).To(ConsistOf(filepath.Join("guidance", "stale.md")))

	_, staleStillThere := fileSystem.files[root+"/guidance/stale.md"]
	g.Expect(staleStillThere).To(BeFalse())

	written, ok := fileSystem.written[root+"/guidance/recall.md"]
	g.Expect(ok).To(BeTrue())
	g.Expect(written).To(Equal([]byte("fresh recall guidance")))
}

func TestRun_EngramRoot_MarkerAbsent_ClaudeFlatGuidanceIsAdoptableNotUnattributable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("fresh recall guidance")
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true

	// Pre-existing engram root WITHOUT the marker: exactly today's Claude
	// state after a prior --with-guidance run (flat guidance .md at the
	// GuidanceTargetRel path, which equals EngramRootRel for Claude — D1).
	root := home + "/.claude/engram"
	fileSystem.dirs[root] = true
	fileSystem.files[root+"/recall.md"] = []byte("old recall guidance")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{WithGuidance: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())

	// The flat file's basename matches an intended guidance basename — it
	// must NOT be reported as an unattributable stray.
	g.Expect(harness.EngramUnattributable).To(BeEmpty())
	// Marker-less root: deletion is still refused this run.
	g.Expect(harness.EngramDeletionRefused).To(BeTrue())

	// The intended (subtree-scoped) path was created with fresh content.
	written, ok := fileSystem.written[root+"/guidance/recall.md"]
	g.Expect(ok).To(BeTrue())
	g.Expect(written).To(Equal([]byte("fresh recall guidance")))

	// The real flat file is ADOPTED (task 5.1/5.3): replaced by a compat
	// symlink into the canonical guidance/ copy — the repo is the source of
	// truth, so the stale flat content is discarded, not refreshed in place.
	target, isSymlink := fileSystem.symlinks[root+"/recall.md"]
	g.Expect(isSymlink).To(BeTrue())
	g.Expect(target).To(Equal(root + "/guidance/recall.md"))
	g.Expect(harness.EngramAdopted).To(ConsistOf(root + "/recall.md"))

	// The adoption pass stamps the marker this run.
	_, markerWritten := fileSystem.written[root+"/.engram-owned"]
	g.Expect(markerWritten).To(BeTrue())
}

func TestRun_EngramRoot_MarkerAbsent_RefusesDeletionAndReportsUnattributable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/learn"] = true
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")

	root := home + "/.claude/engram"
	fileSystem.dirs[root] = true
	fileSystem.files[root+"/mystery.md"] = []byte("nobody knows")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())
	g.Expect(harness.EngramDeletionRefused).To(BeTrue())
	g.Expect(harness.EngramUnattributable).To(ConsistOf("mystery.md"))
	g.Expect(harness.EngramSyncDeleted).To(BeEmpty())

	// Non-destructive create still applies.
	written, ok := fileSystem.written[root+"/skills/learn/SKILL.md"]
	g.Expect(ok).To(BeTrue())
	g.Expect(written).To(Equal([]byte("learn skill")))

	// The stray was left untouched, never deleted.
	g.Expect(fileSystem.files[root+"/mystery.md"]).To(Equal([]byte("nobody knows")))

	// The D6 adoption pass DOES stamp the marker this run, even though
	// sync-deletion of managed-subtree drift stays refused this run.
	_, markerWritten := fileSystem.written[root+"/.engram-owned"]
	g.Expect(markerWritten).To(BeTrue())
}

func TestRun_EngramRoot_MarkerPresent_SyncsIncludingDeletion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/learn"] = true
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill v2")

	root := home + "/.claude/engram"
	fileSystem.dirs[root] = true
	fileSystem.files[root+"/.engram-owned"] = []byte{}
	fileSystem.files[root+"/skills/oldskill/SKILL.md"] = []byte("stale content")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())
	g.Expect(harness.EngramDeletionRefused).To(BeFalse())
	// The stale file's deletion also empties its directory — pruned in the
	// same run and reported alongside the file (D4 continued).
	g.Expect(harness.EngramSyncDeleted).To(ConsistOf(
		filepath.Join("skills", "oldskill", "SKILL.md"),
		filepath.Join("skills", "oldskill"),
	))

	_, staleStillThere := fileSystem.files[root+"/skills/oldskill/SKILL.md"]
	g.Expect(staleStillThere).To(BeFalse())

	_, staleDirStillThere := fileSystem.dirs[root+"/skills/oldskill"]
	g.Expect(staleDirStillThere).To(BeFalse())

	written, ok := fileSystem.written[root+"/skills/learn/SKILL.md"]
	g.Expect(ok).To(BeTrue())
	g.Expect(written).To(Equal([]byte("learn skill v2")))
}

func TestRun_EngramRoot_MarkerStatError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	base.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	base.dirs["/repo/agent-instructions/skills"] = true

	root := home + "/.claude/engram"
	base.dirs[root] = true

	statErr := errors.New("marker stat boom")
	fileSystem := &engramFaultFS{memFS: base, statErr: map[string]error{root + "/.engram-owned": statErr}}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(HaveOccurred())
	g.Expect(report.Harnesses[0].Err.Error()).To(ContainSubstring("marker stat boom"))
}

func TestRun_EngramRoot_PlanReadDirError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	base.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	base.dirs["/repo/agent-instructions/skills"] = true
	base.dirs["/repo/agent-instructions/skills/learn"] = true
	base.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill")

	root := home + "/.claude/engram"
	base.dirs[root] = true
	base.files[root+"/.engram-owned"] = []byte{}

	readDirErr := errors.New("subtree readdir boom")
	fileSystem := &engramFaultFS{memFS: base, readDirErr: map[string]error{root + "/skills": readDirErr}}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(HaveOccurred())
	g.Expect(report.Harnesses[0].Err.Error()).To(ContainSubstring("subtree readdir boom"))
}

// TestRun_EngramRoot_RemovedSkillPrunesSubtreeAndSurfaceLink is the spec
// regression for "Removed source artifact disappears on next update": a
// skill deployed in one Run(), then deleted from the source before the NEXT
// Run() call on the same engram root, must have both its file AND its now-
// empty directory removed from the engram-owned root, AND its harness-visible
// symlink removed — all within that second, single Run() call. Before the
// fix, sync deletion removed only SKILL.md, leaving the emptied "please/"
// directory behind; cleanupDanglingLinks then Stat'd that still-present
// (though hollow) directory as a healthy target and left the surface symlink
// in place — a visibly present but hollow skill.
func TestRun_EngramRoot_RemovedSkillPrunesSubtreeAndSurfaceLink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/please"] = true
	fileSystem.files["/repo/agent-instructions/skills/please/SKILL.md"] = []byte("please skill")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	root := home + "/.claude/engram"
	surfaceLink := home + "/.claude/skills/please"

	// Run 1: deploy the skill.
	firstReport, firstErr := updater.Run(context.Background(), update.Options{})
	g.Expect(firstErr).NotTo(HaveOccurred())
	g.Expect(firstReport.Harnesses).To(HaveLen(1))
	g.Expect(firstReport.Harnesses[0].Err).NotTo(HaveOccurred())

	_, rootFileWritten := fileSystem.written[root+"/skills/please/SKILL.md"]
	g.Expect(rootFileWritten).To(BeTrue())

	firstTarget, firstIsSymlink := fileSystem.symlinks[surfaceLink]
	g.Expect(firstIsSymlink).To(BeTrue())
	g.Expect(firstTarget).To(Equal(root + "/skills/please"))

	// Between runs: the skill is deleted from source.
	delete(fileSystem.files, "/repo/agent-instructions/skills/please/SKILL.md")
	delete(fileSystem.dirs, "/repo/agent-instructions/skills/please")

	// Run 2: same session, same engram root — the removal must propagate
	// AND the dangling surface link must be cleaned up in this SAME run.
	secondReport, secondErr := updater.Run(context.Background(), update.Options{})
	g.Expect(secondErr).NotTo(HaveOccurred())
	g.Expect(secondReport.Harnesses).To(HaveLen(1))

	harness := secondReport.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())
	g.Expect(harness.EngramSyncDeleted).To(ConsistOf(
		filepath.Join("skills", "please", "SKILL.md"),
		filepath.Join("skills", "please"),
	))

	// The skill's whole subtree — file AND directory — is gone from the root.
	_, rootFileStillThere := fileSystem.files[root+"/skills/please/SKILL.md"]
	g.Expect(rootFileStillThere).To(BeFalse())

	_, rootDirStillThere := fileSystem.dirs[root+"/skills/please"]
	g.Expect(rootDirStillThere).To(BeFalse())

	_, statErr := fileSystem.Stat(root + "/skills/please")
	g.Expect(statErr).To(HaveOccurred())

	// The now-target-less harness-visible symlink is removed in this run.
	g.Expect(harness.DanglingLinksRemoved).To(ConsistOf(surfaceLink))

	_, surfaceLinkStillThere := fileSystem.symlinks[surfaceLink]
	g.Expect(surfaceLinkStillThere).To(BeFalse())
}

func TestRun_EngramRoot_RootStatError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	base.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	base.dirs["/repo/agent-instructions/skills"] = true

	root := home + "/.claude/engram"

	statErr := errors.New("root stat boom")
	fileSystem := &engramFaultFS{memFS: base, statErr: map[string]error{root: statErr}}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(HaveOccurred())
	g.Expect(report.Harnesses[0].Err.Error()).To(ContainSubstring("root stat boom"))
}

func TestRun_EngramRoot_UnattributableReadDirError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	base.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	base.dirs["/repo/agent-instructions/skills"] = true

	root := home + "/.claude/engram"
	base.dirs[root] = true
	// No marker present → falls into the unattributable top-level scan.

	readDirErr := errors.New("root readdir boom")
	fileSystem := &engramFaultFS{memFS: base, readDirErr: map[string]error{root: readDirErr}}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(HaveOccurred())
	g.Expect(report.Harnesses[0].Err.Error()).To(ContainSubstring("root readdir boom"))
}

// unexported constants.
const (
	engramRootSyncFixtureOutside = "/home/joe/.claude/skills/untouched.md"
	engramRootSyncFixtureRoot    = "/home/joe/.claude/engram"
)

// --- error-path coverage: sync-engine failures memFS alone cannot reach ----

// engramFaultFS wraps memFS and injects a configurable real I/O error for
// specific operations at specific paths — exercises the sync engine's error
// branches (a genuine failure, distinct from memFS's "not found").
type engramFaultFS struct {
	*memFS

	statErr      map[string]error
	readDirErr   map[string]error
	mkdirErr     map[string]error
	writeFileErr map[string]error
	readFileErr  map[string]error
	removeAllErr map[string]error
}

func (f *engramFaultFS) MkdirAll(path string, perm fs.FileMode) error {
	if err, ok := f.mkdirErr[path]; ok {
		return err
	}

	return f.memFS.MkdirAll(path, perm)
}

func (f *engramFaultFS) ReadDir(path string) ([]update.DirEntry, error) {
	if err, ok := f.readDirErr[path]; ok {
		return nil, err
	}

	return f.memFS.ReadDir(path)
}

func (f *engramFaultFS) ReadFile(path string) ([]byte, error) {
	if err, ok := f.readFileErr[path]; ok {
		return nil, err
	}

	return f.memFS.ReadFile(path)
}

func (f *engramFaultFS) RemoveAll(path string) error {
	if err, ok := f.removeAllErr[path]; ok {
		return err
	}

	return f.memFS.RemoveAll(path)
}

func (f *engramFaultFS) Stat(path string) (update.FileInfo, error) {
	if err, ok := f.statErr[path]; ok {
		return nil, err
	}

	return f.memFS.Stat(path)
}

func (f *engramFaultFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if err, ok := f.writeFileErr[path]; ok {
		return err
	}

	return f.memFS.WriteFile(path, data, perm)
}

// --- 3.2: pure diff engine, rapid property ----------------------------------

// intendedFixtureFile is a property-test-local description of one intended
// engram-root file (relative path under a managed subtree, plus its wanted
// content) — the source bytes are seeded separately per draw.
type intendedFixtureFile struct {
	relPath string
	content string
}

// assertDirNotEmpty fails the rapid run if dir holds zero entries.
func assertDirNotEmpty(rt *rapid.T, fileSystem *memFS, dir string) {
	entries, readErr := fileSystem.ReadDir(dir)
	if readErr != nil {
		rt.Fatalf("readdir %s: %v", dir, readErr)
	}

	if len(entries) == 0 {
		rt.Fatalf("empty directory left behind after sync: %s", dir)
	}
}

// assertNoEmptyDirsInManagedSubtrees checks that, after sync+prune, no
// directory NESTED under any managed subtree root (skills/, commands/,
// guidance/) holds zero entries — pruneEmptyDirs' invariant (a subtree root
// itself is exempt: it is never pruned and legitimately may be empty, e.g.
// no commands were ever intended this run).
func assertNoEmptyDirsInManagedSubtrees(rt *rapid.T, fileSystem *memFS) {
	for _, subtree := range []string{"skills", "commands", "guidance"} {
		assertSubtreeHasNoEmptyDescendantDirs(rt, fileSystem, filepath.Join(engramRootSyncFixtureRoot, subtree))
	}
}

// assertNothingTouchedOutsideRoot checks the outside-root sentinel file is
// unchanged and no write landed anywhere but under the root or the fixture's
// own seeded source/outside paths.
func assertNothingTouchedOutsideRoot(rt *rapid.T, fileSystem *memFS) {
	if string(fileSystem.files[engramRootSyncFixtureOutside]) != "must-not-touch" {
		rt.Fatalf("outside path was modified")
	}

	for path := range fileSystem.files {
		if path == engramRootSyncFixtureRoot || strings.HasPrefix(path, engramRootSyncFixtureRoot+"/") {
			continue
		}

		if path == engramRootSyncFixtureOutside || strings.HasPrefix(path, "/src/") {
			continue // fixture-seeded outside/source paths, not root paths
		}

		rt.Fatalf("unexpected write outside root: %s", path)
	}
}

// assertRootMatchesIntended checks every file under the root's managed
// subtrees matches the intended set exactly (same paths, same content) and
// that no intended file went missing.
func assertRootMatchesIntended(rt *rapid.T, fileSystem *memFS, intended []intendedFixtureFile) {
	want := map[string]string{}
	for _, item := range intended {
		want[filepath.Join(engramRootSyncFixtureRoot, filepath.FromSlash(item.relPath))] = item.content
	}

	for _, subtree := range []string{"skills", "commands", "guidance"} {
		files, listErr := update.ExportListSubtreeFiles(filepath.Join(engramRootSyncFixtureRoot, subtree), fileSystem)
		if listErr != nil {
			rt.Fatalf("list %s: %v", subtree, listErr)
		}

		for _, rel := range files {
			assertSubtreeFileMatches(rt, fileSystem, want, filepath.Join(engramRootSyncFixtureRoot, subtree, rel))
		}
	}

	if len(want) != 0 {
		rt.Fatalf("intended files missing after apply: %v", want)
	}
}

// assertSubtreeFileMatches checks one file found under a managed subtree
// after apply: it must be intended and hold the intended content, and is
// removed from want on success (so leftover want entries mean a missing
// intended file).
func assertSubtreeFileMatches(rt *rapid.T, fileSystem *memFS, want map[string]string, abs string) {
	gotContent, ok := fileSystem.files[abs]
	if !ok {
		rt.Fatalf("file %s missing after apply", abs)
	}

	wantContent, wanted := want[abs]
	if !wanted {
		rt.Fatalf("unexpected file survived sync: %s", abs)
	}

	if string(gotContent) != wantContent {
		rt.Fatalf("content mismatch at %s: got %q want %q", abs, gotContent, wantContent)
	}

	delete(want, abs)
}

// assertSubtreeHasNoEmptyDescendantDirs walks dir (a managed subtree root)
// and fails if any DESCENDANT directory holds zero entries.
func assertSubtreeHasNoEmptyDescendantDirs(rt *rapid.T, fileSystem *memFS, dir string) {
	entries, readErr := fileSystem.ReadDir(dir)
	if readErr != nil {
		return // subtree not created this run — nothing to check
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		childPath := filepath.Join(dir, entry.Name())
		assertDirNotEmpty(rt, fileSystem, childPath)
		assertSubtreeHasNoEmptyDescendantDirs(rt, fileSystem, childPath)
	}
}

// drawEngramRootFixture generates an arbitrary intended set (0-4 files
// spread across the skills/, commands/, guidance/ subtrees, each with
// distinct content) and an arbitrary pre-existing root state (0-4 entries:
// some reusing an intended path with stale content — the overwrite case —
// and some brand-new stray paths under the managed subtrees — the delete
// case).
func drawEngramRootFixture(rt *rapid.T) ([]intendedFixtureFile, map[string]string) {
	intendedCount := rapid.IntRange(0, 4).Draw(rt, "intendedCount")

	seen := map[string]bool{}
	intended := make([]intendedFixtureFile, 0, intendedCount)

	for i := range intendedCount {
		rel := drawSubtreeRelPath(rt, "intended", i, "")
		if seen[rel] {
			continue
		}

		seen[rel] = true

		content := rapid.StringMatching(`[a-z0-9]{1,10}`).Draw(rt, "content"+strconv.Itoa(i))
		intended = append(intended, intendedFixtureFile{relPath: rel, content: content})
	}

	preExisting := map[string]string{}
	preExistingCount := rapid.IntRange(0, 4).Draw(rt, "preExistingCount")

	for i := range preExistingCount {
		if len(intended) > 0 && rapid.Bool().Draw(rt, "reuseIntended"+strconv.Itoa(i)) {
			idx := rapid.IntRange(0, len(intended)-1).Draw(rt, "reuseIdx"+strconv.Itoa(i))
			staleContent := rapid.StringMatching(`[a-z0-9]{1,10}`).Draw(rt, "staleContent"+strconv.Itoa(i))
			preExisting[intended[idx].relPath] = staleContent

			continue
		}

		// Distinct suffix so a stray never accidentally collides with an
		// intended-set path drawn from the same small alphabet.
		rel := drawSubtreeRelPath(rt, "stray", i, "-x")
		preExisting[rel] = "stale-" + strconv.Itoa(i)
	}

	return intended, preExisting
}

// drawSubtreeRelPath draws one relative path under a random managed subtree
// (skills/<name>/SKILL.md, commands/<name+suffix>.md, guidance/<name+suffix>.md).
func drawSubtreeRelPath(rt *rapid.T, label string, i int, suffix string) string {
	subtree := rapid.SampledFrom([]string{"skills", "commands", "guidance"}).
		Draw(rt, label+"-subtree"+strconv.Itoa(i))
	name := rapid.StringMatching(`[a-z]{2,6}`).Draw(rt, label+"-name"+strconv.Itoa(i)) + suffix

	if subtree == "skills" {
		return "skills/" + name + "/SKILL.md"
	}

	return subtree + "/" + name + ".md"
}

// seedIntendedRootFiles writes each intended item's content to a distinct
// fixture source path and returns the corresponding sync-engine intended set.
func seedIntendedRootFiles(fileSystem *memFS, intended []intendedFixtureFile) []update.ExportIntendedRootFile {
	rootIntended := make([]update.ExportIntendedRootFile, 0, len(intended))

	for i, item := range intended {
		src := "/src/" + strconv.Itoa(i) + ".md"
		fileSystem.files[src] = []byte(item.content)
		rootIntended = append(rootIntended, update.ExportIntendedRootFile{
			RelPath: filepath.FromSlash(item.relPath),
			Src:     src,
		})
	}

	return rootIntended
}

// seedPreExistingRootFiles writes the pre-existing root fixture directly
// under the fixture root. MkdirAll on each parent (not a bare map write)
// matters: it is what makes a stray file's directory a REAL, explicitly-
// registered entry in the fixture — exactly how it would exist after a
// genuine prior sync wrote it (every real write goes through
// applyEngramSyncOp's MkdirAll) — so deleting the stray's last file leaves
// a directory that PruneEmptyDirs must remove, not one memFS's prefix-based
// existence check would already treat as gone on its own.
func seedPreExistingRootFiles(fileSystem *memFS, preExisting map[string]string) {
	for rel, content := range preExisting {
		abs := filepath.Join(engramRootSyncFixtureRoot, filepath.FromSlash(rel))

		mkdirErr := fileSystem.MkdirAll(filepath.Dir(abs), 0o755)
		if mkdirErr != nil {
			panic(mkdirErr) // memFS.MkdirAll never errors
		}

		fileSystem.files[abs] = []byte(content)
	}
}

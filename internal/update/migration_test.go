package update_test

// migration_test.go covers task group 5 (D6 first-sync migration): adoption
// of previously-copied intended-set surface artifacts into symlinks, root
// marker stamping on adoption, Claude guidance compat symlinks, and dry-run
// no-trace. Sibling to sync_test.go (group 3: root sync engine) and
// symlink_test.go (group 4: symlink materialization + cleanup).

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

// --- 5.1: materializeOrAdopt (unit) -----------------------------------------

func TestMaterializeOrAdopt_AbsentCreatesLinkNotAdopted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()

	adopted, err := update.ExportMaterializeOrAdopt(
		fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(adopted).To(BeFalse())

	target, ok := fileSystem.symlinks["/home/skills/recall"]
	g.Expect(ok).To(BeTrue())
	g.Expect(target).To(Equal("/home/engram/skills/recall"))
}

func TestMaterializeOrAdopt_DryRun_RealDir_PreviewsAdoptionWithoutWriting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills/recall"] = true
	fileSystem.files["/home/skills/recall/SKILL.md"] = []byte("real pre-migration copy")

	adopted, err := update.ExportMaterializeOrAdopt(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", true)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(adopted).To(BeTrue())

	g.Expect(fileSystem.removed).To(BeEmpty())
	g.Expect(fileSystem.dirs["/home/skills/recall"]).To(BeTrue())
	g.Expect(fileSystem.files["/home/skills/recall/SKILL.md"]).To(Equal([]byte("real pre-migration copy")))
	g.Expect(fileSystem.symlinks["/home/skills/recall"]).To(BeEmpty())
}

func TestMaterializeOrAdopt_ExistingCorrectSymlink_NotAdopted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	symlinkErr := fileSystem.Symlink("/home/engram/skills/recall", "/home/skills/recall")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	adopted, err := update.ExportMaterializeOrAdopt(
		fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(adopted).To(BeFalse())
	g.Expect(fileSystem.removed).To(BeEmpty())
}

func TestMaterializeOrAdopt_RealDir_ReplacedWithSymlink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills/recall"] = true
	fileSystem.files["/home/skills/recall/SKILL.md"] = []byte("real pre-migration copy")

	adopted, err := update.ExportMaterializeOrAdopt(
		fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(adopted).To(BeTrue())

	g.Expect(fileSystem.removed).To(ContainElement("/home/skills/recall"))
	g.Expect(fileSystem.dirs["/home/skills/recall"]).To(BeFalse())
	g.Expect(fileSystem.files["/home/skills/recall/SKILL.md"]).To(BeNil())

	target, ok := fileSystem.symlinks["/home/skills/recall"]
	g.Expect(ok).To(BeTrue())
	g.Expect(target).To(Equal("/home/engram/skills/recall"))
}

func TestMaterializeOrAdopt_RealFile_ReplacedWithSymlink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/home/commands/recall.md"] = []byte("real pre-migration copy")

	adopted, err := update.ExportMaterializeOrAdopt(
		fileSystem, "/home/commands/recall.md", "/home/engram/commands/recall.md", false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(adopted).To(BeTrue())

	g.Expect(fileSystem.removed).To(ContainElement("/home/commands/recall.md"))

	target, ok := fileSystem.symlinks["/home/commands/recall.md"]
	g.Expect(ok).To(BeTrue())
	g.Expect(target).To(Equal("/home/engram/commands/recall.md"))
}

func TestMaterializeOrAdopt_RemoveAllError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/skills/recall"] = true

	removeErr := errors.New("remove boom")
	fileSystem := &symlinkFaultFS{memFS: base, removeAllErr: map[string]error{"/home/skills/recall": removeErr}}

	_, err := update.ExportMaterializeOrAdopt(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).To(MatchError(ContainSubstring("remove boom")))
}

func TestMaterializeOrAdopt_SymlinkCreateError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/skills/recall"] = true

	symlinkErr := errors.New("symlink boom")
	fileSystem := &symlinkFaultFS{memFS: base, symlinkErr: map[string]error{"/home/skills/recall": symlinkErr}}

	_, err := update.ExportMaterializeOrAdopt(fileSystem, "/home/skills/recall", "/home/engram/skills/recall", false)
	g.Expect(err).To(MatchError(ContainSubstring("symlink boom")))
}

func TestRun_Migration_ClaudeCompatLinkDeletedWhenGuidanceRemovedFromSource(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true

	root := home + "/.claude/engram"
	fileSystem.dirs[root] = true
	fileSystem.files[root+"/.engram-owned"] = []byte{}
	fileSystem.dirs[root+"/guidance"] = true
	fileSystem.files[root+"/guidance/gone.md"] = []byte("stale guidance")
	symlinkErr := fileSystem.Symlink(root+"/guidance/gone.md", root+"/gone.md")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

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

	// Sync deletes the now-unintended guidance file, and dangling-link
	// cleanup removes the compat link that pointed at it.
	g.Expect(harness.EngramSyncDeleted).To(ConsistOf(filepath.Join("guidance", "gone.md")))
	g.Expect(harness.DanglingLinksRemoved).To(ConsistOf(root + "/gone.md"))

	_, stillSymlink := fileSystem.symlinks[root+"/gone.md"]
	g.Expect(stillSymlink).To(BeFalse())
}

func TestRun_Migration_ClaudeFlatGuidanceFileAdopted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("fresh recall guidance")

	// Today's deployed state: a REAL flat guidance file, no marker on root.
	root := home + "/.claude/engram"
	fileSystem.dirs[root] = true
	fileSystem.files[root+"/recall.md"] = []byte("old recall guidance, flat-copied")

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

	// The real flat file is REPLACED by a compat symlink (adoption, not a
	// content comparison — the repo is the source of truth, D6).
	target, isSymlink := fileSystem.symlinks[root+"/recall.md"]
	g.Expect(isSymlink).To(BeTrue())
	g.Expect(target).To(Equal(root + "/guidance/recall.md"))
	g.Expect(fileSystem.written[root+"/guidance/recall.md"]).To(Equal([]byte("fresh recall guidance")))

	g.Expect(harness.EngramAdopted).To(ConsistOf(root + "/recall.md"))
	g.Expect(harness.EngramUnattributable).To(BeEmpty())
}

// --- 5.3: Claude guidance compat symlinks -----------------------------------

func TestRun_Migration_ClaudeGuidanceCompatSymlinkCreated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("claude recall guidance")

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

	root := home + "/.claude/engram"

	// The canonical copy lives under guidance/, synced by the root sync engine.
	g.Expect(fileSystem.written[root+"/guidance/recall.md"]).To(Equal([]byte("claude recall guidance")))

	// The flat compat path is a SYMLINK into it — no more flat-copy path.
	target, isSymlink := fileSystem.symlinks[root+"/recall.md"]
	g.Expect(isSymlink).To(BeTrue())
	g.Expect(target).To(Equal(root + "/guidance/recall.md"))
	g.Expect(fileSystem.files[root+"/recall.md"]).To(BeNil())
}

func TestRun_Migration_CopiedCommandFileAdoptsToSymlink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.config/opencode"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/commands"] = true
	fileSystem.files["/repo/agent-instructions/commands/recall.md"] = []byte("fresh recall cmd")

	fileSystem.dirs[home+"/.config/opencode/commands"] = true
	fileSystem.files[home+"/.config/opencode/commands/recall.md"] = []byte("stale pre-migration cmd")

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

	surfacePath := home + "/.config/opencode/commands/recall.md"
	engramPath := home + "/.config/opencode/engram/commands/recall.md"

	target, isSymlink := fileSystem.symlinks[surfacePath]
	g.Expect(isSymlink).To(BeTrue())
	g.Expect(target).To(Equal(engramPath))
	g.Expect(fileSystem.written[engramPath]).To(Equal([]byte("fresh recall cmd")))

	g.Expect(harness.EngramAdopted).To(ConsistOf(surfacePath))
	g.Expect(harness.SurfaceUnattributable).To(BeEmpty())
	g.Expect(harness.CommandFiles).To(ConsistOf("recall.md"))
}

// --- 5.1: adoption via a full Run() -----------------------------------------

func TestRun_Migration_CopiedSkillDirAdoptsToSymlink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/recall"] = true
	fileSystem.files["/repo/agent-instructions/skills/recall/SKILL.md"] = []byte("fresh recall skill")

	// Pre-sync copy-mode deploy: a REAL skill dir already sits at the surface.
	fileSystem.dirs[home+"/.claude/skills"] = true
	fileSystem.dirs[home+"/.claude/skills/recall"] = true
	fileSystem.files[home+"/.claude/skills/recall/SKILL.md"] = []byte("stale pre-migration copy")

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

	surfacePath := home + "/.claude/skills/recall"
	engramPath := home + "/.claude/engram/skills/recall"

	// Content now lives in the root; the surface path is a symlink.
	target, isSymlink := fileSystem.symlinks[surfacePath]
	g.Expect(isSymlink).To(BeTrue())
	g.Expect(target).To(Equal(engramPath))
	g.Expect(fileSystem.written[engramPath+"/SKILL.md"]).To(Equal([]byte("fresh recall skill")))

	// Adopted, not merely reported.
	g.Expect(harness.EngramAdopted).To(ConsistOf(surfacePath))
	g.Expect(harness.SurfaceUnattributable).To(BeEmpty())
}

// --- dry-run no-trace --------------------------------------------------------

func TestRun_Migration_DryRun_NeverSyncedHarness_LeavesNoTrace(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/recall"] = true
	fileSystem.files["/repo/agent-instructions/skills/recall/SKILL.md"] = []byte("recall skill")
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("recall guidance")

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{DryRun: true, WithGuidance: true})
	g.Expect(err).NotTo(HaveOccurred())

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())

	// Not one byte was written anywhere: no root, no marker, no symlink.
	g.Expect(fileSystem.written).To(BeEmpty())
	g.Expect(fileSystem.symlinks).To(BeEmpty())
	g.Expect(fileSystem.dirs[home+"/.claude/engram"]).To(BeFalse())
	g.Expect(fileSystem.dirs[home+"/.claude/skills"]).To(BeFalse())
}

func TestRun_Migration_DryRun_PreExistingMarkerlessRoot_LeavesNoTrace(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/recall"] = true
	fileSystem.files["/repo/agent-instructions/skills/recall/SKILL.md"] = []byte("fresh recall skill")

	fileSystem.dirs[home+"/.claude/skills"] = true
	fileSystem.dirs[home+"/.claude/skills/recall"] = true
	fileSystem.files[home+"/.claude/skills/recall/SKILL.md"] = []byte("stale pre-migration copy")

	root := home + "/.claude/engram"
	fileSystem.dirs[root] = true
	fileSystem.files[root+"/mystery.md"] = []byte("nobody knows")

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

	// The preview still reports what WOULD happen...
	g.Expect(harness.EngramAdopted).To(ConsistOf(home + "/.claude/skills/recall"))
	g.Expect(harness.EngramUnattributable).To(ConsistOf("mystery.md"))

	// ...but nothing was actually written or removed.
	g.Expect(fileSystem.written).To(BeEmpty())
	_, stillReal := fileSystem.files[home+"/.claude/skills/recall/SKILL.md"]
	g.Expect(stillReal).To(BeTrue())
	g.Expect(fileSystem.removed).To(BeEmpty())
	_, markerWritten := fileSystem.written[root+"/.engram-owned"]
	g.Expect(markerWritten).To(BeFalse())
}

// --- 5.1 (continued): root-level marker adoption ----------------------------

func TestRun_Migration_RootMarkerStampedAfterAdoptionPass(t *testing.T) {
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
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())

	// Unattributable stray listed, left in place, excluded from deletion.
	g.Expect(harness.EngramUnattributable).To(ConsistOf("mystery.md"))
	g.Expect(fileSystem.files[root+"/mystery.md"]).To(Equal([]byte("nobody knows")))
	g.Expect(harness.EngramSyncDeleted).To(BeEmpty())

	// The adoption pass stamps the marker this run.
	_, markerWritten := fileSystem.written[root+"/.engram-owned"]
	g.Expect(markerWritten).To(BeTrue())
}

func TestRun_Migration_SubsequentSyncTreatsAdoptedRootAsOwned(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/learn"] = true
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("learn skill v1")

	root := home + "/.claude/engram"
	fileSystem.dirs[root] = true
	fileSystem.files[root+"/mystery.md"] = []byte("nobody knows")

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: home, cwd: "/repo"},
		Spawn: noopSpawner{},
	}

	// First run: adoption pass stamps the marker.
	_, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	_, markerWritten := fileSystem.written[root+"/.engram-owned"]
	g.Expect(markerWritten).To(BeTrue())

	// Source drops "learn" and adds "recall": normal drift.
	delete(fileSystem.dirs, "/repo/agent-instructions/skills/learn")
	delete(fileSystem.files, "/repo/agent-instructions/skills/learn/SKILL.md")
	fileSystem.dirs["/repo/agent-instructions/skills/recall"] = true
	fileSystem.files["/repo/agent-instructions/skills/recall/SKILL.md"] = []byte("recall skill")

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())

	// Normal sync-deletion of intended-set drift now applies — the removed
	// skill's now-empty directory is pruned alongside its file (D4 continued).
	g.Expect(harness.EngramDeletionRefused).To(BeFalse())
	g.Expect(harness.EngramSyncDeleted).To(ConsistOf(
		filepath.Join("skills", "learn", "SKILL.md"),
		filepath.Join("skills", "learn"),
	))

	// The originally-listed unknown is NEVER deleted, even on a normal sync.
	g.Expect(fileSystem.files[root+"/mystery.md"]).To(Equal([]byte("nobody knows")))
}

// --- 5.2: stray surface reporting through a full Run() ----------------------

func TestRun_Migration_SurfaceStrayOutsideIntendedSet_ReportedNeverDeleted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs["/repo/agent-instructions/skills/recall"] = true
	fileSystem.files["/repo/agent-instructions/skills/recall/SKILL.md"] = []byte("recall skill")

	// A stray real skill dir whose name matches NO source skill at all.
	fileSystem.dirs[home+"/.claude/skills"] = true
	fileSystem.dirs[home+"/.claude/skills/user-own-tool"] = true
	fileSystem.files[home+"/.claude/skills/user-own-tool/NOTES.md"] = []byte("not engram's")

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

	strayPath := home + "/.claude/skills/user-own-tool"
	g.Expect(harness.SurfaceUnattributable).To(ConsistOf(strayPath))
	g.Expect(harness.EngramAdopted).To(BeEmpty())

	g.Expect(fileSystem.removed).NotTo(ContainElement(strayPath))
	g.Expect(fileSystem.dirs[strayPath]).To(BeTrue())
	g.Expect(fileSystem.files[strayPath+"/NOTES.md"]).To(Equal([]byte("not engram's")))
}

func TestSurfaceStrays_IntendedEntry_NotReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	fileSystem.dirs["/home/skills/recall"] = true

	strays, err := update.ExportSurfaceStrays(fileSystem, "/home/skills", map[string]bool{"recall": true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strays).To(BeEmpty())
}

func TestSurfaceStrays_LstatError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := newMemFS()
	base.dirs["/home/skills"] = true
	base.dirs["/home/skills/old-skill"] = true

	lstatErr := errors.New("lstat boom")
	fileSystem := &symlinkFaultFS{memFS: base, lstatErr: map[string]error{"/home/skills/old-skill": lstatErr}}

	_, err := update.ExportSurfaceStrays(fileSystem, "/home/skills", map[string]bool{})
	g.Expect(err).To(MatchError(ContainSubstring("lstat boom")))
}

func TestSurfaceStrays_MissingDir_NoStrays(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()

	strays, err := update.ExportSurfaceStrays(fileSystem, "/home/skills", map[string]bool{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strays).To(BeEmpty())
}

func TestSurfaceStrays_ReadDirError_Propagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	readErr := errors.New("readdir boom")
	fileSystem := &symlinkFaultFS{memFS: newMemFS(), readDirErr: map[string]error{"/home/skills": readErr}}

	_, err := update.ExportSurfaceStrays(fileSystem, "/home/skills", map[string]bool{})
	g.Expect(err).To(MatchError(ContainSubstring("readdir boom")))
}

// --- 5.2: surfaceStrays (unit) ----------------------------------------------

func TestSurfaceStrays_RealEntryNotIntended_Reported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	fileSystem.dirs["/home/skills/old-skill"] = true
	fileSystem.files["/home/skills/old-skill/SKILL.md"] = []byte("orphaned")

	strays, err := update.ExportSurfaceStrays(fileSystem, "/home/skills", map[string]bool{"recall": true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strays).To(ConsistOf("/home/skills/old-skill"))
}

func TestSurfaceStrays_SymlinkEntry_SkippedNotReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/skills"] = true
	symlinkErr := fileSystem.Symlink("/opt/other/skill", "/home/skills/foreign")
	g.Expect(symlinkErr).NotTo(HaveOccurred())

	strays, err := update.ExportSurfaceStrays(fileSystem, "/home/skills", map[string]bool{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strays).To(BeEmpty())
}

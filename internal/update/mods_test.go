package update_test

import (
	"context"
	"errors"
	"maps"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

func Test793d31b1_HarnessSpecs_OnlyClaudeCodeTakesMods(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs[modsTestHome+"/.claude"] = true
	fileSystem.dirs[modsTestHome+"/.pi"] = true

	specs, err := update.ExportDetectHarnesses(modsTestHome, fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(specs).To(HaveLen(2))

	if len(specs) != 2 {
		return
	}

	g.Expect(specs[0].Name).To(Equal(update.HarnessClaude))
	g.Expect(specs[0].TakesMods).To(BeTrue())
	g.Expect(specs[1].Name).To(Equal(update.HarnessPi))
	g.Expect(specs[1].TakesMods).To(BeFalse())
}

func Test793d31b1_PlanModCopies_ClaudeOnlyUnderItsSkillsDirectory(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/src/mods/engram-memory/.claude-plugin/plugin.json"] = []byte("{}")
	fileSystem.files["/src/mods/engram-memory/hooks/register.js"] = []byte("export {}")

	harnesses := []update.HarnessSpec{
		{Name: update.HarnessClaude, SkillsTargetRel: ".claude/skills", TakesMods: true},
		{Name: update.HarnessPi, SkillsTargetRel: ".pi/agent/skills"},
	}

	ops, err := update.ExportPlanModCopies("/src/mods", modsTestHome, harnesses, fileSystem)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(ops).To(HaveLen(2))

	for _, op := range ops {
		g.Expect(op.Harness).To(Equal(update.HarnessClaude))
		g.Expect(op.SkillDir).To(Equal("engram-memory"))
		g.Expect(op.Mod).To(BeTrue())
		g.Expect(op.Dst).To(HavePrefix(modsTestHome + "/.claude/skills/engram-memory/"))
	}

	g.Expect(ops[0].Src).To(Equal("/src/mods/engram-memory/.claude-plugin/plugin.json"))
	g.Expect(ops[0].Dst).To(Equal(modsTestHome + "/.claude/skills/engram-memory/.claude-plugin/plugin.json"))
}

func Test793d31b1_PlanModCopies_ListingFailurePropagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	wantErr := errors.New("disk failed")
	harnesses := []update.HarnessSpec{{Name: update.HarnessClaude, SkillsTargetRel: ".claude/skills", TakesMods: true}}

	failing := &readDirFailFS{memFS: newMemFS(), err: wantErr}

	_, err := update.ExportPlanModCopies("/src/mods", modsTestHome, harnesses, failing)
	g.Expect(err).To(MatchError(wantErr))
}

func Test793d31b1_PlanModCopies_MissingSourcePlansNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	harnesses := []update.HarnessSpec{{Name: update.HarnessClaude, SkillsTargetRel: ".claude/skills", TakesMods: true}}

	ops, err := update.ExportPlanModCopies("/src/mods", modsTestHome, harnesses, newMemFS())
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ops).To(BeEmpty())
}

func Test793d31b2_Run_ARemovedModIsSyncDeletedWithItsLink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := modsFixture(false)
	updater := modsUpdater(fileSystem, &fakeCmd{})

	_, firstErr := updater.Run(context.Background(), update.Options{})
	g.Expect(firstErr).NotTo(HaveOccurred())

	for path := range fileSystem.files {
		if strings.HasPrefix(path, "/repo/agent-instructions/mods/") {
			delete(fileSystem.files, path)
		}
	}

	delete(fileSystem.dirs, "/repo/agent-instructions/mods")
	delete(fileSystem.dirs, "/repo/agent-instructions/mods/engram-memory")

	_, secondErr := updater.Run(context.Background(), update.Options{})
	g.Expect(secondErr).NotTo(HaveOccurred())
	g.Expect(touchedPaths(fileSystem, "engram-memory")).To(BeEmpty())
}

func Test793d31b2_Run_DeploysTheModAsALinkedSkillsDirectoryPlugin(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := modsFixture(false)
	command := &fakeCmd{responses: map[string][]byte{"claude --version": []byte("2.1.296 (Claude Code)\n")}}
	updater := modsUpdater(fileSystem, command)

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	if len(report.Harnesses) != 1 {
		return
	}

	root := modsTestHome + "/.claude/engram/skills/engram-memory"
	g.Expect(fileSystem.written[root+"/.claude-plugin/plugin.json"]).To(Equal([]byte(`{"name":"engram-memory"}`)))
	g.Expect(fileSystem.written[root+"/hooks/register.js"]).To(Equal([]byte("export function register() {}")))
	g.Expect(fileSystem.symlinks[modsTestHome+"/.claude/skills/engram-memory"]).To(Equal(root))
	g.Expect(fileSystem.symlinks[modsTestHome+"/.claude/skills/learn"]).
		To(Equal(modsTestHome + "/.claude/engram/skills/learn"))
	g.Expect(report.Harnesses[0].SkillDirs).To(ContainElement(modDirCount))
	g.Expect(report.Harnesses[0].SkillDirs).To(ContainElement(update.SkillDirCount{Name: "learn", Files: 1}))
	g.Expect(report.Harnesses[0].Err).NotTo(HaveOccurred())
}

func Test793d31b3_Run_ChecksTheVersionOncePerRunEvenInADryRun(t *testing.T) {
	t.Parallel()

	for _, dryRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "real run", true: "dry run"}[dryRun], func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			command := &fakeCmd{responses: map[string][]byte{"claude --version": []byte("2.1.296\n")}}
			_, err := modsUpdater(modsFixture(true), command).Run(context.Background(), update.Options{DryRun: dryRun})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(claudeCalls(command)).To(HaveLen(1))
		})
	}
}

func Test793d31b3_Run_NoModInTheSourceMeansNoVersionCheck(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := modsFixture(false)
	for path := range fileSystem.files {
		if strings.HasPrefix(path, "/repo/agent-instructions/mods/") {
			delete(fileSystem.files, path)
		}
	}

	delete(fileSystem.dirs, "/repo/agent-instructions/mods")
	delete(fileSystem.dirs, "/repo/agent-instructions/mods/engram-memory")

	command := &fakeCmd{}

	report, err := modsUpdater(fileSystem, command).Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(claudeCalls(command)).To(BeEmpty())
	g.Expect(touchedPaths(fileSystem, "engram-memory")).To(BeEmpty())

	if len(report.Harnesses) == 1 {
		g.Expect(report.Harnesses[0].ModNotice).To(BeEmpty())
	}
}

func Test793d31b3_Run_ReportsWhenClaudeCodeCannotLoadMods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		runErr   error
		wantText string
	}{
		{name: "current", version: "2.1.296 (Claude Code)\n"},
		{name: "exactly the minimum", version: "2.1.287 (Claude Code)\n"},
		{name: "a later minor", version: "2.2.0 (Claude Code)\n"},
		{name: "a later major", version: "3.0.1 (Claude Code)\n"},
		{name: "older patch", version: "2.1.286 (Claude Code)\n", wantText: "2.1.286"},
		{name: "older minor", version: "2.0.999 (Claude Code)\n", wantText: "2.0.999"},
		{name: "older major", version: "1.9.9 (Claude Code)\n", wantText: "1.9.9"},
		{name: "not on PATH", runErr: update.ErrCommandNotFound, wantText: "not found"},
		{name: "unreadable version", version: "no version here\n", wantText: "could not read"},
		{name: "the command fails", runErr: errors.New("boom"), wantText: "could not read"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			command := &fakeCmd{responses: map[string][]byte{"claude --version": []byte(tc.version)}}
			if tc.runErr != nil {
				command.errs = map[string]error{"claude --version": tc.runErr}
			}

			report, err := modsUpdater(modsFixture(true), command).Run(context.Background(), update.Options{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(report.Harnesses).To(HaveLen(2))

			if len(report.Harnesses) != 2 {
				return
			}

			g.Expect(report.Harnesses[1].ModNotice).To(BeEmpty(), "Pi takes no mod, so it has nothing to report")

			notice := report.Harnesses[0].ModNotice
			if tc.wantText == "" {
				g.Expect(notice).To(BeEmpty())

				return
			}

			g.Expect(notice).To(ContainSubstring(tc.wantText))
			g.Expect(notice).To(ContainSubstring("2.1.287"))
			g.Expect(report.Harnesses[0].Err).NotTo(HaveOccurred(), "the report does not stop the deploy")
		})
	}
}

func Test793d34b1_Run_DryRunWritesNothingAndStillReportsTheMod(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := modsFixture(false)
	before := stateOf(fileSystem)
	updater := modsUpdater(fileSystem, &fakeCmd{})

	report, err := updater.Run(context.Background(), update.Options{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fileSystem.written).To(BeEmpty())
	g.Expect(fileSystem.symlinks).To(BeEmpty())
	g.Expect(fileSystem.removed).To(BeEmpty())
	g.Expect(stateOf(fileSystem)).To(Equal(before))
	g.Expect(report.Harnesses).To(HaveLen(1))

	if len(report.Harnesses) == 1 {
		g.Expect(report.Harnesses[0].SkillDirs).To(ContainElement(modDirCount))
	}
}

func Test793d34b1_Run_PiGetsNoModAndNoPiDirectoryIsCreated(t *testing.T) {
	t.Parallel()

	t.Run("Pi absent", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		fileSystem := modsFixture(false)
		_, err := modsUpdater(fileSystem, &fakeCmd{}).Run(context.Background(), update.Options{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(touchedPaths(fileSystem, "/.pi")).To(BeEmpty())
	})

	t.Run("Pi present", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		fileSystem := modsFixture(true)
		report, err := modsUpdater(fileSystem, &fakeCmd{}).Run(context.Background(), update.Options{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(report.Harnesses).To(HaveLen(2))
		g.Expect(touchedPaths(fileSystem, "engram-memory")).NotTo(BeEmpty())

		for _, path := range touchedPaths(fileSystem, "engram-memory") {
			g.Expect(path).NotTo(ContainSubstring("/.pi/"))
		}

		g.Expect(report.Harnesses[1].SkillDirs).NotTo(ContainElement(HaveField("Name", "engram-memory")))
	})
}

func Test793d34b1_Run_SecondRunChangesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := modsFixture(true)
	updater := modsUpdater(fileSystem, &fakeCmd{})

	_, firstErr := updater.Run(context.Background(), update.Options{})
	g.Expect(firstErr).NotTo(HaveOccurred())

	afterFirst := stateOf(fileSystem)
	removedAfterFirst := len(fileSystem.removed)

	report, secondErr := updater.Run(context.Background(), update.Options{})
	g.Expect(secondErr).NotTo(HaveOccurred())
	g.Expect(stateOf(fileSystem)).To(Equal(afterFirst))
	g.Expect(fileSystem.removed).To(HaveLen(removedAfterFirst))

	for _, harness := range report.Harnesses {
		g.Expect(harness.Err).NotTo(HaveOccurred())
		g.Expect(harness.EngramSyncDeleted).To(BeEmpty())
		g.Expect(harness.EngramAdopted).To(BeEmpty())
		g.Expect(harness.DanglingLinksRemoved).To(BeEmpty())
	}
}

// unexported constants.
const (
	modsTestHome = "/home/joe"
)

// unexported variables.
var (
	modDirCount = update.SkillDirCount{Name: "engram-memory", Files: 2, Mod: true}
)

// readDirFailFS wraps memFS and fails every ReadDir with an error other than not-exist.
type readDirFailFS struct {
	*memFS

	err error
}

func (r *readDirFailFS) ReadDir(_ string) ([]update.DirEntry, error) {
	return nil, r.err
}

// claudeCalls are the runs of the claude program among the commands the updater ran.
func claudeCalls(command *fakeCmd) [][]string {
	calls := make([][]string, 0)

	for _, call := range command.calls {
		if call[0] == "claude" {
			calls = append(calls, call)
		}
	}

	return calls
}

// modsFixture is a local clone whose source holds one skill and the engram-memory mod, with Claude Code present
// (and Pi when withPi).
func modsFixture(withPi bool) *memFS {
	fileSystem := newMemFS()
	fileSystem.dirs[modsTestHome+"/.claude"] = true

	if withPi {
		fileSystem.dirs[modsTestHome+"/.pi"] = true
	}

	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.files["/repo/agent-instructions/skills/learn/SKILL.md"] = []byte("skill body")
	fileSystem.dirs["/repo/agent-instructions/mods"] = true
	fileSystem.dirs["/repo/agent-instructions/mods/engram-memory"] = true
	modRoot := "/repo/agent-instructions/mods/engram-memory"
	fileSystem.files[modRoot+"/.claude-plugin/plugin.json"] = []byte(`{"name":"engram-memory"}`)
	fileSystem.files[modRoot+"/hooks/register.js"] = []byte("export function register() {}")

	return fileSystem
}

func modsUpdater(fileSystem *memFS, command *fakeCmd) *update.Updater {
	return &update.Updater{
		FS:    fileSystem,
		Cmd:   command,
		Env:   &fakeEnv{home: modsTestHome, cwd: "/repo", sentinel: "1"},
		Spawn: noopSpawner{},
	}
}

// stateOf copies the maps of a memFS so that a later comparison sees what changed.
func stateOf(fileSystem *memFS) map[string]string {
	state := map[string]string{}

	for path, data := range fileSystem.files {
		state["file "+path] = string(data)
	}

	for path := range fileSystem.dirs {
		state["dir "+path] = ""
	}

	for link, target := range fileSystem.symlinks {
		state["link "+link] = target
	}

	return state
}

// touchedPaths lists the paths written, linked or created under home that contain the fragment.
func touchedPaths(fileSystem *memFS, fragment string) []string {
	paths := make([]string, 0)

	collect := func(path string) {
		if strings.HasPrefix(path, modsTestHome+"/") && strings.Contains(path, fragment) {
			paths = append(paths, filepath.ToSlash(path))
		}
	}

	for path := range fileSystem.files {
		collect(path)
	}

	for path := range fileSystem.symlinks {
		collect(path)
	}

	for path := range maps.Keys(fileSystem.dirs) {
		collect(path)
	}

	return paths
}

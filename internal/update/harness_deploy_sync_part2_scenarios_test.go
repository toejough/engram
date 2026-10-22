package update_test

import (
	"errors"
	"io/fs"
	"maps"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

// Test792b29c14a3_Run_ADryRunDeployPlansTheSkills verifies Scenario 792b29c14a3: with a source holding one skill and a
// dry run, the report counts the skill's files and nothing is written.
func Test792b29c14a3_Run_ADryRunDeployPlansTheSkills(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := hds2NewSource()
	fileSystem.dirs[hds2ClaudeDir] = true
	hds2SeedSkill(fileSystem, "learn")

	filesBefore := maps.Clone(fileSystem.files)
	dirsBefore := maps.Clone(fileSystem.dirs)

	report, err := hds2NewUpdater(fileSystem).Run(t.Context(), update.Options{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil || len(report.Harnesses) != 1 {
		g.Expect(report.Harnesses).To(HaveLen(1))

		return
	}

	harness := report.Harnesses[0]
	g.Expect(harness.Err).NotTo(HaveOccurred())
	g.Expect(harness.SkillDirs).To(HaveLen(1))

	if len(harness.SkillDirs) != 1 {
		return
	}

	g.Expect(harness.SkillDirs[0].Name).To(Equal("learn"))
	g.Expect(harness.SkillDirs[0].Files).To(Equal(hds2SkillFileCount))

	g.Expect(fileSystem.written).To(BeEmpty())
	g.Expect(fileSystem.removed).To(BeEmpty())
	g.Expect(fileSystem.symlinks).To(BeEmpty())
	g.Expect(fileSystem.files).To(Equal(filesBefore))
	g.Expect(fileSystem.dirs).To(Equal(dirsBefore))
}

// Test792b29c15a2_Run_TheFailureDoesNotStopTheOtherHarness verifies Scenario 792b29c15a2: when Claude Code fails and
// Pi is healthy, Pi is deployed and the run returns no error.
func Test792b29c15a2_Run_TheFailureDoesNotStopTheOtherHarness(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	base := hds2NewSource()
	base.dirs[hds2ClaudeDir] = true
	base.dirs[hds2PiDir] = true
	hds2SeedSkill(base, "learn")

	// Symlink mode writes real skill content under the engram root: fail Claude Code's root only.
	fileSystem := &failWriteFS{memFS: base, failOn: ".claude/engram/skills"}

	updater := &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: hds2Home, cwd: hds2Repo},
		Spawn: noopSpawner{},
	}

	report, err := updater.Run(t.Context(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil || len(report.Harnesses) != 2 {
		g.Expect(report.Harnesses).To(HaveLen(2))

		return
	}

	claude, piHarness := report.Harnesses[0], report.Harnesses[1]

	g.Expect(claude.Name).To(Equal(update.HarnessClaude))
	g.Expect(claude.Err).To(MatchError(ContainSubstring("write boom")))

	g.Expect(piHarness.Name).To(Equal(update.HarnessPi))
	g.Expect(piHarness.Err).NotTo(HaveOccurred())
	g.Expect(piHarness.SkillDirs).To(HaveLen(1))
	g.Expect(base.written).To(HaveKey(hds2PiRoot + "/skills/learn/SKILL.md"))
	g.Expect(base.symlinks).To(HaveKeyWithValue(hds2Home+"/.pi/agent/skills/learn", hds2PiRoot+"/skills/learn"))
}

// Test792b29c1a7_DetectHarnesses_AnUnreadableProbeIsAnError verifies Scenario 792b29c1a7: when statting ~/.claude
// fails for a reason other than not existing, detection fails with an error naming the probe.
func Test792b29c1a7_DetectHarnesses_AnUnreadableProbeIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs[hds2PiDir] = true

	prober := statOnlyProber{
		fileSystem: fileSystem,
		errs:       map[string]error{hds2ClaudeDir: fs.ErrPermission},
	}

	detected, err := update.DetectHarnesses(hds2Home, prober)
	g.Expect(err).To(MatchError(fs.ErrPermission))
	g.Expect(err).To(MatchError(ContainSubstring(hds2ClaudeDir)))
	g.Expect(detected).To(BeEmpty())
}

// Test792b29c3a8_Run_OneHarnessesImportManagesGuidanceForEveryHarness verifies Scenario 792b29c3a8: when only Claude
// Code's config file imports a guidance file, with no flag and both harnesses present, guidance is deployed for Pi
// as well as for Claude Code.
func Test792b29c3a8_Run_OneHarnessesImportManagesGuidanceForEveryHarness(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := hds2NewSource()
	fileSystem.dirs[hds2ClaudeDir] = true
	fileSystem.dirs[hds2PiDir] = true
	fileSystem.files[hds2Repo+"/agent-instructions/guidance/recall.md"] = []byte("recall guidance")
	fileSystem.files[hds2Repo+"/agent-instructions/guidance/delegate.md"] = []byte("delegate guidance")
	fileSystem.files[hds2ClaudeImports] = []byte("# notes\n\n@~/.claude/engram/delegate.md\n")

	report, err := hds2NewUpdater(fileSystem).Run(t.Context(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil || len(report.Harnesses) != 2 {
		g.Expect(report.Harnesses).To(HaveLen(2))

		return
	}

	g.Expect(report.GuidanceImports).To(HaveKey(update.HarnessClaude))
	g.Expect(report.GuidanceImports).NotTo(HaveKey(update.HarnessPi))

	claude, piHarness := report.Harnesses[0], report.Harnesses[1]

	g.Expect(claude.Err).NotTo(HaveOccurred())
	g.Expect(claude.GuidanceFiles).To(ConsistOf("recall.md", "delegate.md"))
	g.Expect(fileSystem.written).To(HaveKeyWithValue(hds2ClaudeRoot+"/guidance/recall.md", []byte("recall guidance")))
	g.Expect(fileSystem.written).To(HaveKeyWithValue(hds2ClaudeRoot+"/guidance/delegate.md", []byte("delegate guidance")))

	g.Expect(piHarness.Err).NotTo(HaveOccurred())
	g.Expect(piHarness.GuidanceFiles).To(ConsistOf("recall.md", "delegate.md"))
	g.Expect(fileSystem.written).To(HaveKeyWithValue(hds2PiRoot+"/guidance/recall.md", []byte("recall guidance")))
	g.Expect(fileSystem.written).To(HaveKeyWithValue(hds2PiRoot+"/guidance/delegate.md", []byte("delegate guidance")))
	g.Expect(fileSystem.symlinks).To(HaveKeyWithValue(hds2PiGuidanceTarget+"/recall.md", hds2PiRoot+"/guidance/recall.md"))
	g.Expect(fileSystem.symlinks).To(
		HaveKeyWithValue(hds2PiGuidanceTarget+"/delegate.md", hds2PiRoot+"/guidance/delegate.md"))
}

// Test792b29c4a5_Run_AFenceHidesImports verifies Scenario 792b29c4a5: an import line inside a closed fence, or after
// an unclosed fence, is not recorded.
func Test792b29c4a5_Run_AFenceHidesImports(t *testing.T) {
	t.Parallel()

	table := map[string]struct {
		content string
		want    []string
	}{
		"closed fence hides its import and the import after it counts": {
			content: "```\n@~/.claude/engram/recall.md\n```\n@~/.claude/engram/delegate.md\n",
			want:    []string{"delegate.md"},
		},
		"fence with a language tag hides its import": {
			content: "```markdown\n@~/.claude/engram/recall.md\n```\n",
		},
		"unclosed fence hides the rest of the file": {
			content: "@~/.claude/engram/delegate.md\n```\n@~/.claude/engram/recall.md\n@~/.claude/engram/other.md\n",
			want:    []string{"delegate.md"},
		},
		"an import only inside an unclosed fence is not recorded": {
			content: "# notes\n```\n@~/.claude/engram/recall.md\n",
		},
	}

	for name, testCase := range table {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fileSystem := hds2NewSource()
			fileSystem.dirs[hds2ClaudeDir] = true
			fileSystem.files[hds2ClaudeImports] = []byte(testCase.content)

			report, err := hds2NewUpdater(fileSystem).Run(t.Context(), update.Options{})
			g.Expect(err).NotTo(HaveOccurred())

			recorded := report.GuidanceImports[update.HarnessClaude]
			g.Expect(recorded).To(HaveLen(len(testCase.want)))

			for _, base := range testCase.want {
				g.Expect(recorded).To(HaveKeyWithValue(base, true))
			}

			g.Expect(report.GuidanceImported).To(Equal(len(testCase.want) > 0))
		})
	}
}

// Test792b29c4a6_Run_StaleAndForeignPrefixesDoNotCount verifies Scenario 792b29c4a6: given the expanded-home form of a
// Pi import, which counts, and ~/.pi/agent/engram/recall.md, which does not, only the first is recorded.
func Test792b29c4a6_Run_StaleAndForeignPrefixesDoNotCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := hds2NewSource()
	fileSystem.dirs[hds2ClaudeDir] = true
	fileSystem.dirs[hds2PiDir] = true
	fileSystem.files[hds2PiImports] = []byte(
		"@" + hds2PiGuidanceTarget + "/delegate.md\n@~/.pi/agent/engram/recall.md\n",
	)

	report, err := hds2NewUpdater(fileSystem).Run(t.Context(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(report.GuidanceImports[update.HarnessPi]).To(Equal(map[string]bool{"delegate.md": true}))
	g.Expect(report.GuidanceImports).NotTo(HaveKey(update.HarnessClaude))
}

// Test792b29c5a6_Run_ARootThatIsAFileIsAnError verifies Scenario 792b29c5a6: when the root path is a regular file,
// that harness fails with ErrEngramRootNotDir and the file is untouched.
func Test792b29c5a6_Run_ARootThatIsAFileIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := hds2NewSource()
	fileSystem.dirs[hds2ClaudeDir] = true
	fileSystem.files[hds2ClaudeRoot] = []byte("not a directory")
	hds2SeedSkill(fileSystem, "learn")

	report, err := hds2NewUpdater(fileSystem).Run(t.Context(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil || len(report.Harnesses) != 1 {
		g.Expect(report.Harnesses).To(HaveLen(1))

		return
	}

	g.Expect(report.Harnesses[0].Err).To(MatchError(update.ErrEngramRootNotDir))
	g.Expect(errors.Is(report.Harnesses[0].Err, update.ErrEngramRootNotDir)).To(BeTrue())

	g.Expect(fileSystem.files[hds2ClaudeRoot]).To(Equal([]byte("not a directory")))
	g.Expect(fileSystem.written).NotTo(HaveKey(hds2ClaudeRoot))
	g.Expect(fileSystem.removed).NotTo(ContainElement(hds2ClaudeRoot))
	g.Expect(fileSystem.written).To(BeEmpty())
}

// unexported constants.
const (
	hds2ClaudeDir        = hds2Home + "/.claude"
	hds2ClaudeImports    = hds2Home + "/.claude/CLAUDE.md"
	hds2ClaudeRoot       = hds2Home + "/.claude/engram"
	hds2Home             = "/home/user"
	hds2PiDir            = hds2Home + "/.pi"
	hds2PiGuidanceTarget = hds2Home + "/.pi/agent/guidance"
	hds2PiImports        = hds2Home + "/.pi/agent/AGENTS.md"
	hds2PiRoot           = hds2Home + "/.pi/agent/engram"
	hds2Repo             = "/repo"
	hds2SkillFileCount   = 2
)

// hds2NewSource returns an in-memory filesystem holding a local checkout of the module at hds2Repo, with an empty
// skills source directory.
func hds2NewSource() *memFS {
	fileSystem := newMemFS()
	fileSystem.files[hds2Repo+"/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs[hds2Repo+"/agent-instructions/skills"] = true

	return fileSystem
}

// hds2NewUpdater returns an Updater over fileSystem with the home at hds2Home and the checkout as the working
// directory.
func hds2NewUpdater(fileSystem update.Filesystem) *update.Updater {
	return &update.Updater{
		FS:    fileSystem,
		Cmd:   &fakeCmd{},
		Env:   &fakeEnv{home: hds2Home, cwd: hds2Repo},
		Spawn: noopSpawner{},
	}
}

// hds2SeedSkill adds a skill of hds2SkillFileCount files to the skills source.
func hds2SeedSkill(fileSystem *memFS, name string) {
	skillDir := hds2Repo + "/agent-instructions/skills/" + name

	fileSystem.dirs[skillDir] = true
	fileSystem.files[skillDir+"/SKILL.md"] = []byte(name + " skill")
	fileSystem.files[skillDir+"/reference.md"] = []byte(name + " reference")
}

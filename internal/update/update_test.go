package update_test

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/update"
)

func TestDetectHarnesses_Both_StableOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/home/joe/.config/opencode"] = true

	detected, err := update.ExportDetectHarnesses("/home/joe", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(detected).To(HaveLen(2))
	g.Expect(detected[0].Name).To(Equal(update.HarnessClaude))
	g.Expect(detected[1].Name).To(Equal(update.HarnessOpencode))
}

func TestDetectHarnesses_ClaudeOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true

	detected, err := update.ExportDetectHarnesses("/home/joe", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(detected).To(HaveLen(1))
	g.Expect(detected[0].Name).To(Equal(update.HarnessClaude))
}

func TestDetectHarnesses_None(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()

	detected, err := update.ExportDetectHarnesses("/home/joe", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(detected).To(BeEmpty())
}

func TestGuidanceImportAttribution_PerHarness(t *testing.T) {
	t.Parallel()

	const home = "/home/joe"

	table := []struct {
		name       string
		claudeMD   string
		agentsMD   string
		wantClaude []string
		wantPi     []string
	}{
		{
			name:       "claude-import-attributes-to-claude-only",
			claudeMD:   "@~/.claude/engram/recall.md\n",
			agentsMD:   "# no imports here\n",
			wantClaude: []string{"recall.md"},
		},
		{
			name:     "pi-guidance-import-attributes-to-pi-only",
			claudeMD: "# no imports here\n",
			agentsMD: "@~/.pi/agent/guidance/recall.md\n",
			wantPi:   []string{"recall.md"},
		},
		{
			name:     "pi-expanded-form-detected",
			agentsMD: "@" + home + "/.pi/agent/guidance/delegate.md\n",
			wantPi:   []string{"delegate.md"},
		},
		{
			name:     "pi-stale-engram-prefix-not-counted",
			agentsMD: "@~/.pi/agent/engram/recall.md\n",
		},
		{
			name:     "pi-import-in-claude-md-not-cross-attributed",
			claudeMD: "@~/.pi/agent/guidance/recall.md\n",
		},
		{
			name:     "claude-import-in-agents-md-not-cross-attributed",
			agentsMD: "@~/.claude/engram/recall.md\n",
		},
		{
			name:     "fence-ignored-in-agents-md",
			agentsMD: "```\n@~/.pi/agent/guidance/recall.md\n```\n",
		},
		{
			name:     "unclosed-fence-ignores-rest-of-agents-md",
			agentsMD: "```\n@~/.pi/agent/guidance/recall.md\n",
		},
		{
			name:       "both-files-attribute-independently",
			claudeMD:   "@~/.claude/engram/recall.md\n",
			agentsMD:   "@~/.pi/agent/guidance/delegate.md\n",
			wantClaude: []string{"recall.md"},
			wantPi:     []string{"delegate.md"},
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fileSystem := newMemFS()
			fileSystem.dirs[home+"/.claude"] = true
			fileSystem.dirs[home+"/.pi"] = true
			fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
			fileSystem.dirs["/repo/agent-instructions/skills"] = true

			if tc.claudeMD != "" {
				fileSystem.files[home+"/.claude/CLAUDE.md"] = []byte(tc.claudeMD)
			}

			if tc.agentsMD != "" {
				fileSystem.files[home+"/.pi/agent/AGENTS.md"] = []byte(tc.agentsMD)
			}

			updater := &update.Updater{
				FS:  fileSystem,
				Cmd: &fakeCmd{},
				Env: &fakeEnv{home: home, cwd: "/repo"},
			}

			report, err := updater.Run(context.Background(), update.Options{})
			g.Expect(err).NotTo(HaveOccurred())

			gotClaude := report.GuidanceImports[update.HarnessClaude]
			g.Expect(gotClaude).To(HaveLen(len(tc.wantClaude)))

			for _, name := range tc.wantClaude {
				g.Expect(gotClaude).To(HaveKeyWithValue(name, true))
			}

			gotPi := report.GuidanceImports[update.HarnessPi]
			g.Expect(gotPi).To(HaveLen(len(tc.wantPi)))

			for _, name := range tc.wantPi {
				g.Expect(gotPi).To(HaveKeyWithValue(name, true))
			}

			g.Expect(report.GuidanceImported).To(Equal(len(tc.wantClaude)+len(tc.wantPi) > 0))
		})
	}
}

func TestGuidanceImportAttribution_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		const home = "/home/joe"

		claudeDoc := drawGuidanceDoc(rt, "claude", home)
		piDoc := drawGuidanceDoc(rt, "pi", home)

		fileSystem := newMemFS()
		fileSystem.dirs[home+"/.claude"] = true
		fileSystem.dirs[home+"/.pi"] = true
		fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
		fileSystem.dirs["/repo/agent-instructions/skills"] = true
		fileSystem.files[home+"/.claude/CLAUDE.md"] = []byte(claudeDoc.text)
		fileSystem.files[home+"/.pi/agent/AGENTS.md"] = []byte(piDoc.text)

		updater := &update.Updater{
			FS:  fileSystem,
			Cmd: &fakeCmd{},
			Env: &fakeEnv{home: home, cwd: "/repo"},
		}

		report, err := updater.Run(context.Background(), update.Options{})
		if err != nil {
			rt.Fatalf("run: %v", err)
		}

		// (a) imports inside fences (closed or unclosed) are never counted;
		// (b) every counted import attributes exactly to the harness whose
		// derived prefix matched inside that harness's own imports file.
		checkImportSet(rt, "claude", report.GuidanceImports[update.HarnessClaude], claudeDoc.wantOwn)
		checkImportSet(rt, "pi", report.GuidanceImports[update.HarnessPi], piDoc.wantOwn)
	})
}

func TestGuidanceImportDetection(t *testing.T) {
	t.Parallel()

	table := []struct {
		name     string
		content  string
		wantBool bool
	}{
		{
			name:     "tilde-form-detected",
			content:  "@~/.claude/engram/recall.md\n",
			wantBool: true,
		},
		{
			name:     "expanded-form-detected",
			content:  "@/home/joe/.claude/engram/recall.md\n",
			wantBool: true,
		},
		{
			name:     "absent-returns-false",
			content:  "# no import here\n",
			wantBool: false,
		},
		{
			name:     "inside-code-fence-ignored",
			content:  "```\n@~/.claude/engram/recall.md\n```\n",
			wantBool: false,
		},
		{
			name:     "nested-path-rejected",
			content:  "@~/.claude/engram/sub/recall.md\n",
			wantBool: false,
		},
		{
			name:     "non-md-suffix-rejected",
			content:  "@~/.claude/engram/recall.txt\n",
			wantBool: false,
		},
		{
			name:     "bare-prefix-no-basename-rejected",
			content:  "@~/.claude/engram/\n",
			wantBool: false,
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fileSystem := newMemFS()
			fileSystem.files["/home/joe/.claude/CLAUDE.md"] = []byte(tc.content)

			const home = "/home/joe"

			updater := &update.Updater{
				FS:  fileSystem,
				Cmd: &fakeCmd{},
				Env: &fakeEnv{home: home, cwd: "/repo"},
			}

			fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
			fileSystem.dirs["/repo/agent-instructions/skills"] = true
			fileSystem.dirs[home+"/.claude"] = true

			report, err := updater.Run(context.Background(), update.Options{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(report.GuidanceImported).To(Equal(tc.wantBool))
		})
	}
}

func TestGuidanceImportDetection_MissingClaudeMD(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	// No CLAUDE.md → GuidanceImported should be false, no error

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.GuidanceImported).To(BeFalse())
}

func TestGuidanceImportDetection_PerFileSet(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.files[home+"/.claude/CLAUDE.md"] = []byte(
		"# joe\n\n@~/.claude/engram/recall.md\n@~/.claude/engram/delegate.md\n",
	)
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs[home+"/.claude"] = true

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report.GuidanceImported).To(BeTrue())
	g.Expect(report.GuidanceImports[update.HarnessClaude]).To(HaveKeyWithValue("recall.md", true))
	g.Expect(report.GuidanceImports[update.HarnessClaude]).To(HaveKeyWithValue("delegate.md", true))
}

func TestGuidanceImportPrefixDerivation_AllHarnesses(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.dirs[home+"/.config/opencode"] = true
	fileSystem.dirs[home+"/.pi"] = true

	detected, err := update.ExportDetectHarnesses(home, fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(detected).To(HaveLen(3))

	want := map[update.Harness][2]string{
		update.HarnessClaude: {"@~/.claude/engram/", "@" + home + "/.claude/engram/"},
		update.HarnessPi:     {"@~/.pi/agent/guidance/", "@" + home + "/.pi/agent/guidance/"},
	}

	for _, spec := range detected {
		if spec.Name == update.HarnessOpencode {
			// OpenCode has no imports file — detection (and thus prefix
			// derivation) is skipped for it entirely.
			g.Expect(spec.ImportsFileRel).To(BeEmpty())

			continue
		}

		tilde, expanded := update.ExportGuidanceImportPrefixes(spec, home)
		g.Expect(tilde).To(Equal(want[spec.Name][0]), string(spec.Name))
		g.Expect(expanded).To(Equal(want[spec.Name][1]), string(spec.Name))
	}
}

func TestHarnessSpecs_WellKnownPaths(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.dirs["/home/joe/.claude"] = true
	fileSystem.dirs["/home/joe/.config/opencode"] = true
	fileSystem.dirs["/home/joe/.pi"] = true

	detected, err := update.ExportDetectHarnesses("/home/joe", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(detected).To(HaveLen(3))

	claude, opencode, piHarness := detected[0], detected[1], detected[2]

	g.Expect(claude.Name).To(Equal(update.HarnessClaude))
	g.Expect(claude.ImportsFileRel).To(Equal(filepath.Join(".claude", "CLAUDE.md")))

	g.Expect(opencode.Name).To(Equal(update.HarnessOpencode))
	g.Expect(opencode.ImportsFileRel).To(BeEmpty())

	g.Expect(piHarness.Name).To(Equal(update.HarnessPi))
	g.Expect(piHarness.ProbeRel).To(Equal(".pi"))
	g.Expect(piHarness.SkillsTargetRel).To(Equal(filepath.Join(".pi", "agent", "skills")))
	g.Expect(piHarness.GuidanceTargetRel).To(Equal(filepath.Join(".pi", "agent", "guidance")))
	g.Expect(piHarness.ImportsFileRel).To(Equal(filepath.Join(".pi", "agent", "AGENTS.md")))
}

func TestPlanGuidanceCopies_FilesUnderHome(t *testing.T) {
	t.Parallel()

	harnesses := []update.HarnessSpec{
		{
			Name:              update.HarnessClaude,
			ProbeRel:          ".claude",
			SkillsTargetRel:   ".claude/skills",
			GuidanceTargetRel: ".claude/engram",
		},
		{
			Name:              update.HarnessOpencode,
			ProbeRel:          ".config/opencode",
			SkillsTargetRel:   ".config/opencode/skills",
			GuidanceTargetRel: "",
		},
	}

	table := []struct {
		name      string
		wantCount int
		wantDst   string
	}{
		{
			name:      "claude-code-gets-op-opencode-skipped",
			wantCount: 1,
			wantDst:   "/home/joe/.claude/engram/recall.md",
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fileSystem := newMemFS()
			fileSystem.files["/src/agent-instructions/guidance/recall.md"] = []byte("guidance")
			fileSystem.dirs["/src/agent-instructions/guidance"] = true

			ops, err := update.ExportPlanGuidanceCopies("/src/agent-instructions/guidance", "/home/joe", harnesses, fileSystem)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(ops).To(HaveLen(tc.wantCount))
			g.Expect(ops[0].Dst).To(Equal(tc.wantDst))
			g.Expect(ops[0].GuidanceFile).To(Equal("recall.md"))
			g.Expect(ops[0].Harness).To(Equal(update.HarnessClaude))
		})
	}
}

func TestPlanGuidanceCopies_MissingSrc(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	harnesses := []update.HarnessSpec{
		{
			Name:              update.HarnessClaude,
			ProbeRel:          ".claude",
			SkillsTargetRel:   ".claude/skills",
			GuidanceTargetRel: ".claude/engram",
		},
	}

	ops, err := update.ExportPlanGuidanceCopies("/nonexistent", "/home/joe", harnesses, fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ops).To(BeNil())
}

func TestPlanSkillCopies_FilesUnderHome_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		homeSeg := rapid.StringMatching(`[a-z]{2,5}`).Draw(rt, "home")
		home := "/h/" + homeSeg

		fileSystem := newMemFS()
		fileSystem.dirs["/src/agent-instructions/skills"] = true
		fileSystem.dirs["/src/agent-instructions/skills/learn"] = true
		fileSystem.files["/src/agent-instructions/skills/learn/SKILL.md"] = []byte("x")
		fileSystem.dirs["/src/agent-instructions/skills/recall"] = true
		fileSystem.files["/src/agent-instructions/skills/recall/SKILL.md"] = []byte("x")

		// Detect Claude only.
		fileSystem.dirs[home+"/.claude"] = true

		harnesses, err := update.ExportDetectHarnesses(home, fileSystem)
		if err != nil {
			rt.Fatalf("detect: %v", err)
		}

		ops, err := update.ExportPlanSkillCopies("/src/agent-instructions/skills", home, harnesses, fileSystem)
		if err != nil {
			rt.Fatalf("plan: %v", err)
		}

		for _, op := range ops {
			if !strings.HasPrefix(op.Dst, home+string(filepath.Separator)) {
				rt.Fatalf("dst %q not under home %q", op.Dst, home)
			}
		}
	})
}

func TestPlanSkillCopies_MissingSrc(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	harnesses := []update.HarnessSpec{
		{Name: update.HarnessClaude, ProbeRel: ".claude", SkillsTargetRel: ".claude/skills"},
	}

	_, err := update.ExportPlanSkillCopies("/nonexistent", "/home/joe", harnesses, fileSystem)
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, update.ErrSkillsSrcMissing)).To(BeTrue())
}

func TestRun_PlainUpdate_DelegateOnlyImport_RefreshesAll(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("recall guidance")
	fileSystem.files["/repo/agent-instructions/guidance/delegate.md"] = []byte("delegate guidance")
	// Only delegate.md is imported — recall.md is not.
	fileSystem.files[home+"/.claude/CLAUDE.md"] = []byte("# joe\n\n@~/.claude/engram/delegate.md\n")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(report.GuidanceImported).To(BeTrue())
	g.Expect(report.Harnesses[0].GuidanceFiles).To(ConsistOf("recall.md", "delegate.md"))
	g.Expect(fileSystem.written[home+"/.claude/engram/delegate.md"]).NotTo(BeNil())
	g.Expect(fileSystem.written[home+"/.claude/engram/recall.md"]).NotTo(BeNil())
}

func TestRun_PlainUpdate_WhenImported_RefreshesGuidance(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("fresh guidance content")
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true
	// The user already imports the guidance (opted in) — no --with-guidance flag.
	fileSystem.files[home+"/.claude/CLAUDE.md"] = []byte("# joe\n\n@~/.claude/engram/recall.md\n")

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	// Plain update MUST refresh the guidance because it is already imported.
	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.GuidanceImported).To(BeTrue())
	g.Expect(report.Harnesses[0].GuidanceFiles).To(ConsistOf("recall.md"))

	written, ok := fileSystem.written[home+"/.claude/engram/recall.md"]
	g.Expect(ok).To(BeTrue())
	g.Expect(written).To(Equal([]byte("fresh guidance content")))
}

func TestRun_WithGuidance_BothHarnesses_OnlyClaudeGetsGuidance(t *testing.T) {
	t.Parallel()

	// Having both harnesses ensures applyGuidanceOps hits the
	// "copyOp.Harness != name" continue branch for OpenCode.
	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.dirs[home+"/.config/opencode"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("recall guidance")
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{WithGuidance: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(2))

	// Claude Code gets the guidance file.
	claudeReport := report.Harnesses[0]
	g.Expect(claudeReport.Name).To(Equal(update.HarnessClaude))
	g.Expect(claudeReport.GuidanceFiles).To(ConsistOf("recall.md"))

	// OpenCode guidance target is empty → no guidance files.
	opencodeReport := report.Harnesses[1]
	g.Expect(opencodeReport.Name).To(Equal(update.HarnessOpencode))
	g.Expect(opencodeReport.GuidanceFiles).To(BeEmpty())
}

func TestRun_WithGuidance_DeploysToClaudeEngram(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("recall guidance content")
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{WithGuidance: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))

	claudeReport := report.Harnesses[0]
	g.Expect(claudeReport.GuidanceFiles).To(ConsistOf("recall.md"))

	written, ok := fileSystem.written[home+"/.claude/engram/recall.md"]
	g.Expect(ok).To(BeTrue())
	g.Expect(written).To(Equal([]byte("recall guidance content")))
}

func TestRun_WithGuidance_GuidanceCopyError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	base := newMemFS()
	base.dirs[home+"/.claude"] = true
	base.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	base.dirs["/repo/agent-instructions/skills"] = true
	base.files["/repo/agent-instructions/guidance/recall.md"] = []byte("recall guidance")
	base.dirs["/repo/agent-instructions/guidance"] = true

	removeErr := errors.New("disk full")
	fileSystem := &errRemoveFS{
		memFS:     base,
		errPath:   home + "/.claude/engram/recall.md",
		removeErr: removeErr,
	}

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{WithGuidance: true})
	g.Expect(err).NotTo(HaveOccurred()) // error is per-harness, not top-level
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].Err).To(MatchError(ContainSubstring("disk full")))
	g.Expect(report.Harnesses[0].GuidanceFiles).To(BeEmpty())
}

func TestRun_WithoutGuidance_SkipsGuidance(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	fileSystem := newMemFS()
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.dirs["/repo/agent-instructions/skills"] = true
	fileSystem.files["/repo/agent-instructions/guidance/recall.md"] = []byte("recall guidance content")
	fileSystem.dirs["/repo/agent-instructions/guidance"] = true

	updater := &update.Updater{
		FS:  fileSystem,
		Cmd: &fakeCmd{},
		Env: &fakeEnv{home: home, cwd: "/repo"},
	}

	report, err := updater.Run(context.Background(), update.Options{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(report.Harnesses).To(HaveLen(1))
	g.Expect(report.Harnesses[0].GuidanceFiles).To(BeEmpty())
	g.Expect(fileSystem.written[home+"/.claude/engram/recall.md"]).To(BeNil())
}

func TestWalkUpForModule_FoundAtStart(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n\ngo 1.25.6\n")

	root, found, err := update.ExportWalkUpForModule("/repo", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(root).To(Equal("/repo"))
}

func TestWalkUpForModule_FoundInAncestor(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/repo/go.mod"] = []byte("module github.com/toejough/engram\n")

	root, found, err := update.ExportWalkUpForModule("/repo/internal/update", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(root).To(Equal("/repo"))
}

func TestWalkUpForModule_NotFound(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	// no go.mod anywhere

	root, found, err := update.ExportWalkUpForModule("/some/where", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
	g.Expect(root).To(BeEmpty())
}

func TestWalkUpForModule_Property_TerminatesForAnyPath(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Generate a posix-style absolute path with 0-6 segments.
		depth := rapid.IntRange(0, 6).Draw(rt, "depth")

		segs := rapid.SliceOfN(rapid.StringMatching(`[a-z]{1,4}`), depth, depth).Draw(rt, "segs")

		path := "/"
		for _, seg := range segs {
			path = filepath.Join(path, seg)
		}

		fileSystem := newMemFS() // no go.mod anywhere

		_, found, err := update.ExportWalkUpForModule(path, fileSystem)
		if err != nil {
			rt.Fatalf("unexpected error: %v", err)
		}

		if found {
			rt.Fatalf("expected not-found for empty fs, got found=true")
		}
	})
}

func TestWalkUpForModule_ReadFileErrorPropagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	wantErr := errors.New("disk failed")
	fileSystem := &errFS{readErr: wantErr}

	_, _, err := update.ExportWalkUpForModule("/anywhere", fileSystem)
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, wantErr)).To(BeTrue())
}

func TestWalkUpForModule_WrongModuleStopsWalk(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := newMemFS()
	fileSystem.files["/somewhere/go.mod"] = []byte("module example.com/other\n")

	// Should NOT keep walking up past a found go.mod with the wrong module.
	root, found, err := update.ExportWalkUpForModule("/somewhere/sub", fileSystem)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
	g.Expect(root).To(BeEmpty())
}

// errFS returns the same error for every ReadFile call.
type errFS struct {
	readErr error
}

func (*errFS) MkdirAll(_ string, _ fs.FileMode) error {
	return nil
}

func (*errFS) ReadDir(_ string) ([]update.DirEntry, error) {
	return nil, fs.ErrNotExist
}

func (e *errFS) ReadFile(_ string) ([]byte, error) {
	return nil, e.readErr
}

func (*errFS) RemoveAll(_ string) error {
	return nil
}

func (*errFS) Stat(_ string) (update.FileInfo, error) {
	return nil, fs.ErrNotExist
}

func (*errFS) WriteFile(_ string, _ []byte, _ fs.FileMode) error {
	return nil
}

// errRemoveFS wraps memFS and returns an error for RemoveAll on a specific path.
type errRemoveFS struct {
	*memFS

	errPath   string
	removeErr error
}

func (e *errRemoveFS) RemoveAll(path string) error {
	if path == e.errPath {
		return e.removeErr
	}

	return e.memFS.RemoveAll(path)
}

// guidanceDocSpec is a generated harness imports file plus the set of
// guidance basenames a correct scanner must count for that harness (its own
// prefix, outside any code fence). Everything else — foreign-harness
// prefixes, the stale pre-guidance/ pi prefix, noise, fenced imports — must
// never be counted.
type guidanceDocSpec struct {
	text    string
	wantOwn map[string]bool
}

type memEntry struct {
	name  string
	isDir bool
}

func (m *memEntry) IsDir() bool { return m.isDir }

func (m *memEntry) Name() string { return m.name }

// --- in-memory test doubles --------------------------------------------

type memFS struct {
	files   map[string][]byte
	dirs    map[string]bool
	written map[string][]byte
	removed []string
}

func (m *memFS) MkdirAll(path string, _ fs.FileMode) error {
	m.dirs[path] = true
	return nil
}

func (m *memFS) ReadDir(path string) ([]update.DirEntry, error) {
	if !m.dirExists(path) {
		return nil, fs.ErrNotExist
	}

	prefix := dirPrefix(path)
	seen := map[string]bool{}
	out := make([]update.DirEntry, 0)

	for filePath := range m.files {
		addChild(filePath, prefix, false, seen, &out)
	}

	for dirPath := range m.dirs {
		addChild(dirPath, prefix, true, seen, &out)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })

	return out, nil
}

func (m *memFS) ReadFile(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return data, nil
}

func (m *memFS) RemoveAll(path string) error {
	m.removed = append(m.removed, path)
	delete(m.dirs, path)
	delete(m.files, path)

	prefix := dirPrefix(path)

	for p := range m.files {
		if strings.HasPrefix(p, prefix) {
			delete(m.files, p)
		}
	}

	for p := range m.dirs {
		if strings.HasPrefix(p, prefix) {
			delete(m.dirs, p)
		}
	}

	return nil
}

func (m *memFS) Stat(path string) (update.FileInfo, error) {
	if m.dirs[path] {
		return &memInfo{isDir: true}, nil
	}

	if _, ok := m.files[path]; ok {
		return &memInfo{isDir: false}, nil
	}

	return nil, fs.ErrNotExist
}

func (m *memFS) WriteFile(path string, data []byte, _ fs.FileMode) error {
	m.written[path] = append([]byte(nil), data...)
	m.files[path] = m.written[path]

	return nil
}

func (m *memFS) dirExists(path string) bool {
	if m.dirs[path] {
		return true
	}

	prefix := dirPrefix(path)

	for filePath := range m.files {
		if strings.HasPrefix(filePath, prefix) {
			return true
		}
	}

	for dirPath := range m.dirs {
		if strings.HasPrefix(dirPath, prefix) {
			return true
		}
	}

	return false
}

type memInfo struct{ isDir bool }

func (m *memInfo) IsDir() bool { return m.isDir }

func addChild(
	fullPath, prefix string,
	forceIsDir bool,
	seen map[string]bool,
	out *[]update.DirEntry,
) {
	if !strings.HasPrefix(fullPath, prefix) {
		return
	}

	rest := strings.TrimPrefix(fullPath, prefix)
	if rest == "" {
		return
	}

	name, _, hasSlash := strings.Cut(rest, "/")
	if seen[name] {
		return
	}

	seen[name] = true
	*out = append(*out, &memEntry{name: name, isDir: forceIsDir || hasSlash})
}

// checkImportSet fails the rapid run unless got holds exactly the want set.
func checkImportSet(rt *rapid.T, label string, got, want map[string]bool) {
	if len(got) != len(want) {
		rt.Fatalf("%s: got %v, want %v", label, got, want)
	}

	for name := range want {
		if !got[name] {
			rt.Fatalf("%s: missing %q — got %v, want %v", label, name, got, want)
		}
	}
}

func dirPrefix(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}

	return path + "/"
}

// drawGuidanceDoc generates an imports file for one harness out of blocks:
// countable own-prefix imports (tilde or expanded form), foreign-harness
// imports, stale-prefix imports, noise lines, closed fenced blocks holding
// own-prefix imports, and optionally a trailing unclosed fence holding one.
// wantOwn is built from the generation structure, not by re-scanning.
func drawGuidanceDoc(rt *rapid.T, label, home string) guidanceDocSpec {
	var ownTilde, ownExpanded, foreignTilde, staleTilde string

	if label == "claude" {
		ownTilde = "@~/.claude/engram/"
		ownExpanded = "@" + home + "/.claude/engram/"
		foreignTilde = "@~/.pi/agent/guidance/"
		staleTilde = "@~/.claude/old-guidance/"
	} else {
		ownTilde = "@~/.pi/agent/guidance/"
		ownExpanded = "@" + home + "/.pi/agent/guidance/"
		foreignTilde = "@~/.claude/engram/"
		staleTilde = "@~/.pi/agent/engram/"
	}

	kinds := []string{"own", "own-expanded", "foreign", "stale", "noise", "fenced"}
	want := map[string]bool{}
	lines := make([]string, 0)

	blockCount := rapid.IntRange(0, 6).Draw(rt, label+"-blocks")
	for i := range blockCount {
		kind := rapid.SampledFrom(kinds).Draw(rt, label+"-kind"+strconv.Itoa(i))
		name := rapid.StringMatching(`[a-z]{1,6}\.md`).Draw(rt, label+"-name"+strconv.Itoa(i))

		switch kind {
		case "own":
			lines = append(lines, ownTilde+name)
			want[name] = true
		case "own-expanded":
			lines = append(lines, ownExpanded+name)
			want[name] = true
		case "foreign":
			lines = append(lines, foreignTilde+name)
		case "stale":
			lines = append(lines, staleTilde+name)
		case "noise":
			lines = append(lines, "# note about "+strings.TrimSuffix(name, ".md"))
		case "fenced":
			lines = append(lines, "```", ownTilde+name, "```")
		}
	}

	if rapid.Bool().Draw(rt, label+"-unclosed-fence") {
		name := rapid.StringMatching(`[a-z]{1,6}\.md`).Draw(rt, label+"-unclosed-name")
		lines = append(lines, "```", ownTilde+name)
	}

	return guidanceDocSpec{text: strings.Join(lines, "\n") + "\n", wantOwn: want}
}

func newMemFS() *memFS {
	return &memFS{
		files:   map[string][]byte{},
		dirs:    map[string]bool{},
		written: map[string][]byte{},
	}
}

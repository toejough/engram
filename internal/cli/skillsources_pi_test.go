package cli_test

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// ---------------------------------------------------------------------------
// ResolvePiProjectTrust (task 1.6)
// ---------------------------------------------------------------------------

func TestResolvePiProjectTrust_DefaultAlwaysTrusts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Spec scenario "Trust fallback to defaultProjectTrust always".
	fsys := newFakeSkillFS().
		dir(projectTop).
		file(piAgentDir+"/trust.json", `{"/elsewhere": true}`).
		file(piAgentDir+"/settings.json", `{"defaultProjectTrust": "always"}`)

	trusted, warnings := cli.ResolvePiProjectTrust(fsys, piAgentDir, projectTop)

	g.Expect(trusted).To(BeTrue())
	g.Expect(warnings).To(BeEmpty())
}

func TestResolvePiProjectTrust_FallbackOtherThanAlwaysIsUntrusted(t *testing.T) {
	t.Parallel()

	for _, settings := range []string{
		"", `{}`, `{"defaultProjectTrust": "ask"}`, `{"defaultProjectTrust": "never"}`,
		`{"defaultProjectTrust": "ALWAYS"}`, `{"defaultProjectTrust": true}`,
	} {
		t.Run(settings, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := newFakeSkillFS().dir(projectTop)
			if settings != "" {
				fsys.file(piAgentDir+"/settings.json", settings)
			}

			trusted, warnings := cli.ResolvePiProjectTrust(fsys, piAgentDir, projectTop)

			g.Expect(trusted).To(BeFalse())
			g.Expect(warnings).To(BeEmpty())
		})
	}
}

// TestResolvePiProjectTrust_FromTrustFile drives the FS interactively: the
// cwd is canonicalized, then trust.json is resolved and read; a decision is
// found, so settings.json is never read.
func TestResolvePiProjectTrust_FromTrustFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mock, imp := MockSkillSourceFS(t)
	done := make(chan bool)

	go func() {
		trusted, _ := cli.ResolvePiProjectTrust(mock, "/a", "/p")
		done <- trusted
	}()

	dirInfo := func(name string) fs.FileInfo {
		return fakeSkillInfo{name: name, node: &fakeSkillNode{kind: fakeSkillDir}}
	}
	fileInfo := fakeSkillInfo{name: "trust.json", node: &fakeSkillNode{kind: fakeSkillFile}}

	imp.Lstat.ArgsEqual("/p").Return(dirInfo("p"), nil)
	imp.Lstat.ArgsEqual("/a").Return(dirInfo("a"), nil)
	imp.Lstat.ArgsEqual("/a/trust.json").Return(fileInfo, nil)
	imp.ReadFile.ArgsEqual("/a/trust.json").Return([]byte(`{"/p": true}`), nil)

	g.Expect(<-done).To(BeTrue())
}

func TestResolvePiProjectTrust_InvalidFilesAreUntrustedWithWarning(t *testing.T) {
	t.Parallel()

	cases := map[string]func(*fakeSkillFS){
		"bad json":  func(f *fakeSkillFS) { f.file(piAgentDir+"/trust.json", `{`) },
		"array":     func(f *fakeSkillFS) { f.file(piAgentDir+"/trust.json", `[]`) },
		"bad value": func(f *fakeSkillFS) { f.file(piAgentDir+"/trust.json", `{"`+projectTop+`": "yes"}`) },
		"unreadable trust": func(f *fakeSkillFS) {
			f.file(piAgentDir+"/trust.json", `{}`).failRead(piAgentDir + "/trust.json")
		},
		"bad settings": func(f *fakeSkillFS) {
			f.file(piAgentDir+"/settings.json", `{"defaultProjectTrust": "always"`)
		},
		"unreadable settings": func(f *fakeSkillFS) {
			f.file(piAgentDir+"/settings.json", `{"defaultProjectTrust": "always"}`).failRead(piAgentDir + "/settings.json")
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := newFakeSkillFS().dir(projectTop).file(piAgentDir+"/settings.json", `{"defaultProjectTrust": "always"}`)
			setup(fsys)

			trusted, warnings := cli.ResolvePiProjectTrust(fsys, piAgentDir, projectTop)

			g.Expect(trusted).To(BeFalse())
			g.Expect(warnings).NotTo(BeEmpty())
		})
	}
}

// TestResolvePiProjectTrust_NearestDecisionProperty: for any assignment of
// saved decisions (true / false / null / none) to the cwd and its ancestors,
// and any defaultProjectTrust, the project is trusted iff the nearest non-null
// decision is true, or there is none and the default is "always" (oracle:
// Pi's findNearestTrustEntry + resolveProjectTrusted).
func TestResolvePiProjectTrust_NearestDecisionProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		cwd := projectTop + "/sub/deep"
		chain := []string{cwd, projectTop + "/sub", projectTop, "/work", "/"}
		entries := map[string]any{"/unrelated": true}

		var nearest *bool

		for index, dir := range chain {
			switch rapid.IntRange(0, trustShapeCount-1).Draw(rt, fmt.Sprintf("decision%d", index)) {
			case trustShapeTrue:
				entries[dir] = true
			case trustShapeFalse:
				entries[dir] = false
			case trustShapeNull:
				entries[dir] = nil

				continue
			default:
				continue
			}

			if nearest == nil {
				decision, _ := entries[dir].(bool)
				nearest = &decision
			}
		}

		defaultTrust := rapid.SampledFrom([]string{"always", "ask", "never", ""}).Draw(rt, "default")

		trustJSON, marshalErr := json.Marshal(entries)
		g.Expect(marshalErr).NotTo(HaveOccurred())

		fsys := newFakeSkillFS().
			dir(cwd).
			file(piAgentDir+"/trust.json", string(trustJSON)).
			file(piAgentDir+"/settings.json", fmt.Sprintf(`{"defaultProjectTrust": %q}`, defaultTrust))

		want := defaultTrust == "always"
		if nearest != nil {
			want = *nearest
		}

		trusted, warnings := cli.ResolvePiProjectTrust(fsys, piAgentDir, cwd)

		g.Expect(trusted).To(Equal(want))
		g.Expect(warnings).To(BeEmpty())
	})
}

func TestResolvePiProjectTrust_SymlinkedCwdIsCanonicalized(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// This machine: trust.json trusts /Users/joe/repos/personal; a cwd
	// reached through a symlink resolves to it before the lookup.
	fsys := newFakeSkillFS().
		dir("/Users/joe/repos/personal/pi-skills/sub").
		link("/home/joe/repos", "/Users/joe/repos").
		file(piAgentDir+"/trust.json", `{"/Users/joe/repos/personal": true, "/Users/joe/repos/personal/vim-mode": true}`)

	trusted, warnings := cli.ResolvePiProjectTrust(fsys, piAgentDir, "/home/joe/repos/personal/pi-skills/sub")

	g.Expect(trusted).To(BeTrue())
	g.Expect(warnings).To(BeEmpty())
}

// ---------------------------------------------------------------------------
// Pi project skills (task 1.4, trust-gated per 1.6)
// ---------------------------------------------------------------------------

// TestScanAgentsProjectSkills_ChainProperty: for any cwd depth below the
// top-level and any set of chain levels holding `.agents/skills/<n>`, the
// scan finds exactly those levels' skills (same name at two levels: both),
// records each existing level as its own scanned root, and never reads above
// the top-level.
func TestScanAgentsProjectSkills_ChainProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		depth := rapid.IntRange(0, maxPropertyDepth).Draw(rt, "depth")
		name := rapid.StringMatching(`[a-z][a-z0-9-]{0,4}`)

		levels := make([]string, 0, depth+1)
		level := projectTop

		levels = append(levels, level)

		for index := range depth {
			level = fmt.Sprintf("%s/s%d", level, index)
			levels = append(levels, level)
		}

		cwd := levels[len(levels)-1]
		fsys := newFakeSkillFS().dir(cwd).file("/work/.agents/skills/above/SKILL.md", "above")

		var (
			wantNames []string
			wantRoots []string
		)

		for index, dir := range levels {
			if !rapid.Bool().Draw(rt, fmt.Sprintf("has%d", index)) {
				continue
			}

			skill := name.Draw(rt, fmt.Sprintf("name%d", index))
			fsys.file(dir+"/.agents/skills/"+skill+"/SKILL.md", skill)
			wantNames = append(wantNames, skill)
			wantRoots = append(wantRoots, dir+"/.agents/skills")
		}

		result := cli.ScanAgentsProjectSkills(fsys, cli.PiProjectScan{
			Cwd: cwd, TopLevel: projectTop, UserAgentsRoot: agentsUserRoot, ScopeID: projectScope, Trusted: true,
		})

		gotNames := candidateNames(result)
		sort.Strings(gotNames)
		sort.Strings(wantNames)

		if wantNames == nil {
			wantNames = []string{}
		}

		g.Expect(gotNames).To(Equal(wantNames))

		gotRoots := make([]string, 0, len(result.Roots))

		for _, root := range result.Roots {
			g.Expect(root.Scanned).To(BeTrue())
			gotRoots = append(gotRoots, root.Path)
		}

		sort.Strings(gotRoots)
		sort.Strings(wantRoots)

		if wantRoots == nil {
			wantRoots = []string{}
		}

		g.Expect(gotRoots).To(Equal(wantRoots))
	})
}

func TestScanAgentsProjectSkills_FromSubdirectoryFindsTopLevel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(projectTop+"/.agents/skills/lint/SKILL.md", "lint").
		file(projectTop+"/.agents/skills/notes.md", "root md ignored").
		file("/work/.agents/skills/outside/SKILL.md", "above the top-level").
		dir(projectTop + "/internal/sub")

	result := cli.ScanAgentsProjectSkills(fsys, cli.PiProjectScan{
		Cwd: projectTop + "/internal/sub", TopLevel: projectTop, UserAgentsRoot: agentsUserRoot,
		ScopeID: projectScope, Trusted: true,
	})

	g.Expect(candidateNames(result)).To(Equal([]string{"lint"}))
	g.Expect(result.Candidates[0].ScopeID).To(Equal(projectScope))
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{
		Path: projectTop + "/.agents/skills", Resolved: projectTop + "/.agents/skills", Scanned: true,
	}}))
}

// TestScanAgentsProjectSkills_SkipsUserAgentsRoot: when the chain reaches
// $HOME, ~/.agents/skills is the user source, never a project level.
func TestScanAgentsProjectSkills_SkipsUserAgentsRoot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(agentsUserRoot+"/mine/SKILL.md", "user").
		file("/home/joe/proj/.agents/skills/lint/SKILL.md", "lint")

	result := cli.ScanAgentsProjectSkills(fsys, cli.PiProjectScan{
		Cwd: "/home/joe/proj", TopLevel: "/home/joe", UserAgentsRoot: agentsUserRoot,
		ScopeID: projectScope, Trusted: true,
	})

	g.Expect(candidateNames(result)).To(Equal([]string{"lint"}))
	_, recorded := rootScanned(result, agentsUserRoot)
	g.Expect(recorded).To(BeFalse())
}

// TestScanAgentsProjectSkills_SymlinkedCwdStopsAtTopLevel: a cwd reached
// through a symlink is resolved, so the walk still meets the (physical)
// top-level and never climbs past it.
func TestScanAgentsProjectSkills_SymlinkedCwdStopsAtTopLevel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(projectTop+"/.agents/skills/lint/SKILL.md", "lint").
		file("/work/.agents/skills/outside/SKILL.md", "above").
		dir(projectTop+"/sub").
		link("/alias", projectTop)

	result := cli.ScanAgentsProjectSkills(fsys, cli.PiProjectScan{
		Cwd: "/alias/sub", TopLevel: projectTop, UserAgentsRoot: agentsUserRoot,
		ScopeID: projectScope, Trusted: true,
	})

	g.Expect(candidateNames(result)).To(Equal([]string{"lint"}))
}

// ---------------------------------------------------------------------------
// ScanPiUserSkills / ScanAgentsUserSkills / ScanPiSkillDir (task 1.4)
// ---------------------------------------------------------------------------

func TestScanAgentsUserSkills_IgnoresRootMarkdown(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(agentsUserRoot+"/notes.md", "notes").
		file(agentsUserRoot+"/lint/SKILL.md", "lint")

	result := cli.ScanAgentsUserSkills(fsys, agentsUserRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"lint"}))
	g.Expect(result.Candidates[0].ScopeID).To(Equal(cli.SkillScopeAgentsUser))
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{Path: agentsUserRoot, Resolved: agentsUserRoot, Scanned: true}}))
}

func TestScanPiProjectPrompts_TrustedScansCwdPiPrompts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().file(projectTop+"/.pi/prompts/review.md", "review")

	result := cli.ScanPiProjectPrompts(fsys, cli.PiProjectScan{Cwd: projectTop, ScopeID: projectScope, Trusted: true})

	g.Expect(candidateNames(result)).To(Equal([]string{"review"}))
	g.Expect(result.Candidates[0].ScopeID).To(Equal(projectScope))
	g.Expect(result.Candidates[0].Kind).To(Equal(cli.SkillSourceKindPrompt))
}

func TestScanPiProjectSkills_TrustedScansCwdPiSkills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(projectTop+"/.pi/skills/foo/SKILL.md", "foo").
		file(projectTop+"/.pi/skills/notes.md", "notes").
		file("/work/.pi/skills/parent/SKILL.md", "cwd only, never ancestors")

	result := cli.ScanPiProjectSkills(fsys, cli.PiProjectScan{Cwd: projectTop, ScopeID: projectScope, Trusted: true})

	g.Expect(candidateNames(result)).To(Equal([]string{"foo", "notes"}))
	g.Expect(result.Candidates[0].ScopeID).To(Equal(projectScope))
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{
		Path: projectTop + "/.pi/skills", Resolved: projectTop + "/.pi/skills", Scanned: true,
	}}))
}

// TestScanPiProjectSources_UntrustedReadsNothing: untrusted → the Pi project
// scanners read nothing and record no root (spec "Untrusted Pi project is not
// scanned").
func TestScanPiProjectSources_UntrustedReadsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fake := newFakeSkillFS().
		file(projectTop+"/.pi/skills/foo/SKILL.md", "foo").
		file(projectTop+"/.pi/prompts/review.md", "review").
		file(projectTop+"/.agents/skills/lint/SKILL.md", "lint")
	recorder := &recordingSkillFS{inner: fake}
	scan := cli.PiProjectScan{
		Cwd: projectTop, TopLevel: projectTop, UserAgentsRoot: agentsUserRoot, ScopeID: projectScope, Trusted: false,
	}

	for _, result := range []cli.SkillScanResult{
		cli.ScanPiProjectSkills(recorder, scan),
		cli.ScanAgentsProjectSkills(recorder, scan),
		cli.ScanPiProjectPrompts(recorder, scan),
	} {
		g.Expect(result.Candidates).To(BeEmpty())
		g.Expect(result.Roots).To(BeEmpty())
	}

	g.Expect(recorder.calls).To(BeEmpty())
}

// ---------------------------------------------------------------------------
// Pi prompt templates (task 1.4b)
// ---------------------------------------------------------------------------

// TestScanPiPromptDir_FlatMarkdownProperty: for any flat mix of entries, the
// prompts are exactly the non-hidden `.md` files (regular or symlinked),
// named by stem; subdirectories are never descended.
func TestScanPiPromptDir_FlatMarkdownProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		names := rapid.SliceOfDistinct(rapid.StringMatching(`\.?[a-z][a-z0-9-]{0,5}`), rapid.ID[string]).
			Draw(rt, "names")
		fsys := newFakeSkillFS().dir(piPromptsRoot)
		want := []string{}

		for index, name := range names {
			switch rapid.IntRange(0, promptShapeCount-1).Draw(rt, fmt.Sprintf("shape%d", index)) {
			case promptShapeMarkdown:
				fsys.file(piPromptsRoot+"/"+name+".md", name)

				if !strings.HasPrefix(name, ".") {
					want = append(want, name)
				}
			case promptShapeLinked:
				fsys.file("/src/prompts/"+name+".md", name).link(piPromptsRoot+"/"+name+".md", "/src/prompts/"+name+".md")

				if !strings.HasPrefix(name, ".") {
					want = append(want, name)
				}
			case promptShapeOther:
				fsys.file(piPromptsRoot+"/"+name+".txt", name)
			case promptShapeSubdir:
				fsys.file(piPromptsRoot+"/"+name+"/nested.md", name)
			}
		}

		sort.Strings(want)

		result := cli.ScanPiPromptDir(fsys, piPromptsRoot, cli.SkillScopePiPrompt)

		g.Expect(candidateNames(result)).To(Equal(want))

		for _, candidate := range result.Candidates {
			g.Expect(candidate.Kind).To(Equal(cli.SkillSourceKindPrompt))
		}

		scanned, _ := rootScanned(result, piPromptsRoot)
		g.Expect(scanned).To(BeTrue())
	})
}

func TestScanPiSkillDir_ChildStatErrorMarksRootNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piUserRoot+"/good/SKILL.md", "good").
		link(piUserRoot+"/odd", "/elsewhere/odd").
		dir("/elsewhere/odd").
		failLstat("/elsewhere/odd")

	result := cli.ScanPiUserSkills(fsys, piUserRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"good"}))

	scanned, _ := rootScanned(result, piUserRoot)
	g.Expect(scanned).To(BeFalse())
	g.Expect(warningsMention(result, piUserRoot+"/odd")).To(BeTrue())
}

func TestScanPiSkillDir_CycleTerminates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piUserRoot+"/a/foo/SKILL.md", "foo").
		link(piUserRoot+"/a/loop", piUserRoot+"/a").
		link(piUserRoot+"/a/up", piUserRoot)

	result := cli.ScanPiUserSkills(fsys, piUserRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"foo"}))

	scanned, _ := rootScanned(result, piUserRoot)
	g.Expect(scanned).To(BeTrue())
}

func TestScanPiSkillDir_DepthLimitMarksRootNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deep := piUserRoot + strings.Repeat("/d", piDeepNesting)
	fsys := newFakeSkillFS().
		file(piUserRoot+"/shallow/SKILL.md", "shallow").
		file(deep+"/buried/SKILL.md", "buried")

	result := cli.ScanPiUserSkills(fsys, piUserRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"shallow"}))

	scanned, _ := rootScanned(result, piUserRoot)
	g.Expect(scanned).To(BeFalse())
	g.Expect(result.Warnings).NotTo(BeEmpty())
}

// TestScanPiSkillDir_DiscoveryProperty: for any tree of directories (some
// holding SKILL.md) plus markdown files, the scan finds exactly the
// directories holding SKILL.md with no SKILL.md-holding ancestor below the
// root, named by directory name, plus — for the Pi mode only — root-level
// `.md` files named by stem (oracle computed from the generated tree, Pi's
// collectSkillEntries rules).
func TestScanPiSkillDir_DiscoveryProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		segment := rapid.StringMatching(`[a-z][a-z0-9-]{0,4}`)
		dirs := rapid.SliceOfNDistinct(
			rapid.SliceOfN(segment, 1, maxPropertyDepth),
			0, maxPropertyFiles,
			func(parts []string) string { return strings.Join(parts, "/") },
		).Draw(rt, "dirs")
		rootFiles := rapid.SliceOfDistinct(segment, rapid.ID[string]).Draw(rt, "rootFiles")
		includeRootFiles := rapid.Bool().Draw(rt, "includeRootFiles")

		fsys := newFakeSkillFS().dir(piUserRoot)
		skillDirs := map[string]bool{}

		for index, parts := range dirs {
			rel := strings.Join(parts, "/")
			fsys.dir(piUserRoot + "/" + rel)

			if rapid.Bool().Draw(rt, fmt.Sprintf("skill%d", index)) {
				fsys.file(piUserRoot+"/"+rel+"/SKILL.md", rel)
				skillDirs[rel] = true
			}

			// A nested markdown file is never a skill on its own.
			fsys.file(piUserRoot+"/"+rel+"/extra.md", "extra")
		}

		var want []string

		for rel := range skillDirs {
			if !hasSkillAncestor(rel, skillDirs) {
				want = append(want, filepath.Base(rel))
			}
		}

		for _, name := range rootFiles {
			fsys.file(piUserRoot+"/"+name+".md", name)

			if includeRootFiles {
				want = append(want, name)
			}
		}

		sort.Strings(want)

		result := cli.ScanPiSkillDir(fsys, piUserRoot, cli.SkillScopePiUser, includeRootFiles)

		got := candidateNames(result)
		sort.Strings(got)

		if want == nil {
			want = []string{}
		}

		g.Expect(got).To(Equal(want))

		scanned, _ := rootScanned(result, piUserRoot)
		g.Expect(scanned).To(BeTrue())
	})
}

func TestScanPiSkillDir_HiddenAndNodeModulesAreSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piUserRoot+"/.hidden/SKILL.md", "hidden").
		file(piUserRoot+"/node_modules/dep/SKILL.md", "dep").
		file(piUserRoot+"/.draft.md", "draft").
		file(piUserRoot+"/kept/SKILL.md", "kept")

	result := cli.ScanPiUserSkills(fsys, piUserRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"kept"}))
}

func TestScanPiSkillDir_MissingRootIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := cli.ScanPiUserSkills(newFakeSkillFS(), piUserRoot)

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.Warnings).To(BeEmpty())
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{Path: piUserRoot}}))
}

func TestScanPiSkillDir_RootHoldingSkillIsOneSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piUserRoot+"/SKILL.md", "root skill").
		file(piUserRoot+"/notes.md", "notes").
		file(piUserRoot+"/inner/SKILL.md", "inner")

	result := cli.ScanPiUserSkills(fsys, piUserRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"skills"}))
	g.Expect(result.Candidates[0].SourcePath).To(Equal(piUserRoot + "/SKILL.md"))
}

func TestScanPiSkillDir_SkillFileReadErrorMarksRootNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piUserRoot+"/good/SKILL.md", "good").
		file(piUserRoot+"/locked/SKILL.md", "locked").
		file(piUserRoot+"/notes.md", "notes").
		failRead(piUserRoot + "/locked/SKILL.md").
		failRead(piUserRoot + "/notes.md")

	result := cli.ScanPiUserSkills(fsys, piUserRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"good"}))

	scanned, _ := rootScanned(result, piUserRoot)
	g.Expect(scanned).To(BeFalse())
	g.Expect(warningsMention(result, piUserRoot+"/locked/SKILL.md")).To(BeTrue())
	g.Expect(warningsMention(result, piUserRoot+"/notes.md")).To(BeTrue())
}

func TestScanPiSkillDir_SubdirReadErrorMarksRootNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piUserRoot+"/good/SKILL.md", "good").
		file(piUserRoot+"/group/locked/SKILL.md", "locked").
		failRead(piUserRoot + "/group")

	result := cli.ScanPiUserSkills(fsys, piUserRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"good"}))

	scanned, _ := rootScanned(result, piUserRoot)
	g.Expect(scanned).To(BeFalse())
	g.Expect(warningsMention(result, piUserRoot+"/group")).To(BeTrue())
}

func TestScanPiUserPrompts_GlobalPromptNamedByStem(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().file(piPromptsRoot+"/review.md", "---\ndescription: Review\n---\nReview it")

	result := cli.ScanPiUserPrompts(fsys, piPromptsRoot)

	g.Expect(result.Candidates).To(Equal([]cli.SkillCandidate{{
		Name:       "review",
		ScopeID:    cli.SkillScopePiPrompt,
		ReadRoot:   piPromptsRoot,
		SourcePath: piPromptsRoot + "/review.md",
		WalkedPath: piPromptsRoot + "/review.md",
		Kind:       cli.SkillSourceKindPrompt,
		Content:    []byte("---\ndescription: Review\n---\nReview it"),
	}}))
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{Path: piPromptsRoot, Resolved: piPromptsRoot, Scanned: true}}))
}

func TestScanPiUserPrompts_MissingDirIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := cli.ScanPiUserPrompts(newFakeSkillFS(), piPromptsRoot)

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.Warnings).To(BeEmpty())
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{Path: piPromptsRoot}}))
}

func TestScanPiUserPrompts_UnreadablePromptMarksRootNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piPromptsRoot+"/good.md", "good").
		file(piPromptsRoot+"/locked.md", "locked").
		failRead(piPromptsRoot+"/locked.md").
		link(piPromptsRoot+"/gone.md", "/nowhere/gone.md")

	result := cli.ScanPiUserPrompts(fsys, piPromptsRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"good"}))

	scanned, _ := rootScanned(result, piPromptsRoot)
	g.Expect(scanned).To(BeFalse())
	g.Expect(warningsMention(result, piPromptsRoot+"/locked.md")).To(BeTrue())
}

func TestScanPiUserSkills_RecursiveDiscoveryAndRootMarkdown(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piUserRoot+"/a/foo/SKILL.md", "foo a").
		file(piUserRoot+"/b/foo/SKILL.md", "foo b").
		file(piUserRoot+"/bar/SKILL.md", "bar").
		file(piUserRoot+"/bar/nested/SKILL.md", "nested stops at bar").
		file(piUserRoot+"/a/deep.md", "not a root file").
		file(piUserRoot+"/notes.md", "notes").
		file(piUserRoot+"/readme.txt", "not markdown")

	result := cli.ScanPiUserSkills(fsys, piUserRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"bar", "foo", "foo", "notes"}))

	for _, candidate := range result.Candidates {
		g.Expect(candidate.ScopeID).To(Equal(cli.SkillScopePiUser))
		g.Expect(candidate.Kind).To(Equal(cli.SkillSourceKindSkill))
		g.Expect(candidate.ReadRoot).To(Equal(piUserRoot))
		g.Expect(candidate.Key).To(BeEmpty())
	}

	g.Expect(result.Candidates[1].SourcePath).To(Equal(piUserRoot + "/a/foo/SKILL.md"))
	g.Expect(string(result.Candidates[1].Content)).To(Equal("foo a"))
	g.Expect(result.Candidates[3].SourcePath).To(Equal(piUserRoot + "/notes.md"))
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{Path: piUserRoot, Resolved: piUserRoot, Scanned: true}}))
}

// TestScanPiUserSkills_SymlinksFollowed mirrors this machine: engram's skills
// are symlinks into ~/.pi/agent/engram/skills; a dangling link is skipped.
func TestScanPiUserSkills_SymlinksFollowed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	engramRoute := piAgentDir + "/engram/skills/route"
	fsys := newFakeSkillFS().
		file(engramRoute+"/SKILL.md", "route").
		link(piUserRoot+"/route", "../engram/skills/route").
		file("/src/ping/SKILL.md", "ping").
		dir(piUserRoot+"/ping").
		link(piUserRoot+"/ping/SKILL.md", "/src/ping/SKILL.md").
		file("/src/tip.md", "tip").
		link(piUserRoot+"/tip.md", "/src/tip.md").
		link(piUserRoot+"/gone", "/nowhere/gone")

	result := cli.ScanPiUserSkills(fsys, piUserRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"ping", "route", "tip"}))
	g.Expect(result.Candidates[0].SourcePath).To(Equal("/src/ping/SKILL.md"))
	g.Expect(result.Candidates[1].SourcePath).To(Equal(engramRoute + "/SKILL.md"))
	g.Expect(result.Candidates[2].SourcePath).To(Equal("/src/tip.md"))

	scanned, _ := rootScanned(result, piUserRoot)
	g.Expect(scanned).To(BeTrue())
}

// unexported constants.
const (
	agentsUserRoot      = "/home/joe/.agents/skills"
	piAgentDir          = "/home/joe/.pi/agent"
	piDeepNesting       = 20
	piPromptsRoot       = piAgentDir + "/prompts"
	piUserRoot          = piAgentDir + "/skills"
	projectScope        = "project:github.com/toejough/engram"
	projectTop          = "/work/engram"
	promptShapeCount    = 4
	promptShapeLinked   = 1
	promptShapeMarkdown = 0
	promptShapeOther    = 2
	promptShapeSubdir   = 3
	trustShapeCount     = 4
	trustShapeFalse     = 1
	trustShapeNull      = 2
	trustShapeTrue      = 0
)

// recordingSkillFS wraps a SkillSourceFS and records every call's path.
type recordingSkillFS struct {
	inner cli.SkillSourceFS
	calls []string
}

func (r *recordingSkillFS) Lstat(path string) (fs.FileInfo, error) {
	r.calls = append(r.calls, "lstat "+path)

	return r.inner.Lstat(path)
}

func (r *recordingSkillFS) ReadDir(path string) ([]fs.DirEntry, error) {
	r.calls = append(r.calls, "readdir "+path)

	return r.inner.ReadDir(path)
}

func (r *recordingSkillFS) ReadFile(path string) ([]byte, error) {
	r.calls = append(r.calls, "readfile "+path)

	return r.inner.ReadFile(path)
}

func (r *recordingSkillFS) Readlink(path string) (string, error) {
	r.calls = append(r.calls, "readlink "+path)

	return r.inner.Readlink(path)
}

// hasSkillAncestor reports whether any proper ancestor of rel (below the
// scan root) is itself a skill directory.
func hasSkillAncestor(rel string, skillDirs map[string]bool) bool {
	parts := strings.Split(rel, "/")

	for index := 1; index < len(parts); index++ {
		if skillDirs[strings.Join(parts[:index], "/")] {
			return true
		}
	}

	return false
}

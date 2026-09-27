package cli_test

import (
	"encoding/json"
	"errors"
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

// TestResolveSkillPath_ChainProperty: for any chain of symlinks (absolute or
// relative targets) ending at a real directory, resolving any link in the
// chain yields the real directory (reference oracle: the generator knows the
// terminal), and resolution is idempotent.
func TestResolveSkillPath_ChainProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		chainLen := rapid.IntRange(1, maxPropertyChain).Draw(rt, "chainLen")
		final := "/real/" + rapid.StringMatching(`[a-z]{1,6}`).Draw(rt, "final")
		fsys := newFakeSkillFS().dir(final)

		target := final
		start := ""

		for index := range chainLen {
			linkPath := fmt.Sprintf("/links/l%d", index)

			linkTarget := target
			if rapid.Bool().Draw(rt, "relative") {
				rel, relErr := filepath.Rel(filepath.Dir(linkPath), target)
				g.Expect(relErr).NotTo(HaveOccurred())

				linkTarget = rel
			}

			fsys.link(linkPath, linkTarget)
			target = linkPath
			start = linkPath
		}

		resolved, err := cli.ResolveSkillPath(fsys, start)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resolved).To(Equal(final))

		again, againErr := cli.ResolveSkillPath(fsys, resolved)
		g.Expect(againErr).NotTo(HaveOccurred())
		g.Expect(again).To(Equal(resolved))
	})
}

//go:generate impgen cli.SkillSourceFS --dependency --import-path github.com/toejough/engram/internal/cli

// ---------------------------------------------------------------------------
// ResolveSkillPath
// ---------------------------------------------------------------------------

func TestResolveSkillPath_DanglingLinkIsNotExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().link("/home/.claude/skills/qa", "/nowhere/qa")

	_, err := cli.ResolveSkillPath(fsys, "/home/.claude/skills/qa")

	g.Expect(errors.Is(err, fs.ErrNotExist)).To(BeTrue())
}

func TestResolveSkillPath_DotDotEndsOnParentDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().dir("/real/x")

	resolved, err := cli.ResolveSkillPath(fsys, "/real/x/..")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved).To(Equal("/real"))

	root, rootErr := cli.ResolveSkillPath(fsys, "/")
	g.Expect(rootErr).NotTo(HaveOccurred())
	g.Expect(root).To(Equal("/"))

	_, relErr := cli.ResolveSkillPath(fsys, "relative/path")
	g.Expect(relErr).To(HaveOccurred())

	_, unreadableErr := cli.ResolveSkillPath(fsys.failLstat("/"), "/real/..")
	g.Expect(errors.Is(unreadableErr, fs.ErrPermission)).To(BeTrue())
}

func TestResolveSkillPath_LinkLoopErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().link("/a", "/b").link("/b", "/a")

	_, err := cli.ResolveSkillPath(fsys, "/a/x")

	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, fs.ErrNotExist)).To(BeFalse())
}

func TestResolveSkillPath_RelativeTargetAndSymlinkedParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// $HOME is itself a symlink, and the skill link's target is relative.
	fsys := newFakeSkillFS().
		link("/home", "/Users").
		file("/Users/joe/.claude/engram/skills/route/SKILL.md", "route").
		link("/Users/joe/.claude/skills/route", "../engram/skills/route")

	resolved, err := cli.ResolveSkillPath(fsys, "/home/joe/.claude/skills/route/SKILL.md")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved).To(Equal("/Users/joe/.claude/engram/skills/route/SKILL.md"))
}

// ---------------------------------------------------------------------------
// ScanClaudeUserSkills (task 1.1)
// ---------------------------------------------------------------------------

func TestScanClaudeUserSkills_ChildReadErrorMarksRootNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(userSkillsRoot+"/good/SKILL.md", "good").
		file(userSkillsRoot+"/locked/SKILL.md", "locked").
		failRead(userSkillsRoot + "/locked/SKILL.md")

	result := cli.ScanClaudeUserSkills(fsys, userSkillsRoot)

	// A child that exists but can't be read must never look absent (it
	// could trigger a removal offer), so the whole root is not scanned.
	g.Expect(candidateNames(result)).To(Equal([]string{"good"}))
	scanned, found := rootScanned(result, userSkillsRoot)
	g.Expect(found).To(BeTrue())
	g.Expect(scanned).To(BeFalse())
	g.Expect(warningsMention(result, userSkillsRoot+"/locked/SKILL.md")).To(BeTrue())
}

func TestScanClaudeUserSkills_ChildStatErrorMarksRootNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(userSkillsRoot+"/good/SKILL.md", "good").
		link(userSkillsRoot+"/odd", "/elsewhere/odd").
		dir("/elsewhere/odd").
		failLstat("/elsewhere/odd")

	result := cli.ScanClaudeUserSkills(fsys, userSkillsRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"good"}))
	scanned, _ := rootScanned(result, userSkillsRoot)
	g.Expect(scanned).To(BeFalse())
	g.Expect(warningsMention(result, userSkillsRoot+"/odd")).To(BeTrue())
}

func TestScanClaudeUserSkills_MissingRootIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := cli.ScanClaudeUserSkills(newFakeSkillFS(), userSkillsRoot)

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{Path: userSkillsRoot}}))
}

// TestScanClaudeUserSkills_ReadDirErrorIsNotScanned drives the FS
// interactively: the root resolves (one Lstat per path component), then its
// ReadDir fails with a non-not-exist error — the root is recorded resolved but
// not scanned, and nothing else is read.
func TestScanClaudeUserSkills_ReadDirErrorIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mock, imp := MockSkillSourceFS(t)
	done := make(chan cli.SkillScanResult)

	go func() {
		done <- cli.ScanClaudeUserSkills(mock, "/r")
	}()

	imp.Lstat.ArgsEqual("/r").Return(fakeSkillInfo{name: "r", node: &fakeSkillNode{kind: fakeSkillDir}}, nil)
	imp.ReadDir.ArgsEqual("/r").Return(nil, errFakeReadDenied)

	result := <-done

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{Path: "/r", Resolved: "/r"}}))
}

func TestScanClaudeUserSkills_ReadErrorOnRootIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().file(userSkillsRoot+"/c4/SKILL.md", "c4").failRead(userSkillsRoot)

	result := cli.ScanClaudeUserSkills(fsys, userSkillsRoot)

	g.Expect(result.Candidates).To(BeEmpty())
	scanned, found := rootScanned(result, userSkillsRoot)
	g.Expect(found).To(BeTrue())
	g.Expect(scanned).To(BeFalse())
}

func TestScanClaudeUserSkills_SkipsNonSkillEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(userSkillsRoot+"/c4/SKILL.md", "c4").
		link(userSkillsRoot+"/qa", "/home/joe/repos/projctl/skills/qa"). // dangling
		dir(userSkillsRoot+"/.obsidian").                                // no SKILL.md
		file(userSkillsRoot+"/commit.md", "root file").
		file(userSkillsRoot+"/synced/SKILL.md", "synced is never a skill").
		file("/elsewhere/notes.md", "file target").
		link(userSkillsRoot+"/notes", "/elsewhere/notes.md") // symlink to a file

	result := cli.ScanClaudeUserSkills(fsys, userSkillsRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"c4"}))
	g.Expect(result.Warnings).To(BeEmpty())
	scanned, _ := rootScanned(result, userSkillsRoot)
	g.Expect(scanned).To(BeTrue())
}

func TestScanClaudeUserSkills_SymlinkedRootIsResolved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Fixture home: ~/.claude/skills is itself a symlink into the real tree.
	fsys := newFakeSkillFS().
		file("/real/skills/c4/SKILL.md", "c4").
		link(userSkillsRoot, "/real/skills")

	result := cli.ScanClaudeUserSkills(fsys, userSkillsRoot)

	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{Path: userSkillsRoot, Resolved: "/real/skills", Scanned: true}}))
	g.Expect(result.Candidates).To(HaveLen(1))

	if len(result.Candidates) != 1 {
		return
	}

	g.Expect(result.Candidates[0].ReadRoot).To(Equal("/real/skills"))
	g.Expect(result.Candidates[0].SourcePath).To(Equal("/real/skills/c4/SKILL.md"))
}

func TestScanClaudeUserSkills_SymlinkedSkillIsFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := "# Route\n"
	fsys := newFakeSkillFS().
		file("/home/joe/.claude/engram/skills/route/SKILL.md", content).
		link(userSkillsRoot+"/route", "/home/joe/.claude/engram/skills/route")

	result := cli.ScanClaudeUserSkills(fsys, userSkillsRoot)

	g.Expect(result.Candidates).To(Equal([]cli.SkillCandidate{{
		Name:       "route",
		ScopeID:    cli.SkillScopeClaudeUser,
		ReadRoot:   userSkillsRoot,
		SourcePath: "/home/joe/.claude/engram/skills/route/SKILL.md",
		Kind:       cli.SkillSourceKindSkill,
		Content:    []byte(content),
	}}))
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{Path: userSkillsRoot, Resolved: userSkillsRoot, Scanned: true}}))
}

// ---------------------------------------------------------------------------
// ScanCommandDir / ScanPluginCommands (task 1.2)
// ---------------------------------------------------------------------------

func TestScanCommandDir_CycleTerminates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(userCommandsRoot+"/a/x.md", "x").
		link(userCommandsRoot+"/a/loop", userCommandsRoot+"/a")

	result := cli.ScanCommandDir(fsys, userCommandsRoot, cli.SkillScopeClaudeCmd)

	g.Expect(candidateNames(result)).To(Equal([]string{"a:x"}))
}

func TestScanCommandDir_MissingRootIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := cli.ScanCommandDir(newFakeSkillFS(), userCommandsRoot, cli.SkillScopeClaudeCmd)

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{Path: userCommandsRoot}}))
}

// TestScanCommandDir_NameProperty: for any tree of .md files (plus non-.md
// noise), the scanned names are exactly the generated relative paths with
// `/` → `:` and `.md` stripped (the generator's segment lists are the
// oracle), and mapping a name back (`:` → `/`, + `.md`) under the root gives
// its SourcePath (round trip).
func TestScanCommandDir_NameProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		fsys := newFakeSkillFS().dir(userCommandsRoot)
		segment := rapid.StringMatching(`[a-z][a-z0-9_-]{0,5}`)
		paths := rapid.SliceOfNDistinct(
			rapid.SliceOfN(segment, 1, maxPropertyDepth),
			0, maxPropertyFiles,
			func(segments []string) string { return strings.Join(segments, "/") },
		).Draw(rt, "paths")

		want := []string{}

		for _, segments := range paths {
			rel := strings.Join(segments, "/")

			if rapid.Bool().Draw(rt, "markdown") {
				fsys.file(userCommandsRoot+"/"+rel+".md", rel)
				want = append(want, strings.Join(segments, ":"))
			} else {
				fsys.file(userCommandsRoot+"/"+rel+".txt", rel)
			}
		}

		sort.Strings(want)

		result := cli.ScanCommandDir(fsys, userCommandsRoot, cli.SkillScopeClaudeCmd)

		g.Expect(candidateNames(result)).To(Equal(want))

		for _, candidate := range result.Candidates {
			g.Expect(candidate.SourcePath).To(Equal(
				userCommandsRoot + "/" + strings.ReplaceAll(candidate.Name, ":", "/") + ".md"))
			g.Expect(candidate.Kind).To(Equal(cli.SkillSourceKindCommand))
		}
	})
}

func TestScanCommandDir_NamesFollowSymlinksAndSkipNonMarkdown(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(userCommandsRoot+"/audit.md", "audit").
		file(userCommandsRoot+"/opsx/apply.md", "apply").
		file(userCommandsRoot+"/notes.txt", "not a command").
		file("/shared/db/migrate.md", "migrate").
		link(userCommandsRoot+"/db", "/shared/db").                                       // symlinked subdir
		file("/shared/one.md", "one").link(userCommandsRoot+"/one.md", "/shared/one.md"). // symlinked file
		link(userCommandsRoot+"/gone.md", "/nowhere.md")                                  // dangling

	result := cli.ScanCommandDir(fsys, userCommandsRoot, cli.SkillScopeClaudeCmd)

	g.Expect(candidateNames(result)).To(Equal([]string{"audit", "db:migrate", "one", "opsx:apply"}))
	g.Expect(result.Warnings).To(BeEmpty())
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{
		{Path: userCommandsRoot, Resolved: userCommandsRoot, Scanned: true},
	}))

	byName := map[string]cli.SkillCandidate{}
	for _, candidate := range result.Candidates {
		byName[candidate.Name] = candidate
	}

	g.Expect(byName["opsx:apply"]).To(Equal(cli.SkillCandidate{
		Name:       "opsx:apply",
		ScopeID:    cli.SkillScopeClaudeCmd,
		ReadRoot:   userCommandsRoot,
		SourcePath: userCommandsRoot + "/opsx/apply.md",
		Kind:       cli.SkillSourceKindCommand,
		Content:    []byte("apply"),
	}))
	g.Expect(byName["db:migrate"].SourcePath).To(Equal("/shared/db/migrate.md"))
	g.Expect(byName["one"].SourcePath).To(Equal("/shared/one.md"))
}

func TestScanCommandDir_SubdirReadErrorMarksRootNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(userCommandsRoot+"/audit.md", "audit").
		file(userCommandsRoot+"/locked/x.md", "x").
		failRead(userCommandsRoot + "/locked")

	result := cli.ScanCommandDir(fsys, userCommandsRoot, "project:github.com/o/r")

	g.Expect(candidateNames(result)).To(Equal([]string{"audit"}))
	g.Expect(result.Candidates[0].ScopeID).To(Equal("project:github.com/o/r"))
	scanned, _ := rootScanned(result, userCommandsRoot)
	g.Expect(scanned).To(BeFalse())
	g.Expect(warningsMention(result, userCommandsRoot+"/locked")).To(BeTrue())
}

func TestScanPluginCommands_AbsentFieldUsesCommandsDir(t *testing.T) {
	t.Parallel()

	for _, field := range []json.RawMessage{nil, json.RawMessage("null")} {
		g := NewWithT(t)

		fsys := newFakeSkillFS().file(pluginRoot+"/commands/db/migrate.md", "m")

		result := cli.ScanPluginCommands(fsys, pluginRoot, field, "plugin:tools")

		g.Expect(candidateNames(result)).To(Equal([]string{"db:migrate"}))
		g.Expect(result.Candidates[0].ScopeID).To(Equal("plugin:tools"))
		scanned, _ := rootScanned(result, pluginRoot+"/commands")
		g.Expect(scanned).To(BeTrue())
	}
}

func TestScanPluginCommands_ArrayReplacesCommandsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(pluginRoot+"/commands/setup.md", "setup").
		file(pluginRoot+"/commands/configure.md", "configure")

	result := cli.ScanPluginCommands(fsys, pluginRoot, json.RawMessage(`["./commands/setup.md"]`), "plugin:claude-hud")

	g.Expect(result.Candidates).To(Equal([]cli.SkillCandidate{{
		Name:       "setup",
		ScopeID:    "plugin:claude-hud",
		ReadRoot:   pluginRoot + "/commands/setup.md",
		SourcePath: pluginRoot + "/commands/setup.md",
		Kind:       cli.SkillSourceKindCommand,
		Content:    []byte("setup"),
	}}))
	scanned, found := rootScanned(result, pluginRoot+"/commands/setup.md")
	g.Expect(found).To(BeTrue())
	g.Expect(scanned).To(BeTrue())
}

func TestScanPluginCommands_DeclaredFileReadErrorIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(pluginRoot+"/commands/setup.md", "setup").
		failRead(pluginRoot + "/commands/setup.md")

	result := cli.ScanPluginCommands(fsys, pluginRoot, json.RawMessage(`"./commands/setup.md"`), "plugin:p")

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{
		{Path: pluginRoot + "/commands/setup.md", Resolved: pluginRoot + "/commands/setup.md"},
	}))
	g.Expect(warningsMention(result, pluginRoot+"/commands/setup.md")).To(BeTrue())
}

func TestScanPluginCommands_MissingDeclaredPathIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().file(pluginRoot+"/extra/b.md", "b")

	result := cli.ScanPluginCommands(fsys, pluginRoot, json.RawMessage(`["./gone.md", "./extra"]`), "plugin:p")

	g.Expect(candidateNames(result)).To(Equal([]string{"b"}))
	gone, _ := rootScanned(result, pluginRoot+"/gone.md")
	g.Expect(gone).To(BeFalse())
	g.Expect(warningsMention(result, "./gone.md")).To(BeTrue(), "%v", result.Warnings)
	extra, _ := rootScanned(result, pluginRoot+"/extra")
	g.Expect(extra).To(BeTrue())
}

func TestScanPluginCommands_ObjectMapIsSkippedWithWarning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().file(pluginRoot+"/commands/setup.md", "setup")

	result := cli.ScanPluginCommands(fsys, pluginRoot,
		json.RawMessage(`{"setup": {"source": "./commands/setup.md"}}`), "plugin:p")

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{Path: pluginRoot + "/commands"}}))
	g.Expect(warningsMention(result, pluginRoot)).To(BeTrue())
}

func TestScanPluginCommands_StringPathToDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(pluginRoot+"/cmds/run.md", "run").
		file(pluginRoot+"/commands/ignored.md", "ignored")

	result := cli.ScanPluginCommands(fsys, pluginRoot, json.RawMessage(`"./cmds"`), "plugin:p")

	g.Expect(candidateNames(result)).To(Equal([]string{"run"}))
}

// TestScanSkillChildren_Property: over any mix of child shapes, the scanner
// emits exactly the children the generator built as skills (a dir, or a
// symlink resolving to a dir, containing SKILL.md, not excluded), each with
// its own bytes and resolved path; the root is scanned (model-based oracle).
func TestScanSkillChildren_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		fsys := newFakeSkillFS().dir(userSkillsRoot)
		names := rapid.SliceOfDistinct(rapid.StringMatching(`[a-z][a-z0-9-]{0,7}`), rapid.ID[string]).
			Draw(rt, "names")
		excluded := rapid.SampledFrom(append([]string{"synced"}, names...)).Draw(rt, "excluded")
		want := map[string]string{}

		for _, name := range names {
			child := userSkillsRoot + "/" + name
			shape := rapid.IntRange(0, childShapeCount-1).Draw(rt, "shape")

			switch shape {
			case childShapeSkillDir:
				fsys.file(child+"/SKILL.md", "dir:"+name)
				want[name] = child + "/SKILL.md"
			case childShapeLinkedSkill:
				fsys.file("/target/"+name+"/SKILL.md", "link:"+name)
				fsys.link(child, "/target/"+name)
				want[name] = "/target/" + name + "/SKILL.md"
			case childShapeDangling:
				fsys.link(child, "/missing/"+name)
			case childShapeBareDir:
				fsys.dir(child)
			case childShapeRootFile:
				fsys.file(child, "file:"+name)
			}
		}

		delete(want, excluded)

		result := cli.ScanSkillChildren(fsys, userSkillsRoot, "scope-x", excluded)

		got := map[string]string{}
		for _, candidate := range result.Candidates {
			got[candidate.Name] = candidate.SourcePath
			g.Expect(candidate.ScopeID).To(Equal("scope-x"))
			g.Expect(candidate.Kind).To(Equal(cli.SkillSourceKindSkill))
			g.Expect(strings.HasSuffix(string(candidate.Content), ":"+candidate.Name)).To(BeTrue())
		}

		g.Expect(got).To(Equal(want))
		g.Expect(result.Warnings).To(BeEmpty())
		scanned, _ := rootScanned(result, userSkillsRoot)
		g.Expect(scanned).To(BeTrue())
	})
}

// ---------------------------------------------------------------------------
// ScanSyncedSkills (task 1.3)
// ---------------------------------------------------------------------------

func TestScanSyncedSkills_BadManifestBucketContributesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(syncedRoot+"/a/manifest.json", `{"skills":[]}`).
		file(syncedRoot+"/a/docx/SKILL.md", "docx").
		file(syncedRoot+"/b/manifest.json", `{not json`).
		file(syncedRoot+"/b/xlsx/SKILL.md", "xlsx").
		file(syncedRoot+"/c/pptx/SKILL.md", "no manifest")

	result := cli.ScanSyncedSkills(fsys, syncedRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"docx"}))

	for bucket, want := range map[string]bool{"a": true, "b": false, "c": false} {
		scanned, found := rootScanned(result, syncedRoot+"/"+bucket)
		g.Expect(found).To(BeTrue(), bucket)
		g.Expect(scanned).To(Equal(want), bucket)
	}

	scanned, _ := rootScanned(result, syncedRoot)
	g.Expect(scanned).To(BeTrue())
}

func TestScanSyncedSkills_DuplicateNameAcrossBucketsEmitsBoth(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Collapse/conflict of the shared key is task 2.3; the scanner reports
	// both copies with their own paths.
	fsys := newFakeSkillFS().
		file(syncedRoot+"/a/manifest.json", `{}`).
		file(syncedRoot+"/a/pdf/SKILL.md", "pdf").
		file(syncedRoot+"/b/manifest.json", `{}`).
		file(syncedRoot+"/b/pdf/SKILL.md", "pdf")

	result := cli.ScanSyncedSkills(fsys, syncedRoot)

	g.Expect(candidateNames(result)).To(Equal([]string{"pdf", "pdf"}))
	g.Expect(result.Candidates[0].SourcePath).NotTo(Equal(result.Candidates[1].SourcePath))
}

func TestScanSyncedSkills_ManifestBucketAndStrayFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bucket := syncedRoot + "/4227b111_e76eb279"
	fsys := newFakeSkillFS().
		file(bucket+"/manifest.json", `{"skills":[{"skillId":"pdf"}]}`).
		file(bucket+"/pdf/SKILL.md", "pdf").
		file(syncedRoot+"/.bucket-4227b111_e76eb279", "")

	result := cli.ScanSyncedSkills(fsys, syncedRoot)

	g.Expect(result.Candidates).To(Equal([]cli.SkillCandidate{{
		Name:       "pdf",
		ScopeID:    cli.SkillScopeSynced,
		ReadRoot:   bucket,
		SourcePath: bucket + "/pdf/SKILL.md",
		Kind:       cli.SkillSourceKindSkill,
		Content:    []byte("pdf"),
	}}))
	g.Expect(result.Warnings).To(BeEmpty())
	g.Expect(result.Roots).To(ConsistOf(
		cli.ScannedRoot{Path: syncedRoot, Resolved: syncedRoot, Scanned: true},
		cli.ScannedRoot{Path: bucket, Resolved: bucket, Scanned: true},
	))
}

func TestScanSyncedSkills_MissingRootIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := cli.ScanSyncedSkills(newFakeSkillFS(), syncedRoot)

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.Roots).To(Equal([]cli.ScannedRoot{{Path: syncedRoot}}))
}

func TestScanSyncedSkills_NoParsedManifestIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(syncedRoot+"/b/manifest.json", `nope`).
		file(syncedRoot+"/.bucket-b", "")

	result := cli.ScanSyncedSkills(fsys, syncedRoot)

	g.Expect(result.Candidates).To(BeEmpty())
	scanned, found := rootScanned(result, syncedRoot)
	g.Expect(found).To(BeTrue())
	g.Expect(scanned).To(BeFalse())
}

// TestScanSyncedSkills_Property: over any mix of buckets (parseable,
// unparseable or missing manifest) and stray files, candidates come only from
// parseable buckets, each bucket's scanned flag equals "manifest parsed", and
// the synced root is scanned iff at least one manifest parsed (model oracle:
// the generator's own bucket labels).
func TestScanSyncedSkills_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		fsys := newFakeSkillFS().dir(syncedRoot)
		buckets := rapid.SliceOfDistinct(rapid.StringMatching(`[a-f0-9]{4}`), rapid.ID[string]).Draw(rt, "buckets")
		wantSources := []string{}
		wantBucketScanned := map[string]bool{}
		anyParsed := false

		for _, bucket := range buckets {
			dir := syncedRoot + "/" + bucket
			fsys.file(dir+"/skill-"+bucket+"/SKILL.md", bucket)
			fsys.file(syncedRoot+"/.bucket-"+bucket, "")

			switch rapid.IntRange(0, manifestShapeCount-1).Draw(rt, "manifest") {
			case manifestShapeValid:
				fsys.file(dir+"/manifest.json", `{"skills":[]}`)
				wantSources = append(wantSources, dir+"/skill-"+bucket+"/SKILL.md")
				wantBucketScanned[dir] = true
				anyParsed = true
			case manifestShapeInvalid:
				fsys.file(dir+"/manifest.json", `{"skills":`)
				wantBucketScanned[dir] = false
			case manifestShapeMissing:
				wantBucketScanned[dir] = false
			}
		}

		result := cli.ScanSyncedSkills(fsys, syncedRoot)

		gotSources := make([]string, 0, len(result.Candidates))
		for _, candidate := range result.Candidates {
			gotSources = append(gotSources, candidate.SourcePath)
			g.Expect(candidate.ReadRoot).To(Equal(filepath.Dir(filepath.Dir(candidate.SourcePath))))
		}

		g.Expect(gotSources).To(ConsistOf(wantSources))

		gotBucketScanned := map[string]bool{}

		for _, root := range result.Roots {
			if root.Path == syncedRoot {
				g.Expect(root.Scanned).To(Equal(anyParsed))

				continue
			}

			gotBucketScanned[root.Path] = root.Scanned
		}

		g.Expect(gotBucketScanned).To(Equal(wantBucketScanned))
	})
}

// unexported constants.
const (
	childShapeBareDir     = 3
	childShapeCount       = 5
	childShapeDangling    = 2
	childShapeLinkedSkill = 1
	childShapeRootFile    = 4
	childShapeSkillDir    = 0
	manifestShapeCount    = 3
	manifestShapeInvalid  = 1
	manifestShapeMissing  = 2
	manifestShapeValid    = 0
	maxPropertyChain      = 5
	maxPropertyDepth      = 3
	maxPropertyFiles      = 8
	pluginRoot            = "/home/joe/.claude/plugins/cache/m/tools/1.0.0"
	syncedRoot            = userSkillsRoot + "/synced"
	userCommandsRoot      = "/home/joe/.claude/commands"
	userSkillsRoot        = "/home/joe/.claude/skills"
)

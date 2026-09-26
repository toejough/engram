package cli_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

func TestNormalizeProjectRemote_DistinctForgesStayDistinct(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	githubID, githubOK := cli.ExportNormalizeProjectRemote("ssh://git@GitHub.com:22/toejough/engram.git")
	gitlabID, gitlabOK := cli.ExportNormalizeProjectRemote("https://gitlab.com/toejough/engram")

	g.Expect(githubOK).To(BeTrue())
	g.Expect(gitlabOK).To(BeTrue())
	g.Expect(githubID).To(Equal("github.com/toejough/engram"))
	g.Expect(gitlabID).To(Equal("gitlab.com/toejough/engram"))
	g.Expect(githubID).NotTo(Equal(gitlabID))
}

func TestNormalizeProjectRemote_Examples(t *testing.T) {
	t.Parallel()

	table := []struct {
		name string
		url  string
		want string
		ok   bool
	}{
		{
			name: "spec ssh with port and case", url: "ssh://git@GitHub.com:22/Toejough/Engram.git",
			want: projectRemoteID, ok: true,
		},
		{name: "https", url: "https://github.com/toejough/engram", want: projectRemoteID, ok: true},
		{name: "scp", url: "git@github.com:toejough/engram.git", want: projectRemoteID, ok: true},
		{name: "scp with token userinfo", url: "user:t0k@github.com:toejough/engram", want: projectRemoteID, ok: true},
		{
			name: "https userinfo and trailing slash", url: "https://user:tok@github.com/toejough/engram.git/",
			want: projectRemoteID, ok: true,
		},
		{name: "surrounding whitespace", url: "  https://github.com/toejough/engram\n", want: projectRemoteID, ok: true},
		{
			name: "scp ssh host alias", url: "git@github-work:toejough/engram.git",
			want: "github-work/toejough/engram", ok: true,
		},
		{name: "scheme dotless host", url: "ssh://gitserver/team/repo", want: "gitserver/team/repo", ok: true},
		{name: "normalized form is a local path to git", url: projectRemoteID, ok: false},
		{name: "plain relative path", url: "mirrors/engram", ok: false},
		{name: "windows drive path", url: "C:/x/y", ok: false},
		{name: "gitlab subgroup", url: "https://gitlab.com/Group/Sub/Repo.git", want: "gitlab.com/group/sub/repo", ok: true},
		{name: "empty", url: "", ok: false},
		{name: "absolute local path", url: "/srv/git/engram.git", ok: false},
		{name: "relative local path", url: "../engram", ok: false},
		{name: "file url", url: "file:///srv/git/engram.git", ok: false},
		{name: "scp without path", url: "git@github.com:", ok: false},
		{name: "host only", url: "https://github.com/", ok: false},
		{name: "ipv6 host keeps a colon", url: "https://[::1]:8443/o/r", ok: false},
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got, ok := cli.ExportNormalizeProjectRemote(testCase.url)

			g.Expect(ok).To(Equal(testCase.ok))

			if testCase.ok {
				g.Expect(got).To(Equal(testCase.want))
			}
		})
	}
}

// TestNormalizeProjectRemote_VariantProperty: every scheme, userinfo, port,
// case and trailing-suffix variant of one remote yields the same
// `<host>/<path>` (the oracle builds it from the generated parts), and
// normalization is idempotent.
func TestNormalizeProjectRemote_VariantProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		host := rapid.StringMatching(`[a-z][a-z0-9-]{0,6}(\.[a-z]{2,4}){1,2}`).Draw(rt, "host")
		segments := rapid.SliceOfN(rapid.StringMatching(`[a-z0-9_][a-z0-9_.-]{0,6}[a-z0-9_]`), 1, 2).
			Draw(rt, "owners")
		repo := rapid.StringMatching(`[a-z0-9_][a-z0-9_-]{0,8}`).Draw(rt, "repo")
		path := strings.Join(append(segments, repo), "/")
		want := host + "/" + path

		url := drawRemoteVariant(rt, host, path)

		got, ok := cli.ExportNormalizeProjectRemote(url)
		g.Expect(ok).To(BeTrue(), url)
		g.Expect(got).To(Equal(want), url)

		// Idempotence over the canonical form re-expressed as a URL (a bare
		// `host/path` is a local path to git, so it is not a raw origin).
		again, againOK := cli.ExportNormalizeProjectRemote("ssh://" + got)
		g.Expect(againOK).To(BeTrue())
		g.Expect(again).To(Equal(got))
	})
}

func TestProbeProjectIdentity_LinkedWorktreeUsesOrigin(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ctx := context.Background()
	worktree := "/Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill"
	cwd := worktree + "/internal"
	mock, imp := MockCommander(t)
	done := make(chan probeOutcome)

	go func() {
		identity, found := cli.ExportProbeProjectIdentity(ctx, cwd, mock)
		done <- probeOutcome{identity: identity, found: found}
	}()

	imp.Run.ArgsEqual(ctx, cwd, "git", "rev-parse", "--show-toplevel").Return([]byte(worktree+"\n"), nil, nil)
	imp.Run.ArgsEqual(ctx, cwd, "git", "remote", "get-url", "origin").
		Return([]byte("ssh://git@github.com/toejough/engram.git\n"), nil, nil)

	outcome := <-done

	g.Expect(outcome.found).To(BeTrue())
	g.Expect(outcome.identity).To(Equal(cli.ProjectIdentity{ID: projectRemoteID, TopLevel: worktree}))
	g.Expect(outcome.identity.ScopeID()).To(Equal(projectScope))
}

func TestProbeProjectIdentity_NoOriginUsesCommonDirParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ctx := context.Background()
	worktree := "/Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill"
	mock, imp := MockCommander(t)
	done := make(chan probeOutcome)

	go func() {
		identity, found := cli.ExportProbeProjectIdentity(ctx, worktree, mock)
		done <- probeOutcome{identity: identity, found: found}
	}()

	imp.Run.ArgsEqual(ctx, worktree, "git", "rev-parse", "--show-toplevel").Return([]byte(worktree+"\n"), nil, nil)
	imp.Run.ArgsEqual(ctx, worktree, "git", "remote", "get-url", "origin").
		Return(nil, []byte("error: No such remote 'origin'"), errProbeGit)
	imp.Run.ArgsEqual(ctx, worktree, "git", "rev-parse", "--path-format=absolute", "--git-common-dir").
		Return([]byte("/Users/joe/repos/personal/engram/.git\n"), nil, nil)

	outcome := <-done

	g.Expect(outcome.found).To(BeTrue())
	g.Expect(outcome.identity).To(Equal(cli.ProjectIdentity{ID: "local/engram", TopLevel: worktree}))
}

func TestProbeProjectIdentity_NotARepoHasNoProject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ctx := context.Background()
	mock, imp := MockCommander(t)
	done := make(chan probeOutcome)

	go func() {
		identity, found := cli.ExportProbeProjectIdentity(ctx, "/tmp", mock)
		done <- probeOutcome{identity: identity, found: found}
	}()

	imp.Run.ArgsEqual(ctx, "/tmp", "git", "rev-parse", "--show-toplevel").
		Return(nil, []byte("fatal: not a git repository"), errProbeGit)

	outcome := <-done

	g.Expect(outcome.found).To(BeFalse())
	g.Expect(outcome.identity).To(Equal(cli.ProjectIdentity{}))
}

func TestProbeProjectIdentity_UnusableFallbacks(t *testing.T) {
	t.Parallel()

	table := []struct {
		name      string
		origin    string
		commonDir string
		commonErr error
		wantID    string
		wantFound bool
	}{
		{
			name: "local-path origin falls back", origin: "/srv/git/engram.git", commonDir: "/srv/engram/.git",
			wantID: "local/engram", wantFound: true,
		},
		{name: "empty origin falls back", origin: "  \n", commonDir: "/w/tool/.git", wantID: "local/tool", wantFound: true},
		{
			name: "relative-path origin falls back", origin: "mirrors/engram", commonDir: "/w/tool/.git",
			wantID: "local/tool", wantFound: true,
		},
		{name: "common dir fails", origin: "", commonErr: errProbeGit, wantFound: false},
		{name: "common dir empty", origin: "", commonDir: "\n", wantFound: false},
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			commander := scriptedGit{
				"rev-parse --show-toplevel": {out: "/w/top\n"},
				"remote get-url origin":     {out: testCase.origin},
				"rev-parse --path-format=absolute --git-common-dir": {
					out: testCase.commonDir, err: testCase.commonErr,
				},
			}

			identity, found := cli.ExportProbeProjectIdentity(context.Background(), "/w/top", commander)

			g.Expect(found).To(Equal(testCase.wantFound))

			if testCase.wantFound {
				g.Expect(identity).To(Equal(cli.ProjectIdentity{ID: testCase.wantID, TopLevel: "/w/top"}))
			}
		})
	}
}

// TestScanClaudeProjectSources_ChainProperty: for any repository depth, cwd
// level, and set of levels holding `.claude/skills`/`.claude/commands`
// (plus a same-named skill at several levels and a `.claude` dir above the
// top-level), the scan yields exactly the entries of the levels from cwd up
// to the top-level (the oracle), each existing directory is recorded as its
// own scanned root, and a level without the directory records no root.
func TestScanClaudeProjectSources_ChainProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		depth := rapid.IntRange(0, maxProjectChainDepth).Draw(rt, "depth")
		fsys := newFakeSkillFS().file("/work/.claude/skills/above/SKILL.md", "above")

		levels := make([]string, 0, depth+1)
		levels = append(levels, projectTop)

		for index := range depth {
			levels = append(levels, fmt.Sprintf("%s/l%d", levels[index], index))
		}

		fsys.dir(levels[len(levels)-1])

		wantIDs := make([]string, 0)
		wantRoots := make([]cli.ScannedRoot, 0)

		for index, level := range levels {
			if rapid.Bool().Draw(rt, fmt.Sprintf("skills-%d", index)) {
				skills := level + "/.claude/skills"
				fsys.file(skills+fmt.Sprintf("/s%d/SKILL.md", index), level).
					file(skills+"/shared/SKILL.md", "shared at "+level).
					file(skills+"/root.md", "root markdown is not a skill")
				wantIDs = append(wantIDs,
					fmt.Sprintf("skill s%d %s/s%d/SKILL.md", index, skills, index),
					fmt.Sprintf("skill shared %s/shared/SKILL.md", skills))
				wantRoots = append(wantRoots, cli.ScannedRoot{Path: skills, Resolved: skills, Scanned: true})
			}

			if rapid.Bool().Draw(rt, fmt.Sprintf("commands-%d", index)) {
				commands := level + "/.claude/commands"
				fsys.file(commands+fmt.Sprintf("/ns/c%d.md", index), level)
				wantIDs = append(wantIDs, fmt.Sprintf("command ns:c%d %s/ns/c%d.md", index, commands, index))
				wantRoots = append(wantRoots, cli.ScannedRoot{Path: commands, Resolved: commands, Scanned: true})
			}
		}

		result := cli.ScanClaudeProjectSources(fsys, claudeProjectScan(levels[len(levels)-1]))

		sort.Strings(wantIDs)
		g.Expect(projectCandidateIDs(result)).To(Equal(wantIDs))
		g.Expect(result.Roots).To(ConsistOf(wantRoots))
		g.Expect(result.Warnings).To(BeEmpty())

		for _, candidate := range result.Candidates {
			g.Expect(candidate.ScopeID).To(Equal(projectScope))
			g.Expect(candidate.SourceSegment).To(BeEmpty())
			g.Expect(candidate.ReadRoot).To(Equal(filepath.Dir(filepath.Dir(candidate.SourcePath))))
		}
	})
}

func TestScanClaudeProjectSources_CwdOutsideTopLevelScansNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file("/elsewhere/.claude/skills/x/SKILL.md", "x").
		file(projectTop+"/.claude/skills/y/SKILL.md", "y")

	scan := claudeProjectScan(projectTop)
	scan.Cwd = "/elsewhere"

	result := cli.ScanClaudeProjectSources(fsys, scan)

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.Roots).To(BeEmpty())
}

func TestScanClaudeProjectSources_SameNameAtTwoLevelsEmitsBoth(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(projectTop+"/.claude/skills/foo/SKILL.md", "top foo").
		file(projectTop+"/sub/.claude/skills/foo/SKILL.md", "sub foo")

	result := cli.ScanClaudeProjectSources(fsys, claudeProjectScan(projectTop+"/sub"))

	g.Expect(projectCandidateIDs(result)).To(Equal([]string{
		"skill foo " + projectTop + "/.claude/skills/foo/SKILL.md",
		"skill foo " + projectTop + "/sub/.claude/skills/foo/SKILL.md",
	}))
}

func TestScanClaudeProjectSources_SkipsUserRootsWhenTopLevelIsHome(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(userSkillsRoot+"/c4/SKILL.md", "user skill").
		file(fakeHome+"/.claude/commands/audit.md", "user command").
		file(fakeHome+"/proj/.claude/skills/p/SKILL.md", "project skill")

	scan := claudeProjectScan(fakeHome + "/proj")
	scan.TopLevel = fakeHome

	result := cli.ScanClaudeProjectSources(fsys, scan)

	g.Expect(candidateNames(result)).To(Equal([]string{"p"}))
	g.Expect(result.Roots).To(HaveLen(1))
}

func TestScanClaudeProjectSources_SubdirectoryEqualsTopLevel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(projectTop+"/.claude/skills/openspec-propose/SKILL.md", "propose").
		file(projectTop+"/.claude/skills/commit.md", "root file").
		file(projectTop+"/.claude/skills/engram-go-conventions.md", "root file").
		file(projectTop+"/.claude/commands/commit.md", "commit").
		file(projectTop+"/.claude/commands/opsx/apply.md", "apply").
		dir(projectTop + "/internal/cli")

	fromTop := cli.ScanClaudeProjectSources(fsys, claudeProjectScan(projectTop))
	fromSub := cli.ScanClaudeProjectSources(fsys, claudeProjectScan(projectTop+"/internal/cli"))

	g.Expect(fromSub).To(Equal(fromTop))
	g.Expect(projectCandidateIDs(fromTop)).To(Equal([]string{
		"command commit " + projectTop + "/.claude/commands/commit.md",
		"command opsx:apply " + projectTop + "/.claude/commands/opsx/apply.md",
		"skill openspec-propose " + projectTop + "/.claude/skills/openspec-propose/SKILL.md",
	}))
	g.Expect(fromTop.Roots).To(ConsistOf(
		cli.ScannedRoot{Path: projectTop + "/.claude/skills", Resolved: projectTop + "/.claude/skills", Scanned: true},
		cli.ScannedRoot{Path: projectTop + "/.claude/commands", Resolved: projectTop + "/.claude/commands", Scanned: true},
	))
}

func TestScanClaudeProjectSources_SymlinkedCwdMeetsTopLevel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(projectTop+"/.claude/skills/x/SKILL.md", "x").
		dir(projectTop+"/sub").
		link("/alias", projectTop)

	result := cli.ScanClaudeProjectSources(fsys, claudeProjectScan("/alias/sub"))

	g.Expect(candidateNames(result)).To(Equal([]string{"x"}))
}

func TestScanClaudeProjectSources_UnreadableLevelIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(projectTop+"/.claude/skills/top/SKILL.md", "top").
		file(projectTop+"/sub/.claude/skills/locked/SKILL.md", "locked").
		failRead(projectTop + "/sub/.claude/skills")

	result := cli.ScanClaudeProjectSources(fsys, claudeProjectScan(projectTop+"/sub"))

	g.Expect(candidateNames(result)).To(Equal([]string{"top"}))

	scanned, found := rootScanned(result, projectTop+"/sub/.claude/skills")
	g.Expect(found).To(BeTrue())
	g.Expect(scanned).To(BeFalse())
	g.Expect(warningsMention(result, projectTop+"/sub/.claude/skills")).To(BeTrue())

	topScanned, _ := rootScanned(result, projectTop+"/.claude/skills")
	g.Expect(topScanned).To(BeTrue())
}

// TestScanPiProjectScanners_StampSourceSegment (ruling R11): the Pi project
// scanners stamp the segment that tells their candidates apart inside one
// `project:<r>` scope — `pi`, `agents`, `pi-prompt` — mirroring the user
// scopes pi-user, agents-user and pi-prompt.
func TestScanPiProjectScanners_StampSourceSegment(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(projectTop+"/.pi/skills/foo/SKILL.md", "pi foo").
		file(projectTop+"/.agents/skills/foo/SKILL.md", "agents foo").
		file(projectTop+"/.pi/prompts/foo.md", "prompt foo")
	scan := cli.PiProjectScan{
		Cwd: projectTop, TopLevel: projectTop, UserAgentsRoot: agentsUserRoot, ScopeID: projectScope, Trusted: true,
	}

	segments := map[string][]string{}

	for _, result := range []cli.SkillScanResult{
		cli.ScanPiProjectSkills(fsys, scan),
		cli.ScanAgentsProjectSkills(fsys, scan),
		cli.ScanPiProjectPrompts(fsys, scan),
	} {
		for _, candidate := range result.Candidates {
			segments[candidate.SourceSegment] = append(segments[candidate.SourceSegment], candidate.Name)
		}
	}

	g.Expect(segments).To(Equal(map[string][]string{
		cli.SkillSegmentPi:     {"foo"},
		cli.SkillSegmentAgents: {"foo"},
		cli.SkillScopePiPrompt: {"foo"},
	}))
}

// unexported constants.
const (
	maxProjectChainDepth = 4
	projectRemoteID      = "github.com/toejough/engram"
	remoteSchemeCount    = 5
)

// unexported variables.
var (
	errProbeGit = errors.New("git: exit status 128")
)

type probeOutcome struct {
	identity cli.ProjectIdentity
	found    bool
}

type scriptedGit map[string]scriptedGitReply

func (s scriptedGit) Run(_ context.Context, _, _ string, args ...string) ([]byte, []byte, error) {
	reply, found := s[strings.Join(args, " ")]
	if !found {
		return nil, nil, errProbeGit
	}

	return []byte(reply.out), nil, reply.err
}

type scriptedGitReply struct {
	out string
	err error
}

func claudeProjectScan(cwd string) cli.ClaudeProjectScan {
	return cli.ClaudeProjectScan{
		Cwd:              cwd,
		TopLevel:         projectTop,
		UserSkillsRoot:   userSkillsRoot,
		UserCommandsRoot: fakeHome + "/.claude/commands",
		ScopeID:          projectScope,
	}
}

// drawRemoteVariant renders host/path as one of git's remote URL forms, with
// optional userinfo, port (scheme forms only), trailing `.git`/`/` and
// random letter case.
func drawRemoteVariant(rt *rapid.T, host, path string) string {
	userinfo := rapid.SampledFrom([]string{"", "git@", "user:t0ken@"}).Draw(rt, "userinfo")
	suffix := rapid.SampledFrom([]string{"", ".git", "/", ".git/"}).Draw(rt, "suffix")

	var url string

	switch form := rapid.IntRange(0, remoteSchemeCount).Draw(rt, "form"); form {
	case remoteSchemeCount:
		url = userinfo + host + ":" + path + suffix
	default:
		scheme := []string{"ssh", "https", "http", "git", "git+ssh"}[form]
		port := rapid.SampledFrom([]string{"", ":22", ":8443"}).Draw(rt, "port")
		url = scheme + "://" + userinfo + host + port + "/" + path + suffix
	}

	upper := rapid.SliceOfN(rapid.Bool(), len(url), len(url)).Draw(rt, "upper")
	runes := []rune(url)

	for index := range runes {
		if upper[index] {
			runes[index] = []rune(strings.ToUpper(string(runes[index])))[0]
		}
	}

	return string(runes)
}

// projectCandidateIDs renders each candidate as "<kind> <name> <source>",
// sorted.
func projectCandidateIDs(result cli.SkillScanResult) []string {
	ids := make([]string, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		ids = append(ids, fmt.Sprintf("%s %s %s", candidate.Kind, candidate.Name, candidate.SourcePath))
	}

	sort.Strings(ids)

	return ids
}

//go:generate impgen update.Commander --dependency --import-path github.com/toejough/engram/internal/update

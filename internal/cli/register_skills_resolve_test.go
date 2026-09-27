package cli_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestRegisterSkillsCLI_DefaultSetResolvesEveryFolder covers task 5.1 (design
// D1): with no --skills-dir, `engram register-skills` resolves the default
// source set from the home and the working directory — a real Claude user
// skill, a user command, engram's own deployed skill (reached through its
// harness symlink) and a project command in the cwd's repository all offer,
// with their resolved source paths.
func TestRegisterSkillsCLI_DefaultSetResolvesEveryFolder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newResolveCLIFixture(t, g)

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t, []string{
		"engram", "register-skills", "--dry-run", "--vault", fixture.vault,
	}, func(d *cli.Deps) {
		fixture.customize(d)
		d.Stdout = &stdout
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout.String()).To(Equal(fixture.expectedDryRun()))
}

// TestRegisterSkillsCLI_SymlinkedHomeRoundTripsSkillSource covers task 5.1's
// symlinked-home case (design D8): accepting offers under a home reached
// through a symlink records each note's skill_source `~`-relative, and a
// second run over the same vault offers nothing — no re-register, and no
// removal of the notes just written.
func TestRegisterSkillsCLI_SymlinkedHomeRoundTripsSkillSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newResolveCLIFixture(t, g)

	stderr := executeForTestWithDeps(t, []string{
		"engram", "register-skills", "--vault", fixture.vault, "--accept", "claude:curate", "--accept", "claude:c4",
		"--accept", "claude:cmd:audit", "--accept", "project:*",
	}, func(d *cli.Deps) {
		fixture.customize(d)
		d.Embed = skillAcceptFakeEmbedder{}
	})
	g.Expect(stderr).NotTo(ContainSubstring("register-skills:"))

	sources := map[string]string{}

	entries, readErr := os.ReadDir(fixture.vault)
	g.Expect(readErr).NotTo(HaveOccurred())

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		content, noteErr := os.ReadFile(filepath.Join(fixture.vault, entry.Name()))
		g.Expect(noteErr).NotTo(HaveOccurred())

		for line := range strings.SplitSeq(string(content), "\n") {
			if source, found := strings.CutPrefix(line, "skill_source: "); found {
				sources[entry.Name()[strings.Index(entry.Name(), ".skill-")+1:]] = source
			}
		}
	}

	g.Expect(sources).To(Equal(map[string]string{
		"skill-claude-curate.md":    "~/.claude/engram/skills/curate/SKILL.md",
		"skill-claude-c4.md":        "~/.claude/skills/c4/SKILL.md",
		"skill-claude-cmd-audit.md": "~/.claude/commands/audit.md",
		"skill-project-github-com-acme-widget-cmd-ship.md": filepath.Join(
			fixture.resolvedCwd, ".claude", "commands", "ship.md"),
	}))

	var stdout bytes.Buffer

	stderr = executeForTestWithDeps(t, []string{
		"engram", "register-skills", "--dry-run", "--vault", fixture.vault,
	}, func(d *cli.Deps) {
		fixture.customize(d)
		d.Stdout = &stdout
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout.String()).To(BeEmpty())
}

// TestUpdateSkillRegistration_RealRunPrintsScanWarnings covers the update
// hook printing the scanner and key-builder warnings on a real (non-dry)
// run: a `:` skill name is skipped with a warning, as under --dry-run.
func TestUpdateSkillRegistration_RealRunPrintsScanWarnings(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newResolveCLIFixture(t, g)

	badSkill := filepath.Join(fixture.resolvedHome, ".claude", "skills", "bad:name")
	g.Expect(os.MkdirAll(badSkill, 0o750)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(badSkill, "SKILL.md"), []byte("# Bad\n"), 0o600)).To(Succeed())

	deps := newTestDeps(&bytes.Buffer{}, &bytes.Buffer{})
	fixture.customize(&deps)

	var updateOut bytes.Buffer

	err := cli.ExportRunUpdateSkillRegistrationFromDeps(
		context.Background(), deps, false, fixture.vault, fixture.home, &updateOut)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(updateOut.String()).To(ContainSubstring(`engram: skipping skill "bad:name"`))
}

// TestUpdateSkillRegistration_SeesTheSameOffersAsRegisterSkills covers task
// 5.2 and update-deploy-sync's "Update and register-skills see the same
// offers": over one set of production-composed Deps (same home, vault and
// working directory), `engram update`'s registration hook and `engram
// register-skills --dry-run` list byte-identical offers.
func TestUpdateSkillRegistration_SeesTheSameOffersAsRegisterSkills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newResolveCLIFixture(t, g)

	var registerOut bytes.Buffer

	stderr := executeForTestWithDeps(t, []string{
		"engram", "register-skills", "--dry-run", "--vault", fixture.vault,
	}, func(d *cli.Deps) {
		fixture.customize(d)
		d.Stdout = &registerOut
	})
	g.Expect(stderr).To(BeEmpty())

	deps := newTestDeps(&bytes.Buffer{}, &bytes.Buffer{})
	fixture.customize(&deps)

	var updateOut bytes.Buffer

	err := cli.ExportRunUpdateSkillRegistrationFromDeps(
		context.Background(), deps, true, fixture.vault, fixture.home, &updateOut)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(registerOut.String()).To(Equal(fixture.expectedDryRun()))
	g.Expect(updateOut.String()).To(Equal(registerOut.String()))
}

// resolveCLIFixture is a real-filesystem home reached through a symlink, a
// vault, and a git working directory with an origin remote.
type resolveCLIFixture struct {
	home         string
	resolvedHome string
	resolvedCwd  string
	vault        string
}

// customize points d at the fixture's home and working directory.
func (f resolveCLIFixture) customize(d *cli.Deps) {
	d.UserHomeDir = func() (string, error) { return f.home, nil }
	d.Getwd = func() (string, error) { return f.resolvedCwd, nil }
}

// expectedDryRun is the dry-run preview of the fixture's four offers, one per
// scope, scopes in sorted order.
func (f resolveCLIFixture) expectedDryRun() string {
	return "@claude-cmd (1)\n" +
		"  would offer: register claude:cmd:audit (" +
		filepath.Join(f.resolvedHome, ".claude", "commands", "audit.md") + ")\n" +
		"@claude-user (2)\n" +
		"  would offer: register claude:c4 (" +
		filepath.Join(f.resolvedHome, ".claude", "skills", "c4", "SKILL.md") + ")\n" +
		"  would offer: register claude:curate (" +
		filepath.Join(f.resolvedHome, ".claude", "engram", "skills", "curate", "SKILL.md") + ")\n" +
		"@project:github.com/acme/widget (1)\n" +
		"  would offer: register project:github.com/acme/widget:cmd:ship (" +
		filepath.Join(f.resolvedCwd, ".claude", "commands", "ship.md") + ")\n"
}

// newResolveCLIFixture builds the fixture: under the real home, engram's
// deployed curate linked into ~/.claude/skills (as `engram update` deploys
// it), a real c4 user skill and an audit user command; the home itself is
// reached through a symlink. The working directory is a git repository
// whose origin is github.com/acme/widget, holding a project command.
func newResolveCLIFixture(t *testing.T, g Gomega) resolveCLIFixture {
	t.Helper()

	realHome, homeErr := filepath.EvalSymlinks(t.TempDir())
	g.Expect(homeErr).NotTo(HaveOccurred())

	home := filepath.Join(t.TempDir(), "home-link")
	g.Expect(os.Symlink(realHome, home)).To(Succeed())

	writeFixtureFile := func(path, content string) {
		g.Expect(os.MkdirAll(filepath.Dir(path), 0o750)).To(Succeed())
		g.Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())
	}

	engramCurate := filepath.Join(realHome, ".claude", "engram", "skills", "curate")
	writeFixtureFile(filepath.Join(engramCurate, "SKILL.md"), "---\nname: curate\n---\n\nJudge offers.\n")
	writeFixtureFile(filepath.Join(realHome, ".claude", "skills", "c4", "SKILL.md"), "---\nname: c4\n---\n\nDiagram.\n")
	writeFixtureFile(filepath.Join(realHome, ".claude", "commands", "audit.md"), "Audit the session.\n")
	g.Expect(os.Symlink(engramCurate, filepath.Join(realHome, ".claude", "skills", "curate"))).To(Succeed())

	cwd, cwdErr := filepath.EvalSymlinks(t.TempDir())
	g.Expect(cwdErr).NotTo(HaveOccurred())

	for _, args := range [][]string{
		{"init", "--quiet"},
		{"remote", "add", "origin", "git@github.com:acme/widget.git"},
	} {
		gitCmd := exec.CommandContext(context.Background(), "git", args...)
		gitCmd.Dir = cwd
		out, gitErr := gitCmd.CombinedOutput()
		g.Expect(gitErr).NotTo(HaveOccurred(), "git %v: %s", args, out)
	}

	writeFixtureFile(filepath.Join(cwd, ".claude", "commands", "ship.md"), "Ship it.\n")

	return resolveCLIFixture{home: home, resolvedHome: realHome, resolvedCwd: cwd, vault: t.TempDir()}
}

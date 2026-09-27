package cli_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestRegisterSkillsCLI_AcceptRegistersRealNote drives an --accept over the
// real filesystem end to end, exercising newSkillRegistrationDeps' full
// composition for the register path (Learn's capture pipeline, embed-on-
// write via a fake embedder).
func TestRegisterSkillsCLI_AcceptRegistersRealNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	home := t.TempDir()
	skillsDir := filepath.Join(home, ".claude", "engram", "skills")

	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "curate"), 0o750)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(skillsDir, "curate", "SKILL.md"),
		[]byte("---\nname: curate\ndescription: judge pending offers\n---\n\nbody\n"),
		0o600,
	)).To(Succeed())
	linkDeployedSkill(g, home, "curate")

	stderr := executeForTestWithDeps(t, []string{
		"engram", "register-skills", "--accept", "claude:curate", "--vault", vault,
	}, func(d *cli.Deps) {
		d.Embed = skillAcceptFakeEmbedder{}
		d.UserHomeDir = func() (string, error) { return home, nil }
		d.Getwd = func() (string, error) { return home, nil }
	})

	// A pending-offer nudge on the freshly-created note is expected
	// (vault-offer-curation) — only an actual command error would be a
	// problem here.
	g.Expect(stderr).NotTo(ContainSubstring("register-skills:"))

	entries, readErr := os.ReadDir(vault)
	g.Expect(readErr).NotTo(HaveOccurred())

	found := false

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".skill-claude-curate.md") {
			found = true
		}
	}

	g.Expect(found).To(BeTrue(), "expected a *.skill-claude-curate.md note in %v", entries)

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".skill-claude-curate.md") {
			continue
		}

		note, noteErr := os.ReadFile(filepath.Join(vault, entry.Name()))
		g.Expect(noteErr).NotTo(HaveOccurred())
		g.Expect(string(note)).To(ContainSubstring("skill_key: claude:curate\n"))
		g.Expect(string(note)).To(ContainSubstring("skill_source: ~/.claude/engram/skills/curate/SKILL.md\n"))
		g.Expect(string(note)).To(ContainSubstring("> Mirrors skill `agent-instructions/skills/curate/SKILL.md`"))
	}
}

// TestRegisterSkillsCLI_AdoptRealNote drives an --adopt over the real
// filesystem end to end, exercising newSkillRegistrationDeps' Adopt
// composition (Scan, Rename, embed-on-write via a fake embedder).
func TestRegisterSkillsCLI_AdoptRealNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	home := t.TempDir()
	skillsDir := filepath.Join(home, ".claude", "engram", "skills")

	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "curate"), 0o750)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(skillsDir, "curate", "SKILL.md"),
		[]byte("---\nname: curate\ndescription: judge pending offers\n---\n\n1. Judge offers.\n"),
		0o600,
	)).To(Succeed())
	linkDeployedSkill(g, home, "curate")

	oldBasename := "1049.2026-09-21.curate-review-pending-offers.md"
	g.Expect(os.WriteFile(
		filepath.Join(vault, oldBasename), []byte(curatePromotedNoteFixture()), 0o600,
	)).To(Succeed())

	stderr := executeForTestWithDeps(t, []string{
		"engram", "register-skills", "--adopt", "claude:curate=1049", "--vault", vault,
	}, func(d *cli.Deps) {
		d.Embed = skillAcceptFakeEmbedder{}
		d.UserHomeDir = func() (string, error) { return home, nil }
		d.Getwd = func() (string, error) { return home, nil }
	})

	g.Expect(stderr).To(BeEmpty())

	_, oldStillThere := os.Stat(filepath.Join(vault, oldBasename))
	g.Expect(oldStillThere).To(HaveOccurred())

	newContent, readErr := os.ReadFile(filepath.Join(vault, "1049.2026-09-21.skill-claude-curate.md"))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(newContent)).To(ContainSubstring("1. Judge offers."))
}

// TestRegisterSkillsCLI_DryRunListsRegisterOffer drives the real CLI wiring
// (registerSkillsTargets, newSkillRegistrationDeps) end to end over temp
// dirs: a shipped skill with no runbook note previews a register offer.
func TestRegisterSkillsCLI_DryRunListsRegisterOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	skillsDir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "curate"), 0o750)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(skillsDir, "curate", "SKILL.md"),
		[]byte("---\nname: curate\ndescription: judge pending offers\n---\n\nbody\n"),
		0o600,
	)).To(Succeed())

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t, []string{
		"engram", "register-skills", "--dry-run", "--vault", vault, "--skills-dir", skillsDir,
	}, func(d *cli.Deps) {
		d.Stdout = &stdout
	})

	resolvedSkillsDir, resolveErr := filepath.EvalSymlinks(skillsDir)
	g.Expect(resolveErr).NotTo(HaveOccurred())

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout.String()).To(Equal("@claude-user (1)\n  would offer: register claude:curate (" +
		filepath.Join(resolvedSkillsDir, "curate", "SKILL.md") + ")\n"))
}

// TestRegisterSkillsCLI_MalformedAdoptFlag covers --adopt values that aren't
// shaped "<name>=<note-ref>".
func TestRegisterSkillsCLI_MalformedAdoptFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	stderr := executeForTest(t, []string{
		"engram", "register-skills", "--dry-run", "--vault", vault, "--adopt", "not-a-pair",
	})

	g.Expect(stderr).To(ContainSubstring("<key>=<note-ref>"))
}

// TestRegisterSkillsCLI_NonTerminalStdinNeverPromptsOrWrites drives the real
// built binary (not the in-process targ wiring, which never exercises
// cmd/engram/main.go's IsTerminal primitive) with stdin redirected from
// /dev/null — a character device, exactly like an agent harness's `</dev/null`
// redirection, but not an interactive terminal. skill-runbook-registration's
// "Registration SHALL never prompt or write without a terminal" must hold
// here: no prompt text, just the one non-interactive summary line, and the
// vault stays untouched (regression coverage for the ModeCharDevice
// misdetection: /dev/null satisfies ModeCharDevice, so os.Stdin.Stat() alone
// can't tell it apart from a real tty).
func TestRegisterSkillsCLI_NonTerminalStdinNeverPromptsOrWrites(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	home := t.TempDir()
	binPath := sharedEngramBinary(t)

	claudeDir := filepath.Join(home, ".claude")
	g.Expect(os.MkdirAll(claudeDir, 0o750)).To(Succeed())
	g.Expect(os.Symlink(
		filepath.Join(projectRoot(t), "agent-instructions", "skills"), filepath.Join(claudeDir, "skills"),
	)).To(Succeed())

	devNull, openErr := os.Open(os.DevNull)
	g.Expect(openErr).NotTo(HaveOccurred())

	t.Cleanup(func() { _ = devNull.Close() })

	run := exec.Command(binPath, "register-skills", "--vault", vault)
	run.Stdin = devNull
	run.Dir = home
	run.Env = append(os.Environ(), "ENGRAM_PARENT=", "HOME="+home)

	out, runErr := run.CombinedOutput()
	g.Expect(runErr).NotTo(HaveOccurred(), "run failed: %s", out)

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	g.Expect(lines).To(HaveLen(1), "expected exactly one non-interactive summary line, got: %s", out)
	g.Expect(lines[0]).To(ContainSubstring("skill runbook offers awaiting an answer"))
	g.Expect(string(out)).NotTo(ContainSubstring("Register skill"))

	entries, readErr := os.ReadDir(vault)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).To(BeEmpty(), "expected vault to remain untouched, found: %v", entries)
}

// TestRegisterSkillsCLI_RefusesOverServer covers "Refuse when ENGRAM_SERVER
// is set (host-local only)", mirroring amend --discard's refusal.
func TestRegisterSkillsCLI_RefusesOverServer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stderr := executeForTestWithDeps(t, []string{"engram", "register-skills", "--dry-run"}, func(d *cli.Deps) {
		d.Getenv = func(key string) string {
			if key == "ENGRAM_SERVER" {
				return "http://example.invalid"
			}

			return ""
		}
	})

	g.Expect(stderr).To(ContainSubstring("host-local only"))
}

// TestRegisterSkillsCLI_RelativeSkillsDirNeedsWorkingDir: a relative
// --skills-dir with no resolvable working directory is an error, not a scan
// of some other directory.
func TestRegisterSkillsCLI_RelativeSkillsDirNeedsWorkingDir(t *testing.T) {
	t.Parallel()

	for name, getwd := range map[string]func() (string, error){
		"getwd fails": func() (string, error) { return "", os.ErrPermission },
		"no getwd":    nil,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			stderr := executeForTestWithDeps(t, []string{
				"engram", "register-skills", "--vault", t.TempDir(), "--skills-dir", "skills",
			}, func(d *cli.Deps) {
				d.Getwd = getwd
			})

			g.Expect(stderr).To(ContainSubstring("--skills-dir: resolving a relative dir"))
		})
	}
}

// TestRegisterSkillsCLI_SkillsDirIsRepeatablePreview covers `--skills-dir`
// (design D9): repeatable, relative dirs resolve against the working
// directory, and the run is a dry-run preview of `claude:` keys that writes
// nothing to the vault.
func TestRegisterSkillsCLI_SkillsDirIsRepeatablePreview(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	cwd, resolveErr := filepath.EvalSymlinks(t.TempDir())
	g.Expect(resolveErr).NotTo(HaveOccurred())

	for _, skill := range []string{"one/curate", "two/c4"} {
		g.Expect(os.MkdirAll(filepath.Join(cwd, skill), 0o750)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(cwd, skill, "SKILL.md"), []byte(skill), 0o600)).To(Succeed())
	}

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t, []string{
		"engram", "register-skills", "--vault", vault, "--skills-dir", "one", "--skills-dir", filepath.Join(cwd, "two"),
	}, func(d *cli.Deps) {
		d.Stdout = &stdout
		d.Getwd = func() (string, error) { return cwd, nil }
		d.IsTerminal = func() bool { return true }
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout.String()).To(Equal("@claude-user (2)\n" +
		"  would offer: register claude:c4 (" + filepath.Join(cwd, "two", "c4", "SKILL.md") + ")\n" +
		"  would offer: register claude:curate (" + filepath.Join(cwd, "one", "curate", "SKILL.md") + ")\n"))

	entries, readErr := os.ReadDir(vault)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}

// TestRegisterSkillsCLI_SkillsDirRefusesAccept covers "Skills-dir runs are
// preview-only": `--skills-dir agent-instructions/skills --accept claude:route` is
// refused with an error, and nothing is written.
func TestRegisterSkillsCLI_SkillsDirRefusesAccept(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	stderr := executeForTest(t, []string{
		"engram", "register-skills", "--vault", vault,
		"--skills-dir", filepath.Join(projectRoot(t), "agent-instructions", "skills"), "--accept", "claude:route",
	})

	g.Expect(stderr).To(ContainSubstring("--skills-dir is a read-only preview"))

	entries, readErr := os.ReadDir(vault)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}

// linkDeployedSkill links ~/.claude/skills/<name> to engram's deployed copy
// under ~/.claude/engram/skills, as `engram update` deploys it.
func linkDeployedSkill(g Gomega, home, name string) {
	userSkills := filepath.Join(home, ".claude", "skills")
	g.Expect(os.MkdirAll(userSkills, 0o750)).To(Succeed())
	g.Expect(os.Symlink(
		filepath.Join(home, ".claude", "engram", "skills", name), filepath.Join(userSkills, name),
	)).To(Succeed())
}

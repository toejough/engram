package cli_test

import (
	"bytes"
	"os"
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
	skillsDir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "curate"), 0o750)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(skillsDir, "curate", "SKILL.md"),
		[]byte("---\nname: curate\ndescription: judge pending offers\n---\n\nbody\n"),
		0o600,
	)).To(Succeed())

	stderr := executeForTestWithDeps(t, []string{
		"engram", "register-skills", "--accept", "curate", "--vault", vault, "--skills-dir", skillsDir,
	}, func(d *cli.Deps) {
		d.Embed = skillAcceptFakeEmbedder{}
	})

	// A pending-offer nudge on the freshly-created note is expected
	// (vault-offer-curation) — only an actual command error would be a
	// problem here.
	g.Expect(stderr).NotTo(ContainSubstring("register-skills:"))

	entries, readErr := os.ReadDir(vault)
	g.Expect(readErr).NotTo(HaveOccurred())

	found := false

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".skill-curate.md") {
			found = true
		}
	}

	g.Expect(found).To(BeTrue(), "expected a *.skill-curate.md note in %v", entries)
}

// TestRegisterSkillsCLI_AdoptRealNote drives an --adopt over the real
// filesystem end to end, exercising newSkillRegistrationDeps' Adopt
// composition (Scan, Rename, embed-on-write via a fake embedder).
func TestRegisterSkillsCLI_AdoptRealNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	skillsDir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "curate"), 0o750)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(skillsDir, "curate", "SKILL.md"),
		[]byte("---\nname: curate\ndescription: judge pending offers\n---\n\n1. Judge offers.\n"),
		0o600,
	)).To(Succeed())

	oldBasename := "1049.2026-09-21.curate-review-pending-offers.md"
	g.Expect(os.WriteFile(
		filepath.Join(vault, oldBasename), []byte(curatePromotedNoteFixture()), 0o600,
	)).To(Succeed())

	stderr := executeForTestWithDeps(t, []string{
		"engram", "register-skills", "--adopt", "curate=1049", "--vault", vault, "--skills-dir", skillsDir,
	}, func(d *cli.Deps) {
		d.Embed = skillAcceptFakeEmbedder{}
	})

	g.Expect(stderr).To(BeEmpty())

	_, oldStillThere := os.Stat(filepath.Join(vault, oldBasename))
	g.Expect(oldStillThere).To(HaveOccurred())

	newContent, readErr := os.ReadFile(filepath.Join(vault, "1049.2026-09-21.skill-curate.md"))
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

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout.String()).To(Equal("would offer: register curate\n"))
}

// TestRegisterSkillsCLI_MalformedAdoptFlag covers --adopt values that aren't
// shaped "<name>=<note-ref>".
func TestRegisterSkillsCLI_MalformedAdoptFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	skillsDir := t.TempDir()

	stderr := executeForTest(t, []string{
		"engram", "register-skills", "--dry-run",
		"--vault", vault, "--skills-dir", skillsDir, "--adopt", "not-a-pair",
	})

	g.Expect(stderr).To(ContainSubstring("<name>=<note-ref>"))
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

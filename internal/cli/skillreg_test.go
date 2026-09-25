package cli_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestCompareSkillOffers_ContentChangeAfterDecline_ReOffersExactlyOnce is
// the second half of tasks.md 1.4's property test: changing a skill's
// content re-yields exactly one offer for it (and none for any other
// skill, whose declines still hold).
func TestCompareSkillOffers_ContentChangeAfterDecline_ReOffersExactlyOnce(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		skills, names, vault := genSkillregFixture(rt)
		if len(skills) == 0 {
			return
		}

		baseline, err := cli.CompareSkillOffers("/vault", skills, names, vault.readFile, map[string]string{})
		if err != nil {
			rt.Fatalf("CompareSkillOffers: %v", err)
		}

		declined := make(map[string]string, len(baseline))
		for _, offer := range baseline {
			declined[offer.Skill] = offer.Hash
		}

		changedIdx := rapid.IntRange(0, len(skills)-1).Draw(rt, "changedIdx")
		extra := rapid.SliceOfN(rapid.Byte(), 1, 8).Draw(rt, "extraBytes")
		skills[changedIdx].Content = append(append([]byte{}, skills[changedIdx].Content...), extra...)

		offersAfter, afterErr := cli.CompareSkillOffers("/vault", skills, names, vault.readFile, declined)
		if afterErr != nil {
			rt.Fatalf("CompareSkillOffers (after content change): %v", afterErr)
		}

		matchCount := 0

		for _, offer := range offersAfter {
			if offer.Skill != skills[changedIdx].Name {
				rt.Fatalf("unexpected offer for unrelated skill %q: %+v", offer.Skill, offer)
			}

			matchCount++
		}

		if matchCount != 1 {
			rt.Fatalf("expected exactly one offer for changed skill %q, got %d: %+v",
				skills[changedIdx].Name, matchCount, offersAfter)
		}
	})
}

// TestCompareSkillOffers_DeclineAllThenRecompute_YieldsNoOffers is the
// property test tasks.md 1.4 asks for: after recording a decline for every
// offer CompareSkillOffers produces, recomputing over the same fixture
// yields zero offers.
func TestCompareSkillOffers_DeclineAllThenRecompute_YieldsNoOffers(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		skills, names, vault := genSkillregFixture(rt)

		offers, err := cli.CompareSkillOffers("/vault", skills, names, vault.readFile, map[string]string{})
		if err != nil {
			rt.Fatalf("CompareSkillOffers: %v", err)
		}

		declined := make(map[string]string, len(offers))
		for _, offer := range offers {
			declined[offer.Skill] = offer.Hash
		}

		offersAfter, afterErr := cli.CompareSkillOffers("/vault", skills, names, vault.readFile, declined)
		if afterErr != nil {
			rt.Fatalf("CompareSkillOffers (recompute): %v", afterErr)
		}

		if len(offersAfter) != 0 {
			rt.Fatalf("expected no offers after declining every prior offer, got %+v", offersAfter)
		}
	})
}

// TestCompareSkillOffers_DeclineSuppression covers per-kind decline
// suppression (skill-runbook-registration: "no offer SHALL be made ... when
// the skill's current hash is recorded as declined") and re-offering once
// the acting hash changes.
func TestCompareSkillOffers_DeclineSuppression(t *testing.T) {
	t.Parallel()

	t.Run("declined register is not re-offered at the same hash", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		content := []byte("curate procedure v1")
		hash := cli.SkillContentHash(content)
		vault := newSkillregFixtureVault()
		skills := []cli.ShippedSkill{{Name: "curate", Content: content}}

		offers, err := cli.CompareSkillOffers("/vault", skills, nil, vault.readFile, map[string]string{"curate": hash})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(BeEmpty())
	})

	t.Run("declined register is re-offered once content changes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		declined := map[string]string{"curate": cli.SkillContentHash([]byte("curate procedure v1"))}
		skills := []cli.ShippedSkill{{Name: "curate", Content: []byte("curate procedure v2")}}

		offers, err := cli.CompareSkillOffers("/vault", skills, nil, vault.readFile, declined)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(HaveLen(1))
		if len(offers) != 1 {
			return
		}

		g.Expect(offers[0].Kind).To(Equal(cli.SkillOfferRegister))
	})

	t.Run("declined refresh is not re-offered at the same hash", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		content := []byte("curate procedure v2")
		hash := cli.SkillContentHash(content)
		vault := newSkillregFixtureVault()
		vault.put("1049.2026-09-21.skill-curate.md", runbookNote("stale-hash"))
		skills := []cli.ShippedSkill{{Name: "curate", Content: content}}
		names := []string{"1049.2026-09-21.skill-curate.md"}

		offers, err := cli.CompareSkillOffers("/vault", skills, names, vault.readFile, map[string]string{"curate": hash})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(BeEmpty())
	})

	t.Run("declined removal is not re-offered at the same hash", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1053.2026-09-21.skill-write-memory.md", runbookNote("wm-hash"))
		names := []string{"1053.2026-09-21.skill-write-memory.md"}
		declined := map[string]string{"write-memory": "wm-hash"}

		offers, err := cli.CompareSkillOffers("/vault", nil, names, vault.readFile, declined)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(BeEmpty())
	})
}

// TestCompareSkillOffers_EachKind covers the four offer outcomes (skill-
// runbook-registration: "Registration SHALL offer, not act"): register when
// no note exists, refresh when the note's hash is stale, remove when a
// note's skill is no longer shipped, and no offer when hashes match.
func TestCompareSkillOffers_EachKind(t *testing.T) {
	t.Parallel()

	t.Run("register when no note exists", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		skills := []cli.ShippedSkill{{Name: "curate", Content: []byte("curate procedure v1")}}

		offers, err := cli.CompareSkillOffers("/vault", skills, nil, vault.readFile, map[string]string{})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(HaveLen(1))
		if len(offers) != 1 {
			return
		}

		g.Expect(offers[0].Kind).To(Equal(cli.SkillOfferRegister))
		g.Expect(offers[0].Skill).To(Equal("curate"))
		g.Expect(offers[0].Basename).To(BeEmpty())
		g.Expect(offers[0].Hash).To(Equal(cli.SkillContentHash([]byte("curate procedure v1"))))
	})

	t.Run("refresh when the note's hash is stale", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1049.2026-09-21.skill-curate.md", runbookNote("stale-hash"))
		skills := []cli.ShippedSkill{{Name: "curate", Content: []byte("curate procedure v2")}}
		names := []string{"1049.2026-09-21.skill-curate.md"}

		offers, err := cli.CompareSkillOffers("/vault", skills, names, vault.readFile, map[string]string{})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(HaveLen(1))
		if len(offers) != 1 {
			return
		}

		g.Expect(offers[0].Kind).To(Equal(cli.SkillOfferRefresh))
		g.Expect(offers[0].Skill).To(Equal("curate"))
		g.Expect(offers[0].Basename).To(Equal("1049.2026-09-21.skill-curate"))
		g.Expect(offers[0].Hash).To(Equal(cli.SkillContentHash([]byte("curate procedure v2"))))
	})

	t.Run("remove when a note's skill is no longer shipped", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1053.2026-09-21.skill-write-memory.md", runbookNote("wm-hash"))
		names := []string{"1053.2026-09-21.skill-write-memory.md"}

		offers, err := cli.CompareSkillOffers("/vault", nil, names, vault.readFile, map[string]string{})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(HaveLen(1))
		if len(offers) != 1 {
			return
		}

		g.Expect(offers[0].Kind).To(Equal(cli.SkillOfferRemove))
		g.Expect(offers[0].Skill).To(Equal("write-memory"))
		g.Expect(offers[0].Basename).To(Equal("1053.2026-09-21.skill-write-memory"))
		g.Expect(offers[0].Hash).To(Equal("wm-hash"))
	})

	t.Run("no offer when hashes match", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		content := []byte("curate procedure, unchanged")
		hash := cli.SkillContentHash(content)

		vault := newSkillregFixtureVault()
		vault.put("1049.2026-09-21.skill-curate.md", runbookNote(hash))
		skills := []cli.ShippedSkill{{Name: "curate", Content: content}}
		names := []string{"1049.2026-09-21.skill-curate.md"}

		offers, err := cli.CompareSkillOffers("/vault", skills, names, vault.readFile, map[string]string{})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(BeEmpty())
	})

	t.Run("duplicate skill note aborts the whole comparison", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1049.2026-09-21.skill-curate.md", runbookNote("abc"))
		vault.put("1050.2026-09-22.skill-curate.md", runbookNote("def"))
		skills := []cli.ShippedSkill{{Name: "curate", Content: []byte("v")}}
		names := []string{"1049.2026-09-21.skill-curate.md", "1050.2026-09-22.skill-curate.md"}

		offers, err := cli.CompareSkillOffers("/vault", skills, names, vault.readFile, map[string]string{})

		g.Expect(err).To(MatchError(cli.ErrDuplicateSkillNoteForTest))
		g.Expect(offers).To(BeNil())
	})
}

// TestFindSkillNote covers the lookup identity rule (skill-runbook-
// registration D3): none, one, duplicate, a non-runbook note with a
// matching slug, and a runbook note with the matching slug but no
// skill_hash.
func TestFindSkillNote(t *testing.T) {
	t.Parallel()

	t.Run("none", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1.2026-01-01.some-other-note.md", runbookNote("abc"))

		basename, hash, found, err := cli.FindSkillNote(
			"/vault", "curate", []string{"1.2026-01-01.some-other-note.md"}, vault.readFile,
		)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(found).To(BeFalse())
		g.Expect(basename).To(BeEmpty())
		g.Expect(hash).To(BeEmpty())
	})

	t.Run("one", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1049.2026-09-21.skill-curate.md", runbookNote("abc123"))

		basename, hash, found, err := cli.FindSkillNote(
			"/vault", "curate", []string{"1049.2026-09-21.skill-curate.md"}, vault.readFile,
		)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(found).To(BeTrue())
		g.Expect(basename).To(Equal("1049.2026-09-21.skill-curate"))
		g.Expect(hash).To(Equal("abc123"))
	})

	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1049.2026-09-21.skill-curate.md", runbookNote("abc"))
		vault.put("1050.2026-09-22.skill-curate.md", runbookNote("def"))

		_, _, found, err := cli.FindSkillNote(
			"/vault", "curate",
			[]string{"1049.2026-09-21.skill-curate.md", "1050.2026-09-22.skill-curate.md"},
			vault.readFile,
		)

		g.Expect(found).To(BeFalse())
		g.Expect(err).To(MatchError(cli.ErrDuplicateSkillNoteForTest))
		g.Expect(err).To(MatchError(ContainSubstring("1049.2026-09-21.skill-curate")))
		g.Expect(err).To(MatchError(ContainSubstring("1050.2026-09-22.skill-curate")))
	})

	t.Run("non-runbook note with a matching slug is not a skill note", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1.2026-01-01.skill-curate.md",
			"---\ntype: fact\nsituation: s\nsubject: s\npredicate: p\nobject: o\n---\n\nbody\n")

		_, _, found, err := cli.FindSkillNote(
			"/vault", "curate", []string{"1.2026-01-01.skill-curate.md"}, vault.readFile,
		)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(found).To(BeFalse())
	})

	t.Run("runbook note with a matching slug but no skill_hash is not a skill note", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1.2026-01-01.skill-curate.md", "---\ntype: runbook\nsituation: s\ndone_when: d\n---\n\nbody\n")

		_, _, found, err := cli.FindSkillNote(
			"/vault", "curate", []string{"1.2026-01-01.skill-curate.md"}, vault.readFile,
		)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(found).To(BeFalse())
	})
}

// TestFindSkillNote_IgnoresNonSkillOrMalformedEntries drives every early-out
// branch of the internal slug and frontmatter probes: a listed name with no
// .md suffix, one whose stem has no dot segment at all, one whose slug is
// exactly "skill-" (empty skill name), a "skill-curate" note with no
// frontmatter delimiter at all, and one with unparseable YAML frontmatter.
// None of these should ever be mistaken for curate's note.
func TestFindSkillNote_IgnoresNonSkillOrMalformedEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()
	vault.put("notitle.md", runbookNote("x"))
	vault.put("1.2026-01-01.skill-.md", runbookNote("y"))
	vault.put("1.2026-01-01.skill-curate.md", "not frontmatter at all\n")
	vault.put("2.2026-01-01.skill-curate.md", "---\ntype: [unterminated\n---\n\nbody\n")

	names := []string{
		"readme.txt", // never in files: skillNameFromNoteName must reject before any read
		"notitle.md",
		"1.2026-01-01.skill-.md",
		"1.2026-01-01.skill-curate.md",
		"2.2026-01-01.skill-curate.md",
	}

	_, _, found, err := cli.FindSkillNote("/vault", "curate", names, vault.readFile)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
}

// TestReadSkillRegistrations_AbsentFileIsEmptyState covers tasks.md 1.5's
// "Read tolerates an absent file (empty state)" scenario.
func TestReadSkillRegistrations_AbsentFileIsEmptyState(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()

	declined, err := cli.ReadSkillRegistrations("/vault", vault.readFile)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(declined).To(BeEmpty())
}

// TestReadSkillRegistrations_PropagatesOtherReadErrors distinguishes an
// absent file (tolerated) from any other read failure (surfaced) — a
// present-but-unreadable decline-state file must not be silently treated as
// "nothing declined".
func TestReadSkillRegistrations_PropagatesOtherReadErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	readFile := func(string) ([]byte, error) { return nil, errPermissionDeniedForTest }

	_, err := cli.ReadSkillRegistrations("/vault", readFile)

	g.Expect(err).To(HaveOccurred())
}

// TestReadSkillRegistrations_RejectsUnknownFields documents the tasks.md
// 1.5 "unknown JSON keys preserved or rejected" choice: this file has no
// third-party writer to stay forward-compatible with, so an unrecognized
// top-level field is rejected rather than silently dropped.
func TestReadSkillRegistrations_RejectsUnknownFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()
	vault.put("skill-registrations.json", `{"schema_version":1,"declined":{},"extra_field":true}`)

	_, err := cli.ReadSkillRegistrations("/vault", vault.readFile)

	g.Expect(err).To(HaveOccurred())
}

// TestRecordSkillDeclined_PropagatesReadError covers RecordSkillDeclined's
// read-failure path: a decline is never written when the existing state
// can't be loaded.
func TestRecordSkillDeclined_PropagatesReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	readFile := func(string) ([]byte, error) { return nil, errPermissionDeniedForTest }
	writeFile := func(string, []byte) error {
		g.Fail("writeFile must not be called when the read fails")

		return nil
	}

	err := cli.RecordSkillDeclined("/vault", "curate", "abc", readFile, writeFile)

	g.Expect(err).To(HaveOccurred())
}

// TestRecordSkillDeclined_PropagatesWriteError covers RecordSkillDeclined's
// write-failure path.
func TestRecordSkillDeclined_PropagatesWriteError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()
	writeFile := func(string, []byte) error { return errPermissionDeniedForTest }

	err := cli.RecordSkillDeclined("/vault", "curate", "abc", vault.readFile, writeFile)

	g.Expect(err).To(HaveOccurred())
}

// TestRecordSkillDeclined_SetsHashAndLeavesOtherEntriesIntact covers
// tasks.md 1.5's "decline write leaves other entries intact" scenario.
func TestRecordSkillDeclined_SetsHashAndLeavesOtherEntriesIntact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()
	vault.put("skill-registrations.json", `{"schema_version":1,"declined":{"route":"aaa"}}`)

	var (
		writtenPath string
		writtenData []byte
	)

	writeFile := func(path string, data []byte) error {
		writtenPath, writtenData = path, data

		return nil
	}

	err := cli.RecordSkillDeclined("/vault", "curate", "bbb", vault.readFile, writeFile)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(writtenPath).To(Equal("/vault/skill-registrations.json"))

	var doc struct {
		Declined map[string]string `json:"declined"`
	}

	g.Expect(json.Unmarshal(writtenData, &doc)).To(Succeed())
	g.Expect(doc.Declined).To(Equal(map[string]string{"route": "aaa", "curate": "bbb"}))
}

// TestRecordSkillDeclined_WritesFirstEntryWhenFileIsAbsent covers recording
// a decline when skill-registrations.json doesn't exist yet.
func TestRecordSkillDeclined_WritesFirstEntryWhenFileIsAbsent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()

	var writtenData []byte

	writeFile := func(_ string, data []byte) error {
		writtenData = data

		return nil
	}

	err := cli.RecordSkillDeclined("/vault", "curate", "abc", vault.readFile, writeFile)

	g.Expect(err).NotTo(HaveOccurred())

	var doc struct {
		Declined map[string]string `json:"declined"`
	}

	g.Expect(json.Unmarshal(writtenData, &doc)).To(Succeed())
	g.Expect(doc.Declined).To(Equal(map[string]string{"curate": "abc"}))
}

// unexported constants.
const (
	skillregFixtureVaultRoot = "/vault"
)

// unexported variables.
var (
	errPermissionDeniedForTest = errors.New("permission denied")
)

// unexported test helpers.

// skillregFixtureVault is a fake vault: full .md filenames plus a
// vault-joined-path→content map, backing readFile closures for the tests
// below without touching a real filesystem (DI everywhere). Every test
// fixture uses the same root (skillregFixtureVaultRoot) since the vault
// path itself is never under test.
type skillregFixtureVault struct {
	root  string
	files map[string]string
}

func (v *skillregFixtureVault) put(name, content string) {
	v.files[v.root+"/"+name] = content
}

func (v *skillregFixtureVault) readFile(path string) ([]byte, error) {
	content, ok := v.files[path]
	if !ok {
		return nil, fmt.Errorf("skillreg fixture: %w: %s", fs.ErrNotExist, path)
	}

	return []byte(content), nil
}

// genSkillregFixture draws a random set of shipped skills (1-4, unique
// names) and, for each, an independent note state: no note, a note whose
// skill_hash matches the skill's current content, or one whose skill_hash
// is stale. Returns the skills, the vault's full .md filename listing, and
// the backing fixture vault.
func genSkillregFixture(rt *rapid.T) ([]cli.ShippedSkill, []string, *skillregFixtureVault) {
	const (
		stateNoNote = iota
		stateMatching
		stateStale
	)

	count := rapid.IntRange(1, 4).Draw(rt, "skillCount")
	skills := make([]cli.ShippedSkill, count)
	vault := newSkillregFixtureVault()
	names := make([]string, 0, count)

	for i := range count {
		suffix := rapid.StringMatching(`[a-z0-9]{1,8}`).Draw(rt, fmt.Sprintf("skillNameSuffix%d", i))
		content := rapid.SliceOfN(rapid.Byte(), 1, 24).Draw(rt, fmt.Sprintf("content%d", i))
		name := fmt.Sprintf("skill%d-%s", i, suffix)
		skills[i] = cli.ShippedSkill{Name: name, Content: content}

		state := rapid.IntRange(stateNoNote, stateStale).Draw(rt, fmt.Sprintf("state%d", i))
		if state == stateNoNote {
			continue
		}

		hash := cli.SkillContentHash(content)
		if state == stateStale {
			hash = cli.SkillContentHash(append(append([]byte{}, content...), 0xAA))
		}

		noteName := fmt.Sprintf("%d.2026-01-01.skill-%s.md", i+1, name)
		vault.put(noteName, runbookNote(hash))
		names = append(names, noteName)
	}

	return skills, names, vault
}

func newSkillregFixtureVault() *skillregFixtureVault {
	return &skillregFixtureVault{root: skillregFixtureVaultRoot, files: map[string]string{}}
}

// runbookNote renders a minimal runbook note carrying skill_hash — just
// enough frontmatter for skillHashFromFrontmatter's probe to parse.
func runbookNote(hash string) string {
	return "---\ntype: runbook\nsituation: s\ndone_when: d\nskill_hash: \"" + hash + "\"\n---\n\nbody\n"
}

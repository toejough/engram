package cli_test

import (
	"bytes"
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

		baseline, err := compareShippedSkills(skills, names, vault.readFile, map[string]string{})
		if err != nil {
			rt.Fatalf("CompareSkillOffers: %v", err)
		}

		declined := make(map[string]string, len(baseline))
		for _, offer := range baseline {
			declined[offer.Key] = offer.Hash
		}

		changedIdx := rapid.IntRange(0, len(skills)-1).Draw(rt, "changedIdx")
		// A stale note's hash is content+staleSkillMarker, so appending exactly
		// that byte would make the note current (no offer, correctly).
		extra := rapid.SliceOfN(rapid.Byte(), 1, 8).
			Filter(func(extra []byte) bool { return !bytes.Equal(extra, []byte{staleSkillMarker}) }).
			Draw(rt, "extraBytes")
		skills[changedIdx].Content = append(append([]byte{}, skills[changedIdx].Content...), extra...)

		offersAfter, afterErr := compareShippedSkills(skills, names, vault.readFile, declined)
		if afterErr != nil {
			rt.Fatalf("CompareSkillOffers (after content change): %v", afterErr)
		}

		matchCount := 0

		for _, offer := range offersAfter {
			if offer.Key != "claude:"+skills[changedIdx].Name {
				rt.Fatalf("unexpected offer for unrelated skill %q: %+v", offer.Key, offer)
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

		offers, err := compareShippedSkills(skills, names, vault.readFile, map[string]string{})
		if err != nil {
			rt.Fatalf("CompareSkillOffers: %v", err)
		}

		declined := make(map[string]string, len(offers))
		for _, offer := range offers {
			declined[offer.Key] = offer.Hash
		}

		offersAfter, afterErr := compareShippedSkills(skills, names, vault.readFile, declined)
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
		skills := []engramSkill{{Name: "curate", Content: content}}

		offers, err := compareShippedSkills(skills, nil, vault.readFile, map[string]string{"claude:curate": hash})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(BeEmpty())
	})

	t.Run("declined register is re-offered once content changes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		declined := map[string]string{"claude:curate": cli.SkillContentHash([]byte("curate procedure v1"))}
		skills := []engramSkill{{Name: "curate", Content: []byte("curate procedure v2")}}

		offers, err := compareShippedSkills(skills, nil, vault.readFile, declined)

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
		vault.put("1049.2026-09-21.skill-curate.md", claudeSkillNote("curate", "stale-hash"))
		skills := []engramSkill{{Name: "curate", Content: content}}
		names := []string{"1049.2026-09-21.skill-curate.md"}

		offers, err := compareShippedSkills(skills, names, vault.readFile, map[string]string{"claude:curate": hash})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(BeEmpty())
	})

	t.Run("declined removal is not re-offered at the same hash", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1053.2026-09-21.skill-write-memory.md", claudeSkillNote("write-memory", "wm-hash"))
		names := []string{"1053.2026-09-21.skill-write-memory.md"}
		declined := map[string]string{"claude:write-memory": "wm-hash"}

		offers, err := compareShippedSkills(nil, names, vault.readFile, declined)

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
		skills := []engramSkill{{Name: "curate", Content: []byte("curate procedure v1")}}

		offers, err := compareShippedSkills(skills, nil, vault.readFile, map[string]string{})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(HaveLen(1))
		if len(offers) != 1 {
			return
		}

		g.Expect(offers[0].Kind).To(Equal(cli.SkillOfferRegister))
		g.Expect(offers[0].Key).To(Equal("claude:curate"))
		g.Expect(offers[0].Basename).To(BeEmpty())
		g.Expect(offers[0].Hash).To(Equal(cli.SkillContentHash([]byte("curate procedure v1"))))
	})

	t.Run("refresh when the note's hash is stale", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1049.2026-09-21.skill-curate.md", claudeSkillNote("curate", "stale-hash"))
		skills := []engramSkill{{Name: "curate", Content: []byte("curate procedure v2")}}
		names := []string{"1049.2026-09-21.skill-curate.md"}

		offers, err := compareShippedSkills(skills, names, vault.readFile, map[string]string{})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(HaveLen(1))
		if len(offers) != 1 {
			return
		}

		g.Expect(offers[0].Kind).To(Equal(cli.SkillOfferRefresh))
		g.Expect(offers[0].Key).To(Equal("claude:curate"))
		g.Expect(offers[0].Basename).To(Equal("1049.2026-09-21.skill-curate"))
		g.Expect(offers[0].Hash).To(Equal(cli.SkillContentHash([]byte("curate procedure v2"))))
	})

	t.Run("remove when a note's skill is no longer shipped", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1053.2026-09-21.skill-write-memory.md", claudeSkillNote("write-memory", "wm-hash"))
		names := []string{"1053.2026-09-21.skill-write-memory.md"}

		offers, err := compareShippedSkills(nil, names, vault.readFile, map[string]string{})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(HaveLen(1))
		if len(offers) != 1 {
			return
		}

		g.Expect(offers[0].Kind).To(Equal(cli.SkillOfferRemove))
		g.Expect(offers[0].Key).To(Equal("claude:write-memory"))
		g.Expect(offers[0].Basename).To(Equal("1053.2026-09-21.skill-write-memory"))
		g.Expect(offers[0].Hash).To(Equal("wm-hash"))
	})

	t.Run("no offer when hashes match", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		content := []byte("curate procedure, unchanged")
		hash := cli.SkillContentHash(content)

		vault := newSkillregFixtureVault()
		vault.put("1049.2026-09-21.skill-curate.md", claudeSkillNote("curate", hash))
		skills := []engramSkill{{Name: "curate", Content: content}}
		names := []string{"1049.2026-09-21.skill-curate.md"}

		offers, err := compareShippedSkills(skills, names, vault.readFile, map[string]string{})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(offers).To(BeEmpty())
	})

	t.Run("duplicate skill note aborts the whole comparison", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillregFixtureVault()
		vault.put("1049.2026-09-21.skill-curate.md", claudeSkillNote("curate", "abc"))
		vault.put("1050.2026-09-22.skill-curate.md", claudeSkillNote("curate", "def"))
		skills := []engramSkill{{Name: "curate", Content: []byte("v")}}
		names := []string{"1049.2026-09-21.skill-curate.md", "1050.2026-09-22.skill-curate.md"}

		offers, err := compareShippedSkills(skills, names, vault.readFile, map[string]string{})

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
			"/vault", "claude:curate", []string{"1.2026-01-01.some-other-note.md"}, vault.readFile,
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
		vault.put("1049.2026-09-21.skill-curate.md", claudeSkillNote("curate", "abc123"))

		basename, hash, found, err := cli.FindSkillNote(
			"/vault", "claude:curate", []string{"1049.2026-09-21.skill-curate.md"}, vault.readFile,
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
		vault.put("1049.2026-09-21.skill-curate.md", claudeSkillNote("curate", "abc"))
		vault.put("1050.2026-09-22.skill-curate.md", claudeSkillNote("curate", "def"))

		_, _, found, err := cli.FindSkillNote(
			"/vault", "claude:curate",
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
			"/vault", "claude:curate", []string{"1.2026-01-01.skill-curate.md"}, vault.readFile,
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
			"/vault", "claude:curate", []string{"1.2026-01-01.skill-curate.md"}, vault.readFile,
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
	vault.put("notitle.md", claudeSkillNote("curate", "x"))
	vault.put("1.2026-01-01.skill-.md", claudeSkillNote("curate", "y"))
	vault.put("1.2026-01-01.skill-curate.md", "not frontmatter at all\n")
	vault.put("2.2026-01-01.skill-curate.md", "---\ntype: [unterminated\n---\n\nbody\n")

	names := []string{
		"readme.txt", // never in files: skillNameFromNoteName must reject before any read
		"notitle.md",
		"1.2026-01-01.skill-.md",
		"1.2026-01-01.skill-curate.md",
		"2.2026-01-01.skill-curate.md",
	}

	_, _, found, err := cli.FindSkillNote("/vault", "claude:curate", names, vault.readFile)

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
	// staleSkillMarker is the byte genSkillregFixture appends to a skill's
	// content to give a stale note a hash that does not match.
	staleSkillMarker byte = 0xAA
)

// unexported variables.
var (
	errPermissionDeniedForTest = errors.New("permission denied")
)

// engramSkill is one of engram's own skills, as raw SKILL.md bytes.
type engramSkill struct {
	Name    string
	Content []byte
}

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

// claudeSkillNote renders a minimal skill note for Claude user skill name:
// a runbook carrying skill_hash and skill_key `claude:<name>`.
func claudeSkillNote(name, hash string) string {
	return keyedSkillNote(hash, "claude:"+name)
}

// compareShippedSkills runs CompareSkillOffers over engram's own skills
// (engramSkillSources over /skills) and returns just the offers.
func compareShippedSkills(
	skills []engramSkill, names []string, readFile func(string) ([]byte, error), declined map[string]string,
) ([]cli.SkillOffer, error) {
	comparison, err := cli.CompareSkillOffers(cli.SkillOfferInput{
		Vault:    skillregFixtureVaultRoot,
		Names:    names,
		ReadFile: readFile,
		Declined: declined,
		Sources:  engramSkillSources("/skills", skills),
	})
	if err != nil {
		return nil, err
	}

	return comparison.Offers, nil
}

// engramSkillSources presents skills as resolved sources read from one
// skills dir: Claude-user candidates keyed `claude:<name>`, and skillsDir
// recorded as the one read Claude-user root.
func engramSkillSources(skillsDir string, skills []engramSkill) cli.ResolvedSkillSources {
	var sources cli.ResolvedSkillSources

	sources.Roots = []cli.ScannedRoot{{
		Path: skillsDir, Resolved: skillsDir, Scanned: true, Form: cli.SkillRootFormClaudeUser,
	}}
	sources.Candidates = make([]cli.SkillCandidate, 0, len(skills))

	for _, skill := range skills {
		sources.Candidates = append(sources.Candidates, cli.SkillCandidate{
			Key:        "claude:" + skill.Name,
			Name:       skill.Name,
			ScopeID:    cli.SkillScopeClaudeUser,
			ReadRoot:   skillsDir,
			SourcePath: skillsDir + "/" + skill.Name + "/SKILL.md",
			Kind:       cli.SkillSourceKindSkill,
			Content:    skill.Content,
		})
	}

	return sources
}

// genSkillregFixture draws a random set of shipped skills (1-4, unique
// names and distinct contents) and, for each, an independent note state: no note, a note whose
// skill_hash matches the skill's current content, or one whose skill_hash
// is stale. Returns the skills, the vault's full .md filename listing, and
// the backing fixture vault.
func genSkillregFixture(rt *rapid.T) ([]engramSkill, []string, *skillregFixtureVault) {
	const (
		stateNoNote = iota
		stateMatching
		stateStale
	)

	count := rapid.IntRange(1, 4).Draw(rt, "skillCount")
	skills := make([]engramSkill, count)
	vault := newSkillregFixtureVault()
	names := make([]string, 0, count)

	for i := range count {
		suffix := rapid.StringMatching(`[a-z0-9]{1,8}`).Draw(rt, fmt.Sprintf("skillNameSuffix%d", i))
		name := fmt.Sprintf("skill%d-%s", i, suffix)
		// The name prefix keeps every skill's bytes distinct, so no skill is an
		// alias of another (design D4) and each is offered on its own.
		content := append([]byte(name+":"), rapid.SliceOfN(rapid.Byte(), 1, 24).Draw(rt, fmt.Sprintf("content%d", i))...)
		skills[i] = engramSkill{Name: name, Content: content}

		state := rapid.IntRange(stateNoNote, stateStale).Draw(rt, fmt.Sprintf("state%d", i))
		if state == stateNoNote {
			continue
		}

		hash := cli.SkillContentHash(content)
		if state == stateStale {
			hash = cli.SkillContentHash(append(append([]byte{}, content...), staleSkillMarker))
		}

		noteName := fmt.Sprintf("%d.2026-01-01.skill-%s.md", i+1, name)
		vault.put(noteName, claudeSkillNote(name, hash))
		names = append(names, noteName)
	}

	return skills, names, vault
}

func newSkillregFixtureVault() *skillregFixtureVault {
	return &skillregFixtureVault{root: skillregFixtureVaultRoot, files: map[string]string{}}
}

// runbookNote renders a minimal unkeyed runbook note carrying skill_hash
// and no skill_key — no skill note (design D3).
func runbookNote(hash string) string {
	return "---\ntype: runbook\nsituation: s\ndone_when: d\nskill_hash: \"" + hash + "\"\n---\n\nbody\n"
}

package cli_test

// Tests for openspec change register-skills-all-skill-folders tasks 8.3 and
// 8.4 (design D4, D5, D8): each key form has exactly one removal rule, which
// consults only that form's roots, and engram's installed skills are an
// ordinary source — a Claude and a Pi copy are two keys that alias only when
// byte-identical.

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestCompareSkillOffers_AnotherNamespacesMissingRootDoesNotBlockRemoval is
// the spec scenario "Another namespace's missing root does not block
// removal" (the old ruling R39 case), end to end through ResolveSkillSources:
// no ~/.agents/skills, a permission error listing ~/.pi/agent/skills, and a
// read ~/.claude/skills without c4 → the claude:c4 note is offered for
// removal.
func TestCompareSkillOffers_AnotherNamespacesMissingRootDoesNotBlockRemoval(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(userSkillsRoot+"/route/SKILL.md", "route").
		dir(piUserRoot).
		failRead(piUserRoot)

	resolved, err := cli.ResolveSkillSources(t.Context(), fakeHome, "/tmp",
		cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}})
	g.Expect(err).NotTo(HaveOccurred())

	read := map[cli.SkillRootForm]bool{}
	for _, root := range resolved.Roots {
		read[root.Form] = root.Scanned
	}

	g.Expect(read).To(HaveKeyWithValue(cli.SkillRootFormClaudeUser, true))
	g.Expect(read).To(HaveKeyWithValue(cli.SkillRootFormPiUser, false))
	g.Expect(read[cli.SkillRootFormAgentsUser]).To(BeFalse())

	fixture := newOfferFixture()
	fixture.home = fakeHome
	fixture.sources = resolved
	fixture.note("claude:c4", "h", "~/.claude/skills/c4/SKILL.md")

	g.Expect(offerScopes(fixture.compare(g))).To(Equal([]string{
		"register claude:route claude-user",
		"remove claude:c4 claude-user",
	}))
}

// TestCompareSkillOffers_InstalledEngramSkillsAreOrdinaryKeys covers design
// D3/D4's round-3 rule: engram's installed skills are keyed by their folder, with
// no collision rule and no diverged-copy exception.
func TestCompareSkillOffers_InstalledEngramSkillsAreOrdinaryKeys(t *testing.T) {
	t.Parallel()

	t.Run("a real claude route and an engram-linked pi route are two keys", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := newFakeSkillFS().
			file(userSkillsRoot+"/route/SKILL.md", "my route").
			file(engramPiSkills+"/route/SKILL.md", "engram route").
			link(piUserRoot+"/route", engramPiSkills+"/route")

		resolved, err := cli.ResolveSkillSources(t.Context(), fakeHome, "/tmp",
			cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}})
		g.Expect(err).NotTo(HaveOccurred())

		fixture := newOfferFixture()
		fixture.home = fakeHome
		fixture.sources = resolved

		comparison := fixture.compare(g)

		g.Expect(resolved.Warnings).To(BeEmpty())
		g.Expect(comparison.Conflicts).To(BeEmpty())
		g.Expect(offerSummaries(comparison)).To(ConsistOf(
			"register claude:route "+userSkillsRoot+"/route/SKILL.md",
			"register pi:route "+engramPiSkills+"/route/SKILL.md",
		))
	})

	t.Run("diverged claude and pi copies offer pi:route, with no conflict or warning", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		fixture.candidate(offerCand("claude:route", cli.SkillScopeClaudeUser,
			engramClaudeSkills+"/route/SKILL.md", "route v2"))
		fixture.candidate(offerCand("pi:route", cli.SkillScopePiUser, engramPiSkills+"/route/SKILL.md", "route v1"))
		fixture.note("claude:route", cli.SkillContentHash([]byte("route v2")), "")

		comparison := fixture.compare(g)

		g.Expect(comparison.Conflicts).To(BeEmpty())
		g.Expect(offerSummaries(comparison)).To(Equal([]string{
			"register pi:route " + engramPiSkills + "/route/SKILL.md",
		}))
	})

	t.Run("a byte-identical pi copy is an alias", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		fixture.candidate(offerCand("claude:route", cli.SkillScopeClaudeUser,
			engramClaudeSkills+"/route/SKILL.md", "route"))
		fixture.candidate(offerCand("pi:route", cli.SkillScopePiUser, engramPiSkills+"/route/SKILL.md", "route"))
		fixture.note("claude:route", cli.SkillContentHash([]byte("route")), "")

		comparison := fixture.compare(g)

		g.Expect(comparison.Conflicts).To(BeEmpty())
		g.Expect(comparison.Offers).To(BeEmpty())
	})
}

// TestCompareSkillOffers_UnkeyedLegacyNoteIsNeverOfferedForRemoval is the
// spec scenario "Unkeyed legacy note is never offered for removal": with
// every fixed root read and nothing scanned, runbook note 1036 carrying
// skill_hash and no skill_key is no removal candidate.
func TestCompareSkillOffers_UnkeyedLegacyNoteIsNeverOfferedForRemoval(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := everyRootReadFixture()

	comparison, err := compareWithUnkeyedNotes(fixture, map[string]string{
		legacyRouteNote: legacySkillNote("1036", "route", cli.SkillContentHash([]byte("route"))),
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(comparison.Offers).To(BeEmpty())
}

// TestSkillRemovalEligibility_FixedFormFollowsOnlyItsOwnRoots is 8.4's
// property (b), from design D5's rule for the single-root user forms: a note
// of fixed form F is eligible exactly when at least one root is recorded
// with form F and every root recorded with form F was read — whatever the
// roots of every other form, the note's source, or the plugin facts.
func TestSkillRemovalEligibility_FixedFormFollowsOnlyItsOwnRoots(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		fixture := drawNamespaceRoots(rt)
		name := rapid.SampledFrom(slices.Sorted(maps.Keys(removalFixedForms()))).Draw(rt, "form")
		spec := removalFixedForms()[name]
		key := spec.prefix + rapid.StringMatching(`[a-z][a-z0-9-]{0,5}`).Draw(rt, "name")

		recorded, allRead := false, true

		for _, root := range fixture.sources.Roots {
			if root.Form == spec.form {
				recorded = true
				allRead = allRead && root.Scanned
			}
		}

		got := cli.ExportSkillKeyRemovalEligible(fixture.input(), key, drawNoteSource(rt))
		if want := recorded && allRead; got != want {
			rt.Fatalf("%s note %q: eligible=%v, want %v (roots %+v)", name, key, got, want, fixture.sources.Roots)
		}
	})
}

// TestSkillRemovalEligibility_NamespaceIsolation is 8.4's property (a), a
// metamorphic relation from design D5 ("a note's eligibility depends only
// on the roots of its own key form"): flipping the read status of any root
// whose form differs from the note's key form never changes the note's
// eligibility, for every recognized key form (plugin keys, whose rule reads
// no root, included).
func TestSkillRemovalEligibility_NamespaceIsolation(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		fixture := drawNamespaceRoots(rt)
		key, recognized := drawSkillKeyForParsing(rt)

		// Half the notes take a fixed user form, whose rule reads roots of
		// its form alone, so a leak from another form's roots shows up often.
		if rapid.Bool().Draw(rt, "fixedFormNote") {
			spec := removalFixedForms()[rapid.SampledFrom(slices.Sorted(maps.Keys(removalFixedForms()))).Draw(rt, "form")]
			key, recognized = spec.prefix+"x", true
		}

		if !recognized {
			return
		}

		_, form, _, _, _ := cli.ExportParseSkillKey(key)
		source := drawNoteSource(rt)

		others := make([]int, 0, len(fixture.sources.Roots))

		for index, root := range fixture.sources.Roots {
			if root.Form != form {
				others = append(others, index)
			}
		}

		if len(others) == 0 {
			return
		}

		before := cli.ExportSkillKeyRemovalEligible(fixture.input(), key, source)

		flip := rapid.SampledFrom(others).Draw(rt, "flip")
		roots := slices.Clone(fixture.sources.Roots)
		roots[flip].Scanned = !roots[flip].Scanned
		fixture.sources.Roots = roots

		after := cli.ExportSkillKeyRemovalEligible(fixture.input(), key, source)
		if before != after {
			rt.Fatalf("note %q: flipping %s root %s changed eligibility %v → %v",
				key, roots[flip].Form, roots[flip].Path, before, after)
		}
	})
}

// drawNamespaceRoots draws, for every fixed user form, zero to two roots of
// that form with drawn read states, plus source-rooted roots holding the
// drawn note sources, and the plugin facts over a small plugin pool.
func drawNamespaceRoots(rt *rapid.T) *offerFixture {
	fixture := drawEligibilityUniverse(rt)
	fixture.sources.Roots = nil

	for _, name := range slices.Sorted(maps.Keys(removalFixedForms())) {
		spec := removalFixedForms()[name]

		count := rapid.IntRange(0, 2).Draw(rt, "fixedCount:"+name)
		for index := range count {
			fixture.root(fmt.Sprintf("%s/%d", spec.root, index), spec.form,
				rapid.Bool().Draw(rt, fmt.Sprintf("fixedRead:%s:%d", name, index)))
		}
	}

	for _, name := range slices.Sorted(maps.Keys(removalSourcedForms())) {
		spec := removalSourcedForms()[name]
		if rapid.Bool().Draw(rt, "sourcedRoot:"+name) {
			fixture.root("/r/own", spec.form, rapid.Bool().Draw(rt, "sourcedRead:"+name), spec.prefix)
		}
	}

	return fixture
}

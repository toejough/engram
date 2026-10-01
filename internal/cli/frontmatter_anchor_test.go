package cli_test

// Node edits refuse a note whose edited key carries an anchor
// (fix-show-amend-reparent-frontmatter review finding 1, ruling V3): replacing
// or deleting that value would drop the anchor and leave every alias to it
// dangling, so the written frontmatter would not decode.

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestAdoptSkillNote_RefusesAnchoredPending: adopt deletes pending:; an
// anchor on it refuses the adopt before any rename or write.
func TestAdoptSkillNote_RefusesAnchoredPending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := strings.Replace(curatePromotedNoteFixturePending(), "pending: true\n", "pending: &p true\nx_flag: *p\n", 1)

	after, err := runSkillAcceptMode(t.Context(), "adopt", before)
	g.Expect(err).To(MatchError(cli.ErrFrontmatterAnchoredKeyForTest))
	g.Expect(err).To(MatchError(ContainSubstring("pending")))
	g.Expect(after).To(BeEmpty(), "the note must not be renamed or written")
}

// TestRefreshSkill_RefusesAnchoredSkillKey: refresh sets skill_key:; an
// anchor on it refuses the refresh unwritten.
func TestRefreshSkill_RefusesAnchoredSkillKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := strings.Replace(curatePromotedNoteFixture(), "vault: personal\n",
		"vault: personal\nskill_key: &k \"claude:curate\"\nx_k: *k\n", 1)

	after, err := runSkillAcceptMode(t.Context(), "refresh", before)
	g.Expect(err).To(MatchError(cli.ErrFrontmatterAnchoredKeyForTest))
	g.Expect(err).To(MatchError(ContainSubstring("skill_key")))
	g.Expect(after).To(Equal(before), "the note must not be written")
}

// TestRunResituate_RefusesAnchoredEditedKey: situation: and created: are
// the keys resituate sets; an anchor on either refuses the note unwritten.
func TestRunResituate_RefusesAnchoredEditedKey(t *testing.T) {
	t.Parallel()

	for name, edit := range map[string][2]string{
		"situation": {"situation: working on it\n", "situation: &s working on it\n"},
		"created":   {"created: \"2026-01-01\"\n", "created: &c \"2026-01-01\"\n"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			alias := "x_ref: *" + strings.TrimPrefix(strings.Fields(edit[1])[1], "&") + "\n"
			before := strings.Replace(resituateUnknownKeysNote("fact", alias), edit[0], edit[1], 1)

			_, writes, err := runExchangeSurvivalResituate(t.Context(), before)
			g.Expect(err).To(MatchError(cli.ErrFrontmatterAnchoredKeyForTest))
			g.Expect(err).To(MatchError(ContainSubstring(name)))
			g.Expect(writes).To(BeZero())
		})
	}
}

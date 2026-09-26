package cli_test

import (
	"bytes"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestAcceptSkillOffer_MissedSourceErrorsAndWritesNothing: an accepted
// Register or Refresh offer whose key and source path name no candidate's
// note source is an error, never a note registered with an empty key or
// slug — and nothing is written to the vault.
func TestAcceptSkillOffer_MissedSourceErrorsAndWritesNothing(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		key := rapid.StringMatching(`[a-z][a-z0-9:-]{0,20}`).Draw(rt, "key")
		path := "/" + rapid.StringMatching(`[a-z0-9/]{1,30}`).Draw(rt, "path")
		kind := rapid.SampledFrom([]cli.SkillOfferKind{cli.SkillOfferRegister, cli.SkillOfferRefresh}).Draw(rt, "kind")

		// A source exists, but only for a different path of the same key.
		noteSources := map[string]cli.SkillNoteSource{}
		if rapid.Bool().Draw(rt, "decoy") {
			noteSources[key+"\x00"+path+"-other"] = cli.SkillNoteSource{Key: key, Name: key, Content: []byte("x")}
		}

		vault := newSkillAcceptFixtureVault()
		vault.put("1049.2026-09-21.skill-curate.md", curateSkillNoteFixture("old-hash"))

		offer := cli.SkillOffer{Kind: kind, Key: key, SourcePath: path, Basename: "1049.2026-09-21.skill-curate"}

		err := cli.ExportAcceptSkillOffer(t.Context(), "/vault", "personal", offer, noteSources,
			skillRegistrationDepsFor(vault, nil), &bytes.Buffer{})

		g.Expect(err).To(MatchError(cli.ErrSkillOfferSourceMissingForTest))
		g.Expect(vault.files).To(HaveLen(1))
		g.Expect(vault.files).To(HaveKeyWithValue("/vault/1049.2026-09-21.skill-curate.md",
			curateSkillNoteFixture("old-hash")))
	})
}

// TestAcceptSkillOffer_UnknownKindErrorsAndWritesNothing guards the
// dispatch's default: an offer kind CompareSkillOffers never emits is an
// error, and nothing is written.
func TestAcceptSkillOffer_UnknownKindErrorsAndWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	offer := cli.SkillOffer{Kind: cli.SkillOfferKind("rename"), Key: "curate", SourcePath: "/s/curate/SKILL.md"}
	noteSources := map[string]cli.SkillNoteSource{
		"curate\x00/s/curate/SKILL.md": {Key: "curate", Name: "curate", Content: []byte("x")},
	}

	err := cli.ExportAcceptSkillOffer(t.Context(), "/vault", "personal", offer, noteSources,
		skillRegistrationDepsFor(vault, nil), &bytes.Buffer{})

	g.Expect(err).To(MatchError(cli.ErrUnknownSkillOfferKindForTest))
	g.Expect(vault.files).To(BeEmpty())
}

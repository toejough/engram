package auto_test

import (
	"slices"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test793d25b1_AKindWithoutARowIsAnErrorNeverADefaultModel(t *testing.T) {
	t.Parallel()

	for _, kind := range tierKinds() {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			tiers := auto.DefaultTiers()
			delete(tiers, kind)

			request, err := tierBuild(tiers, kind)

			g.Expect(err).To(MatchError(ContainSubstring(`no judgement tier for kind "` + kind + `"`)))
			g.Expect(request.Model).To(BeEmpty())

			_, err = auto.Tiers{}.Request(kind, "a prompt")
			g.Expect(err).To(HaveOccurred())
		})
	}
}

func Test793d25b1_EveryRequestCarriesTheModelOfItsKindAndNotTheSessionModel(t *testing.T) {
	t.Parallel()

	for _, kind := range tierKinds() {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			request, err := tierBuild(auto.DefaultTiers(), kind)
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			g.Expect(request.Model).To(Equal(auto.DefaultTiers()[kind]))
			g.Expect(request.Model).To(Equal(tierDefaultModel))
			g.Expect(request.Model).NotTo(Equal(tierRaisedModel))
		})
	}
}

func Test793d25b1_EveryRequestRunsAtTheLowestAcceptedEffort(t *testing.T) {
	t.Parallel()

	for _, kind := range tierKinds() {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			request, err := tierBuild(auto.DefaultTiers(), kind)
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			g.Expect(request.Effort).To(Equal(tierLowestEffort))
			g.Expect(request.Prompt).NotTo(BeEmpty())
			g.Expect(request.MaxTokens).To(BeNumerically(">", 0))
			g.Expect(request.TimeoutMs).To(BeNumerically(">", 0))
		})
	}
}

func Test793d25b1_RaisingOneKindChangesOnlyTheRequestsOfThatKind(t *testing.T) {
	t.Parallel()

	for _, raised := range tierKinds() {
		t.Run(raised, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			tiers := auto.DefaultTiers()
			tiers[raised] = tierRaisedModel

			for _, kind := range tierKinds() {
				before, err := tierBuild(auto.DefaultTiers(), kind)
				g.Expect(err).NotTo(HaveOccurred())

				after, err := tierBuild(tiers, kind)
				g.Expect(err).NotTo(HaveOccurred())

				if kind == raised {
					g.Expect(after.Model).To(Equal(tierRaisedModel))

					before.Model = tierRaisedModel
				}

				g.Expect(after).To(Equal(before), kind)
			}
		})
	}
}

func Test793d25b1_TheDefaultTableHasARowForExactlyTheFourKinds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	keys := sortedTierKeys(auto.DefaultTiers())

	g.Expect(keys).To(Equal([]string{tierKindCapture, tierKindClassify, tierKindCuration, tierKindRelevance}))
}

// unexported constants.
const (
	tierDefaultModel  = "haiku"
	tierKindCapture   = "capture"
	tierKindClassify  = "classify"
	tierKindCuration  = "curation"
	tierKindRelevance = "relevance"
	tierLowestEffort  = "low"
	tierRaisedModel   = "sonnet"
)

func sortedTierKeys(tiers auto.Tiers) []string {
	keys := make([]string, 0, len(tiers))
	for key := range tiers {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	return keys
}

func tierBuild(tiers auto.Tiers, kind string) (auto.Request, error) {
	note := tierNote()

	switch kind {
	case tierKindClassify:
		return tiers.ClassifyWith("User: is it done?")
	case tierKindRelevance:
		return tiers.RelevanceWith("declaring work done", "I just finished", []auto.Note{note})
	case tierKindCapture:
		return tiers.CaptureWith("User: no, use the formatter", []auto.Note{note})
	default:
		return tiers.CurationWith(auto.Note{Situation: "renaming a package", Body: "run go build first"}, []auto.Note{note})
	}
}

func tierKinds() []string {
	return []string{tierKindClassify, tierKindRelevance, tierKindCapture, tierKindCuration}
}

func tierNote() auto.Note {
	return auto.Note{
		Path: "1.2026-10-12.note-n.md", Kind: "fact", Situation: "renaming a Go package", Body: "run go build first",
	}
}

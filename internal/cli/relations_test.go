package cli_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestMigrateRelationLinks_LeavesTranscriptLinksUntouched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	idToBasename := map[string]string{"105": "105.2026-05-30.foo"}
	// [[105]] in the transcript (above "Related to:") must NOT be rewritten.
	body := "verbatim transcript with [[105]] inside\n\nRelated to:\n- [[105]] — x.\n"

	got, n := cli.ExportMigrateRelationLinks(body, idToBasename)

	g.Expect(n).To(Equal(1)) // only the Related-section link
	g.Expect(got).To(ContainSubstring("verbatim transcript with [[105]] inside"))
	g.Expect(strings.Count(got, "[[105.2026-05-30.foo]]")).To(Equal(1))
}

func TestMigrateRelationLinks_NoRelatedSectionNoChange(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got, n := cli.ExportMigrateRelationLinks("just a body [[105]]\n", map[string]string{"105": "105.foo"})

	g.Expect(n).To(BeZero())
	g.Expect(got).To(Equal("just a body [[105]]\n"))
}

func TestMigrateRelationLinks_RewritesBareIDInRelatedSection(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	idToBasename := map[string]string{"105": "105.2026-05-30.foo"}
	body := "Information learned: when in X, S P O.\n\nRelated to:\n- [[105]] — because.\n"

	got, n := cli.ExportMigrateRelationLinks(body, idToBasename)

	g.Expect(n).To(Equal(1))
	g.Expect(got).To(ContainSubstring("[[105.2026-05-30.foo]]"))
	g.Expect(got).NotTo(ContainSubstring("[[105]] —"))
}

func TestResolveRelationTargetsStrict_AlreadyBasename_Passthrough(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "105.2026-01-01.thing.md"

	got, err := cli.ExportResolveRelationTargetsStrict([]string{basename + "|why"}, []string{basename})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal([]string{basename + "|why"}))
}

func TestResolveRelationTargetsStrict_RationalePreserved(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "105.2026-01-01.thing.md"

	got, err := cli.ExportResolveRelationTargetsStrict([]string{"105|because it matters"}, []string{basename})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal([]string{basename + "|because it matters"}))
}

func TestResolveRelationTargetsStrict_ResolvedID_OK(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "105.2026-01-01.thing.md"

	got, err := cli.ExportResolveRelationTargetsStrict([]string{"105|why"}, []string{basename})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal([]string{basename + "|why"}))
}

func TestResolveRelationTargetsStrict_UnresolvedID_Errors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportResolveRelationTargetsStrict([]string{"105|why"}, []string{"999.2026-01-01.other.md"})
	g.Expect(err).To(MatchError(ContainSubstring("unresolved relation target")))
	g.Expect(err).To(MatchError(ContainSubstring("105")))
}

func TestResolveRelationTargets_BareIDBecomesBasename(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	basenames := []string{"105.2026-05-30.foo", "1a.2026-05-30.bar"}

	got := cli.ExportResolveRelationTargets(
		[]string{"105|extracted from this chunk"},
		basenames,
	)

	g.Expect(got).To(Equal([]string{"105.2026-05-30.foo|extracted from this chunk"}))
}

func TestResolveRelationTargets_LeavesBasenameAndUnknownUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	basenames := []string{"105.2026-05-30.foo"}

	got := cli.ExportResolveRelationTargets(
		// already a basename · bare id with no matching note (dangling)
		[]string{"105.2026-05-30.foo|x", "999|y"},
		basenames,
	)

	g.Expect(got).To(Equal([]string{"105.2026-05-30.foo|x", "999|y"}))
}

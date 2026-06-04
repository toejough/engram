package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

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

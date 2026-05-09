package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestValidateSlug_AcceptsKebabCase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateSlug("graph-connectedness-recall-axis")).To(Succeed())
	g.Expect(cli.ExportValidateSlug("a")).To(Succeed())
	g.Expect(cli.ExportValidateSlug("note-1")).To(Succeed())
}

func TestValidateSlug_RejectsInvalid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateSlug("")).To(MatchError(ContainSubstring("slug")))
	g.Expect(cli.ExportValidateSlug("Has-Caps")).To(MatchError(ContainSubstring("slug")))
	g.Expect(cli.ExportValidateSlug("has space")).To(MatchError(ContainSubstring("slug")))
	g.Expect(cli.ExportValidateSlug("dot.in.it")).To(MatchError(ContainSubstring("slug")))
	g.Expect(cli.ExportValidateSlug("under_score")).To(MatchError(ContainSubstring("slug")))
}

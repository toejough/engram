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

func TestResolveVault_FlagWins(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	getenv := func(string) string { return "/from/env" }
	got, err := cli.ExportResolveVault("/from/flag", getenv)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("/from/flag"))
}

func TestResolveVault_FallsBackToEnv(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	getenv := func(name string) string {
		if name == "ENGRAM_VAULT_DIR" {
			return "/from/env"
		}
		return ""
	}
	got, err := cli.ExportResolveVault("", getenv)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(got).To(Equal("/from/env"))
}

func TestResolveVault_ErrorsWhenNeitherSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	getenv := func(string) string { return "" }
	_, err := cli.ExportResolveVault("", getenv)
	g.Expect(err).To(MatchError(ContainSubstring("vault")))
}

package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

func TestPrintLinkExamples_CapsAtMax(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	links := make([]vaultgraph.UnresolvedLink, 12)
	for i := range links {
		links[i] = vaultgraph.UnresolvedLink{Source: "s", Target: "t"}
	}

	var out bytes.Buffer

	cli.ExportPrintLinkExamples(&out, links)

	g.Expect(out.String()).To(ContainSubstring("and 2 more"))
}

func TestRunCheck_DanglingLinkIsWarnNotFail(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o700)).To(Succeed())
	// [[999]] targets no note → dangling (G3/WARN), not a FAIL.
	g.Expect(os.WriteFile(filepath.Join(vault, "Permanent", "1.2026-05-30.foo.md"),
		[]byte("---\ntype: fact\n---\nbody\n\nRelated to:\n- [[999]] — x.\n"), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunCheck(context.Background(), cli.CheckArgs{VaultPath: vault}, cli.ExportNewOsCheckDeps(), &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("WARN"))
	g.Expect(out.String()).To(ContainSubstring("PASS  G0"))
}

func TestRunCheck_FailsOnUnresolvedG0Links(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.CheckDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{
				// Bare-id link — resolves to no basename (the G0 bug).
				{Basename: "105.2026-05-30.foo", Outgoing: []string{"105"}},
			}, nil
		},
	}

	var out bytes.Buffer

	err := cli.RunCheck(context.Background(), cli.CheckArgs{VaultPath: "v"}, deps, &out)

	g.Expect(err).To(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("FAIL"))
	g.Expect(out.String()).To(ContainSubstring("G0"))
	g.Expect(out.String()).To(ContainSubstring("105"))
}

func TestRunCheck_PassesWhenAllLinksResolve(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.CheckDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{
				{Basename: "A", Outgoing: []string{"B"}},
				{Basename: "B"},
			}, nil
		},
	}

	var out bytes.Buffer

	err := cli.RunCheck(context.Background(), cli.CheckArgs{VaultPath: "v"}, deps, &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("PASS"))
	g.Expect(out.String()).To(ContainSubstring("G0"))
}

func TestRunCheck_RealDepsFlagBareIDLinks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o700)).To(Succeed())
	// Note 2 exists, so the bare-id link [[2]] *should* resolve but doesn't by
	// form — a G0 resolver failure (FAIL).
	g.Expect(os.WriteFile(filepath.Join(vault, "Permanent", "2.2026-05-30.bar.md"),
		[]byte("---\ntype: fact\n---\nbody\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(vault, "Permanent", "1.2026-05-30.foo.md"),
		[]byte("---\ntype: fact\n---\nbody\n\nRelated to:\n- [[2]] — x.\n"),
		0o600,
	)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunCheck(context.Background(), cli.CheckArgs{VaultPath: vault}, cli.ExportNewOsCheckDeps(), &out)

	g.Expect(err).To(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("FAIL"))
	g.Expect(out.String()).To(ContainSubstring("[[2]]"))
}

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

func TestRunCheck_RealDepsFlagBareIDLinks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o700)).To(Succeed())
	// Body links a bare id [[2]] — no note "2" exists, so it never resolves (G0).
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

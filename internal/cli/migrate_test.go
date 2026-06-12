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

func TestRunMigrateLinks_ApplyWritesResolvedLinks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	wrote := map[string][]byte{}

	var out bytes.Buffer

	err := cli.RunMigrateLinks(
		context.Background(),
		cli.MigrateArgs{VaultPath: "v", Apply: true},
		twoNoteMigrateDeps(wrote),
		&out,
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(wrote).To(HaveLen(1))

	for _, data := range wrote {
		g.Expect(string(data)).To(ContainSubstring("[[1.2026-05-30.foo]]"))
	}

	g.Expect(out.String()).To(ContainSubstring("applied: 1 notes, 1 links"))
}

func TestRunMigrateLinks_DryRunDoesNotWrite(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	wrote := map[string][]byte{}

	var out bytes.Buffer

	err := cli.RunMigrateLinks(context.Background(), cli.MigrateArgs{VaultPath: "v"}, twoNoteMigrateDeps(wrote), &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(wrote).To(BeEmpty())
	g.Expect(out.String()).To(ContainSubstring("would-rewrite"))
	g.Expect(out.String()).To(ContainSubstring("dry-run"))
}

func TestRunMigrateLinks_RealDepsRoundTrip(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o700)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "1.2026-05-30.foo.md"),
		[]byte("---\ntype: fact\n---\nbody\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "2.2026-05-30.bar.md"),
		[]byte("---\ntype: fact\n---\nbody\n\nRelated to:\n- [[1]] — x.\n"), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunMigrateLinks(
		context.Background(),
		cli.MigrateArgs{VaultPath: vault, Apply: true},
		cli.ExportNewOsMigrateDeps(),
		&out,
	)
	g.Expect(err).NotTo(HaveOccurred())

	got, rErr := os.ReadFile(filepath.Join(vault, "2.2026-05-30.bar.md"))
	g.Expect(rErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(ContainSubstring("[[1.2026-05-30.foo]]"))
}

func twoNoteMigrateDeps(wrote map[string][]byte) cli.MigrateDeps {
	return cli.MigrateDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{
				{Basename: "1.2026-05-30.foo"},
				{Basename: "2.2026-05-30.bar"},
			}, nil
		},
		Read: func(path string) ([]byte, error) {
			if filepath.Base(path) == "2.2026-05-30.bar.md" {
				return []byte("body\n\nRelated to:\n- [[1]] — x.\n"), nil
			}

			return []byte("body\n"), nil
		},
		Write: func(path string, data []byte) error { wrote[path] = data; return nil },
	}
}

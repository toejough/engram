package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestRunShow_EmptyRefErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	var out bytes.Buffer

	err := cli.RunShow(context.Background(),
		cli.ShowArgs{Ref: "  ", VaultPath: vault}, newShowDeps(memFS), &out)

	g.Expect(err).To(HaveOccurred())
}

func TestRunShow_ListsOutboundTargetsFenceAware(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantHub(t, memFS, vault)

	var out bytes.Buffer

	err := cli.RunShow(context.Background(),
		cli.ShowArgs{Ref: "1.hub", VaultPath: vault}, newShowDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	// Scope the fence assertion to the outbound-links section: the raw note
	// content legitimately contains the fenced [[9.fenced]] text, so we must
	// inspect only the section the parser produced, not the whole output.
	_, outbound, found := strings.Cut(out.String(), "# outbound links")
	g.Expect(found).To(BeTrue(), "outbound links section must be present")
	g.Expect(outbound).To(ContainSubstring("2.alpha"), "outbound target alpha must be listed")
	g.Expect(outbound).To(ContainSubstring("3.beta"), "outbound target beta must be listed")
	g.Expect(outbound).NotTo(ContainSubstring("9.fenced"),
		"a fenced wikilink must not be reported as an outbound target")
}

func TestRunShow_NotFoundErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantHub(t, memFS, vault)

	var out bytes.Buffer

	err := cli.RunShow(context.Background(),
		cli.ShowArgs{Ref: "99.nonexistent", VaultPath: vault}, newShowDeps(memFS), &out)

	g.Expect(err).To(MatchError(ContainSubstring("not found")))
}

func TestRunShow_OsDepsReadRealVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	perm := vault
	g.Expect(os.MkdirAll(perm, 0o755)).To(Succeed())

	body := "---\ntype: fact\ntier: L2\n---\nreal note body, see [[2.other]].\n"
	g.Expect(os.WriteFile(filepath.Join(perm, "1.real.md"), []byte(body), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(perm, "2.other.md"),
		[]byte("---\ntype: fact\n---\nother\n"), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunShow(context.Background(),
		cli.ShowArgs{Ref: "1.real", VaultPath: vault}, cli.ExportNewShowDeps(realFSForTest()), &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("real note body"))
	g.Expect(out.String()).To(ContainSubstring("2.other"))
}

func TestRunShow_ResolvesBareLuhmannID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantHub(t, memFS, vault)

	var out bytes.Buffer

	// `engram show 1` must resolve the bare Luhmann id to 1.hub even though a
	// bare id is not a resolvable wikilink target (it's a normalization
	// pre-step layered over the basename match).
	err := cli.RunShow(context.Background(),
		cli.ShowArgs{Ref: "1", VaultPath: vault}, newShowDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("hub body links to"))
}

func TestRunShow_ResolvesFullBasename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantHub(t, memFS, vault)

	var out bytes.Buffer

	err := cli.RunShow(context.Background(),
		cli.ShowArgs{Ref: "1.hub", VaultPath: vault}, newShowDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("hub body links to"))
	g.Expect(out.String()).To(ContainSubstring("tier: L3"))
}

func TestRunShow_ToleratesWikilinkBracketsAndMdSuffix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantHub(t, memFS, vault)

	for _, ref := range []string{"[[1.hub]]", "1.hub.md", "[[1.hub|the hub]]"} {
		var out bytes.Buffer

		err := cli.RunShow(context.Background(),
			cli.ShowArgs{Ref: ref, VaultPath: vault}, newShowDeps(memFS), &out)

		g.Expect(err).NotTo(HaveOccurred(), "ref %q should resolve", ref)
		g.Expect(out.String()).To(ContainSubstring("hub body links to"), "ref %q", ref)
	}
}

func newShowDeps(memFS *inMemoryFS) cli.ShowDeps {
	return cli.ShowDeps{Scan: memFS.Scan, Read: memFS.Read}
}

// plantHub seeds a small linked vault: hub -> {alpha, beta}, with a fenced
// wikilink that the fence-aware parser must ignore.
func plantHub(t *testing.T, memFS *inMemoryFS, vault string) {
	t.Helper()

	memFS.files[filepath.Join(vault, "1.hub.md")] =
		[]byte("---\ntype: fact\ntier: L3\n---\nhub body links to [[2.alpha]] and [[3.beta]].\n```\n[[9.fenced]]\n```\n")
	memFS.files[filepath.Join(vault, "2.alpha.md")] =
		[]byte("---\ntype: fact\ntier: L2\n---\nalpha body\n")
	memFS.files[filepath.Join(vault, "3.beta.md")] =
		[]byte("---\ntype: fact\ntier: L2\n---\nbeta body\n")
}

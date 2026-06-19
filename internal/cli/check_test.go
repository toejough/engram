package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
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

func TestPrintNoteExamples_CapsAtMax(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	names := make([]string, 12)
	for i := range names {
		names[i] = "n"
	}

	var out bytes.Buffer

	cli.ExportPrintNoteExamples(&out, names)

	g.Expect(out.String()).To(ContainSubstring("and 2 more"))
}

func TestRunCheck_DanglingLinkIsWarnNotFail(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o700)).To(Succeed())
	// [[999]] targets no note → dangling (G3/WARN), not a FAIL.
	g.Expect(os.WriteFile(filepath.Join(vault, "1.2026-05-30.foo.md"),
		[]byte("---\ntype: fact\nsituation: testing the checker\n---\nbody\n\nRelated to:\n- [[999]] — x.\n"),
		0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunCheck(context.Background(), cli.CheckArgs{VaultPath: vault}, cli.ExportNewOsCheckDeps(), &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("WARN"))
	g.Expect(out.String()).To(ContainSubstring("PASS  G0"))
}

func TestRunCheck_FailsOnMissingSituation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o700)).To(Succeed())
	// A fact note with no situation field → S1 FAIL (catches a pre-M5 note that
	// slipped in without one — no write-path test can find that).
	g.Expect(os.WriteFile(filepath.Join(vault, "1.2026-05-30.foo.md"),
		[]byte("---\ntype: fact\nsubject: x\n---\nbody\n"), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunCheck(context.Background(), cli.CheckArgs{VaultPath: vault}, cli.ExportNewOsCheckDeps(), &out)

	g.Expect(err).To(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("FAIL"))
	g.Expect(out.String()).To(ContainSubstring("M5 situation-presence"))
	g.Expect(out.String()).To(ContainSubstring("1.2026-05-30.foo"))
}

func TestRunCheck_FailsOnOldSchemaSidecar(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.CheckDeps{
		Scan:     func(string) ([]vaultgraph.Note, error) { return []vaultgraph.Note{{Basename: "1.fact"}}, nil },
		ReadNote: func(string) ([]byte, error) { return []byte("---\ntype: fact\nsituation: x\n---\n\nb\n"), nil },
		ReadSidecar: func(string) ([]byte, error) {
			return []byte(`{"embedding_model_id":"m@4","dims":4,"vector":[1,0,0,0],"content_hash":"sha256:x"}`), nil
		},
	}

	var out bytes.Buffer

	err := cli.RunCheck(context.Background(), cli.CheckArgs{VaultPath: "v"}, deps, &out)
	g.Expect(err).To(MatchError(cli.ErrCheckFailedForTest))
	g.Expect(out.String()).To(ContainSubstring("FAIL  S1"))
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

func TestRunCheck_PassesOnCurrentSidecar(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	good := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@4",
		Dims:             4,
		SituationVector:  []float32{1, 0, 0, 0},
		BodyVector:       []float32{1, 0, 0, 0},
		ContentHash:      "sha256:x",
	})
	deps := cli.CheckDeps{
		Scan:        func(string) ([]vaultgraph.Note, error) { return []vaultgraph.Note{{Basename: "1.fact"}}, nil },
		ReadNote:    func(string) ([]byte, error) { return []byte("---\ntype: fact\nsituation: x\n---\n\nb\n"), nil },
		ReadSidecar: func(string) ([]byte, error) { return good, nil },
	}

	var out bytes.Buffer

	g.Expect(cli.RunCheck(context.Background(), cli.CheckArgs{VaultPath: "v"}, deps, &out)).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("PASS  S1"))
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
	g.Expect(os.MkdirAll(vault, 0o700)).To(Succeed())
	// Note 2 exists, so the bare-id link [[2]] *should* resolve but doesn't by
	// form — a G0 resolver failure (FAIL).
	g.Expect(os.WriteFile(filepath.Join(vault, "2.2026-05-30.bar.md"),
		[]byte("---\ntype: fact\nsituation: x\n---\nbody\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(vault, "1.2026-05-30.foo.md"),
		[]byte("---\ntype: fact\nsituation: x\n---\nbody\n\nRelated to:\n- [[2]] — x.\n"),
		0o600,
	)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunCheck(context.Background(), cli.CheckArgs{VaultPath: vault}, cli.ExportNewOsCheckDeps(), &out)

	g.Expect(err).To(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("FAIL"))
	g.Expect(out.String()).To(ContainSubstring("[[2]]"))
}

func TestRunCheck_SituationPresenceSkipsNonBearingNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o700)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o700)).To(Succeed())
	// Situation-bearing note WITH a situation → not flagged.
	g.Expect(os.WriteFile(filepath.Join(vault, "1.2026-05-30.foo.md"),
		[]byte("---\ntype: fact\nsituation: writing the checker\n---\nbody\n"), 0o600)).To(Succeed())
	// Malformed frontmatter YAML → skipped (frontmatter does not parse).
	g.Expect(os.WriteFile(filepath.Join(vault, "2.2026-05-30.bad.md"),
		[]byte("---\ntype: fact\nsituation: [unterminated\n---\nbody\n"), 0o600)).To(Succeed())
	// No frontmatter block at all → skipped.
	g.Expect(os.WriteFile(filepath.Join(vault, "3.2026-05-30.raw.md"),
		[]byte("just prose, no frontmatter\n"), 0o600)).To(Succeed())
	// MOC note → not situation-bearing, skipped.
	g.Expect(os.WriteFile(filepath.Join(vault, "MOCs", "index.md"),
		[]byte("---\ntype: moc\n---\nmap of content\n"), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunCheck(context.Background(), cli.CheckArgs{VaultPath: vault}, cli.ExportNewOsCheckDeps(), &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("PASS  M5 situation-presence"))
}

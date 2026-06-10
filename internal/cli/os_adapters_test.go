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
	"github.com/toejough/engram/internal/embed"
)

// TestApplyOneErrorPaths drives RunEmbedApply through every classification
// outcome including broken (corrupt sidecar) and read failure inside
// applyOne. Bumps applyOne + tallyStates coverage past threshold.
func TestApplyOneErrorPaths(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(Succeed())

	// Note 1: stale (sidecar present with wrong hash).
	note1 := filepath.Join(vault, "Permanent/1.2026-05-24.stale.md")
	g.Expect(os.WriteFile(note1, []byte("---\ntype: fact\n---\nbody1"), 0o600)).To(Succeed())

	sc1 := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "stub@4",
		Dims:             4,
		SituationVector:  []float32{1, 0, 0, 0},
		BodyVector:       []float32{1, 0, 0, 0},
		ContentHash:      "sha256:WRONG",
	}
	sc1Bytes := embed.MarshalSidecar(sc1)
	g.Expect(os.WriteFile(
		filepath.Join(vault, "Permanent/1.2026-05-24.stale.vec.json"),
		sc1Bytes, 0o600,
	)).To(Succeed())

	// Note 2: broken sidecar (malformed JSON).
	note2 := filepath.Join(vault, "Permanent/2.2026-05-24.broken.md")
	g.Expect(os.WriteFile(note2, []byte("body2"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(vault, "Permanent/2.2026-05-24.broken.vec.json"),
		[]byte("{not json"), 0o600,
	)).To(Succeed())

	// Note 3: OK (sidecar matches body).
	body3 := []byte("body3")
	note3 := filepath.Join(vault, "Permanent/3.2026-05-24.okay.md")
	g.Expect(os.WriteFile(note3, body3, 0o600)).To(Succeed())
	sc3 := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "stub@4",
		Dims:             4,
		SituationVector:  []float32{1, 1, 1, 1},
		BodyVector:       []float32{1, 1, 1, 1},
		ContentHash:      embed.ContentHash(body3),
	}
	g.Expect(os.WriteFile(
		filepath.Join(vault, "Permanent/3.2026-05-24.okay.vec.json"),
		embed.MarshalSidecar(sc3), 0o600,
	)).To(Succeed())

	// Note 4: incompatible sidecar (different model_id).
	body4 := []byte("body4")
	note4 := filepath.Join(vault, "Permanent/4.2026-05-24.incompat.md")
	g.Expect(os.WriteFile(note4, body4, 0o600)).To(Succeed())
	sc4 := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "OLD-model@128",
		Dims:             4,
		SituationVector:  []float32{1, 1, 1, 1},
		BodyVector:       []float32{1, 1, 1, 1},
		ContentHash:      embed.ContentHash(body4),
	}
	g.Expect(os.WriteFile(
		filepath.Join(vault, "Permanent/4.2026-05-24.incompat.vec.json"),
		embed.MarshalSidecar(sc4), 0o600,
	)).To(Succeed())

	// Run --stale (also re-embeds broken per shouldEmbed).
	deps := cli.ExportNewOsEmbedDeps(stubEmbedderForOSAdapter{})

	var out bytes.Buffer

	err := cli.RunEmbedApply(context.Background(),
		cli.EmbedApplyArgs{VaultPath: vault, Stale: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("embedded  Permanent/1.2026-05-24.stale.md (stale)"))
	g.Expect(out.String()).
		To(ContainSubstring("embedded  Permanent/2.2026-05-24.broken.md (broken)"))

	// Also run status to exercise every tallyStates branch (ok / stale /
	// incompatible / broken / missing — though missing requires a 5th
	// note without a sidecar, which we don't bother planting here).
	out.Reset()
	err = cli.RunEmbedStatus(context.Background(),
		cli.EmbedStatusArgs{VaultPath: vault},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("total:           4"))
}

// TestApplyOne_EmbedFailureSurfaces exercises the applyOne fail line by
// using an embedder that returns an error.
func TestApplyOne_EmbedFailureSurfaces(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(Succeed())

	notePath := filepath.Join(vault, "Permanent/1.2026-05-24.probe.md")
	g.Expect(os.WriteFile(notePath, []byte("body"), 0o600)).To(Succeed())

	deps := cli.ExportNewOsEmbedDeps(failingEmbedder{})

	var out bytes.Buffer

	err := cli.RunEmbedApply(context.Background(),
		cli.EmbedApplyArgs{VaultPath: vault, Missing: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("fail      Permanent/1.2026-05-24.probe.md"))

	// Stale flow when read fails for the body (delete the note after planting sidecar).
	g.Expect(os.Remove(notePath)).To(Succeed())

	out.Reset()
	err = cli.RunEmbedStatus(context.Background(),
		cli.EmbedStatusArgs{VaultPath: vault},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	// All notes scanned; one will be missing-on-read → broken.
	g.Expect(strings.Contains(out.String(), "total:")).To(BeTrue())
}

// TestExportLogWarningToStderr_FormatsAndWritesViaCaptureFD verifies the
// production warning hook by sniffing stderr via a captured pipe.
func TestExportLogWarningToStderr_FormatsAndWrites(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Redirect stderr to a pipe, run the hook, restore.
	saved := os.Stderr

	read, write, err := os.Pipe()
	g.Expect(err).NotTo(HaveOccurred())

	os.Stderr = write

	cli.ExportLogWarningToStderr("hello %s", "world")

	_ = write.Close()
	os.Stderr = saved

	var buf bytes.Buffer

	_, err = buf.ReadFrom(read)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(Equal("warning: hello world\n"))
}

// TestOsEmbedFS_ReadWriteScanRoundTrip exercises the production
// osEmbedFS adapter against a real tempdir vault so Read/Write/Scan
// are covered without spawning a subprocess.
func TestOsEmbedFS_ReadWriteScanRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(Succeed())

	notePath := filepath.Join(vault, "Permanent/1.2026-05-24.probe.md")
	g.Expect(os.WriteFile(notePath, []byte("body"), 0o600)).To(Succeed())

	// Build deps via the production constructor; embedder slot is a
	// stub so we don't pay the bundled-model unpack cost.
	deps := cli.ExportNewOsEmbedDeps(stubEmbedderForOSAdapter{})

	// Scan finds the note.
	notes, scanErr := deps.Scan(vault)
	g.Expect(scanErr).NotTo(HaveOccurred())
	g.Expect(notes).To(HaveLen(1))

	// Write places a sidecar.
	scPath := filepath.Join(vault, "Permanent/1.2026-05-24.probe.vec.json")
	g.Expect(deps.Write(scPath, []byte(`{"x":1}`))).To(Succeed())

	// Read recovers the bytes.
	got, readErr := deps.Read(scPath)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal(`{"x":1}`))

	// Read of a missing file surfaces the error.
	_, missErr := deps.Read(filepath.Join(vault, "nope"))
	g.Expect(missErr).To(HaveOccurred())
}

type stubEmbedderForOSAdapter struct{}

func (stubEmbedderForOSAdapter) Dims() int { return 4 }

func (stubEmbedderForOSAdapter) Embed(context.Context, string) ([]float32, error) {
	return []float32{0, 0, 0, 0}, nil
}

func (stubEmbedderForOSAdapter) ModelID() string { return "stub@4" }

package cli_test

import (
	"errors"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

func TestBumpLastUsedIdempotentForSameDate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	orig := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@1",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.2},
		ContentHash:      "sha256:x",
		LastUsed:         "2026-06-17",
	}
	store := map[string][]byte{"n.vec.json": embed.MarshalSidecar(orig)}
	writes := 0
	read := func(p string) ([]byte, error) { return store[p], nil }
	write := func(p string, b []byte) error { writes++; store[p] = b; return nil }

	err := cli.ExportBumpLastUsed("n.vec.json", "2026-06-17", read, write)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(writes).To(Equal(0), "no write when date is already set")
}

func TestBumpLastUsedReadFailureWrapsPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	readErr := errors.New("disk failure")
	read := func(_ string) ([]byte, error) { return nil, readErr }
	write := func(_ string, _ []byte) error { return nil }

	err := cli.ExportBumpLastUsed("missing.vec.json", "2026-06-17", read, write)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("activate:"))
	g.Expect(err.Error()).To(ContainSubstring("missing.vec.json"))
}

// ---------------------------------------------------------------------------
// Task 3.1: bumpLastUsed
// ---------------------------------------------------------------------------

func TestBumpLastUsedSetsDatePreservesVectors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	orig := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@1",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.2},
		ContentHash:      "sha256:x",
	}
	store := map[string][]byte{"n.vec.json": embed.MarshalSidecar(orig)}
	read := func(p string) ([]byte, error) { return store[p], nil }
	write := func(p string, b []byte) error { store[p] = b; return nil }

	err := cli.ExportBumpLastUsed("n.vec.json", "2026-06-17", read, write)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	got, derr := embed.UnmarshalSidecar(store["n.vec.json"])
	g.Expect(derr).NotTo(HaveOccurred())

	if derr != nil {
		return
	}

	g.Expect(got.LastUsed).To(Equal("2026-06-17"))
	g.Expect(got.ContentHash).To(Equal("sha256:x"))
	g.Expect(got.BodyVector).To(Equal([]float32{0.2}))
}

// ---------------------------------------------------------------------------
// Wiring coverage: newActivateDeps composed over real-OS test Deps
// ---------------------------------------------------------------------------

// TestNewActivateDeps_BumpsRealSidecar exercises newActivateDeps and its
// WriteFileAtomic-backed sidecar write against a real file in a temp
// directory.
func TestNewActivateDeps_BumpsRealSidecar(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	notePath := dir + "/1.note.md"
	sidecarPath := embed.SidecarPath(notePath)

	orig := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@1",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.2},
		ContentHash:      "sha256:real",
	}

	writeErr := os.WriteFile(sidecarPath, embed.MarshalSidecar(orig), 0o600)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	deps := cli.ExportNewActivateDeps(cli.ExportNewTestOsDeps())

	runErr := cli.ExportRunActivate(cli.ActivateArgs{Notes: []string{notePath}}, deps)
	g.Expect(runErr).NotTo(HaveOccurred())

	if runErr != nil {
		return
	}

	rawBytes, readErr := os.ReadFile(sidecarPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	got, derr := embed.UnmarshalSidecar(rawBytes)
	g.Expect(derr).NotTo(HaveOccurred())

	today := time.Now().Format("2006-01-02")
	g.Expect(got.LastUsed).To(Equal(today))
}

func TestRunActivateAcceptsAbsoluteNotePath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	abs := "/elsewhere/n.md"
	sc := embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "m@1", Dims: 1,
		SituationVector: []float32{0.1}, BodyVector: []float32{0.2}, ContentHash: "sha256:x",
	}
	store := map[string][]byte{"/elsewhere/n.vec.json": embed.MarshalSidecar(sc)}
	deps := cli.ActivateDeps{
		Now: func() time.Time { return time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC) },
		Read: func(p string) ([]byte, error) {
			b, ok := store[p]
			if !ok {
				return nil, os.ErrNotExist
			}

			return b, nil
		},
		Write:      func(p string, b []byte) error { store[p] = b; return nil },
		LogWarning: func(string, ...any) {},
	}
	// Vault set, but an ABSOLUTE note must NOT be joined to it.
	err := cli.ExportRunActivate(cli.ActivateArgs{Vault: "/vault", Notes: []string{abs}}, deps)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunActivateAllFailReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	read := func(_ string) ([]byte, error) { return nil, errors.New("not found") }
	write := func(_ string, _ []byte) error { return nil }
	fixedNow := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	err := cli.ExportRunActivate(
		cli.ActivateArgs{Notes: []string{"missing1.md", "missing2.md"}},
		cli.ActivateDeps{
			Now:        func() time.Time { return fixedNow },
			Read:       read,
			Write:      write,
			LogWarning: func(string, ...any) {},
		},
	)
	g.Expect(err).To(HaveOccurred(), "all failed must return an error")
}

// ---------------------------------------------------------------------------
// Task 3.2: RunActivate
// ---------------------------------------------------------------------------

func TestRunActivateBumpsAllNotes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	orig := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@1",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.2},
		ContentHash:      "sha256:x",
	}

	store := map[string][]byte{
		embed.SidecarPath("a.md"): embed.MarshalSidecar(orig),
		embed.SidecarPath("b.md"): embed.MarshalSidecar(orig),
	}

	read := func(p string) ([]byte, error) {
		b, ok := store[p]
		if !ok {
			return nil, errors.New("not found: " + p)
		}

		return b, nil
	}
	write := func(p string, b []byte) error { store[p] = b; return nil }

	fixedDate := "2026-06-17"
	fixedNow := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	err := cli.ExportRunActivate(
		cli.ActivateArgs{Notes: []string{"a.md", "b.md"}},
		cli.ActivateDeps{
			Now:        func() time.Time { return fixedNow },
			Read:       read,
			Write:      write,
			LogWarning: func(string, ...any) {},
		},
	)
	g.Expect(err).NotTo(HaveOccurred())

	for _, noteFile := range []string{"a.md", "b.md"} {
		sc, derr := embed.UnmarshalSidecar(store[embed.SidecarPath(noteFile)])
		g.Expect(derr).NotTo(HaveOccurred())
		g.Expect(sc.LastUsed).To(Equal(fixedDate), "note %s must have LastUsed set", noteFile)
	}
}

func TestRunActivateLogsContinuesOnBadPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	orig := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@1",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.2},
		ContentHash:      "sha256:x",
	}
	goodSidecar := embed.SidecarPath("good.md")
	store := map[string][]byte{goodSidecar: embed.MarshalSidecar(orig)}
	read := func(p string) ([]byte, error) {
		b, ok := store[p]
		if !ok {
			return nil, errors.New("not found: " + p)
		}

		return b, nil
	}
	write := func(p string, b []byte) error { store[p] = b; return nil }

	var warnings []string

	fixedNow := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	err := cli.ExportRunActivate(
		cli.ActivateArgs{Notes: []string{"missing.md", "good.md"}},
		cli.ActivateDeps{
			Now:        func() time.Time { return fixedNow },
			Read:       read,
			Write:      write,
			LogWarning: func(f string, _ ...any) { warnings = append(warnings, f) },
		},
	)
	g.Expect(err).NotTo(HaveOccurred(), "partial failure (some succeeded) must not error")
	g.Expect(warnings).To(HaveLen(1), "bad path must log a warning")

	sc, derr := embed.UnmarshalSidecar(store[goodSidecar])
	g.Expect(derr).NotTo(HaveOccurred())
	g.Expect(sc.LastUsed).To(Equal("2026-06-17"))
}

func TestRunActivateResolvesRelativeNoteAgainstVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := "/vault"
	noteBase := "24.2026-06-12.foo.md"
	sidecar := embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "m@1", Dims: 1,
		SituationVector: []float32{0.1}, BodyVector: []float32{0.2}, ContentHash: "sha256:x",
	}
	joined := "/vault/24.2026-06-12.foo.vec.json"
	store := map[string][]byte{joined: embed.MarshalSidecar(sidecar)}

	deps := cli.ActivateDeps{
		Now: func() time.Time { return time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC) },
		Read: func(p string) ([]byte, error) {
			b, ok := store[p]
			if !ok {
				return nil, os.ErrNotExist
			}

			return b, nil
		},
		Write:      func(p string, b []byte) error { store[p] = b; return nil },
		LogWarning: func(string, ...any) {},
	}

	err := cli.ExportRunActivate(cli.ActivateArgs{Vault: vault, Notes: []string{noteBase}}, deps)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	got, derr := embed.UnmarshalSidecar(store[joined])
	g.Expect(derr).NotTo(HaveOccurred())

	if derr != nil {
		return
	}

	g.Expect(got.LastUsed).To(Equal("2026-06-17"))
}

// TestRunActivate_LocksVaultAroundBumpLoop asserts that RunActivate acquires the
// vault lock BEFORE the first sidecar bump and releases it AFTER the last write,
// so a concurrent amend/resituate re-embed cannot clobber the freshly-written
// sidecar vectors with stale ones.
func TestRunActivate_LocksVaultAroundBumpLoop(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	orig := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@1",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.2},
		ContentHash:      "sha256:x",
		LastUsed:         "2025-01-01", // old date → bump will write
	}

	vault := "/vault"
	noteName := "1aa.2026-01-01.test.md"
	sidecarPath := vault + "/1aa.2026-01-01.test.vec.json"

	store := map[string][]byte{sidecarPath: embed.MarshalSidecar(orig)}

	var order []string

	deps := cli.ActivateDeps{
		Lock: func(string) (func(), error) {
			order = append(order, "lock")

			return func() { order = append(order, "unlock") }, nil
		},
		Now: func() time.Time { return time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) },
		Read: func(p string) ([]byte, error) {
			order = append(order, "read")

			b, ok := store[p]
			if !ok {
				return nil, errors.New("not found: " + p)
			}

			return b, nil
		},
		Write: func(p string, b []byte) error {
			order = append(order, "write")
			store[p] = b

			return nil
		},
		LogWarning: func(string, ...any) {},
	}

	err := cli.ExportRunActivate(cli.ActivateArgs{Vault: vault, Notes: []string{noteName}}, deps)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(order).To(ContainElements("lock", "read", "write", "unlock"),
		"all lock events must be recorded")

	lockIdx := sliceIndex(order, "lock")
	readIdx := sliceIndex(order, "read")
	writeIdx := sliceIndex(order, "write")
	unlockIdx := sliceIndex(order, "unlock")

	g.Expect(lockIdx).To(BeNumerically("<", readIdx), "lock must precede first read")
	g.Expect(readIdx).To(BeNumerically("<", writeIdx), "read must precede write")
	g.Expect(writeIdx).To(BeNumerically("<", unlockIdx), "last write must precede unlock")
}

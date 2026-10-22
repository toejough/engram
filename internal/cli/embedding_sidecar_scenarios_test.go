package cli_test

// Scenario tests of the embedding-sidecar component (docs/specs/technical/embedding-sidecar.md): the census,
// the selection of states by mode flag, and the embedding run. Helpers carry the prefix es1.

import (
	"bytes"
	"errors"
	"maps"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// Test792b10c10a2_EmbedStatus_EveryStateIsCounted verifies 792b10c10a2: one ok, one stale, one incompatible and one
// broken note give a count of 1 each, no missing notes, and a total of 4.
func Test792b10c10a2_EmbedStatus_EveryStateIsCounted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	es1PutNote(memFS, vault, "1.ok", es1Sidecar(es1ModelID, es1NoteBody))
	es1PutNote(memFS, vault, "2.stale", es1Sidecar(es1ModelID, "an earlier body\n"))
	es1PutNote(memFS, vault, "3.incompatible", es1Sidecar(es1OtherModelID, es1NoteBody))
	es1PutNote(memFS, vault, "4.broken", []byte("{not json"))

	var out bytes.Buffer

	err := cli.RunEmbedStatus(
		t.Context(),
		cli.EmbedStatusArgs{VaultPath: vault},
		newEmbedDeps(memFS),
		&out,
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("total:           4\n" +
		"with-embeddings: 1\n" +
		"without:         0\n" +
		"stale:           1\n" +
		"incompatible:    1\n" +
		"broken:          1\n"))
}

// Test792b10c10a3_EmbedStatus_AScanFailureIsReturned verifies 792b10c10a3: a failing scan fails the census with an
// error beginning "embed status: scan:" that wraps the scan error, and prints nothing.
func Test792b10c10a3_EmbedStatus_AScanFailureIsReturned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := newEmbedDeps(newInMemoryFS())
	deps.Scan = func(string) ([]vaultgraph.Note, error) { return nil, errEs1Scan }

	var out bytes.Buffer

	err := cli.RunEmbedStatus(t.Context(), cli.EmbedStatusArgs{VaultPath: t.TempDir()}, deps, &out)

	g.Expect(err).To(MatchError(HavePrefix("embed status: scan:")))
	g.Expect(err).To(MatchError(errEs1Scan))
	g.Expect(out.String()).To(BeEmpty())
}

// Test792b10c11a2_SelectStates_TheAllFlagSelectsEveryState verifies 792b10c11a2.
func Test792b10c11a2_SelectStates_TheAllFlagSelectsEveryState(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	es1ExpectSelection(g, cli.EmbedApplyArgs{All: true}, map[embed.State]bool{
		embed.StateOK:           true,
		embed.StateMissing:      true,
		embed.StateStale:        true,
		embed.StateIncompatible: true,
		embed.StateBroken:       true,
	})
}

// Test792b10c11a3_SelectStates_TheStaleFlagSelectsStaleAndBroken verifies 792b10c11a3.
func Test792b10c11a3_SelectStates_TheStaleFlagSelectsStaleAndBroken(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	es1ExpectSelection(g, cli.EmbedApplyArgs{Stale: true}, map[embed.State]bool{
		embed.StateOK:           false,
		embed.StateMissing:      false,
		embed.StateStale:        true,
		embed.StateIncompatible: false,
		embed.StateBroken:       true,
	})
}

// Test792b10c11a4_SelectStates_TheForceFlagSelectsIncompatibleAndBroken verifies 792b10c11a4.
func Test792b10c11a4_SelectStates_TheForceFlagSelectsIncompatibleAndBroken(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	es1ExpectSelection(g, cli.EmbedApplyArgs{Force: true}, map[embed.State]bool{
		embed.StateOK:           false,
		embed.StateMissing:      false,
		embed.StateStale:        false,
		embed.StateIncompatible: true,
		embed.StateBroken:       true,
	})
}

// Test792b10c11a5_SelectStates_TheMissingFlagIsTheExplicitDefault verifies 792b10c11a5: the missing flag selects
// what no flag selects.
func Test792b10c11a5_SelectStates_TheMissingFlagIsTheExplicitDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	es1ExpectSelection(g, cli.EmbedApplyArgs{Missing: true}, map[embed.State]bool{
		embed.StateOK:           false,
		embed.StateMissing:      true,
		embed.StateStale:        false,
		embed.StateIncompatible: false,
		embed.StateBroken:       true,
	})
}

// Test792b10c11a6_SelectStates_ModeFlagsCombine verifies 792b10c11a6: the stale and force flags together select
// stale, incompatible and broken, and not missing or ok.
func Test792b10c11a6_SelectStates_ModeFlagsCombine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	es1ExpectSelection(g, cli.EmbedApplyArgs{Stale: true, Force: true}, map[embed.State]bool{
		embed.StateOK:           false,
		embed.StateMissing:      false,
		embed.StateStale:        true,
		embed.StateIncompatible: true,
		embed.StateBroken:       true,
	})
}

// Test792b10c12a10_EmbedApply_OnlySidecarsAreWritten verifies 792b10c12a10: a run over missing, stale, incompatible,
// broken and ok notes with every mode selected leaves every note file byte-for-byte unchanged, and every path
// written is a sidecar beside a note.
func Test792b10c12a10_EmbedApply_OnlySidecarsAreWritten(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	es1PutNote(memFS, vault, "1.missing", nil)
	es1PutNote(memFS, vault, "2.stale", es1Sidecar(es1ModelID, "an earlier body\n"))
	es1PutNote(memFS, vault, "3.incompatible", es1Sidecar(es1OtherModelID, es1NoteBody))
	es1PutNote(memFS, vault, "4.broken", []byte("{not json"))
	es1PutNote(memFS, vault, "5.ok", es1Sidecar(es1ModelID, es1NoteBody))

	before := make(map[string][]byte, len(memFS.files))
	maps.Copy(before, memFS.files)

	deps := newEmbedDeps(memFS)
	const noteCount = 5

	written := make([]string, 0, noteCount)
	deps.Write = func(path string, data []byte) error {
		written = append(written, path)

		return memFS.Write(path, data)
	}

	var out bytes.Buffer

	err := cli.RunEmbedApply(t.Context(), cli.EmbedApplyArgs{VaultPath: vault, All: true}, deps, &out)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(written).To(HaveLen(noteCount))

	for _, path := range written {
		g.Expect(path).To(HaveSuffix(".vec.json"))
		g.Expect(memFS.files).To(HaveKey(strings.TrimSuffix(path, ".vec.json") + ".md"))
	}

	for path, data := range before {
		if strings.HasSuffix(path, ".md") {
			g.Expect(memFS.files[path]).To(Equal(data), "note %s changed", path)
		}
	}
}

// Test792b10c12a7_EmbedApply_ANoteThatCannotBeReadIsSkipped verifies 792b10c12a7: an unreadable note prints a line
// beginning "skip      " with the read error, no sidecar is written for it, the run goes on to the next note, and no
// error is returned.
func Test792b10c12a7_EmbedApply_ANoteThatCannotBeReadIsSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	es1PutNote(memFS, vault, "1.locked", nil)
	es1PutNote(memFS, vault, "2.open", nil)

	lockedPath := filepath.Join(vault, "1.locked.md")
	deps := newEmbedDeps(memFS)
	deps.Read = func(path string) ([]byte, error) {
		if path == lockedPath {
			return nil, errEs1Read
		}

		return memFS.Read(path)
	}

	var out bytes.Buffer

	err := cli.RunEmbedApply(t.Context(), cli.EmbedApplyArgs{VaultPath: vault}, deps, &out)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("skip      1.locked.md: read error: permission denied\n" +
		"embedded  2.open.md (missing)\n"))
	g.Expect(memFS.files).NotTo(HaveKey(filepath.Join(vault, "1.locked.vec.json")))
	g.Expect(memFS.files).To(HaveKey(filepath.Join(vault, "2.open.vec.json")))
}

// Test792b10c12a8_EmbedApply_AScanFailureIsReturned verifies 792b10c12a8: a failing scan fails the run with an error
// beginning "embed apply: scan:" that wraps the scan error, and nothing is written or printed.
func Test792b10c12a8_EmbedApply_AScanFailureIsReturned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	writes := 0
	deps := newEmbedDeps(newInMemoryFS())
	deps.Scan = func(string) ([]vaultgraph.Note, error) { return nil, errEs1Scan }
	deps.Write = func(string, []byte) error {
		writes++

		return nil
	}

	var out bytes.Buffer

	err := cli.RunEmbedApply(t.Context(), cli.EmbedApplyArgs{VaultPath: t.TempDir(), All: true}, deps, &out)

	g.Expect(err).To(MatchError(HavePrefix("embed apply: scan:")))
	g.Expect(err).To(MatchError(errEs1Scan))
	g.Expect(writes).To(BeZero())
	g.Expect(out.String()).To(BeEmpty())
}

// Test792b10c12a9_EmbedApply_NotesAreEmbeddedInAscendingOrder verifies 792b10c12a9: a scan that returns the notes in
// descending order still gives output lines, and writes, in ascending basename order.
func Test792b10c12a9_EmbedApply_NotesAreEmbeddedInAscendingOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	for _, base := range []string{"1.a", "2.b", "3.c"} {
		es1PutNote(memFS, vault, base, nil)
	}

	deps := newEmbedDeps(memFS)
	deps.Scan = func(string) ([]vaultgraph.Note, error) {
		return []vaultgraph.Note{{Basename: "3.c"}, {Basename: "1.a"}, {Basename: "2.b"}}, nil
	}

	written := make([]string, 0, len(memFS.files))
	deps.Write = func(path string, data []byte) error {
		written = append(written, path)

		return memFS.Write(path, data)
	}

	var out bytes.Buffer

	err := cli.RunEmbedApply(t.Context(), cli.EmbedApplyArgs{VaultPath: vault}, deps, &out)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(Equal("embedded  1.a.md (missing)\n" +
		"embedded  2.b.md (missing)\n" +
		"embedded  3.c.md (missing)\n"))
	g.Expect(written).To(Equal([]string{
		filepath.Join(vault, "1.a.vec.json"),
		filepath.Join(vault, "2.b.vec.json"),
		filepath.Join(vault, "3.c.vec.json"),
	}))
}

// unexported constants.
const (
	es1Dims         = 4
	es1ModelID      = "m@4"
	es1NoteBody     = "---\ntype: fact\n---\nbody\n"
	es1OtherModelID = "OLD@384"
)

// unexported variables.
var (
	errEs1Read = errors.New("permission denied")
	errEs1Scan = errors.New("vault is not readable")
)

// es1ExpectSelection asserts, for every state, whether the selection made by args picks it.
func es1ExpectSelection(g *WithT, args cli.EmbedApplyArgs, want map[embed.State]bool) {
	for state, selected := range want {
		g.Expect(cli.ExportShouldEmbed(args, state)).To(Equal(selected),
			"args=%+v state=%s", args, state)
	}
}

// es1PutNote stores the note es1NoteBody (and its sidecar when sidecar is not nil) in memFS under the vault.
func es1PutNote(memFS *inMemoryFS, vault, basename string, sidecar []byte) {
	memFS.files[filepath.Join(vault, basename+".md")] = []byte(es1NoteBody)

	if sidecar != nil {
		memFS.files[filepath.Join(vault, basename+".vec.json")] = sidecar
	}
}

// es1Sidecar returns a well-formed sidecar naming modelID whose content hash is that of hashedBody.
func es1Sidecar(modelID, hashedBody string) []byte {
	vector := make([]float32, es1Dims)

	return embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: modelID,
		Dims:             es1Dims,
		SituationVector:  vector,
		BodyVector:       vector,
		ContentHash:      embed.ContentHash([]byte(hashedBody)),
	})
}

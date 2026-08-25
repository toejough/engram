package cli_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestTargets_Query_BothEnvVarsSet_ServerTakesPrecedence verifies that when
// both ENGRAM_SERVER and ENGRAM_PARENT are set, `engram query` behaves
// exactly as it does with only ENGRAM_SERVER set — a single fetch to the
// server, no merge attempt, ENGRAM_PARENT inert.
func TestTargets_Query_BothEnvVarsSet_ServerTakesPrecedence(t *testing.T) {
	g := NewWithT(t)

	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")
	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	fetchCalls := 0

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "query", "--phrase", "x"},
		func(d *cli.Deps) {
			d.Fetch = func(_ context.Context, _, url string, _ []byte) (cli.FetchResponse, error) {
				fetchCalls++

				g.Expect(url).To(ContainSubstring("http://vault-host:8420/query"))

				return cli.FetchResponse{Status: 200, Body: []byte("version: 1\nitems: []\n")}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("version: 1\nitems: []\n"))
	g.Expect(fetchCalls).To(Equal(1), "only ENGRAM_SERVER's single exclusive fetch, no parent merge attempt")
}

// TestTargets_Query_MergedMode_CombinesLocalAndParent exercises
// vault-merged-recall end-to-end through Targets(): ENGRAM_PARENT set,
// a real local vault with one note, a mocked parent response with one
// item — the merged payload includes both, tagged with their origin.
func TestTargets_Query_MergedMode_CombinesLocalAndParent(t *testing.T) {
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())
	plantRealVaultNote(t, vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nlocal body\n", []float32{1, 0, 0, 0}, "m@4")

	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "query", "--phrase", "x", "--vault", vault, "--chunks-dir", t.TempDir()},
		func(d *cli.Deps) {
			d.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
			d.Fetch = func(_ context.Context, _, url string, _ []byte) (cli.FetchResponse, error) {
				g.Expect(url).To(ContainSubstring("http://parent-host:8420/query"))

				return cli.FetchResponse{Status: 200, Body: []byte(
					"version: 1\nmodel_id: m@4\nitems:\n" +
						"  - path: parent-note.md\n    kind: fact\n    score: 0.5\n    provenances: [direct]\n",
				)}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())

	var parsed queryParsed
	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())

	paths := make([]string, len(parsed.Items))
	for i, item := range parsed.Items {
		paths[i] = item.Path
	}

	g.Expect(paths).To(ContainElements("1.fact.md", "parent-note.md"))
}

// TestTargets_Query_MergedMode_LocalErrorSurfacesWhenParentSucceeds
// verifies that when the parent fetch succeeds but the local query itself
// fails (e.g. a note with no matching sidecar), the error surfaces rather
// than being silently absorbed — merging can't proceed without a local
// side, so this is a hard failure, unlike parent-unavailable's fallback.
func TestTargets_Query_MergedMode_LocalErrorSurfacesWhenParentSucceeds(t *testing.T) {
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())
	// A note with no matching sidecar triggers RunQuery's errQueryNoEmbeddings.
	g.Expect(os.WriteFile(filepath.Join(vault, "1.fact.md"),
		[]byte("---\ntype: fact\ntier: L2\nsituation: x\n---\n\nbody\n"), 0o600)).To(Succeed())

	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	_, stderr := executeCapturingBoth(t,
		[]string{"engram", "query", "--phrase", "x", "--vault", vault, "--chunks-dir", t.TempDir()},
		func(d *cli.Deps) {
			d.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				return cli.FetchResponse{Status: 200, Body: []byte("version: 1\nitems: []\n")}, nil
			}
		})

	g.Expect(stderr).NotTo(BeEmpty())
}

// TestTargets_Query_MergedMode_ParentUnavailableFallsBackToLocal verifies
// that when the parent is unreachable, `engram query` still returns the
// local vault's results and emits a non-fatal warning, rather than failing
// the whole command (spec: "Parent unavailability degrades to local-only
// results").
func TestTargets_Query_MergedMode_ParentUnavailableFallsBackToLocal(t *testing.T) {
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())
	plantRealVaultNote(t, vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nlocal body\n", []float32{1, 0, 0, 0}, "m@4")

	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "query", "--phrase", "x", "--vault", vault, "--chunks-dir", t.TempDir()},
		func(d *cli.Deps) {
			d.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				return cli.FetchResponse{}, errors.New("connection refused")
			}
		})

	g.Expect(stderr).To(ContainSubstring("warning"))
	g.Expect(stderr).To(ContainSubstring("parent"))

	var parsed queryParsed
	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())
	g.Expect(parsed.Items).To(HaveLen(1))
	g.Expect(parsed.Items[0].Path).To(Equal("1.fact.md"))
}

// plantRealVaultNote writes a note and a matching sidecar to a real vault
// directory on disk — the real-filesystem analog of query_helpers_test.go's
// plantWithFixedVector, needed here because merged-mode dispatch tests run
// through Targets() with a real EdgeFS, not the in-memory test double.
func plantRealVaultNote(t *testing.T, vault, relPath, body string, vec []float32, modelID string) {
	t.Helper()

	notePath := filepath.Join(vault, relPath)
	g := NewWithT(t)
	g.Expect(os.WriteFile(notePath, []byte(body), 0o600)).To(Succeed())

	sidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: modelID,
		Dims:             len(vec),
		SituationVector:  vec,
		BodyVector:       vec,
		ContentHash:      embed.ContentHash([]byte(body)),
	}

	g.Expect(os.WriteFile(
		filepath.Join(vault, embed.SidecarPath(relPath)), embed.MarshalSidecar(sidecar), 0o600,
	)).To(Succeed())
}

package cli_test

import (
	"bytes"
	"encoding/json"
	"hash/fnv"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
)

// TestEngramQuery_F6F91_EndToEnd builds the real binary and runs
// `engram query` against a tempdir vault prestamped with synthetic
// minilm-l6-v2@384 sidecars (~30 notes, deterministic). Verifies that
// the YAML payload includes the new clusters[] section and budget fields.
func TestEngramQuery_F6F91_EndToEnd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	permDir := vault
	g.Expect(os.MkdirAll(permDir, 0o700)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o700)).To(Succeed())

	// Plant 30 synthetic notes: 10 in each of 3 themed clusters, all
	// linking to a central hub note in their cluster. Each cluster's
	// notes share a per-cluster vector tail so they cluster together.
	const notesPerCluster = 10

	const numClusters = 3

	const dims = 384

	for clusterIdx := range numClusters {
		hubBasename := "hub-" + strconv.Itoa(clusterIdx)
		hubBody := buildSyntheticBody(
			"hub for cluster "+strconv.Itoa(clusterIdx), clusterIdx, notesPerCluster,
		)

		writeSyntheticNote(g, permDir, hubBasename, hubBody, syntheticVector(clusterIdx, 0, dims))

		for memberIdx := 1; memberIdx <= notesPerCluster; memberIdx++ {
			memberBase := "n" + strconv.Itoa(clusterIdx) + "-" + strconv.Itoa(memberIdx)
			memberBody := "---\ntype: fact\n---\nmember of cluster " +
				strconv.Itoa(clusterIdx) + " body\n[[" + hubBasename + "]]\n"

			writeSyntheticNote(
				g,
				permDir,
				memberBase,
				memberBody,
				syntheticVector(clusterIdx, memberIdx, dims),
			)
		}
	}

	binPath := filepath.Join(t.TempDir(), "engram")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/engram")
	build.Dir = projectRoot(t)
	buildOut, buildErr := build.CombinedOutput()
	g.Expect(buildErr).NotTo(HaveOccurred(), "build failed: %s", buildOut)

	if buildErr != nil {
		return
	}

	run := exec.Command(binPath, "query", "--phrase", "hub for cluster 1", "--vault", vault, "--limit", "5")

	var stdout bytes.Buffer

	run.Stdout = &stdout
	run.Stderr = os.Stderr
	runErr := run.Run()

	g.Expect(runErr).NotTo(HaveOccurred())

	if runErr != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(stdout.Bytes(), &parsed)).NotTo(HaveOccurred())

	g.Expect(parsed.Budget.TotalNotes).To(Equal(33))
	g.Expect(parsed.Budget.WithEmbeddings).To(Equal(33))
	g.Expect(parsed.Items).NotTo(BeEmpty())
	// The new payload sections always render (clusters may be empty).
	g.Expect(strings.Contains(stdout.String(), "clusters:")).To(BeTrue())
}

// unexported constants.
const (
	maxNibbleShift = 28
	nibbleMask     = 0xF
	nibbleMaxFloat = 15.0
)

// buildSyntheticBody returns a 100-char body that mentions the cluster
// theme and lists wikilinks to N members.
func buildSyntheticBody(theme string, clusterID, members int) string {
	var sb strings.Builder

	sb.WriteString("---\ntype: fact\n---\n")
	sb.WriteString(theme)
	sb.WriteString("\n")

	for i := 1; i <= members; i++ {
		sb.WriteString("[[n")
		sb.WriteString(strconv.Itoa(clusterID))
		sb.WriteString("-")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("]]\n")
	}

	return sb.String()
}

// syntheticVector returns a normalized 384-dim vector that puts cluster
// members near each other in cosine space.
//
// Cluster center direction comes from cluster index; per-member noise
// is small enough to keep silhouettes high.
func syntheticVector(clusterID, memberID, dims int) []float32 {
	const noiseScale = 0.05

	vec := make([]float32, dims)

	// Center direction: hot-encoded bin per cluster.
	for d := range vec {
		switch {
		case d == clusterID:
			vec[d] = 1.0
		case d < dims:
			vec[d] = 0
		}
	}

	// Small deterministic noise so members aren't pixel-perfect identical.
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(strconv.Itoa(clusterID) + "-" + strconv.Itoa(memberID)))

	prn := hasher.Sum32()

	for d := range dims {
		nibble := (prn >> uint(d%maxNibbleShift)) & nibbleMask
		vec[d] += noiseScale * float32(nibble) / nibbleMaxFloat
	}

	// L2-normalize so cosine ≈ dot product (sidecars are normalized at write).
	var norm float64

	for _, v := range vec {
		norm += float64(v) * float64(v)
	}

	norm = math.Sqrt(norm)
	if norm == 0 {
		return vec
	}

	for d := range vec {
		vec[d] = float32(float64(vec[d]) / norm)
	}

	return vec
}

// writeSyntheticNote plants a note and a hand-stamped minilm-l6-v2@384
// sidecar (with synthetic vector) under permDir.
func writeSyntheticNote(g Gomega, permDir, basename, body string, vector []float32) {
	notePath := filepath.Join(permDir, basename+".md")
	g.Expect(os.WriteFile(notePath, []byte(body), 0o600)).To(Succeed())

	sidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: embed.BundledModelID,
		Dims:             len(vector),
		SituationVector:  vector,
		BodyVector:       vector,
		ContentHash:      embed.ContentHash([]byte(body)),
	}

	scBytes, marshalErr := json.Marshal(sidecar)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	scPath := filepath.Join(permDir, basename+".vec.json")
	g.Expect(os.WriteFile(scPath, scBytes, 0o600)).To(Succeed())
}

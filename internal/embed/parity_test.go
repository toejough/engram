//go:build parity

// UAT 13 gate for the engram query spike. Compares Hugot+GoMLX cosines
// against Python sentence-transformers reference cosines on a fixed set
// of 5 pairs.
//
// Build-tag-gated because it loads a ~90MB ONNX model from disk and
// would slow targ test substantially. Run with:
//
//	CGO_ENABLED=0 go test -tags=parity -run=TestT13_ParityWithPythonReference ./internal/embed/...
package embed_test

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/knights-analytics/hugot"
)

func TestT13_ParityWithPythonReference(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	ctx := t.Context()

	refBytes, err := os.ReadFile("testdata/parity-reference.json")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var ref referencePayload

	g.Expect(json.Unmarshal(refBytes, &ref)).NotTo(HaveOccurred())

	modelDir, absErr := filepath.Abs("assets/model")
	g.Expect(absErr).NotTo(HaveOccurred())

	if absErr != nil {
		return
	}

	session, sessErr := hugot.NewGoSession(ctx)
	g.Expect(sessErr).NotTo(HaveOccurred())

	if sessErr != nil {
		return
	}

	defer func() {
		_ = session.Destroy()
	}()

	config := hugot.FeatureExtractionConfig{
		ModelPath:    modelDir,
		Name:         "engram-arctic-xs",
		OnnxFilename: "model.onnx",
	}

	pipeline, pipeErr := hugot.NewPipeline(session, config)
	g.Expect(pipeErr).NotTo(HaveOccurred())

	if pipeErr != nil {
		return
	}

	// Probe dims with a single sentence so a wrong-dims surface fails
	// loudly before we compare cosines.
	probe, probeErr := pipeline.RunPipeline(ctx, []string{"probe"})
	g.Expect(probeErr).NotTo(HaveOccurred())

	if probeErr != nil {
		return
	}

	g.Expect(probe.Embeddings).To(HaveLen(1))
	g.Expect(probe.Embeddings[0]).To(HaveLen(ref.Dims))

	for _, pair := range ref.Pairs {
		left, leftErr := pipeline.RunPipeline(ctx, []string{pair.Left})
		g.Expect(leftErr).NotTo(HaveOccurred())

		if leftErr != nil {
			continue
		}

		right, rightErr := pipeline.RunPipeline(ctx, []string{pair.Right})
		g.Expect(rightErr).NotTo(HaveOccurred())

		if rightErr != nil {
			continue
		}

		got := float64(cosine32(left.Embeddings[0], right.Embeddings[0]))
		diff := math.Abs(got - pair.Cosine)

		if diff > parityTolerance {
			t.Errorf(
				"pair %q vs %q: Go cosine %.6f, Python cosine %.6f, diff %.6f > tol %.6f",
				pair.Left, pair.Right, got, pair.Cosine, diff, parityTolerance,
			)
		} else {
			t.Logf(
				"OK: %q vs %q: Go %.6f Python %.6f diff %.2e",
				pair.Left, pair.Right, got, pair.Cosine, diff,
			)
		}
	}
}

// unexported constants.
const (
	parityTolerance = 1e-3
)

type referencePayload struct {
	ModelID string `json:"model_id"`
	Dims    int    `json:"dims"`
	Pairs   []struct {
		Left   string  `json:"left"`
		Right  string  `json:"right"`
		Cosine float64 `json:"cosine"`
	} `json:"pairs"`
}

// cosine32 — local copy. The package's exported Cosine lives in
// cosine.go; importing it here would require parity to build under the
// normal test tag too. Inline keeps the parity test self-contained.
func cosine32(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64

	for i := range a {
		af := float64(a[i])
		bf := float64(b[i])
		dot += af * bf
		normA += af * af
		normB += bf * bf
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}

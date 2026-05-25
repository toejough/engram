package cli_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestAutoEmbedNote_EmbedFailureWarnsButReturns asserts the warn-and-
// proceed semantics (UAT 3 spec): embed failure is logged, sidecar is
// not written, but the caller's learn write succeeds.
func TestAutoEmbedNote_EmbedFailureWarnsButReturns(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var warned bool

	var wroteSidecar bool

	deps := cli.LearnDeps{
		Embedder: failingEmbedder{},
		WriteSidecar: func(string, []byte) error {
			wroteSidecar = true

			return nil
		},
		LogWarning: func(string, ...any) { warned = true },
	}

	cli.ExportAutoEmbedNote(t.Context(), deps, "Permanent/1.foo.md", "body")

	g.Expect(warned).To(BeTrue())
	g.Expect(wroteSidecar).To(BeFalse())
}

// TestAutoEmbedNote_HappyPathWritesValidSidecar asserts the happy path:
// embedder returns a vector, sidecar is marshaled and written.
func TestAutoEmbedNote_HappyPathWritesValidSidecar(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var capturedPath string

	var capturedBytes []byte

	deps := cli.LearnDeps{
		Embedder: successEmbedder{},
		WriteSidecar: func(path string, data []byte) error {
			capturedPath = path
			capturedBytes = data

			return nil
		},
		LogWarning: func(string, ...any) {
			t.Fatal("happy path should not warn")
		},
	}

	cli.ExportAutoEmbedNote(t.Context(), deps, "Permanent/1.foo.md", "body")

	g.Expect(capturedPath).To(Equal("Permanent/1.foo.vec.json"))

	var parsed embed.Sidecar
	g.Expect(json.Unmarshal(capturedBytes, &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.EmbeddingModelID).To(Equal("m@4"))
	g.Expect(parsed.Dims).To(Equal(4))
	g.Expect(parsed.Vector).To(HaveLen(4))
	g.Expect(parsed.ContentHash).To(HavePrefix("sha256:"))
}

// TestAutoEmbedNote_NilEmbedderIsNoOp asserts the helper returns early
// when no embedder is wired — used by tests that don't exercise the
// embedding pipeline.
func TestAutoEmbedNote_NilEmbedderIsNoOp(t *testing.T) {
	t.Parallel()

	deps := cli.LearnDeps{
		WriteSidecar: func(string, []byte) error {
			t.Fatal("WriteSidecar should not be called when Embedder is nil")

			return nil
		},
	}

	cli.ExportAutoEmbedNote(t.Context(), deps, "Permanent/1.foo.md", "body")
}

// TestAutoEmbedNote_NilWriterIsNoOp asserts the helper returns early
// when WriteSidecar is unset — same shape, opposite arm.
func TestAutoEmbedNote_NilWriterIsNoOp(t *testing.T) {
	t.Parallel()

	deps := cli.LearnDeps{
		Embedder: failingEmbedder{},
	}

	cli.ExportAutoEmbedNote(t.Context(), deps, "Permanent/1.foo.md", "body")
}

// TestAutoEmbedNote_WriteFailureLoggedButSwallowed asserts that even a
// failed sidecar write doesn't propagate — warn-and-proceed.
func TestAutoEmbedNote_WriteFailureLoggedButSwallowed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var warned bool

	deps := cli.LearnDeps{
		Embedder:     successEmbedder{},
		WriteSidecar: func(string, []byte) error { return errors.New("disk full") },
		LogWarning:   func(string, ...any) { warned = true },
	}

	cli.ExportAutoEmbedNote(t.Context(), deps, "Permanent/1.foo.md", "body")
	g.Expect(warned).To(BeTrue())
}

// unexported variables.
var (
	errEmbedDown = errors.New("embedder down")
)

type failingEmbedder struct{}

func (failingEmbedder) Dims() int { return 4 }

func (failingEmbedder) Embed(context.Context, string) ([]float32, error) { return nil, errEmbedDown }

func (failingEmbedder) ModelID() string { return "m@4" }

type successEmbedder struct{}

func (successEmbedder) Dims() int { return 4 }

func (successEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3, 0.4}, nil
}

func (successEmbedder) ModelID() string { return "m@4" }

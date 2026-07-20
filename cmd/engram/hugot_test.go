package main

import (
	stdembed "embed"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// TestBundledEmbedder_Smoke exercises the full production wiring
// end-to-end: real hugot runtime (cmd's thin hugotRuntime), internally
// composed backend + cache FS, bundled model assets. Skipped under -short
// because it unpacks the ~90MB ONNX.
func TestBundledEmbedder_Smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bundled-embedder smoke test under -short")
	}

	t.Parallel()

	g := NewWithT(t)

	embedder, err := embed.NewBundledHugotEmbedder(
		t.Context(), embed.NewRuntimeBackend(hugotRuntime{}), realCacheFS(),
		filepath.Join(t.TempDir(), "model-cache"))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	defer func() {
		_ = embedder.Close()
	}()

	const expectedDims = 384

	g.Expect(embedder.ModelID()).To(Equal(embed.BundledModelID))
	g.Expect(embedder.Dims()).To(Equal(expectedDims))

	vec, embErr := embedder.Embed(t.Context(), "hello world")
	g.Expect(embErr).NotTo(HaveOccurred())
	g.Expect(vec).To(HaveLen(expectedDims))

	const longLen = 4000

	longText := make([]byte, longLen)
	for i := range longText {
		longText[i] = 'a' + byte(i%26)
	}

	vec2, embErr2 := embedder.Embed(t.Context(), string(longText))
	g.Expect(embErr2).NotTo(HaveOccurred())
	g.Expect(vec2).To(HaveLen(expectedDims))
}

// TestHugotRejectsInvalidModelDir exercises the embedder-construction
// error branch through the real runtime: extraction succeeds (files
// exist) but hugot rejects the directory because it has no valid
// model.onnx.
func TestHugotRejectsInvalidModelDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cacheDir := filepath.Join(t.TempDir(), "model-cache")
	_, err := embed.NewHugotEmbedderFromFS(
		t.Context(), embed.NewRuntimeBackend(hugotRuntime{}), realCacheFS(),
		testModelFS, "testdata", "fake@1", cacheDir)
	g.Expect(err).To(HaveOccurred())
}

//go:embed testdata
var testModelFS stdembed.FS

// realCacheFS mirrors the CacheFSPrims wiring cli.NewDeps builds from the
// production Primitives literal.
func realCacheFS() embed.CacheFS {
	return embed.NewCacheFS(embed.CacheFSPrims{
		Stat:      os.Stat,
		MkdirAll:  os.MkdirAll,
		MkdirTemp: os.MkdirTemp,
		WriteFile: os.WriteFile,
		Rename:    os.Rename,
		RemoveAll: os.RemoveAll,
	})
}

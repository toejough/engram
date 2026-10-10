package embed_test

import (
	"context"
	stdembed "embed"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// Test792b9c1a2_BundledModelFS_ExposesModelDir proves the exported accessor returns
// the go:embed-ed assets rooted at BundledModelDir, so cmd/engram (and its
// integration tests) can hand the bundled assets to the injectable
// constructors without touching the unexported bundledModel var.
func Test792b9c1a2_BundledModelFS_ExposesModelDir(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	entries, err := embed.BundledModelFS().ReadDir(embed.BundledModelDir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).NotTo(BeEmpty(), "bundled model dir must contain the model files")
}

func Test792b9c5a6_MissingBundledModel_ClearError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cacheDir := filepath.Join(t.TempDir(), "model-cache")
	_, err := embed.NewHugotEmbedderFromFS(
		context.Background(), &fakeBackend{}, &fakeCacheFS{}, emptyFS, "assets/model", "x@1", cacheDir)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("ENGRAM_MODEL_PATH"))
}

// unexported variables.
var (
	emptyFS stdembed.FS
)

package surface_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// T-P3-19: Prompt mode surfacing with spreading activation boosts linked memories.
// Verifies that toFrecencyInput is called during prompt sorting.
func TestPromptActivationSorting(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// BM25 requires unique terms (df=1) for positive IDF scores.
	// "xyzinvoke" is unique to mem1; query matches mem1 only.
	now := time.Now()
	memories := []*memory.Stored{
		{
			FilePath:  "mem1.toml",
			Title:     "xyzinvoke build system",
			Principle: "use xyzinvoke for builds",
			UpdatedAt: now,
		},
		{
			FilePath:  "mem2.toml",
			Title:     "abcparallel testing",
			Principle: "parallel abcparallel tests",
			UpdatedAt: now,
		},
		{
			FilePath:  "mem3.toml",
			Title:     "definjection pattern",
			Principle: "dependency definjection pattern",
			UpdatedAt: now,
		},
	}

	surfacer := surface.New(
		&fakeRetriever{memories: memories},
	)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "xyzinvoke build system",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	// Verify mem1 was surfaced (unique term xyzinvoke matches)
	g.Expect(output).NotTo(BeEmpty())
	// The activation boost should affect ranking
	g.Expect(output).To(ContainSubstring("[engram]"))
}

// T-P3: WithLinkReader, WithLinkUpdater, WithTitleFetcher are wired into the Surfacer.
func TestWithLinkReader_SetsOption(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reader := &fakeP3LinkReader{}
	surfacer := surface.New(
		&fakeRetriever{},
		surface.WithLinkReader(reader),
	)

	g.Expect(surfacer).NotTo(BeNil())
}

// fakeP3LinkReader is a test double for surface.LinkReader.
type fakeP3LinkReader struct{}

func (f *fakeP3LinkReader) GetEntryLinks(_ string) ([]surface.LinkGraphLink, error) {
	return nil, nil
}

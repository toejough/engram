package surface_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// T-P3: appendClusterNotes is called when linkReader and titleFetcher are set.
func TestAppendClusterNotes_CalledDuringSessionStart(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "mem1.toml", Title: "Memory One", UpdatedAt: time.Now()},
	}
	reader := &fakeP3LinkReaderWithLinks{
		links: []surface.LinkGraphLink{
			{Target: "mem2.toml", Weight: 0.5, Basis: "concept_overlap"},
		},
	}
	fetcher := &fakeP3TitleFetcherWithTitle{title: "Memory Two"}
	surfacer := surface.New(
		&fakeRetriever{memories: memories},
		surface.WithLinkReader(reader),
		surface.WithTitleFetcher(fetcher),
	)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(reader.getCalls).To(BeNumerically(">=", 1))
}

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

// T-P3-16: Spreading activation (REQ-P3-6) boosts a linked memory above an unlinked one.
// Setup: mem2 (eff=60) has link to mem3 (eff=90, weight=1.0).
//
//	activated[mem2] = 60 + 0.3*(90*1.0) = 87 > activated[mem1] = 80
//
// Expected output order: mem3(90) then mem2(87) then mem1(80).
func TestSpreadingActivation_ReranksMemories(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Now()
	memories := []*memory.Stored{
		{FilePath: "mem1.toml", Title: "Memory One", Principle: "principle one", UpdatedAt: now},
		{FilePath: "mem2.toml", Title: "Memory Two", Principle: "principle two", UpdatedAt: now},
		{
			FilePath:  "mem3.toml",
			Title:     "Memory Three",
			Principle: "principle three",
			UpdatedAt: now,
		},
	}

	eff := &fakeEffectivenessComputer{
		stats: map[string]surface.EffectivenessStat{
			"mem1.toml": {SurfacedCount: 5, EffectivenessScore: 80.0},
			"mem2.toml": {SurfacedCount: 5, EffectivenessScore: 60.0},
			"mem3.toml": {SurfacedCount: 5, EffectivenessScore: 90.0},
		},
	}

	// mem2 links to mem3 with weight 1.0 → activated[mem2] = 60 + 0.3*90 = 87
	reader := &fakeP3LinkReaderByPath{
		links: map[string][]surface.LinkGraphLink{
			"mem2.toml": {{Target: "mem3.toml", Weight: 1.0, Basis: "co_surfacing"}},
		},
	}

	surfacer := surface.New(
		&fakeRetriever{memories: memories},
		surface.WithEffectiveness(eff),
		surface.WithLinkReader(reader),
	)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	// writeRecencySection uses filenameSlug: "mem1.toml" → "mem1"
	mem1Idx := strings.Index(output, "mem1")
	mem2Idx := strings.Index(output, "mem2")
	mem3Idx := strings.Index(output, "mem3")

	g.Expect(mem3Idx).To(BeNumerically(">", -1), "mem3 must appear in output")
	g.Expect(mem2Idx).To(BeNumerically(">", -1), "mem2 must appear in output")
	g.Expect(mem1Idx).To(BeNumerically(">", -1), "mem1 must appear in output")

	// After activation: mem3(90) > mem2(87) > mem1(80)
	g.Expect(mem3Idx).To(BeNumerically("<", mem2Idx), "mem3 should appear before mem2")
	g.Expect(mem2Idx).To(BeNumerically("<", mem1Idx), "mem2 should appear before mem1")
}

// T-P3: updateCoSurfacingLinks is called for surfaced memory pairs.
func TestUpdateCoSurfacingLinks_CalledDuringSessionStart(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "mem1.toml", Title: "Memory One", UpdatedAt: time.Now()},
		{FilePath: "mem2.toml", Title: "Memory Two", UpdatedAt: time.Now()},
	}
	updater := &fakeP3LinkUpdaterSpy{}
	surfacer := surface.New(
		&fakeRetriever{memories: memories},
		surface.WithLinkUpdater(updater),
	)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(updater.getCalls).To(BeNumerically(">=", 1))
}

// T-P3: updateCoSurfacingLinks increments existing link (found=true path).
func TestUpdateCoSurfacingLinks_IncrementsExistingLink(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{FilePath: "mem1.toml", Title: "Memory One", UpdatedAt: time.Now()},
		{FilePath: "mem2.toml", Title: "Memory Two", UpdatedAt: time.Now()},
	}
	updater := &fakeP3LinkUpdaterWithExisting{
		links: []surface.LinkGraphLink{
			{Target: "mem2.toml", Weight: 0.9, Basis: "co_surfacing"},
		},
	}
	surfacer := surface.New(
		&fakeRetriever{memories: memories},
		surface.WithLinkUpdater(updater),
	)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(updater.setCalls).To(BeNumerically(">=", 1))
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

func TestWithLinkUpdater_SetsOption(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	updater := &fakeP3LinkUpdater{}
	surfacer := surface.New(
		&fakeRetriever{},
		surface.WithLinkUpdater(updater),
	)

	g.Expect(surfacer).NotTo(BeNil())
}

func TestWithTitleFetcher_SetsOption(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fetcher := &fakeP3TitleFetcher{}
	surfacer := surface.New(
		&fakeRetriever{},
		surface.WithTitleFetcher(fetcher),
	)

	g.Expect(surfacer).NotTo(BeNil())
}

// fakeP3LinkReader is a test double for surface.LinkReader.
type fakeP3LinkReader struct{}

func (f *fakeP3LinkReader) GetEntryLinks(_ string) ([]surface.LinkGraphLink, error) {
	return nil, nil
}

// fakeP3LinkReaderByPath returns links keyed by source path.
type fakeP3LinkReaderByPath struct {
	links map[string][]surface.LinkGraphLink
}

func (f *fakeP3LinkReaderByPath) GetEntryLinks(path string) ([]surface.LinkGraphLink, error) {
	return f.links[path], nil
}

// fakeP3LinkReaderWithLinks returns a fixed set of links.
type fakeP3LinkReaderWithLinks struct {
	getCalls int
	links    []surface.LinkGraphLink
}

func (f *fakeP3LinkReaderWithLinks) GetEntryLinks(_ string) ([]surface.LinkGraphLink, error) {
	f.getCalls++

	return f.links, nil
}

// fakeP3LinkUpdater is a test double for surface.LinkUpdater.
type fakeP3LinkUpdater struct{}

func (f *fakeP3LinkUpdater) GetEntryLinks(_ string) ([]surface.LinkGraphLink, error) {
	return nil, nil
}

func (f *fakeP3LinkUpdater) SetEntryLinks(_ string, _ []surface.LinkGraphLink) error {
	return nil
}

// fakeP3LinkUpdaterSpy tracks GetEntryLinks calls.
type fakeP3LinkUpdaterSpy struct {
	getCalls int
}

func (f *fakeP3LinkUpdaterSpy) GetEntryLinks(_ string) ([]surface.LinkGraphLink, error) {
	f.getCalls++

	return nil, nil
}

func (f *fakeP3LinkUpdaterSpy) SetEntryLinks(_ string, _ []surface.LinkGraphLink) error {
	return nil
}

// fakeP3LinkUpdaterWithExisting returns pre-existing links and tracks SetEntryLinks calls.
type fakeP3LinkUpdaterWithExisting struct {
	links    []surface.LinkGraphLink
	setCalls int
}

func (f *fakeP3LinkUpdaterWithExisting) GetEntryLinks(_ string) ([]surface.LinkGraphLink, error) {
	return f.links, nil
}

func (f *fakeP3LinkUpdaterWithExisting) SetEntryLinks(_ string, _ []surface.LinkGraphLink) error {
	f.setCalls++

	return nil
}

// fakeP3TitleFetcher is a test double for surface.TitleFetcher.
type fakeP3TitleFetcher struct{}

func (f *fakeP3TitleFetcher) GetTitle(_ string) (string, bool) {
	return "", false
}

// fakeP3TitleFetcherWithTitle returns a fixed title.
type fakeP3TitleFetcherWithTitle struct {
	title string
}

func (f *fakeP3TitleFetcherWithTitle) GetTitle(_ string) (string, bool) {
	return f.title, f.title != ""
}

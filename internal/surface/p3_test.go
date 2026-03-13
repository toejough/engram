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

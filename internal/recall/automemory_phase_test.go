package recall_test

import (
	"context"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
	"engram/internal/recall"
)

func TestExtractFromAutoMemory_BufferFullSkipsExtract(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindAutoMemory, Path: "/m/MEMORY.md"},
		{Kind: externalsources.KindAutoMemory, Path: "/m/topic.md"},
	}

	cache := externalsources.NewFileCache(func(_ string) ([]byte, error) {
		return []byte("body"), nil
	})

	summarizer := &autoMemoryFakeSummarizer{
		rankResponse: "topic.md",
		extractMap:   map[string]string{"body": "snippet"},
	}

	var buffer strings.Builder

	const cap1 = 100

	bytesUsed := recall.ExtractFromAutoMemory(
		context.Background(), files, "query", cache, summarizer, &buffer, cap1, cap1,
	)

	g.Expect(bytesUsed).To(Equal(0), "buffer already full, should skip extraction")
}

func TestExtractFromAutoMemory_NilSummarizerReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindAutoMemory, Path: "/m/MEMORY.md"},
	}

	var buffer strings.Builder

	bytesUsed := recall.ExtractFromAutoMemory(
		context.Background(), files, "query", nil, nil, &buffer, 0, 1024,
	)

	g.Expect(bytesUsed).To(Equal(0))
}

func TestExtractFromAutoMemory_NoMemoryIndexFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindAutoMemory, Path: "/m/debugging.md"},
	}

	cache := externalsources.NewFileCache(func(_ string) ([]byte, error) {
		return []byte("body"), nil
	})

	summarizer := &autoMemoryFakeSummarizer{}

	var buffer strings.Builder

	bytesUsed := recall.ExtractFromAutoMemory(
		context.Background(), files, "query", cache, summarizer, &buffer, 0, 1024,
	)

	g.Expect(bytesUsed).To(Equal(0))
	g.Expect(buffer.String()).To(BeEmpty())
}

func TestExtractFromAutoMemory_RanksAndExtractsUntilBufferFills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindAutoMemory, Path: "/m/MEMORY.md"},
		{Kind: externalsources.KindAutoMemory, Path: "/m/debugging.md"},
		{Kind: externalsources.KindAutoMemory, Path: "/m/architecture.md"},
	}

	contents := map[string][]byte{
		"/m/MEMORY.md":       []byte("Index: debugging.md, architecture.md"),
		"/m/debugging.md":    []byte("debugging body"),
		"/m/architecture.md": []byte("architecture body"),
	}

	cache := externalsources.NewFileCache(func(p string) ([]byte, error) {
		return contents[p], nil
	})

	summarizer := &autoMemoryFakeSummarizer{
		rankResponse: "debugging.md\narchitecture.md",
		extractMap: map[string]string{
			"debugging body":    "debugging snippet",
			"architecture body": "architecture snippet",
		},
	}

	var buffer strings.Builder

	const cap1 = 1024

	bytesUsed := recall.ExtractFromAutoMemory(
		context.Background(), files, "query", cache, summarizer, &buffer, 0, cap1,
	)

	g.Expect(bytesUsed).To(BeNumerically(">", 0))
	g.Expect(buffer.String()).To(ContainSubstring("debugging snippet"))
}

func TestExtractFromAutoMemory_UnknownRankedNameSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindAutoMemory, Path: "/m/MEMORY.md"},
		{Kind: externalsources.KindAutoMemory, Path: "/m/topic.md"},
	}

	contents := map[string][]byte{
		"/m/MEMORY.md": []byte("Index"),
		"/m/topic.md":  []byte("topic body"),
	}

	cache := externalsources.NewFileCache(func(p string) ([]byte, error) {
		return contents[p], nil
	})

	summarizer := &autoMemoryFakeSummarizer{
		rankResponse: "missing.md\ntopic.md",
		extractMap:   map[string]string{"topic body": "topic-snippet"},
	}

	var buffer strings.Builder

	bytesUsed := recall.ExtractFromAutoMemory(
		context.Background(), files, "query", cache, summarizer, &buffer, 0, 1024,
	)

	g.Expect(bytesUsed).To(BeNumerically(">", 0))
	g.Expect(buffer.String()).To(Equal("topic-snippet"))
}

// autoMemoryFakeSummarizer satisfies recall.SummarizerI; rank queries match by prefix.
type autoMemoryFakeSummarizer struct {
	rankResponse string
	extractMap   map[string]string
}

func (s *autoMemoryFakeSummarizer) ExtractRelevant(_ context.Context, content, query string) (string, error) {
	if strings.HasPrefix(query, "Rank") {
		return s.rankResponse, nil
	}

	return s.extractMap[content], nil
}

func (s *autoMemoryFakeSummarizer) SummarizeFindings(_ context.Context, content, _ string) (string, error) {
	return content, nil
}

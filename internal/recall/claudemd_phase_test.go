package recall_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
	"engram/internal/recall"
)

func TestExtractFromClaudeMd_BufferFullReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cache := externalsources.NewFileCache(func(_ string) ([]byte, error) {
		return []byte("body"), nil
	})

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindClaudeMd, Path: "/proj/CLAUDE.md"},
	}

	var buffer strings.Builder

	added := recall.ExtractFromClaudeMd(
		context.Background(), files, "query", cache,
		&claudeMdFakeSummarizer{}, &buffer, 1024, 1024,
	)

	g.Expect(added).To(Equal(0))
}

func TestExtractFromClaudeMd_ConcatenatesAndExtractsOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindClaudeMd, Path: "/proj/CLAUDE.md"},
		{Kind: externalsources.KindRules, Path: "/proj/.claude/rules/code.md"},
	}

	contents := map[string][]byte{
		"/proj/CLAUDE.md":             []byte("Project-level rules.\n"),
		"/proj/.claude/rules/code.md": []byte("Code style rules.\n"),
	}

	cache := externalsources.NewFileCache(func(p string) ([]byte, error) {
		return contents[p], nil
	})

	summarizer := &claudeMdFakeSummarizer{
		extractMap: map[string]string{
			"Project-level rules.\nCode style rules.\n": "combined-snippet",
		},
	}

	var buffer strings.Builder

	added := recall.ExtractFromClaudeMd(
		context.Background(), files, "query", cache, summarizer, &buffer, 0, 1024,
	)

	g.Expect(added).To(BeNumerically(">", 0))
	g.Expect(buffer.String()).To(ContainSubstring("combined-snippet"))
}

func TestExtractFromClaudeMd_ExtractError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindClaudeMd, Path: "/proj/CLAUDE.md"},
	}

	cache := externalsources.NewFileCache(func(_ string) ([]byte, error) {
		return []byte("rules"), nil
	})

	summarizer := &claudeMdFakeSummarizer{extractError: true}

	var buffer strings.Builder

	added := recall.ExtractFromClaudeMd(
		context.Background(), files, "query", cache, summarizer, &buffer, 0, 1024,
	)

	g.Expect(added).To(Equal(0))
}

func TestExtractFromClaudeMd_NilSummarizerReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buffer strings.Builder

	added := recall.ExtractFromClaudeMd(
		context.Background(), nil, "query", nil, nil, &buffer, 0, 1024,
	)

	g.Expect(added).To(Equal(0))
}

func TestExtractFromClaudeMd_NoFilesReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cache := externalsources.NewFileCache(func(_ string) ([]byte, error) {
		return nil, nil
	})

	var buffer strings.Builder

	added := recall.ExtractFromClaudeMd(
		context.Background(), []externalsources.ExternalFile{}, "query", cache,
		&claudeMdFakeSummarizer{}, &buffer, 0, 1024,
	)

	g.Expect(added).To(Equal(0))
}

func TestExtractFromClaudeMd_ReadErrorSkipsFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindClaudeMd, Path: "/proj/CLAUDE.md"},
		{Kind: externalsources.KindRules, Path: "/proj/.claude/rules/code.md"},
	}

	cache := externalsources.NewFileCache(func(p string) ([]byte, error) {
		if p == "/proj/CLAUDE.md" {
			return nil, errors.New("permission denied")
		}

		return []byte("Code rules.\n"), nil
	})

	summarizer := &claudeMdFakeSummarizer{
		extractMap: map[string]string{"Code rules.\n": "code-snippet"},
	}

	var buffer strings.Builder

	added := recall.ExtractFromClaudeMd(
		context.Background(), files, "query", cache, summarizer, &buffer, 0, 1024,
	)

	g.Expect(added).To(BeNumerically(">", 0))
	g.Expect(buffer.String()).To(Equal("code-snippet"))
}

// unexported variables.
var (
	errClaudeMdExtractFailed = recall.ErrNilCaller
)

// claudeMdFakeSummarizer satisfies recall.SummarizerI.
type claudeMdFakeSummarizer struct {
	extractMap   map[string]string
	extractError bool
}

func (s *claudeMdFakeSummarizer) ExtractRelevant(_ context.Context, content, _ string) (string, error) {
	if s.extractError {
		return "", errClaudeMdExtractFailed
	}

	return s.extractMap[content], nil
}

func (s *claudeMdFakeSummarizer) SummarizeFindings(_ context.Context, content, _ string) (string, error) {
	return content, nil
}

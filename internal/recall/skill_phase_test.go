package recall_test

import (
	"context"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
	"engram/internal/recall"
)

func TestExtractFromSkills_BufferFullSkipsExtract(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindSkill, Path: "/skills/foo/SKILL.md"},
	}

	cache := externalsources.NewFileCache(func(_ string) ([]byte, error) {
		return []byte("---\nname: foo\ndescription: a\n---\nbody"), nil
	})

	summarizer := &skillFakeSummarizer{
		rankResponse: "foo",
		extractMap:   map[string]string{"foo": "snippet"},
	}

	var buffer strings.Builder

	added := recall.ExtractFromSkills(
		context.Background(), files, "query", cache, summarizer, &buffer, 1024, 1024,
	)

	g.Expect(added).To(Equal(0))
}

func TestExtractFromSkills_NilSummarizerReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buffer strings.Builder

	added := recall.ExtractFromSkills(
		context.Background(), nil, "query", nil, nil, &buffer, 0, 1024,
	)

	g.Expect(added).To(Equal(0))
}

func TestExtractFromSkills_NoSkillsReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cache := externalsources.NewFileCache(func(_ string) ([]byte, error) {
		return nil, nil
	})

	var buffer strings.Builder

	added := recall.ExtractFromSkills(
		context.Background(), []externalsources.ExternalFile{}, "query", cache,
		&skillFakeSummarizer{}, &buffer, 0, 1024,
	)

	g.Expect(added).To(Equal(0))
}

func TestExtractFromSkills_RanksFrontmatterAndExtractsBodies(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindSkill, Path: "/skills/prepare/SKILL.md"},
		{Kind: externalsources.KindSkill, Path: "/skills/learn/SKILL.md"},
	}

	contents := map[string][]byte{
		"/skills/prepare/SKILL.md": []byte(`---
name: prepare
description: Use before starting new work to load context
---

# Prepare body
`),
		"/skills/learn/SKILL.md": []byte(`---
name: learn
description: Use after completing work to capture learnings
---

# Learn body
`),
	}

	cache := externalsources.NewFileCache(func(p string) ([]byte, error) {
		return contents[p], nil
	})

	summarizer := &skillFakeSummarizer{
		rankResponse: "prepare",
		extractMap:   map[string]string{"prepare": "prepare-snippet"},
	}

	var buffer strings.Builder

	added := recall.ExtractFromSkills(
		context.Background(), files, "query", cache, summarizer, &buffer, 0, 1024,
	)

	g.Expect(added).To(BeNumerically(">", 0))
	g.Expect(buffer.String()).To(ContainSubstring("prepare-snippet"))
}

func TestExtractFromSkills_UnknownRankedNameSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []externalsources.ExternalFile{
		{Kind: externalsources.KindSkill, Path: "/skills/foo/SKILL.md"},
	}

	cache := externalsources.NewFileCache(func(_ string) ([]byte, error) {
		return []byte("---\nname: foo\ndescription: a\n---\nfoo body"), nil
	})

	summarizer := &skillFakeSummarizer{
		rankResponse: "missing\nfoo",
		extractMap:   map[string]string{"foo": "foo-snippet"},
	}

	var buffer strings.Builder

	added := recall.ExtractFromSkills(
		context.Background(), files, "query", cache, summarizer, &buffer, 0, 1024,
	)

	g.Expect(added).To(BeNumerically(">", 0))
	g.Expect(buffer.String()).To(Equal("foo-snippet"))
}

// skillFakeSummarizer routes by query prefix.
type skillFakeSummarizer struct {
	rankResponse string
	extractMap   map[string]string // skill name → snippet
}

func (s *skillFakeSummarizer) ExtractRelevant(_ context.Context, content, query string) (string, error) {
	if strings.HasPrefix(query, "Rank") {
		return s.rankResponse, nil
	}

	for name, snippet := range s.extractMap {
		if strings.Contains(content, name) || strings.Contains(content, "body") {
			_ = name
			return snippet, nil
		}
	}

	return "", nil
}

func (s *skillFakeSummarizer) SummarizeFindings(_ context.Context, content, _ string) (string, error) {
	return content, nil
}

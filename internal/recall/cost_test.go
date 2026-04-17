package recall_test

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
	"engram/internal/recall"
)

func TestRecall_HaikuCallCountStaysBounded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		numSkills = 50
		numTopics = 20
	)

	files := make([]externalsources.ExternalFile, 0, numSkills+numTopics+1)
	contents := make(map[string][]byte, numSkills+numTopics+1)

	files = append(files,
		externalsources.ExternalFile{Kind: externalsources.KindAutoMemory, Path: "/m/MEMORY.md"},
	)
	contents["/m/MEMORY.md"] = []byte("Index of topics")

	for i := range numTopics {
		path := fmt.Sprintf("/m/topic%d.md", i)
		files = append(files, externalsources.ExternalFile{
			Kind: externalsources.KindAutoMemory, Path: path,
		})
		contents[path] = fmt.Appendf(nil, "body of topic %d", i)
	}

	for i := range numSkills {
		path := fmt.Sprintf("/s/skill%d/SKILL.md", i)
		files = append(files, externalsources.ExternalFile{
			Kind: externalsources.KindSkill, Path: path,
		})
		contents[path] = fmt.Appendf(nil,
			"---\nname: skill%d\ndescription: a skill\n---\nbody %d", i, i,
		)
	}

	cache := externalsources.NewFileCache(func(p string) ([]byte, error) {
		return contents[p], nil
	})

	finder := &fakeFinder{entries: []recall.FileEntry{
		{Path: "/sessions/now.jsonl", Mtime: time.Now()},
	}}
	reader := &fakeReader{contents: map[string]string{
		"/sessions/now.jsonl": "session content",
	}}

	counter := &countingHaikuSummarizer{returnSnippet: strings.Repeat("x", 1000)}

	orch := recall.NewOrchestrator(finder, reader, counter, nil, "",
		recall.WithExternalSources(files, cache),
	)

	_, err := orch.Recall(context.Background(), "/anywhere", "query")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Bound: 1 (engram rank) + 1 (auto-memory rank) + small N (auto extracts capped by buffer)
	//      + small N (session extracts capped) + 1 (skill rank) + small N (skill extracts capped)
	//      + 1 (claude.md combined) + 1 (synthesis).
	// With a 10KB buffer and 1KB per snippet, no phase should exceed ~10 extracts.
	const maxAcceptableCalls = 50

	g.Expect(int(counter.calls.Load())).To(BeNumerically("<=", maxAcceptableCalls),
		"Haiku call count must stay bounded under buffer pressure")
}

// countingHaikuSummarizer counts every ExtractRelevant + SummarizeFindings call.
type countingHaikuSummarizer struct {
	calls         atomic.Int64
	returnSnippet string
}

func (c *countingHaikuSummarizer) ExtractRelevant(_ context.Context, _, query string) (string, error) {
	c.calls.Add(1)

	if strings.Contains(query, "Rank topic files") {
		return "topic0.md\ntopic1.md\ntopic2.md", nil
	}

	if strings.Contains(query, "Rank skills") {
		return "skill0\nskill1\nskill2", nil
	}

	return c.returnSnippet, nil
}

func (c *countingHaikuSummarizer) SummarizeFindings(_ context.Context, content, _ string) (string, error) {
	c.calls.Add(1)
	return content, nil
}

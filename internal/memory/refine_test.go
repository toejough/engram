package memory_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// MockExtractor implements memory.LLMExtractor for testing
type MockExtractor struct {
	RewriteFunc      func(ctx context.Context, content string) (string, error)
	AddRationaleFunc func(ctx context.Context, content string) (string, error)
}

func (m *MockExtractor) AddRationale(ctx context.Context, content string) (string, error) {
	if m.AddRationaleFunc != nil {
		return m.AddRationaleFunc(ctx, content)
	}

	return content, nil
}

func (m *MockExtractor) Curate(ctx context.Context, query string, candidates []memory.QueryResult) ([]memory.CuratedResult, error) {
	return nil, nil
}

func (m *MockExtractor) Decide(ctx context.Context, newContent string, existing []memory.ExistingMemory) (*memory.IngestDecision, error) {
	return nil, nil
}

func (m *MockExtractor) Extract(ctx context.Context, content string) (*memory.Observation, error) {
	return nil, nil
}

func (m *MockExtractor) Filter(ctx context.Context, query string, candidates []memory.QueryResult) ([]memory.FilterResult, error) {
	return nil, nil
}

func (m *MockExtractor) PostEval(_ context.Context, _, _ string) (*memory.PostEvalResult, error) {
	return &memory.PostEvalResult{Faithfulness: 0.5, Signal: "positive"}, nil
}

func (m *MockExtractor) Rewrite(ctx context.Context, content string) (string, error) {
	if m.RewriteFunc != nil {
		return m.RewriteFunc(ctx, content)
	}

	return content, nil
}

func (m *MockExtractor) Synthesize(ctx context.Context, memories []string) (string, error) {
	return "", nil
}

// TEST-2104: ScanForRefinements detects CLAUDE.md entries with code blocks
func TestScanForRefinements_ClaudeMDCodeBlocks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	db, err := memory.InitTestDB(tmpDir + "/test.db")
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Create CLAUDE.md with entries mixing rule + code block
	claudeMDPath := tmpDir + "/CLAUDE.md"
	claudeMDContent := `# Working With Joe

## Promoted Learnings

- Use go test -tags sqlite_fts5 for all tests. Example: ` + "`go test -tags sqlite_fts5 -count=1`" + `
- Always validate inputs at system boundaries
`
	err = memory.WriteFile(claudeMDPath, []byte(claudeMDContent))
	g.Expect(err).ToNot(HaveOccurred())

	// Create mock extractor
	extractor := &memory.MockExtractor{}

	// Call ScanForRefinements
	proposals, err := memory.ScanForRefinements(db, claudeMDPath, extractor)
	g.Expect(err).ToNot(HaveOccurred())

	// Should generate "extract-examples" proposals
	var extractProposals []memory.MaintenanceProposal

	for _, p := range proposals {
		if p.Action == "extract-examples" {
			extractProposals = append(extractProposals, p)
		}
	}

	g.Expect(extractProposals).ToNot(BeEmpty())

	if len(extractProposals) < 1 {
		t.Fatal("expected at least 1 extract proposal")
	}

	g.Expect(extractProposals[0].Tier).To(Equal("claude-md"))
	g.Expect(extractProposals[0].Reason).To(ContainSubstring("code block"))
}

// TEST-2103: ScanForRefinements detects CLAUDE.md entries without rationale
func TestScanForRefinements_ClaudeMDEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	db, err := memory.InitTestDB(tmpDir + "/test.db")
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Create CLAUDE.md with imperative entries without explanation
	claudeMDPath := tmpDir + "/CLAUDE.md"
	claudeMDContent := `# Working With Joe

## Promoted Learnings

- Always use TDD for all code
- Never use git amend on pushed commits
`
	err = memory.WriteFile(claudeMDPath, []byte(claudeMDContent))
	g.Expect(err).ToNot(HaveOccurred())

	// Create mock extractor
	extractor := &memory.MockExtractor{
		AddRationaleFunc: func(ctx context.Context, content string) (string, error) {
			return content + " - this prevents bugs and improves design", nil
		},
	}

	// Call ScanForRefinements
	proposals, err := memory.ScanForRefinements(db, claudeMDPath, extractor)
	g.Expect(err).ToNot(HaveOccurred())

	// Should generate "add-rationale" proposals for CLAUDE.md entries
	var addRationaleProposals []memory.MaintenanceProposal

	for _, p := range proposals {
		if p.Action == "add-rationale" && p.Tier == "claude-md" {
			addRationaleProposals = append(addRationaleProposals, p)
		}
	}

	g.Expect(addRationaleProposals).ToNot(BeEmpty())

	if len(addRationaleProposals) < 1 {
		t.Fatal("expected at least 1 add-rationale proposal")
	}

	g.Expect(addRationaleProposals[0].Preview).To(ContainSubstring("prevents bugs"))
}

// TEST-2101: ScanForRefinements detects flagged_for_rewrite entries
func TestScanForRefinements_FlaggedForRewrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	db, err := memory.InitTestDB(tmpDir + "/test.db")
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Insert test data with flagged_for_rewrite=1
	_, err = db.Exec(`
		INSERT INTO embeddings (content, source, source_type, confidence, promoted, flagged_for_rewrite)
		VALUES ('Test content to rewrite', 'memory', 'user', 0.8, 0, 1)
	`)
	g.Expect(err).ToNot(HaveOccurred())

	claudeMDPath := tmpDir + "/CLAUDE.md"

	// Create mock extractor
	extractor := &memory.MockExtractor{
		RewriteFunc: func(ctx context.Context, content string) (string, error) {
			return "Refined: " + content, nil
		},
	}

	// Call ScanForRefinements
	proposals, err := memory.ScanForRefinements(db, claudeMDPath, extractor)
	g.Expect(err).ToNot(HaveOccurred())

	// Should generate "rewrite" proposals
	var rewriteProposals []memory.MaintenanceProposal

	for _, p := range proposals {
		if p.Action == "rewrite" {
			rewriteProposals = append(rewriteProposals, p)
		}
	}

	g.Expect(rewriteProposals).ToNot(BeEmpty())

	if len(rewriteProposals) < 1 {
		t.Fatal("expected at least 1 rewrite proposal")
	}

	g.Expect(rewriteProposals[0].Tier).To(Equal("embeddings"))
	g.Expect(rewriteProposals[0].Preview).To(ContainSubstring("Refined:"))
}

// TEST-2102: ScanForRefinements detects entries missing rationale
func TestScanForRefinements_MissingRationale(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	db, err := memory.InitTestDB(tmpDir + "/test.db")
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Insert test data with principle but empty rationale
	_, err = db.Exec(`
		INSERT INTO embeddings (content, source, source_type, confidence, promoted, principle, rationale)
		VALUES ('Test principle', 'memory', 'user', 0.8, 0, 'Always use TDD', '')
	`)
	g.Expect(err).ToNot(HaveOccurred())

	claudeMDPath := tmpDir + "/CLAUDE.md"

	// Create mock extractor
	extractor := &memory.MockExtractor{
		AddRationaleFunc: func(ctx context.Context, content string) (string, error) {
			return content + " because it improves code quality", nil
		},
	}

	// Call ScanForRefinements
	proposals, err := memory.ScanForRefinements(db, claudeMDPath, extractor)
	g.Expect(err).ToNot(HaveOccurred())

	// Should generate "add-rationale" proposals
	var addRationaleProposals []memory.MaintenanceProposal

	for _, p := range proposals {
		if p.Action == "add-rationale" {
			addRationaleProposals = append(addRationaleProposals, p)
		}
	}

	g.Expect(addRationaleProposals).ToNot(BeEmpty())

	if len(addRationaleProposals) < 1 {
		t.Fatal("expected at least 1 add-rationale proposal")
	}

	g.Expect(addRationaleProposals[0].Tier).To(Equal("embeddings"))
	g.Expect(addRationaleProposals[0].Preview).To(ContainSubstring("because it improves"))
}

// ============================================================================
// Unit tests for content refinement operations (ISSUE-218)
// ============================================================================

// TEST-2100: ScanForRefinements with nil extractor returns empty
func TestScanForRefinements_NilExtractor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	db, err := memory.InitTestDB(tmpDir + "/test.db")
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO embeddings (content, source, source_type, confidence, promoted, flagged_for_rewrite)
		VALUES ('Test content to rewrite', 'memory', 'user', 0.8, 0, 1)
	`)
	g.Expect(err).ToNot(HaveOccurred())

	claudeMDPath := tmpDir + "/CLAUDE.md"

	// Call with nil extractor
	proposals, err := memory.ScanForRefinements(db, claudeMDPath, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeEmpty())
}

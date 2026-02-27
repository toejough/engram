package memory

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	. "github.com/onsi/gomega"
)

func TestCommandExists_ExistingCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(commandExists("sh")).To(BeTrue(), "sh should exist on all Unix systems")
}

func TestCommandExists_NonExistentCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(commandExists("totally-fake-cmd-xyz-not-on-path")).To(BeFalse(), "non-existent command should return false")
}

// ─── EnforceClaudeMDBudget tests ─────────────────────────────────────────────

func TestEnforceClaudeMDBudget_FileNotFound(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposals, err := EnforceClaudeMDBudget("/nonexistent/CLAUDE.md", db, newQualityTestFS())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeEmpty(), "missing file should produce no proposals")
}

func TestEnforceClaudeMDBudget_OverBudget(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert promoted memories with varying effectiveness
	for i, eff := range []float64{0.9, 0.5, 0.1} {
		_, err := db.Exec(`INSERT INTO embeddings (content, source, quadrant, promoted, effectiveness)
			VALUES (?, 'test', 'working', 1, ?)`,
			fmt.Sprintf("promoted entry content item %c with extra words", rune('A'+i)), eff)
		g.Expect(err).ToNot(HaveOccurred())
	}

	fs := newQualityTestFS()

	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "content line here"
	}

	fs.setFile("/test/CLAUDE.md", strings.Join(lines, "\n"))

	proposals, err := EnforceClaudeMDBudget("/test/CLAUDE.md", db, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).ToNot(BeEmpty(), "over-budget file should produce demotion proposals")

	for _, p := range proposals {
		g.Expect(p.Action).To(Equal("remove"), "budget enforcement proposals should be 'remove' actions")
		g.Expect(p.Recommendation.Category).ToNot(BeEmpty(), "each proposal should have a recommendation category")
		g.Expect(p.Recommendation.Text).ToNot(BeEmpty(), "each proposal should have recommendation text")
	}
}

func TestEnforceClaudeMDBudget_RecommendationCategories(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Enforcement pattern (NEVER) → hook recommendation
	_, err = db.Exec(`INSERT INTO embeddings (content, source, quadrant, promoted, effectiveness)
		VALUES ('NEVER run git reset without checking git status first', 'test', 'working', 1, 0.1)`)
	g.Expect(err).ToNot(HaveOccurred())

	fs := newQualityTestFS()

	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line content"
	}

	fs.setFile("/test/CLAUDE.md", strings.Join(lines, "\n"))

	proposals, err := EnforceClaudeMDBudget("/test/CLAUDE.md", db, fs)
	g.Expect(err).ToNot(HaveOccurred())

	found := false

	for _, p := range proposals {
		if p.Recommendation.Category == "claude-md-demotion-to-hook" {
			found = true
			break
		}
	}

	g.Expect(found).To(BeTrue(), "enforcement pattern (NEVER) should recommend hook demotion")
}

func TestEnforceClaudeMDBudget_SortsByEffectivenessAscending(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert promoted memories with varying effectiveness — lowest is 0.2
	for i, eff := range []float64{0.8, 0.2, 0.5} {
		_, err := db.Exec(`INSERT INTO embeddings (content, source, quadrant, promoted, effectiveness)
			VALUES (?, 'test', 'working', 1, ?)`,
			fmt.Sprintf("promoted memory content number %c with extra padding words here", rune('A'+i)), eff)
		g.Expect(err).ToNot(HaveOccurred())
	}

	fs := newQualityTestFS()

	lines := make([]string, 150)
	for i := range lines {
		lines[i] = "line content here for padding"
	}

	fs.setFile("/test/CLAUDE.md", strings.Join(lines, "\n"))

	proposals, err := EnforceClaudeMDBudget("/test/CLAUDE.md", db, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).ToNot(BeEmpty())

	// First proposal should be lowest effectiveness (0.2)
	if len(proposals) >= 1 {
		g.Expect(proposals[0].Reason).To(ContainSubstring("0.20"),
			"first demotion proposal should be for lowest-effectiveness entry (0.20)")
	}
}

func TestEnforceClaudeMDBudget_WithinBudget(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	fs := newQualityTestFS()
	fs.setFile("/test/CLAUDE.md", "# Commands\n\nUse go test.\n")

	proposals, err := EnforceClaudeMDBudget("/test/CLAUDE.md", db, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeEmpty(), "file within budget should produce no proposals")
}

// ─── commandExists tests ──────────────────────────────────────────────────────

// ─── isAlphanumericDash tests ─────────────────────────────────────────────────

func TestIsAlphanumericDash_InvalidWithDot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isAlphanumericDash("file.go")).To(BeFalse())
}

func TestIsAlphanumericDash_InvalidWithSpace(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isAlphanumericDash("my command")).To(BeFalse())
}

func TestIsAlphanumericDash_InvalidWithSpecialChar(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isAlphanumericDash("cmd!")).To(BeFalse())
}

func TestIsAlphanumericDash_ValidAlphanumeric(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isAlphanumericDash("abc123")).To(BeTrue())
}

func TestIsAlphanumericDash_ValidWithDash(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isAlphanumericDash("my-command")).To(BeTrue())
}

func TestIsAlphanumericDash_ValidWithUnderscore(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isAlphanumericDash("my_var")).To(BeTrue())
}

func TestParseClaudeMDSections_ArchitectureTree(t *testing.T) {
	g := NewWithT(t)
	content := "# Project Structure\n\n```\nproject/\n├── main.go\n└── internal/\n```\n"
	sections, err := ParseClaudeMDSections(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sections).To(HaveLen(1))

	if len(sections) < 1 {
		t.Fatal("expected at least 1 section")
	}

	g.Expect(sections[0].Type).To(Equal("architecture"), "directory tree with ├── should be classified as architecture")
}

func TestParseClaudeMDSections_CodeStyle(t *testing.T) {
	g := NewWithT(t)
	content := `# Code Style

Use camelCase for variables.
`
	sections, err := ParseClaudeMDSections(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sections).To(HaveLen(1))

	if len(sections) < 1 {
		t.Fatal("expected at least 1 section")
	}

	g.Expect(sections[0].Type).To(Equal("code_style"), "short rules without special patterns should be classified as code_style")
}

func TestParseClaudeMDSections_CommandsTable(t *testing.T) {
	g := NewWithT(t)
	content := `# Commands

| Task | Command |
|------|---------|
| Build | go build |
| Test | go test |
`
	sections, err := ParseClaudeMDSections(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sections).To(HaveLen(1))

	if len(sections) < 1 {
		t.Fatal("expected at least 1 section")
	}

	g.Expect(sections[0].Type).To(Equal("commands"), "table with Task/Command columns should be classified as commands")
}

// ─── ParseClaudeMDSections tests ──────────────────────────────────────────────

func TestParseClaudeMDSections_EmptyContent(t *testing.T) {
	g := NewWithT(t)
	sections, err := ParseClaudeMDSections("")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sections).To(BeEmpty())
}

func TestParseClaudeMDSections_Gotchas(t *testing.T) {
	g := NewWithT(t)
	content := `# Critical Warnings

- **NEVER** run git reset --hard without checking status first
- **ALWAYS** use git stash before switching branches
`
	sections, err := ParseClaudeMDSections(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sections).To(HaveLen(1))

	if len(sections) < 1 {
		t.Fatal("expected at least 1 section")
	}

	g.Expect(sections[0].Type).To(Equal("gotchas"), "bullets with NEVER/ALWAYS should be classified as gotchas")
}

func TestParseClaudeMDSections_LineCount(t *testing.T) {
	g := NewWithT(t)
	content := `# Section

Line 1
Line 2
Line 3
`
	sections, err := ParseClaudeMDSections(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sections).To(HaveLen(1))

	if len(sections) < 1 {
		t.Fatal("expected at least 1 section")
	}

	g.Expect(sections[0].LineCount).To(BeNumerically(">", 0), "LineCount should reflect section body lines")
}

func TestParseClaudeMDSections_MultipleSections(t *testing.T) {
	g := NewWithT(t)
	content := `# Commands

| Task | Command |
|------|---------|
| Build | make |

# Critical Warnings

- **NEVER** do this
- **ALWAYS** do that
`
	sections, err := ParseClaudeMDSections(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sections).To(HaveLen(2))

	if len(sections) < 2 {
		t.Fatal("expected at least 2 sections")
	}

	g.Expect(sections[0].Name).To(Equal("Commands"))
	g.Expect(sections[0].Type).To(Equal("commands"))
	g.Expect(sections[1].Name).To(Equal("Critical Warnings"))
	g.Expect(sections[1].Type).To(Equal("gotchas"))
}

func TestParseClaudeMDSections_Other(t *testing.T) {
	g := NewWithT(t)
	content := `# Background

This project is a memory management system that stores learnings from Claude sessions.
It uses SQLite for persistence and ONNX for embedding generation.
The system includes decay, deduplication, and promotion features.
These are general notes about the project architecture and philosophy.
They help understand the overall design decisions made during development.
More context lines to ensure this doesn't match short code_style heuristic.
`
	sections, err := ParseClaudeMDSections(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sections).To(HaveLen(1))

	if len(sections) < 1 {
		t.Fatal("expected at least 1 section")
	}

	g.Expect(sections[0].Type).To(Equal("other"), "long generic content should be classified as other")
}

func TestParseClaudeMDSections_Testing(t *testing.T) {
	g := NewWithT(t)
	content := `# Testing

Use go test -tags sqlite_fts5 for all tests.
Follow the full red/green/refactor TDD cycle.
`
	sections, err := ParseClaudeMDSections(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sections).To(HaveLen(1))

	if len(sections) < 1 {
		t.Fatal("expected at least 1 section")
	}

	g.Expect(sections[0].Type).To(Equal("testing"), "go test pattern should be classified as testing")
}

func TestProposeClaudeMDChange_AllChecksPassed(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings (content, source, quadrant, projects_retrieved)
		VALUES ('Use go test -tags sqlite_fts5 for running tests', 'test', 'working', 'proj1,proj2,proj3')`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	// LLM: actionable=true, tier=claude-md, section=testing
	server := makeQualityCheckServer([]string{
		`{"actionable": true}`,
		`{"tier": "claude-md"}`,
		`{"section_type": "testing"}`,
	})
	defer server.Close()

	llm := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	proposal, err := ProposeClaudeMDChange(db, id, newQualityTestFS(), llm)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposal).ToNot(BeNil(), "all checks passing should produce a proposal")

	if proposal == nil {
		t.Fatal("proposal is nil")
	}

	g.Expect(proposal.Action).To(Equal("add"))
	g.Expect(proposal.Section).To(Equal("testing"))
	g.Expect(proposal.QualityChecks["working_knowledge"]).To(BeTrue())
	g.Expect(proposal.QualityChecks["universal"]).To(BeTrue())
	g.Expect(proposal.QualityChecks["actionable"]).To(BeTrue())
	g.Expect(proposal.QualityChecks["non_redundant"]).To(BeTrue())
	g.Expect(proposal.QualityChecks["right_tier"]).To(BeTrue())
	g.Expect(proposal.Recommendation.Category).To(Equal("claude-md-promotion"))
	g.Expect(proposal.Recommendation.Text).ToNot(BeEmpty())
}

func TestProposeClaudeMDChange_InsufficientProjects(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings (content, source, quadrant, projects_retrieved)
		VALUES ('Always verify inputs', 'test', 'working', 'proj1,proj2')`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	proposal, err := ProposeClaudeMDChange(db, id, newQualityTestFS(), nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposal).To(BeNil(), "fewer than 3 projects should not produce a proposal")
}

func TestProposeClaudeMDChange_LLMSaysNotActionable(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings (content, source, quadrant, projects_retrieved)
		VALUES ('Some vague thought about things', 'test', 'working', 'proj1,proj2,proj3')`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	// LLM returns not actionable
	server := makeQualityCheckServer([]string{`{"actionable": false}`})
	defer server.Close()

	llm := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	proposal, err := ProposeClaudeMDChange(db, id, newQualityTestFS(), llm)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposal).To(BeNil(), "LLM returning not actionable should produce nil proposal")
}

func TestProposeClaudeMDChange_LLMSaysWrongTier(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings (content, source, quadrant, projects_retrieved)
		VALUES ('Always use git hooks for enforcement patterns', 'test', 'working', 'proj1,proj2,proj3')`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	// LLM: actionable=true, tier=hook (not claude-md)
	server := makeQualityCheckServer([]string{
		`{"actionable": true}`,
		`{"tier": "hook"}`,
	})
	defer server.Close()

	llm := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	proposal, err := ProposeClaudeMDChange(db, id, newQualityTestFS(), llm)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposal).To(BeNil(), "LLM saying hook tier should produce nil proposal")
}

// ─── ProposeClaudeMDChange tests ──────────────────────────────────────────────

func TestProposeClaudeMDChange_MemoryNotFound(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	_, err = ProposeClaudeMDChange(db, 9999, newQualityTestFS(), nil)
	g.Expect(err).To(HaveOccurred(), "should error when memory not found")
}

func TestProposeClaudeMDChange_NonWorkingQuadrant(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings (content, source, quadrant, projects_retrieved)
		VALUES ('Always verify inputs', 'test', 'leech', 'proj1,proj2,proj3')`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	proposal, err := ProposeClaudeMDChange(db, id, newQualityTestFS(), nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposal).To(BeNil(), "non-working quadrant should not produce a proposal")
}

func TestProposeClaudeMDChange_PassesWithNoLLM(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings (content, source, quadrant, projects_retrieved)
		VALUES ('Always verify inputs before processing', 'test', 'working', 'proj1,proj2,proj3')`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	// nil LLM: LLM-dependent checks are skipped (treated as passing)
	proposal, err := ProposeClaudeMDChange(db, id, newQualityTestFS(), nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposal).ToNot(BeNil(), "working memory with 3+ projects and nil LLM should produce proposal")

	if proposal == nil {
		t.Fatal("proposal is nil")
	}

	g.Expect(proposal.Action).To(Equal("add"))
	g.Expect(proposal.SourceMemoryID).To(Equal(id))
	g.Expect(proposal.QualityChecks["working_knowledge"]).To(BeTrue())
	g.Expect(proposal.QualityChecks["universal"]).To(BeTrue())
}

func TestProposeClaudeMDChange_RedundantPromotedMemory(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert already-promoted memory with nearly identical content
	_, err = db.Exec(`INSERT INTO embeddings (content, source, quadrant, projects_retrieved, promoted)
		VALUES ('Always verify inputs before processing user data', 'test', 'working', 'proj1,proj2,proj3', 1)`)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert candidate memory with similar content (should be detected as redundant)
	res, err := db.Exec(`INSERT INTO embeddings (content, source, quadrant, projects_retrieved)
		VALUES ('Always verify inputs before processing user data carefully', 'test', 'working', 'proj1,proj2,proj3')`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	proposal, err := ProposeClaudeMDChange(db, id, newQualityTestFS(), nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposal).To(BeNil(), "redundant content should produce nil proposal")
}

func TestScoreClaudeMD_ConcisenessDropsWhenOverBudget(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	fs := newQualityTestFS()

	// Over budget (200 lines)
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = "some content line"
	}

	fs.setFile("/test/CLAUDE.md", strings.Join(lines, "\n"))
	scoreOver, err := ScoreClaudeMD("/test/CLAUDE.md", db, fs, nil)
	g.Expect(err).ToNot(HaveOccurred())

	// Within budget (20 lines)
	shortLines := make([]string, 20)
	for i := range shortLines {
		shortLines[i] = "content"
	}

	fs.setFile("/test/CLAUDE.md", strings.Join(shortLines, "\n"))
	scoreUnder, err := ScoreClaudeMD("/test/CLAUDE.md", db, fs, nil)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(scoreOver.Conciseness).To(BeNumerically("<", scoreUnder.Conciseness),
		"over-budget file should have lower conciseness score")
}

func TestScoreClaudeMD_CoverageReflectsWorkingMemories(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert working universal memories (not promoted)
	for i := range 5 {
		_, _ = db.Exec(`INSERT INTO embeddings (content, source, quadrant, projects_retrieved)
			VALUES (?, 'test', 'working', 'proj1,proj2,proj3')`,
			fmt.Sprintf("working memory content item %d for testing", i))
	}

	fs := newQualityTestFS()
	fs.setFile("/test/CLAUDE.md", "# Commands\n\nUse go test.\n")

	score, err := ScoreClaudeMD("/test/CLAUDE.md", db, fs, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(score.Coverage).To(BeNumerically(">=", 0.0))
	g.Expect(score.Coverage).To(BeNumerically("<=", 100.0))
	// With 5 working memories and 0 promoted, coverage should be 0
	g.Expect(score.Coverage).To(BeNumerically("~", 0.0, 1.0),
		"no promoted memories → coverage should be near 0")
}

// ─── ScoreClaudeMD tests ──────────────────────────────────────────────────────

func TestScoreClaudeMD_FileNotFound(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	score, err := ScoreClaudeMD("/nonexistent/CLAUDE.md", db, newQualityTestFS(), nil)
	g.Expect(err).ToNot(HaveOccurred(), "file not found should not return error, just report in Issues")
	g.Expect(score).ToNot(BeNil())
	g.Expect(score.OverallGrade).To(Equal("F"), "missing file should result in F grade")
	g.Expect(score.Issues).To(ContainElement(ContainSubstring("not found")),
		"should report file not found issue")
}

func TestScoreClaudeMD_GradeScale(t *testing.T) {
	g := NewWithT(t)

	cases := []struct {
		score float64
		grade string
	}{
		{95, "A"},
		{85, "B"},
		{75, "C"},
		{65, "D"},
		{55, "F"},
	}
	for _, tc := range cases {
		g.Expect(gradeFromScore(tc.score)).To(Equal(tc.grade),
			"score %.0f should give grade %s", tc.score, tc.grade)
	}
}

func TestScoreClaudeMD_ReturnsAllDimensions(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	fs := newQualityTestFS()
	fs.setFile("/test/CLAUDE.md", "# Commands\n\nUse go test for tests.\n")

	score, err := ScoreClaudeMD("/test/CLAUDE.md", db, fs, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(score).ToNot(BeNil())
	g.Expect(score.OverallScore).To(BeNumerically(">=", 0.0))
	g.Expect(score.OverallScore).To(BeNumerically("<=", 100.0))
	g.Expect(score.OverallGrade).To(MatchRegexp(`^[A-F]$`))
	g.Expect(score.ContextPrecision).To(BeNumerically(">=", 0.0))
	g.Expect(score.Faithfulness).To(BeNumerically(">=", 0.0))
	g.Expect(score.Currency).To(BeNumerically(">=", 0.0))
	g.Expect(score.Conciseness).To(BeNumerically(">=", 0.0))
	g.Expect(score.Coverage).To(BeNumerically(">=", 0.0))
}

// qualityTestFS is a minimal in-memory FileSystem for CLAUDE.md quality tests.
type qualityTestFS struct {
	files map[string][]byte
}

func (f *qualityTestFS) MkdirAll(_ string, _ os.FileMode) error { return nil }

func (f *qualityTestFS) ReadDir(_ string) ([]os.DirEntry, error) { return nil, nil }

func (f *qualityTestFS) ReadFile(path string) ([]byte, error) {
	data, ok := f.files[path]
	if !ok {
		return nil, fmt.Errorf("open %s: %w", path, os.ErrNotExist)
	}

	return data, nil
}

func (f *qualityTestFS) Remove(_ string) error { return nil }

func (f *qualityTestFS) Rename(_, _ string) error { return nil }

func (f *qualityTestFS) Stat(path string) (os.FileInfo, error) { return nil, os.ErrNotExist }

func (f *qualityTestFS) WriteFile(path string, data []byte, _ os.FileMode) error {
	f.files[path] = data
	return nil
}

func (f *qualityTestFS) setFile(path, content string) { f.files[path] = []byte(content) }

// makeQualityCheckServer creates a mock httptest server that returns sequential JSON responses.
// Each call to the server returns the next response in the slice.
func makeQualityCheckServer(responses []string) *httptest.Server {
	var callCount int32

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(atomic.AddInt32(&callCount, 1)) - 1

		text := `{"actionable": true}` // safe default
		if n < len(responses) {
			text = responses[n]
		}

		response := map[string]any{
			"id":    "msg_test",
			"type":  "message",
			"model": "claude-haiku-4-5-20251001",
			"content": []map[string]any{
				{"type": "text", "text": text},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
}

func newQualityTestFS() *qualityTestFS { return &qualityTestFS{files: make(map[string][]byte)} }

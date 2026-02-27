package memory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestAppendToClaudeMDWithFS_ExistingSection_BeforeNextSection_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := newQualityTestFS()
	fs.setFile("/path/CLAUDE.md", "## Promoted Learnings\n\n- Old entry\n\n## Commands\n\nstuff\n")

	err := appendToClaudeMDWithFS(fs, "/path/CLAUDE.md", []string{"Inserted learning"})
	g.Expect(err).ToNot(HaveOccurred())

	data, readErr := fs.ReadFile("/path/CLAUDE.md")
	g.Expect(readErr).ToNot(HaveOccurred())

	result := string(data)
	insertedIdx := strings.Index(result, "- Inserted learning")
	commandsIdx := strings.Index(result, "## Commands")
	g.Expect(insertedIdx).To(BeNumerically("<", commandsIdx))
}

func TestAppendToClaudeMDWithFS_ExistingSection_NoNextSection_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := newQualityTestFS()
	fs.setFile("/path/CLAUDE.md", "# Config\n\n## Promoted Learnings\n\n- Old entry\n")

	err := appendToClaudeMDWithFS(fs, "/path/CLAUDE.md", []string{"New learning"})
	g.Expect(err).ToNot(HaveOccurred())

	data, readErr := fs.ReadFile("/path/CLAUDE.md")
	g.Expect(readErr).ToNot(HaveOccurred())

	result := string(data)
	g.Expect(result).To(ContainSubstring("- Old entry"))
	g.Expect(result).To(ContainSubstring("- New learning"))
}

// ─── appendToClaudeMDWithFS tests ─────────────────────────────────────────────

func TestAppendToClaudeMDWithFS_NewFile_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := newNotagSimpleFS()

	err := appendToClaudeMDWithFS(fs, "/path/CLAUDE.md", []string{"New learning entry"})
	g.Expect(err).ToNot(HaveOccurred())

	data, readErr := fs.ReadFile("/path/CLAUDE.md")
	g.Expect(readErr).ToNot(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("## Promoted Learnings"))
	g.Expect(string(data)).To(ContainSubstring("- New learning entry"))
}

func TestAppendToClaudeMDWithFS_NonEmptyNoSection_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := newQualityTestFS()
	fs.setFile("/path/CLAUDE.md", "# My Config\n\nExisting content here\n")

	err := appendToClaudeMDWithFS(fs, "/path/CLAUDE.md", []string{"Added learning"})
	g.Expect(err).ToNot(HaveOccurred())

	data, readErr := fs.ReadFile("/path/CLAUDE.md")
	g.Expect(readErr).ToNot(HaveOccurred())

	result := string(data)
	g.Expect(result).To(ContainSubstring("## Promoted Learnings"))
	g.Expect(result).To(ContainSubstring("- Added learning"))
	g.Expect(result).To(ContainSubstring("# My Config"))
}

// ─── ApplyClaudeMDProposal error path tests ────────────────────────────────────

func TestApplyClaudeMDProposal_ConsolidateBadTarget_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := newQualityTestFS()
	fs.setFile("/path/CLAUDE.md", "## Promoted Learnings\n\n- entry one\n")

	proposal := MaintenanceProposal{
		Action:  "consolidate",
		Target:  "bad-target-without-pipe",
		Preview: "merged",
	}

	err := ApplyClaudeMDProposal(fs, "/path/CLAUDE.md", proposal)
	g.Expect(err).To(MatchError(ContainSubstring("consolidate target must be")))
}

func TestApplyClaudeMDProposal_SplitTooFewParts_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := newQualityTestFS()
	fs.setFile("/path/CLAUDE.md", "## Promoted Learnings\n\n- long entry here\n")

	proposal := MaintenanceProposal{
		Action:  "split",
		Target:  "long entry here",
		Preview: "single-part-no-pipe",
	}

	err := ApplyClaudeMDProposal(fs, "/path/CLAUDE.md", proposal)
	g.Expect(err).To(MatchError(ContainSubstring("split preview must contain at least 2 parts")))
}

// ─── checkActionabilityLLM tests ─────────────────────────────────────────────

func TestCheckActionabilityLLM_CallerError_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{err: context.DeadlineExceeded}

	_, err := checkActionabilityLLM(context.Background(), caller, "some content")
	g.Expect(err).To(HaveOccurred())
}

func TestCheckActionabilityLLM_False_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte(`{"actionable": false}`)}

	result, err := checkActionabilityLLM(context.Background(), caller, "vague thought")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeFalse())
}

func TestCheckActionabilityLLM_ParseError_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte("not valid json")}

	_, err := checkActionabilityLLM(context.Background(), caller, "some content")
	g.Expect(err).To(MatchError(ContainSubstring("parse")))
}

func TestCheckActionabilityLLM_True_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte(`{"actionable": true}`)}

	result, err := checkActionabilityLLM(context.Background(), caller, "Always use TDD")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeTrue())
}

// ─── checkContextErr tests ────────────────────────────────────────────────────

func TestCheckContextErr_Background(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := checkContextErr(context.Background())
	g.Expect(err).ToNot(HaveOccurred())
}

func TestCheckContextErr_Canceled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := checkContextErr(ctx)
	g.Expect(err).To(Equal(context.Canceled))
}

func TestCheckContextErr_DeadlineExceeded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))

	defer cancel()

	err := checkContextErr(ctx)
	g.Expect(err).To(Equal(context.DeadlineExceeded))
}

func TestCheckContextErr_Nil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := checkContextErr(nil) //nolint:staticcheck // testing nil context branch explicitly
	g.Expect(err).ToNot(HaveOccurred())
}

// ─── checkTierFitLLM tests ────────────────────────────────────────────────────

func TestCheckTierFitLLM_CallerError_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{err: context.DeadlineExceeded}

	_, err := checkTierFitLLM(context.Background(), caller, "some content")
	g.Expect(err).To(HaveOccurred())
}

func TestCheckTierFitLLM_ClaudeMD_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte(`{"tier": "claude-md"}`)}

	result, err := checkTierFitLLM(context.Background(), caller, "universal rule")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeTrue())
}

func TestCheckTierFitLLM_Hook_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte(`{"tier": "hook"}`)}

	result, err := checkTierFitLLM(context.Background(), caller, "enforcement pattern")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeFalse())
}

func TestCheckTierFitLLM_ParseError_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte("not valid json")}

	_, err := checkTierFitLLM(context.Background(), caller, "some content")
	g.Expect(err).To(MatchError(ContainSubstring("parse")))
}

func TestCheckTierFitLLM_Skill_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte(`{"tier": "skill"}`)}

	result, err := checkTierFitLLM(context.Background(), caller, "domain-specific knowledge")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeFalse())
}

// ─── classifySectionLLM tests ─────────────────────────────────────────────────

func TestClassifySectionLLM_CallerError_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{err: context.DeadlineExceeded}

	_, err := classifySectionLLM(context.Background(), caller, "some content")
	g.Expect(err).To(HaveOccurred())
}

func TestClassifySectionLLM_GotchasSection_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte(`{"section_type": "gotchas"}`)}

	result, err := classifySectionLLM(context.Background(), caller, "NEVER do this")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("gotchas"))
}

func TestClassifySectionLLM_ParseError_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte("not valid json")}

	_, err := classifySectionLLM(context.Background(), caller, "some content")
	g.Expect(err).To(MatchError(ContainSubstring("parse")))
}

func TestClassifySectionLLM_ValidSection_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte(`{"section_type": "testing"}`)}

	result, err := classifySectionLLM(context.Background(), caller, "go test content")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("testing"))
}

func TestCountFillerLines_AllPatterns_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lines := []string{
		"TODO: fix this later",
		"FIXME: broken code here",
		"NOTE: important caveat",
		"see also the documentation",
		"refer to the spec",
	}

	count := countFillerLines(lines)
	g.Expect(count).To(Equal(5))
}

func TestCountFillerLines_CaseInsensitive_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lines := []string{"todo: lowercase check", "FIXME: uppercase check"}

	count := countFillerLines(lines)
	g.Expect(count).To(Equal(2))
}

// ─── countFillerLines tests ───────────────────────────────────────────────────

func TestCountFillerLines_Empty_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	count := countFillerLines(nil)
	g.Expect(count).To(Equal(0))
}

func TestCountFillerLines_MultiPatternOneLine_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// A single line with multiple patterns should count only once (break after first match)
	lines := []string{"TODO: FIXME: also broken"}

	count := countFillerLines(lines)
	g.Expect(count).To(Equal(1))
}

func TestCountFillerLines_NoFillers_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lines := []string{"normal content", "another line", "more stuff"}

	count := countFillerLines(lines)
	g.Expect(count).To(Equal(0))
}

func TestExtractCommands_Deduplicates_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmds := extractCommands("Use `go` then `go` again later")
	goCount := 0

	for _, c := range cmds {
		if c == "go" {
			goCount++
		}
	}

	g.Expect(goCount).To(Equal(1))
}

func TestExtractCommands_EmptyBackticks_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Empty backtick pair → fields[0] doesn't exist
	cmds := extractCommands("Empty `` backtick pair")
	g.Expect(cmds).To(BeEmpty())
}

func TestExtractCommands_EmptyContent_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmds := extractCommands("")
	g.Expect(cmds).To(BeEmpty())
}

func TestExtractCommands_MultipleCommands_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmds := extractCommands("Use `go` and `make` commands")
	g.Expect(cmds).To(ContainElements("go", "make"))
}

// ─── extractCommands tests ────────────────────────────────────────────────────

func TestExtractCommands_NoBackticks_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmds := extractCommands("plain text without any backtick commands")
	g.Expect(cmds).To(BeEmpty())
}

func TestExtractCommands_TooLong_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// > 20 chars → excluded
	cmds := extractCommands("Use `this-is-way-too-long-xx` here")
	g.Expect(cmds).To(BeEmpty())
}

func TestExtractCommands_TooShort_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// 1 char → < 2 → excluded
	cmds := extractCommands("Use `x` command")
	g.Expect(cmds).To(BeEmpty())
}

func TestExtractCommands_ValidCommand_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmds := extractCommands("Run `go` to build the project")
	g.Expect(cmds).To(ContainElement("go"))
}

func TestExtractCommands_WithDot_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// file.go has dot → not alphanumeric-dash
	cmds := extractCommands("Edit `file.go` here")
	g.Expect(cmds).To(BeEmpty())
}

// TestMergeEntries_ExtractorFails_Notag verifies mergeEntries falls back to heuristic on LLM error.
func TestMergeEntries_ExtractorFails_Notag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ext := &notagMergeExtractor{synthesizeResult: "", synthesizeErr: errors.New("api failed")}

	result := mergeEntries("longer entry wins here", "short", ext)

	g.Expect(result).To(Equal("longer entry wins here"))
}

// TestMergeEntries_ExtractorSucceeds_Notag verifies mergeEntries uses the LLM result when Synthesize succeeds.
func TestMergeEntries_ExtractorSucceeds_Notag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ext := &notagMergeExtractor{synthesizeResult: "merged principle", synthesizeErr: nil}

	result := mergeEntries("entry one", "entry two", ext)

	g.Expect(result).To(Equal("merged principle"))
}

func TestMergeEntries_NilExtractor_EqualLength_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Equal length: entry1 is NOT longer, so returns entry2
	result := mergeEntries("abc", "xyz", nil)
	g.Expect(result).To(Equal("xyz"))
}

// ─── mergeEntries tests ───────────────────────────────────────────────────────

func TestMergeEntries_NilExtractor_LongerFirst_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := mergeEntries("this longer entry wins the comparison", "short", nil)
	g.Expect(result).To(Equal("this longer entry wins the comparison"))
}

func TestMergeEntries_NilExtractor_LongerSecond_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := mergeEntries("short", "this longer entry wins the comparison", nil)
	g.Expect(result).To(Equal("this longer entry wins the comparison"))
}

// ─── OptimizeInteractive early path tests ─────────────────────────────────────

func TestOptimizeInteractive_WithNoLLMEval_Notag(t *testing.T) {
	g := NewWithT(t)

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "embeddings.db")
	claudeMDPath := filepath.Join(dir, "CLAUDE.md")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	err = os.WriteFile(claudeMDPath, []byte("# Config\n\n## Promoted Learnings\n\n- Entry one\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := OptimizeInteractiveOpts{
		DBPath:       dbPath,
		ClaudeMDPath: claudeMDPath,
		ReviewFunc:   func(_ MaintenanceProposal) bool { return false },
		Context:      context.Background(),
		NoLLMEval:    true,
		Extractor:    nil,
	}

	result, err := OptimizeInteractive(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result != nil {
		g.Expect(result.Approved).To(Equal(0))
	}
}

func TestScanClaudeMDFeedback_EmptyContent_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMDPath := filepath.Join(dir, "CLAUDE.md")

	err := os.WriteFile(claudeMDPath, []byte(""), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(dir, "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposals, err := ScanClaudeMDFeedback(db, claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeNil())
}

func TestScanClaudeMDFeedback_EmptyPromotedSection_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMDPath := filepath.Join(dir, "CLAUDE.md")

	// Section header present but no actual entries
	err := os.WriteFile(claudeMDPath, []byte("## Promoted Learnings\n\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(dir, "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposals, err := ScanClaudeMDFeedback(db, claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeNil())
}

// ─── ScanClaudeMDFeedback early-return path tests ─────────────────────────────

func TestScanClaudeMDFeedback_FileNotFound_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposals, err := ScanClaudeMDFeedback(db, "/nonexistent/path/CLAUDE.md")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeNil())
}

func TestScanClaudeMDFeedback_NoFlaggedEmbeddings_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMDPath := filepath.Join(dir, "CLAUDE.md")

	err := os.WriteFile(claudeMDPath, []byte("## Promoted Learnings\n\n- Always use TDD\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(dir, "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert a promoted embedding but NOT flagged
	_, err = db.Exec(`INSERT INTO embeddings (content, source, promoted, flagged_for_review)
		VALUES ('Always use TDD', 'test', 1, 0)`)
	g.Expect(err).ToNot(HaveOccurred())

	proposals, err := ScanClaudeMDFeedback(db, claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeNil())
}

func TestScanClaudeMDFeedback_NoPromotedLearnings_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMDPath := filepath.Join(dir, "CLAUDE.md")

	err := os.WriteFile(claudeMDPath, []byte("# Some Section\n\nContent without Promoted Learnings.\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(dir, "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposals, err := ScanClaudeMDFeedback(db, claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeNil())
}

func TestScoreCurrency_ExistingCommand_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// sh is always on PATH
	score := scoreCurrency("Run `sh` for shell access")
	g.Expect(score).To(BeNumerically("~", 100.0, 0.1))
}

func TestScoreCurrency_FakeCommand_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	score := scoreCurrency("Use `zzz-fake-cmd-xyz` for everything")
	g.Expect(score).To(BeNumerically("~", 0.0, 0.1))
}

func TestScoreCurrency_MixedCommands_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// sh exists, zzz-fake-cmd-xyz does not → 50%
	score := scoreCurrency("Run `sh` or `zzz-fake-cmd-xyz`")
	g.Expect(score).To(BeNumerically("~", 50.0, 0.1))
}

// ─── scoreCurrency tests ──────────────────────────────────────────────────────

func TestScoreCurrency_NoCommands_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	score := scoreCurrency("no backtick commands in this text")
	g.Expect(score).To(BeNumerically("~", 75.0, 0.1))
}

// ─── scoreFaithfulness tests ──────────────────────────────────────────────────

func TestScoreFaithfulness_EmptyDB_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	score, err := scoreFaithfulness(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(score).To(BeNumerically("~", 50.0, 0.1))
}

func TestScoreFaithfulness_HighEffectivenessCapped_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// effectiveness > 1 should be capped at 100
	_, err = db.Exec(`INSERT INTO embeddings (content, source, promoted, effectiveness)
		VALUES ('some content', 'test', 1, 2.0)`)
	g.Expect(err).ToNot(HaveOccurred())

	score, err := scoreFaithfulness(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(score).To(BeNumerically("~", 100.0, 0.1))
}

func TestScoreFaithfulness_WithPromotedMemory_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	_, err = db.Exec(`INSERT INTO embeddings (content, source, promoted, effectiveness)
		VALUES ('always use TDD for all changes', 'test', 1, 0.8)`)
	g.Expect(err).ToNot(HaveOccurred())

	score, err := scoreFaithfulness(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(score).To(BeNumerically("~", 80.0, 0.1))
}

func TestSplitLongEntry_ConjunctionSplit_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// No ". " but has "; " - falls through to conjunction split
	result := splitLongEntry("First part; second part here")
	g.Expect(result).ToNot(BeEmpty())
}

func TestSplitLongEntry_MultipleSentences_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := splitLongEntry("First sentence. Second sentence. Third sentence")
	g.Expect(len(result)).To(BeNumerically(">=", 2))

	if len(result) >= 1 {
		g.Expect(result[0]).To(ContainSubstring("First"))
	}
}

func TestSplitLongEntry_SemicolonOnly_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := splitLongEntry("Part one; part two; part three")
	// semicolons → splits into multiple parts
	g.Expect(len(result)).To(BeNumerically(">=", 2))
}

func TestSplitLongEntry_SentenceEndsWithPeriod_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := splitLongEntry("Use TDD always. Write tests first.")
	g.Expect(len(result)).To(BeNumerically(">=", 2))
	// Each part should have a period
	for _, part := range result {
		if len(part) > 0 {
			g.Expect(part).To(MatchRegexp(`[.!?]$`))
		}
	}
}

// ─── splitLongEntry tests ─────────────────────────────────────────────────────

func TestSplitLongEntry_SingleSentence_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := splitLongEntry("single sentence without split point")
	g.Expect(result).To(HaveLen(1))

	if len(result) > 0 {
		g.Expect(result[0]).To(Equal("single sentence without split point"))
	}
}

// ─── triageOneProposal tests ──────────────────────────────────────────────────

func TestTriageOneProposal_CallerError_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{err: context.DeadlineExceeded}
	proposal := MaintenanceProposal{
		Action:  "consolidate",
		Tier:    "embeddings",
		Reason:  "high similarity",
		Preview: "merged entry",
	}

	valid, rationale, err := triageOneProposal(context.Background(), caller, proposal)
	g.Expect(err).To(HaveOccurred())
	g.Expect(valid).To(BeFalse())
	g.Expect(rationale).To(BeEmpty())
}

func TestTriageOneProposal_ParseError_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte("not valid json at all")}
	proposal := MaintenanceProposal{
		Action:  "demote",
		Tier:    "claude-md",
		Reason:  "too specific",
		Preview: "use targ for builds",
	}

	valid, rationale, err := triageOneProposal(context.Background(), caller, proposal)
	g.Expect(err).To(HaveOccurred())
	g.Expect(valid).To(BeFalse())
	g.Expect(rationale).To(BeEmpty())
}

func TestTriageOneProposal_ValidFalse_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte(`{"valid": false, "rationale": "too project-specific"}`)}
	proposal := MaintenanceProposal{
		Action:  "split",
		Tier:    "claude-md",
		Reason:  "multiple topics",
		Preview: "part one|part two",
	}

	valid, rationale, err := triageOneProposal(context.Background(), caller, proposal)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(valid).To(BeFalse())
	g.Expect(rationale).To(Equal("too project-specific"))
}

func TestTriageOneProposal_ValidResponse_Notag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	caller := &notagAPICaller{response: []byte(`{"valid": true, "rationale": "universal learning"}`)}
	proposal := MaintenanceProposal{
		Action:  "promote",
		Tier:    "embeddings",
		Reason:  "universal pattern",
		Preview: "always use TDD",
	}

	valid, rationale, err := triageOneProposal(context.Background(), caller, proposal)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(valid).To(BeTrue())
	g.Expect(rationale).To(Equal("universal learning"))
}

// notagAPICaller implements APIMessageCaller for tests without build tags.
type notagAPICaller struct {
	response []byte
	err      error
}

func (m *notagAPICaller) CallAPIWithMessages(_ context.Context, _ APIMessageParams) ([]byte, error) {
	return m.response, m.err
}

// notagMergeExtractor is a minimal LLMExtractor for testing mergeEntries without the sqlite_fts5 tag.
type notagMergeExtractor struct {
	synthesizeResult string
	synthesizeErr    error
}

func (m *notagMergeExtractor) AddRationale(_ context.Context, _ string) (string, error) {
	return "", errors.New("not implemented")
}

func (m *notagMergeExtractor) Curate(_ context.Context, _ string, _ []QueryResult) ([]CuratedResult, error) {
	return nil, errors.New("not implemented")
}

func (m *notagMergeExtractor) Decide(_ context.Context, _ string, _ []ExistingMemory) (*IngestDecision, error) {
	return nil, errors.New("not implemented")
}

func (m *notagMergeExtractor) Extract(_ context.Context, _ string) (*Observation, error) {
	return nil, errors.New("not implemented")
}

func (m *notagMergeExtractor) Filter(_ context.Context, _ string, _ []QueryResult) ([]FilterResult, error) {
	return nil, errors.New("not implemented")
}

func (m *notagMergeExtractor) PostEval(_ context.Context, _, _ string) (*PostEvalResult, error) {
	return nil, errors.New("not implemented")
}

func (m *notagMergeExtractor) Rewrite(_ context.Context, _ string) (string, error) {
	return "", errors.New("not implemented")
}

func (m *notagMergeExtractor) Synthesize(_ context.Context, _ []string) (string, error) {
	return m.synthesizeResult, m.synthesizeErr
}

// notagSimpleFS is a minimal in-memory FileSystem that returns bare os.ErrNotExist
// (not wrapped) for missing files, so os.IsNotExist works correctly.
type notagSimpleFS struct {
	files map[string][]byte
}

func (f *notagSimpleFS) MkdirAll(_ string, _ os.FileMode) error { return nil }

func (f *notagSimpleFS) ReadDir(_ string) ([]os.DirEntry, error) { return nil, nil }

func (f *notagSimpleFS) ReadFile(path string) ([]byte, error) {
	data, ok := f.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	return data, nil
}

func (f *notagSimpleFS) Remove(_ string) error { return nil }

func (f *notagSimpleFS) Rename(_, _ string) error { return nil }

func (f *notagSimpleFS) Stat(_ string) (os.FileInfo, error) { return nil, os.ErrNotExist }

func (f *notagSimpleFS) WriteFile(path string, data []byte, _ os.FileMode) error {
	f.files[path] = data
	return nil
}

func newNotagSimpleFS() *notagSimpleFS {
	return &notagSimpleFS{files: make(map[string][]byte)}
}

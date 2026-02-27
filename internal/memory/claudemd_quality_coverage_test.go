//go:build sqlite_fts5

package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// ─── checkActionabilityLLM tests ─────────────────────────────────────────────

func TestCheckActionabilityLLM_ParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	server := makeInvalidJSONLLMServer()
	defer server.Close()

	caller := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))
	apiCaller, ok := any(caller).(APIMessageCaller)
	g.Expect(ok).To(BeTrue(), "DirectAPIExtractor should implement APIMessageCaller")

	_, err := checkActionabilityLLM(context.Background(), apiCaller, "some content")
	g.Expect(err).To(HaveOccurred(), "invalid JSON from LLM should return parse error")
	g.Expect(err.Error()).To(ContainSubstring("parse"))
}

func TestCheckActionabilityLLM_ValidTrue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	server := makeFixedJSONLLMServer(`{"actionable": true}`)
	defer server.Close()

	caller := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))
	apiCaller, ok := any(caller).(APIMessageCaller)
	g.Expect(ok).To(BeTrue())

	result, err := checkActionabilityLLM(context.Background(), apiCaller, "some content")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeTrue())
}

// ─── checkTierFitLLM tests ────────────────────────────────────────────────────

func TestCheckTierFitLLM_ParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	server := makeInvalidJSONLLMServer()
	defer server.Close()

	caller := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))
	apiCaller, ok := any(caller).(APIMessageCaller)
	g.Expect(ok).To(BeTrue())

	_, err := checkTierFitLLM(context.Background(), apiCaller, "some content")
	g.Expect(err).To(HaveOccurred(), "invalid JSON from LLM should return parse error")
	g.Expect(err.Error()).To(ContainSubstring("parse"))
}

func TestCheckTierFitLLM_ValidClaudeMD(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	server := makeFixedJSONLLMServer(`{"tier": "claude-md"}`)
	defer server.Close()

	caller := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))
	apiCaller, ok := any(caller).(APIMessageCaller)
	g.Expect(ok).To(BeTrue())

	result, err := checkTierFitLLM(context.Background(), apiCaller, "some content")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeTrue(), "tier=claude-md should return true")
}

// ─── classifySectionLLM tests ─────────────────────────────────────────────────

func TestClassifySectionLLM_ParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	server := makeInvalidJSONLLMServer()
	defer server.Close()

	caller := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))
	apiCaller, ok := any(caller).(APIMessageCaller)
	g.Expect(ok).To(BeTrue())

	_, err := classifySectionLLM(context.Background(), apiCaller, "some content")
	g.Expect(err).To(HaveOccurred(), "invalid JSON from LLM should return parse error")
	g.Expect(err.Error()).To(ContainSubstring("parse"))
}

func TestClassifySectionLLM_ValidResponse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	server := makeFixedJSONLLMServer(`{"section_type": "testing"}`)
	defer server.Close()

	caller := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))
	apiCaller, ok := any(caller).(APIMessageCaller)
	g.Expect(ok).To(BeTrue())

	sectionType, err := classifySectionLLM(context.Background(), apiCaller, "some content")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sectionType).To(Equal("testing"))
}

// ─── countFillerLines tests ───────────────────────────────────────────────────

func TestCountFillerLines_AllPatterns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lines := []string{
		"TODO: fix this later",
		"FIXME: broken code",
		"NOTE: important caveat",
		"see also the documentation",
		"refer to the spec",
	}

	count := countFillerLines(lines)
	g.Expect(count).To(Equal(5), "each filler pattern should be counted once")
}

func TestCountFillerLines_MultiplePatternsSameLine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// A line with multiple patterns should only be counted once (break after first match)
	lines := []string{"TODO: FIXME this broken code"}
	count := countFillerLines(lines)
	g.Expect(count).To(Equal(1), "line with multiple filler patterns should be counted only once")
}

func TestCountFillerLines_NoFillerLines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lines := []string{"regular content", "no filler here", "clean code"}
	count := countFillerLines(lines)
	g.Expect(count).To(Equal(0))
}

// ─── extractCommands tests ────────────────────────────────────────────────────

func TestExtractCommands_CommandTooLong(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// command > 20 chars should be excluded
	cmds := extractCommands("Use `this-is-a-very-long-cmd-name-x` here")
	g.Expect(cmds).To(BeEmpty(), "commands > 20 chars should be excluded")
}

func TestExtractCommands_CommandTooShort(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// single char command is < 2, should be excluded
	cmds := extractCommands("Use `x` command")
	g.Expect(cmds).To(BeEmpty(), "1-char commands should be excluded")
}

func TestExtractCommands_DeduplicatesCommands(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmds := extractCommands("Use `targ` first then `targ` again")
	g.Expect(cmds).To(HaveLen(1))
	g.Expect(cmds[0]).To(Equal("targ"))
}

func TestExtractCommands_EmptyBackticks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmds := extractCommands("Empty `` backtick here")
	g.Expect(cmds).To(BeEmpty(), "empty backtick should produce no commands")
}

func TestExtractCommands_EmptyContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmds := extractCommands("")
	g.Expect(cmds).To(BeEmpty())
}

func TestExtractCommands_MultipleUniqueCommands(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmds := extractCommands("Use `targ` and `go` commands")
	g.Expect(cmds).To(HaveLen(2))
	g.Expect(cmds).To(ContainElements("targ", "go"))
}

func TestExtractCommands_NoBackticks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmds := extractCommands("run some command without backticks")
	g.Expect(cmds).To(BeEmpty())
}

func TestExtractCommands_NonAlphanumericCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Contains dot → isAlphanumericDash returns false
	cmds := extractCommands("Use `cmd.exe` tool")
	g.Expect(cmds).To(BeEmpty(), "commands with dots should be excluded")
}

func TestExtractCommands_ValidCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmds := extractCommands("Run `go` to build")
	g.Expect(cmds).To(ContainElement("go"))
}

// ─── scoreCurrency tests ──────────────────────────────────────────────────────

func TestScoreCurrency_ExistingCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Use "sh" which should always be on PATH
	score := scoreCurrency("Run `sh` for shell operations")
	g.Expect(score).To(BeNumerically("~", 100.0, 0.1), "existing command → 100%")
}

func TestScoreCurrency_NoCommands(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	score := scoreCurrency("no commands here, just text")
	g.Expect(score).To(BeNumerically("~", 75.0, 0.1), "no commands → neutral 75.0")
}

func TestScoreCurrency_NonExistentCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	score := scoreCurrency("Use `totally-fake-xyz123` command")
	g.Expect(score).To(BeNumerically("~", 0.0, 0.1), "non-existent command → 0%")
}

// ─── scoreFaithfulness tests ──────────────────────────────────────────────────

func TestScoreFaithfulness_EffectivenessOverMax(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert with effectiveness > 1.0 to test the capping logic (v > 100 branch)
	_, err = db.Exec(`INSERT INTO embeddings (content, source, quadrant, promoted, effectiveness)
		VALUES ('test content', 'test', 'working', 1, 2.0)`)
	g.Expect(err).ToNot(HaveOccurred())

	score, err := scoreFaithfulness(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(score).To(BeNumerically("~", 100.0, 0.1), "effectiveness > 1 should be capped at 100")
}

func TestScoreFaithfulness_NormalEffectiveness(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tempDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	_, err = db.Exec(`INSERT INTO embeddings (content, source, quadrant, promoted, effectiveness)
		VALUES ('test content', 'test', 'working', 1, 0.8)`)
	g.Expect(err).ToNot(HaveOccurred())

	score, err := scoreFaithfulness(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(score).To(BeNumerically("~", 80.0, 0.1), "effectiveness 0.8 → score 80.0")
}

// ─── test helpers ─────────────────────────────────────────────────────────────

// makeFixedJSONLLMServer returns a mock LLM server that always responds with the given text.
func makeFixedJSONLLMServer(text string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":    "msg_test",
			"type":  "message",
			"model": "claude-haiku",
			"content": []map[string]any{
				{"type": "text", "text": text},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
}

// makeInvalidJSONLLMServer returns a mock LLM server that responds with non-JSON text,
// causing JSON parse errors in LLM helper functions.
func makeInvalidJSONLLMServer() *httptest.Server {
	return makeFixedJSONLLMServer("not-valid-json")
}

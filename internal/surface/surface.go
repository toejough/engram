// Package surface implements memory surfacing and enforcement for UC-2 (ARCH-12).
// Routes to SessionStart, UserPromptSubmit, or PreToolUse mode based on options.
package surface

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"engram/internal/memory"
)

// Exported constants.
const (
	ModePrompt       = "prompt"
	ModeSessionStart = "session-start"
	ModeTool         = "tool"
)

// Exported variables.
var (
	ErrUnknownMode = errors.New("surface: unknown mode")
)

// MemoryRetriever lists stored memories from disk (ARCH-9).
type MemoryRetriever interface {
	ListMemories(ctx context.Context, dataDir string) ([]*memory.Stored, error)
}

// Options configures a surface invocation.
type Options struct {
	Mode      string
	DataDir   string
	Message   string // for prompt mode
	ToolName  string // for tool mode
	ToolInput string // for tool mode
	Token     string // API token for LLM judgment
}

// Surfacer orchestrates memory surfacing and enforcement.
type Surfacer struct {
	retriever MemoryRetriever
	enforcer  ToolEnforcer
	stderr    io.Writer
}

// New creates a Surfacer. The enforcer may be nil if tool mode is not used.
// The stderr writer may be nil (defaults to no warnings).
func New(retriever MemoryRetriever, enforcer ToolEnforcer, stderr io.Writer) *Surfacer {
	return &Surfacer{
		retriever: retriever,
		enforcer:  enforcer,
		stderr:    stderr,
	}
}

// Run executes the surface subcommand, writing output to w.
func (s *Surfacer) Run(ctx context.Context, w io.Writer, opts Options) error {
	switch opts.Mode {
	case ModeSessionStart:
		return s.runSessionStart(ctx, w, opts.DataDir)
	case ModePrompt:
		return s.runPrompt(ctx, w, opts.DataDir, opts.Message)
	case ModeTool:
		return s.runTool(ctx, w, opts)
	default:
		return fmt.Errorf("%w: %s", ErrUnknownMode, opts.Mode)
	}
}

func (s *Surfacer) runPrompt(ctx context.Context, w io.Writer, dataDir, message string) error {
	memories, err := s.retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return fmt.Errorf("surface: %w", err)
	}

	type matchResult struct {
		mem      *memory.Stored
		keywords []string
	}

	var matches []matchResult

	lowerMessage := strings.ToLower(message)

	for _, mem := range memories {
		var matched []string

		for _, kw := range mem.Keywords {
			if matchesWholeWord(lowerMessage, strings.ToLower(kw)) {
				matched = append(matched, kw)
			}
		}

		for _, concept := range mem.Concepts {
			if matchesWholeWord(lowerMessage, strings.ToLower(concept)) {
				matched = append(matched, concept)
			}
		}

		if len(matched) > 0 {
			matches = append(matches, matchResult{mem: mem, keywords: matched})
		}
	}

	if len(matches) == 0 {
		return nil
	}

	_, _ = fmt.Fprintf(w, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(w, "[engram] Relevant memories:\n")

	for _, m := range matches {
		_, _ = fmt.Fprintf(w, "  - \"%s\" (%s) [matched: %s]\n",
			m.mem.Title, m.mem.FilePath, strings.Join(m.keywords, ", "))
	}

	_, _ = fmt.Fprintf(w, "</system-reminder>\n")

	return nil
}

func (s *Surfacer) runSessionStart(ctx context.Context, w io.Writer, dataDir string) error {
	memories, err := s.retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return fmt.Errorf("surface: %w", err)
	}

	if len(memories) == 0 {
		return nil
	}

	// Take top N by recency (already sorted by retriever).
	count := len(memories)
	if count > sessionStartLimit {
		count = sessionStartLimit
		memories = memories[:count]
	}

	_, _ = fmt.Fprintf(w, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(w, "[engram] Loaded %d memories.\n", count)

	for _, mem := range memories {
		_, _ = fmt.Fprintf(w, "  - \"%s\" (%s)\n", mem.Title, mem.FilePath)
	}

	_, _ = fmt.Fprintf(w, "</system-reminder>\n")

	return nil
}

func (s *Surfacer) runTool(ctx context.Context, w io.Writer, opts Options) error {
	memories, err := s.retriever.ListMemories(ctx, opts.DataDir)
	if err != nil {
		return fmt.Errorf("surface: %w", err)
	}

	// Pre-filter: only memories with anti_pattern and keyword match.
	candidates := matchToolMemories(opts.ToolName, opts.ToolInput, memories)

	if len(candidates) == 0 {
		return nil
	}

	if s.enforcer == nil {
		return nil
	}

	// LLM judgment for each candidate.
	for _, mem := range candidates {
		violated, judgeErr := s.enforcer.JudgeViolation(
			ctx, opts.ToolName, opts.ToolInput, mem, opts.Token)
		if judgeErr != nil {
			if s.stderr != nil {
				_, _ = fmt.Fprintf(
					s.stderr,
					"[engram] Warning: enforcement skipped (%v). Tool call allowed.\n",
					judgeErr,
				)
			}

			continue
		}

		if violated {
			// DES-7: block response format.
			_, _ = fmt.Fprintf(
				w,
				`{"decision": "block", "reason": "[engram] Blocked: \"%s\" — %s. Memory file: %s"}`,
				mem.Title,
				mem.Principle,
				mem.FilePath,
			)

			return nil
		}
	}

	return nil
}

// ToolEnforcer judges whether a tool call violates a memory's anti-pattern (ARCH-11).
type ToolEnforcer interface {
	JudgeViolation(ctx context.Context, toolName, toolInput string,
		mem *memory.Stored, token string) (violated bool, err error)
}

// unexported constants.
const (
	sessionStartLimit = 20
)

// matchToolMemories returns memories with non-empty anti_pattern that have at least
// one keyword matching in toolName or toolInput (ARCH-10).
func matchToolMemories(toolName, toolInput string, memories []*memory.Stored) []*memory.Stored {
	lowerInput := strings.ToLower(toolName + " " + toolInput)

	result := make([]*memory.Stored, 0)

	for _, mem := range memories {
		if mem.AntiPattern == "" {
			continue
		}

		for _, kw := range mem.Keywords {
			if matchesWholeWord(lowerInput, strings.ToLower(kw)) {
				result = append(result, mem)

				break
			}
		}
	}

	return result
}

// matchesWholeWord checks if keyword appears as a whole word in text (case-insensitive).
// Uses \b word boundary regex.
func matchesWholeWord(text, keyword string) bool {
	pattern := `\b` + regexp.QuoteMeta(keyword) + `\b`

	matched, err := regexp.MatchString(pattern, text)
	if err != nil {
		return false
	}

	return matched
}

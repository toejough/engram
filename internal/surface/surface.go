// Package surface implements memory surfacing for UC-2 (ARCH-12).
// Routes to SessionStart, UserPromptSubmit, or PreToolUse mode based on options.
package surface

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"engram/internal/memory"
)

// Exported constants.
const (
	FormatJSON       = "json"
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
	Format    string // output format: "" (plain) or "json"
}

// Result holds the structured output of a surface invocation.
type Result struct {
	Summary string `json:"summary"`
	Context string `json:"context"`
}

// Surfacer orchestrates memory surfacing.
type Surfacer struct {
	retriever MemoryRetriever
}

// New creates a Surfacer.
func New(retriever MemoryRetriever) *Surfacer {
	return &Surfacer{
		retriever: retriever,
	}
}

// Run executes the surface subcommand, writing output to w.
func (s *Surfacer) Run(ctx context.Context, w io.Writer, opts Options) error {
	var (
		result Result
		err    error
	)

	switch opts.Mode {
	case ModeSessionStart:
		result, err = s.runSessionStart(ctx, opts.DataDir)
	case ModePrompt:
		result, err = s.runPrompt(ctx, opts.DataDir, opts.Message)
	case ModeTool:
		result, err = s.runTool(ctx, opts)
	default:
		return fmt.Errorf("%w: %s", ErrUnknownMode, opts.Mode)
	}

	if err != nil {
		return err
	}

	if result.Context == "" {
		return nil
	}

	if opts.Format == FormatJSON {
		encodeErr := json.NewEncoder(w).Encode(result)
		if encodeErr != nil {
			return fmt.Errorf("surface: encoding JSON: %w", encodeErr)
		}

		return nil
	}

	_, _ = fmt.Fprint(w, result.Context)

	return nil
}

func (s *Surfacer) runPrompt(ctx context.Context, dataDir, message string) (Result, error) {
	memories, err := s.retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return Result{}, fmt.Errorf("surface: %w", err)
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
		return Result{}, nil
	}

	var buf strings.Builder

	_, _ = fmt.Fprintf(&buf, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(&buf, "[engram] Relevant memories:\n")

	for _, match := range matches {
		_, _ = fmt.Fprintf(&buf, "  - \"%s\" (%s) [matched: %s]\n",
			match.mem.Title, match.mem.FilePath, strings.Join(match.keywords, ", "))
	}

	_, _ = fmt.Fprintf(&buf, "</system-reminder>\n")

	promptMems := make([]*memory.Stored, 0, len(matches))
	for _, m := range matches {
		promptMems = append(promptMems, m.mem)
	}
	names := memoryNames(promptMems)
	summary := fmt.Sprintf("[engram] %d relevant memories: %s",
		len(matches), strings.Join(names, ", "))

	return Result{
		Summary: summary,
		Context: buf.String(),
	}, nil
}

func (s *Surfacer) runSessionStart(ctx context.Context, dataDir string) (Result, error) {
	memories, err := s.retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return Result{}, fmt.Errorf("surface: %w", err)
	}

	if len(memories) == 0 {
		return Result{}, nil
	}

	// Take top N by recency (already sorted by retriever).
	count := len(memories)
	if count > sessionStartLimit {
		count = sessionStartLimit
		memories = memories[:count]
	}

	var buf strings.Builder

	summary := fmt.Sprintf("[engram] Loaded %d memories: %s",
		count, strings.Join(memoryNames(memories), ", "))

	_, _ = fmt.Fprintf(&buf, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(&buf, "%s\n", summary)

	for _, mem := range memories {
		_, _ = fmt.Fprintf(&buf, "  - \"%s\" (%s)\n", mem.Title, mem.FilePath)
	}

	_, _ = fmt.Fprintf(&buf, "</system-reminder>\n")

	return Result{
		Summary: summary,
		Context: buf.String(),
	}, nil
}

func (s *Surfacer) runTool(ctx context.Context, opts Options) (Result, error) {
	memories, err := s.retriever.ListMemories(ctx, opts.DataDir)
	if err != nil {
		return Result{}, fmt.Errorf("surface: %w", err)
	}

	candidates := matchToolMemories(opts.ToolName, opts.ToolInput, memories)
	if len(candidates) == 0 {
		return Result{}, nil
	}

	var buf strings.Builder

	summary := fmt.Sprintf("[engram] %d tool advisories: %s",
		len(candidates), strings.Join(memoryNames(candidates), ", "))

	_, _ = fmt.Fprintf(&buf, "<system-reminder source=\"engram\">\n")
	_, _ = fmt.Fprintf(&buf, "[engram] Tool call advisory:\n")

	for _, mem := range candidates {
		_, _ = fmt.Fprintf(&buf, "  - \"%s\" — %s (%s)\n",
			mem.Title, mem.Principle, mem.FilePath)
	}

	_, _ = fmt.Fprintf(&buf, "</system-reminder>\n")

	return Result{
		Summary: summary,
		Context: buf.String(),
	}, nil
}

// unexported constants.
const (
	sessionStartLimit = 20
)

// memoryNames returns the basenames (without extension) of memory file paths.
func memoryNames(memories []*memory.Stored) []string {
	names := make([]string, 0, len(memories))
	for _, mem := range memories {
		name := filepath.Base(mem.FilePath)
		name = strings.TrimSuffix(name, filepath.Ext(name))
		names = append(names, name)
	}
	return names
}

// matchToolMemories returns memories with non-empty anti_pattern that have at least
// one keyword matching in toolName or toolInput (ARCH-10).
func matchToolMemories(_, toolInput string, memories []*memory.Stored) []*memory.Stored {
	lowerInput := strings.ToLower(toolInput)

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

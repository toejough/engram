package recall

import (
	"context"
	"fmt"
	"strings"
)

// Exported constants.
const (
	DefaultExtractCap  = 1500      // 1500 bytes of extracted content
	DefaultModeABudget = 15 * 1024 // 15KB for mode A raw transcript
	DefaultStripBudget = 50 * 1024 // 50KB per-session read budget (mode B)
)

// Finder finds session transcript files.
type Finder interface {
	Find(projectDir string) ([]string, error)
}

// MemorySurfacer surfaces relevant memories for a query.
type MemorySurfacer interface {
	Surface(query string) (string, error)
}

// Orchestrator composes the recall pipeline.
type Orchestrator struct {
	finder     Finder
	reader     Reader
	summarizer SummarizerI
	surfacer   MemorySurfacer
}

// NewOrchestrator creates an Orchestrator with the given collaborators.
func NewOrchestrator(
	finder Finder,
	reader Reader,
	summarizer SummarizerI,
	surfacer MemorySurfacer,
) *Orchestrator {
	return &Orchestrator{
		finder:     finder,
		reader:     reader,
		summarizer: summarizer,
		surfacer:   surfacer,
	}
}

// Recall executes the recall pipeline.
// If query is empty (mode A): find sessions, read+strip, return raw content.
// If query is non-empty (mode B): for each session, extract relevant content via LLM.
func (o *Orchestrator) Recall(
	ctx context.Context,
	projectDir, query string,
) (*Result, error) {
	sessions, err := o.finder.Find(projectDir)
	if err != nil {
		return nil, fmt.Errorf("recalling: %w", err)
	}

	if len(sessions) == 0 {
		return &Result{}, nil
	}

	if query == "" {
		return o.recallModeA(ctx, sessions)
	}

	return o.recallModeB(ctx, sessions, query)
}

func (o *Orchestrator) recallModeA(
	_ context.Context,
	sessions []string,
) (*Result, error) {
	var builder strings.Builder

	bytesRead := 0

	for _, path := range sessions {
		content, size, readErr := o.reader.Read(path, DefaultModeABudget-bytesRead)
		if readErr != nil {
			continue
		}

		builder.WriteString(content)

		bytesRead += size
		if bytesRead >= DefaultModeABudget {
			break
		}
	}

	accumulated := builder.String()
	memories := o.surfaceMemories(accumulated)

	return &Result{Summary: accumulated, Memories: memories}, nil
}

func (o *Orchestrator) recallModeB(
	ctx context.Context,
	sessions []string,
	query string,
) (*Result, error) {
	if o.summarizer == nil {
		return &Result{}, nil
	}

	var builder strings.Builder

	for _, path := range sessions {
		content, _, readErr := o.reader.Read(path, DefaultStripBudget)
		if readErr != nil {
			continue
		}

		extracted, extErr := o.summarizer.ExtractRelevant(ctx, content, query)
		if extErr != nil {
			continue
		}

		builder.WriteString(extracted)

		if builder.Len() >= DefaultExtractCap {
			break
		}
	}

	memories := o.surfaceMemories(query)

	return &Result{Summary: builder.String(), Memories: memories}, nil
}

func (o *Orchestrator) surfaceMemories(query string) string {
	if o.surfacer == nil {
		return ""
	}

	memories, err := o.surfacer.Surface(query)
	if err != nil {
		return ""
	}

	return memories
}

// Reader reads and strips a transcript file.
type Reader interface {
	Read(path string, budgetBytes int) (string, int, error)
}

// Result holds the output of a recall operation.
type Result struct {
	Summary  string `json:"summary"`
	Memories string `json:"memories,omitempty"`
}

// SummarizerI extracts relevant content from transcripts via LLM.
type SummarizerI interface {
	ExtractRelevant(ctx context.Context, content, query string) (string, error)
}

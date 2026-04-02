// Package correct implements the Remember & Correct pipeline (ARCH-1).
// It orchestrates detection, context gathering, BM25 retrieval, extraction,
// and disposition into a single Run call.
package correct

import (
	"context"
	"fmt"

	"engram/internal/bm25"
	"engram/internal/memory"
	"engram/internal/policy"
)

// Corrector orchestrates the correction pipeline: detect -> context -> BM25 -> extract -> disposition.
type Corrector struct {
	caller           CallerFunc
	transcriptReader TranscriptReaderFunc
	memoryRetriever  MemoryRetrieverFunc
	writer           MemoryWriter
	modifier         MemoryModifier
	pol              policy.Policy
}

// New creates a Corrector with the given options.
func New(opts ...Option) *Corrector {
	corrector := &Corrector{
		pol: policy.Defaults(),
	}

	for _, opt := range opts {
		opt(corrector)
	}

	return corrector
}

// Run executes the full correction pipeline.
//
//nolint:cyclop // Pipeline orchestrator has necessary sequential branching.
func (c *Corrector) Run(
	ctx context.Context,
	message, transcriptPath, dataDir, projectSlug string,
) (string, error) {
	// Step 1: Detect — fast-path keywords OR Haiku classification.
	isCorrection := DetectFastPath(message, c.pol.DetectFastPathKeywords)

	if !isCorrection {
		var err error

		isCorrection, err = DetectHaiku(ctx, c.caller, message, c.pol.DetectHaikuPrompt)
		if err != nil {
			return "", fmt.Errorf("detecting correction: %w", err)
		}
	}

	if !isCorrection {
		return "", nil
	}

	// Step 2: Context — read transcript tail if available.
	transcriptContext := ""

	if transcriptPath != "" && c.transcriptReader != nil {
		var err error

		transcriptContext, _, err = c.transcriptReader(transcriptPath, c.pol.ContextByteBudget)
		if err != nil {
			return "", fmt.Errorf("reading transcript: %w", err)
		}
	}

	// Step 3: BM25 candidates — retrieve memories and score.
	candidates, err := c.findCandidates(message, transcriptContext, dataDir)
	if err != nil {
		return "", err
	}

	// Step 4: Extract — call Sonnet with candidates.
	extraction, err := Extract(
		ctx, c.caller, message, transcriptContext, candidates, c.pol.ExtractSonnetPrompt,
	)
	if err != nil {
		return "", fmt.Errorf("extracting correction: %w", err)
	}

	// Step 5: Disposition — handle the extraction result.
	result, err := HandleDisposition(extraction, c.writer, c.modifier, dataDir, projectSlug)
	if err != nil {
		return "", fmt.Errorf("handling disposition: %w", err)
	}

	if result == nil {
		return "", nil
	}

	return formatResult(result), nil
}

// findCandidates retrieves all memories, scores them with BM25, and returns
// the top candidates above the threshold.
func (c *Corrector) findCandidates(
	message, transcriptContext, dataDir string,
) ([]*memory.Stored, error) {
	if c.memoryRetriever == nil {
		return nil, nil
	}

	allMemories, err := c.memoryRetriever(memory.MemoriesDir(dataDir))
	if err != nil {
		return nil, fmt.Errorf("retrieving memories: %w", err)
	}

	if len(allMemories) == 0 {
		return nil, nil
	}

	// Build BM25 documents from memories.
	documents := make([]bm25.Document, 0, len(allMemories))
	memoryByID := make(map[string]*memory.Stored, len(allMemories))

	for _, mem := range allMemories {
		doc := bm25.Document{
			ID:   mem.FilePath,
			Text: mem.SearchText(),
		}
		documents = append(documents, doc)
		memoryByID[mem.FilePath] = mem
	}

	// Score and filter.
	query := message + " " + transcriptContext
	scorer := bm25.New()
	scored := scorer.Score(query, documents)

	candidates := make([]*memory.Stored, 0, c.pol.ExtractCandidateCountMax)

	for _, scoredDoc := range scored {
		if scoredDoc.Score < c.pol.ExtractBM25Threshold {
			break
		}

		if len(candidates) >= c.pol.ExtractCandidateCountMax {
			break
		}

		if mem, ok := memoryByID[scoredDoc.ID]; ok {
			candidates = append(candidates, mem)
		}
	}

	return candidates, nil
}

// MemoryRetrieverFunc retrieves all stored memories from a directory.
type MemoryRetrieverFunc func(dir string) ([]*memory.Stored, error)

// Option configures a Corrector.
type Option func(*Corrector)

// TranscriptReaderFunc reads transcript context from a file path within a byte budget.
// Returns the context string, bytes read, and any error.
type TranscriptReaderFunc func(path string, budgetBytes int) (string, int, error)

// WithCaller sets the LLM caller function.
func WithCaller(caller CallerFunc) Option {
	return func(c *Corrector) { c.caller = caller }
}

// WithMemoryRetriever sets the memory retriever function.
func WithMemoryRetriever(retriever MemoryRetrieverFunc) Option {
	return func(c *Corrector) { c.memoryRetriever = retriever }
}

// WithModifier sets the memory modifier.
func WithModifier(modifier MemoryModifier) Option {
	return func(c *Corrector) { c.modifier = modifier }
}

// WithPolicy sets the policy configuration.
func WithPolicy(pol policy.Policy) Option {
	return func(c *Corrector) { c.pol = pol }
}

// WithTranscriptReader sets the transcript reader function.
func WithTranscriptReader(reader TranscriptReaderFunc) Option {
	return func(c *Corrector) { c.transcriptReader = reader }
}

// WithWriter sets the memory writer.
func WithWriter(writer MemoryWriter) Option {
	return func(c *Corrector) { c.writer = writer }
}

// formatResult builds a human-readable result string from a DispositionResult.
func formatResult(result *DispositionResult) string {
	if result.Reason != "" {
		return fmt.Sprintf("[engram] Correction %s: %s", result.Action, result.Reason)
	}

	return fmt.Sprintf("[engram] Correction %s: %s", result.Action, result.Path)
}

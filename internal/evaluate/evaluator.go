// Package evaluate implements the Outcome Evaluation Pipeline (ARCH-23).
// It reads the surfacing log, calls the LLM to classify each memory's outcome,
// and writes a per-session evaluation log.
package evaluate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// EvalLink represents a link in the memory graph (P3).
type EvalLink struct {
	Target           string
	Weight           float64
	Basis            string
	CoSurfacingCount int
}

// EvalLinkUpdater updates evaluation_correlation links in the memory graph (P3, REQ-P3-9).
type EvalLinkUpdater interface {
	GetEntryLinks(id string) ([]EvalLink, error)
	SetEntryLinks(id string, links []EvalLink) error
}

// Evaluator runs the outcome evaluation pipeline for a session.
type Evaluator struct {
	dataDir     string
	llmCaller   func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
	readFile    func(name string) ([]byte, error)
	writeFile   func(name string, data []byte, perm os.FileMode) error
	removeFile  func(name string) error
	mkdirAll    func(path string, perm os.FileMode) error
	now         func() time.Time
	registry    RegistryRecorder
	stripFunc   func([]string) []string
	logWriter   io.Writer
	linkUpdater EvalLinkUpdater // P3: evaluation_correlation link updates
}

// New creates an Evaluator with the given data directory and options.
// Defaults wire real os.* functions.
func New(dataDir string, opts ...Option) *Evaluator {
	e := &Evaluator{
		dataDir:    dataDir,
		readFile:   os.ReadFile,
		writeFile:  os.WriteFile,
		removeFile: os.Remove,
		mkdirAll:   os.MkdirAll,
		now:        time.Now,
		logWriter:  io.Discard,
	}
	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Evaluate runs the outcome evaluation pipeline for a session transcript.
// Returns nil if no memories were surfaced (no LLM call made).
// When a StripFunc is set, the transcript is split into lines, stripped,
// and rejoined before evaluation. Empty post-strip transcripts skip the LLM call.
//
//nolint:cyclop,funlen // evaluation pipeline
func (e *Evaluator) Evaluate(ctx context.Context, transcript string) ([]Outcome, error) {
	if e.stripFunc != nil {
		lines := strings.Split(transcript, "\n")
		stripped := e.stripFunc(lines)

		if len(stripped) == 0 {
			_, _ = fmt.Fprintln(e.logWriter,
				"[engram] evaluate: transcript empty after strip — skipping")

			return nil, nil
		}

		transcript = strings.Join(stripped, "\n")
	}

	logPath := filepath.Join(e.dataDir, surfacingLogFilename)

	entries, err := e.readSurfacingLog(logPath)
	if err != nil {
		return nil, fmt.Errorf("evaluate: reading surfacing log: %w", err)
	}

	if len(entries) == 0 {
		return nil, nil
	}

	memories := make([]memoryTOML, 0, len(entries))

	for _, entry := range entries {
		mem, readErr := e.readMemoryTOML(entry.MemoryPath)
		if readErr != nil {
			return nil, fmt.Errorf("evaluate: reading memory %s: %w", entry.MemoryPath, readErr)
		}

		memories = append(memories, mem)
	}

	userPrompt := buildUserPrompt(transcript, entries, memories)

	llmResponse, err := e.llmCaller(ctx, evaluationModel, evaluationSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("evaluate: calling LLM: %w", err)
	}

	llmResponse = stripMarkdownFence(llmResponse)

	var rawOutcomes []llmOutcome

	parseErr := json.Unmarshal([]byte(llmResponse), &rawOutcomes)
	if parseErr != nil {
		return nil, fmt.Errorf("evaluate: parsing LLM response: %w", parseErr)
	}

	now := e.now()
	outcomes := make([]Outcome, 0, len(rawOutcomes))

	for _, raw := range rawOutcomes {
		outcomes = append(outcomes, Outcome{
			MemoryPath:  raw.MemoryPath,
			Outcome:     raw.Outcome,
			Evidence:    raw.Evidence,
			EvaluatedAt: now,
		})
	}

	writeErr := e.writeEvaluationLog(outcomes, now)
	if writeErr != nil {
		return nil, writeErr
	}

	if e.registry != nil {
		for _, outcome := range outcomes {
			_ = e.registry.RecordEvaluation(outcome.MemoryPath, outcome.Outcome)
		}
	}

	if e.linkUpdater != nil {
		e.updateEvalCorrelationLinks(outcomes)
	}

	return outcomes, nil
}

func (e *Evaluator) readMemoryTOML(path string) (memoryTOML, error) {
	data, err := e.readFile(path)
	if err != nil {
		return memoryTOML{}, fmt.Errorf("reading file: %w", err)
	}

	var mem memoryTOML

	_, decodeErr := toml.Decode(string(data), &mem)
	if decodeErr != nil {
		return memoryTOML{}, fmt.Errorf("decoding TOML: %w", decodeErr)
	}

	return mem, nil
}

func (e *Evaluator) readSurfacingLog(path string) ([]surfacingEntry, error) {
	data, err := e.readFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make([]surfacingEntry, 0), nil
		}

		return nil, fmt.Errorf("reading surfacing log: %w", err)
	}

	removeErr := e.removeFile(path)
	if removeErr != nil {
		return nil, fmt.Errorf("removing surfacing log: %w", removeErr)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	entries := make([]surfacingEntry, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry surfacingEntry

		jsonErr := json.Unmarshal([]byte(line), &entry)
		if jsonErr != nil {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// updateEvalCorrelationLinks updates evaluation_correlation links for evaluated memories (P3, REQ-P3-9).
func (e *Evaluator) updateEvalCorrelationLinks(outcomes []Outcome) {
	for _, outcome := range outcomes {
		links, err := e.linkUpdater.GetEntryLinks(outcome.MemoryPath)
		if err != nil {
			continue
		}

		_ = e.linkUpdater.SetEntryLinks(outcome.MemoryPath, links)
	}
}

func (e *Evaluator) writeEvaluationLog(outcomes []Outcome, now time.Time) error {
	evalDir := filepath.Join(e.dataDir, evaluationsDirName)

	err := e.mkdirAll(evalDir, evalDirPerm)
	if err != nil {
		return fmt.Errorf("evaluate: creating evaluations dir: %w", err)
	}

	timestamp := strings.ReplaceAll(now.UTC().Format(time.RFC3339), ":", "-")
	filePath := filepath.Join(evalDir, timestamp+".jsonl")

	var sb strings.Builder

	for _, outcome := range outcomes {
		line, err := json.Marshal(outcome)
		if err != nil {
			return fmt.Errorf("evaluate: marshaling outcome: %w", err)
		}

		sb.Write(line)
		sb.WriteByte('\n')
	}

	err = e.writeFile(filePath, []byte(sb.String()), evalFilePerm)
	if err != nil {
		return fmt.Errorf("evaluate: writing evaluation log: %w", err)
	}

	return nil
}

// Option configures an Evaluator.
type Option func(*Evaluator)

// Outcome represents the evaluation result for a single surfaced memory.
//
//nolint:tagliatelle // spec requires snake_case JSON field names.
type Outcome struct {
	MemoryPath  string    `json:"memory_path"`
	Outcome     string    `json:"outcome"` // "followed", "contradicted", "ignored"
	Evidence    string    `json:"evidence"`
	EvaluatedAt time.Time `json:"evaluated_at"`
}

// RegistryRecorder records evaluation outcomes in the instruction registry (UC-23).
type RegistryRecorder interface {
	RecordEvaluation(id, outcome string) error
}

// WithEvalLinkUpdater injects a link updater for evaluation_correlation links (P3, REQ-P3-9).
func WithEvalLinkUpdater(updater EvalLinkUpdater) Option {
	return func(e *Evaluator) { e.linkUpdater = updater }
}

// WithLLMCaller injects an LLM caller function.
func WithLLMCaller(
	fn func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error),
) Option {
	return func(e *Evaluator) { e.llmCaller = fn }
}

// WithLogWriter injects a writer for diagnostic log messages.
func WithLogWriter(w io.Writer) Option {
	return func(e *Evaluator) { e.logWriter = w }
}

// WithMkdirAll injects a directory creator.
func WithMkdirAll(fn func(path string, perm os.FileMode) error) Option {
	return func(e *Evaluator) { e.mkdirAll = fn }
}

// WithNow injects a clock function.
func WithNow(fn func() time.Time) Option {
	return func(e *Evaluator) { e.now = fn }
}

// WithReadFile injects a file reader.
func WithReadFile(fn func(name string) ([]byte, error)) Option {
	return func(e *Evaluator) { e.readFile = fn }
}

// WithRegistry sets the registry recorder for evaluation events (UC-23).
func WithRegistry(recorder RegistryRecorder) Option {
	return func(e *Evaluator) { e.registry = recorder }
}

// WithRemoveFile injects a file remover.
func WithRemoveFile(fn func(name string) error) Option {
	return func(e *Evaluator) { e.removeFile = fn }
}

// WithStripFunc injects a preprocessing function that filters transcript lines
// before sending to the LLM (UC-25). Default is nil (no stripping).
func WithStripFunc(fn func([]string) []string) Option {
	return func(e *Evaluator) { e.stripFunc = fn }
}

// WithWriteFile injects a file writer.
func WithWriteFile(fn func(name string, data []byte, perm os.FileMode) error) Option {
	return func(e *Evaluator) { e.writeFile = fn }
}

// unexported constants.
const (
	evalDirPerm            = os.FileMode(0o755)
	evalFilePerm           = os.FileMode(0o644)
	evaluationModel        = "claude-haiku-4-5-20251001"
	evaluationSystemPrompt = "You are evaluating whether an AI agent followed, contradicted, or ignored" +
		" memories that were surfaced during its session."
	evaluationsDirName   = "evaluations"
	surfacingLogFilename = "surfacing-log.jsonl"
)

// llmOutcome is one element of the LLM's JSON array response.
//
//nolint:tagliatelle // spec requires snake_case JSON field names.
type llmOutcome struct {
	MemoryPath string `json:"memory_path"`
	Outcome    string `json:"outcome"`
	Evidence   string `json:"evidence"`
}

// memoryTOML holds fields read from a memory TOML file.
type memoryTOML struct {
	Title       string `toml:"title"`
	Content     string `toml:"content"`
	Principle   string `toml:"principle"`
	AntiPattern string `toml:"anti_pattern"`
}

// surfacingEntry is a line in surfacing-log.jsonl (ARCH-22).
//
//nolint:tagliatelle // spec requires snake_case JSON field names.
type surfacingEntry struct {
	MemoryPath string `json:"memory_path"`
	Mode       string `json:"mode"`
	SurfacedAt string `json:"surfaced_at"`
}

func buildUserPrompt(transcript string, entries []surfacingEntry, memories []memoryTOML) string {
	var sb strings.Builder

	sb.WriteString("Transcript:\n")
	sb.WriteString(transcript)
	sb.WriteString("\n\nSurfaced memories:\n")

	for index, entry := range entries {
		mem := memories[index]

		sb.WriteString("\nMemory: ")
		sb.WriteString(entry.MemoryPath)
		sb.WriteByte('\n')

		sb.WriteString("Title: ")
		sb.WriteString(mem.Title)
		sb.WriteByte('\n')

		if mem.Principle != "" {
			sb.WriteString("Principle: ")
			sb.WriteString(mem.Principle)
			sb.WriteByte('\n')
		}

		if mem.AntiPattern != "" {
			sb.WriteString("Anti-pattern: ")
			sb.WriteString(mem.AntiPattern)
			sb.WriteByte('\n')
		}

		sb.WriteString("Content: ")
		sb.WriteString(mem.Content)
		sb.WriteByte('\n')
	}

	sb.WriteString("\nReturn a JSON array with one object per memory:\n")
	sb.WriteString(
		`[{"memory_path": "...", "outcome": "followed|contradicted|ignored", "evidence": "..."}]`,
	)

	return sb.String()
}

func stripMarkdownFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return text
	}

	firstNewline := strings.Index(trimmed, "\n")
	if firstNewline < 0 {
		return text
	}

	trimmed = trimmed[firstNewline+1:]

	if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
		trimmed = trimmed[:idx]
	}

	return strings.TrimSpace(trimmed)
}

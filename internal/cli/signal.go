package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"engram/internal/effectiveness"
	"engram/internal/memory"
	regpkg "engram/internal/registry"
	"engram/internal/retrieve"
	reviewpkg "engram/internal/review"
	"engram/internal/signal"
)

// unexported constants.
const (
	signalQueueFilename = "signal-queue.jsonl"
)

// unexported variables.
var (
	errApplyProposalMissingFlags = errNew(
		"apply-proposal: --data-dir, --action, and --memory required",
	)
	errSignalDetectMissingFlags  = errNew("signal-detect: --data-dir required")
	errSignalSurfaceMissingFlags = errNew("signal-surface: --data-dir required")
)

// classifierAdapter adapts reviewpkg.Classify to signal.Classifier.
type classifierAdapter struct{}

func (c *classifierAdapter) Classify(
	stats map[string]effectiveness.Stat,
	tracking map[string]reviewpkg.TrackingData,
) []reviewpkg.ClassifiedMemory {
	return reviewpkg.Classify(stats, tracking)
}

// effectivenessReaderAdapter wraps a pre-computed stats map for the Consolidator.
type effectivenessReaderAdapter struct {
	stats map[string]effectiveness.Stat
}

func (e *effectivenessReaderAdapter) EffectivenessScore(
	path string,
) (float64, bool, error) {
	stat, ok := e.stats[path]
	if !ok {
		return 0, false, nil
	}

	return stat.EffectivenessScore, true, nil
}

// fileMergeExecutor merges two memories on disk (UC-34 / REQ-133 fallback).
type fileMergeExecutor struct {
	writer *storedMemoryWriter
	remove func(string) error
}

func (f *fileMergeExecutor) Merge(
	_ context.Context,
	survivor, absorbed *memory.Stored,
) error {
	unionKeywords(survivor, absorbed)
	unionConcepts(survivor, absorbed)
	keepLongerPrinciple(survivor, absorbed)

	writeErr := f.writer.Write(survivor.FilePath, survivor)
	if writeErr != nil {
		return fmt.Errorf("writing survivor: %w", writeErr)
	}

	removeErr := f.remove(absorbed.FilePath)
	if removeErr != nil {
		return fmt.Errorf("removing absorbed: %w", removeErr)
	}

	return nil
}

// memoryListerAdapter wraps retrieve.Retriever for the Consolidator.
type memoryListerAdapter struct {
	retriever *retrieve.Retriever
	dataDir   string
}

func (m *memoryListerAdapter) ListAll(ctx context.Context) ([]*memory.Stored, error) {
	memories, err := m.retriever.ListMemories(ctx, m.dataDir)
	if err != nil {
		return nil, fmt.Errorf("listing memories: %w", err)
	}

	return memories, nil
}

// memoryStoredLoader adapts file-based TOML reading to signal.MemoryLoader.
type memoryStoredLoader struct{}

func (l *memoryStoredLoader) Load(path string) (*memory.Stored, error) {
	return readStoredMemory(path)
}

// registryUpdaterAdapter adapts regpkg.Registry to signal.RegistryUpdater.
// dataDir is used to relativize absolute memory paths before passing to the
// TOML directory store, which expects IDs relative to dataDir.
type registryUpdaterAdapter struct {
	reg     regpkg.Registry
	dataDir string
}

func (r *registryUpdaterAdapter) Remove(id string) error {
	return r.reg.Remove(r.relID(id))
}

func (r *registryUpdaterAdapter) SetEnforcementLevel(id, level, reason string) error {
	return r.reg.SetEnforcementLevel(r.relID(id), regpkg.EnforcementLevel(level), reason)
}

func (r *registryUpdaterAdapter) UpdateContentHash(_, _ string) error {
	// Registry does not yet expose UpdateContentHash; this is a no-op stub.
	return nil
}

func (r *registryUpdaterAdapter) relID(id string) string {
	if r.dataDir == "" {
		return id
	}

	rel, err := filepath.Rel(r.dataDir, id)
	if err != nil {
		return id
	}

	return rel
}

// storedMemoryWriter writes a memory.Stored back to its TOML path atomically.
type storedMemoryWriter struct {
	createTemp func(dir, pattern string) (*os.File, error)
	rename     func(oldpath, newpath string) error
	remove     func(name string) error
}

func (w *storedMemoryWriter) Write(path string, stored *memory.Stored) error {
	type writeRecord struct {
		Title       string   `toml:"title"`
		Content     string   `toml:"content"`
		Concepts    []string `toml:"concepts"`
		Keywords    []string `toml:"keywords"`
		AntiPattern string   `toml:"anti_pattern"`
		Principle   string   `toml:"principle"`
		UpdatedAt   string   `toml:"updated_at"`
	}

	record := writeRecord{
		Title:       stored.Title,
		Content:     stored.Content,
		Concepts:    stored.Concepts,
		Keywords:    stored.Keywords,
		AntiPattern: stored.AntiPattern,
		Principle:   stored.Principle,
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	tmpFile, err := w.createTemp(filepath.Dir(path), "engram-mem-*.toml")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	tmpPath := tmpFile.Name()

	encodeErr := toml.NewEncoder(tmpFile).Encode(record)

	closeErr := tmpFile.Close()

	if encodeErr != nil {
		_ = w.remove(tmpPath)

		return fmt.Errorf("encoding TOML: %w", encodeErr)
	}

	if closeErr != nil {
		_ = w.remove(tmpPath)

		return fmt.Errorf("closing temp file: %w", closeErr)
	}

	renameErr := w.rename(tmpPath, path)
	if renameErr != nil {
		_ = w.remove(tmpPath)

		return fmt.Errorf("renaming temp file: %w", renameErr)
	}

	return nil
}

func errNew(s string) error {
	return fmt.Errorf("%s", s) //nolint:err113 // package-level sentinel via fmt.Errorf
}

func keepLongerPrinciple(survivor, absorbed *memory.Stored) {
	if len(absorbed.Principle) > len(survivor.Principle) {
		survivor.Principle = absorbed.Principle
	}
}

func newStoredMemoryWriter() *storedMemoryWriter {
	return &storedMemoryWriter{
		createTemp: os.CreateTemp,
		rename:     os.Rename,
		remove:     os.Remove,
	}
}

// readStoredMemory reads a memory TOML file into a memory.Stored.
func readStoredMemory(path string) (*memory.Stored, error) {
	type readRecord struct {
		Title       string   `toml:"title"`
		Content     string   `toml:"content"`
		Concepts    []string `toml:"concepts"`
		Keywords    []string `toml:"keywords"`
		AntiPattern string   `toml:"anti_pattern"`
		Principle   string   `toml:"principle"`
		UpdatedAt   string   `toml:"updated_at"`
	}

	data, err := os.ReadFile(path) //nolint:gosec // path is from trusted flag/internal source
	if err != nil {
		return nil, fmt.Errorf("reading memory file: %w", err)
	}

	var record readRecord

	_, decodeErr := toml.Decode(string(data), &record)
	if decodeErr != nil {
		return nil, fmt.Errorf("decoding memory TOML: %w", decodeErr)
	}

	return &memory.Stored{
		Title:       record.Title,
		Content:     record.Content,
		Concepts:    record.Concepts,
		Keywords:    record.Keywords,
		AntiPattern: record.AntiPattern,
		Principle:   record.Principle,
		FilePath:    path,
	}, nil
}

// runApplyProposal implements the apply-proposal subcommand (UC-28 Phase C).
//
//nolint:funlen // CLI flag parsing and DI wiring
func runApplyProposal(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("apply-proposal", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	action := fs.String("action", "", "action: remove, rewrite, broaden_keywords, escalate")
	memPath := fs.String("memory", "", "path to memory file")
	fieldsJSON := fs.String("fields", "", "JSON object of fields to update")
	keywordsStr := fs.String("keywords", "", "comma-separated keywords to add")
	level := fs.Int("level", 0, "escalation level")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("apply-proposal: %w", parseErr)
	}

	if *dataDir == "" || *action == "" || *memPath == "" {
		return errApplyProposalMissingFlags
	}

	var fields map[string]any
	if *fieldsJSON != "" {
		unmarshalErr := json.Unmarshal([]byte(*fieldsJSON), &fields)
		if unmarshalErr != nil {
			return fmt.Errorf("apply-proposal: parsing --fields: %w", unmarshalErr)
		}
	}

	var keywords []string
	if *keywordsStr != "" {
		keywords = strings.Split(*keywordsStr, ",")

		for i := range keywords {
			keywords[i] = strings.TrimSpace(keywords[i])
		}
	}

	queuePath := filepath.Join(*dataDir, signalQueueFilename)
	reg := openRegistry(*dataDir)
	queue := signal.NewQueueStore()

	regAdapter := &registryUpdaterAdapter{reg: reg, dataDir: *dataDir}

	gradStore := signal.NewGraduationStore()
	gradPath := filepath.Join(*dataDir, "graduation-queue.jsonl")
	gradEmitter := signal.NewGraduationQueueEmitter(gradStore, gradPath)

	applier := signal.NewApplier(
		signal.WithReadMemory(readStoredMemory),
		signal.WithWriteMemory(newStoredMemoryWriter()),
		signal.WithRegistry(regAdapter),
		signal.WithQueue(queue, queuePath),
		signal.WithEnforcementApplier(regAdapter),
		signal.WithGraduationEmitter(gradEmitter),
		signal.WithNow(time.Now),
	)

	ctx := context.Background()

	applyAction := signal.ApplyAction{
		Action:   *action,
		Memory:   *memPath,
		Fields:   fields,
		Keywords: keywords,
		Level:    *level,
	}

	result, applyErr := applier.Apply(ctx, applyAction)
	if applyErr != nil {
		result.Error = applyErr.Error()
	}

	//nolint:wrapcheck // thin JSON encoding at CLI boundary
	return json.NewEncoder(stdout).Encode(result)
}

// runSignalDetect implements the signal-detect subcommand (UC-28 Phase C).
//
//nolint:funlen // CLI flag parsing and DI wiring
func runSignalDetect(args []string) error {
	fs := flag.NewFlagSet("signal-detect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("signal-detect: %w", parseErr)
	}

	if *dataDir == "" {
		return errSignalDetectMissingFlags
	}

	evalDir := filepath.Join(*dataDir, "evaluations")

	stats, err := effectiveness.New(evalDir).Aggregate()
	if err != nil {
		return fmt.Errorf("signal-detect: aggregating effectiveness: %w", err)
	}

	ctx := context.Background()

	// UC-34: consolidate duplicates before classification.
	consolidator := signal.NewConsolidator(
		signal.WithLister(&memoryListerAdapter{
			retriever: retrieve.New(),
			dataDir:   *dataDir,
		}),
		signal.WithMerger(&fileMergeExecutor{
			writer: newStoredMemoryWriter(),
			remove: os.Remove,
		}),
		signal.WithEffectiveness(&effectivenessReaderAdapter{stats: stats}),
		signal.WithStderr(os.Stderr),
	)

	_, consolidateErr := consolidator.Consolidate(ctx)
	if consolidateErr != nil {
		return fmt.Errorf("signal-detect: consolidating: %w", consolidateErr)
	}

	tracking := buildTrackingMap(*dataDir)

	detector := signal.NewDetector(
		signal.WithClassifier(&classifierAdapter{}),
	)

	detected, detectErr := detector.Detect(ctx, stats, tracking)
	if detectErr != nil {
		return fmt.Errorf("signal-detect: detecting signals: %w", detectErr)
	}

	queuePath := filepath.Join(*dataDir, signalQueueFilename)
	store := signal.NewQueueStore()

	pruneErr := store.Prune(queuePath, func(path string) bool {
		_, statErr := os.Stat(path)
		return statErr == nil
	})
	if pruneErr != nil {
		return fmt.Errorf("signal-detect: pruning queue: %w", pruneErr)
	}

	if len(detected) == 0 {
		return nil
	}

	appendErr := store.Append(detected, queuePath)
	if appendErr != nil {
		return fmt.Errorf("signal-detect: appending to queue: %w", appendErr)
	}

	return nil
}

// runSignalSurface implements the signal-surface subcommand (UC-28 Phase C).
//
//nolint:funlen // CLI flag parsing and DI wiring
func runSignalSurface(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("signal-surface", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	format := fs.String("format", "text", "output format: text or json")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("signal-surface: %w", parseErr)
	}

	if *dataDir == "" {
		return errSignalSurfaceMissingFlags
	}

	queuePath := filepath.Join(*dataDir, signalQueueFilename)
	store := signal.NewQueueStore()

	signals, err := store.Read(queuePath)
	if err != nil {
		return fmt.Errorf("signal-surface: reading queue: %w", err)
	}

	if len(signals) == 0 {
		return nil
	}

	loader := &memoryStoredLoader{}
	surfacer := signal.NewSurfacer(signal.WithLoader(loader))

	enriched, enrichErr := surfacer.Surface(signals)
	if enrichErr != nil {
		return fmt.Errorf("signal-surface: enriching signals: %w", enrichErr)
	}

	context, fmtErr := signal.FormatContext(enriched)
	if fmtErr != nil {
		return fmt.Errorf("signal-surface: formatting context: %w", fmtErr)
	}

	if context == "" {
		return nil
	}

	if *format == formatJSON {
		type jsonOutput struct {
			Summary string `json:"summary"`
			Context string `json:"context"`
		}

		out := jsonOutput{
			Summary: fmt.Sprintf("[engram] %d pending maintenance signals", len(enriched)),
			Context: context,
		}

		//nolint:wrapcheck // thin JSON encoding at CLI boundary
		return json.NewEncoder(stdout).Encode(out)
	}

	_, _ = fmt.Fprint(stdout, context)

	return nil
}

func unionConcepts(survivor, absorbed *memory.Stored) {
	existing := make(map[string]struct{}, len(survivor.Concepts))

	for _, concept := range survivor.Concepts {
		existing[strings.ToLower(concept)] = struct{}{}
	}

	for _, concept := range absorbed.Concepts {
		if _, ok := existing[strings.ToLower(concept)]; !ok {
			survivor.Concepts = append(survivor.Concepts, concept)
			existing[strings.ToLower(concept)] = struct{}{}
		}
	}
}

func unionKeywords(survivor, absorbed *memory.Stored) {
	existing := make(map[string]struct{}, len(survivor.Keywords))

	for _, keyword := range survivor.Keywords {
		existing[strings.ToLower(keyword)] = struct{}{}
	}

	for _, keyword := range absorbed.Keywords {
		if _, ok := existing[strings.ToLower(keyword)]; !ok {
			survivor.Keywords = append(survivor.Keywords, keyword)
			existing[strings.ToLower(keyword)] = struct{}{}
		}
	}
}

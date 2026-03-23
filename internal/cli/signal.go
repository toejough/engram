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

	"engram/internal/crossref"
	"engram/internal/effectiveness"
	graph "engram/internal/graph"
	"engram/internal/memory"
	"engram/internal/merge"
	"engram/internal/retrieve"
	"engram/internal/signal"
)

// unexported variables.
var (
	errApplyProposalMissingFlags = errNew(
		"apply-proposal: --action and --memory required",
	)
)

// crossRefSourceEntry holds the ID and full text of one cross-source file.
type crossRefSourceEntry struct {
	id   string
	text string
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

// fileMergeExecutor computes the in-memory merge of two memories (UC-34).
// I/O (write, delete, backup) is handled by the Consolidator's DI seams.
type fileMergeExecutor struct{}

func (f *fileMergeExecutor) Merge(
	_ context.Context,
	survivor, absorbed *memory.Stored,
) error {
	unionKeywords(survivor, absorbed)
	unionConcepts(survivor, absorbed)
	keepLongerPrinciple(survivor, absorbed)

	return nil
}

// funcEnforcementApplier adapts a func to maintain.EnforcementApplier.
type funcEnforcementApplier struct {
	fn func(path, level, reason string) error
}

func (f *funcEnforcementApplier) SetEnforcementLevel(id, level, reason string) error {
	return f.fn(id, level, reason)
}

// graphLinkRecomputer implements signal.LinkRecomputer using graph.Builder (REQ-138).
//
// readStoredMemory calls os.ReadFile directly: internal/cli is the I/O wiring edge,
// so direct filesystem access in adapters here is intentional (not a DI violation).
type graphLinkRecomputer struct {
	builder *graph.Builder
	dataDir string
}

func (r *graphLinkRecomputer) RecomputeAfterMerge(survivorPath, absorbedPath string) error {
	survivor, err := readStoredMemory(survivorPath)
	if err != nil {
		return fmt.Errorf("reading survivor for link recompute: %w", err)
	}

	memoriesDir := filepath.Join(r.dataDir, "memories")
	lister := &memoryDirLister{dir: memoriesDir, dataDir: r.dataDir}
	writer := &readModifyWriteLinkWriter{dataDir: r.dataDir}

	// Relativize paths to dataDir so they match link target format in TOML files.
	survivorID := toRelID(r.dataDir, survivorPath)
	absorbedID := toRelID(r.dataDir, absorbedPath)

	result := graph.MergeResult{
		MergedMemoryID:   survivorID,
		AbsorbedMemoryID: absorbedID,
		MergedTitle:      survivor.Title,
		MergedContent:    survivor.Content,
		MergedConceptSet: survivor.Keywords,
	}

	return r.builder.RecomputeMergeLinks(result, lister, writer)
}

// llmPrincipleSynthesizer wraps merge.MemoryMerger to implement signal.PrincipleSynthesizer (REQ-139).
// Principles are folded left: merge(merge(p1, p2), p3), ...
type llmPrincipleSynthesizer struct {
	merger merge.MemoryMerger
}

func (s *llmPrincipleSynthesizer) SynthesizePrinciples(
	ctx context.Context,
	principles []string,
) (string, error) {
	if len(principles) == 0 {
		return "", nil
	}

	result := principles[0]

	for _, principle := range principles[1:] {
		merged, mergeErr := s.merger.MergePrinciples(ctx, result, principle)
		if mergeErr != nil {
			return "", fmt.Errorf("merging principles: %w", mergeErr)
		}

		result = merged
	}

	return result, nil
}

// memoryDirLister implements graph.MemoryLister by reading all TOML files from a directory.
// Paths are relativized to dataDir so they match the link target format stored in TOML.
type memoryDirLister struct {
	dir     string
	dataDir string
}

func (l *memoryDirLister) ListAll() ([]memory.StoredRecord, error) {
	records, err := memory.ListAll(l.dir)
	if err != nil {
		return nil, fmt.Errorf("listing memories: %w", err)
	}

	for i := range records {
		records[i].Path = toRelID(l.dataDir, records[i].Path)
	}

	return records, nil
}

// memoryListerAdapter wraps retrieve.Retriever for the Consolidator.
type memoryListerAdapter struct {
	retriever *retrieve.Retriever
	dataDir   string
}

func (m *memoryListerAdapter) ListAll(ctx context.Context) ([]*memory.Stored, error) {
	return m.retriever.ListMemories(ctx, m.dataDir)
}

// osBackupWriter copies absorbed memory files to a timestamped backup path (REQ-135).
type osBackupWriter struct {
	now func() time.Time
}

func (w *osBackupWriter) Backup(absorbedPath, backupDir string) error {
	const dirPerms = 0o755

	mkErr := os.MkdirAll(backupDir, dirPerms)
	if mkErr != nil {
		return fmt.Errorf("creating backup dir: %w", mkErr)
	}

	data, err := os.ReadFile(absorbedPath) //nolint:gosec // path from trusted internal source
	if err != nil {
		return fmt.Errorf("reading absorbed file for backup: %w", err)
	}

	ts := w.now().UTC().Format("20060102-150405.000000000")
	filename := ts + "-" + filepath.Base(absorbedPath)
	destPath := filepath.Join(backupDir, filename)

	const filePerms = 0o600

	writeErr := os.WriteFile(destPath, data, filePerms) //nolint:gosec // destPath uses filepath.Base, no traversal
	if writeErr != nil {
		return fmt.Errorf("writing backup: %w", writeErr)
	}

	return nil
}

// osFileDeleter deletes absorbed memory files after merge (REQ-136).
type osFileDeleter struct{}

func (d *osFileDeleter) Delete(path string) error {
	rmErr := os.Remove(path)
	if rmErr != nil {
		return fmt.Errorf("deleting absorbed file: %w", rmErr)
	}

	return nil
}

// readModifyWriteLinkWriter implements graph.LinkWriter using memory.ReadModifyWrite.
// Paths are relative to dataDir, so they are resolved back to absolute before writing.
type readModifyWriteLinkWriter struct {
	dataDir string
}

func (w *readModifyWriteLinkWriter) WriteLinks(path string, links []memory.LinkRecord) error {
	absPath := filepath.Join(w.dataDir, path)

	return memory.ReadModifyWrite(absPath, func(record *memory.MemoryRecord) {
		record.Links = links
	})
}

// sourceCrossRefChecker implements surface.CrossRefChecker using keyword overlap
// against pre-loaded CLAUDE.md, rule, and skill source texts (REQ-P4f-2).
type sourceCrossRefChecker struct {
	sources  []crossRefSourceEntry
	keywords map[string][]string // filePath → memory keywords
}

func (c *sourceCrossRefChecker) IsCoveredBySource(memPath string) (bool, string, error) {
	kws := c.keywords[memPath]
	if len(kws) == 0 {
		return false, "", nil
	}

	for _, src := range c.sources {
		lower := strings.ToLower(src.text)

		for _, kw := range kws {
			if strings.Contains(lower, strings.ToLower(kw)) {
				return true, src.id, nil
			}
		}
	}

	return false, "", nil
}

// storedMemoryWriter writes a memory.Stored back to its TOML path atomically.
type storedMemoryWriter struct {
	createTemp func(dir, pattern string) (*os.File, error)
	rename     func(oldpath, newpath string) error
	remove     func(name string) error
}

func (w *storedMemoryWriter) Write(path string, stored *memory.Stored) error {
	record := memory.MemoryRecord{
		Title:             stored.Title,
		Content:           stored.Content,
		Concepts:          stored.Concepts,
		Keywords:          stored.Keywords,
		AntiPattern:       stored.AntiPattern,
		Principle:         stored.Principle,
		SurfacedCount:     stored.SurfacedCount,
		FollowedCount:     stored.FollowedCount,
		ContradictedCount: stored.ContradictedCount,
		IgnoredCount:      stored.IgnoredCount,
		IrrelevantCount:   stored.IrrelevantCount,
		UpdatedAt:         time.Now().UTC().Format(time.RFC3339),
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

// loadCrossRefSources reads CLAUDE.md and rules/*.md from claudeDir.
func loadCrossRefSources(claudeDir string) []crossRefSourceEntry {
	var sources []crossRefSourceEntry

	// CLAUDE.md: extract as individual bullet entries for precise coverage.
	claudeMDPath := filepath.Join(claudeDir, "CLAUDE.md")

	data, readErr := os.ReadFile(claudeMDPath) //nolint:gosec // path from trusted CLI flag
	if readErr == nil {
		ext := crossref.ClaudeMDExtractor{Content: string(data), SourcePath: claudeMDPath}

		instrs, extractErr := ext.Extract()
		if extractErr != nil {
			// fire-and-forget diagnostic at the CLI wiring edge (ARCH-6).
			fmt.Fprintf(os.Stderr, "[engram] warning: parsing CLAUDE.md: %v\n", extractErr)
		}

		for _, instr := range instrs {
			sources = append(sources, crossRefSourceEntry{id: instr.SourcePath, text: instr.Content})
		}
	}

	// Rules: one entry per *.md file in <claudeDir>/rules/.
	rulesDir := filepath.Join(claudeDir, "rules")

	// filepath.Glob only fails with malformed patterns; ours is static.
	ruleFiles, _ := filepath.Glob(filepath.Join(rulesDir, "*.md"))

	for _, ruleFile := range ruleFiles {
		ruleData, ruleErr := os.ReadFile(ruleFile) //nolint:gosec // path from filepath.Glob within trusted dir
		if ruleErr != nil {
			continue
		}

		sources = append(sources, crossRefSourceEntry{
			id:   filepath.Base(ruleFile),
			text: string(ruleData),
		})
	}

	return sources
}

func newGraphLinkRecomputer(dataDir string) *graphLinkRecomputer {
	return &graphLinkRecomputer{
		builder: graph.New(),
		dataDir: dataDir,
	}
}

// newPrincipleSynthesizer returns an LLM-backed synthesizer when token is available,
// or nil to use the fallback (longest principle). REQ-139 AC5.
func newPrincipleSynthesizer(token string) signal.PrincipleSynthesizer {
	if token == "" {
		return nil
	}

	return &llmPrincipleSynthesizer{
		merger: merge.New(&cliLLMCaller{token: token}),
	}
}

// newSourceCrossRefChecker builds a real CrossRefChecker from claudeDir source files
// and the pre-loaded memory slice. Returns nil if no sources are found.
func newSourceCrossRefChecker(claudeDir string, memories []*memory.Stored) *sourceCrossRefChecker {
	sources := loadCrossRefSources(claudeDir)

	if len(sources) == 0 {
		return nil
	}

	keywords := make(map[string][]string, len(memories))

	for _, mem := range memories {
		keywords[mem.FilePath] = mem.Keywords
	}

	return &sourceCrossRefChecker{
		sources:  sources,
		keywords: keywords,
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
	data, err := os.ReadFile(path) //nolint:gosec // path is from trusted flag/internal source
	if err != nil {
		return nil, fmt.Errorf("reading memory file: %w", err)
	}

	var record memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &record)
	if decodeErr != nil {
		return nil, fmt.Errorf("decoding memory TOML: %w", decodeErr)
	}

	updatedAt, parseErr := time.Parse(time.RFC3339, record.UpdatedAt)
	if parseErr != nil {
		updatedAt = time.Time{}
	}

	return &memory.Stored{
		Title:             record.Title,
		Content:           record.Content,
		Concepts:          record.Concepts,
		Keywords:          record.Keywords,
		AntiPattern:       record.AntiPattern,
		Principle:         record.Principle,
		SurfacedCount:     record.SurfacedCount,
		FollowedCount:     record.FollowedCount,
		ContradictedCount: record.ContradictedCount,
		IgnoredCount:      record.IgnoredCount,
		IrrelevantCount:   record.IrrelevantCount,
		UpdatedAt:         updatedAt,
		FilePath:          path,
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

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("apply-proposal: %w", defaultErr)
	}

	if *action == "" || *memPath == "" {
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

	enforcementFunc := func(path, level, _ string) error {
		return memory.ReadModifyWrite(path, func(record *memory.MemoryRecord) {
			record.EnforcementLevel = level
		})
	}

	applier := signal.NewApplier(
		signal.WithReadMemory(readStoredMemory),
		signal.WithWriteMemory(newStoredMemoryWriter()),
		signal.WithEnforcementApplier(&funcEnforcementApplier{fn: enforcementFunc}),
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

func toRelID(dataDir, absPath string) string {
	rel, err := filepath.Rel(dataDir, absPath)
	if err != nil {
		return absPath
	}

	return rel
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

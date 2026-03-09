// Package register orchestrates auto-registration of non-memory instruction
// sources into the unified instruction registry (ARCH-69).
package register

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"engram/internal/registry"
)

// Option configures a Registrar.
type Option func(*Registrar)

// Registrar orchestrates discovery, registration, pruning, and implicit
// surfacing of non-memory instruction sources.
type Registrar struct {
	registry     Registry
	surfacingLog SurfacingLogger
	readFile     func(string) ([]byte, error)
	readDir      func(string) ([]os.DirEntry, error)
	glob         func(string) ([]string, error)
	now          func() time.Time
	stderr       io.Writer
}

// NewRegistrar creates a Registrar with the given registry and surfacing
// logger, applying any functional options. Defaults use real os.* functions.
func NewRegistrar(reg Registry, logger SurfacingLogger, opts ...Option) *Registrar {
	r := &Registrar{
		registry:     reg,
		surfacingLog: logger,
		readFile:     os.ReadFile,
		readDir:      os.ReadDir,
		glob:         filepath.Glob,
		now:          time.Now,
		stderr:       os.Stderr,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Run executes the 4-phase registration pipeline:
// 1. Discover sources, 2. Register/update, 3. Prune stale, 4. Record surfacing.
func (r *Registrar) Run(config SourceConfig) error {
	// Phase 1: Discover.
	discovered := r.discover(config)

	// Phase 2: Register / update.
	r.registerEntries(discovered)

	// Phase 3: Prune stale non-memory entries.
	r.pruneStale(discovered)

	// Phase 4: Record implicit surfacing.
	r.recordSurfacing(discovered)

	return nil
}

// discover extracts instruction entries from all configured sources.
func (r *Registrar) discover(config SourceConfig) []registry.InstructionEntry {
	discovered := make([]registry.InstructionEntry, 0, 4) //nolint:mnd // four source types

	discovered = append(discovered, r.discoverClaudeMD(config.ClaudeMDPaths)...)
	discovered = append(discovered, r.discoverMemoryMD(config.MemoryMDPaths)...)
	discovered = append(discovered, r.discoverRules(config.RulesDir)...)
	discovered = append(discovered, r.discoverSkills(config.SkillsDir)...)

	return discovered
}

// discoverClaudeMD reads and extracts entries from CLAUDE.md files.
func (r *Registrar) discoverClaudeMD(paths []string) []registry.InstructionEntry {
	var result []registry.InstructionEntry

	for _, path := range paths {
		content, err := r.readFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			r.logErrorf("reading claude-md %s: %v", path, err)

			continue
		}

		extractor := registry.ClaudeMDExtractor{
			Content:    string(content),
			SourcePath: path,
		}

		entries, extractErr := extractor.Extract()
		if extractErr != nil {
			r.logErrorf("extracting claude-md %s: %v", path, extractErr)

			continue
		}

		// Override timestamps with injected now.
		now := r.now()
		for idx := range entries {
			entries[idx].RegisteredAt = now
			entries[idx].UpdatedAt = now
		}

		result = append(result, entries...)
	}

	return result
}

// discoverMemoryMD reads and extracts entries from MEMORY.md files.
func (r *Registrar) discoverMemoryMD(paths []string) []registry.InstructionEntry {
	var result []registry.InstructionEntry

	for _, path := range paths {
		content, err := r.readFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			r.logErrorf("reading memory-md %s: %v", path, err)

			continue
		}

		extractor := registry.MemoryMDExtractor{
			Content:    string(content),
			SourcePath: path,
		}

		entries, extractErr := extractor.Extract()
		if extractErr != nil {
			r.logErrorf("extracting memory-md %s: %v", path, extractErr)

			continue
		}

		now := r.now()
		for idx := range entries {
			entries[idx].RegisteredAt = now
			entries[idx].UpdatedAt = now
		}

		result = append(result, entries...)
	}

	return result
}

// discoverRules reads rule files from the rules directory.
func (r *Registrar) discoverRules(rulesDir string) []registry.InstructionEntry {
	if rulesDir == "" {
		return nil
	}

	dirEntries, err := r.readDir(rulesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		r.logErrorf("reading rules dir %s: %v", rulesDir, err)

		return nil
	}

	var result []registry.InstructionEntry

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}

		filePath := rulesDir + "/" + dirEntry.Name()

		content, readErr := r.readFile(filePath)
		if readErr != nil {
			r.logErrorf("reading rule %s: %v", filePath, readErr)

			continue
		}

		extractor := registry.RuleExtractor{
			Filename: dirEntry.Name(),
			Content:  string(content),
		}

		entries, extractErr := extractor.Extract()
		if extractErr != nil {
			r.logErrorf("extracting rule %s: %v", filePath, extractErr)

			continue
		}

		// Override timestamps with injected now.
		now := r.now()
		for idx := range entries {
			entries[idx].RegisteredAt = now
			entries[idx].UpdatedAt = now
		}

		result = append(result, entries...)
	}

	return result
}

// discoverSkills finds skill directories and extracts entries from SKILL.md files.
func (r *Registrar) discoverSkills(skillsDir string) []registry.InstructionEntry {
	if skillsDir == "" {
		return nil
	}

	pattern := skillsDir + "/*/SKILL.md"

	matches, err := r.glob(pattern)
	if err != nil {
		r.logErrorf("globbing skills %s: %v", pattern, err)

		return nil
	}

	var result []registry.InstructionEntry

	for _, matchPath := range matches {
		content, readErr := r.readFile(matchPath)
		if readErr != nil {
			r.logErrorf("reading skill %s: %v", matchPath, readErr)

			continue
		}

		// Extract skill name from path: /skills/<name>/SKILL.md
		dir := filepath.Dir(matchPath)
		skillName := filepath.Base(dir)

		extractor := registry.SkillExtractor{
			SkillName: skillName,
			Content:   string(content),
		}

		entries, extractErr := extractor.Extract()
		if extractErr != nil {
			r.logErrorf("extracting skill %s: %v", matchPath, extractErr)

			continue
		}

		now := r.now()
		for idx := range entries {
			entries[idx].RegisteredAt = now
			entries[idx].UpdatedAt = now
		}

		result = append(result, entries...)
	}

	return result
}

// logErrorf writes an error message to stderr without failing the pipeline.
func (r *Registrar) logErrorf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}

	_, _ = fmt.Fprintf(r.stderr, "engram: register: %s", msg)
}

// pruneStale removes non-memory entries from the registry that are not in the
// discovered set.
func (r *Registrar) pruneStale(discovered []registry.InstructionEntry) {
	// Build set of discovered IDs.
	discoveredIDs := make(map[string]struct{}, len(discovered))
	for _, entry := range discovered {
		discoveredIDs[entry.ID] = struct{}{}
	}

	existing, err := r.registry.List()
	if err != nil {
		r.logErrorf("listing for prune: %v", err)

		return
	}

	for _, entry := range existing {
		// Never prune memory entries.
		if entry.SourceType == sourceTypeMemory {
			continue
		}

		if _, found := discoveredIDs[entry.ID]; found {
			continue
		}

		removeErr := r.registry.Remove(entry.ID)
		if removeErr != nil {
			r.logErrorf("pruning %s: %v", entry.ID, removeErr)
		}
	}
}

// recordSurfacing records implicit surfacing for all discovered (always-loaded)
// entries via both registry and surfacing logger.
func (r *Registrar) recordSurfacing(discovered []registry.InstructionEntry) {
	now := r.now()

	for _, entry := range discovered {
		surfErr := r.registry.RecordSurfacing(entry.ID)
		if surfErr != nil {
			r.logErrorf("recording surfacing %s: %v", entry.ID, surfErr)
		}

		logErr := r.surfacingLog.LogSurfacing(entry.ID, "session-start", now)
		if logErr != nil {
			r.logErrorf("logging surfacing %s: %v", entry.ID, logErr)
		}
	}
}

// registerEntries registers new entries and updates changed ones.
func (r *Registrar) registerEntries(discovered []registry.InstructionEntry) {
	for _, entry := range discovered {
		existing, err := r.registry.Get(entry.ID)
		if err != nil {
			if !errors.Is(err, registry.ErrNotFound) {
				r.logErrorf("getting entry %s: %v", entry.ID, err)

				continue
			}

			// New entry — register it.
			regErr := r.registry.Register(entry)
			if regErr != nil {
				r.logErrorf("registering %s: %v", entry.ID, regErr)
			}

			continue
		}

		// Entry exists — check if content changed.
		if existing.ContentHash == entry.ContentHash {
			continue // same content, skip
		}

		// Content changed — update: preserve counters, remove + re-register.
		entry.SurfacedCount = existing.SurfacedCount
		entry.LastSurfaced = existing.LastSurfaced
		entry.Evaluations = existing.Evaluations
		entry.Absorbed = existing.Absorbed
		entry.RegisteredAt = existing.RegisteredAt

		removeErr := r.registry.Remove(entry.ID)
		if removeErr != nil {
			r.logErrorf("removing for update %s: %v", entry.ID, removeErr)

			continue
		}

		regErr := r.registry.Register(entry)
		if regErr != nil {
			r.logErrorf("re-registering %s: %v", entry.ID, regErr)
		}
	}
}

// Registry is the subset of registry.Registry needed by Registrar.
type Registry interface {
	Register(entry registry.InstructionEntry) error
	RecordSurfacing(id string) error
	Remove(id string) error
	List() ([]registry.InstructionEntry, error)
	Get(id string) (*registry.InstructionEntry, error)
}

// SourceConfig defines the file paths to scan for instruction sources.
type SourceConfig struct {
	ClaudeMDPaths []string
	MemoryMDPaths []string
	RulesDir      string
	SkillsDir     string
}

// SurfacingLogger logs surfacing events for the evaluate pipeline.
type SurfacingLogger interface {
	LogSurfacing(memoryPath, mode string, timestamp time.Time) error
}

// WithGlob injects a glob function.
func WithGlob(fn func(string) ([]string, error)) Option {
	return func(r *Registrar) { r.glob = fn }
}

// WithNow injects a time provider function.
func WithNow(fn func() time.Time) Option {
	return func(r *Registrar) { r.now = fn }
}

// WithReadDir injects a directory reader function.
func WithReadDir(fn func(string) ([]os.DirEntry, error)) Option {
	return func(r *Registrar) { r.readDir = fn }
}

// WithReadFile injects a file reader function.
func WithReadFile(fn func(string) ([]byte, error)) Option {
	return func(r *Registrar) { r.readFile = fn }
}

// WithStderr injects a writer for error logging.
func WithStderr(w io.Writer) Option {
	return func(r *Registrar) { r.stderr = w }
}

// unexported constants.
const (
	sourceTypeMemory = "memory"
)

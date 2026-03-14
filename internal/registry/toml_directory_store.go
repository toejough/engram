package registry

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
)

// Unexported constants.
const (
	memoriesSubdir = "memories"
	tomlExt        = ".toml"
	tmpSuffix      = ".tmp"
	tomlFilePerm   = os.FileMode(0o644)
	tomlDirPerm    = os.FileMode(0o750)
)

// TOMLDirOption configures a TOMLDirectoryStore.
type TOMLDirOption func(*TOMLDirectoryStore)

// TOMLDirectoryStore implements Registry backed by per-file TOML documents.
// Each memory's TOML file contains both content and embedded runtime metrics.
// All I/O is performed through injected functions. Per-file flock locking
// serializes concurrent writes to the same file without global contention.
type TOMLDirectoryStore struct {
	dataDir    string
	readFile   func(name string) ([]byte, error)
	writeFile  func(name string, data []byte, perm os.FileMode) error
	readDir    func(name string) ([]os.DirEntry, error)
	remove     func(name string) error
	rename     func(oldpath, newpath string) error
	mkdirAll   func(path string, perm os.FileMode) error
	openFile   func(name string, flag int, perm os.FileMode) (*os.File, error)
	lockFile   func(f *os.File) error
	unlockFile func(f *os.File) error
	nowFn      func() time.Time
}

// NewTOMLDirectoryStore creates a new TOML directory-backed registry store.
func NewTOMLDirectoryStore(dataDir string, opts ...TOMLDirOption) *TOMLDirectoryStore {
	store := &TOMLDirectoryStore{
		dataDir:    dataDir,
		readFile:   os.ReadFile,
		writeFile:  os.WriteFile,
		readDir:    os.ReadDir,
		remove:     os.Remove,
		rename:     os.Rename,
		mkdirAll:   os.MkdirAll,
		openFile:   os.OpenFile,
		lockFile:   flockExclusive,
		unlockFile: flockUnlock,
	}

	for _, opt := range opts {
		opt(store)
	}

	return store
}

// Get returns a single entry by ID (relative path from dataDir).
func (s *TOMLDirectoryStore) Get(id string) (*InstructionEntry, error) {
	absPath := filepath.Join(s.dataDir, id)

	data, err := s.readFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	var record memoryRecord

	if _, decodeErr := toml.Decode(string(data), &record); decodeErr != nil {
		return nil, fmt.Errorf("decoding TOML for %s: %w", id, decodeErr)
	}

	entry := recordToEntry(id, record)

	return &entry, nil
}

// List scans the memories directory and returns all InstructionEntry values.
func (s *TOMLDirectoryStore) List() ([]InstructionEntry, error) {
	memoriesDir := filepath.Join(s.dataDir, memoriesSubdir)

	dirEntries, err := s.readDir(memoriesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return make([]InstructionEntry, 0), nil
		}

		return nil, fmt.Errorf("reading memories directory: %w", err)
	}

	entries := make([]InstructionEntry, 0, len(dirEntries))

	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), tomlExt) {
			continue
		}

		id := memoriesSubdir + "/" + de.Name()

		entry, getErr := s.Get(id)
		if getErr != nil {
			continue // skip unreadable files
		}

		entries = append(entries, *entry)
	}

	return entries, nil
}

// Register writes a new TOML file for the given entry.
// Returns ErrDuplicateID if a file already exists at entry.ID.
func (s *TOMLDirectoryStore) Register(entry InstructionEntry) error {
	absPath := filepath.Join(s.dataDir, entry.ID)

	// Check if file already exists.
	if _, readErr := s.readFile(absPath); readErr == nil {
		return fmt.Errorf("%w: %s", ErrDuplicateID, entry.ID)
	}

	if mkdirErr := s.mkdirAll(filepath.Dir(absPath), tomlDirPerm); mkdirErr != nil {
		return fmt.Errorf("creating directory for %s: %w", entry.ID, mkdirErr)
	}

	if entry.EnforcementLevel == "" {
		entry.EnforcementLevel = EnforcementAdvisory
	}

	if entry.SourceType == "" {
		entry.SourceType = SourceTypeMemory
	}

	record := entryToRecord(entry)

	return s.writeAtomic(absPath, record)
}

// RecordSurfacing increments the surfaced_count and sets last_surfaced_at.
func (s *TOMLDirectoryStore) RecordSurfacing(id string) error {
	return s.withFileLocked(id, func(record *memoryRecord) error {
		record.SurfacedCount++
		record.LastSurfacedAt = s.now().UTC().Format(time.RFC3339)

		return nil
	})
}

// RecordEvaluation increments the appropriate evaluation counter.
func (s *TOMLDirectoryStore) RecordEvaluation(id string, outcome Outcome) error {
	return s.withFileLocked(id, func(record *memoryRecord) error {
		switch outcome {
		case Followed:
			record.FollowedCount++
		case Contradicted:
			record.ContradictedCount++
		case Ignored:
			record.IgnoredCount++
		}

		return nil
	})
}

// SetEnforcementLevel updates the enforcement level and records the transition.
func (s *TOMLDirectoryStore) SetEnforcementLevel(id string, level EnforcementLevel, reason string) error {
	return s.withFileLocked(id, func(record *memoryRecord) error {
		current := EnforcementLevel(record.EnforcementLevel)
		if current == "" {
			current = EnforcementAdvisory
		}

		if current == level {
			return nil
		}

		transition := transitionTOML{
			From:   string(current),
			To:     string(level),
			At:     s.now().UTC().Format(time.RFC3339),
			Reason: reason,
		}

		record.Transitions = append(record.Transitions, transition)
		record.EnforcementLevel = string(level)

		return nil
	})
}

// Merge absorbs the source entry into the target entry, then removes the source.
func (s *TOMLDirectoryStore) Merge(sourceID, targetID string) error {
	sourcePath := filepath.Join(s.dataDir, sourceID)

	// Read source.
	sourceData, readErr := s.readFile(sourcePath)
	if readErr != nil {
		return fmt.Errorf("%w: source=%s target=%s", ErrMergeNotFound, sourceID, targetID)
	}

	var sourceRecord memoryRecord

	if _, decodeErr := toml.Decode(string(sourceData), &sourceRecord); decodeErr != nil {
		return fmt.Errorf("decoding source %s: %w", sourceID, decodeErr)
	}

	sourceType := sourceRecord.SourceType
	if sourceType == "" {
		sourceType = SourceTypeMemory
	}

	if sourceType != SourceTypeMemory {
		return fmt.Errorf("%w: source=%s target=%s", ErrMergeSourceType, sourceID, targetID)
	}

	// Modify target under flock.
	modErr := s.withFileLocked(targetID, func(record *memoryRecord) error {
		targetType := record.SourceType
		if targetType == "" {
			targetType = SourceTypeMemory
		}

		if targetType != SourceTypeMemory {
			return fmt.Errorf("%w: source=%s target=%s", ErrMergeSourceType, sourceID, targetID)
		}

		absorbed := absorbedTOML{
			From:          sourceID,
			SurfacedCount: sourceRecord.SurfacedCount,
			ContentHash:   sourceRecord.ContentHash,
			MergedAt:      s.now().UTC().Format(time.RFC3339),
			Evaluations: evalTOML{
				Followed:     sourceRecord.FollowedCount,
				Contradicted: sourceRecord.ContradictedCount,
				Ignored:      sourceRecord.IgnoredCount,
			},
		}

		record.Absorbed = append(record.Absorbed, absorbed)
		record.SurfacedCount += sourceRecord.SurfacedCount
		record.FollowedCount += sourceRecord.FollowedCount
		record.ContradictedCount += sourceRecord.ContradictedCount
		record.IgnoredCount += sourceRecord.IgnoredCount

		return nil
	})
	if modErr != nil {
		if errors.Is(modErr, ErrNotFound) {
			return fmt.Errorf("%w: source=%s target=%s", ErrMergeNotFound, sourceID, targetID)
		}

		return modErr
	}

	// Delete source after successful target update.
	if err := s.remove(sourcePath); err != nil {
		return fmt.Errorf("removing source %s: %w", sourceID, err)
	}

	return nil
}

// Remove deletes the TOML file for the given ID.
func (s *TOMLDirectoryStore) Remove(id string) error {
	absPath := filepath.Join(s.dataDir, id)

	if _, err := s.readFile(absPath); err != nil {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	if err := s.remove(absPath); err != nil {
		return fmt.Errorf("removing %s: %w", id, err)
	}

	return nil
}

// UpdateLinks replaces the links array in the TOML file.
func (s *TOMLDirectoryStore) UpdateLinks(id string, links []Link) error {
	return s.withFileLocked(id, func(record *memoryRecord) error {
		if len(links) == 0 {
			record.Links = nil
			return nil
		}

		linkRecords := make([]linkTOML, len(links))
		for i, link := range links {
			linkRecords[i] = linkTOML(link)
		}

		record.Links = linkRecords

		return nil
	})
}

// --- Private helpers ---

// withFileLocked acquires an exclusive flock on a stable sidecar lock file,
// reads the data file, calls fn with the decoded record, then writes atomically.
//
// A sidecar .lock file (not the data file) is used for locking so that the
// inode is stable across atomic renames. All goroutines contending on the same
// id block on the same inode regardless of how many rename cycles have occurred.
func (s *TOMLDirectoryStore) withFileLocked(id string, fn func(*memoryRecord) error) error {
	absPath := filepath.Join(s.dataDir, id)
	lockPath := absPath + ".lock"

	// Check that the data file exists before acquiring the lock.
	if _, existErr := s.readFile(absPath); existErr != nil {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	// Open (or create) the sidecar lock file. Its inode is never renamed.
	f, openErr := s.openFile(lockPath, os.O_RDWR|os.O_CREATE, tomlFilePerm)
	if openErr != nil {
		return fmt.Errorf("opening lock file for %s: %w", id, openErr)
	}

	defer func() { _ = f.Close() }()

	if lockErr := s.lockFile(f); lockErr != nil {
		return fmt.Errorf("acquiring lock on %s: %w", id, lockErr)
	}

	defer func() { _ = s.unlockFile(f) }()

	// Read after acquiring lock to get the latest state.
	data, readErr := s.readFile(absPath)
	if readErr != nil {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	var record memoryRecord

	if _, decodeErr := toml.Decode(string(data), &record); decodeErr != nil {
		return fmt.Errorf("decoding TOML for %s: %w", id, decodeErr)
	}

	if err := fn(&record); err != nil {
		return err
	}

	return s.writeAtomic(absPath, record)
}

// writeAtomic encodes record as TOML and renames a temp file into place.
func (s *TOMLDirectoryStore) writeAtomic(path string, record memoryRecord) error {
	var buf bytes.Buffer

	if err := toml.NewEncoder(&buf).Encode(record); err != nil {
		return fmt.Errorf("encoding TOML for %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	tmpPath := filepath.Join(dir, filepath.Base(path)+tmpSuffix)

	if err := s.writeFile(tmpPath, buf.Bytes(), tomlFilePerm); err != nil {
		return fmt.Errorf("writing temp file for %s: %w", path, err)
	}

	if err := s.rename(tmpPath, path); err != nil {
		_ = s.remove(tmpPath) // best-effort cleanup

		return fmt.Errorf("renaming to final %s: %w", path, err)
	}

	return nil
}

func (s *TOMLDirectoryStore) now() time.Time {
	if s.nowFn != nil {
		return s.nowFn()
	}

	return time.Now()
}

// flockExclusive acquires an exclusive flock on f.
func flockExclusive(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX) //nolint:wrapcheck // thin syscall wrapper
}

// flockUnlock releases the flock on f.
func flockUnlock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:wrapcheck // thin syscall wrapper
}

// --- TOML record types ---

// memoryRecord is the on-disk TOML representation of a memory with embedded metrics.
type memoryRecord struct {
	// Content fields.
	Title           string   `toml:"title"`
	Content         string   `toml:"content,omitempty"`
	ObservationType string   `toml:"observation_type,omitempty"`
	Concepts        []string `toml:"concepts,omitempty"`
	Keywords        []string `toml:"keywords,omitempty"`
	Principle       string   `toml:"principle,omitempty"`
	AntiPattern     string   `toml:"anti_pattern,omitempty"`
	Rationale       string   `toml:"rationale,omitempty"`
	Confidence      string   `toml:"confidence,omitempty"`
	CreatedAt       string   `toml:"created_at,omitempty"`
	UpdatedAt       string   `toml:"updated_at,omitempty"`
	SourceType      string   `toml:"source_type,omitempty"`
	SourcePath      string   `toml:"source_path,omitempty"`

	// Registry metrics.
	SurfacedCount     int    `toml:"surfaced_count"`
	FollowedCount     int    `toml:"followed_count"`
	ContradictedCount int    `toml:"contradicted_count"`
	IgnoredCount      int    `toml:"ignored_count"`
	LastSurfacedAt    string `toml:"last_surfaced_at,omitempty"`
	EnforcementLevel  string `toml:"enforcement_level,omitempty"`
	ContentHash       string `toml:"content_hash,omitempty"`

	// Relationships.
	Links       []linkTOML       `toml:"links,omitempty"`
	Absorbed    []absorbedTOML   `toml:"absorbed,omitempty"`
	Transitions []transitionTOML `toml:"transitions,omitempty"`
}

type linkTOML struct {
	Target           string  `toml:"target"`
	Weight           float64 `toml:"weight"`
	Basis            string  `toml:"basis"`
	CoSurfacingCount int     `toml:"co_surfacing_count,omitempty"`
}

type absorbedTOML struct {
	From          string   `toml:"from"`
	SurfacedCount int      `toml:"surfaced_count"`
	ContentHash   string   `toml:"content_hash,omitempty"`
	MergedAt      string   `toml:"merged_at,omitempty"`
	Evaluations   evalTOML `toml:"evaluations"`
}

type evalTOML struct {
	Followed     int `toml:"followed"`
	Contradicted int `toml:"contradicted"`
	Ignored      int `toml:"ignored"`
}

type transitionTOML struct {
	From   string `toml:"from"`
	To     string `toml:"to"`
	At     string `toml:"at"`
	Reason string `toml:"reason"`
}

// --- Conversion functions ---

func entryToRecord(entry InstructionEntry) memoryRecord {
	record := memoryRecord{
		Title:             entry.Title,
		Content:           entry.Content,
		SourceType:        entry.SourceType,
		SourcePath:        entry.SourcePath,
		ContentHash:       entry.ContentHash,
		SurfacedCount:     entry.SurfacedCount,
		FollowedCount:     entry.Evaluations.Followed,
		ContradictedCount: entry.Evaluations.Contradicted,
		IgnoredCount:      entry.Evaluations.Ignored,
		EnforcementLevel:  string(entry.EnforcementLevel),
	}

	if !entry.RegisteredAt.IsZero() {
		record.CreatedAt = entry.RegisteredAt.UTC().Format(time.RFC3339)
	}

	if !entry.UpdatedAt.IsZero() {
		record.UpdatedAt = entry.UpdatedAt.UTC().Format(time.RFC3339)
	}

	if entry.LastSurfaced != nil {
		record.LastSurfacedAt = entry.LastSurfaced.UTC().Format(time.RFC3339)
	}

	if len(entry.Links) > 0 {
		record.Links = make([]linkTOML, len(entry.Links))
		for i, link := range entry.Links {
			record.Links[i] = linkTOML(link)
		}
	}

	if len(entry.Absorbed) > 0 {
		record.Absorbed = make([]absorbedTOML, len(entry.Absorbed))
		for i, abs := range entry.Absorbed {
			record.Absorbed[i] = absorbedTOML{
				From:          abs.From,
				SurfacedCount: abs.SurfacedCount,
				ContentHash:   abs.ContentHash,
				MergedAt:      abs.MergedAt.UTC().Format(time.RFC3339),
				Evaluations: evalTOML{
					Followed:     abs.Evaluations.Followed,
					Contradicted: abs.Evaluations.Contradicted,
					Ignored:      abs.Evaluations.Ignored,
				},
			}
		}
	}

	if len(entry.Transitions) > 0 {
		record.Transitions = make([]transitionTOML, len(entry.Transitions))
		for i, tr := range entry.Transitions {
			record.Transitions[i] = transitionTOML{
				From:   string(tr.From),
				To:     string(tr.To),
				At:     tr.At.UTC().Format(time.RFC3339),
				Reason: tr.Reason,
			}
		}
	}

	return record
}

func recordToEntry(id string, record memoryRecord) InstructionEntry {
	sourceType := record.SourceType
	if sourceType == "" {
		sourceType = SourceTypeMemory
	}

	sourcePath := record.SourcePath
	if sourcePath == "" {
		sourcePath = id
	}

	level := EnforcementLevel(record.EnforcementLevel)
	if level == "" {
		level = EnforcementAdvisory
	}

	entry := InstructionEntry{
		ID:         id,
		SourceType: sourceType,
		SourcePath: sourcePath,
		Title:      record.Title,
		Content:    record.Content,
		ContentHash: record.ContentHash,
		SurfacedCount: record.SurfacedCount,
		Evaluations: EvaluationCounters{
			Followed:     record.FollowedCount,
			Contradicted: record.ContradictedCount,
			Ignored:      record.IgnoredCount,
		},
		EnforcementLevel: level,
	}

	if record.CreatedAt != "" {
		parsed, parseErr := time.Parse(time.RFC3339, record.CreatedAt)
		if parseErr == nil {
			entry.RegisteredAt = parsed
		}
	}

	if record.UpdatedAt != "" {
		parsed, parseErr := time.Parse(time.RFC3339, record.UpdatedAt)
		if parseErr == nil {
			entry.UpdatedAt = parsed
		}
	}

	if record.LastSurfacedAt != "" {
		parsed, parseErr := time.Parse(time.RFC3339, record.LastSurfacedAt)
		if parseErr == nil {
			entry.LastSurfaced = &parsed
		}
	}

	if len(record.Links) > 0 {
		entry.Links = make([]Link, len(record.Links))
		for i, link := range record.Links {
			entry.Links[i] = Link(link)
		}
	}

	if len(record.Absorbed) > 0 {
		entry.Absorbed = make([]AbsorbedRecord, len(record.Absorbed))
		for i, abs := range record.Absorbed {
			mergedAt, _ := time.Parse(time.RFC3339, abs.MergedAt)
			entry.Absorbed[i] = AbsorbedRecord{
				From:          abs.From,
				SurfacedCount: abs.SurfacedCount,
				ContentHash:   abs.ContentHash,
				MergedAt:      mergedAt,
				Evaluations: EvaluationCounters{
					Followed:     abs.Evaluations.Followed,
					Contradicted: abs.Evaluations.Contradicted,
					Ignored:      abs.Evaluations.Ignored,
				},
			}
		}
	}

	if len(record.Transitions) > 0 {
		entry.Transitions = make([]EnforcementTransition, len(record.Transitions))
		for i, tr := range record.Transitions {
			at, _ := time.Parse(time.RFC3339, tr.At)
			entry.Transitions[i] = EnforcementTransition{
				From:   EnforcementLevel(tr.From),
				To:     EnforcementLevel(tr.To),
				At:     at,
				Reason: tr.Reason,
			}
		}
	}

	return entry
}

// --- Functional options ---

// WithTOMLReadFile injects a file reader.
func WithTOMLReadFile(fn func(string) ([]byte, error)) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.readFile = fn }
}

// WithTOMLWriteFile injects a file writer.
func WithTOMLWriteFile(fn func(string, []byte, os.FileMode) error) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.writeFile = fn }
}

// WithTOMLReadDir injects a directory reader.
func WithTOMLReadDir(fn func(string) ([]os.DirEntry, error)) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.readDir = fn }
}

// WithTOMLRemove injects a file removal function.
func WithTOMLRemove(fn func(string) error) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.remove = fn }
}

// WithTOMLRename injects a file rename function.
func WithTOMLRename(fn func(string, string) error) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.rename = fn }
}

// WithTOMLMkdirAll injects a directory creation function.
func WithTOMLMkdirAll(fn func(string, os.FileMode) error) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.mkdirAll = fn }
}

// WithTOMLOpenFile injects a file-open function (used for flock).
func WithTOMLOpenFile(fn func(string, int, os.FileMode) (*os.File, error)) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.openFile = fn }
}

// WithTOMLLockFile injects a file-lock function.
func WithTOMLLockFile(fn func(*os.File) error) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.lockFile = fn }
}

// WithTOMLUnlockFile injects a file-unlock function.
func WithTOMLUnlockFile(fn func(*os.File) error) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.unlockFile = fn }
}

// WithTOMLClock injects a clock function.
func WithTOMLClock(fn func() time.Time) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.nowFn = fn }
}

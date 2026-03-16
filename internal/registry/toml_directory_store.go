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
	id = s.normalizeID(id)
	absPath := filepath.Join(s.dataDir, id)

	data, err := s.readFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	var record memoryRecord

	_, decodeErr := toml.Decode(string(data), &record)
	if decodeErr != nil {
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

// Merge absorbs the source entry into the target entry, then removes the source.
func (s *TOMLDirectoryStore) Merge(sourceID, targetID string) error {
	sourceID = s.normalizeID(sourceID)
	targetID = s.normalizeID(targetID)
	sourcePath := filepath.Join(s.dataDir, sourceID)

	// Read source.
	sourceData, readErr := s.readFile(sourcePath)
	if readErr != nil {
		return fmt.Errorf("%w: source=%s target=%s", ErrMergeNotFound, sourceID, targetID)
	}

	var sourceRecord memoryRecord

	_, decodeErr := toml.Decode(string(sourceData), &sourceRecord)
	if decodeErr != nil {
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
	modErr := s.applyMergeToTarget(sourceID, targetID, sourceRecord)
	if modErr != nil {
		if errors.Is(modErr, ErrNotFound) {
			return fmt.Errorf("%w: source=%s target=%s", ErrMergeNotFound, sourceID, targetID)
		}

		return modErr
	}

	// Delete source after successful target update.
	err := s.remove(sourcePath)
	if err != nil {
		return fmt.Errorf("removing source %s: %w", sourceID, err)
	}

	return nil
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

// RecordSurfacing increments the surfaced_count and sets last_surfaced_at.
func (s *TOMLDirectoryStore) RecordSurfacing(id string) error {
	return s.withFileLocked(id, func(record *memoryRecord) error {
		record.SurfacedCount++
		record.LastSurfacedAt = s.now().UTC().Format(time.RFC3339)

		return nil
	})
}

// Register writes a new TOML file for the given entry.
// Returns ErrDuplicateID if a file already exists at entry.ID.
func (s *TOMLDirectoryStore) Register(entry InstructionEntry) error {
	entry.ID = s.normalizeID(entry.ID)
	absPath := filepath.Join(s.dataDir, entry.ID)

	// Check if file already exists.
	_, readErr := s.readFile(absPath)
	if readErr == nil {
		return fmt.Errorf("%w: %s", ErrDuplicateID, entry.ID)
	}

	mkdirErr := s.mkdirAll(filepath.Dir(absPath), tomlDirPerm)
	if mkdirErr != nil {
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

// Remove deletes the TOML file for the given ID.
func (s *TOMLDirectoryStore) Remove(id string) error {
	id = s.normalizeID(id)
	absPath := filepath.Join(s.dataDir, id)

	_, existErr := s.readFile(absPath)
	if existErr != nil {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	err := s.remove(absPath)
	if err != nil {
		return fmt.Errorf("removing %s: %w", id, err)
	}

	return nil
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

// applyMergeToTarget mutates the target record under flock, absorbing source counters.
func (s *TOMLDirectoryStore) applyMergeToTarget(sourceID, targetID string, sourceRecord memoryRecord) error {
	return s.withFileLocked(targetID, func(record *memoryRecord) error {
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
}

// --- Private helpers ---

// withFileLocked acquires an exclusive flock on a stable sidecar lock file,
// reads the data file, calls fn with the decoded record, then writes atomically.
//
// A sidecar .lock file (not the data file) is used for locking so that the
// inode is stable across atomic renames. All goroutines contending on the same
// id block on the same inode regardless of how many rename cycles have occurred.
// normalizeID converts an absolute path within dataDir to a relative ID.
// If id is already relative or outside dataDir, it is returned unchanged.
func (s *TOMLDirectoryStore) normalizeID(id string) string {
	prefix := s.dataDir + string(filepath.Separator)
	if trimmed, ok := strings.CutPrefix(id, prefix); ok {
		return trimmed
	}

	return id
}

func (s *TOMLDirectoryStore) now() time.Time {
	if s.nowFn != nil {
		return s.nowFn()
	}

	return time.Now()
}

func (s *TOMLDirectoryStore) withFileLocked(id string, fn func(*memoryRecord) error) error {
	id = s.normalizeID(id)
	absPath := filepath.Join(s.dataDir, id)
	lockPath := absPath + ".lock"

	// Check that the data file exists before acquiring the lock.
	_, existErr := s.readFile(absPath)
	if existErr != nil {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	// Open (or create) the sidecar lock file. Its inode is never renamed.
	f, openErr := s.openFile(lockPath, os.O_RDWR|os.O_CREATE, tomlFilePerm)
	if openErr != nil {
		return fmt.Errorf("opening lock file for %s: %w", id, openErr)
	}

	defer func() { _ = f.Close() }()

	lockErr := s.lockFile(f)
	if lockErr != nil {
		return fmt.Errorf("acquiring lock on %s: %w", id, lockErr)
	}

	defer func() { _ = s.unlockFile(f) }()

	// Read after acquiring lock to get the latest state.
	data, readErr := s.readFile(absPath)
	if readErr != nil {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	var record memoryRecord

	_, decodeErr := toml.Decode(string(data), &record)
	if decodeErr != nil {
		return fmt.Errorf("decoding TOML for %s: %w", id, decodeErr)
	}

	err := fn(&record)
	if err != nil {
		return err
	}

	return s.writeAtomic(absPath, record)
}

// writeAtomic encodes record as TOML and renames a temp file into place.
func (s *TOMLDirectoryStore) writeAtomic(path string, record memoryRecord) error {
	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(record)
	if err != nil {
		return fmt.Errorf("encoding TOML for %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	tmpPath := filepath.Join(dir, filepath.Base(path)+tmpSuffix)

	err = s.writeFile(tmpPath, buf.Bytes(), tomlFilePerm)
	if err != nil {
		return fmt.Errorf("writing temp file for %s: %w", path, err)
	}

	err = s.rename(tmpPath, path)
	if err != nil {
		_ = s.remove(tmpPath) // best-effort cleanup

		return fmt.Errorf("renaming to final %s: %w", path, err)
	}

	return nil
}

// WithTOMLClock injects a clock function.
func WithTOMLClock(fn func() time.Time) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.nowFn = fn }
}

// WithTOMLLockFile injects a file-lock function.
func WithTOMLLockFile(fn func(*os.File) error) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.lockFile = fn }
}

// WithTOMLMkdirAll injects a directory creation function.
func WithTOMLMkdirAll(fn func(string, os.FileMode) error) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.mkdirAll = fn }
}

// WithTOMLOpenFile injects a file-open function (used for flock).
func WithTOMLOpenFile(fn func(string, int, os.FileMode) (*os.File, error)) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.openFile = fn }
}

// WithTOMLReadDir injects a directory reader.
func WithTOMLReadDir(fn func(string) ([]os.DirEntry, error)) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.readDir = fn }
}

// --- Functional options ---

// WithTOMLReadFile injects a file reader.
func WithTOMLReadFile(fn func(string) ([]byte, error)) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.readFile = fn }
}

// WithTOMLRemove injects a file removal function.
func WithTOMLRemove(fn func(string) error) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.remove = fn }
}

// WithTOMLRename injects a file rename function.
func WithTOMLRename(fn func(string, string) error) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.rename = fn }
}

// WithTOMLUnlockFile injects a file-unlock function.
func WithTOMLUnlockFile(fn func(*os.File) error) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.unlockFile = fn }
}

// WithTOMLWriteFile injects a file writer.
func WithTOMLWriteFile(fn func(string, []byte, os.FileMode) error) TOMLDirOption {
	return func(s *TOMLDirectoryStore) { s.writeFile = fn }
}

// unexported constants.
const (
	memoriesSubdir = "memories"
	tmpSuffix      = ".tmp"
	tomlDirPerm    = os.FileMode(0o750)
	tomlExt        = ".toml"
	tomlFilePerm   = os.FileMode(0o644)
)

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

type linkTOML struct {
	Target           string  `toml:"target"`
	Weight           float64 `toml:"weight"`
	Basis            string  `toml:"basis"`
	CoSurfacingCount int     `toml:"co_surfacing_count,omitempty"`
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

type transitionTOML struct {
	From   string `toml:"from"`
	To     string `toml:"to"`
	At     string `toml:"at"`
	Reason string `toml:"reason"`
}

func entryAbsorbedToRecord(absorbed []AbsorbedRecord) []absorbedTOML {
	if len(absorbed) == 0 {
		return nil
	}

	result := make([]absorbedTOML, len(absorbed))

	for i, abs := range absorbed {
		result[i] = absorbedTOML{
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

	return result
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

	record.Absorbed = entryAbsorbedToRecord(entry.Absorbed)
	record.Transitions = entryTransitionsToRecord(entry.Transitions)

	return record
}

func entryTransitionsToRecord(transitions []EnforcementTransition) []transitionTOML {
	if len(transitions) == 0 {
		return nil
	}

	result := make([]transitionTOML, len(transitions))

	for i, transition := range transitions {
		result[i] = transitionTOML{
			From:   string(transition.From),
			To:     string(transition.To),
			At:     transition.At.UTC().Format(time.RFC3339),
			Reason: transition.Reason,
		}
	}

	return result
}

// flockExclusive acquires an exclusive flock on f.
func flockExclusive(f *os.File) error {
	//nolint:wrapcheck,gosec // thin syscall wrapper; Fd fits int on 64-bit
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

// flockUnlock releases the flock on f.
func flockUnlock(f *os.File) error {
	//nolint:wrapcheck,gosec // thin syscall wrapper; Fd fits int on 64-bit
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}

func parseOptionalTimeField(s string) *time.Time {
	if s == "" {
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}

	return &parsed
}

func parseTimeField(s string) time.Time {
	parsed, _ := time.Parse(time.RFC3339, s)

	return parsed
}

func recordAbsorbedToAbsorbed(absorbed []absorbedTOML) []AbsorbedRecord {
	if len(absorbed) == 0 {
		return nil
	}

	result := make([]AbsorbedRecord, len(absorbed))

	for i, abs := range absorbed {
		result[i] = AbsorbedRecord{
			From:          abs.From,
			SurfacedCount: abs.SurfacedCount,
			ContentHash:   abs.ContentHash,
			MergedAt:      parseTimeField(abs.MergedAt),
			Evaluations: EvaluationCounters{
				Followed:     abs.Evaluations.Followed,
				Contradicted: abs.Evaluations.Contradicted,
				Ignored:      abs.Evaluations.Ignored,
			},
		}
	}

	return result
}

func recordLinksToLinks(links []linkTOML) []Link {
	if len(links) == 0 {
		return nil
	}

	result := make([]Link, len(links))

	for i, link := range links {
		result[i] = Link(link)
	}

	return result
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
		ID:            id,
		SourceType:    sourceType,
		SourcePath:    sourcePath,
		Title:         record.Title,
		Content:       record.Content,
		ContentHash:   record.ContentHash,
		SurfacedCount: record.SurfacedCount,
		Evaluations: EvaluationCounters{
			Followed:     record.FollowedCount,
			Contradicted: record.ContradictedCount,
			Ignored:      record.IgnoredCount,
		},
		EnforcementLevel: level,
	}

	entry.RegisteredAt = parseTimeField(record.CreatedAt)
	entry.UpdatedAt = parseTimeField(record.UpdatedAt)
	entry.LastSurfaced = parseOptionalTimeField(record.LastSurfacedAt)
	entry.Links = recordLinksToLinks(record.Links)
	entry.Absorbed = recordAbsorbedToAbsorbed(record.Absorbed)
	entry.Transitions = recordTransitionsToTransitions(record.Transitions)

	return entry
}

func recordTransitionsToTransitions(transitions []transitionTOML) []EnforcementTransition {
	if len(transitions) == 0 {
		return nil
	}

	result := make([]EnforcementTransition, len(transitions))

	for i, tr := range transitions {
		result[i] = EnforcementTransition{
			From:   EnforcementLevel(tr.From),
			To:     EnforcementLevel(tr.To),
			At:     parseTimeField(tr.At),
			Reason: tr.Reason,
		}
	}

	return result
}

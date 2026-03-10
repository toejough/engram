package registry

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// FileReader reads a file and returns its contents.
type FileReader func(path string) ([]byte, error)

// FileWriter writes content to a file.
type FileWriter func(path string, content []byte) error

// JSONLOption configures a JSONLStore.
type JSONLOption func(*JSONLStore)

// JSONLStore implements Registry backed by a JSONL file.
// All I/O is performed through injected reader/writer functions.
// All public methods are safe for concurrent use.
type JSONLStore struct {
	mu      sync.Mutex
	path    string
	read    FileReader
	write   FileWriter
	now     func() time.Time
	entries map[string]*InstructionEntry
	loaded  bool
}

// NewJSONLStore creates a new JSONL-backed registry store.
func NewJSONLStore(path string, opts ...JSONLOption) *JSONLStore {
	store := &JSONLStore{
		path:    path,
		entries: make(map[string]*InstructionEntry),
	}

	for _, opt := range opts {
		opt(store)
	}

	return store
}

// BulkLoad replaces all entries in the store (used by backfill init).
func (s *JSONLStore) BulkLoad(entries []InstructionEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make(map[string]*InstructionEntry, len(entries))

	for idx := range entries {
		s.entries[entries[idx].ID] = &entries[idx]
	}

	s.loaded = true

	return s.save()
}

// Get returns a single entry by ID.
func (s *JSONLStore) Get(id string) (*InstructionEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.ensureLoaded()
	if err != nil {
		return nil, err
	}

	entry, ok := s.entries[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	copied := *entry

	return &copied, nil
}

// List returns all entries in the store.
func (s *JSONLStore) List() ([]InstructionEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.ensureLoaded()
	if err != nil {
		return nil, err
	}

	result := make([]InstructionEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		result = append(result, *entry)
	}

	return result, nil
}

// Merge absorbs the source entry into the target entry, then removes the source.
func (s *JSONLStore) Merge(sourceID, targetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.ensureLoaded()
	if err != nil {
		return err
	}

	source, sourceOK := s.entries[sourceID]
	target, targetOK := s.entries[targetID]

	if !sourceOK || !targetOK {
		return fmt.Errorf("%w: source=%s target=%s",
			ErrMergeNotFound, sourceID, targetID)
	}

	if source.SourceType != SourceTypeMemory || target.SourceType != SourceTypeMemory {
		return fmt.Errorf("%w: source=%s target=%s",
			ErrMergeSourceType, sourceID, targetID)
	}

	record := AbsorbedRecord{
		From:          source.ID,
		SurfacedCount: source.SurfacedCount,
		Evaluations:   source.Evaluations,
		ContentHash:   source.ContentHash,
		MergedAt:      s.clock(),
	}

	target.Absorbed = append(target.Absorbed, record)
	target.SurfacedCount += source.SurfacedCount
	target.Evaluations.Followed += source.Evaluations.Followed
	target.Evaluations.Contradicted += source.Evaluations.Contradicted
	target.Evaluations.Ignored += source.Evaluations.Ignored

	delete(s.entries, sourceID)

	return s.save()
}

// RecordEvaluation increments the appropriate evaluation counter.
func (s *JSONLStore) RecordEvaluation(id string, outcome Outcome) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.ensureLoaded()
	if err != nil {
		return err
	}

	entry, ok := s.entries[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	switch outcome {
	case Followed:
		entry.Evaluations.Followed++
	case Contradicted:
		entry.Evaluations.Contradicted++
	case Ignored:
		entry.Evaluations.Ignored++
	}

	return s.save()
}

// RecordSurfacing increments the surfaced count and updates last_surfaced.
func (s *JSONLStore) RecordSurfacing(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.ensureLoaded()
	if err != nil {
		return err
	}

	entry, ok := s.entries[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	entry.SurfacedCount++

	now := s.clock()
	entry.LastSurfaced = &now

	return s.save()
}

// Register adds a new instruction entry to the store.
func (s *JSONLStore) Register(entry InstructionEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.ensureLoaded()
	if err != nil {
		return err
	}

	if _, exists := s.entries[entry.ID]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateID, entry.ID)
	}

	if entry.EnforcementLevel == "" {
		entry.EnforcementLevel = EnforcementAdvisory
	}

	s.entries[entry.ID] = &entry

	return s.save()
}

// Remove deletes an entry from the store.
func (s *JSONLStore) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.ensureLoaded()
	if err != nil {
		return err
	}

	if _, ok := s.entries[id]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	delete(s.entries, id)

	return s.save()
}

func (s *JSONLStore) clock() time.Time {
	if s.now != nil {
		return s.now()
	}

	return time.Now()
}

func (s *JSONLStore) ensureLoaded() error { //nolint:unparam // error return kept for future use
	if s.loaded {
		return nil
	}

	data, err := s.read(s.path)
	if err != nil {
		// Missing file is fine — start empty.
		s.loaded = true

		return nil //nolint:nilerr // intentional: skip missing file
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry InstructionEntry

		jsonErr := json.Unmarshal([]byte(line), &entry)
		if jsonErr != nil {
			continue // skip malformed lines
		}

		if entry.EnforcementLevel == "" {
			entry.EnforcementLevel = EnforcementAdvisory
		}

		s.entries[entry.ID] = &entry
	}

	s.loaded = true

	return nil
}

func (s *JSONLStore) save() error {
	var sb strings.Builder

	for _, entry := range s.entries {
		line, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("marshaling entry %s: %w", entry.ID, err)
		}

		sb.Write(line)
		sb.WriteByte('\n')
	}

	writeErr := s.write(s.path, []byte(sb.String()))
	if writeErr != nil {
		return fmt.Errorf("writing registry: %w", writeErr)
	}

	return nil
}

// WithNow injects a clock function.
func WithNow(now func() time.Time) JSONLOption {
	return func(s *JSONLStore) { s.now = now }
}

// WithReader injects a file reader.
func WithReader(reader FileReader) JSONLOption {
	return func(s *JSONLStore) { s.read = reader }
}

// WithWriter injects a file writer.
func WithWriter(writer FileWriter) JSONLOption {
	return func(s *JSONLStore) { s.write = writer }
}

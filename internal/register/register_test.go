package register_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/register"
	"engram/internal/registry"
)

// TestPruneStale_ListError verifies a List error during pruning is logged.
func TestPruneStale_ListError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := &fakeRegistryWithListErr{
		fakeRegistry: newFakeRegistry(),
		listErr:      errors.New("list failed"),
	}
	logger := &fakeSurfacingLogger{}

	var stderrBuf strings.Builder

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderrBuf.String()).To(ContainSubstring("list failed"))
}

// TestPruneStale_RemoveError verifies a Remove error during pruning is logged.
func TestPruneStale_RemoveError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := &fakeRegistryWithRemoveErr{
		fakeRegistry: newFakeRegistry(),
		removeErr:    errors.New("remove failed"),
	}
	logger := &fakeSurfacingLogger{}

	var stderrBuf strings.Builder

	// Pre-populate with a stale rule entry.
	reg.entries["rule:stale.md"] = registry.InstructionEntry{
		ID: "rule:stale.md", SourceType: "rule",
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderrBuf.String()).To(ContainSubstring("remove failed"))
}

// TestRecordSurfacing_LoggerError verifies logger error is logged.
func TestRecordSurfacing_LoggerError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLoggerWithErr{err: errors.New("log write error")}

	var stderrBuf strings.Builder

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	entries := []registry.InstructionEntry{
		{
			ID:          "claude-md:CLAUDE.md:rule-one",
			SourceType:  "claude-md",
			Title:       "Rule one",
			Content:     "Rule one",
			ContentHash: "hash1",
		},
	}

	err := registrar.Run(entries)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderrBuf.String()).To(ContainSubstring("log write error"))
}

// TestRecordSurfacing_RegistryError verifies registry surfacing error is logged.
func TestRecordSurfacing_RegistryError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := &fakeRegistryWithSurfacingErr{
		fakeRegistry: newFakeRegistry(),
		surfacingErr: errors.New("surfacing failed"),
	}
	logger := &fakeSurfacingLogger{}

	var stderrBuf strings.Builder

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	entries := []registry.InstructionEntry{
		{
			ID:          "claude-md:CLAUDE.md:rule-one",
			SourceType:  "claude-md",
			Title:       "Rule one",
			Content:     "Rule one",
			ContentHash: "hash1",
		},
	}

	err := registrar.Run(entries)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderrBuf.String()).To(ContainSubstring("surfacing failed"))
}

// TestRegisterEntries_GetNonNotFoundError verifies non-ErrNotFound get
// errors are logged and entry is skipped.
func TestRegisterEntries_GetNonNotFoundError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := &fakeRegistryWithGetErr{
		fakeRegistry: newFakeRegistry(),
		getErr:       errors.New("database corrupt"),
	}
	logger := &fakeSurfacingLogger{}

	var stderrBuf strings.Builder

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	entries := []registry.InstructionEntry{
		{
			ID:          "claude-md:CLAUDE.md:rule-one",
			SourceType:  "claude-md",
			Title:       "Rule one",
			Content:     "Rule one",
			ContentHash: "hash1",
		},
	}

	err := registrar.Run(entries)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderrBuf.String()).To(ContainSubstring("database corrupt"))
}

// TestRegisterEntries_RemoveErrorDuringUpdate verifies remove error during
// content-changed update is logged.
func TestRegisterEntries_RemoveErrorDuringUpdate(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := &fakeRegistryWithRemoveErr{
		fakeRegistry: newFakeRegistry(),
		removeErr:    errors.New("cannot remove"),
	}
	logger := &fakeSurfacingLogger{}

	var stderrBuf strings.Builder

	oldTime := fixedTime.Add(-24 * time.Hour)

	reg.entries["rule:go.md"] = registry.InstructionEntry{
		ID:           "rule:go.md",
		SourceType:   "rule",
		SourcePath:   "go.md",
		Title:        "go.md",
		ContentHash:  "old-hash",
		RegisteredAt: oldTime,
	}

	entries := []registry.InstructionEntry{
		{
			ID:           "rule:go.md",
			SourceType:   "rule",
			SourcePath:   "go.md",
			Title:        "go.md",
			Content:      "## Updated Go rules\nUse gofmt always.",
			ContentHash:  "new-hash",
			RegisteredAt: fixedTime,
			UpdatedAt:    fixedTime,
		},
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(entries)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderrBuf.String()).To(ContainSubstring("cannot remove"))
}

// traces: T-273
func TestT273_RegisterNewEntries(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	entries := []registry.InstructionEntry{
		{
			ID:           "claude-md:CLAUDE.md:rule-one",
			SourceType:   "claude-md",
			Title:        "Rule one",
			Content:      "Rule one",
			ContentHash:  "hash1",
			RegisteredAt: fixedTime,
			UpdatedAt:    fixedTime,
		},
		{
			ID:           "claude-md:CLAUDE.md:rule-two",
			SourceType:   "claude-md",
			Title:        "Rule two",
			Content:      "Rule two",
			ContentHash:  "hash2",
			RegisteredAt: fixedTime,
			UpdatedAt:    fixedTime,
		},
		{
			ID:           "rule:go.md",
			SourceType:   "rule",
			Title:        "go.md",
			Content:      "Use gofmt.",
			ContentHash:  "hash3",
			RegisteredAt: fixedTime,
			UpdatedAt:    fixedTime,
		},
	}

	err := registrar.Run(entries)
	g.Expect(err).NotTo(HaveOccurred())

	// 3 entries registered.
	g.Expect(reg.registered).To(HaveLen(3))

	// Each entry should have correct timestamps.
	for _, entry := range reg.registered {
		g.Expect(entry.RegisteredAt).To(Equal(fixedTime))
		g.Expect(entry.UpdatedAt).To(Equal(fixedTime))
		g.Expect(entry.ContentHash).NotTo(BeEmpty())
	}
}

// traces: T-274
func TestT274_UpdateChangedEntries(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	oldTime := fixedTime.Add(-24 * time.Hour)

	// Pre-populate registry with an entry that has different content hash.
	reg.entries["rule:go.md"] = registry.InstructionEntry{
		ID:            "rule:go.md",
		SourceType:    "rule",
		SourcePath:    "go.md",
		Title:         "go.md",
		ContentHash:   "old-hash-value",
		RegisteredAt:  oldTime,
		UpdatedAt:     oldTime,
		SurfacedCount: 5,
		Evaluations: registry.EvaluationCounters{
			Followed: 3, Contradicted: 1, Ignored: 1,
		},
	}

	entries := []registry.InstructionEntry{
		{
			ID:           "rule:go.md",
			SourceType:   "rule",
			SourcePath:   "go.md",
			Title:        "go.md",
			Content:      "## Updated Go rules\nUse gofmt always.",
			ContentHash:  "new-hash-value",
			RegisteredAt: fixedTime,
			UpdatedAt:    fixedTime,
		},
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(entries)
	g.Expect(err).NotTo(HaveOccurred())

	// The entry should have been removed and re-registered.
	g.Expect(reg.removed).To(ContainElement("rule:go.md"))

	updated, getErr := reg.Get("rule:go.md")
	g.Expect(getErr).NotTo(HaveOccurred())

	if getErr != nil || updated == nil {
		return
	}

	// Content hash should be new.
	g.Expect(updated.ContentHash).NotTo(Equal("old-hash-value"))
	// UpdatedAt should be the new time.
	g.Expect(updated.UpdatedAt).To(Equal(fixedTime))
	// Counters should be preserved (5 original + 1 from implicit surfacing phase).
	g.Expect(updated.SurfacedCount).To(Equal(6))
	g.Expect(updated.Evaluations.Followed).To(Equal(3))
	g.Expect(updated.Evaluations.Contradicted).To(Equal(1))
	g.Expect(updated.Evaluations.Ignored).To(Equal(1))
}

// traces: T-275
func TestT275_PruneStaleRemovesAbsentNonMemoryEntries(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	// Pre-populate with a rule entry that won't be in the discovered set.
	reg.entries["rule:old.md"] = registry.InstructionEntry{
		ID: "rule:old.md", SourceType: "rule",
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	// Run with empty entries — rule:old.md is stale.
	err := registrar.Run(nil)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(reg.removed).To(ContainElement("rule:old.md"))
}

// traces: T-275
func TestT275_StalePruningRemovesAbsentNonMemory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	// Pre-populate with two rule entries and one memory entry.
	reg.entries["rule:go.md"] = registry.InstructionEntry{
		ID: "rule:go.md", SourceType: "rule",
	}
	reg.entries["rule:deleted.md"] = registry.InstructionEntry{
		ID: "rule:deleted.md", SourceType: "rule",
	}
	reg.entries["memory:foo.toml"] = registry.InstructionEntry{
		ID: "memory:foo.toml", SourceType: "memory",
	}

	// Only go.md is in the entries set.
	entries := []registry.InstructionEntry{
		{
			ID:          "rule:go.md",
			SourceType:  "rule",
			SourcePath:  "go.md",
			Title:       "go.md",
			Content:     "Use gofmt.",
			ContentHash: "hash-go",
		},
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(entries)
	g.Expect(err).NotTo(HaveOccurred())

	// rule:deleted.md should be removed.
	g.Expect(reg.removed).To(ContainElement("rule:deleted.md"))

	// memory:foo.toml should NOT be removed.
	_, memErr := reg.Get("memory:foo.toml")
	g.Expect(memErr).NotTo(HaveOccurred())
}

// traces: T-276
func TestT276_MemoryEntriesNeverPruned(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	// Pre-populate with memory entries only — none in discovered set.
	reg.entries["memory:alpha.toml"] = registry.InstructionEntry{
		ID: "memory:alpha.toml", SourceType: "memory",
	}
	reg.entries["memory:beta.toml"] = registry.InstructionEntry{
		ID: "memory:beta.toml", SourceType: "memory",
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	// Run with empty entries — no sources discovered.
	err := registrar.Run(nil)
	g.Expect(err).NotTo(HaveOccurred())

	// No removals should have happened.
	g.Expect(reg.removed).To(BeEmpty())

	// Both memory entries should still exist.
	_, err1 := reg.Get("memory:alpha.toml")
	g.Expect(err1).NotTo(HaveOccurred())

	_, err2 := reg.Get("memory:beta.toml")
	g.Expect(err2).NotTo(HaveOccurred())
}

// traces: T-277
func TestT277_ImplicitSurfacingRecordsForAlwaysLoaded(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	entries := []registry.InstructionEntry{
		{
			ID:          "claude-md:CLAUDE.md:rule-one",
			SourceType:  "claude-md",
			Title:       "Rule one",
			Content:     "Rule one",
			ContentHash: "hash1",
		},
		{
			ID:          "rule:go.md",
			SourceType:  "rule",
			Title:       "go.md",
			Content:     "Use gofmt.",
			ContentHash: "hash2",
		},
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(entries)
	g.Expect(err).NotTo(HaveOccurred())

	// 2 entries.
	// RecordSurfacing should be called for each.
	g.Expect(reg.surfaced).To(HaveLen(2))

	// SurfacingLogger should also be called for each.
	g.Expect(logger.events).To(HaveLen(2))

	for _, event := range logger.events {
		g.Expect(event.mode).To(Equal("session-start"))
		g.Expect(event.timestamp).To(Equal(fixedTime))
	}
}

// traces: T-279
func TestT279_IdempotentSecondRunIsNoOp(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	entries := []registry.InstructionEntry{
		{
			ID:           "claude-md:CLAUDE.md:rule-one",
			SourceType:   "claude-md",
			Title:        "Rule one",
			Content:      "Rule one",
			ContentHash:  "hash1",
			RegisteredAt: fixedTime,
			UpdatedAt:    fixedTime,
		},
		{
			ID:           "claude-md:CLAUDE.md:rule-two",
			SourceType:   "claude-md",
			Title:        "Rule two",
			Content:      "Rule two",
			ContentHash:  "hash2",
			RegisteredAt: fixedTime,
			UpdatedAt:    fixedTime,
		},
		{
			ID:           "rule:go.md",
			SourceType:   "rule",
			Title:        "go.md",
			Content:      "Use gofmt.",
			ContentHash:  "hash3",
			RegisteredAt: fixedTime,
			UpdatedAt:    fixedTime,
		},
	}

	// First run.
	err := registrar.Run(entries)
	g.Expect(err).NotTo(HaveOccurred())

	firstRegCount := len(reg.registered)
	g.Expect(firstRegCount).To(Equal(3))

	// Reset tracking slices but keep entries.
	reg.registered = reg.registered[:0]
	reg.removed = reg.removed[:0]
	reg.surfaced = reg.surfaced[:0]
	logger.events = logger.events[:0]

	// Second run — same entries.
	err = registrar.Run(entries)
	g.Expect(err).NotTo(HaveOccurred())

	// No new registrations.
	g.Expect(reg.registered).To(BeEmpty())
	// No removals.
	g.Expect(reg.removed).To(BeEmpty())
	// But surfacing SHOULD still be recorded.
	g.Expect(reg.surfaced).To(HaveLen(3))
	g.Expect(logger.events).To(HaveLen(3))
}

// traces: T-280
func TestT280_FireAndForgetErrorsDontFail(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	reg.registerErr = errors.New("database locked")

	logger := &fakeSurfacingLogger{}

	var stderrBuf strings.Builder

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	entries := []registry.InstructionEntry{
		{ID: "e1", SourceType: "claude-md", ContentHash: "h1"},
		{ID: "e2", SourceType: "claude-md", ContentHash: "h2"},
		{ID: "e3", SourceType: "claude-md", ContentHash: "h3"},
	}

	err := registrar.Run(entries)

	// Run should NOT return an error despite registry failures.
	g.Expect(err).NotTo(HaveOccurred())

	// Errors should be logged to stderr.
	stderrOutput := stderrBuf.String()
	g.Expect(stderrOutput).To(ContainSubstring("database locked"))

	// All 3 entries should have been attempted (not short-circuited).
	errorLines := strings.Count(stderrOutput, "database locked")
	g.Expect(errorLines).To(Equal(3))
}

// unexported variables.
var (
	fixedTime = time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
)

// fakeRegistry is a test double for register.Registry.
type fakeRegistry struct {
	entries     map[string]registry.InstructionEntry
	registered  []registry.InstructionEntry
	surfaced    []string
	removed     []string
	registerErr error
}

func (r *fakeRegistry) Get(id string) (*registry.InstructionEntry, error) {
	entry, ok := r.entries[id]
	if !ok {
		return nil, registry.ErrNotFound
	}

	return &entry, nil
}

func (r *fakeRegistry) List() ([]registry.InstructionEntry, error) {
	result := make([]registry.InstructionEntry, 0, len(r.entries))
	for _, entry := range r.entries {
		result = append(result, entry)
	}

	return result, nil
}

func (r *fakeRegistry) RecordSurfacing(id string) error {
	r.surfaced = append(r.surfaced, id)

	entry, ok := r.entries[id]
	if ok {
		entry.SurfacedCount++
		r.entries[id] = entry
	}

	return nil
}

func (r *fakeRegistry) Register(entry registry.InstructionEntry) error {
	if r.registerErr != nil {
		return r.registerErr
	}

	r.entries[entry.ID] = entry
	r.registered = append(r.registered, entry)

	return nil
}

func (r *fakeRegistry) Remove(id string) error {
	delete(r.entries, id)
	r.removed = append(r.removed, id)

	return nil
}

// fakeRegistryWithGetErr returns an error on Get (non-ErrNotFound).
type fakeRegistryWithGetErr struct {
	*fakeRegistry

	getErr error
}

func (r *fakeRegistryWithGetErr) Get(_ string) (*registry.InstructionEntry, error) {
	return nil, r.getErr
}

// fakeRegistryWithListErr returns an error on List.
type fakeRegistryWithListErr struct {
	*fakeRegistry

	listErr error
}

func (r *fakeRegistryWithListErr) List() ([]registry.InstructionEntry, error) {
	return nil, r.listErr
}

// fakeRegistryWithRemoveErr returns an error on Remove.
type fakeRegistryWithRemoveErr struct {
	*fakeRegistry

	removeErr error
}

func (r *fakeRegistryWithRemoveErr) Remove(_ string) error {
	return r.removeErr
}

// fakeRegistryWithSurfacingErr returns an error on RecordSurfacing.
type fakeRegistryWithSurfacingErr struct {
	*fakeRegistry

	surfacingErr error
}

func (r *fakeRegistryWithSurfacingErr) RecordSurfacing(_ string) error {
	return r.surfacingErr
}

// fakeSurfacingLogger is a test double for register.SurfacingLogger.
type fakeSurfacingLogger struct {
	events []surfacingEvent
}

func (l *fakeSurfacingLogger) LogSurfacing(
	_, mode string, timestamp time.Time,
) error {
	l.events = append(l.events, surfacingEvent{
		mode:      mode,
		timestamp: timestamp,
	})

	return nil
}

// fakeSurfacingLoggerWithErr returns an error on LogSurfacing.
type fakeSurfacingLoggerWithErr struct {
	err error
}

func (l *fakeSurfacingLoggerWithErr) LogSurfacing(_, _ string, _ time.Time) error {
	return l.err
}

type surfacingEvent struct {
	mode      string
	timestamp time.Time
}

func newFakeRegistry() *fakeRegistry {
	return &fakeRegistry{
		entries:    make(map[string]registry.InstructionEntry),
		registered: make([]registry.InstructionEntry, 0),
		surfaced:   make([]string, 0),
		removed:    make([]string, 0),
	}
}

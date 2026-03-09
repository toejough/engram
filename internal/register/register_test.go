package register_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/register"
	"engram/internal/registry"
)

// TestDiscoverClaudeMD_NonErrNotExistReadError verifies non-ErrNotExist read
// errors are logged but pipeline continues.
func TestDiscoverClaudeMD_NonErrNotExistReadError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	var stderrBuf strings.Builder

	readFile := func(path string) ([]byte, error) {
		if path == "/bad/CLAUDE.md" {
			return nil, errors.New("permission denied")
		}

		return nil, os.ErrNotExist
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(readFile),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(register.SourceConfig{
		ClaudeMDPaths: []string{"/bad/CLAUDE.md"},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderrBuf.String()).To(ContainSubstring("permission denied"))
}

// TestDiscoverMemoryMD_MissingFile verifies missing MEMORY.md is silently skipped.
func TestDiscoverMemoryMD_MissingFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(nil)),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(register.SourceConfig{
		MemoryMDPaths: []string{"/nonexistent/MEMORY.md"},
	})
	g.Expect(err).NotTo(HaveOccurred())

	listed, listErr := reg.List()
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(listed).To(BeEmpty())
}

// TestDiscoverMemoryMD_NonErrNotExistReadError verifies non-ErrNotExist
// read errors are logged.
func TestDiscoverMemoryMD_NonErrNotExistReadError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	var stderrBuf strings.Builder

	readFile := func(path string) ([]byte, error) {
		if path == "/bad/MEMORY.md" {
			return nil, errors.New("io error")
		}

		return nil, os.ErrNotExist
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(readFile),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(register.SourceConfig{
		MemoryMDPaths: []string{"/bad/MEMORY.md"},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderrBuf.String()).To(ContainSubstring("io error"))
}

// TestDiscoverMemoryMD_ValidFile verifies MEMORY.md files are discovered.
func TestDiscoverMemoryMD_ValidFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	files := map[string]string{
		"/project/MEMORY.md": "## User Preferences\n\n- Always be concise\n",
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(register.SourceConfig{
		MemoryMDPaths: []string{"/project/MEMORY.md"},
	})
	g.Expect(err).NotTo(HaveOccurred())

	listed, listErr := reg.List()
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(listed).ToNot(BeEmpty())
}

// TestDiscoverRules_ReadFileError verifies rule file read error is logged.
func TestDiscoverRules_ReadFileError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	var stderrBuf strings.Builder

	dirs := map[string][]os.DirEntry{
		"/rules": {
			fakeDirEntry{name: "bad.md", isDir: false},
		},
	}

	readFile := func(_ string) ([]byte, error) {
		return nil, errors.New("read error")
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(readFile),
		register.WithReadDir(fakeReadDir(dirs)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(register.SourceConfig{
		RulesDir: "/rules",
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderrBuf.String()).To(ContainSubstring("read error"))
}

// TestDiscoverSkills_GlobError verifies glob error is logged.
func TestDiscoverSkills_GlobError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	var stderrBuf strings.Builder

	globFn := func(_ string) ([]string, error) {
		return nil, errors.New("bad pattern")
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(nil)),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(globFn),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(register.SourceConfig{
		SkillsDir: "/skills",
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderrBuf.String()).To(ContainSubstring("bad pattern"))
}

// TestDiscoverSkills_ReadFileError verifies skill file read error is logged.
func TestDiscoverSkills_ReadFileError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	var stderrBuf strings.Builder

	globResults := map[string][]string{
		"/skills/*/SKILL.md": {"/skills/broken/SKILL.md"},
	}

	readFile := func(_ string) ([]byte, error) {
		return nil, errors.New("file corrupt")
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(readFile),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(fakeGlob(globResults)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(register.SourceConfig{
		SkillsDir: "/skills",
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderrBuf.String()).To(ContainSubstring("file corrupt"))
}

// TestRecordSurfacing_LoggerError verifies logger error is logged.
func TestRecordSurfacing_LoggerError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLoggerWithErr{err: errors.New("log write error")}

	var stderrBuf strings.Builder

	files := map[string]string{
		"/project/CLAUDE.md": "- Rule one\n",
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(register.SourceConfig{
		ClaudeMDPaths: []string{"/project/CLAUDE.md"},
	})
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

	files := map[string]string{
		"/project/CLAUDE.md": "- Rule one\n",
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(register.SourceConfig{
		ClaudeMDPaths: []string{"/project/CLAUDE.md"},
	})
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

	files := map[string]string{
		"/project/CLAUDE.md": "- Rule one\n",
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(register.SourceConfig{
		ClaudeMDPaths: []string{"/project/CLAUDE.md"},
	})
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

	files := map[string]string{
		"/rules/go.md": "## Updated Go rules\nUse gofmt always.",
	}

	dirs := map[string][]os.DirEntry{
		"/rules": {
			fakeDirEntry{name: "go.md", isDir: false},
		},
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(dirs)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(register.SourceConfig{
		RulesDir: "/rules",
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderrBuf.String()).To(ContainSubstring("cannot remove"))
}

// traces: T-270
func TestT270_DiscoverClaudeMDSources(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	files := map[string]string{
		"/project/CLAUDE.md": "- Use DI everywhere\n- No direct I/O\n",
		"/home/CLAUDE.md":    "- Be concise\n",
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(register.SourceConfig{
		ClaudeMDPaths: []string{"/project/CLAUDE.md", "/home/CLAUDE.md"},
	})
	g.Expect(err).NotTo(HaveOccurred())

	// Should have registered entries from both files.
	// CLAUDE.md with 2 bullets + CLAUDE.md with 1 bullet = 3 entries.
	listed, listErr := reg.List()
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(listed).To(HaveLen(3))

	// All should be claude-md source type.
	for _, entry := range listed {
		g.Expect(entry.SourceType).To(Equal("claude-md"))
	}
}

// traces: T-271
func TestT271_DiscoverRuleFiles(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	files := map[string]string{
		"/rules/go.md":     "## Go rules\nUse gofmt.",
		"/rules/python.md": "## Python rules\nUse black.",
	}

	dirs := map[string][]os.DirEntry{
		"/rules": {
			fakeDirEntry{name: "go.md", isDir: false},
			fakeDirEntry{name: "python.md", isDir: false},
		},
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(dirs)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(register.SourceConfig{
		RulesDir: "/rules",
	})
	g.Expect(err).NotTo(HaveOccurred())

	listed, listErr := reg.List()
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(listed).To(HaveLen(2))

	for _, entry := range listed {
		g.Expect(entry.SourceType).To(Equal("rule"))
	}
}

// traces: T-272
func TestT272_DiscoverSkillFiles(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	files := map[string]string{
		"/skills/commit/SKILL.md": "# Commit skill\nRun /commit.",
		"/skills/review/SKILL.md": "# Review skill\nRun /review.",
	}

	dirs := map[string][]os.DirEntry{
		"/skills": {
			fakeDirEntry{name: "commit", isDir: true},
			fakeDirEntry{name: "review", isDir: true},
		},
	}

	globResults := map[string][]string{
		"/skills/*/SKILL.md": {
			"/skills/commit/SKILL.md",
			"/skills/review/SKILL.md",
		},
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(dirs)),
		register.WithGlob(fakeGlob(globResults)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(register.SourceConfig{
		SkillsDir: "/skills",
	})
	g.Expect(err).NotTo(HaveOccurred())

	listed, listErr := reg.List()
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(listed).To(HaveLen(2))

	for _, entry := range listed {
		g.Expect(entry.SourceType).To(Equal("skill"))
	}
}

// traces: T-273
func TestT273_RegisterNewEntries(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	files := map[string]string{
		"/project/CLAUDE.md": "- Rule one\n- Rule two\n",
		"/rules/go.md":       "## Go rules\nUse gofmt.",
	}

	dirs := map[string][]os.DirEntry{
		"/rules": {
			fakeDirEntry{name: "go.md", isDir: false},
		},
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(dirs)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(register.SourceConfig{
		ClaudeMDPaths: []string{"/project/CLAUDE.md"},
		RulesDir:      "/rules",
	})
	g.Expect(err).NotTo(HaveOccurred())

	// 2 claude-md bullets + 1 rule = 3 entries registered.
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

	files := map[string]string{
		"/rules/go.md": "## Updated Go rules\nUse gofmt always.",
	}

	dirs := map[string][]os.DirEntry{
		"/rules": {
			fakeDirEntry{name: "go.md", isDir: false},
		},
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(dirs)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(register.SourceConfig{
		RulesDir: "/rules",
	})
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

	// Only go.md exists in the rules dir.
	files := map[string]string{
		"/rules/go.md": "## Go rules\nUse gofmt.",
	}

	dirs := map[string][]os.DirEntry{
		"/rules": {
			fakeDirEntry{name: "go.md", isDir: false},
		},
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(dirs)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(register.SourceConfig{
		RulesDir: "/rules",
	})
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
		register.WithReadFile(fakeReadFile(nil)),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	// Run with empty config — no sources discovered.
	err := registrar.Run(register.SourceConfig{})
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

	files := map[string]string{
		"/project/CLAUDE.md": "- Rule one\n",
		"/rules/go.md":       "## Go rules\nUse gofmt.",
	}

	dirs := map[string][]os.DirEntry{
		"/rules": {
			fakeDirEntry{name: "go.md", isDir: false},
		},
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(dirs)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(register.SourceConfig{
		ClaudeMDPaths: []string{"/project/CLAUDE.md"},
		RulesDir:      "/rules",
	})
	g.Expect(err).NotTo(HaveOccurred())

	// 1 claude-md bullet + 1 rule = 2 entries.
	// RecordSurfacing should be called for each.
	g.Expect(reg.surfaced).To(HaveLen(2))

	// SurfacingLogger should also be called for each.
	g.Expect(logger.events).To(HaveLen(2))

	for _, event := range logger.events {
		g.Expect(event.mode).To(Equal("session-start"))
		g.Expect(event.timestamp).To(Equal(fixedTime))
	}
}

// traces: T-278
func TestT278_MissingSourcePathsSilentlySkipped(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	// All paths will return os.ErrNotExist.
	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(nil)),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	err := registrar.Run(register.SourceConfig{
		ClaudeMDPaths: []string{"/nonexistent/CLAUDE.md"},
		RulesDir:      "/nonexistent/rules",
		SkillsDir:     "/nonexistent/skills",
	})
	g.Expect(err).NotTo(HaveOccurred())

	listed, listErr := reg.List()
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(listed).To(BeEmpty())
}

// traces: T-279
func TestT279_IdempotentSecondRunIsNoOp(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	reg := newFakeRegistry()
	logger := &fakeSurfacingLogger{}

	files := map[string]string{
		"/project/CLAUDE.md": "- Rule one\n- Rule two\n",
		"/rules/go.md":       "## Go rules\nUse gofmt.",
	}

	dirs := map[string][]os.DirEntry{
		"/rules": {
			fakeDirEntry{name: "go.md", isDir: false},
		},
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(dirs)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(io.Discard),
	)

	config := register.SourceConfig{
		ClaudeMDPaths: []string{"/project/CLAUDE.md"},
		RulesDir:      "/rules",
	}

	// First run.
	err := registrar.Run(config)
	g.Expect(err).NotTo(HaveOccurred())

	firstRegCount := len(reg.registered)
	g.Expect(firstRegCount).To(Equal(3)) // 2 claude-md + 1 rule

	// Reset tracking slices but keep entries.
	reg.registered = reg.registered[:0]
	reg.removed = reg.removed[:0]
	reg.surfaced = reg.surfaced[:0]
	logger.events = logger.events[:0]

	// Second run — same files.
	err = registrar.Run(config)
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

	files := map[string]string{
		"/project/CLAUDE.md": "- Rule one\n- Rule two\n- Rule three\n",
	}

	registrar := register.NewRegistrar(
		reg, logger,
		register.WithReadFile(fakeReadFile(files)),
		register.WithReadDir(fakeReadDir(nil)),
		register.WithGlob(fakeGlob(nil)),
		register.WithNow(func() time.Time { return fixedTime }),
		register.WithStderr(&stderrBuf),
	)

	err := registrar.Run(register.SourceConfig{
		ClaudeMDPaths: []string{"/project/CLAUDE.md"},
	})

	// Run should NOT return an error despite registry failures.
	g.Expect(err).NotTo(HaveOccurred())

	// Errors should be logged to stderr.
	stderrOutput := stderrBuf.String()
	g.Expect(stderrOutput).To(ContainSubstring("database locked"))

	// All 3 bullets should have been attempted (not short-circuited).
	// Since register fails, count stderr lines.
	errorLines := strings.Count(stderrOutput, "database locked")
	g.Expect(errorLines).To(Equal(3))
}

// unexported variables.
var (
	_         = fmt.Sprintf
	fixedTime = time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
)

// fakeDirEntry implements os.DirEntry for testing.
type fakeDirEntry struct {
	name  string
	isDir bool
}

//nolint:nilnil // test fake returns nil,nil intentionally
func (e fakeDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func (e fakeDirEntry) IsDir() bool { return e.isDir }

func (e fakeDirEntry) Name() string { return e.name }

func (e fakeDirEntry) Type() os.FileMode { return 0 }

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

// fakeGlob returns a fake glob function for known patterns.
func fakeGlob(results map[string][]string) func(string) ([]string, error) {
	return func(pattern string) ([]string, error) {
		matches, ok := results[pattern]
		if !ok {
			return nil, nil
		}

		return matches, nil
	}
}

// helper to build a fake readDir returning entries for known dirs.
func fakeReadDir(dirs map[string][]os.DirEntry) func(string) ([]os.DirEntry, error) {
	return func(path string) ([]os.DirEntry, error) {
		entries, ok := dirs[path]
		if !ok {
			return nil, os.ErrNotExist
		}

		return entries, nil
	}
}

// helper to build a fake readFile that returns content for known paths.
func fakeReadFile(files map[string]string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		content, ok := files[path]
		if !ok {
			return nil, os.ErrNotExist
		}

		return []byte(content), nil
	}
}

func newFakeRegistry() *fakeRegistry {
	return &fakeRegistry{
		entries:    make(map[string]registry.InstructionEntry),
		registered: make([]registry.InstructionEntry, 0),
		surfaced:   make([]string, 0),
		removed:    make([]string, 0),
	}
}

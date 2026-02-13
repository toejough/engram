package trace_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/trace"
	"pgregory.net/rapid"
)

// MockFS implements trace.FileSystem for testing.
type MockFS struct {
	Files map[string][]byte
	Dirs  map[string]bool
}

type mockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() os.FileMode  { return 0o644 }
func (m mockFileInfo) ModTime() time.Time { return time.Now() }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() interface{}   { return nil }

func (m *MockFS) ReadFile(path string) ([]byte, error) {
	content, exists := m.Files[path]
	if !exists {
		return nil, os.ErrNotExist
	}
	return content, nil
}

func (m *MockFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	m.Files[path] = data
	return nil
}

func (m *MockFS) Stat(path string) (os.FileInfo, error) {
	if m.Dirs[path] {
		return mockFileInfo{name: filepath.Base(path), isDir: true}, nil
	}
	content, exists := m.Files[path]
	if !exists {
		return nil, os.ErrNotExist
	}
	return mockFileInfo{name: filepath.Base(path), size: int64(len(content))}, nil
}

func (m *MockFS) MkdirAll(path string, perm os.FileMode) error {
	// Add all parent directories
	current := path
	for current != "." && current != "/" && current != "" {
		m.Dirs[current] = true
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return nil
}

func (m *MockFS) Walk(root string, walkFn filepath.WalkFunc) error {
	// Walk through all files and directories
	visited := make(map[string]bool)
	skipped := make(map[string]bool)

	// Collect and sort directories to ensure parents are visited before children
	var dirs []string
	for dir := range m.Dirs {
		if strings.HasPrefix(dir, root) {
			dirs = append(dirs, dir)
		}
	}
	// Simple bubble sort by path length (shorter paths = parent directories)
	for i := 0; i < len(dirs); i++ {
		for j := i + 1; j < len(dirs); j++ {
			if len(dirs[i]) > len(dirs[j]) {
				dirs[i], dirs[j] = dirs[j], dirs[i]
			}
		}
	}

	// Walk directories in sorted order
	for _, dir := range dirs {
		if visited[dir] {
			continue
		}

		// Check if this directory is inside a skipped directory
		insideSkipped := false
		for skipDir := range skipped {
			if strings.HasPrefix(dir, skipDir+string(filepath.Separator)) {
				insideSkipped = true
				break
			}
		}
		if insideSkipped {
			continue
		}

		visited[dir] = true
		info := mockFileInfo{name: filepath.Base(dir), isDir: true}
		if err := walkFn(dir, info, nil); err != nil {
			if err == filepath.SkipDir {
				skipped[dir] = true
				continue
			}
			return err
		}
	}

	// Then walk files
	for path := range m.Files {
		if strings.HasPrefix(path, root) && !visited[path] {
			// Check if this file is inside a skipped directory
			insideSkipped := false
			for skipDir := range skipped {
				if strings.HasPrefix(path, skipDir+string(filepath.Separator)) {
					insideSkipped = true
					break
				}
			}
			if insideSkipped {
				continue
			}

			visited[path] = true
			content := m.Files[path]
			info := mockFileInfo{name: filepath.Base(path), size: int64(len(content))}
			if err := walkFn(path, info, nil); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *MockFS) Glob(pattern string) ([]string, error) {
	var matches []string
	for path := range m.Files {
		matched, err := filepath.Match(pattern, path)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, path)
		}
	}
	return matches, nil
}

func TestValidID(t *testing.T) {
	t.Parallel()
	t.Run("accepts valid IDs", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(trace.ValidID("ISSUE-1")).To(BeTrue())
		g.Expect(trace.ValidID("REQ-1")).To(BeTrue())
		g.Expect(trace.ValidID("DES-042")).To(BeTrue())
		g.Expect(trace.ValidID("ARCH-123")).To(BeTrue())
		g.Expect(trace.ValidID("TASK-007")).To(BeTrue())
		// Simple incrementing numbers are valid
		g.Expect(trace.ValidID("REQ-1")).To(BeTrue())
		g.Expect(trace.ValidID("REQ-42")).To(BeTrue())
		g.Expect(trace.ValidID("REQ-0001")).To(BeTrue())
	})

	t.Run("rejects invalid IDs", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(trace.ValidID("CONF-001")).To(BeFalse()) // invalid prefix
		g.Expect(trace.ValidID("req-001")).To(BeFalse())  // lowercase
		g.Expect(trace.ValidID("REQ001")).To(BeFalse())   // missing hyphen
		g.Expect(trace.ValidID("")).To(BeFalse())         // empty
		g.Expect(trace.ValidID("REQ-")).To(BeFalse())     // missing number
		g.Expect(trace.ValidID("-001")).To(BeFalse())     // missing prefix
	})
}

func TestValidIDProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		prefix := rapid.SampledFrom([]string{"ISSUE", "REQ", "DES", "ARCH", "TASK"}).Draw(rt, "prefix")
		num := rapid.IntRange(0, 999).Draw(rt, "num")

		id := prefix + "-" + padNumber(num)
		g.Expect(trace.ValidID(id)).To(BeTrue())
	})
}

func padNumber(n int) string {
	return fmt.Sprintf("%d", n)
}

func TestAdd(t *testing.T) {
	t.Parallel()
	t.Run("adds link to empty matrix", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		err := trace.Add(dir, "REQ-1", []string{"DES-1", "ARCH-1"}, fs)
		g.Expect(err).ToNot(HaveOccurred())

		// File should exist
		_, err = fs.Stat(filepath.Join(dir, trace.TraceFile))
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("appends to existing link", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		err := trace.Add(dir, "REQ-1", []string{"DES-1"}, fs)
		g.Expect(err).ToNot(HaveOccurred())

		err = trace.Add(dir, "REQ-1", []string{"ARCH-1"}, fs)
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("rejects duplicate link", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		err := trace.Add(dir, "REQ-1", []string{"DES-1"}, fs)
		g.Expect(err).ToNot(HaveOccurred())

		err = trace.Add(dir, "REQ-1", []string{"DES-1"}, fs)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("duplicate"))
	})

	t.Run("rejects invalid source ID", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		err := trace.Add(dir, "INVALID", []string{"DES-1"}, fs)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("invalid source"))
	})

	t.Run("rejects invalid target ID", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		err := trace.Add(dir, "REQ-1", []string{"bad"}, fs)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("invalid target"))
	})

	t.Run("supports comma-separated targets", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		err := trace.Add(dir, "REQ-002", []string{"DES-1", "DES-2", "ARCH-1"}, fs)
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("accepts ISSUE linking to REQ", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		err := trace.Add(dir, "ISSUE-1", []string{"REQ-1"}, fs)
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("rejects ISSUE linking to non-REQ", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		err := trace.Add(dir, "ISSUE-1", []string{"DES-1"}, fs)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("ISSUE can only link to REQ"))
	})

	t.Run("rejects ISSUE linking to ARCH", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		err := trace.Add(dir, "ISSUE-1", []string{"ARCH-1"}, fs)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("ISSUE can only link to REQ"))
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()
	t.Run("passes with complete coverage", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// Create artifacts with IDs
		writeArtifact(t, fs, dir, "requirements.md", "### REQ-001: Feature\n")
		writeArtifact(t, fs, dir, "architecture.md", "### ARCH-001: Decision\n")
		writeArtifact(t, fs, dir, "tasks.md", "### TASK-001: Implement\n")

		// Add trace links
		g.Expect(trace.Add(dir, "REQ-1", []string{"ARCH-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-1", []string{"TASK-1"}, fs)).To(Succeed())

		result, err := trace.Validate(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeTrue())
		g.Expect(result.OrphanIDs).To(BeEmpty())
		g.Expect(result.UnlinkedIDs).To(BeEmpty())
		g.Expect(result.MissingCoverage).To(BeEmpty())
	})

	t.Run("detects unlinked IDs", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeArtifact(t, fs, dir, "requirements.md", "### REQ-001: Feature\n### REQ-002: Other\n")
		writeArtifact(t, fs, dir, "architecture.md", "### ARCH-001: Decision\n")
		writeArtifact(t, fs, dir, "tasks.md", "### TASK-001: Implement\n")

		g.Expect(trace.Add(dir, "REQ-1", []string{"ARCH-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-1", []string{"TASK-1"}, fs)).To(Succeed())

		result, err := trace.Validate(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeFalse())
		g.Expect(result.UnlinkedIDs).To(ContainElement("REQ-2"))
	})

	t.Run("detects missing coverage", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeArtifact(t, fs, dir, "requirements.md", "### REQ-001: Feature\n")
		writeArtifact(t, fs, dir, "architecture.md", "### ARCH-001: Decision\n")
		writeArtifact(t, fs, dir, "tasks.md", "### TASK-001: Implement\n")

		// REQ-001 → ARCH-001 but ARCH-001 has no TASK link
		g.Expect(trace.Add(dir, "REQ-1", []string{"ARCH-1"}, fs)).To(Succeed())

		result, err := trace.Validate(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeFalse())
		g.Expect(result.MissingCoverage).ToNot(BeEmpty())
	})

	t.Run("REQ to ARCH satisfies coverage (DES or ARCH rule)", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeArtifact(t, fs, dir, "requirements.md", "### REQ-001: Feature\n")
		writeArtifact(t, fs, dir, "architecture.md", "### ARCH-001: Decision\n")
		writeArtifact(t, fs, dir, "tasks.md", "### TASK-001: Implement\n")

		// REQ→ARCH is sufficient (DES is not required for every REQ)
		g.Expect(trace.Add(dir, "REQ-1", []string{"ARCH-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-1", []string{"TASK-1"}, fs)).To(Succeed())

		result, err := trace.Validate(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeTrue())
	})

	t.Run("detects ISSUE IDs in issues.md", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// Create artifacts including issues.md
		writeArtifact(t, fs, dir, "issues.md", "### ISSUE-1: Bug report\n")
		writeArtifact(t, fs, dir, "requirements.md", "### REQ-001: Feature\n")
		writeArtifact(t, fs, dir, "architecture.md", "### ARCH-001: Decision\n")
		writeArtifact(t, fs, dir, "tasks.md", "### TASK-001: Implement\n")

		// Add trace links including ISSUE
		g.Expect(trace.Add(dir, "ISSUE-1", []string{"REQ-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "REQ-1", []string{"ARCH-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-1", []string{"TASK-1"}, fs)).To(Succeed())

		result, err := trace.Validate(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		// ISSUE-1 should be detected in issues.md, so no orphans
		g.Expect(result.OrphanIDs).ToNot(ContainElement("ISSUE-1"))
	})

	t.Run("ISSUE with no downstream passes (optional coverage)", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// ISSUE exists but has no downstream links
		writeArtifact(t, fs, dir, "issues.md", "### ISSUE-1: Bug report\n")

		// No links from ISSUE-1

		result, err := trace.Validate(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		// ISSUE should NOT appear in MissingCoverage (no mandatory downstream)
		for _, mc := range result.MissingCoverage {
			g.Expect(mc.ID).ToNot(HavePrefix("ISSUE-"), "ISSUE should not have coverage requirements")
		}
	})

	t.Run("ISSUE with downstream REQ passes coverage", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeArtifact(t, fs, dir, "issues.md", "### ISSUE-1: Bug report\n")
		writeArtifact(t, fs, dir, "requirements.md", "### REQ-001: Feature\n")
		writeArtifact(t, fs, dir, "architecture.md", "### ARCH-001: Decision\n")
		writeArtifact(t, fs, dir, "tasks.md", "### TASK-001: Implement\n")

		// Complete chain starting from ISSUE
		g.Expect(trace.Add(dir, "ISSUE-1", []string{"REQ-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "REQ-1", []string{"ARCH-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-1", []string{"TASK-1"}, fs)).To(Succeed())

		result, err := trace.Validate(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeTrue())
	})
}

func TestImpact(t *testing.T) {
	t.Parallel()
	t.Run("forward impact", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		g.Expect(trace.Add(dir, "REQ-1", []string{"DES-1", "ARCH-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-1", []string{"TASK-1", "TASK-002"}, fs)).To(Succeed())

		result, err := trace.Impact(dir, "REQ-1", false, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.AffectedIDs).To(ContainElements("DES-1", "ARCH-1", "TASK-1", "TASK-002"))
		g.Expect(result.Reverse).To(BeFalse())
	})

	t.Run("backward impact", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		g.Expect(trace.Add(dir, "REQ-1", []string{"ARCH-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-1", []string{"TASK-1"}, fs)).To(Succeed())

		result, err := trace.Impact(dir, "TASK-1", true, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.AffectedIDs).To(ContainElements("ARCH-1", "REQ-1"))
		g.Expect(result.Reverse).To(BeTrue())
	})

	t.Run("handles cycles gracefully", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// Shouldn't happen but test it doesn't infinite loop
		g.Expect(trace.Add(dir, "REQ-1", []string{"ARCH-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-1", []string{"REQ-1"}, fs)).To(Succeed())

		result, err := trace.Impact(dir, "REQ-1", false, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.AffectedIDs).To(ContainElement("ARCH-1"))
	})

	t.Run("rejects invalid ID", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		_, err := trace.Impact(dir, "bad-id", false, fs)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("forward impact from ISSUE", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		g.Expect(trace.Add(dir, "ISSUE-1", []string{"REQ-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "REQ-1", []string{"ARCH-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-1", []string{"TASK-1"}, fs)).To(Succeed())

		result, err := trace.Impact(dir, "ISSUE-1", false, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.AffectedIDs).To(ContainElements("REQ-1", "ARCH-1", "TASK-1"))
		g.Expect(result.Reverse).To(BeFalse())
	})

	t.Run("backward impact to ISSUE", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		g.Expect(trace.Add(dir, "ISSUE-1", []string{"REQ-1"}, fs)).To(Succeed())
		g.Expect(trace.Add(dir, "REQ-1", []string{"ARCH-1"}, fs)).To(Succeed())

		result, err := trace.Impact(dir, "ARCH-1", true, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.AffectedIDs).To(ContainElements("REQ-1", "ISSUE-1"))
		g.Expect(result.Reverse).To(BeTrue())
	})
}

func writeArtifact(t *testing.T, fs *MockFS, dir, name, content string) {
	t.Helper()

	// Write to project root directory
	path := filepath.Join(dir, name)
	fs.Files[path] = []byte(content)
}

func writeTestFile(t *testing.T, fs *MockFS, dir, pkg, name, content string) {
	t.Helper()

	// Write Go test file to specified package directory
	pkgDir := filepath.Join(dir, pkg)

	// Add all parent directories to Dirs
	current := pkgDir
	for current != dir && current != "." && current != "/" {
		fs.Dirs[current] = true
		current = filepath.Dir(current)
	}

	path := filepath.Join(pkgDir, name)
	fs.Files[path] = []byte(content)
}

// TEST-230 traces: TASK-003
func TestRepair(t *testing.T) {
	t.Parallel()
	t.Run("detects dangling references", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// Create a design doc that references a non-existent requirement
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Some Design

Design description.

**Traces to:** REQ-999
`)

		result, err := trace.Repair(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.DanglingRefs).To(ContainElement("REQ-999"))
	})

	t.Run("detects duplicate IDs in different files", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// Create two design files with same ID
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: First Design

First.

**Traces to:** REQ-001
`)
		writeArtifact(t, fs, dir, "design-feature.md", `# Feature Design

### DES-001: Duplicate Design

Duplicate.

**Traces to:** REQ-002
`)

		result, err := trace.Repair(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.DuplicateIDs).To(ContainElement("DES-001"))
	})

	t.Run("returns empty result when no issues", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-001: Requirement

Description.
`)
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Design

Description.

**Traces to:** REQ-001
`)

		result, err := trace.Repair(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.DanglingRefs).To(BeEmpty())
		g.Expect(result.DuplicateIDs).To(BeEmpty())
	})

	t.Run("reports all issues found", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: First

First.

**Traces to:** REQ-999, REQ-998
`)

		result, err := trace.Repair(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.DanglingRefs).To(HaveLen(2))
		g.Expect(result.DanglingRefs).To(ContainElements("REQ-999", "REQ-998"))
	})

	t.Run("auto-renumbers duplicate IDs", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: First Design

First.

**Traces to:** REQ-001
`)
		writeArtifact(t, fs, dir, "design-feature.md", `# Feature Design

### DES-001: Duplicate Design

Duplicate.

**Traces to:** REQ-002
`)

		result, err := trace.Repair(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())

		// Should have renumbered the duplicate
		g.Expect(result.Renumbered).To(HaveLen(1))
		g.Expect(result.Renumbered[0].OldID).To(Equal("DES-001"))
		g.Expect(result.Renumbered[0].NewID).To(Equal("DES-2"))

		// Check the file was actually updated
		content, err := fs.ReadFile(filepath.Join(dir, "design-feature.md"))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).To(ContainSubstring("DES-2"))
		g.Expect(string(content)).ToNot(ContainSubstring("DES-1"))
	})

	t.Run("creates escalation for dangling refs", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Some Design

Design.

**Traces to:** REQ-999
`)

		result, err := trace.Repair(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())

		// Should create escalation for dangling ref
		g.Expect(result.Escalations).To(HaveLen(1))
		g.Expect(result.Escalations[0].ID).To(Equal("REQ-999"))
		g.Expect(result.Escalations[0].Reason).To(ContainSubstring("dangling"))
	})
}

// TEST-240: Validates orphan detection by scanning artifact Traces to: fields
// traces: TASK-007
func TestValidateOrphanDetection(t *testing.T) {
	t.Parallel()
	t.Run("orphan is ID in Traces to but not defined", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// DES-001 is defined but references undefined REQ-999
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Some Design

**Traces to:** REQ-999
`)

		result, err := trace.ValidateV2Artifacts(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeFalse())
		g.Expect(result.OrphanIDs).To(ContainElement("REQ-999"), "REQ-999 is referenced but not defined")
	})

	t.Run("unlinked is ID defined but nothing traces to it", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// DES-001: ARCH-001 traces to it (linked)
		// DES-002: nothing traces to it (unlinked - should be flagged)
		// REQ-001, REQ-002: nothing traces to them but that's OK (REQs can be roots)
		writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-001: First Requirement

Description.

### REQ-002: Second Requirement

Description.
`)
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Design

**Traces to:** REQ-001

### DES-002: Another Design

**Traces to:** REQ-002
`)
		writeArtifact(t, fs, dir, "architecture.md", `# Architecture

### ARCH-001: Architecture

**Traces to:** DES-001
`)

		result, err := trace.ValidateV2Artifacts(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeFalse())
		g.Expect(result.UnlinkedIDs).To(ContainElement("DES-2"), "DES-002 has nothing tracing to it")
		g.Expect(result.UnlinkedIDs).To(ContainElement("ARCH-1"), "ARCH-001 has nothing tracing to it")
		g.Expect(result.UnlinkedIDs).ToNot(ContainElement("DES-1"), "DES-001 is traced to by ARCH-001")
		g.Expect(result.UnlinkedIDs).ToNot(ContainElement("REQ-1"), "REQ can be root")
		g.Expect(result.UnlinkedIDs).ToNot(ContainElement("REQ-002"), "REQ can be root - issues are optional")
	})

	t.Run("passes when all IDs are defined and traced", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-001: Requirement

Description.
`)
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Design

**Traces to:** REQ-001
`)
		writeArtifact(t, fs, dir, "architecture.md", `# Architecture

### ARCH-001: Architecture

**Traces to:** DES-001
`)

		result, err := trace.ValidateV2Artifacts(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.OrphanIDs).To(BeEmpty())
		// Note: REQ-001 has nothing tracing to it, but that's OK for top-level items
	})

	t.Run("TEST in source file with traces comment is linked", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// Create a Go test file with TEST-NNN comment
		writeTestFile(t, fs, dir, "internal/parser", "parser_test.go", `package parser_test

import "testing"

// TEST-901: Parses positional arguments
// traces: TASK-001
func TestParsePositionalArgs(t *testing.T) {
	// test implementation
}

// TEST-902: Missing traces comment
func TestSomethingElse(t *testing.T) {
	// no traces comment - should be unlinked
}
`)

		writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement parsing

Description.
`)

		result, err := trace.ValidateV2Artifacts(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.UnlinkedIDs).ToNot(ContainElement("TEST-901"), "TEST-901 has traces: comment")
		// Note: TEST-902 would be unlinked if we detect it, but we only detect commented tests
	})

	t.Run("TEST in source file without traces comment is unlinked", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// Create a Go test file with TEST-NNN but no traces
		writeTestFile(t, fs, dir, "internal/config", "config_test.go", `package config_test

import "testing"

// TEST-903: Config loading test
// No traces comment here
func TestConfigLoad(t *testing.T) {
	// test implementation
}
`)

		result, err := trace.ValidateV2Artifacts(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeFalse())
		g.Expect(result.UnlinkedIDs).To(ContainElement("TEST-903"), "TEST-903 has no traces: comment")
	})

	t.Run("scans test files from multiple packages", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeTestFile(t, fs, dir, "internal/parser", "parser_test.go", `package parser_test

import "testing"

// TEST-910: Parser test
// traces: TASK-001
func TestParser(t *testing.T) {}
`)

		writeTestFile(t, fs, dir, "internal/config", "config_test.go", `package config_test

import "testing"

// TEST-911: Config test
// traces: TASK-002
func TestConfig(t *testing.T) {}
`)

		writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Parser task

### TASK-002: Config task
`)

		result, err := trace.ValidateV2Artifacts(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.UnlinkedIDs).ToNot(ContainElement("TEST-910"))
		g.Expect(result.UnlinkedIDs).ToNot(ContainElement("TEST-911"))
	})
}

// TEST-005: Show function returns ASCII tree representation
// traces: TASK-005
// TEST-300: normalizeID strips leading zeros from trace IDs
// traces: TASK-1
func TestNormalizeID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"strips single leading zero", "REQ-013", "REQ-13"},
		{"strips multiple leading zeros", "REQ-0013", "REQ-13"},
		{"preserves non-zero", "REQ-13", "REQ-13"},
		{"preserves single digit", "REQ-1", "REQ-1"},
		{"handles zero", "REQ-0", "REQ-0"},
		{"handles ARCH prefix", "ARCH-060", "ARCH-60"},
		{"handles TASK prefix", "TASK-1", "TASK-1"},
		{"handles TEST prefix", "TEST-042", "TEST-42"},
		{"handles DES prefix", "DES-007", "DES-7"},
		{"handles ISSUE prefix", "ISSUE-003", "ISSUE-3"},
		{"preserves malformed ID", "INVALID", "INVALID"},
		{"preserves ID without hyphen", "REQ001", "REQ001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := trace.NormalizeID(tt.input)
			g.Expect(result).To(Equal(tt.expected))
		})
	}
}

// TEST-301: ValidateV2Artifacts normalizes padded IDs to prevent false orphans
// traces: TASK-1
func TestValidateV2Artifacts_PaddedIDs(t *testing.T) {
	t.Parallel()
	t.Run("padded and non-padded IDs match", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// REQ-13 is defined (non-padded)
		// DES-001 references REQ-013 (padded)
		// ARCH-001 references DES-001 (padded) to complete the chain
		// Should normalize to REQ-13 and not report orphan
		writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-13: Feature

Description.
`)
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Design

**Traces to:** REQ-013
`)
		writeArtifact(t, fs, dir, "architecture.md", `# Architecture

### ARCH-001: Architecture

**Traces to:** DES-001
`)

		result, err := trace.ValidateV2Artifacts(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.OrphanIDs).To(BeEmpty(), "no orphans - REQ-013 normalizes to REQ-13 which is defined")
		// DES-001 and ARCH-001 will be unlinked without TASK, but no orphans
	})

	t.Run("mixed padding formats are treated as same ID", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// Mix of REQ-013, REQ-13, and REQ-0013 all normalize to REQ-13
		// Focus: no false orphans from padding mismatch
		writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-013: Feature

Description.
`)
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Design One

**Traces to:** REQ-13

### DES-002: Design Two

**Traces to:** REQ-0013
`)
		writeArtifact(t, fs, dir, "architecture.md", `# Architecture

### ARCH-001: Arch One

**Traces to:** DES-001

### ARCH-002: Arch Two

**Traces to:** DES-002
`)

		result, err := trace.ValidateV2Artifacts(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.OrphanIDs).To(BeEmpty(), "no orphans - REQ-13, REQ-013, REQ-0013 all normalize to REQ-13")
	})

	t.Run("normalizes test file trace targets", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// TEST-001 references TASK-013 (padded)
		// TASK-13 is defined (non-padded)
		writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", `package feature_test

import "testing"

// TEST-001: Feature test
// traces: TASK-013
func TestFeature(t *testing.T) {}
`)

		writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-13: Implement feature

Description.
`)

		result, err := trace.ValidateV2Artifacts(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeTrue(), "test trace target should normalize")
		g.Expect(result.OrphanIDs).ToNot(ContainElement("TASK-013"), "TASK-013 should normalize to TASK-13")
	})

	t.Run("normalizes test IDs", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// TEST-042 (padded) should normalize to TEST-42
		writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", `package feature_test

import "testing"

// TEST-042: Feature test
// traces: TASK-13
func TestFeature(t *testing.T) {}
`)

		writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-13: Implement feature

Description.
`)

		result, err := trace.ValidateV2Artifacts(dir, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeTrue())
		g.Expect(result.UnlinkedIDs).ToNot(ContainElement("TEST-042"), "TEST-042 should normalize to TEST-42")
	})
}

func TestShow(t *testing.T) {
	t.Parallel()
	t.Run("returns ASCII tree for simple chain", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-001: Feature

Description.
`)
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Design

**Traces to:** REQ-001
`)
		writeArtifact(t, fs, dir, "architecture.md", `# Architecture

### ARCH-001: Architecture

**Traces to:** DES-001
`)
		writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implementation

**Traces to:** ARCH-001
`)

		output, err := trace.Show(dir, "ascii", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(output).To(ContainSubstring("REQ-1"))
		g.Expect(output).To(ContainSubstring("DES-1"))
		g.Expect(output).To(ContainSubstring("ARCH-1"))
		g.Expect(output).To(ContainSubstring("TASK-1"))
	})

	t.Run("marks orphan IDs in ASCII output", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// DES-001 references undefined REQ-999
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Design

**Traces to:** REQ-999
`)

		output, err := trace.Show(dir, "ascii", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(output).To(ContainSubstring("REQ-999"))
		g.Expect(output).To(ContainSubstring("[ORPHAN]"))
	})

	t.Run("marks unlinked IDs in ASCII output", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// DES-002 is defined but nothing traces to it
		writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-001: Feature

Description.
`)
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Design

**Traces to:** REQ-001

### DES-002: Orphan Design

**Traces to:** REQ-001
`)
		writeArtifact(t, fs, dir, "architecture.md", `# Architecture

### ARCH-001: Architecture

**Traces to:** DES-001
`)

		output, err := trace.Show(dir, "ascii", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(output).To(ContainSubstring("DES-2"))
		g.Expect(output).To(ContainSubstring("[UNLINKED]"))
	})

	t.Run("returns JSON graph format", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-001: Feature

Description.
`)
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Design

**Traces to:** REQ-001
`)

		output, err := trace.Show(dir, "json", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(output).To(ContainSubstring(`"nodes"`))
		g.Expect(output).To(ContainSubstring(`"edges"`))
		g.Expect(output).To(ContainSubstring(`"REQ-1"`))
		g.Expect(output).To(ContainSubstring(`"DES-1"`))
	})

	t.Run("JSON includes orphan and unlinked markers", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		// Create a design that references undefined REQ-999
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Design

**Traces to:** REQ-999

### DES-002: Unlinked Design

**Traces to:** REQ-999
`)

		output, err := trace.Show(dir, "json", fs)
		g.Expect(err).ToNot(HaveOccurred())
		// JSON uses spaces around colon in pretty-printed format
		g.Expect(output).To(ContainSubstring(`"orphan": true`))
		g.Expect(output).To(ContainSubstring(`"unlinked": true`))
	})

	t.Run("rejects invalid format", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		_, err := trace.Show(dir, "invalid", fs)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("invalid format"))
	})

	t.Run("handles empty directory", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		output, err := trace.Show(dir, "ascii", fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(output).ToNot(BeEmpty())
	})

	t.Run("ASCII tree shows hierarchical structure", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
		dir := t.TempDir()

		writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-001: Feature

Description.
`)
		writeArtifact(t, fs, dir, "design.md", `# Design

### DES-001: Design

**Traces to:** REQ-001
`)
		writeArtifact(t, fs, dir, "architecture.md", `# Architecture

### ARCH-001: Architecture

**Traces to:** DES-001

### ARCH-002: Another Arch

**Traces to:** DES-001
`)

		output, err := trace.Show(dir, "ascii", fs)
		g.Expect(err).ToNot(HaveOccurred())
		// Should show tree structure with indentation
		g.Expect(output).To(MatchRegexp(`REQ-1.*\n.*DES-1`))
	})
}

package trace_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/trace"
)

// TEST-300: Promote finds test files with TASK traces
// traces: TASK-007
func TestPromote_FindsTaskTraces(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Create tasks.md with TASK-001 tracing to ARCH-001
	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`)

	// Create a Go test file with TASK trace
	writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", `package feature_test

import "testing"

// TEST-100: Feature test
// traces: TASK-001
func TestFeature(t *testing.T) {
}
`)

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(HaveLen(1))
	g.Expect(result.Promotions[0].File).To(Equal("internal/feature/feature_test.go"))
	g.Expect(result.Promotions[0].OldTrace).To(Equal("TASK-001"))
	g.Expect(result.Promotions[0].NewTrace).To(Equal("ARCH-001"))
}

// TEST-301: Promote updates test file content
// traces: TASK-007
func TestPromote_UpdatesFileContent(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`)

	writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", `package feature_test

import "testing"

// TEST-100: Feature test
// traces: TASK-001
func TestFeature(t *testing.T) {
}
`)

	_, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())

	// Read the file back and verify content changed
	content, err := fs.ReadFile(filepath.Join(dir, "internal/feature/feature_test.go"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("// traces: ARCH-001"))
	g.Expect(string(content)).ToNot(ContainSubstring("// traces: TASK-001"))
}

// TEST-302: Promote handles multiple tasks in same file
// traces: TASK-007
func TestPromote_MultipleTasksInFile(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: First feature

**Traces to:** ARCH-001

### TASK-002: Second feature

**Traces to:** ARCH-002
`)

	writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", `package feature_test

import "testing"

// TEST-100: First test
// traces: TASK-001
func TestFirst(t *testing.T) {
}

// TEST-101: Second test
// traces: TASK-002
func TestSecond(t *testing.T) {
}
`)

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(HaveLen(2))
}

// TEST-303: Promote handles TypeScript test files
// traces: TASK-007
func TestPromote_TypeScriptFiles(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`)

	// Create TypeScript test file
	tsTestContent := `// TEST-100: Feature test
// traces: TASK-001
describe('Feature', () => {
  it('should work', () => {
    expect(true).toBe(true);
  });
});
`
	pkgDir := filepath.Join(dir, "src")
	g.Expect(fs.MkdirAll(pkgDir, 0o755)).To(Succeed())
	g.Expect(fs.WriteFile(filepath.Join(pkgDir, "feature.test.ts"), []byte(tsTestContent), 0o644)).To(Succeed())

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(HaveLen(1))
	g.Expect(result.Promotions[0].File).To(Equal("src/feature.test.ts"))
}

// TEST-304: Promote handles JavaScript test files
// traces: TASK-007
func TestPromote_JavaScriptFiles(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`)

	// Create JavaScript test file
	jsTestContent := `// TEST-100: Feature test
// traces: TASK-001
describe('Feature', () => {
  it('should work', () => {
    expect(true).toBe(true);
  });
});
`
	pkgDir := filepath.Join(dir, "src")
	g.Expect(fs.MkdirAll(pkgDir, 0o755)).To(Succeed())
	g.Expect(fs.WriteFile(filepath.Join(pkgDir, "feature.test.js"), []byte(jsTestContent), 0o644)).To(Succeed())

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(HaveLen(1))
	g.Expect(result.Promotions[0].File).To(Equal("src/feature.test.js"))
}

// TEST-305: Promote skips non-TASK traces
// traces: TASK-007
func TestPromote_SkipsNonTaskTraces(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`)

	// Test file already has permanent trace (ARCH)
	writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", `package feature_test

import "testing"

// TEST-100: Feature test
// traces: ARCH-001
func TestFeature(t *testing.T) {
}
`)

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(BeEmpty())
}

// TEST-306: Promote handles task with no Traces-to
// traces: TASK-007
func TestPromote_TaskWithNoTracesTo(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Task has no Traces-to field
	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

No traces to field here.
`)

	writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", `package feature_test

import "testing"

// TEST-100: Feature test
// traces: TASK-001
func TestFeature(t *testing.T) {
}
`)

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	// Should report as skipped, not promoted
	g.Expect(result.Promotions).To(BeEmpty())
	g.Expect(result.Skipped).To(HaveLen(1))
	g.Expect(result.Skipped[0].Reason).To(ContainSubstring("no Traces-to"))
}

// TEST-307: Promote handles task not found in tasks.md
// traces: TASK-007
func TestPromote_TaskNotFound(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// tasks.md doesn't have TASK-999
	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`)

	writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", `package feature_test

import "testing"

// TEST-100: Feature test
// traces: TASK-999
func TestFeature(t *testing.T) {
}
`)

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(BeEmpty())
	g.Expect(result.Skipped).To(HaveLen(1))
	g.Expect(result.Skipped[0].Reason).To(ContainSubstring("not found"))
}

// TEST-308: Promote handles multiple trace targets
// traces: TASK-007
func TestPromote_MultipleTraceTargets(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Task traces to multiple targets
	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001, ARCH-002
`)

	writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", `package feature_test

import "testing"

// TEST-100: Feature test
// traces: TASK-001
func TestFeature(t *testing.T) {
}
`)

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(HaveLen(1))
	// Should promote to all targets
	g.Expect(result.Promotions[0].NewTrace).To(Equal("ARCH-001, ARCH-002"))

	// Verify file content
	content, err := fs.ReadFile(filepath.Join(dir, "internal/feature/feature_test.go"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("// traces: ARCH-001, ARCH-002"))
}

// TEST-309: Promote returns empty result when no test files
// traces: TASK-007
func TestPromote_NoTestFiles(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`)

	// No test files

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(BeEmpty())
	g.Expect(result.Skipped).To(BeEmpty())
}

// TEST-310: Promote handles .spec.ts files (alternate TS test pattern)
// traces: TASK-007
func TestPromote_SpecTsFiles(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`)

	// Create .spec.ts test file
	specContent := `// TEST-100: Feature spec
// traces: TASK-001
describe('Feature', () => {
  it('should work', () => {
    expect(true).toBe(true);
  });
});
`
	pkgDir := filepath.Join(dir, "src")
	g.Expect(fs.MkdirAll(pkgDir, 0o755)).To(Succeed())
	g.Expect(fs.WriteFile(filepath.Join(pkgDir, "feature.spec.ts"), []byte(specContent), 0o644)).To(Succeed())

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(HaveLen(1))
	g.Expect(result.Promotions[0].File).To(Equal("src/feature.spec.ts"))
}

// TEST-311: Promote handles .spec.js files
// traces: TASK-007
func TestPromote_SpecJsFiles(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`)

	// Create .spec.js test file
	specContent := `// TEST-100: Feature spec
// traces: TASK-001
describe('Feature', () => {
  it('should work', () => {
    expect(true).toBe(true);
  });
});
`
	pkgDir := filepath.Join(dir, "src")
	g.Expect(fs.MkdirAll(pkgDir, 0o755)).To(Succeed())
	g.Expect(fs.WriteFile(filepath.Join(pkgDir, "feature.spec.js"), []byte(specContent), 0o644)).To(Succeed())

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(HaveLen(1))
	g.Expect(result.Promotions[0].File).To(Equal("src/feature.spec.js"))
}

// TEST-312: Promote skips vendor directory
// traces: TASK-007
func TestPromote_SkipsVendorDir(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`)

	// Create test file in vendor directory (should be skipped)
	writeTestFile(t, fs, dir, "vendor/somelib", "lib_test.go", `package somelib_test

import "testing"

// TEST-100: Vendor test
// traces: TASK-001
func TestVendor(t *testing.T) {
}
`)

	// Create test file in normal directory (should be processed)
	writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", `package feature_test

import "testing"

// TEST-101: Feature test
// traces: TASK-001
func TestFeature(t *testing.T) {
}
`)

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(HaveLen(1))
	g.Expect(result.Promotions[0].File).To(Equal("internal/feature/feature_test.go"))
}

// TEST-313: Promote skips node_modules directory
// traces: TASK-007
func TestPromote_SkipsNodeModules(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`)

	// Create test file in node_modules (should be skipped)
	nmDir := filepath.Join(dir, "node_modules", "somelib")
	g.Expect(fs.MkdirAll(nmDir, 0o755)).To(Succeed())
	g.Expect(fs.WriteFile(filepath.Join(nmDir, "lib.test.js"), []byte(`// TEST-100: Vendor test
// traces: TASK-001
describe('test', () => {});
`), 0o644)).To(Succeed())

	// Create test file in normal directory
	srcDir := filepath.Join(dir, "src")
	g.Expect(fs.MkdirAll(srcDir, 0o755)).To(Succeed())
	g.Expect(fs.WriteFile(filepath.Join(srcDir, "feature.test.js"), []byte(`// TEST-101: Feature test
// traces: TASK-001
describe('feature', () => {});
`), 0o644)).To(Succeed())

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(HaveLen(1))
	g.Expect(result.Promotions[0].File).To(Equal("src/feature.test.js"))
}

// TEST-314: Promote with dryRun=true reports changes without modifying files
// traces: TASK-008
func TestPromote_DryRun(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`)

	originalContent := `package feature_test

import "testing"

// TEST-100: Feature test
// traces: TASK-001
func TestFeature(t *testing.T) {
}
`
	writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", originalContent)

	// Run with dryRun=true
	result, err := trace.Promote(dir, fs, true)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(HaveLen(1))
	g.Expect(result.Promotions[0].OldTrace).To(Equal("TASK-001"))
	g.Expect(result.Promotions[0].NewTrace).To(Equal("ARCH-001"))

	// Verify file was NOT modified
	content, err := fs.ReadFile(filepath.Join(dir, "internal/feature/feature_test.go"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal(originalContent))
}

// TEST-320: Promote handles simple number IDs (TASK-1, TASK-42, etc.)
// traces: TASK-001
func TestPromote_SimpleNumberIDs(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-1: First feature

**Traces to:** ARCH-1

### TASK-42: Another feature

**Traces to:** ARCH-42
`)

	writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", `package feature_test

import "testing"

// TEST-100: First test
// traces: TASK-1
func TestFeature(t *testing.T) {
}

// TEST-101: Second test
// traces: TASK-42
func TestAnotherFeature(t *testing.T) {
}
`)

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(HaveLen(2))
	g.Expect(result.Promotions[0].OldTrace).To(Equal("TASK-1"))
	g.Expect(result.Promotions[0].NewTrace).To(Equal("ARCH-1"))
	g.Expect(result.Promotions[1].OldTrace).To(Equal("TASK-42"))
	g.Expect(result.Promotions[1].NewTrace).To(Equal("ARCH-42"))

	// Verify file content
	content, err := fs.ReadFile(filepath.Join(dir, "internal/feature/feature_test.go"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("// traces: ARCH-1"))
	g.Expect(string(content)).To(ContainSubstring("// traces: ARCH-42"))
}

// TEST-321: Promote handles backward compatibility with 3-digit IDs
// traces: TASK-001
func TestPromote_BackwardCompatWithPaddedIDs(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Mix of old padded and new unpadded IDs
	writeArtifact(t, fs, dir, "tasks.md", `# Tasks

### TASK-001: Old style task

**Traces to:** ARCH-001

### TASK-5: New style task

**Traces to:** ARCH-5
`)

	writeTestFile(t, fs, dir, "internal/feature", "feature_test.go", `package feature_test

import "testing"

// TEST-100: Old style test
// traces: TASK-001
func TestOldStyle(t *testing.T) {
}

// TEST-101: New style test
// traces: TASK-5
func TestNewStyle(t *testing.T) {
}
`)

	result, err := trace.Promote(dir, fs, false)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Promotions).To(HaveLen(2))

	// Verify both formats work
	content, err := fs.ReadFile(filepath.Join(dir, "internal/feature/feature_test.go"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("// traces: ARCH-001"))
	g.Expect(string(content)).To(ContainSubstring("// traces: ARCH-5"))
}

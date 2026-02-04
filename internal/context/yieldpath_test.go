package context_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/context"
	"pgregory.net/rapid"
)

// TestGenerateYieldPath_SequentialPathPattern verifies the sequential path pattern.
// Traces to: TASK-2 AC "Path pattern for sequential"
func TestGenerateYieldPath_SequentialPathPattern(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a state.toml with a known project UUID for testing
	stateContent := `[project]
name = "test-project"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "abc123def456"
`
	err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(dir, ".claude", "state.toml"), []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Sequential execution: no taskID
	path, err := context.GenerateYieldPath(dir, "pm", "")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(path).ToNot(BeEmpty())

	// Path pattern: .claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{fileUUID}.toml
	// Expected pattern: .claude/context/YYYY-MM-DD-test-project-abc123def456/YYYY-MM-DD.HH-mm-SS-pm-UUID.toml
	pattern := regexp.MustCompile(`\.claude/context/\d{4}-\d{2}-\d{2}-test-project-[a-f0-9]+/\d{4}-\d{2}-\d{2}\.\d{2}-\d{2}-\d{2}-pm-[a-f0-9-]+\.toml$`)
	g.Expect(pattern.MatchString(path)).To(BeTrue(), "path %s should match sequential pattern", path)
}

// TestGenerateYieldPath_ParallelPathPattern verifies the parallel path pattern with taskID.
// Traces to: TASK-2 AC "Path pattern for parallel"
func TestGenerateYieldPath_ParallelPathPattern(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a state.toml with a known project UUID
	stateContent := `[project]
name = "test-project"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "abc123def456"
`
	err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(dir, ".claude", "state.toml"), []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Parallel execution: with taskID
	path, err := context.GenerateYieldPath(dir, "impl", "TASK-001")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(path).ToNot(BeEmpty())

	// Path pattern: .claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{taskID}-{fileUUID}.toml
	// Expected: .claude/context/YYYY-MM-DD-test-project-abc123def456/YYYY-MM-DD.HH-mm-SS-impl-TASK-001-UUID.toml
	pattern := regexp.MustCompile(`\.claude/context/\d{4}-\d{2}-\d{2}-test-project-[a-f0-9]+/\d{4}-\d{2}-\d{2}\.\d{2}-\d{2}-\d{2}-impl-TASK-001-[a-f0-9-]+\.toml$`)
	g.Expect(pattern.MatchString(path)).To(BeTrue(), "path %s should match parallel pattern", path)
}

// TestGenerateYieldPath_DateFormat verifies date is formatted as YYYY-MM-DD.
// Traces to: TASK-2 AC "Date format: YYYY-MM-DD"
func TestGenerateYieldPath_DateFormat(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create state file
	stateContent := `[project]
name = "date-test"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "abc123"
`
	err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(dir, ".claude", "state.toml"), []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	path, err := context.GenerateYieldPath(dir, "test", "")
	g.Expect(err).ToNot(HaveOccurred())

	// Extract date from path (first component after .claude/context/)
	parts := strings.Split(path, string(filepath.Separator))
	var dirName string
	for i, part := range parts {
		if part == "context" && i+1 < len(parts) {
			dirName = parts[i+1]
			break
		}
	}
	g.Expect(dirName).ToNot(BeEmpty())

	// Directory should start with date in YYYY-MM-DD format
	datePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)
	g.Expect(datePattern.MatchString(dirName)).To(BeTrue(), "directory %s should start with YYYY-MM-DD", dirName)

	// Extract and validate the date matches current date
	datePart := dirName[:10]
	today := time.Now().Format("2006-01-02")
	g.Expect(datePart).To(Equal(today), "date should be current date")
}

// TestGenerateYieldPath_DatetimeFormat verifies datetime is formatted as YYYY-MM-DD.HH-mm-SS.
// Traces to: TASK-2 AC "Datetime format: YYYY-MM-DD.HH-mm-SS"
func TestGenerateYieldPath_DatetimeFormat(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	stateContent := `[project]
name = "datetime-test"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "def456"
`
	err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(dir, ".claude", "state.toml"), []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	path, err := context.GenerateYieldPath(dir, "test", "")
	g.Expect(err).ToNot(HaveOccurred())

	// Extract filename
	filename := filepath.Base(path)
	// Filename format: YYYY-MM-DD.HH-mm-SS-{phase}-{uuid}.toml
	// Should start with datetime pattern
	datetimePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\.\d{2}-\d{2}-\d{2}-`)
	g.Expect(datetimePattern.MatchString(filename)).To(BeTrue(), "filename %s should start with YYYY-MM-DD.HH-mm-SS", filename)
}

// TestGenerateYieldPath_ProjectUUIDRetrievedFromState verifies project UUID is read from state.toml.
// Traces to: TASK-2 AC "Project UUID retrieved from or stored in state.toml"
func TestGenerateYieldPath_ProjectUUIDRetrievedFromState(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create state with known UUID
	knownUUID := "test-uuid-12345"
	stateContent := `[project]
name = "uuid-test"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "` + knownUUID + `"
`
	err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(dir, ".claude", "state.toml"), []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	path, err := context.GenerateYieldPath(dir, "test", "")
	g.Expect(err).ToNot(HaveOccurred())

	// Path should contain the project UUID from state.toml
	g.Expect(path).To(ContainSubstring(knownUUID), "path should contain project UUID from state.toml")
}

// TestGenerateYieldPath_ProjectUUIDStoredIfMissing verifies project UUID is generated and stored if missing.
// Traces to: TASK-2 AC "Project UUID retrieved from or stored in state.toml"
func TestGenerateYieldPath_ProjectUUIDStoredIfMissing(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create state WITHOUT uuid field
	stateContent := `[project]
name = "no-uuid-test"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
`
	err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	g.Expect(err).ToNot(HaveOccurred())
	statePath := filepath.Join(dir, ".claude", "state.toml")
	err = os.WriteFile(statePath, []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	path, err := context.GenerateYieldPath(dir, "test", "")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(path).ToNot(BeEmpty())

	// Read state.toml and verify UUID was added
	data, err := os.ReadFile(statePath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("uuid"), "state.toml should now contain uuid field")

	// Verify UUID is in path
	uuidPattern := regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)
	g.Expect(uuidPattern.MatchString(path)).To(BeTrue(), "path should contain a valid UUID")
}

// TestGenerateYieldPath_FileUUIDGenerated verifies each invocation generates a unique file UUID.
// Traces to: TASK-2 AC "File UUID generated per invocation using github.com/google/uuid"
func TestGenerateYieldPath_FileUUIDGenerated(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	stateContent := `[project]
name = "file-uuid-test"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "project-uuid"
`
	err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(dir, ".claude", "state.toml"), []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Generate two paths
	path1, err := context.GenerateYieldPath(dir, "test", "")
	g.Expect(err).ToNot(HaveOccurred())

	path2, err := context.GenerateYieldPath(dir, "test", "")
	g.Expect(err).ToNot(HaveOccurred())

	// Paths should be different due to different file UUIDs
	g.Expect(path1).ToNot(Equal(path2), "each invocation should generate a unique file UUID")

	// Both should contain valid UUID format
	uuidPattern := regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)
	g.Expect(uuidPattern.MatchString(path1)).To(BeTrue())
	g.Expect(uuidPattern.MatchString(path2)).To(BeTrue())
}

// TestGenerateYieldPath_ReturnsAbsolutePath verifies the function returns an absolute path.
// Traces to: TASK-2 AC "Returns absolute path (via filepath.Abs)"
func TestGenerateYieldPath_ReturnsAbsolutePath(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	stateContent := `[project]
name = "abs-path-test"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "abs-uuid"
`
	err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(dir, ".claude", "state.toml"), []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	path, err := context.GenerateYieldPath(dir, "test", "")
	g.Expect(err).ToNot(HaveOccurred())

	// Path should be absolute
	g.Expect(filepath.IsAbs(path)).To(BeTrue(), "path %s should be absolute", path)
}

// TestGenerateYieldPath_CreatesParentDirectories verifies parent directories are created with mode 0755.
// Traces to: TASK-2 AC "Creates parent directories with mode 0755 (via os.MkdirAll)"
func TestGenerateYieldPath_CreatesParentDirectories(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	stateContent := `[project]
name = "mkdir-test"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "mkdir-uuid"
`
	err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(dir, ".claude", "state.toml"), []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	path, err := context.GenerateYieldPath(dir, "test", "")
	g.Expect(err).ToNot(HaveOccurred())

	// Verify parent directory was created
	parentDir := filepath.Dir(path)
	info, err := os.Stat(parentDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(info.IsDir()).To(BeTrue())

	// Verify permissions (0755)
	g.Expect(info.Mode().Perm()).To(Equal(os.FileMode(0o755)))
}

// TestGenerateYieldPath_ErrorOnPermissionFailure verifies error is returned with context on permission failure.
// Traces to: TASK-2 AC "Returns error with context on permission failures"
func TestGenerateYieldPath_ErrorOnPermissionFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Cannot test permission failures as root")
	}

	g := NewWithT(t)
	dir := t.TempDir()

	stateContent := `[project]
name = "perm-test"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "perm-uuid"
`
	err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(dir, ".claude", "state.toml"), []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Make project directory read-only to prevent subdirectory creation
	err = os.Chmod(dir, 0o444)
	g.Expect(err).ToNot(HaveOccurred())
	defer os.Chmod(dir, 0o755) // Restore for cleanup

	_, err = context.GenerateYieldPath(dir, "test", "")
	g.Expect(err).To(HaveOccurred(), "should return error on permission failure")
	g.Expect(err.Error()).To(ContainSubstring("permission"), "error should mention permission issue")
}

// TestGenerateYieldPath_MultipleInvocationsReturnUniquePaths verifies multiple invocations return unique paths.
// Traces to: TASK-2 AC "Unit tests verify multiple invocations return unique paths"
func TestGenerateYieldPath_MultipleInvocationsReturnUniquePaths(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	stateContent := `[project]
name = "unique-test"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "unique-uuid"
`
	err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(dir, ".claude", "state.toml"), []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Generate multiple paths
	const numPaths = 10
	paths := make(map[string]bool, numPaths)

	for i := 0; i < numPaths; i++ {
		path, err := context.GenerateYieldPath(dir, "test", "")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(paths[path]).To(BeFalse(), "path %s should be unique", path)
		paths[path] = true
	}

	g.Expect(paths).To(HaveLen(numPaths), "all paths should be unique")
}

// TestGenerateYieldPath_PropertyUniquenessAcrossParallelInvocations verifies uniqueness with property-based testing.
// Traces to: TASK-2 AC "Property tests verify uniqueness across parallel invocations"
func TestGenerateYieldPath_PropertyUniquenessAcrossParallelInvocations(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Setup state
		projectName := rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "projectName")
		phase := rapid.StringMatching(`[a-z]{2,10}`).Draw(rt, "phase")
		taskID := rapid.StringMatching(`TASK-[0-9]{3}`).Draw(rt, "taskID")

		stateContent := `[project]
name = "` + projectName + `"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "prop-uuid"
`
		err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
		g.Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(filepath.Join(dir, ".claude", "state.toml"), []byte(stateContent), 0o644)
		g.Expect(err).ToNot(HaveOccurred())

		// Generate paths concurrently
		const numGoroutines = 5
		var wg sync.WaitGroup
		pathsChan := make(chan string, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				path, err := context.GenerateYieldPath(dir, phase, taskID)
				if err == nil {
					pathsChan <- path
				}
			}()
		}

		wg.Wait()
		close(pathsChan)

		// Collect and verify uniqueness
		paths := make(map[string]bool)
		for path := range pathsChan {
			g.Expect(paths[path]).To(BeFalse(), "path %s should be unique across parallel invocations", path)
			paths[path] = true
		}

		// At least some paths should have been generated
		g.Expect(len(paths)).To(BeNumerically(">", 0))
	})
}

// TestGenerateYieldPath_PathFormatMatchesExpectedPattern verifies the complete path format.
// Traces to: TASK-2 AC "Unit tests verify path format matches pattern"
func TestGenerateYieldPath_PathFormatMatchesExpectedPattern(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	stateContent := `[project]
name = "pattern-test"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "pattern-uuid"
`
	err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(dir, ".claude", "state.toml"), []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	t.Run("sequential pattern", func(t *testing.T) {
		g := NewWithT(t)
		path, err := context.GenerateYieldPath(dir, "architect", "")
		g.Expect(err).ToNot(HaveOccurred())

		// Pattern: .claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{fileUUID}.toml
		pattern := regexp.MustCompile(`\.claude/context/\d{4}-\d{2}-\d{2}-pattern-test-[a-z0-9-]+/\d{4}-\d{2}-\d{2}\.\d{2}-\d{2}-\d{2}-architect-[a-f0-9-]+\.toml$`)
		g.Expect(pattern.MatchString(path)).To(BeTrue(), "path should match sequential pattern")
	})

	t.Run("parallel pattern with task ID", func(t *testing.T) {
		g := NewWithT(t)
		path, err := context.GenerateYieldPath(dir, "impl", "TASK-042")
		g.Expect(err).ToNot(HaveOccurred())

		// Pattern: .claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{taskID}-{fileUUID}.toml
		pattern := regexp.MustCompile(`\.claude/context/\d{4}-\d{2}-\d{2}-pattern-test-[a-z0-9-]+/\d{4}-\d{2}-\d{2}\.\d{2}-\d{2}-\d{2}-impl-TASK-042-[a-f0-9-]+\.toml$`)
		g.Expect(pattern.MatchString(path)).To(BeTrue(), "path should match parallel pattern with task ID")
	})
}

// TestGenerateYieldPath_ErrorOnMissingProjectDirectory verifies error when project directory doesn't exist.
// Traces to: TASK-2 AC "Returns error with context on permission failures"
func TestGenerateYieldPath_ErrorOnMissingProjectDirectory(t *testing.T) {
	g := NewWithT(t)
	nonExistentDir := filepath.Join(t.TempDir(), "does-not-exist")

	_, err := context.GenerateYieldPath(nonExistentDir, "test", "")
	g.Expect(err).To(HaveOccurred(), "should return error when project directory doesn't exist")
}

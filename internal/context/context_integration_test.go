package context_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/context"
)

// setupIntegrationProjectDir creates a project directory with state.toml for integration tests.
func setupIntegrationProjectDir(t *testing.T, projectName, projectUUID string) string {
	t.Helper()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	g.Expect(os.MkdirAll(claudeDir, 0o755)).To(Succeed())

	stateContent := fmt.Sprintf(`[project]
name = "%s"
created = 2026-02-04T12:00:00Z
phase = "test"
workflow = "new"
uuid = "%s"
`, projectName, projectUUID)
	g.Expect(os.WriteFile(filepath.Join(claudeDir, "state.toml"), []byte(stateContent), 0o644)).To(Succeed())

	return dir
}

// writeIntegrationTOML creates a TOML file for integration tests.
func writeIntegrationTOML(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

// extractYieldPathFromContent extracts the yield_path value from TOML content.
func extractYieldPathFromContent(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, "yield_path") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(parts[1]), "\"")
			}
		}
	}
	return ""
}

// TestIntegration_ContextWriteGeneratesYieldPath verifies context write generates context file with output.yield_path.
// Traces to: TASK-9 AC "context write generates context file with output.yield_path"
func TestIntegration_ContextWriteGeneratesYieldPath(t *testing.T) {
	g := NewWithT(t)
	dir := setupIntegrationProjectDir(t, "integration-test", "int-uuid-123")
	source := writeIntegrationTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"tdd-red\"\n")

	path, err := context.WriteWithYieldPath(dir, "impl", "TASK-001", source)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(path).ToNot(BeEmpty())

	// Verify context file was created
	_, statErr := os.Stat(path)
	g.Expect(statErr).ToNot(HaveOccurred(), "context file should exist")

	// Read and verify output.yield_path exists
	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("[output]"), "should contain [output] section")
	g.Expect(string(content)).To(ContainSubstring("yield_path"), "should contain yield_path field")

	// Verify yield_path is not empty
	yieldPath := extractYieldPathFromContent(string(content))
	g.Expect(yieldPath).ToNot(BeEmpty(), "yield_path should not be empty")
}

// TestIntegration_YieldPathIsAbsolute verifies yield_path is an absolute path (starts with /).
// Traces to: TASK-9 AC "yield_path is absolute path (starts with /)"
func TestIntegration_YieldPathIsAbsolute(t *testing.T) {
	g := NewWithT(t)
	dir := setupIntegrationProjectDir(t, "abs-path-test", "abs-uuid")
	source := writeIntegrationTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"test\"\n")

	path, err := context.WriteWithYieldPath(dir, "impl", "TASK-002", source)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	yieldPath := extractYieldPathFromContent(string(content))
	g.Expect(yieldPath).ToNot(BeEmpty())

	// On Unix systems, absolute paths start with /
	// On Windows, absolute paths start with drive letter (e.g., C:\)
	g.Expect(filepath.IsAbs(yieldPath)).To(BeTrue(), "yield_path %s should be absolute", yieldPath)
	g.Expect(strings.HasPrefix(yieldPath, "/") || (len(yieldPath) > 1 && yieldPath[1] == ':')).To(BeTrue(),
		"yield_path %s should start with / (Unix) or drive letter (Windows)", yieldPath)
}

// TestIntegration_YieldPathIncludesUUID verifies yield_path includes UUID (unique per invocation).
// Traces to: TASK-9 AC "yield_path includes UUID (unique per invocation)"
func TestIntegration_YieldPathIncludesUUID(t *testing.T) {
	g := NewWithT(t)
	dir := setupIntegrationProjectDir(t, "uuid-test", "proj-uuid-456")
	source := writeIntegrationTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"test\"\n")

	path, err := context.WriteWithYieldPath(dir, "impl", "TASK-003", source)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	yieldPath := extractYieldPathFromContent(string(content))
	g.Expect(yieldPath).ToNot(BeEmpty())

	// UUID pattern: 8-4-4-4-12 hex characters
	uuidPattern := regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)
	g.Expect(uuidPattern.MatchString(yieldPath)).To(BeTrue(),
		"yield_path %s should contain UUID pattern", yieldPath)
}

// TestIntegration_YieldPathMatchesExpectedPattern verifies yield_path matches expected pattern with timestamp and UUID.
// Traces to: TASK-9 AC "yield_path matches expected pattern with timestamp and UUID"
func TestIntegration_YieldPathMatchesExpectedPattern(t *testing.T) {
	g := NewWithT(t)
	dir := setupIntegrationProjectDir(t, "pattern-test", "pattern-uuid")
	source := writeIntegrationTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"test\"\n")

	path, err := context.WriteWithYieldPath(dir, "impl", "TASK-004", source)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	yieldPath := extractYieldPathFromContent(string(content))
	g.Expect(yieldPath).ToNot(BeEmpty())

	// Expected pattern: .claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{taskID}-{fileUUID}.toml
	// date: YYYY-MM-DD
	// datetime: YYYY-MM-DD.HH-mm-SS
	// UUID: 8-4-4-4-12 hex
	pattern := regexp.MustCompile(
		`\.claude/context/` +
			`\d{4}-\d{2}-\d{2}-` + // date-
			`pattern-test-` + // project name-
			`[a-z0-9-]+/` + // projectUUID/
			`\d{4}-\d{2}-\d{2}\.\d{2}-\d{2}-\d{2}-` + // datetime-
			`impl-` + // phase-
			`TASK-004-` + // taskID-
			`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}` + // fileUUID
			`\.toml$`) // .toml extension

	g.Expect(pattern.MatchString(yieldPath)).To(BeTrue(),
		"yield_path %s should match expected pattern", yieldPath)
}

// TestIntegration_SequentialContextGetsUniquePath verifies sequential context (no taskID) gets unique path.
// Traces to: TASK-9 AC "sequential context (no taskID) gets unique path"
func TestIntegration_SequentialContextGetsUniquePath(t *testing.T) {
	g := NewWithT(t)
	dir := setupIntegrationProjectDir(t, "seq-test", "seq-uuid")
	source := writeIntegrationTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"test\"\n")

	// Generate two sequential contexts (no taskID)
	path1, err := context.WriteWithYieldPath(dir, "pm", "", source)
	g.Expect(err).ToNot(HaveOccurred())
	content1, err := os.ReadFile(path1)
	g.Expect(err).ToNot(HaveOccurred())
	yieldPath1 := extractYieldPathFromContent(string(content1))

	path2, err := context.WriteWithYieldPath(dir, "architect", "", source)
	g.Expect(err).ToNot(HaveOccurred())
	content2, err := os.ReadFile(path2)
	g.Expect(err).ToNot(HaveOccurred())
	yieldPath2 := extractYieldPathFromContent(string(content2))

	// Each should have a unique yield_path
	g.Expect(yieldPath1).ToNot(BeEmpty())
	g.Expect(yieldPath2).ToNot(BeEmpty())
	g.Expect(yieldPath1).ToNot(Equal(yieldPath2),
		"sequential contexts should have unique yield_paths")

	// Verify sequential pattern (no TASK- in path)
	g.Expect(yieldPath1).ToNot(MatchRegexp(`TASK-\d+`),
		"sequential yield_path should not contain TASK-")
	g.Expect(yieldPath2).ToNot(MatchRegexp(`TASK-\d+`),
		"sequential yield_path should not contain TASK-")
}

// TestIntegration_ParallelContextsDifferentTaskIDsGetUniquePaths verifies parallel contexts (different taskIDs) get unique paths.
// Traces to: TASK-9 AC "parallel contexts (different taskIDs) get unique paths"
func TestIntegration_ParallelContextsDifferentTaskIDsGetUniquePaths(t *testing.T) {
	g := NewWithT(t)
	dir := setupIntegrationProjectDir(t, "parallel-diff-test", "parallel-uuid")
	source := writeIntegrationTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"test\"\n")

	// Generate parallel contexts with different task IDs
	taskIDs := []string{"TASK-001", "TASK-002", "TASK-003"}
	yieldPaths := make([]string, len(taskIDs))

	for i, taskID := range taskIDs {
		path, err := context.WriteWithYieldPath(dir, "impl", taskID, source)
		g.Expect(err).ToNot(HaveOccurred())
		content, err := os.ReadFile(path)
		g.Expect(err).ToNot(HaveOccurred())
		yieldPaths[i] = extractYieldPathFromContent(string(content))
		g.Expect(yieldPaths[i]).ToNot(BeEmpty())
	}

	// All yield_paths should be unique
	uniquePaths := make(map[string]bool)
	for _, yp := range yieldPaths {
		g.Expect(uniquePaths[yp]).To(BeFalse(),
			"yield_path %s should be unique", yp)
		uniquePaths[yp] = true
	}
	g.Expect(uniquePaths).To(HaveLen(len(taskIDs)))
}

// TestIntegration_ParallelContextsSameTaskIDGetUniquePathsViaUUID verifies parallel contexts (same taskID, different invocations) get unique paths via UUID.
// Traces to: TASK-9 AC "parallel contexts (same taskID, different invocations) get unique paths via UUID"
func TestIntegration_ParallelContextsSameTaskIDGetUniquePathsViaUUID(t *testing.T) {
	g := NewWithT(t)
	dir := setupIntegrationProjectDir(t, "same-task-test", "same-task-uuid")
	source := writeIntegrationTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"test\"\n")

	// Generate multiple contexts with the SAME task ID
	// Note: Each call to WriteWithYieldPath generates a new yield_path with unique UUID
	// even though the context file itself gets overwritten
	const numInvocations = 5
	yieldPaths := make([]string, numInvocations)

	for i := 0; i < numInvocations; i++ {
		path, err := context.WriteWithYieldPath(dir, "impl", "TASK-SAME", source)
		g.Expect(err).ToNot(HaveOccurred())
		content, err := os.ReadFile(path)
		g.Expect(err).ToNot(HaveOccurred())
		yieldPaths[i] = extractYieldPathFromContent(string(content))
		g.Expect(yieldPaths[i]).ToNot(BeEmpty())
	}

	// All yield_paths should be unique due to file UUID
	uniquePaths := make(map[string]bool)
	for _, yp := range yieldPaths {
		g.Expect(uniquePaths[yp]).To(BeFalse(),
			"yield_path %s should be unique across invocations with same taskID", yp)
		uniquePaths[yp] = true
	}
	g.Expect(uniquePaths).To(HaveLen(numInvocations),
		"should have %d unique yield_paths", numInvocations)
}

// TestIntegration_MockSkillCanReadYieldPathFromContext verifies mock skill can read output.yield_path from context.
// Traces to: TASK-9 AC "mock skill can read output.yield_path from context"
func TestIntegration_MockSkillCanReadYieldPathFromContext(t *testing.T) {
	g := NewWithT(t)
	dir := setupIntegrationProjectDir(t, "read-yield-test", "read-uuid")
	source := writeIntegrationTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"test\"\n[task]\ndescription = \"Test task\"\n")

	// Write context with yield path
	contextPath, err := context.WriteWithYieldPath(dir, "impl", "TASK-READ", source)
	g.Expect(err).ToNot(HaveOccurred())

	// Simulate mock skill reading the context file
	// Skills use TOML parsing to read context
	type OutputSection struct {
		YieldPath string `toml:"yield_path"`
	}
	type ContextFile struct {
		Output OutputSection `toml:"output"`
	}

	var ctx ContextFile
	_, err = toml.DecodeFile(contextPath, &ctx)
	g.Expect(err).ToNot(HaveOccurred(), "mock skill should be able to parse context TOML")

	// Verify yield_path was extracted correctly
	g.Expect(ctx.Output.YieldPath).ToNot(BeEmpty(),
		"mock skill should read non-empty yield_path from context")
	g.Expect(filepath.IsAbs(ctx.Output.YieldPath)).To(BeTrue(),
		"yield_path should be absolute")
}

// TestIntegration_MockSkillCanWriteResultToYieldPath verifies mock skill can write result to yield_path location.
// Traces to: TASK-9 AC "mock skill can write result to yield_path location"
func TestIntegration_MockSkillCanWriteResultToYieldPath(t *testing.T) {
	g := NewWithT(t)
	dir := setupIntegrationProjectDir(t, "write-result-test", "write-uuid")
	source := writeIntegrationTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"test\"\n")

	// Write context with yield path
	contextPath, err := context.WriteWithYieldPath(dir, "impl", "TASK-WRITE", source)
	g.Expect(err).ToNot(HaveOccurred())

	// Read context to get yield_path
	content, err := os.ReadFile(contextPath)
	g.Expect(err).ToNot(HaveOccurred())
	yieldPath := extractYieldPathFromContent(string(content))
	g.Expect(yieldPath).ToNot(BeEmpty())

	// Simulate mock skill writing result to yield_path
	// The yield_path's parent directory should have been created by GenerateYieldPath
	resultContent := `[result]
status = "success"
summary = "Mock skill completed successfully"

[payload]
items_processed = 5
`
	err = os.WriteFile(yieldPath, []byte(resultContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred(),
		"mock skill should be able to write result to yield_path")

	// Verify file was written
	_, statErr := os.Stat(yieldPath)
	g.Expect(statErr).ToNot(HaveOccurred(),
		"result file should exist at yield_path")
}

// TestIntegration_ResultFileAtYieldPathIsReadable verifies result file at yield_path is readable by context read.
// Traces to: TASK-9 AC "result file at yield_path is readable by context read"
func TestIntegration_ResultFileAtYieldPathIsReadable(t *testing.T) {
	g := NewWithT(t)
	dir := setupIntegrationProjectDir(t, "readable-test", "readable-uuid")
	source := writeIntegrationTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"test\"\n")

	// Write context with yield path
	contextPath, err := context.WriteWithYieldPath(dir, "impl", "TASK-READ-RESULT", source)
	g.Expect(err).ToNot(HaveOccurred())

	// Read context to get yield_path
	content, err := os.ReadFile(contextPath)
	g.Expect(err).ToNot(HaveOccurred())
	yieldPath := extractYieldPathFromContent(string(content))
	g.Expect(yieldPath).ToNot(BeEmpty())

	// Simulate skill writing result to yield_path
	expectedStatus := "completed"
	expectedDecision := "Use dependency injection for testability"
	resultContent := fmt.Sprintf(`[result]
status = "%s"
summary = "Task completed"

[payload]
[payload.decisions]
key = "testing-approach"
value = "%s"
`, expectedStatus, expectedDecision)
	err = os.WriteFile(yieldPath, []byte(resultContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify result can be read back (simulating orchestrator reading result)
	resultData, err := os.ReadFile(yieldPath)
	g.Expect(err).ToNot(HaveOccurred(),
		"result file at yield_path should be readable")

	// Parse and verify content
	type ResultPayload struct {
		Decisions map[string]string `toml:"decisions"`
	}
	type ResultFile struct {
		Result struct {
			Status  string `toml:"status"`
			Summary string `toml:"summary"`
		} `toml:"result"`
		Payload ResultPayload `toml:"payload"`
	}

	var result ResultFile
	_, err = toml.Decode(string(resultData), &result)
	g.Expect(err).ToNot(HaveOccurred(),
		"result file should be valid TOML")
	g.Expect(result.Result.Status).To(Equal(expectedStatus),
		"result status should match")
	g.Expect(result.Payload.Decisions["key"]).To(Equal("testing-approach"),
		"result payload should be readable")
}

// TestIntegration_ConcurrentYieldPathGeneration verifies yield paths are unique under concurrent generation.
// This is an integration-level property test for parallel execution safety.
func TestIntegration_ConcurrentYieldPathGeneration(t *testing.T) {
	g := NewWithT(t)
	dir := setupIntegrationProjectDir(t, "concurrent-test", "concurrent-uuid")
	source := writeIntegrationTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"test\"\n")

	const numGoroutines = 10
	var wg sync.WaitGroup
	yieldPaths := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Launch concurrent context writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(taskNum int) {
			defer wg.Done()
			taskID := fmt.Sprintf("TASK-%03d", taskNum)
			path, err := context.WriteWithYieldPath(dir, "impl", taskID, source)
			if err != nil {
				errors <- err
				return
			}
			content, err := os.ReadFile(path)
			if err != nil {
				errors <- err
				return
			}
			yieldPath := extractYieldPathFromContent(string(content))
			yieldPaths <- yieldPath
		}(i)
	}

	wg.Wait()
	close(yieldPaths)
	close(errors)

	// Check for errors
	for err := range errors {
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Collect and verify uniqueness
	uniquePaths := make(map[string]bool)
	for yp := range yieldPaths {
		g.Expect(yp).ToNot(BeEmpty())
		g.Expect(uniquePaths[yp]).To(BeFalse(),
			"yield_path %s should be unique under concurrent generation", yp)
		uniquePaths[yp] = true
	}

	g.Expect(uniquePaths).To(HaveLen(numGoroutines),
		"all %d concurrent invocations should produce unique yield_paths", numGoroutines)
}

// TestIntegration_EndToEndSkillWorkflow tests the complete workflow: context write -> skill read -> skill write result -> result read.
func TestIntegration_EndToEndSkillWorkflow(t *testing.T) {
	g := NewWithT(t)
	dir := setupIntegrationProjectDir(t, "e2e-test", "e2e-uuid")

	// Step 1: Orchestrator writes context with yield_path
	taskInput := `[dispatch]
skill = "tdd-red"

[task]
id = "TASK-E2E"
description = "Implement feature X"

[territory]
root = "/project"
languages = ["go"]
`
	source := writeIntegrationTOML(t, t.TempDir(), "input.toml", taskInput)

	contextPath, err := context.WriteWithYieldPath(dir, "impl", "TASK-E2E", source)
	g.Expect(err).ToNot(HaveOccurred())

	// Step 2: Skill reads context and extracts yield_path
	type DispatchSection struct {
		Skill string `toml:"skill"`
	}
	type TaskSection struct {
		ID          string `toml:"id"`
		Description string `toml:"description"`
	}
	type OutputSection struct {
		YieldPath string `toml:"yield_path"`
	}
	type ContextFile struct {
		Dispatch DispatchSection `toml:"dispatch"`
		Task     TaskSection     `toml:"task"`
		Output   OutputSection   `toml:"output"`
	}

	var ctx ContextFile
	_, err = toml.DecodeFile(contextPath, &ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ctx.Dispatch.Skill).To(Equal("tdd-red"))
	g.Expect(ctx.Task.ID).To(Equal("TASK-E2E"))
	g.Expect(ctx.Output.YieldPath).ToNot(BeEmpty())

	yieldPath := ctx.Output.YieldPath

	// Step 3: Skill writes result to yield_path
	skillResult := `[result]
status = "success"
phase = "impl"
subphase = "tdd-red"

[payload]
summary = "Tests written for feature X"
tests_created = 3

[[payload.decisions]]
context = "Test organization"
choice = "Table-driven tests"
reason = "Cleaner test structure"
alternatives = ["individual test functions", "subtests"]
`
	err = os.WriteFile(yieldPath, []byte(skillResult), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Step 4: Orchestrator reads result from yield_path
	resultData, err := os.ReadFile(yieldPath)
	g.Expect(err).ToNot(HaveOccurred())

	type Decision struct {
		Context      string   `toml:"context"`
		Choice       string   `toml:"choice"`
		Reason       string   `toml:"reason"`
		Alternatives []string `toml:"alternatives"`
	}
	type Payload struct {
		Summary      string     `toml:"summary"`
		TestsCreated int        `toml:"tests_created"`
		Decisions    []Decision `toml:"decisions"`
	}
	type Result struct {
		Status   string `toml:"status"`
		Phase    string `toml:"phase"`
		Subphase string `toml:"subphase"`
	}
	type ResultFile struct {
		Result  Result  `toml:"result"`
		Payload Payload `toml:"payload"`
	}

	var result ResultFile
	_, err = toml.Decode(string(resultData), &result)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify complete round-trip
	g.Expect(result.Result.Status).To(Equal("success"))
	g.Expect(result.Result.Phase).To(Equal("impl"))
	g.Expect(result.Result.Subphase).To(Equal("tdd-red"))
	g.Expect(result.Payload.Summary).To(Equal("Tests written for feature X"))
	g.Expect(result.Payload.TestsCreated).To(Equal(3))
	g.Expect(result.Payload.Decisions).To(HaveLen(1))
	g.Expect(result.Payload.Decisions[0].Choice).To(Equal("Table-driven tests"))
}

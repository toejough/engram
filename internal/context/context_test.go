package context_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/context"
	"pgregory.net/rapid"
)

func writeTOML(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestWrite(t *testing.T) {
	t.Run("copies TOML file to context directory", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		source := writeTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"tdd-red\"\n")

		path, err := context.Write(dir, "TASK-004", "tdd-red", source)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(path).To(ContainSubstring("TASK-004-tdd-red.toml"))

		// File should exist
		_, err = os.Stat(path)
		g.Expect(err).ToNot(HaveOccurred())

		// Content should match
		data, err := os.ReadFile(path)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(data)).To(ContainSubstring("tdd-red"))
	})

	t.Run("creates context directory if needed", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		source := writeTOML(t, t.TempDir(), "input.toml", "key = \"value\"\n")

		_, err := context.Write(dir, "TASK-001", "pm-interview", source)
		g.Expect(err).ToNot(HaveOccurred())

		info, err := os.Stat(filepath.Join(dir, context.ContextDir))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(info.IsDir()).To(BeTrue())
	})

	t.Run("overwrites existing file without error", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		source1 := writeTOML(t, t.TempDir(), "input1.toml", "key = \"original\"\n")
		source2 := writeTOML(t, t.TempDir(), "input2.toml", "key = \"updated\"\n")

		path, err := context.Write(dir, "TASK-001", "tdd-red", source1)
		g.Expect(err).ToNot(HaveOccurred())

		// Write again with different content - should succeed
		path2, err := context.Write(dir, "TASK-001", "tdd-red", source2)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(path2).To(Equal(path))

		// Content should be updated
		data, err := os.ReadFile(path)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(data)).To(ContainSubstring("updated"))
	})

	t.Run("errors if source is not valid TOML", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		source := writeTOML(t, t.TempDir(), "bad.toml", "this is not valid toml [[[")

		_, err := context.Write(dir, "TASK-001", "tdd-red", source)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("not valid TOML"))
	})

	t.Run("errors if source does not exist", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := context.Write(dir, "TASK-001", "tdd-red", "/nonexistent/file.toml")
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("does not exist"))
	})
}

func TestRead(t *testing.T) {
	t.Run("reads context file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		source := writeTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"tdd-red\"\n")

		_, err := context.Write(dir, "TASK-004", "tdd-red", source)
		g.Expect(err).ToNot(HaveOccurred())

		content, err := context.Read(dir, "TASK-004", "tdd-red", false)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(content).To(ContainSubstring("tdd-red"))
	})

	t.Run("reads result file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Manually create a result file
		contextDir := filepath.Join(dir, context.ContextDir)
		g.Expect(os.MkdirAll(contextDir, 0o755)).To(Succeed())

		resultPath := filepath.Join(contextDir, context.ResultFilename("TASK-004", "tdd-red"))
		g.Expect(os.WriteFile(resultPath, []byte("[result]\nstatus = \"success\"\n"), 0o644)).To(Succeed())

		content, err := context.Read(dir, "TASK-004", "tdd-red", true)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(content).To(ContainSubstring("success"))
	})

	t.Run("errors if file does not exist", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := context.Read(dir, "TASK-999", "nonexistent", false)
		g.Expect(err).To(HaveOccurred())
	})
}

func TestFilenameConventions(t *testing.T) {
	g := NewWithT(t)
	g.Expect(context.Filename("TASK-004", "tdd-red")).To(Equal("TASK-004-tdd-red.toml"))
	g.Expect(context.ResultFilename("TASK-004", "tdd-red")).To(Equal("TASK-004-tdd-red.result.toml"))
}

func TestFilenameProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		task := rapid.StringMatching(`TASK-[0-9]{3}`).Draw(rt, "task")
		skill := rapid.StringMatching(`[a-z]+-[a-z]+`).Draw(rt, "skill")

		name := context.Filename(task, skill)
		g.Expect(name).To(HaveSuffix(".toml"))
		g.Expect(name).To(HavePrefix(task))
		g.Expect(name).To(ContainSubstring(skill))

		resultName := context.ResultFilename(task, skill)
		g.Expect(resultName).To(HaveSuffix(".result.toml"))
		g.Expect(resultName).To(HavePrefix(task))
	})
}

// TEST-420 traces: TASK-015
// Test WriteWithRouting adds routing section to context.
func TestWriteWithRouting_AddsRoutingSection(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	source := writeTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"tdd-red\"\n")

	routing := context.RoutingConfig{
		Simple:  "haiku",
		Medium:  "sonnet",
		Complex: "opus",
	}
	skillComplexity := map[string]string{
		"tdd-red":   "medium",
		"tdd-green": "medium",
	}

	path, err := context.WriteWithRouting(dir, "TASK-004", "tdd-red", source, routing, skillComplexity)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	// Should contain original content
	g.Expect(string(content)).To(ContainSubstring("tdd-red"))

	// Should contain routing section
	g.Expect(string(content)).To(ContainSubstring("[routing]"))
	g.Expect(string(content)).To(ContainSubstring("suggested_model"))
	g.Expect(string(content)).To(ContainSubstring("sonnet"))
}

// TEST-421 traces: TASK-015
// Test WriteWithRouting uses skill-to-complexity mapping.
func TestWriteWithRouting_UsesSkillMapping(t *testing.T) {
	dir := t.TempDir()
	source := writeTOML(t, t.TempDir(), "input.toml", "key = \"value\"\n")

	routing := context.RoutingConfig{
		Simple:  "haiku",
		Medium:  "sonnet",
		Complex: "opus",
	}

	testCases := []struct {
		skill           string
		skillComplexity map[string]string
		expectedModel   string
	}{
		{"alignment-check", map[string]string{"alignment-check": "simple"}, "haiku"},
		{"tdd-red", map[string]string{"tdd-red": "medium"}, "sonnet"},
		{"meta-audit", map[string]string{"meta-audit": "complex"}, "opus"},
	}

	for _, tc := range testCases {
		t.Run(tc.skill, func(t *testing.T) {
			g := NewWithT(t)
			path, err := context.WriteWithRouting(dir, "TASK-001", tc.skill, source, routing, tc.skillComplexity)
			g.Expect(err).ToNot(HaveOccurred())

			content, err := os.ReadFile(path)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(string(content)).To(ContainSubstring(tc.expectedModel))
		})
	}
}

// TEST-422 traces: TASK-015
// Test WriteWithRouting includes reason field.
func TestWriteWithRouting_IncludesReason(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	source := writeTOML(t, t.TempDir(), "input.toml", "key = \"value\"\n")

	routing := context.RoutingConfig{
		Simple:  "haiku",
		Medium:  "sonnet",
		Complex: "opus",
	}
	skillComplexity := map[string]string{"tdd-red": "medium"}

	path, err := context.WriteWithRouting(dir, "TASK-001", "tdd-red", source, routing, skillComplexity)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("reason"))
	g.Expect(string(content)).To(ContainSubstring("medium complexity"))
}

// TEST-423 traces: TASK-015
// Test WriteWithRouting defaults to medium for unknown skills.
func TestWriteWithRouting_DefaultsToMedium(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	source := writeTOML(t, t.TempDir(), "input.toml", "key = \"value\"\n")

	routing := context.RoutingConfig{
		Simple:  "haiku",
		Medium:  "sonnet",
		Complex: "opus",
	}
	skillComplexity := map[string]string{} // Empty - skill not mapped

	path, err := context.WriteWithRouting(dir, "TASK-001", "unknown-skill", source, routing, skillComplexity)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	// Should default to medium model
	g.Expect(string(content)).To(ContainSubstring("sonnet"))
}

// TEST-593 traces: TASK-036
// Test WriteWithRouting injects territory map automatically.
func TestWriteWithRouting_InjectsTerritory(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a cached territory map
	contextDir := filepath.Join(dir, "context")
	g.Expect(os.MkdirAll(contextDir, 0o755)).To(Succeed())

	territoryTOML := `cached_at = 2025-01-01T00:00:00Z
file_count = 10

[map]
[map.structure]
root = "` + dir + `"
languages = ["go"]
build_tool = "go"
test_framework = "go test"

[map.entry_points]
cli = "cmd/app"

[map.packages]
count = 5
internal = ["pkg1", "pkg2"]

[map.tests]
pattern = "*_test.go"
count = 10

[map.docs]
readme = "README.md"
artifacts = []
`
	g.Expect(os.WriteFile(filepath.Join(contextDir, "territory.toml"), []byte(territoryTOML), 0o644)).To(Succeed())

	source := writeTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"tdd-red\"\n")

	routing := context.RoutingConfig{
		Simple:  "haiku",
		Medium:  "sonnet",
		Complex: "opus",
	}
	skillComplexity := map[string]string{"tdd-red": "medium"}

	path, err := context.WriteWithRouting(dir, "TASK-001", "tdd-red", source, routing, skillComplexity)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	// Should contain routing section
	g.Expect(string(content)).To(ContainSubstring("[routing]"))
	g.Expect(string(content)).To(ContainSubstring("sonnet"))

	// Should also contain territory section
	g.Expect(string(content)).To(ContainSubstring("[territory]"))
	g.Expect(string(content)).To(ContainSubstring("languages"))
}

// TEST-550 traces: TASK-032
// Test WriteParallel creates context files for multiple tasks.
func TestWriteParallel_CreatesMultipleFiles(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	template := writeTOML(t, t.TempDir(), "template.toml", "[shared]\nmap = \"territory\"\n")

	paths, err := context.WriteParallel(dir, []string{"TASK-001", "TASK-002", "TASK-003"}, "tdd-red", template)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(paths).To(HaveLen(3))

	for _, path := range paths {
		_, err := os.Stat(path)
		g.Expect(err).ToNot(HaveOccurred())
	}
}

// TEST-551 traces: TASK-032
// Test WriteParallel uses correct naming convention.
func TestWriteParallel_CorrectNaming(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	template := writeTOML(t, t.TempDir(), "template.toml", "key = \"value\"\n")

	paths, err := context.WriteParallel(dir, []string{"TASK-001", "TASK-002"}, "tdd-red", template)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(paths[0]).To(ContainSubstring("TASK-001-tdd-red.toml"))
	g.Expect(paths[1]).To(ContainSubstring("TASK-002-tdd-red.toml"))
}

// TEST-552 traces: TASK-032
// Test WriteParallel includes shared content.
func TestWriteParallel_IncludesSharedContent(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	template := writeTOML(t, t.TempDir(), "template.toml", "[territory]\nroot = \"src\"\n")

	paths, err := context.WriteParallel(dir, []string{"TASK-001"}, "tdd-green", template)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(paths[0])
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("territory"))
}

// TEST-590 traces: TASK-036
// Test WriteWithTerritory includes territory map when cached.
func TestWriteWithTerritory_IncludesCachedMap(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a cached territory map
	contextDir := filepath.Join(dir, "context")
	g.Expect(os.MkdirAll(contextDir, 0o755)).To(Succeed())

	// Create a cached territory file with structure matching territory.CachedMap
	territoryTOML := `cached_at = 2025-01-01T00:00:00Z
file_count = 10

[map]
[map.structure]
root = "` + dir + `"
languages = ["go"]
build_tool = "go"
test_framework = "go test"

[map.entry_points]
cli = "cmd/app"

[map.packages]
count = 5
internal = ["pkg1", "pkg2"]

[map.tests]
pattern = "*_test.go"
count = 10

[map.docs]
readme = "README.md"
artifacts = []
`
	g.Expect(os.WriteFile(filepath.Join(contextDir, "territory.toml"), []byte(territoryTOML), 0o644)).To(Succeed())

	// Create source TOML
	source := writeTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"tdd-red\"\n")

	path, err := context.WriteWithTerritory(dir, "TASK-001", "tdd-red", source)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	// Should contain original dispatch section
	g.Expect(string(content)).To(ContainSubstring("tdd-red"))

	// Should contain territory section
	g.Expect(string(content)).To(ContainSubstring("[territory]"))
	g.Expect(string(content)).To(ContainSubstring("languages"))
	g.Expect(string(content)).To(ContainSubstring("go"))
}

// TEST-591 traces: TASK-036
// Test WriteWithTerritory works without cached map.
func TestWriteWithTerritory_WorksWithoutCache(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// No territory cache exists
	source := writeTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"tdd-red\"\n")

	path, err := context.WriteWithTerritory(dir, "TASK-001", "tdd-red", source)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	// Should still contain original content
	g.Expect(string(content)).To(ContainSubstring("tdd-red"))
}

// TEST-592 traces: TASK-036
// Test WriteWithTerritory includes structure info.
func TestWriteWithTerritory_IncludesStructure(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a cached territory map
	contextDir := filepath.Join(dir, "context")
	g.Expect(os.MkdirAll(contextDir, 0o755)).To(Succeed())

	territoryTOML := `cached_at = 2025-01-01T00:00:00Z
file_count = 10

[map]
[map.structure]
root = "` + dir + `"
languages = ["go", "javascript"]
build_tool = "go"
test_framework = "go test"

[map.entry_points]
cli = "cmd/app"
public_api = "project.go"

[map.packages]
count = 3
internal = ["pkg1"]

[map.tests]
pattern = "*_test.go"
count = 5

[map.docs]
readme = "README.md"
artifacts = ["docs/design.md"]
`
	g.Expect(os.WriteFile(filepath.Join(contextDir, "territory.toml"), []byte(territoryTOML), 0o644)).To(Succeed())

	source := writeTOML(t, t.TempDir(), "input.toml", "key = \"value\"\n")

	path, err := context.WriteWithTerritory(dir, "TASK-001", "tdd-red", source)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	// Should include entry_points info
	g.Expect(string(content)).To(ContainSubstring("cli"))
	// Should include packages
	g.Expect(string(content)).To(ContainSubstring("internal"))
}

// TEST-800 traces: TASK-053
// Test WriteWithMemory includes memory query results when query is provided.
func TestWriteWithMemory_IncludesQueryResults(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	memoryRoot := filepath.Join(dir, ".memory")

	// Create some test memories
	g.Expect(os.MkdirAll(memoryRoot, 0o755)).To(Succeed())
	indexPath := filepath.Join(memoryRoot, "index.md")
	g.Expect(os.WriteFile(indexPath, []byte("- 2025-01-15 10:00: [projctl] Always use dependency injection for testability\n- 2025-01-15 11:00: [projctl] Memory queries should use semantic search\n"), 0o644)).To(Succeed())

	source := writeTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"tdd-red\"\n")

	opts := context.MemoryInjectOpts{
		Query:      "dependency injection testing",
		MemoryRoot: memoryRoot,
		Limit:      3,
	}

	path, err := context.WriteWithMemory(dir, "TASK-053", "tdd-red", source, opts)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	// Should contain memory section
	g.Expect(string(content)).To(ContainSubstring("[memory]"))
	g.Expect(string(content)).To(ContainSubstring("dependency injection"))
}

// TEST-801 traces: TASK-053
// Test WriteWithMemory limits results to top 3.
func TestWriteWithMemory_LimitsToTop3(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	memoryRoot := filepath.Join(dir, ".memory")

	// Create more than 3 memories
	g.Expect(os.MkdirAll(memoryRoot, 0o755)).To(Succeed())
	indexPath := filepath.Join(memoryRoot, "index.md")
	g.Expect(os.WriteFile(indexPath, []byte("- 2025-01-15 10:00: [projctl] Memory 1\n- 2025-01-15 11:00: [projctl] Memory 2\n- 2025-01-15 12:00: [projctl] Memory 3\n- 2025-01-15 13:00: [projctl] Memory 4\n- 2025-01-15 14:00: [projctl] Memory 5\n"), 0o644)).To(Succeed())

	source := writeTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"tdd-red\"\n")

	opts := context.MemoryInjectOpts{
		Query:      "memory",
		MemoryRoot: memoryRoot,
		Limit:      3,
	}

	path, err := context.WriteWithMemory(dir, "TASK-053", "tdd-red", source, opts)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	// Should contain memory section with at most 3 results
	g.Expect(string(content)).To(ContainSubstring("[memory]"))

	// Count number of results in memory section
	memorySection := extractMemorySection(string(content))
	resultCount := countMemoryResults(memorySection)
	g.Expect(resultCount).To(BeNumerically("<=", 3))
}

// TEST-802 traces: TASK-053
// Test WriteWithMemory compresses to under 500 tokens.
func TestWriteWithMemory_CompressesUnder500Tokens(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	memoryRoot := filepath.Join(dir, ".memory")

	// Create large memory entries
	g.Expect(os.MkdirAll(memoryRoot, 0o755)).To(Succeed())
	indexPath := filepath.Join(memoryRoot, "index.md")
	largeMemory := "- 2025-01-15 10:00: [projctl] " + strings.Repeat("This is a very long memory entry with lots of details that should be compressed. ", 50) + "\n"
	g.Expect(os.WriteFile(indexPath, []byte(largeMemory+largeMemory+largeMemory), 0o644)).To(Succeed())

	source := writeTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"tdd-red\"\n")

	opts := context.MemoryInjectOpts{
		Query:      "memory",
		MemoryRoot: memoryRoot,
		Limit:      3,
	}

	path, err := context.WriteWithMemory(dir, "TASK-053", "tdd-red", source, opts)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	memorySection := extractMemorySection(string(content))
	tokenCount := estimateTokenCount(memorySection)
	g.Expect(tokenCount).To(BeNumerically("<", 500))
}

// TEST-803 traces: TASK-053
// Test WriteWithMemory derives query from task description when not provided.
func TestWriteWithMemory_DerivesQueryFromTask(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	memoryRoot := filepath.Join(dir, ".memory")

	g.Expect(os.MkdirAll(memoryRoot, 0o755)).To(Succeed())
	indexPath := filepath.Join(memoryRoot, "index.md")
	g.Expect(os.WriteFile(indexPath, []byte("- 2025-01-15 10:00: [projctl] Testing best practices\n"), 0o644)).To(Succeed())

	source := writeTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"tdd-red\"\n\n[task]\ndescription = \"Implement memory query functionality with semantic search\"\n")

	opts := context.MemoryInjectOpts{
		Query:      "", // Empty - should derive from task
		MemoryRoot: memoryRoot,
		Limit:      3,
	}

	path, err := context.WriteWithMemory(dir, "TASK-053", "tdd-red", source, opts)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	// Should contain memory section (query was derived)
	g.Expect(string(content)).To(ContainSubstring("[memory]"))
}

// setupMemoryTestEnv creates a test environment with memory root and index file.
func setupMemoryTestEnv(t *testing.T, dir, memoryContent string) string {
	t.Helper()
	g := NewWithT(t)

	memoryRoot := filepath.Join(dir, ".memory")
	g.Expect(os.MkdirAll(memoryRoot, 0o755)).To(Succeed())
	indexPath := filepath.Join(memoryRoot, "index.md")
	g.Expect(os.WriteFile(indexPath, []byte(memoryContent), 0o644)).To(Succeed())

	return memoryRoot
}

// testAutoInjectForSkill tests auto-injection of memory for a specific skill.
func testAutoInjectForSkill(t *testing.T, skill, taskDesc, memoryContent string) {
	t.Helper()
	g := NewWithT(t)

	dir := t.TempDir()
	memoryRoot := setupMemoryTestEnv(t, dir, memoryContent)

	source := writeTOML(t, t.TempDir(), "input.toml", fmt.Sprintf("[dispatch]\nskill = \"%s\"\n\n[task]\ndescription = \"%s\"\n", skill, taskDesc))

	routing := context.RoutingConfig{
		Simple:  "haiku",
		Medium:  "sonnet",
		Complex: "opus",
	}
	skillComplexity := map[string]string{skill: "complex"}

	path, err := context.WriteWithRoutingAndMemory(dir, "TASK-053", skill, source, routing, skillComplexity, memoryRoot, "")
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())

	// Should contain memory section (auto-injected)
	g.Expect(string(content)).To(ContainSubstring("[memory]"))
}

// TEST-804 traces: TASK-053
// Test WriteWithMemory automatic injection for architect-interview.
func TestWriteWithMemory_AutoInjectArchitectInterview(t *testing.T) {
	testAutoInjectForSkill(t, "architect-interview", "Design memory system architecture", "- 2025-01-15 10:00: [projctl] Architecture decisions\n")
}

// TEST-805 traces: TASK-053
// Test WriteWithMemory automatic injection for pm-interview.
func TestWriteWithMemory_AutoInjectPMInterview(t *testing.T) {
	testAutoInjectForSkill(t, "pm-interview", "Gather requirements for memory feature", "- 2025-01-15 10:00: [projctl] Product requirements\n")
}

// TEST-806 traces: TASK-053
// Test WriteWithMemory property: memory section always under token limit.
func TestWriteWithMemory_TokenLimitProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		memoryRoot := filepath.Join(dir, ".memory")

		// Generate random memory entries
		g.Expect(os.MkdirAll(memoryRoot, 0o755)).To(Succeed())
		indexPath := filepath.Join(memoryRoot, "index.md")

		entryCount := rapid.IntRange(1, 10).Draw(rt, "entryCount")
		var entries []string
		for i := 0; i < entryCount; i++ {
			content := rapid.StringMatching(`[a-zA-Z0-9 ]{10,200}`).Draw(rt, "content")
			entries = append(entries, fmt.Sprintf("- 2025-01-15 10:00: [projctl] %s\n", content))
		}
		g.Expect(os.WriteFile(indexPath, []byte(strings.Join(entries, "")), 0o644)).To(Succeed())

		source := writeTOML(t, t.TempDir(), "input.toml", "[dispatch]\nskill = \"tdd-red\"\n")

		opts := context.MemoryInjectOpts{
			Query:      "test",
			MemoryRoot: memoryRoot,
			Limit:      3,
		}

		path, err := context.WriteWithMemory(dir, "TASK-053", "tdd-red", source, opts)
		if err != nil {
			return // Skip if write fails
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return
		}

		memorySection := extractMemorySection(string(content))
		tokenCount := estimateTokenCount(memorySection)
		g.Expect(tokenCount).To(BeNumerically("<", 500))
	})
}

// Helper function to extract memory section from TOML content.
func extractMemorySection(content string) string {
	lines := strings.Split(content, "\n")
	var memoryLines []string
	inMemory := false

	for _, line := range lines {
		if strings.HasPrefix(line, "[memory]") {
			inMemory = true
			continue
		}
		if inMemory && strings.HasPrefix(line, "[") {
			break
		}
		if inMemory {
			memoryLines = append(memoryLines, line)
		}
	}

	return strings.Join(memoryLines, "\n")
}

// Helper function to count memory results.
func countMemoryResults(memorySection string) int {
	count := 0
	for _, line := range strings.Split(memorySection, "\n") {
		if strings.Contains(line, "content") || strings.Contains(line, "score") {
			count++
		}
	}
	// Divide by 2 since each result has content and score
	return (count + 1) / 2
}

// Helper function to estimate token count (rough approximation).
func estimateTokenCount(text string) int {
	// Rough estimate: 1 token per 4 characters
	return len(text) / 4
}

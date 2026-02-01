package context_test

import (
	"os"
	"path/filepath"
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

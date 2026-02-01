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

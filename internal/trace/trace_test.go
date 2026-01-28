package trace_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/trace"
	"pgregory.net/rapid"
)

func TestValidID(t *testing.T) {
	t.Run("accepts valid IDs", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(trace.ValidID("REQ-001")).To(BeTrue())
		g.Expect(trace.ValidID("DES-042")).To(BeTrue())
		g.Expect(trace.ValidID("ARCH-123")).To(BeTrue())
		g.Expect(trace.ValidID("TASK-007")).To(BeTrue())
	})

	t.Run("rejects invalid IDs", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(trace.ValidID("REQ-1")).To(BeFalse())
		g.Expect(trace.ValidID("CONF-001")).To(BeFalse())
		g.Expect(trace.ValidID("req-001")).To(BeFalse())
		g.Expect(trace.ValidID("REQ001")).To(BeFalse())
		g.Expect(trace.ValidID("")).To(BeFalse())
		g.Expect(trace.ValidID("REQ-0001")).To(BeFalse())
	})
}

func TestValidIDProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		prefix := rapid.SampledFrom([]string{"REQ", "DES", "ARCH", "TASK"}).Draw(rt, "prefix")
		num := rapid.IntRange(0, 999).Draw(rt, "num")

		id := prefix + "-" + padNumber(num)
		g.Expect(trace.ValidID(id)).To(BeTrue())
	})
}

func padNumber(n int) string {
	return fmt.Sprintf("%03d", n)
}

func TestAdd(t *testing.T) {
	t.Run("adds link to empty matrix", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := trace.Add(dir, "REQ-001", []string{"DES-001", "ARCH-001"})
		g.Expect(err).ToNot(HaveOccurred())

		// File should exist
		_, err = os.Stat(filepath.Join(dir, trace.TraceFile))
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("appends to existing link", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := trace.Add(dir, "REQ-001", []string{"DES-001"})
		g.Expect(err).ToNot(HaveOccurred())

		err = trace.Add(dir, "REQ-001", []string{"ARCH-001"})
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("rejects duplicate link", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := trace.Add(dir, "REQ-001", []string{"DES-001"})
		g.Expect(err).ToNot(HaveOccurred())

		err = trace.Add(dir, "REQ-001", []string{"DES-001"})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("duplicate"))
	})

	t.Run("rejects invalid source ID", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := trace.Add(dir, "INVALID", []string{"DES-001"})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("invalid source"))
	})

	t.Run("rejects invalid target ID", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := trace.Add(dir, "REQ-001", []string{"bad"})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("invalid target"))
	})

	t.Run("supports comma-separated targets", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := trace.Add(dir, "REQ-002", []string{"DES-001", "DES-002", "ARCH-001"})
		g.Expect(err).ToNot(HaveOccurred())
	})
}

func TestValidate(t *testing.T) {
	t.Run("passes with complete coverage", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create artifacts with IDs
		writeArtifact(t, dir, "requirements.md", "### REQ-001: Feature\n")
		writeArtifact(t, dir, "architecture.md", "### ARCH-001: Decision\n")
		writeArtifact(t, dir, "tasks.md", "### TASK-001: Implement\n")

		// Add trace links
		g.Expect(trace.Add(dir, "REQ-001", []string{"ARCH-001"})).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-001", []string{"TASK-001"})).To(Succeed())

		result, err := trace.Validate(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeTrue())
		g.Expect(result.OrphanIDs).To(BeEmpty())
		g.Expect(result.UnlinkedIDs).To(BeEmpty())
		g.Expect(result.MissingCoverage).To(BeEmpty())
	})

	t.Run("detects unlinked IDs", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		writeArtifact(t, dir, "requirements.md", "### REQ-001: Feature\n### REQ-002: Other\n")
		writeArtifact(t, dir, "architecture.md", "### ARCH-001: Decision\n")
		writeArtifact(t, dir, "tasks.md", "### TASK-001: Implement\n")

		g.Expect(trace.Add(dir, "REQ-001", []string{"ARCH-001"})).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-001", []string{"TASK-001"})).To(Succeed())

		result, err := trace.Validate(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeFalse())
		g.Expect(result.UnlinkedIDs).To(ContainElement("REQ-002"))
	})

	t.Run("detects missing coverage", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		writeArtifact(t, dir, "requirements.md", "### REQ-001: Feature\n")
		writeArtifact(t, dir, "architecture.md", "### ARCH-001: Decision\n")
		writeArtifact(t, dir, "tasks.md", "### TASK-001: Implement\n")

		// REQ-001 → ARCH-001 but ARCH-001 has no TASK link
		g.Expect(trace.Add(dir, "REQ-001", []string{"ARCH-001"})).To(Succeed())

		result, err := trace.Validate(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeFalse())
		g.Expect(result.MissingCoverage).ToNot(BeEmpty())
	})

	t.Run("REQ to ARCH satisfies coverage (DES or ARCH rule)", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		writeArtifact(t, dir, "requirements.md", "### REQ-001: Feature\n")
		writeArtifact(t, dir, "architecture.md", "### ARCH-001: Decision\n")
		writeArtifact(t, dir, "tasks.md", "### TASK-001: Implement\n")

		// REQ→ARCH is sufficient (DES is not required for every REQ)
		g.Expect(trace.Add(dir, "REQ-001", []string{"ARCH-001"})).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-001", []string{"TASK-001"})).To(Succeed())

		result, err := trace.Validate(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Pass).To(BeTrue())
	})
}

func TestImpact(t *testing.T) {
	t.Run("forward impact", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		g.Expect(trace.Add(dir, "REQ-001", []string{"DES-001", "ARCH-001"})).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-001", []string{"TASK-001", "TASK-002"})).To(Succeed())

		result, err := trace.Impact(dir, "REQ-001", false)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.AffectedIDs).To(ContainElements("DES-001", "ARCH-001", "TASK-001", "TASK-002"))
		g.Expect(result.Reverse).To(BeFalse())
	})

	t.Run("backward impact", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		g.Expect(trace.Add(dir, "REQ-001", []string{"ARCH-001"})).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-001", []string{"TASK-001"})).To(Succeed())

		result, err := trace.Impact(dir, "TASK-001", true)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.AffectedIDs).To(ContainElements("ARCH-001", "REQ-001"))
		g.Expect(result.Reverse).To(BeTrue())
	})

	t.Run("handles cycles gracefully", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Shouldn't happen but test it doesn't infinite loop
		g.Expect(trace.Add(dir, "REQ-001", []string{"ARCH-001"})).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-001", []string{"REQ-001"})).To(Succeed())

		result, err := trace.Impact(dir, "REQ-001", false)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.AffectedIDs).To(ContainElement("ARCH-001"))
	})

	t.Run("rejects invalid ID", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := trace.Impact(dir, "bad-id", false)
		g.Expect(err).To(HaveOccurred())
	})
}

func writeArtifact(t *testing.T, dir, name, content string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}

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
		g.Expect(trace.ValidID("ISSUE-001")).To(BeTrue())
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
		prefix := rapid.SampledFrom([]string{"ISSUE", "REQ", "DES", "ARCH", "TASK"}).Draw(rt, "prefix")
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

	t.Run("accepts ISSUE linking to REQ", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := trace.Add(dir, "ISSUE-001", []string{"REQ-001"})
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("rejects ISSUE linking to non-REQ", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := trace.Add(dir, "ISSUE-001", []string{"DES-001"})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("ISSUE can only link to REQ"))
	})

	t.Run("rejects ISSUE linking to ARCH", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := trace.Add(dir, "ISSUE-001", []string{"ARCH-001"})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("ISSUE can only link to REQ"))
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

	t.Run("detects ISSUE IDs in issues.md", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create artifacts including issues.md
		writeArtifact(t, dir, "issues.md", "### ISSUE-001: Bug report\n")
		writeArtifact(t, dir, "requirements.md", "### REQ-001: Feature\n")
		writeArtifact(t, dir, "architecture.md", "### ARCH-001: Decision\n")
		writeArtifact(t, dir, "tasks.md", "### TASK-001: Implement\n")

		// Add trace links including ISSUE
		g.Expect(trace.Add(dir, "ISSUE-001", []string{"REQ-001"})).To(Succeed())
		g.Expect(trace.Add(dir, "REQ-001", []string{"ARCH-001"})).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-001", []string{"TASK-001"})).To(Succeed())

		result, err := trace.Validate(dir)
		g.Expect(err).ToNot(HaveOccurred())
		// ISSUE-001 should be detected in issues.md, so no orphans
		g.Expect(result.OrphanIDs).ToNot(ContainElement("ISSUE-001"))
	})

	t.Run("ISSUE with no downstream passes (optional coverage)", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// ISSUE exists but has no downstream links
		writeArtifact(t, dir, "issues.md", "### ISSUE-001: Bug report\n")

		// No links from ISSUE-001

		result, err := trace.Validate(dir)
		g.Expect(err).ToNot(HaveOccurred())
		// ISSUE should NOT appear in MissingCoverage (no mandatory downstream)
		for _, mc := range result.MissingCoverage {
			g.Expect(mc.ID).ToNot(HavePrefix("ISSUE-"), "ISSUE should not have coverage requirements")
		}
	})

	t.Run("ISSUE with downstream REQ passes coverage", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		writeArtifact(t, dir, "issues.md", "### ISSUE-001: Bug report\n")
		writeArtifact(t, dir, "requirements.md", "### REQ-001: Feature\n")
		writeArtifact(t, dir, "architecture.md", "### ARCH-001: Decision\n")
		writeArtifact(t, dir, "tasks.md", "### TASK-001: Implement\n")

		// Complete chain starting from ISSUE
		g.Expect(trace.Add(dir, "ISSUE-001", []string{"REQ-001"})).To(Succeed())
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

	t.Run("forward impact from ISSUE", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		g.Expect(trace.Add(dir, "ISSUE-001", []string{"REQ-001"})).To(Succeed())
		g.Expect(trace.Add(dir, "REQ-001", []string{"ARCH-001"})).To(Succeed())
		g.Expect(trace.Add(dir, "ARCH-001", []string{"TASK-001"})).To(Succeed())

		result, err := trace.Impact(dir, "ISSUE-001", false)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.AffectedIDs).To(ContainElements("REQ-001", "ARCH-001", "TASK-001"))
		g.Expect(result.Reverse).To(BeFalse())
	})

	t.Run("backward impact to ISSUE", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		g.Expect(trace.Add(dir, "ISSUE-001", []string{"REQ-001"})).To(Succeed())
		g.Expect(trace.Add(dir, "REQ-001", []string{"ARCH-001"})).To(Succeed())

		result, err := trace.Impact(dir, "ARCH-001", true)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.AffectedIDs).To(ContainElements("REQ-001", "ISSUE-001"))
		g.Expect(result.Reverse).To(BeTrue())
	})
}

func writeArtifact(t *testing.T, dir, name, content string) {
	t.Helper()

	// Write to docs/ subdirectory to match default config
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err := os.WriteFile(filepath.Join(docsDir, name), []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}

// TEST-230 traces: TASK-003
func TestRepair(t *testing.T) {
	t.Run("detects dangling references", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create a design doc that references a non-existent requirement
		writeArtifact(t, dir, "design.md", `# Design

### DES-001: Some Design

Design description.

**Traces to:** REQ-999
`)

		result, err := trace.Repair(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.DanglingRefs).To(ContainElement("REQ-999"))
	})

	t.Run("detects duplicate IDs in different files", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create two design files with same ID
		writeArtifact(t, dir, "design.md", `# Design

### DES-001: First Design

First.

**Traces to:** REQ-001
`)
		writeArtifact(t, dir, "design-feature.md", `# Feature Design

### DES-001: Duplicate Design

Duplicate.

**Traces to:** REQ-002
`)

		result, err := trace.Repair(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.DuplicateIDs).To(ContainElement("DES-001"))
	})

	t.Run("returns empty result when no issues", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		writeArtifact(t, dir, "requirements.md", `# Requirements

### REQ-001: Requirement

Description.
`)
		writeArtifact(t, dir, "design.md", `# Design

### DES-001: Design

Description.

**Traces to:** REQ-001
`)

		result, err := trace.Repair(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.DanglingRefs).To(BeEmpty())
		g.Expect(result.DuplicateIDs).To(BeEmpty())
	})

	t.Run("reports all issues found", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		writeArtifact(t, dir, "design.md", `# Design

### DES-001: First

First.

**Traces to:** REQ-999, REQ-998
`)

		result, err := trace.Repair(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.DanglingRefs).To(HaveLen(2))
		g.Expect(result.DanglingRefs).To(ContainElements("REQ-999", "REQ-998"))
	})

	t.Run("auto-renumbers duplicate IDs", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		writeArtifact(t, dir, "design.md", `# Design

### DES-001: First Design

First.

**Traces to:** REQ-001
`)
		writeArtifact(t, dir, "design-feature.md", `# Feature Design

### DES-001: Duplicate Design

Duplicate.

**Traces to:** REQ-002
`)

		result, err := trace.Repair(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Should have renumbered the duplicate
		g.Expect(result.Renumbered).To(HaveLen(1))
		g.Expect(result.Renumbered[0].OldID).To(Equal("DES-001"))
		g.Expect(result.Renumbered[0].NewID).To(Equal("DES-002"))

		// Check the file was actually updated
		content, err := os.ReadFile(filepath.Join(dir, "docs", "design-feature.md"))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).To(ContainSubstring("DES-002"))
		g.Expect(string(content)).ToNot(ContainSubstring("DES-001"))
	})

	t.Run("creates escalation for dangling refs", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		writeArtifact(t, dir, "design.md", `# Design

### DES-001: Some Design

Design.

**Traces to:** REQ-999
`)

		result, err := trace.Repair(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Should create escalation for dangling ref
		g.Expect(result.Escalations).To(HaveLen(1))
		g.Expect(result.Escalations[0].ID).To(Equal("REQ-999"))
		g.Expect(result.Escalations[0].Reason).To(ContainSubstring("dangling"))
	})
}

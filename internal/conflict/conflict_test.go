package conflict_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/conflict"
)

func TestCreate(t *testing.T) {
	t.Run("creates first conflict", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		id, err := conflict.Create(dir, "pm, architect", "REQ-001, ARCH-003", "PM requires sync but architect says eventual consistency")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(id).To(Equal("CONF-001"))

		// File should exist
		_, err = os.Stat(filepath.Join(dir, conflict.ConflictFile))
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("auto-increments ID", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		id1, err := conflict.Create(dir, "pm, design", "REQ-001", "First conflict")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(id1).To(Equal("CONF-001"))

		id2, err := conflict.Create(dir, "design, architect", "DES-002", "Second conflict")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(id2).To(Equal("CONF-002"))
	})
}

func TestCheck(t *testing.T) {
	t.Run("finds resolved conflicts", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Write a conflicts file with one resolved entry
		content := `# Conflicts

### CONF-001

**Skills:** pm, architect
**Traceability:** REQ-001
**Status:** resolved
**Description:** Sync vs eventual consistency
**Resolution:** Use eventual consistency with UI optimistic updates
`
		g.Expect(os.WriteFile(filepath.Join(dir, conflict.ConflictFile), []byte(content), 0o644)).To(Succeed())

		result, err := conflict.Check(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Resolved).To(HaveLen(1))
		g.Expect(result.Resolved[0].ID).To(Equal("CONF-001"))
		g.Expect(result.Resolved[0].Resolution).To(ContainSubstring("eventual consistency"))
	})

	t.Run("returns empty for no resolved", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := conflict.Create(dir, "pm, design", "REQ-001", "Unresolved conflict")
		g.Expect(err).ToNot(HaveOccurred())

		result, err := conflict.Check(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Resolved).To(BeEmpty())
	})

	t.Run("returns empty for no file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		result, err := conflict.Check(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Resolved).To(BeEmpty())
	})
}

func TestList(t *testing.T) {
	t.Run("lists all conflicts", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		content := `# Conflicts

### CONF-001

**Skills:** pm, architect
**Traceability:** REQ-001
**Status:** open
**Description:** First issue
**Resolution:** (pending)

### CONF-002

**Skills:** design, pm
**Traceability:** DES-002
**Status:** resolved
**Description:** Second issue
**Resolution:** Use design's approach
`
		g.Expect(os.WriteFile(filepath.Join(dir, conflict.ConflictFile), []byte(content), 0o644)).To(Succeed())

		result, err := conflict.List(dir, "")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Conflicts).To(HaveLen(2))
		g.Expect(result.Open).To(Equal(1))
		g.Expect(result.Resolved).To(Equal(1))
	})

	t.Run("filters by status", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		content := `# Conflicts

### CONF-001

**Skills:** pm, architect
**Traceability:** REQ-001
**Status:** open
**Description:** Open issue
**Resolution:** (pending)

### CONF-002

**Skills:** design, pm
**Traceability:** DES-002
**Status:** resolved
**Description:** Resolved issue
**Resolution:** Done
`
		g.Expect(os.WriteFile(filepath.Join(dir, conflict.ConflictFile), []byte(content), 0o644)).To(Succeed())

		result, err := conflict.List(dir, "open")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Conflicts).To(HaveLen(1))
		g.Expect(result.Conflicts[0].ID).To(Equal("CONF-001"))
	})

	t.Run("returns empty for no file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		result, err := conflict.List(dir, "")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Conflicts).To(BeEmpty())
	})
}

package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for metadata table get/set operations
// traces: ISSUE-184
// ============================================================================

// TEST-1100: GetMetadata returns empty string for missing key
func TestGetMetadataMissingKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	val, err := memory.GetMetadataForTest(memoryRoot, "nonexistent")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(val).To(Equal(""))
}

// TEST-1101: SetMetadata and GetMetadata round-trip
func TestMetadataRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	err := memory.SetMetadataForTest(memoryRoot, "last_optimized_at", "2026-02-09T00:00:00Z")
	g.Expect(err).ToNot(HaveOccurred())

	val, err := memory.GetMetadataForTest(memoryRoot, "last_optimized_at")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(val).To(Equal("2026-02-09T00:00:00Z"))
}

// TEST-1102: SetMetadata overwrites existing value
func TestMetadataOverwrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	err := memory.SetMetadataForTest(memoryRoot, "key1", "value1")
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.SetMetadataForTest(memoryRoot, "key1", "value2")
	g.Expect(err).ToNot(HaveOccurred())

	val, err := memory.GetMetadataForTest(memoryRoot, "key1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(val).To(Equal("value2"))
}

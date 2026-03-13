package signal_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"engram/internal/signal"
)

// T-P6f-6: GraduationQueueEmitter EmitGraduation appends pending entry.
func TestP6f6_EmitGraduationAppendsPendingEntry(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tmpDir := t.TempDir()
	queuePath := filepath.Join(tmpDir, "graduation-queue.jsonl")

	store := signal.NewGraduationStore()
	emitter := signal.NewGraduationQueueEmitter(store, queuePath)

	now := time.Now()
	err := emitter.EmitGraduation("mem/foo.toml", "CLAUDE.md", now)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	entries, listErr := store.List(queuePath)
	g.Expect(listErr).NotTo(gomega.HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(1))
	g.Expect(entries[0].MemoryPath).To(gomega.Equal("mem/foo.toml"))
	g.Expect(entries[0].Recommendation).To(gomega.Equal("CLAUDE.md"))
	g.Expect(entries[0].Status).To(gomega.Equal("pending"))
	g.Expect(entries[0].ID).NotTo(gomega.BeEmpty())
}

// T-P6f-7: GraduationQueueEmitter ID is deterministic.
func TestP6f7_IDIsDeterministicFromMemoryPath(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tmpDir := t.TempDir()
	queuePath := filepath.Join(tmpDir, "graduation-queue.jsonl")

	store := signal.NewGraduationStore()
	emitter := signal.NewGraduationQueueEmitter(store, queuePath)

	now := time.Now()
	_ = emitter.EmitGraduation("mem/foo.toml", "CLAUDE.md", now)
	_ = emitter.EmitGraduation("mem/foo.toml", "skill", now)

	entries, listErr := store.List(queuePath)
	g.Expect(listErr).NotTo(gomega.HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(2))
	g.Expect(entries[0].ID).To(gomega.Equal(entries[1].ID))
}

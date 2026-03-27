package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

func TestMemoryRecord_MaintenanceHistory_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rec := memory.MemoryRecord{
		Title:     "test memory",
		Content:   "test content",
		CreatedAt: "2026-03-27T10:00:00Z",
		UpdatedAt: "2026-03-27T10:00:00Z",
		MaintenanceHistory: []memory.MaintenanceAction{
			{
				Action:              "rewrite",
				AppliedAt:           "2026-03-20T10:00:00Z",
				EffectivenessBefore: 25.0,
				SurfacedCountBefore: 12,
				EffectivenessAfter:  0.0,
				SurfacedCountAfter:  0,
				Measured:            false,
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")
	f, err := os.Create(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = toml.NewEncoder(f).Encode(rec)
	_ = f.Close()

	g.Expect(err).NotTo(HaveOccurred())

	var loaded memory.MemoryRecord

	_, err = toml.DecodeFile(path, &loaded)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(loaded.MaintenanceHistory).To(HaveLen(1))
	g.Expect(loaded.MaintenanceHistory[0].Action).To(Equal("rewrite"))
	g.Expect(loaded.MaintenanceHistory[0].EffectivenessBefore).To(BeNumerically("~", 25.0, 0.001))
	g.Expect(loaded.MaintenanceHistory[0].Measured).To(BeFalse())
}

func TestMaintenanceAction_FeedbackCountBefore_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rec := memory.MemoryRecord{
		Title:     "feedback count test",
		Content:   "test content",
		CreatedAt: "2026-03-27T10:00:00Z",
		UpdatedAt: "2026-03-27T10:00:00Z",
		MaintenanceHistory: []memory.MaintenanceAction{
			{
				Action:              "rewrite",
				AppliedAt:           "2026-03-27T10:00:00Z",
				EffectivenessBefore: 30.0,
				SurfacedCountBefore: 10,
				FeedbackCountBefore: 7,
				Measured:            false,
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "feedback_count_test.toml")
	f, err := os.Create(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = toml.NewEncoder(f).Encode(rec)
	_ = f.Close()

	g.Expect(err).NotTo(HaveOccurred())

	var loaded memory.MemoryRecord

	_, err = toml.DecodeFile(path, &loaded)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(loaded.MaintenanceHistory).To(HaveLen(1))
	g.Expect(loaded.MaintenanceHistory[0].FeedbackCountBefore).To(Equal(7))
}

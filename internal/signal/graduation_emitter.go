package signal

import (
	"time"
)

// GraduationQueueEmitter implements maintain.GraduationEmitter by writing to GraduationStore.
type GraduationQueueEmitter struct {
	store *GraduationStore
	path  string
}

// NewGraduationQueueEmitter creates a GraduationQueueEmitter.
func NewGraduationQueueEmitter(store *GraduationStore, path string) *GraduationQueueEmitter {
	return &GraduationQueueEmitter{
		store: store,
		path:  path,
	}
}

// EmitGraduation writes a new pending graduation entry to the store.
func (g *GraduationQueueEmitter) EmitGraduation(
	memoryPath, recommendation string,
	detectedAt time.Time,
) error {
	id := GenerateGraduationID(memoryPath)

	entry := GraduationEntry{
		ID:             id,
		MemoryPath:     memoryPath,
		Recommendation: recommendation,
		Status:         "pending",
		DetectedAt:     detectedAt,
	}

	return g.store.Append(entry, g.path)
}

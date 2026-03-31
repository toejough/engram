package correct_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/correct"
	"engram/internal/memory"
)

func TestHandleDisposition_Contradiction(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{returnPath: "/data/memories/contradiction.toml"}
	modifier := &fakeModifier{}
	extraction := &correct.ExtractionResult{
		Situation: "user runs tests",
		Behavior:  "never use targ",
		Impact:    "contradicts existing memory",
		Action:    "go test directly",
		Candidates: []correct.CandidateResult{
			{
				Name:        "use-targ",
				Disposition: correct.DispositionContradiction,
				Reason:      "directly contradicts existing behavior",
			},
		},
	}

	result, err := correct.HandleDisposition(extraction, writer, modifier, "/data", "myproject")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Action).To(Equal("contradiction"))
	g.Expect(result.Path).To(Equal("/data/memories/contradiction.toml"))
	g.Expect(result.Reason).To(ContainSubstring("memory-triage"))
	g.Expect(writer.writtenRecord).NotTo(BeNil())
}

func TestHandleDisposition_Duplicate(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{}
	modifier := &fakeModifier{}
	extraction := &correct.ExtractionResult{
		Situation: "user runs tests",
		Behavior:  "always use targ",
		Impact:    "prevents direct go test usage",
		Action:    "use targ test",
		Candidates: []correct.CandidateResult{
			{
				Name:        "use-targ",
				Disposition: correct.DispositionDuplicate,
				Reason:      "same behavior already recorded",
			},
		},
	}

	result, err := correct.HandleDisposition(extraction, writer, modifier, "/data", "myproject")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Action).To(Equal("duplicate_skipped"))
	g.Expect(result.Reason).To(ContainSubstring("use-targ"))
	g.Expect(result.Reason).To(ContainSubstring("same behavior already recorded"))
	g.Expect(writer.writtenRecord).To(BeNil(), "should not write on duplicate")
}

func TestHandleDisposition_ImpactUpdate(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{}
	modifier := &fakeModifier{}
	extraction := &correct.ExtractionResult{
		Situation: "user runs tests",
		Behavior:  "always use targ",
		Impact:    "new impact text",
		Action:    "use targ test",
		Candidates: []correct.CandidateResult{
			{
				Name:        "use-targ",
				Disposition: correct.DispositionImpactUpdate,
				Reason:      "impact has changed",
			},
		},
	}

	result, err := correct.HandleDisposition(extraction, writer, modifier, "/data", "myproject")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Action).To(Equal("updated"))
	g.Expect(modifier.calledPath).To(ContainSubstring("use-targ"))
	g.Expect(modifier.calledMutate).NotTo(BeNil())

	// Verify the mutate function updates impact.
	rec := &memory.MemoryRecord{Impact: "old impact"}
	modifier.calledMutate(rec)
	g.Expect(rec.Impact).To(Equal("new impact text"))
}

func TestHandleDisposition_LegitSeparate(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{returnPath: "/data/memories/separate.toml"}
	modifier := &fakeModifier{}
	extraction := &correct.ExtractionResult{
		Situation: "user is in a new project",
		Behavior:  "use different tooling",
		Impact:    "legitimate separate memory",
		Action:    "use make",
		Candidates: []correct.CandidateResult{
			{
				Name:        "use-make",
				Disposition: correct.DispositionLegitSeparate,
				Reason:      "legitimately separate memory",
			},
		},
	}

	result, err := correct.HandleDisposition(extraction, writer, modifier, "/data", "myproject")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Action).To(Equal("stored"))
	g.Expect(writer.writtenRecord).NotTo(BeNil())
}

func TestHandleDisposition_ModifierError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	modifyErr := errors.New("file not found")
	writer := &fakeWriter{}
	modifier := &fakeModifier{returnErr: modifyErr}
	extraction := &correct.ExtractionResult{
		Situation: "user runs tests",
		Behavior:  "always use targ",
		Impact:    "new impact",
		Action:    "use targ test",
		Candidates: []correct.CandidateResult{
			{
				Name:        "use-targ",
				Disposition: correct.DispositionImpactUpdate,
				Reason:      "impact changed",
			},
		},
	}

	_, err := correct.HandleDisposition(extraction, writer, modifier, "/data", "myproject")

	g.Expect(err).To(MatchError(modifyErr))
}

func TestHandleDisposition_PotentialGeneralization(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{}
	modifier := &fakeModifier{}
	extraction := &correct.ExtractionResult{
		Situation: "new broader situation",
		Behavior:  "always use targ",
		Impact:    "prevents direct go test usage",
		Action:    "use targ test",
		Candidates: []correct.CandidateResult{
			{
				Name:        "use-targ",
				Disposition: correct.DispositionPotentialGeneralization,
				Reason:      "situation is broader",
			},
		},
	}

	result, err := correct.HandleDisposition(extraction, writer, modifier, "/data", "myproject")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Action).To(Equal("updated"))
	g.Expect(modifier.calledPath).To(ContainSubstring("use-targ"))
	g.Expect(modifier.calledMutate).NotTo(BeNil())

	// Verify the mutate function updates situation.
	rec := &memory.MemoryRecord{Situation: "old narrow situation"}
	modifier.calledMutate(rec)
	g.Expect(rec.Situation).To(Equal("new broader situation"))
}

func TestHandleDisposition_Refinement(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{returnPath: "/data/memories/refined.toml"}
	modifier := &fakeModifier{}
	extraction := &correct.ExtractionResult{
		Situation: "user runs tests in CI",
		Behavior:  "use targ with flags",
		Impact:    "more specific usage",
		Action:    "targ test -v",
		Candidates: []correct.CandidateResult{
			{
				Name:        "use-targ",
				Disposition: correct.DispositionRefinement,
				Reason:      "more specific case of existing memory",
			},
		},
	}

	result, err := correct.HandleDisposition(extraction, writer, modifier, "/data", "myproject")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Action).To(Equal("refinement"))
	g.Expect(result.Path).To(Equal("/data/memories/refined.toml"))
	g.Expect(result.Reason).To(ContainSubstring("memory-triage"))
	g.Expect(writer.writtenRecord).NotTo(BeNil())
}

func TestHandleDisposition_Store(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{returnPath: "/data/memories/new.toml"}
	modifier := &fakeModifier{}
	extraction := newStoreExtraction()

	result, err := correct.HandleDisposition(extraction, writer, modifier, "/data", "myproject")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Action).To(Equal("stored"))
	g.Expect(result.Path).To(Equal("/data/memories/new.toml"))
	g.Expect(writer.writtenRecord).NotTo(BeNil())
	g.Expect(writer.writtenRecord.Situation).To(Equal("user runs tests"))
	g.Expect(writer.writtenRecord.Behavior).To(Equal("always use targ"))
}

func TestHandleDisposition_StoreBoth(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{returnPath: "/data/memories/new.toml"}
	modifier := &fakeModifier{}
	extraction := &correct.ExtractionResult{
		Situation: "user runs tests",
		Behavior:  "use targ",
		Impact:    "prevents direct go test usage",
		Action:    "targ test",
		Candidates: []correct.CandidateResult{
			{
				Name:        "use-targ",
				Disposition: correct.DispositionStoreBoth,
				Reason:      "different enough to store separately",
			},
		},
	}

	result, err := correct.HandleDisposition(extraction, writer, modifier, "/data", "myproject")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Action).To(Equal("stored"))
	g.Expect(writer.writtenRecord).NotTo(BeNil())
}

func TestHandleDisposition_Store_ProjectScoped(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{returnPath: "/data/memories/scoped.toml"}
	modifier := &fakeModifier{}
	extraction := &correct.ExtractionResult{
		Situation:     "user runs tests",
		Behavior:      "always use targ",
		Impact:        "prevents direct go test usage",
		Action:        "use targ test",
		ProjectScoped: true,
		Candidates:    []correct.CandidateResult{},
	}

	result, err := correct.HandleDisposition(extraction, writer, modifier, "/data", "myproject")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Action).To(Equal("stored"))
	g.Expect(writer.writtenRecord).NotTo(BeNil())
	g.Expect(writer.writtenRecord.ProjectScoped).To(BeTrue())
	g.Expect(writer.writtenRecord.ProjectSlug).To(Equal("myproject"))
}

func TestHandleDisposition_WriterError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writeErr := errors.New("disk full")
	writer := &fakeWriter{returnErr: writeErr}
	modifier := &fakeModifier{}
	extraction := newStoreExtraction()

	_, err := correct.HandleDisposition(extraction, writer, modifier, "/data", "myproject")

	g.Expect(err).To(MatchError(writeErr))
}

func TestHandleDisposition_WriterErrorContradiction(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writeErr := errors.New("disk full on contradiction")
	writer := &fakeWriter{returnErr: writeErr}
	modifier := &fakeModifier{}
	extraction := &correct.ExtractionResult{
		Situation: "user runs tests",
		Behavior:  "never use targ",
		Impact:    "contradicts existing",
		Action:    "go test directly",
		Candidates: []correct.CandidateResult{
			{
				Name:        "use-targ",
				Disposition: correct.DispositionContradiction,
				Reason:      "directly contradicts",
			},
		},
	}

	_, err := correct.HandleDisposition(extraction, writer, modifier, "/data", "myproject")

	g.Expect(err).To(MatchError(writeErr))
}

// fakeModifier is a test double for MemoryModifier.
type fakeModifier struct {
	calledPath   string
	calledMutate func(*memory.MemoryRecord)
	returnErr    error
}

func (fm *fakeModifier) ReadModifyWrite(path string, mutate func(*memory.MemoryRecord)) error {
	fm.calledPath = path
	fm.calledMutate = mutate

	return fm.returnErr
}

// fakeWriter is a test double for MemoryWriter.
type fakeWriter struct {
	writtenRecord *memory.MemoryRecord
	returnPath    string
	returnErr     error
}

func (fw *fakeWriter) Write(record *memory.MemoryRecord, _, _ string) (string, error) {
	fw.writtenRecord = record

	return fw.returnPath, fw.returnErr
}

func newStoreExtraction() *correct.ExtractionResult {
	return &correct.ExtractionResult{
		Situation:     "user runs tests",
		Behavior:      "always use targ",
		Impact:        "prevents direct go test usage",
		Action:        "use targ test",
		ProjectScoped: false,
		Candidates:    []correct.CandidateResult{},
	}
}

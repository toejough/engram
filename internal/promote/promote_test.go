package promote_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/promote"
	"engram/internal/registry"
)

func TestCandidatesExcludesInsufficientQuadrant(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Entry with no evaluations → Insufficient quadrant.
	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:            "memory:no-evals",
				SourceType:    "memory",
				SurfacedCount: 100,
				Evaluations:   registry.EvaluationCounters{},
			},
		},
	}

	promoter := &promote.Promoter{Registry: reg}

	candidates, err := promoter.Candidates(50)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(candidates).To(BeEmpty())
}

func TestPromoteGetError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{entries: []registry.InstructionEntry{}}
	promoter := &promote.Promoter{Registry: reg}

	err := promoter.Promote(context.Background(), "nonexistent")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, registry.ErrNotFound)).To(BeTrue())
}

// TestPromote_ConfirmError verifies Promote returns error on confirm failure.
func TestPromote_ConfirmError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "memory:test.toml",
				SourceType: "memory",
				SourcePath: "memories/test.toml",
				Title:      "Test",
			},
		},
	}

	promoter := &promote.Promoter{
		Registry:  reg,
		Generator: &fakeGenerator{content: "# skill"},
		Confirmer: &fakeConfirmerWithErr{err: errors.New("tty error")},
		MemoryLoader: func(_ string) (*promote.MemoryContent, error) {
			return &promote.MemoryContent{
				Title: "Test", Content: "c", Keywords: []string{"k"},
			}, nil
		},
	}

	err := promoter.Promote(context.Background(), "memory:test.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("confirming promotion")))
}

// TestPromote_GenerateError verifies Promote returns error on generate failure.
func TestPromote_GenerateError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "memory:test.toml",
				SourceType: "memory",
				SourcePath: "memories/test.toml",
				Title:      "Test",
			},
		},
	}

	promoter := &promote.Promoter{
		Registry:  reg,
		Generator: &fakeGenerator{err: errors.New("llm failed")},
		MemoryLoader: func(_ string) (*promote.MemoryContent, error) {
			return &promote.MemoryContent{
				Title: "Test", Content: "c", Keywords: []string{"k"},
			}, nil
		},
	}

	err := promoter.Promote(context.Background(), "memory:test.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("generating skill")))
}

// TestPromote_MemoryLoadError verifies Promote returns error on memory load failure.
func TestPromote_MemoryLoadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "memory:broken.toml",
				SourceType: "memory",
				SourcePath: "memories/broken.toml",
				Title:      "Broken",
			},
		},
	}

	promoter := &promote.Promoter{
		Registry: reg,
		MemoryLoader: func(_ string) (*promote.MemoryContent, error) {
			return nil, errors.New("corrupt toml")
		},
	}

	err := promoter.Promote(context.Background(), "memory:broken.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("loading memory")))
}

// TestPromote_MergeError verifies Promote returns error on merge failure.
func TestPromote_MergeError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "memory:test.toml",
				SourceType: "memory",
				SourcePath: "memories/test.toml",
				Title:      "Test",
			},
		},
	}

	promoter := &promote.Promoter{
		Registry:   reg,
		Generator:  &fakeGenerator{content: "# skill"},
		Writer:     newFakeWriter(),
		Registerer: &fakeRegisterer{},
		Merger:     &fakeMergerWithErr{err: errors.New("merge conflict")},
		Confirmer:  &fakeConfirmer{response: true},
		MemoryLoader: func(_ string) (*promote.MemoryContent, error) {
			return &promote.MemoryContent{
				Title: "Test", Content: "c", Keywords: []string{"k"},
			}, nil
		},
	}

	err := promoter.Promote(context.Background(), "memory:test.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("merging registry")))
}

// TestPromote_RegisterError verifies Promote returns error on register failure.
func TestPromote_RegisterError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "memory:test.toml",
				SourceType: "memory",
				SourcePath: "memories/test.toml",
				Title:      "Test",
			},
		},
	}

	promoter := &promote.Promoter{
		Registry:   reg,
		Generator:  &fakeGenerator{content: "# skill"},
		Writer:     newFakeWriter(),
		Registerer: &fakeRegistererWithErr{err: errors.New("db locked")},
		Confirmer:  &fakeConfirmer{response: true},
		MemoryLoader: func(_ string) (*promote.MemoryContent, error) {
			return &promote.MemoryContent{
				Title: "Test", Content: "c", Keywords: []string{"k"},
			}, nil
		},
	}

	err := promoter.Promote(context.Background(), "memory:test.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("registering skill")))
}

// TestPromote_RemoveError verifies Promote returns error on memory remove failure.
func TestPromote_RemoveError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "memory:test.toml",
				SourceType: "memory",
				SourcePath: "memories/test.toml",
				Title:      "Test",
			},
		},
	}

	promoter := &promote.Promoter{
		Registry:   reg,
		Generator:  &fakeGenerator{content: "# skill"},
		Writer:     newFakeWriter(),
		Registerer: &fakeRegisterer{},
		Merger:     &fakeMerger{},
		Remover:    &fakeRemoverWithErr{err: errors.New("permission denied")},
		Confirmer:  &fakeConfirmer{response: true},
		MemoryLoader: func(_ string) (*promote.MemoryContent, error) {
			return &promote.MemoryContent{
				Title: "Test", Content: "c", Keywords: []string{"k"},
			}, nil
		},
	}

	err := promoter.Promote(context.Background(), "memory:test.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("removing memory")))
}

// TestPromote_WriteError verifies Promote returns error on skill write failure.
func TestPromote_WriteError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "memory:test.toml",
				SourceType: "memory",
				SourcePath: "memories/test.toml",
				Title:      "Test",
			},
		},
	}

	promoter := &promote.Promoter{
		Registry:  reg,
		Generator: &fakeGenerator{content: "# skill"},
		Writer:    &fakeWriter{returnErr: errors.New("disk full")},
		Confirmer: &fakeConfirmer{response: true},
		MemoryLoader: func(_ string) (*promote.MemoryContent, error) {
			return &promote.MemoryContent{
				Title: "Test", Content: "c", Keywords: []string{"k"},
			}, nil
		},
	}

	err := promoter.Promote(context.Background(), "memory:test.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("writing skill")))
}

func TestSlugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"Use Targ Build System", "use-targ-build-system"},
		{"DI everywhere — no direct I/O", "di-everywhere-no-direct-io"},
		{"simple", "simple"},
		{"One Two Three Four Five Six Seven", "one-two-three-four-five"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(promote.Slugify(tt.input)).To(Equal(tt.expected))
		})
	}
}

func TestT238_CandidateDetectionThresholdFiltering(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			makeMemoryEntry("m1", 10, 8, 2),
			makeMemoryEntry("m2", 50, 40, 10),
			makeMemoryEntry("m3", 75, 60, 15),
			makeMemoryEntry("m4", 100, 80, 20),
			makeMemoryEntry("m5", 200, 160, 40),
		},
	}

	promoter := &promote.Promoter{Registry: reg}

	candidates, err := promoter.Candidates(50)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(candidates).To(HaveLen(4))
	// Sorted by surfaced_count descending.
	g.Expect(candidates[0].Entry.SurfacedCount).To(Equal(200))
	g.Expect(candidates[1].Entry.SurfacedCount).To(Equal(100))
	g.Expect(candidates[2].Entry.SurfacedCount).To(Equal(75))
	g.Expect(candidates[3].Entry.SurfacedCount).To(Equal(50))
}

func TestT239_CandidateDetectionExcludesNonMemorySources(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			makeMemoryEntry("mem1", 100, 80, 20),
			{
				ID:            "claude-md:1",
				SourceType:    "claude-md",
				SurfacedCount: 200,
				Evaluations:   registry.EvaluationCounters{Followed: 160, Ignored: 40},
			},
			{
				ID:            "skill:1",
				SourceType:    "skill",
				SurfacedCount: 150,
				Evaluations:   registry.EvaluationCounters{Followed: 120, Ignored: 30},
			},
		},
	}

	promoter := &promote.Promoter{Registry: reg}

	candidates, err := promoter.Candidates(50)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(candidates).To(HaveLen(1))
	g.Expect(candidates[0].Entry.ID).To(Equal("mem1"))
}

func TestT240_SkillFileGenerationValidFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mem := promote.MemoryContent{
		Title:       "Use Targ Build System",
		Content:     "Always use targ for builds, not raw go commands.",
		Principle:   "Use targ test, targ check, targ build for all operations.",
		AntiPattern: "Running go test directly without targ.",
		Keywords:    []string{"targ", "build", "test"},
	}

	result := promote.FormatSkill(mem)

	g.Expect(result).To(ContainSubstring("---"))
	g.Expect(result).To(ContainSubstring(`description: "Use when targ, build, test"`))
	g.Expect(result).To(ContainSubstring("# Use Targ Build System"))
	g.Expect(result).To(ContainSubstring(
		"Use targ test, targ check, targ build for all operations.",
	))
	g.Expect(result).To(ContainSubstring("## What to avoid"))
	g.Expect(result).To(ContainSubstring(
		"Running go test directly without targ.",
	))
	g.Expect(result).To(ContainSubstring("## Context"))
	g.Expect(result).To(ContainSubstring(
		"Always use targ for builds, not raw go commands.",
	))
}

func TestT241_SkillFileGenerationNoAntiPattern(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mem := promote.MemoryContent{
		Title:     "Use DI Everywhere",
		Content:   "All I/O through injected interfaces.",
		Principle: "No direct I/O calls in internal packages.",
		Keywords:  []string{"DI", "interfaces"},
	}

	result := promote.FormatSkill(mem)

	g.Expect(result).NotTo(ContainSubstring("## What to avoid"))
	g.Expect(result).To(ContainSubstring("# Use DI Everywhere"))
	g.Expect(result).To(ContainSubstring("## Context"))
	g.Expect(result).To(ContainSubstring(
		"No direct I/O calls in internal packages.",
	))
}

func TestT242_PluginRegistrationWriteToSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	writer := newFakeWriter()

	path, err := writer.Write("use-targ-build", "# skill content")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(path).To(Equal("skills/use-targ-build.md"))
	g.Expect(writer.written).To(HaveKey("use-targ-build"))
}

func TestT243_PluginRegistrationNameCollisionError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	writer := newFakeWriter()

	_, err := writer.Write("use-targ-build", "# first")
	g.Expect(err).NotTo(HaveOccurred())

	_, err = writer.Write("use-targ-build", "# second")
	g.Expect(err).To(HaveOccurred())
}

func TestT244_SourceRetirementMergeAndDelete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	merger := &fakeMerger{}
	remover := &fakeRemover{}
	registerer := &fakeRegisterer{}
	writer := newFakeWriter()

	memContent := &promote.MemoryContent{
		Title:     "Use Targ Build",
		Content:   "Always use targ.",
		Principle: "Use targ for builds.",
		Keywords:  []string{"targ", "build"},
	}

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "memory:use-targ-build.toml",
				SourceType: "memory",
				SourcePath: "memories/use-targ-build.toml",
				Title:      "Use Targ Build",
			},
		},
	}

	promoter := &promote.Promoter{
		Registry:   reg,
		Generator:  &fakeGenerator{content: "# skill"},
		Writer:     writer,
		Merger:     merger,
		Remover:    remover,
		Registerer: registerer,
		Confirmer:  &fakeConfirmer{response: true},
		MemoryLoader: func(_ string) (*promote.MemoryContent, error) {
			return memContent, nil
		},
	}

	err := promoter.Promote(context.Background(), "memory:use-targ-build.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Merger called with correct source→target.
	g.Expect(merger.merged).To(HaveLen(1))
	g.Expect(merger.merged[0].sourceID).To(Equal("memory:use-targ-build.toml"))
	g.Expect(merger.merged[0].targetID).To(Equal("skill:use-targ-build"))

	// Memory TOML deleted.
	g.Expect(remover.removed).To(HaveLen(1))
	g.Expect(remover.removed[0]).To(Equal("memories/use-targ-build.toml"))
}

func TestT245_PromoteFlowFullEndToEnd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	merger := &fakeMerger{}
	remover := &fakeRemover{}
	registerer := &fakeRegisterer{}
	writer := newFakeWriter()

	skillContent := promote.FormatSkill(promote.MemoryContent{
		Title:       "Use Targ Build",
		Content:     "Always use targ.",
		Principle:   "Use targ for builds.",
		AntiPattern: "Don't use go test directly.",
		Keywords:    []string{"targ", "build"},
	})

	memContent := &promote.MemoryContent{
		Title:       "Use Targ Build",
		Content:     "Always use targ.",
		Principle:   "Use targ for builds.",
		AntiPattern: "Don't use go test directly.",
		Keywords:    []string{"targ", "build"},
	}

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:            "memory:use-targ.toml",
				SourceType:    "memory",
				SourcePath:    "memories/use-targ.toml",
				Title:         "Use Targ Build",
				SurfacedCount: 100,
			},
		},
	}

	promoter := &promote.Promoter{
		Registry:   reg,
		Generator:  &fakeGenerator{content: skillContent},
		Writer:     writer,
		Merger:     merger,
		Remover:    remover,
		Registerer: registerer,
		Confirmer:  &fakeConfirmer{response: true},
		MemoryLoader: func(_ string) (*promote.MemoryContent, error) {
			return memContent, nil
		},
	}

	err := promoter.Promote(context.Background(), "memory:use-targ.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Skill file written.
	g.Expect(writer.written).To(HaveKey("use-targ-build"))

	// Registered in registry.
	g.Expect(registerer.registered).To(HaveLen(1))
	g.Expect(registerer.registered[0].ID).To(Equal("skill:use-targ-build"))
	g.Expect(registerer.registered[0].SourceType).To(Equal("skill"))

	// Merged.
	g.Expect(merger.merged).To(HaveLen(1))

	// Memory removed.
	g.Expect(remover.removed).To(ConsistOf("memories/use-targ.toml"))
}

func TestT246_PromoteFlowUserDeclines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	merger := &fakeMerger{}
	remover := &fakeRemover{}
	registerer := &fakeRegisterer{}
	writer := newFakeWriter()

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:            "memory:skip.toml",
				SourceType:    "memory",
				SourcePath:    "memories/skip.toml",
				Title:         "Skip This",
				SurfacedCount: 100,
			},
		},
	}

	promoter := &promote.Promoter{
		Registry:   reg,
		Generator:  &fakeGenerator{content: "# skill"},
		Writer:     writer,
		Merger:     merger,
		Remover:    remover,
		Registerer: registerer,
		Confirmer:  &fakeConfirmer{response: false},
		MemoryLoader: func(_ string) (*promote.MemoryContent, error) {
			return &promote.MemoryContent{
				Title:    "Skip This",
				Content:  "content",
				Keywords: []string{"skip"},
			}, nil
		},
	}

	err := promoter.Promote(context.Background(), "memory:skip.toml")
	g.Expect(err).NotTo(HaveOccurred())

	// Nothing should have been written, merged, or removed.
	g.Expect(writer.written).To(BeEmpty())
	g.Expect(merger.merged).To(BeEmpty())
	g.Expect(remover.removed).To(BeEmpty())
	g.Expect(registerer.registered).To(BeEmpty())
}

// TestT319_PromoteUsesPreGeneratedContent verifies Promoter.Promote uses Content
// when set, skipping the Generator (ARCH-78).
func TestT319_PromoteUsesPreGeneratedContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "memory:test.toml",
				SourceType: "memory",
				SourcePath: "memories/test.toml",
				Title:      "Test",
				Evaluations: registry.EvaluationCounters{
					Followed:     5,
					Contradicted: 0,
					Ignored:      0,
				},
				SurfacedCount: 10,
			},
		},
	}

	writer := &fakeWriter{written: make(map[string]string)}
	merger := &fakeMerger{}
	registerer := &fakeRegisterer{}
	remover := &fakeRemover{}

	generatorCalled := false
	generatorSpy := &spyGenerator{
		onGenerate: func() { generatorCalled = true },
		content:    "generated content",
	}

	promoter := &promote.Promoter{
		Registry:   reg,
		Generator:  generatorSpy,
		Writer:     writer,
		Merger:     merger,
		Remover:    remover,
		Registerer: registerer,
		Confirmer:  &fakeConfirmer{response: true},
		MemoryLoader: func(_ string) (*promote.MemoryContent, error) {
			return &promote.MemoryContent{Title: "Test", Content: "c", Keywords: []string{"k"}}, nil
		},
		Content:     "pre-built skill content",
		SkipConfirm: true,
	}

	err := promoter.Promote(context.Background(), "memory:test.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Generator must not have been called.
	g.Expect(generatorCalled).To(BeFalse())

	// Content written should be the pre-built content.
	for _, written := range writer.written {
		g.Expect(written).To(Equal("pre-built skill content"))
	}
}

// TestT320_PromoteSkipsConfirmWithSkipConfirm verifies Promoter.Promote skips
// Confirmer when SkipConfirm is true (ARCH-78).
func TestT320_PromoteSkipsConfirmWithSkipConfirm(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "memory:test.toml",
				SourceType: "memory",
				SourcePath: "memories/test.toml",
				Title:      "Test",
				Evaluations: registry.EvaluationCounters{
					Followed:     5,
					Contradicted: 0,
					Ignored:      0,
				},
				SurfacedCount: 10,
			},
		},
	}

	writer := &fakeWriter{written: make(map[string]string)}
	merger := &fakeMerger{}
	registerer := &fakeRegisterer{}
	remover := &fakeRemover{}

	// Confirmer returns false — but SkipConfirm=true should bypass it.
	confirmer := &fakeConfirmer{response: false}

	promoter := &promote.Promoter{
		Registry:   reg,
		Generator:  &fakeGenerator{content: "skill content"},
		Writer:     writer,
		Merger:     merger,
		Remover:    remover,
		Registerer: registerer,
		Confirmer:  confirmer,
		MemoryLoader: func(_ string) (*promote.MemoryContent, error) {
			return &promote.MemoryContent{Title: "Test", Content: "c", Keywords: []string{"k"}}, nil
		},
		SkipConfirm: true,
	}

	err := promoter.Promote(context.Background(), "memory:test.toml")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Promotion must have happened (writer was called).
	g.Expect(writer.written).NotTo(BeEmpty())
}

// fakeConfirmer implements Confirmer.
type fakeConfirmer struct {
	response bool
}

func (f *fakeConfirmer) Confirm(_ string) (bool, error) {
	return f.response, nil
}

// fakeConfirmerWithErr always returns an error.
type fakeConfirmerWithErr struct {
	err error
}

func (f *fakeConfirmerWithErr) Confirm(_ string) (bool, error) {
	return false, f.err
}

// fakeGenerator implements SkillGenerator.
type fakeGenerator struct {
	content string
	err     error
}

func (f *fakeGenerator) Generate(_ context.Context, _ promote.MemoryContent) (string, error) {
	return f.content, f.err
}

// fakeMerger implements RegistryMerger.
type fakeMerger struct {
	merged []mergeCall
}

func (f *fakeMerger) Merge(sourceID, targetID string) error {
	f.merged = append(f.merged, mergeCall{sourceID: sourceID, targetID: targetID})

	return nil
}

// fakeMergerWithErr always returns an error.
type fakeMergerWithErr struct {
	err error
}

func (f *fakeMergerWithErr) Merge(_, _ string) error {
	return f.err
}

// fakeRegisterer implements RegistryRegisterer.
type fakeRegisterer struct {
	registered []registry.InstructionEntry
}

func (f *fakeRegisterer) Register(entry registry.InstructionEntry) error {
	f.registered = append(f.registered, entry)

	return nil
}

// fakeRegistererWithErr always returns an error.
type fakeRegistererWithErr struct {
	err error
}

func (f *fakeRegistererWithErr) Register(_ registry.InstructionEntry) error {
	return f.err
}

// fakeRegistry implements RegistryReader.
type fakeRegistry struct {
	entries []registry.InstructionEntry
}

func (f *fakeRegistry) Get(id string) (*registry.InstructionEntry, error) {
	for i := range f.entries {
		if f.entries[i].ID == id {
			return &f.entries[i], nil
		}
	}

	return nil, registry.ErrNotFound
}

func (f *fakeRegistry) List() ([]registry.InstructionEntry, error) {
	return f.entries, nil
}

// fakeRemover implements MemoryRemover.
type fakeRemover struct {
	removed []string
}

func (f *fakeRemover) Remove(path string) error {
	f.removed = append(f.removed, path)

	return nil
}

// fakeRemoverWithErr always returns an error.
type fakeRemoverWithErr struct {
	err error
}

func (f *fakeRemoverWithErr) Remove(_ string) error {
	return f.err
}

// fakeWriter implements SkillWriter.
type fakeWriter struct {
	written   map[string]string
	returnErr error
}

func (f *fakeWriter) Write(name, content string) (string, error) {
	if f.returnErr != nil {
		return "", f.returnErr
	}

	path := "skills/" + name + ".md"
	if _, exists := f.written[name]; exists {
		return "", fmt.Errorf("skill %q already exists", name)
	}

	f.written[name] = content

	return path, nil
}

type mergeCall struct {
	sourceID string
	targetID string
}

// spyGenerator tracks calls to Generate.
type spyGenerator struct {
	onGenerate func()
	content    string
	err        error
}

func (s *spyGenerator) Generate(_ context.Context, _ promote.MemoryContent) (string, error) {
	if s.onGenerate != nil {
		s.onGenerate()
	}

	return s.content, s.err
}

func makeMemoryEntry(id string, surfacedCount, followed, ignored int) registry.InstructionEntry {
	return registry.InstructionEntry{
		ID:            id,
		SourceType:    "memory",
		SourcePath:    "memories/" + id + ".toml",
		Title:         id,
		SurfacedCount: surfacedCount,
		Evaluations: registry.EvaluationCounters{
			Followed: followed,
			Ignored:  ignored,
		},
	}
}

func newFakeWriter() *fakeWriter {
	return &fakeWriter{written: make(map[string]string)}
}

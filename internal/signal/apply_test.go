package signal_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/signal"
)

func TestApply_Broaden(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writer := &stubMemoryWriter{written: make(map[string]*memory.Stored)}

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return &memory.Stored{
				Title:    "Gem",
				Keywords: []string{"existing"},
			}, nil
		}),
		signal.WithWriteMemory(writer),
	)

	action := signal.ApplyAction{
		Action:   "broaden_keywords",
		Memory:   "memories/gem.toml",
		Keywords: []string{"new1", "new2"},
	}

	result, err := applier.Apply(context.Background(), action)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Success).To(gomega.BeTrue())

	stored := writer.written["memories/gem.toml"]
	g.Expect(stored).NotTo(gomega.BeNil())

	if stored == nil {
		return
	}

	g.Expect(stored.Keywords).To(gomega.Equal([]string{"existing", "new1", "new2"}))
}

func TestApply_BroadenNilMemory(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return nil, nil //nolint:nilnil // testing nil path
		}),
	)

	_, err := applier.Apply(context.Background(), signal.ApplyAction{
		Action: "broaden_keywords", Memory: "test.toml",
	})
	g.Expect(err).To(gomega.HaveOccurred())
}

func TestApply_BroadenNormalizesKeywords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writer := &stubMemoryWriter{written: make(map[string]*memory.Stored)}

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return &memory.Stored{Keywords: []string{"existing_kw"}}, nil
		}),
		signal.WithWriteMemory(writer),
	)

	_, err := applier.Apply(context.Background(), signal.ApplyAction{
		Action:   "broaden_keywords",
		Memory:   "memories/gem.toml",
		Keywords: []string{"Mixed-Case", "hyphen-sep"},
	})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	stored := writer.written["memories/gem.toml"]
	g.Expect(stored).NotTo(gomega.BeNil())

	if stored == nil {
		return
	}

	g.Expect(stored.Keywords).To(gomega.ConsistOf("existing_kw", "mixed_case", "hyphen_sep"))
}

func TestApply_BroadenReadError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return nil, errors.New("disk error")
		}),
	)

	result, err := applier.Apply(context.Background(), signal.ApplyAction{
		Action: "broaden_keywords", Memory: "test.toml",
	})
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(result.Success).To(gomega.BeFalse())
}

func TestApply_Consolidate(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	extractedMem := &memory.MemoryRecord{
		Title:     "Consolidated principle",
		Principle: "Generalized principle",
		Keywords:  []string{"general"},
	}
	extractor := &stubExtractor{result: extractedMem}
	archiver := &stubArchiver{}

	records := map[string]*memory.MemoryRecord{
		"/mem/a.toml": {Title: "Memory A", SourcePath: "/mem/a.toml", FollowedCount: 2},
		"/mem/b.toml": {Title: "Memory B", SourcePath: "/mem/b.toml", FollowedCount: 1},
		"/mem/c.toml": {Title: "Memory C", SourcePath: "/mem/c.toml", FollowedCount: 3},
	}

	writer := &stubMemoryWriter{written: make(map[string]*memory.Stored)}

	applier := signal.NewApplier(
		signal.WithWriteMemory(writer),
		signal.WithApplyExtractor(extractor),
		signal.WithApplyArchiver(archiver),
		signal.WithLoadRecord(func(path string) (*memory.MemoryRecord, error) {
			rec, ok := records[path]
			if !ok {
				return nil, fmt.Errorf("not found: %s", path)
			}

			return rec, nil
		}),
	)

	membersJSON, marshalErr := json.Marshal([]map[string]string{
		{"path": "/mem/a.toml", "title": "Memory A"},
		{"path": "/mem/b.toml", "title": "Memory B"},
		{"path": "/mem/c.toml", "title": "Memory C"},
	})
	g.Expect(marshalErr).NotTo(gomega.HaveOccurred())

	if marshalErr != nil {
		return
	}

	action := signal.ApplyAction{
		Action: "consolidate",
		Memory: "/mem/a.toml",
		Fields: map[string]any{
			"members": json.RawMessage(membersJSON),
		},
	}

	result, err := applier.Apply(context.Background(), action)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Success).To(gomega.BeTrue())

	g.Expect(extractor.calledWith).NotTo(gomega.BeNil())

	if extractor.calledWith == nil {
		return
	}

	g.Expect(extractor.calledWith.Members).To(gomega.HaveLen(3))

	g.Expect(archiver.archived).To(gomega.ConsistOf("/mem/b.toml", "/mem/c.toml"))
}

func TestApply_ConsolidateNilExtractor(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier()

	action := signal.ApplyAction{
		Action: "consolidate",
		Memory: "/mem/a.toml",
		Fields: map[string]any{"members": json.RawMessage(`[{"path":"/mem/a.toml"}]`)},
	}

	_, err := applier.Apply(context.Background(), action)
	g.Expect(err).To(gomega.HaveOccurred())
}

func TestApply_EscalateCallsEnforcementApplier(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	enfApplier := &stubEnforcementApplier{}

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return &memory.Stored{Content: "some content"}, nil
		}),
		signal.WithEnforcementApplier(enfApplier),
	)

	action := signal.ApplyAction{
		Action: "escalate",
		Memory: "memories/problem.toml",
		Level:  2, // emphasized_advisory
	}

	result, err := applier.Apply(context.Background(), action)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Success).To(gomega.BeTrue())
	g.Expect(enfApplier.calls).To(gomega.HaveLen(1))
	g.Expect(enfApplier.calls[0].level).To(gomega.Equal("emphasized_advisory"))
	g.Expect(enfApplier.calls[0].id).To(gomega.Equal("memories/problem.toml"))
}

func TestApply_EscalateNilApplierNoOp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return &memory.Stored{Content: "test"}, nil
		}),
	)

	action := signal.ApplyAction{
		Action: "escalate",
		Memory: "memories/problem.toml",
		Level:  2,
	}

	result, err := applier.Apply(context.Background(), action)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Success).To(gomega.BeTrue())
}

func TestApply_EscalateZeroLevel(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier()

	result, err := applier.Apply(context.Background(), signal.ApplyAction{
		Action: "escalate", Memory: "test.toml", Level: 0,
	})
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(result.Success).To(gomega.BeFalse())
}

func TestApply_RefineKeywords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writer := &stubMemoryWriter{written: make(map[string]*memory.Stored)}

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return &memory.Stored{
				Title:             "Noisy",
				Keywords:          []string{"code", "testing", "specific-good"},
				IrrelevantQueries: []string{"how to test", "dependency injection"},
			}, nil
		}),
		signal.WithWriteMemory(writer),
	)

	action := signal.ApplyAction{
		Action: "refine_keywords",
		Memory: "memories/noisy.toml",
		Fields: map[string]any{
			"remove_keywords": []any{"code", "testing"},
			"add_keywords":    []any{"go-test-isolation", "parallel-test-state"},
		},
	}

	result, err := applier.Apply(context.Background(), action)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Success).To(gomega.BeTrue())

	stored := writer.written["memories/noisy.toml"]
	g.Expect(stored).NotTo(gomega.BeNil())

	if stored == nil {
		return
	}

	// "code" and "testing" removed, two new ones added (normalized), "specific-good" kept.
	g.Expect(stored.Keywords).To(gomega.ConsistOf(
		"specific-good", "go_test_isolation", "parallel_test_state",
	))
	// IrrelevantQueries cleared after refinement.
	g.Expect(stored.IrrelevantQueries).To(gomega.BeEmpty())
}

func TestApply_RefineKeywordsNilMemory(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return nil, nil //nolint:nilnil // testing nil memory path
		}),
	)

	action := signal.ApplyAction{
		Action: "refine_keywords",
		Memory: "memories/gone.toml",
		Fields: map[string]any{
			"remove_keywords": []any{"old"},
			"add_keywords":    []any{"new"},
		},
	}

	_, err := applier.Apply(context.Background(), action)
	g.Expect(err).To(gomega.HaveOccurred())
}

func TestApply_Remove(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var removedPath string

	applier := signal.NewApplier(
		signal.WithRemoveFile(func(path string) error {
			removedPath = path

			return nil
		}),
	)

	action := signal.ApplyAction{
		Action: "remove",
		Memory: "memories/stale.toml",
	}

	result, err := applier.Apply(context.Background(), action)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Success).To(gomega.BeTrue())
	g.Expect(removedPath).To(gomega.Equal("memories/stale.toml"))
}

func TestApply_RemoveError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier(
		signal.WithRemoveFile(func(_ string) error {
			return errors.New("permission denied")
		}),
	)

	result, err := applier.Apply(context.Background(), signal.ApplyAction{
		Action: "remove", Memory: "test.toml",
	})
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(result.Success).To(gomega.BeFalse())
	g.Expect(result.Error).To(gomega.ContainSubstring("permission denied"))
}

func TestApply_Rewrite(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writer := &stubMemoryWriter{written: make(map[string]*memory.Stored)}

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return &memory.Stored{
				Title:   "Old Title",
				Content: "Old content",
			}, nil
		}),
		signal.WithWriteMemory(writer),
	)

	action := signal.ApplyAction{
		Action: "rewrite",
		Memory: "memories/leech.toml",
		Fields: map[string]any{
			"title":        "New Title",
			"content":      "New content",
			"principle":    "Be clear",
			"anti_pattern": "Don't be unclear",
		},
	}

	result, err := applier.Apply(context.Background(), action)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Success).To(gomega.BeTrue())

	stored := writer.written["memories/leech.toml"]
	g.Expect(stored).NotTo(gomega.BeNil())

	if stored == nil {
		return
	}

	g.Expect(stored.Title).To(gomega.Equal("New Title"))
	g.Expect(stored.Content).To(gomega.Equal("New content"))
	g.Expect(stored.Principle).To(gomega.Equal("Be clear"))
	g.Expect(stored.AntiPattern).To(gomega.Equal("Don't be unclear"))
}

func TestApply_RewriteNilMemory(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return nil, nil //nolint:nilnil // testing nil path
		}),
	)

	_, err := applier.Apply(context.Background(), signal.ApplyAction{
		Action: "rewrite", Memory: "test.toml",
	})
	g.Expect(err).To(gomega.HaveOccurred())
}

func TestApply_RewriteNormalizesKeywords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writer := &stubMemoryWriter{written: make(map[string]*memory.Stored)}

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return &memory.Stored{Title: "Old"}, nil
		}),
		signal.WithWriteMemory(writer),
	)

	_, err := applier.Apply(context.Background(), signal.ApplyAction{
		Action: "rewrite",
		Memory: "memories/leech.toml",
		Fields: map[string]any{
			"keywords": []any{"Mixed-Case", "hyphen-sep"},
		},
	})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	stored := writer.written["memories/leech.toml"]
	g.Expect(stored).NotTo(gomega.BeNil())

	if stored == nil {
		return
	}

	g.Expect(stored.Keywords).To(gomega.ConsistOf("mixed_case", "hyphen_sep"))
}

func TestApply_RewriteReadError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return nil, errors.New("disk error")
		}),
	)

	result, err := applier.Apply(context.Background(), signal.ApplyAction{
		Action: "rewrite", Memory: "test.toml",
	})
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(result.Success).To(gomega.BeFalse())
}

func TestApply_RewriteWithKeywords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writer := &stubMemoryWriter{written: make(map[string]*memory.Stored)}

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return &memory.Stored{
				Title:    "Old",
				Keywords: []string{"old"},
			}, nil
		}),
		signal.WithWriteMemory(writer),
	)

	action := signal.ApplyAction{
		Action: "rewrite",
		Memory: "memories/kw.toml",
		Fields: map[string]any{
			"keywords": []any{"new1", "new2"},
		},
	}

	result, err := applier.Apply(context.Background(), action)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Success).To(gomega.BeTrue())

	stored := writer.written["memories/kw.toml"]
	g.Expect(stored).NotTo(gomega.BeNil())

	if stored == nil {
		return
	}

	g.Expect(stored.Keywords).To(gomega.Equal([]string{"new1", "new2"}))
}

func TestApply_RewriteWriteError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return &memory.Stored{Title: "x"}, nil
		}),
		signal.WithWriteMemory(&errorWriter{}),
	)

	result, err := applier.Apply(context.Background(), signal.ApplyAction{
		Action: "rewrite", Memory: "test.toml",
	})
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(result.Success).To(gomega.BeFalse())
}

func TestApply_UnsupportedAction(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	applier := signal.NewApplier()

	_, err := applier.Apply(context.Background(), signal.ApplyAction{
		Action: "unknown", Memory: "test.toml",
	})
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(errors.Is(err, signal.ErrUnsupportedAction)).To(gomega.BeTrue())
}

type enforcementCall struct {
	id    string
	level string
}

type errorWriter struct{}

func (e *errorWriter) Write(_ string, _ *memory.Stored) error {
	return errors.New("write failed")
}

type stubArchiver struct {
	archived []string
}

func (s *stubArchiver) Archive(path string) error {
	s.archived = append(s.archived, path)
	return nil
}

type stubEnforcementApplier struct {
	calls []enforcementCall
}

func (s *stubEnforcementApplier) SetEnforcementLevel(id, level, _ string) error {
	s.calls = append(s.calls, enforcementCall{id: id, level: level})

	return nil
}

type stubExtractor struct {
	result     *memory.MemoryRecord
	calledWith *signal.ConfirmedCluster
}

func (s *stubExtractor) ExtractPrinciple(
	_ context.Context, cluster signal.ConfirmedCluster,
) (*memory.MemoryRecord, error) {
	s.calledWith = &cluster
	return s.result, nil
}

type stubMemoryWriter struct {
	written map[string]*memory.Stored
}

func (s *stubMemoryWriter) Write(path string, stored *memory.Stored) error {
	s.written[path] = stored

	return nil
}

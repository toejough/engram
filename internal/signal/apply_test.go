package signal_test

import (
	"context"
	"errors"
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

type stubEnforcementApplier struct {
	calls []enforcementCall
}

func (s *stubEnforcementApplier) SetEnforcementLevel(id, level, _ string) error {
	s.calls = append(s.calls, enforcementCall{id: id, level: level})

	return nil
}

type stubMemoryWriter struct {
	written map[string]*memory.Stored
}

func (s *stubMemoryWriter) Write(path string, stored *memory.Stored) error {
	s.written[path] = stored

	return nil
}

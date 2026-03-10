package surface_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// T-P6e-11: emphasized_advisory renders with IMPORTANT: prefix in tool mode.
func TestP6e11_EmphasizedAdvisoryToolMode(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Commit Rule",
			FilePath:    "commit-rule.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit skill",
		},
		{
			Title:       "Filler A",
			FilePath:    "filler-a.toml",
			AntiPattern: "logging without context",
			Keywords:    []string{"logging"},
			Principle:   "always log with context",
		},
		{
			Title:       "Filler B",
			FilePath:    "filler-b.toml",
			AntiPattern: "skipping tests",
			Keywords:    []string{"testing"},
			Principle:   "always write tests",
		},
		{
			Title:       "Filler C",
			FilePath:    "filler-c.toml",
			AntiPattern: "no monitoring",
			Keywords:    []string{"monitoring"},
			Principle:   "monitor everything",
		},
		{
			Title:       "Filler D",
			FilePath:    "filler-d.toml",
			AntiPattern: "no alerts",
			Keywords:    []string{"alerts"},
			Principle:   "respond to alerts",
		},
	}

	// commit-rule.toml has emphasized_advisory level.
	levelReader := &fakeEnforcementReader{
		levels: map[string]string{
			"commit-rule.toml": "emphasized_advisory",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEnforcementReader(levelReader))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m 'fix bug'",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("IMPORTANT:"))
	g.Expect(output).To(ContainSubstring("commit-rule"))
}

// T-P6e-12: reminder renders with REMINDER: prefix in tool mode.
func TestP6e12_ReminderToolMode(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Commit Rule",
			FilePath:    "commit-rule.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit skill",
		},
		{
			Title:       "Filler A",
			FilePath:    "filler-a.toml",
			AntiPattern: "logging without context",
			Keywords:    []string{"logging"},
			Principle:   "always log with context",
		},
		{
			Title:       "Filler B",
			FilePath:    "filler-b.toml",
			AntiPattern: "skipping tests",
			Keywords:    []string{"testing"},
			Principle:   "always write tests",
		},
		{
			Title:       "Filler C",
			FilePath:    "filler-c.toml",
			AntiPattern: "no monitoring",
			Keywords:    []string{"monitoring"},
			Principle:   "monitor everything",
		},
		{
			Title:       "Filler D",
			FilePath:    "filler-d.toml",
			AntiPattern: "no alerts",
			Keywords:    []string{"alerts"},
			Principle:   "respond to alerts",
		},
	}

	levelReader := &fakeEnforcementReader{
		levels: map[string]string{
			"commit-rule.toml": "reminder",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEnforcementReader(levelReader))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m 'fix bug'",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("REMINDER:"))
	g.Expect(output).To(ContainSubstring("commit-rule"))
}

// T-P6e-13: advisory level renders with normal format (no prefix).
func TestP6e13_AdvisoryToolModeNormalFormat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Commit Rule",
			FilePath:    "commit-rule.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit skill",
		},
		{
			Title:       "Filler A",
			FilePath:    "filler-a.toml",
			AntiPattern: "logging without context",
			Keywords:    []string{"logging"},
			Principle:   "always log with context",
		},
		{
			Title:       "Filler B",
			FilePath:    "filler-b.toml",
			AntiPattern: "skipping tests",
			Keywords:    []string{"testing"},
			Principle:   "always write tests",
		},
		{
			Title:       "Filler C",
			FilePath:    "filler-c.toml",
			AntiPattern: "no monitoring",
			Keywords:    []string{"monitoring"},
			Principle:   "monitor everything",
		},
		{
			Title:       "Filler D",
			FilePath:    "filler-d.toml",
			AntiPattern: "no alerts",
			Keywords:    []string{"alerts"},
			Principle:   "respond to alerts",
		},
	}

	// No enforcement reader — all memories at default advisory.
	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m 'fix bug'",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).NotTo(ContainSubstring("IMPORTANT:"))
	g.Expect(output).NotTo(ContainSubstring("REMINDER:"))
	g.Expect(output).To(ContainSubstring("commit-rule"))
}

// T-P6e-14: emphasized_advisory memories are prioritized before advisory in tool mode.
func TestP6e14_EmphasizedAdvisoryPrioritizedFirst(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Title:       "Commit Rule",
			FilePath:    "commit-rule.toml",
			AntiPattern: "manual git commit",
			Keywords:    []string{"commit", "git"},
			Principle:   "use /commit skill",
		},
		{
			Title:       "Deploy Rule",
			FilePath:    "deploy-rule.toml",
			AntiPattern: "manual git deploy commit",
			Keywords:    []string{"commit", "deploy", "git"},
			Principle:   "use deploy script for git commit",
		},
		{
			Title:       "Filler A",
			FilePath:    "filler-a.toml",
			AntiPattern: "logging",
			Keywords:    []string{"logging"},
			Principle:   "log with context",
		},
		{
			Title:       "Filler B",
			FilePath:    "filler-b.toml",
			AntiPattern: "testing",
			Keywords:    []string{"testing"},
			Principle:   "write tests",
		},
		{
			Title:       "Filler C",
			FilePath:    "filler-c.toml",
			AntiPattern: "monitoring",
			Keywords:    []string{"monitoring"},
			Principle:   "monitor",
		},
	}

	// deploy-rule.toml is emphasized_advisory, commit-rule.toml is advisory.
	levelReader := &fakeEnforcementReader{
		levels: map[string]string{
			"deploy-rule.toml": "emphasized_advisory",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEnforcementReader(levelReader))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m 'fix bug'",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	// deploy-rule should appear before commit-rule since it's emphasized_advisory.
	deployPos := strings.Index(output, "deploy-rule")
	commitPos := strings.Index(output, "commit-rule")
	g.Expect(deployPos).To(BeNumerically("<", commitPos))
}

// --- Fakes for enforcement reader ---

type fakeEnforcementReader struct {
	levels map[string]string
}

func (f *fakeEnforcementReader) GetEnforcementLevel(id string) (string, error) {
	if f.levels == nil {
		return "", errors.New("not found")
	}

	level, ok := f.levels[id]
	if !ok {
		return "", errors.New("not found")
	}

	return level, nil
}

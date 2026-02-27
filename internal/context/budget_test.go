package context_test

import (
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/context"
	"github.com/toejough/projctl/internal/log"
)

// MockFS implements log.FileSystem for testing
type MockFS struct {
	Files map[string][]byte
}

func (m *MockFS) AppendFile(path string, data []byte) error {
	if m.Files == nil {
		m.Files = make(map[string][]byte)
	}

	m.Files[path] = append(m.Files[path], data...)

	return nil
}

func (m *MockFS) FileExists(path string) bool {
	_, exists := m.Files[path]
	return exists
}

func (m *MockFS) ReadFile(path string) ([]byte, error) {
	content, exists := m.Files[path]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	return content, nil
}

// TEST-700 traces: TASK-065
func TestBudgetCheck(t *testing.T) {
	t.Parallel()
	t.Run("returns OK when under warning threshold", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte)}

		// Create log with context estimates under warning threshold
		logContent := `{"timestamp":"2026-01-31T12:00:00Z","level":"status","subject":"task-status","message":"test","context_estimate":50000}
{"timestamp":"2026-01-31T12:01:00Z","level":"status","subject":"task-status","message":"test2","context_estimate":60000}
`
		fs.Files[filepath.Join("testdir", "updates.jsonl")] = []byte(logContent)

		result, err := context.CheckBudget("testdir", context.BudgetThresholds{
			Warning: 80000,
			Limit:   90000,
		}, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Status).To(Equal(context.BudgetOK))
		g.Expect(result.CurrentEstimate).To(Equal(60000))
		g.Expect(result.ExitCode).To(Equal(0))
	})

	t.Run("returns warning when over warning but under limit", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte)}

		logContent := `{"timestamp":"2026-01-31T12:00:00Z","level":"status","subject":"task-status","message":"test","context_estimate":85000}
`
		fs.Files[filepath.Join("testdir", "updates.jsonl")] = []byte(logContent)

		result, err := context.CheckBudget("testdir", context.BudgetThresholds{
			Warning: 80000,
			Limit:   90000,
		}, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Status).To(Equal(context.BudgetWarning))
		g.Expect(result.CurrentEstimate).To(Equal(85000))
		g.Expect(result.ExitCode).To(Equal(1))
		g.Expect(result.Message).To(ContainSubstring("consider compaction"))
	})

	t.Run("returns limit exceeded when over limit", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte)}

		logContent := `{"timestamp":"2026-01-31T12:00:00Z","level":"status","subject":"task-status","message":"test","context_estimate":95000}
`
		fs.Files[filepath.Join("testdir", "updates.jsonl")] = []byte(logContent)

		result, err := context.CheckBudget("testdir", context.BudgetThresholds{
			Warning: 80000,
			Limit:   90000,
		}, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Status).To(Equal(context.BudgetExceeded))
		g.Expect(result.CurrentEstimate).To(Equal(95000))
		g.Expect(result.ExitCode).To(Equal(2))
	})

	t.Run("uses most recent context estimate from log", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte)}

		// Multiple entries with different estimates - should use the most recent
		logContent := `{"timestamp":"2026-01-31T12:00:00Z","level":"status","subject":"task-status","message":"test","context_estimate":50000}
{"timestamp":"2026-01-31T12:01:00Z","level":"status","subject":"task-status","message":"test2","context_estimate":75000}
{"timestamp":"2026-01-31T12:02:00Z","level":"status","subject":"task-status","message":"test3","context_estimate":60000}
`
		fs.Files[filepath.Join("testdir", "updates.jsonl")] = []byte(logContent)

		result, err := context.CheckBudget("testdir", context.BudgetThresholds{
			Warning: 80000,
			Limit:   90000,
		}, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.CurrentEstimate).To(Equal(60000)) // Most recent
	})

	t.Run("returns OK when no log entries", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte)}

		// No log file
		result, err := context.CheckBudget("testdir", context.BudgetThresholds{
			Warning: 80000,
			Limit:   90000,
		}, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Status).To(Equal(context.BudgetOK))
		g.Expect(result.CurrentEstimate).To(Equal(0))
	})

	t.Run("calculates percentage correctly", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte)}

		logContent := `{"timestamp":"2026-01-31T12:00:00Z","level":"status","subject":"task-status","message":"test","context_estimate":45000}
`
		fs.Files[filepath.Join("testdir", "updates.jsonl")] = []byte(logContent)

		result, err := context.CheckBudget("testdir", context.BudgetThresholds{
			Warning: 80000,
			Limit:   90000,
		}, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Percentage).To(Equal(50)) // 45000 / 90000 = 50%
	})
}

func TestDefaultThresholds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	thresholds := context.DefaultBudgetThresholds()
	g.Expect(thresholds.Warning).To(Equal(80000))
	g.Expect(thresholds.Limit).To(Equal(90000))
}

// unexported variables.
var (
	_ log.FileSystem = (*MockFS)(nil)
)

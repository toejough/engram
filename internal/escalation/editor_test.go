package escalation_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/escalation"
)

func TestOpenInEditor_CallsExecutor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	exec := &mockCmdExecutor{}

	err := escalation.OpenInEditor("/tmp/test.md", "vim", exec)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exec.called).To(ContainElement("vim"))
}

func TestOpenInEditor_PropagatesError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	exec := &mockCmdExecutor{returnErr: errors.New("editor failed")}

	err := escalation.OpenInEditor("/tmp/test.md", "nano", exec)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("editor failed"))
	}
}

func TestReviewEscalations_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &mockFS{files: make(map[string]string)}
	exec := &mockCmdExecutor{}

	items := []escalation.Escalation{
		{ID: "ESC-001", Category: "requirement", Context: "ctx", Question: "Q?", Status: "pending"},
	}

	result, err := escalation.ReviewEscalations(items, "/tmp/esc.md", func(_ string) string { return "vim" }, exec, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(1))

	if len(result) < 1 {
		t.Fatal("expected at least 1 escalation")
	}

	g.Expect(result[0].ID).To(Equal("ESC-001"))
}

func TestReviewEscalations_WriteFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	errFS := &errorWriteFS{}
	exec := &mockCmdExecutor{}

	items := []escalation.Escalation{
		{ID: "ESC-001", Category: "requirement", Context: "ctx", Question: "Q?", Status: "pending"},
	}

	_, err := escalation.ReviewEscalations(items, "/tmp/esc.md", func(_ string) string { return "vim" }, exec, errFS)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("write failed"))
}

func TestSelectEditor_EnvEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	editor := escalation.SelectEditor(func(_ string) string { return "" })
	g.Expect(editor).To(Equal("vim"))
}

func TestSelectEditor_EnvSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	editor := escalation.SelectEditor(func(key string) string {
		if key == "EDITOR" {
			return "nvim"
		}

		return ""
	})
	g.Expect(editor).To(Equal("nvim"))
}

// errorFS returns an error on WriteFile.
type errorWriteFS struct {
	readContent string
}

func (e *errorWriteFS) ReadFile(_ string) (string, error) {
	return e.readContent, nil
}

func (e *errorWriteFS) WriteFile(_ string, _ string) error {
	return errors.New("write failed")
}

// mockCmdExecutor records what it was called with.
type mockCmdExecutor struct {
	returnErr error
	called    []string
}

func (m *mockCmdExecutor) Run(name string, args ...string) error {
	m.called = append(m.called, name)
	return m.returnErr
}

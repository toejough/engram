package maintain_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/onsi/gomega"

	"engram/internal/maintain"
	"engram/internal/memory"
)

// writeMemoryTOML is a test helper that encodes a MemoryRecord to TOML and writes it to path.
func writeMemoryTOML(t *testing.T, path string, rec memory.MemoryRecord) {
	t.Helper()

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(rec)
	if err != nil {
		t.Fatalf("encoding memory record: %v", err)
	}

	err = os.WriteFile(path, buf.Bytes(), 0o644)
	if err != nil {
		t.Fatalf("writing memory file: %v", err)
	}
}

// TestTOMLHistoryRecorder_ReadRecord_OK verifies a valid TOML file is parsed correctly.
func TestTOMLHistoryRecorder_ReadRecord_OK(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "memory.toml")

	rec := memory.MemoryRecord{
		Title:         "test memory",
		Content:       "some content",
		SurfacedCount: 5,
		FollowedCount: 3,
	}

	writeMemoryTOML(t, path, rec)

	recorder := maintain.NewTOMLHistoryRecorder()

	result, err := recorder.ReadRecord(path)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(gomega.BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Title).To(gomega.Equal("test memory"))
	g.Expect(result.SurfacedCount).To(gomega.Equal(5))
	g.Expect(result.FollowedCount).To(gomega.Equal(3))
}

// TestTOMLHistoryRecorder_ReadRecord_ReadError verifies error when file cannot be read.
func TestTOMLHistoryRecorder_ReadRecord_ReadError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	recorder := maintain.NewTOMLHistoryRecorder(
		maintain.WithHistoryReadFile(func(_ string) ([]byte, error) {
			return nil, errors.New("permission denied")
		}),
	)

	result, err := recorder.ReadRecord("/fake/memory.toml")
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("reading record")))
	g.Expect(result).To(gomega.BeNil())
}

// TestTOMLHistoryRecorder_ReadRecord_DecodeError verifies error when TOML is invalid.
func TestTOMLHistoryRecorder_ReadRecord_DecodeError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	recorder := maintain.NewTOMLHistoryRecorder(
		maintain.WithHistoryReadFile(func(_ string) ([]byte, error) {
			return []byte("not = [valid toml"), nil
		}),
	)

	result, err := recorder.ReadRecord("/fake/memory.toml")
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("decoding record")))
	g.Expect(result).To(gomega.BeNil())
}

// TestTOMLHistoryRecorder_AppendAction_OK verifies an action is appended and persisted.
func TestTOMLHistoryRecorder_AppendAction_OK(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "memory.toml")

	initial := memory.MemoryRecord{
		Title:         "appendable memory",
		Content:       "content",
		SurfacedCount: 10,
		FollowedCount: 4,
	}

	writeMemoryTOML(t, path, initial)

	recorder := maintain.NewTOMLHistoryRecorder()

	action := memory.MaintenanceAction{
		Action:              "rewrite",
		AppliedAt:           "2026-03-27T10:00:00Z",
		EffectivenessBefore: 40.0,
		SurfacedCountBefore: 10,
		FeedbackCountBefore: 5,
		Measured:            false,
	}

	err := recorder.AppendAction(path, action)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Read back and verify.
	data, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	var loaded memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &loaded)
	g.Expect(decodeErr).NotTo(gomega.HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(loaded.Title).To(gomega.Equal("appendable memory"))
	g.Expect(loaded.MaintenanceHistory).To(gomega.HaveLen(1))
	g.Expect(loaded.MaintenanceHistory[0].Action).To(gomega.Equal("rewrite"))
	g.Expect(loaded.MaintenanceHistory[0].AppliedAt).To(gomega.Equal("2026-03-27T10:00:00Z"))
	g.Expect(loaded.MaintenanceHistory[0].EffectivenessBefore).To(gomega.BeNumerically("~", 40.0, 0.001))
	g.Expect(loaded.MaintenanceHistory[0].SurfacedCountBefore).To(gomega.Equal(10))
	g.Expect(loaded.MaintenanceHistory[0].FeedbackCountBefore).To(gomega.Equal(5))
	g.Expect(loaded.MaintenanceHistory[0].Measured).To(gomega.BeFalse())
}

// TestTOMLHistoryRecorder_AppendAction_MultipleActions verifies multiple actions accumulate.
func TestTOMLHistoryRecorder_AppendAction_MultipleActions(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "memory.toml")

	initial := memory.MemoryRecord{
		Title:   "multi-action memory",
		Content: "content",
	}

	writeMemoryTOML(t, path, initial)

	recorder := maintain.NewTOMLHistoryRecorder()

	firstAction := memory.MaintenanceAction{
		Action:    "rewrite",
		AppliedAt: "2026-03-20T10:00:00Z",
	}

	secondAction := memory.MaintenanceAction{
		Action:    "broaden_keywords",
		AppliedAt: "2026-03-27T10:00:00Z",
	}

	err := recorder.AppendAction(path, firstAction)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	err = recorder.AppendAction(path, secondAction)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	result, readErr := recorder.ReadRecord(path)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(result).NotTo(gomega.BeNil())

	if result == nil {
		return
	}

	g.Expect(result.MaintenanceHistory).To(gomega.HaveLen(2))
	g.Expect(result.MaintenanceHistory[0].Action).To(gomega.Equal("rewrite"))
	g.Expect(result.MaintenanceHistory[1].Action).To(gomega.Equal("broaden_keywords"))
}

// TestTOMLHistoryRecorder_AppendAction_ReadError verifies error when file cannot be read.
func TestTOMLHistoryRecorder_AppendAction_ReadError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	recorder := maintain.NewTOMLHistoryRecorder(
		maintain.WithHistoryReadFile(func(_ string) ([]byte, error) {
			return nil, errors.New("disk error")
		}),
	)

	action := memory.MaintenanceAction{Action: "rewrite", AppliedAt: "2026-03-27T10:00:00Z"}

	err := recorder.AppendAction("/fake/memory.toml", action)
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("reading record for history")))
}

// TestTOMLHistoryRecorder_AppendAction_DecodeError verifies error when TOML is invalid.
func TestTOMLHistoryRecorder_AppendAction_DecodeError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	recorder := maintain.NewTOMLHistoryRecorder(
		maintain.WithHistoryReadFile(func(_ string) ([]byte, error) {
			return []byte("not = [valid toml"), nil
		}),
	)

	action := memory.MaintenanceAction{Action: "rewrite", AppliedAt: "2026-03-27T10:00:00Z"}

	err := recorder.AppendAction("/fake/memory.toml", action)
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("decoding record for history")))
}

// TestTOMLHistoryRecorder_AppendAction_WriteError verifies error when file cannot be written.
func TestTOMLHistoryRecorder_AppendAction_WriteError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	recorder := maintain.NewTOMLHistoryRecorder(
		maintain.WithHistoryReadFile(func(_ string) ([]byte, error) {
			return []byte("title = \"test\"\n"), nil
		}),
		maintain.WithHistoryWriteFile(func(_ string, _ []byte, _ os.FileMode) error {
			return errors.New("no space left")
		}),
	)

	action := memory.MaintenanceAction{Action: "rewrite", AppliedAt: "2026-03-27T10:00:00Z"}

	err := recorder.AppendAction("/fake/memory.toml", action)
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("writing record with history")))
}

// TestTOMLHistoryRecorder_WithOptions verifies functional options are applied.
func TestTOMLHistoryRecorder_WithOptions(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	readCalled := false
	writeCalled := false

	recorder := maintain.NewTOMLHistoryRecorder(
		maintain.WithHistoryReadFile(func(_ string) ([]byte, error) {
			readCalled = true

			return []byte("title = \"test\"\n"), nil
		}),
		maintain.WithHistoryWriteFile(func(_ string, _ []byte, _ os.FileMode) error {
			writeCalled = true

			return nil
		}),
	)

	action := memory.MaintenanceAction{Action: "rewrite", AppliedAt: "2026-03-27T10:00:00Z"}

	err := recorder.AppendAction("/fake/memory.toml", action)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(readCalled).To(gomega.BeTrue())
	g.Expect(writeCalled).To(gomega.BeTrue())
}

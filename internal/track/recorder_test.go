package track_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
	"engram/internal/track"
)

// TestEmptyFilePathSkipped verifies that memories with empty FilePath are skipped.
func TestEmptyFilePathSkipped(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	readCalled := false

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			readCalled = true

			return nil, errors.New("should not be called")
		}),
	)

	memories := []*memory.Stored{
		{FilePath: ""},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(readCalled).To(BeFalse())
}

// TestPartialFailureContinues verifies first memory failure doesn't stop second.
func TestPartialFailureContinues(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	capture := &writeCapture{}

	sbiaTOML := `situation = "test situation"
action = "test action"
`

	readFile := func(path string) ([]byte, error) {
		if path == "/nonexistent/bad.toml" {
			return nil, errors.New("file not found")
		}

		return []byte(sbiaTOML), nil
	}

	recorder := track.NewRecorder(
		track.WithReadFile(readFile),
		track.WithWriter(tomlwriter.New(
			tomlwriter.WithCreateTemp(capture.createTemp(t)),
			tomlwriter.WithRename(func(_, _ string) error { return nil }),
			tomlwriter.WithRemove(func(_ string) error { return nil }),
		)),
	)

	memories := []*memory.Stored{
		{FilePath: "/nonexistent/bad.toml"},
		{FilePath: "/fake/valid.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")

	g.Expect(err).To(HaveOccurred())

	record := capture.decodeTOML(g)
	g.Expect(record.Situation).To(Equal("test situation"))
}

// TestRecordSurfacing_PreservesSBIAFields verifies all SBIA fields are preserved.
func TestRecordSurfacing_PreservesSBIAFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	capture := &writeCapture{}

	sbiaTOML := `situation = "when running tests"
behavior = "use go test directly"
impact = "misses coverage"
action = "use targ test"
project_scoped = true
project_slug = "engram"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-06-01T00:00:00Z"
surfaced_count = 5
followed_count = 3
not_followed_count = 1
irrelevant_count = 2
`

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(sbiaTOML), nil
		}),
		track.WithWriter(tomlwriter.New(
			tomlwriter.WithCreateTemp(capture.createTemp(t)),
			tomlwriter.WithRename(func(_, _ string) error { return nil }),
			tomlwriter.WithRemove(func(_ string) error { return nil }),
		)),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	record := capture.decodeTOML(g)

	g.Expect(record.Situation).To(Equal("when running tests"))
	g.Expect(record.Behavior).To(Equal("use go test directly"))
	g.Expect(record.Impact).To(Equal("misses coverage"))
	g.Expect(record.Action).To(Equal("use targ test"))
	g.Expect(record.ProjectScoped).To(BeTrue())
	g.Expect(record.ProjectSlug).To(Equal("engram"))
	g.Expect(record.SurfacedCount).To(Equal(5))
	g.Expect(record.FollowedCount).To(Equal(3))
	g.Expect(record.NotFollowedCount).To(Equal(1))
	g.Expect(record.IrrelevantCount).To(Equal(2))
}

// TestWriteAtomicCreateTempFailure verifies error path when createTemp fails.
func TestWriteAtomicCreateTempFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	sbiaTOML := `situation = "test"
action = "test"
`

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(sbiaTOML), nil
		}),
		track.WithWriter(tomlwriter.New(
			tomlwriter.WithCreateTemp(func(_, _ string) (*os.File, error) {
				return nil, errors.New("disk full")
			}),
		)),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("disk full"))
}

// writeCapture tracks the temp file path so its content can be read back.
type writeCapture struct {
	tmpPath string
}

func (w *writeCapture) createTemp(
	t *testing.T,
) func(string, string) (*os.File, error) {
	t.Helper()

	return func(_, pattern string) (*os.File, error) {
		f, err := os.CreateTemp(t.TempDir(), pattern)
		if err != nil {
			return nil, err
		}

		w.tmpPath = f.Name()

		return f, nil
	}
}

func (w *writeCapture) decodeTOML(g Gomega) memory.MemoryRecord {
	g.Expect(w.tmpPath).NotTo(BeEmpty(), "no temp file was written")

	data, err := os.ReadFile(w.tmpPath)
	g.Expect(err).NotTo(HaveOccurred())

	var record memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &record)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	return record
}

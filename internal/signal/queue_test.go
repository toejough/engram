package signal_test

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"engram/internal/signal"
)

func TestQueueStore_AppendCreateTempError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return nil, os.ErrNotExist
		}),
		signal.WithQueueCreateTemp(func(_, _ string) (*os.File, error) {
			return nil, errors.New("no space")
		}),
	)

	err := store.Append([]signal.Signal{{
		Type: signal.TypeMaintain, SourceID: "x.toml",
		SignalKind: signal.KindNoiseRemoval, DetectedAt: fixedTime,
	}}, "/tmp/queue.jsonl")
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("creating temp file")))
}

func TestQueueStore_AppendCreatesNew(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var written string

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return nil, os.ErrNotExist
		}),
		signal.WithQueueCreateTemp(func(_, _ string) (*os.File, error) {
			return os.CreateTemp(t.TempDir(), "test-*.jsonl")
		}),
		signal.WithQueueRename(func(old, _ string) error {
			data, readErr := os.ReadFile(old)
			if readErr != nil {
				return readErr
			}

			written = string(data)

			return nil
		}),
	)

	sigs := []signal.Signal{{
		Type:       signal.TypeMaintain,
		SourceID:   "new.toml",
		SignalKind: signal.KindNoiseRemoval,
		Summary:    "new signal",
		DetectedAt: fixedTime,
	}}

	err := store.Append(sigs, "/tmp/queue.jsonl")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(written).To(gomega.ContainSubstring("new.toml"))
}

func TestQueueStore_AppendEmptyNoop(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	store := signal.NewQueueStore()

	err := store.Append(nil, "/tmp/queue.jsonl")
	g.Expect(err).NotTo(gomega.HaveOccurred())
}

func TestQueueStore_AppendReadError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return nil, errors.New("permission denied")
		}),
	)

	err := store.Append([]signal.Signal{{
		Type: signal.TypeMaintain, SourceID: "x.toml",
		SignalKind: signal.KindNoiseRemoval, DetectedAt: fixedTime,
	}}, "/tmp/queue.jsonl")
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("reading signal queue")))
}

func TestQueueStore_AppendRenameError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var removedTmp bool

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return nil, os.ErrNotExist
		}),
		signal.WithQueueCreateTemp(func(_, _ string) (*os.File, error) {
			return os.CreateTemp(t.TempDir(), "test-*.jsonl")
		}),
		signal.WithQueueRename(func(_, _ string) error {
			return errors.New("rename failed")
		}),
		signal.WithQueueRemove(func(_ string) error {
			removedTmp = true

			return nil
		}),
	)

	err := store.Append([]signal.Signal{{
		Type: signal.TypeMaintain, SourceID: "x.toml",
		SignalKind: signal.KindNoiseRemoval, DetectedAt: fixedTime,
	}}, "/tmp/queue.jsonl")
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("renaming temp file")))

	if err != nil {
		g.Expect(removedTmp).To(gomega.BeTrue())
	}
}

func TestQueueStore_AppendToExisting(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	existing := `{"type":"maintain","source_id":"old.toml","signal":"noise_removal"}` + "\n"

	var written string

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return []byte(existing), nil
		}),
		signal.WithQueueCreateTemp(func(_, _ string) (*os.File, error) {
			return os.CreateTemp(t.TempDir(), "test-*.jsonl")
		}),
		signal.WithQueueRename(func(old, _ string) error {
			data, readErr := os.ReadFile(old)
			if readErr != nil {
				return readErr
			}

			written = string(data)

			return nil
		}),
	)

	err := store.Append([]signal.Signal{{
		Type: signal.TypeMaintain, SourceID: "new.toml",
		SignalKind: signal.KindLeechRewrite, DetectedAt: fixedTime,
	}}, "/tmp/queue.jsonl")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(written).To(gomega.ContainSubstring("old.toml"))
	g.Expect(written).To(gomega.ContainSubstring("new.toml"))
}

func TestQueueStore_AppendWriteError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return nil, os.ErrNotExist
		}),
		signal.WithQueueCreateTemp(func(_, _ string) (*os.File, error) {
			f, err := os.CreateTemp(t.TempDir(), "test-*.jsonl")
			if err != nil {
				return nil, err
			}
			// Close immediately so WriteString fails.
			_ = f.Close()

			return f, nil
		}),
		signal.WithQueueRemove(func(_ string) error { return nil }),
	)

	err := store.Append([]signal.Signal{{
		Type: signal.TypeMaintain, SourceID: "x.toml",
		SignalKind: signal.KindNoiseRemoval, DetectedAt: fixedTime,
	}}, "/tmp/queue.jsonl")
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("writing signal queue")))
}

func TestQueueStore_ClearBySourceID(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	sig1 := signal.Signal{
		Type: signal.TypeMaintain, SourceID: "keep.toml",
		SignalKind: signal.KindNoiseRemoval, DetectedAt: fixedTime,
	}
	sig2 := signal.Signal{
		Type: signal.TypeMaintain, SourceID: "remove.toml",
		SignalKind: signal.KindLeechRewrite, DetectedAt: fixedTime,
	}

	//nolint:errchkjson // test data
	l1, _ := json.Marshal(sig1)
	//nolint:errchkjson // test data
	l2, _ := json.Marshal(sig2)
	fileData := string(l1) + "\n" + string(l2) + "\n"

	var written string

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return []byte(fileData), nil
		}),
		signal.WithQueueCreateTemp(func(_, _ string) (*os.File, error) {
			return os.CreateTemp(t.TempDir(), "test-*.jsonl")
		}),
		signal.WithQueueRename(func(old, _ string) error {
			data, readErr := os.ReadFile(old)
			if readErr != nil {
				return readErr
			}

			written = string(data)

			return nil
		}),
	)

	err := store.ClearBySourceID("/tmp/queue.jsonl", "remove.toml")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(written).To(gomega.ContainSubstring("keep.toml"))
	g.Expect(written).NotTo(gomega.ContainSubstring("remove.toml"))
}

func TestQueueStore_PruneDedup(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	sig1 := signal.Signal{
		Type: signal.TypeMaintain, SourceID: "dup.toml",
		SignalKind: signal.KindNoiseRemoval, Summary: "first", DetectedAt: fixedTime,
	}
	sig2 := signal.Signal{
		Type: signal.TypeMaintain, SourceID: "dup.toml",
		SignalKind: signal.KindNoiseRemoval, Summary: "second", DetectedAt: fixedTime,
	}

	//nolint:errchkjson // test data
	l1, _ := json.Marshal(sig1)
	//nolint:errchkjson // test data
	l2, _ := json.Marshal(sig2)
	fileData := string(l1) + "\n" + string(l2) + "\n"

	var written string

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return []byte(fileData), nil
		}),
		signal.WithQueueCreateTemp(func(_, _ string) (*os.File, error) {
			return os.CreateTemp(t.TempDir(), "test-*.jsonl")
		}),
		signal.WithQueueRename(func(old, _ string) error {
			data, readErr := os.ReadFile(old)
			if readErr != nil {
				return readErr
			}

			written = string(data)

			return nil
		}),
		signal.WithQueueNow(func() time.Time { return fixedTime }),
	)

	err := store.Prune("/tmp/queue.jsonl", func(_ string) bool { return true })
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(written), "\n")
	g.Expect(lines).To(gomega.HaveLen(1))
}

func TestQueueStore_PruneDeletedSources(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	sig := signal.Signal{
		Type: signal.TypeMaintain, SourceID: "deleted.toml",
		SignalKind: signal.KindNoiseRemoval, DetectedAt: fixedTime,
	}
	//nolint:errchkjson // test data
	line, _ := json.Marshal(sig)

	var written string

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return append(line, '\n'), nil
		}),
		signal.WithQueueCreateTemp(func(_, _ string) (*os.File, error) {
			return os.CreateTemp(t.TempDir(), "test-*.jsonl")
		}),
		signal.WithQueueRename(func(old, _ string) error {
			data, readErr := os.ReadFile(old)
			if readErr != nil {
				return readErr
			}

			written = string(data)

			return nil
		}),
		signal.WithQueueNow(func() time.Time { return fixedTime }),
	)

	err := store.Prune("/tmp/queue.jsonl", func(_ string) bool { return false })
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(strings.TrimSpace(written)).To(gomega.BeEmpty())
}

func TestQueueStore_PruneOldEntries(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	now := fixedTime
	oldTime := now.Add(-31 * 24 * time.Hour)

	oldSig := signal.Signal{
		Type: signal.TypeMaintain, SourceID: "old.toml",
		SignalKind: signal.KindNoiseRemoval, DetectedAt: oldTime,
	}
	newSig := signal.Signal{
		Type: signal.TypeMaintain, SourceID: "new.toml",
		SignalKind: signal.KindLeechRewrite, DetectedAt: now,
	}

	//nolint:errchkjson // test data
	oldLine, _ := json.Marshal(oldSig)
	//nolint:errchkjson // test data
	newLine, _ := json.Marshal(newSig)
	fileData := string(oldLine) + "\n" + string(newLine) + "\n"

	var written string

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return []byte(fileData), nil
		}),
		signal.WithQueueCreateTemp(func(_, _ string) (*os.File, error) {
			return os.CreateTemp(t.TempDir(), "test-*.jsonl")
		}),
		signal.WithQueueRename(func(old, _ string) error {
			data, readErr := os.ReadFile(old)
			if readErr != nil {
				return readErr
			}

			written = string(data)

			return nil
		}),
		signal.WithQueueNow(func() time.Time { return now }),
	)

	err := store.Prune("/tmp/queue.jsonl", func(_ string) bool { return true })
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(written).To(gomega.ContainSubstring("new.toml"))
	g.Expect(written).NotTo(gomega.ContainSubstring("old.toml"))
}

func TestQueueStore_ReadExisting(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	sig := signal.Signal{
		Type:       signal.TypeMaintain,
		SourceID:   "test.toml",
		SignalKind: signal.KindNoiseRemoval,
		Summary:    "test",
		DetectedAt: fixedTime,
	}
	//nolint:errchkjson // test data
	line, _ := json.Marshal(sig)

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return append(line, '\n'), nil
		}),
	)

	signals, err := store.Read("/tmp/queue.jsonl")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(signals).To(gomega.HaveLen(1))
	g.Expect(signals[0].SourceID).To(gomega.Equal("test.toml"))
}

func TestQueueStore_ReadMissing(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return nil, os.ErrNotExist
		}),
	)

	signals, err := store.Read("/tmp/queue.jsonl")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(signals).To(gomega.BeEmpty())
}

func TestQueueStore_ReadSkipsMalformed(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	store := signal.NewQueueStore(
		signal.WithQueueReadFile(func(_ string) ([]byte, error) {
			return []byte("not json\n{\"type\":\"maintain\"}\n"), nil
		}),
	)

	signals, err := store.Read("/tmp/queue.jsonl")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(signals).To(gomega.HaveLen(1))
}

func TestWithQueueRemove(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var removeCalled bool

	store := signal.NewQueueStore(
		signal.WithQueueRemove(func(_ string) error {
			removeCalled = true

			return nil
		}),
	)

	g.Expect(store).NotTo(gomega.BeNil())

	_ = removeCalled
}

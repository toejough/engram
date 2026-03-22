package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"engram/internal/cli"
	"engram/internal/toolgate"

	. "github.com/onsi/gomega"
)

func TestFileCounterStore_Load_CorruptJSON(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "tool-frecency.json")

	err := os.WriteFile(path, []byte(`{corrupt`), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	store := cli.ExportNewFileCounterStore(dir)

	counters, loadErr := store.Load()
	g.Expect(loadErr).NotTo(HaveOccurred())

	if loadErr != nil {
		return
	}

	g.Expect(counters).To(BeEmpty())
}

func TestFileCounterStore_Load_FileNotExist(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := cli.ExportNewFileCounterStore(t.TempDir())

	counters, err := store.Load()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(counters).To(BeEmpty())
}

func TestFileCounterStore_Load_ReadError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Point at a directory instead of a file to trigger a read error.
	dir := t.TempDir()
	subdir := filepath.Join(dir, "tool-frecency.json")

	mkErr := os.Mkdir(subdir, 0o755)
	g.Expect(mkErr).NotTo(HaveOccurred())

	store := cli.ExportNewFileCounterStore(dir)

	_, loadErr := store.Load()
	g.Expect(loadErr).To(HaveOccurred())
}

func TestFileCounterStore_Load_ValidJSON(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "tool-frecency.json")

	err := os.WriteFile(path, []byte(`{"grep":{"count":5,"last":"2026-03-21T00:00:00Z"}}`), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	store := cli.ExportNewFileCounterStore(dir)

	counters, loadErr := store.Load()
	g.Expect(loadErr).NotTo(HaveOccurred())

	if loadErr != nil {
		return
	}

	g.Expect(counters).To(HaveKey("grep"))
	g.Expect(counters["grep"].Count).To(Equal(5))
}

func TestFileCounterStore_SaveAndLoad(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := cli.ExportNewFileCounterStore(t.TempDir())

	counters := map[string]toolgate.CounterEntry{
		"go test": {Count: 3},
	}

	err := store.Save(counters)
	g.Expect(err).NotTo(HaveOccurred())

	loaded, loadErr := store.Load()
	g.Expect(loadErr).NotTo(HaveOccurred())

	if loadErr != nil {
		return
	}

	g.Expect(loaded).To(HaveKey("go test"))
	g.Expect(loaded["go test"].Count).To(Equal(3))
}

func TestFileCounterStore_Save_WriteError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Use a non-existent directory to trigger write error.
	store := cli.ExportNewFileCounterStore("/nonexistent/dir")

	err := store.Save(map[string]toolgate.CounterEntry{"x": {Count: 1}})
	g.Expect(err).To(HaveOccurred())
}

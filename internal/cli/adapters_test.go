package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
	"engram/internal/surface"
)

func TestHaikuCallerAdapter_Call(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedModel string

	adapter := cli.ExportNewHaikuCallerAdapter(
		func(_ context.Context, model, _, _ string) (string, error) {
			capturedModel = model
			return "response text", nil
		},
	)

	result, err := adapter.Call(context.Background(), "system prompt", "user prompt")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(Equal("response text"))
	g.Expect(capturedModel).To(Equal("claude-haiku-4-5-20251001"))
}

func TestHaikuCallerAdapter_CallError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	adapter := cli.ExportNewHaikuCallerAdapter(
		func(_ context.Context, _, _, _ string) (string, error) {
			return "", errors.New("api error")
		},
	)

	_, err := adapter.Call(context.Background(), "sys", "usr")
	g.Expect(err).To(MatchError("api error"))
}

func TestOsDirLister_ListJSONL(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "session1.jsonl"), []byte("{}"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "session2.jsonl"), []byte("{}"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not jsonl"), 0o644)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(dir, "subdir"), 0o755)).To(Succeed())

	lister := cli.ExportNewOsDirLister()
	entries, err := lister.ListJSONL(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(2))
}

func TestOsDirLister_ListJSONL_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := cli.ExportNewOsDirLister()
	_, err := lister.ListJSONL("/nonexistent/path")
	g.Expect(err).To(HaveOccurred())
}

func TestOsFileReader_Read(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "test.txt")
	g.Expect(os.WriteFile(path, []byte("hello world"), 0o644)).To(Succeed())

	reader := cli.ExportNewOsFileReader()
	data, err := reader.Read(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(Equal("hello world"))
}

func TestOsFileReader_ReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reader := cli.ExportNewOsFileReader()
	_, err := reader.Read("/nonexistent/file.txt")
	g.Expect(err).To(HaveOccurred())
}

func TestSurfaceRunnerAdapter_Run(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// The adapter wraps a real Surfacer. With no memories, it returns empty.
	lister := memory.NewLister()
	surfacer := surface.New(lister)
	adapter := cli.ExportNewSurfaceRunnerAdapter(surfacer)

	var buf bytes.Buffer

	err := adapter.Run(context.Background(), &buf, cli.SurfaceRunnerOptions{
		Mode:    surface.ModePrompt,
		DataDir: dataDir,
		Message: "test query",
	})
	g.Expect(err).NotTo(HaveOccurred())
}

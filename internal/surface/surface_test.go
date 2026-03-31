package surface_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// TestPromptMode_JSONFormat verifies JSON output.
func TestPromptMode_JSONFormat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "when committing code",
			Behavior:  "manual commit",
			Action:    "use /commit skill",
			FilePath:  "commit-conventions.toml",
		},
		{
			Situation: "when testing",
			Behavior:  "run go test",
			Action:    "use targ test",
			FilePath:  "testing.toml",
		},
		{
			Situation: "when linting",
			Behavior:  "skip linting",
			Action:    "run linter",
			FilePath:  "linting.toml",
		},
		{
			Situation: "when deploying",
			Behavior:  "skip review",
			Action:    "get review",
			FilePath:  "docker.toml",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "I want to commit this change",
		Format:  surface.FormatJSON,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result surface.Result

	decodeErr := json.Unmarshal(buf.Bytes(), &result)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(result.Summary).To(ContainSubstring("commit-conventions"))
	g.Expect(result.Context).To(ContainSubstring("commit-conventions"))
}

// TestPromptMode_KeywordMatch_SurfacesRelevant verifies BM25 keyword matching.
func TestPromptMode_KeywordMatch_SurfacesRelevant(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "when committing code",
			Behavior:  "manual git commit",
			Action:    "use /commit skill",
			FilePath:  "commit-conventions.toml",
		},
		{
			Situation: "when building",
			Behavior:  "run go build directly",
			Action:    "use targ build",
			FilePath:  "build-tools.toml",
		},
		{
			Situation: "when testing",
			Behavior:  "run go test directly",
			Action:    "use targ test",
			FilePath:  "testing.toml",
		},
		{
			Situation: "when linting",
			Behavior:  "skip lint checks",
			Action:    "run targ check-full",
			FilePath:  "linting.toml",
		},
		{
			Situation: "when deploying",
			Behavior:  "deploy without review",
			Action:    "require code review",
			FilePath:  "docker.toml",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "I want to commit this change",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("[engram] Memories"))
	g.Expect(output).To(ContainSubstring("commit-conventions"))
}

// TestPromptMode_NoMatch_ProducesEmpty verifies empty output on no match.
func TestPromptMode_NoMatch_ProducesEmpty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "when committing code",
			Behavior:  "manual commit",
			Action:    "use /commit skill",
			FilePath:  "commit-conventions.toml",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "hello world banana",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(BeEmpty())
}

// TestSurfacingLogger verifies surfacing log callback.
func TestSurfacingLogger(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{Situation: "alpha context", Behavior: "alpha bad", Action: "alpha good",
			FilePath: "mem/alpha.toml"},
		{Situation: "beta context", Behavior: "beta bad", Action: "beta good",
			FilePath: "mem/beta.toml"},
		{Situation: "gamma context", Behavior: "gamma bad", Action: "gamma good",
			FilePath: "mem/gamma.toml"},
		{Situation: "delta context", Behavior: "delta bad", Action: "delta good",
			FilePath: "mem/delta.toml"},
	}

	retriever := &fakeRetriever{memories: memories}
	logger := &fakeSurfacingLogger{}

	surfacer := surface.New(retriever, surface.WithSurfacingLogger(logger))

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/data",
		Message: "alpha beta",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(logger.calls).To(HaveLen(2))
	g.Expect(logger.calls[0].memoryPath).To(Equal("mem/alpha.toml"))
	g.Expect(logger.calls[0].mode).To(Equal(surface.ModePrompt))
	g.Expect(logger.calls[1].memoryPath).To(Equal("mem/beta.toml"))
	g.Expect(logger.calls[1].mode).To(Equal(surface.ModePrompt))
}

// TestUnknownMode_ReturnsError verifies unknown mode handling.
func TestUnknownMode_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    "unknown-mode",
		DataDir: "/tmp/data",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(surface.ErrUnknownMode))
}

// fakeRetriever is a test double for surface.MemoryRetriever.
type fakeRetriever struct {
	memories []*memory.Stored
	err      error
}

func (f *fakeRetriever) ListMemories(_ context.Context, _ string) ([]*memory.Stored, error) {
	return f.memories, f.err
}

// fakeSurfacingLogger is a test double for surface.SurfacingEventLogger.
type fakeSurfacingLogger struct {
	calls     []surfacingLogCall
	returnErr error
}

func (f *fakeSurfacingLogger) LogSurfacing(memoryPath, mode string, _ time.Time) error {
	f.calls = append(f.calls, surfacingLogCall{memoryPath: memoryPath, mode: mode})
	return f.returnErr
}

type surfacingLogCall struct {
	memoryPath string
	mode       string
}

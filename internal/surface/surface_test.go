package surface_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

func TestFilenameSlug_NoExtension_ReturnsBasename(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	g.Expect(surface.ExportFilenameSlug("mem/no-extension")).To(Equal("no-extension"))
}

func TestFilenameSlug_StripsDirectoryAndExtension(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	g.Expect(surface.ExportFilenameSlug("mem/commit-safety.toml")).To(Equal("commit-safety"))
	g.Expect(surface.ExportFilenameSlug("/abs/path/build-tools.toml")).To(Equal("build-tools"))
	g.Expect(surface.ExportFilenameSlug("bare-name.toml")).To(Equal("bare-name"))
}

func TestMatchPromptMemories_EmptyMemories_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	matches := surface.ExportMatchPromptMemories("some query", []*memory.Stored{}, 5)

	g.Expect(matches).To(BeEmpty())
}

func TestMatchPromptMemories_ReturnsRankedMatches(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mems := []*memory.Stored{
		{
			FilePath:  "mem/commit.toml",
			Situation: "when committing code",
			Content:   memory.ContentFields{Action: "use /commit skill"},
		},
		{
			FilePath:  "mem/build.toml",
			Situation: "when building software",
			Content:   memory.ContentFields{Action: "use targ build"},
		},
		{
			FilePath:  "mem/deploy.toml",
			Situation: "when deploying application",
			Content:   memory.ContentFields{Action: "review first"},
		},
	}

	matches := surface.ExportMatchPromptMemories("I want to commit some code", mems, 5)

	g.Expect(matches).NotTo(BeEmpty())
	// The commit memory should score highest for a "commit" query.
	g.Expect(matches[0].Mem.FilePath).To(Equal("mem/commit.toml"))
}

// TestPromptMode_BM25Threshold_FiltersLowScores verifies that memories with scores below the
// threshold are filtered out.
func TestPromptMode_BM25Threshold_FiltersLowScores(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "when committing code",
			Content: memory.ContentFields{
				Behavior: "manual git commit",
				Action:   "use /commit skill",
			},
			FilePath: "commit-conventions.toml",
		},
		{
			Situation: "when building",
			Content:   memory.ContentFields{Behavior: "run go build", Action: "use targ build"},
			FilePath:  "build-tools.toml",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever, surface.WithSurfaceConfig(surface.SurfaceConfig{
		BM25Threshold:       999.0,
		CandidateCountMax:   8,
		IrrelevanceHalfLife: 5,
	}))

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "I want to commit this change",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(BeEmpty())
}

// TestPromptMode_JSONFormat verifies JSON output.
func TestPromptMode_JSONFormat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "when committing code",
			Content:   memory.ContentFields{Behavior: "manual commit", Action: "use /commit skill"},
			FilePath:  "commit-conventions.toml",
		},
		{
			Situation: "when testing",
			Content:   memory.ContentFields{Behavior: "run go test", Action: "use targ test"},
			FilePath:  "testing.toml",
		},
		{
			Situation: "when linting",
			Content:   memory.ContentFields{Behavior: "skip linting", Action: "run linter"},
			FilePath:  "linting.toml",
		},
		{
			Situation: "when deploying",
			Content:   memory.ContentFields{Behavior: "skip review", Action: "get review"},
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
			Content: memory.ContentFields{
				Behavior: "manual git commit",
				Action:   "use /commit skill",
			},
			FilePath: "commit-conventions.toml",
		},
		{
			Situation: "when building",
			Content: memory.ContentFields{
				Behavior: "run go build directly",
				Action:   "use targ build",
			},
			FilePath: "build-tools.toml",
		},
		{
			Situation: "when testing",
			Content: memory.ContentFields{
				Behavior: "run go test directly",
				Action:   "use targ test",
			},
			FilePath: "testing.toml",
		},
		{
			Situation: "when linting",
			Content: memory.ContentFields{
				Behavior: "skip lint checks",
				Action:   "run targ check-full",
			},
			FilePath: "linting.toml",
		},
		{
			Situation: "when deploying",
			Content: memory.ContentFields{
				Behavior: "deploy without review",
				Action:   "require code review",
			},
			FilePath: "docker.toml",
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
			Content:   memory.ContentFields{Behavior: "manual commit", Action: "use /commit skill"},
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

func TestPromptMode_SBIADisplayFormat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "when committing code",
			Content: memory.ContentFields{
				Behavior: "manual git commit bypass",
				Impact:   "skips required commit workflow",
				Action:   "use /commit skill",
			},
			FilePath: "commit-safety.toml",
		},
		{
			Situation: "when building software",
			Content: memory.ContentFields{
				Behavior: "run go build directly",
				Impact:   "misses build checks",
				Action:   "use targ build",
			},
			FilePath: "build-tools.toml",
		},
		{
			Situation: "when testing code",
			Content: memory.ContentFields{
				Behavior: "run go test directly",
				Impact:   "misses coverage checks",
				Action:   "use targ test",
			},
			FilePath: "testing-tools.toml",
		},
		{
			Situation: "when linting",
			Content: memory.ContentFields{
				Behavior: "skip lint checks",
				Impact:   "lets bugs through",
				Action:   "run targ check-full",
			},
			FilePath: "linting-tools.toml",
		},
		{
			Situation: "when deploying",
			Content: memory.ContentFields{
				Behavior: "deploy without review",
				Impact:   "introduces unreviewed changes",
				Action:   "require code review",
			},
			FilePath: "deploy.toml",
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
	g.Expect(output).To(ContainSubstring("Situation: when committing code"))
	g.Expect(output).To(ContainSubstring("Behavior to avoid: manual git commit bypass"))
	g.Expect(output).To(ContainSubstring("Impact if ignored: skips required commit workflow"))
	g.Expect(output).To(ContainSubstring("Action: use /commit skill"))
	g.Expect(output).To(ContainSubstring("1. commit-safety"))
}

// TestRecordInstrumentation_LoggerError verifies error logging in surfacing logger does not halt.
func TestRecordInstrumentation_LoggerError(t *testing.T) {
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
	logger := &fakeSurfacingLogger{returnErr: errors.New("log write failed")}

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
}

// TestRecordInstrumentation_RecorderError verifies error logging in surfacing recorder does not halt.
func TestRecordInstrumentation_RecorderError(t *testing.T) {
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
	surfacer := surface.New(retriever, surface.WithSurfacingRecorder(func(_ string) error {
		return errors.New("record failed")
	}))

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/data",
		Message: "alpha beta",
	})

	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_TranscriptWindowSuppression(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "commit context",
			Content:   memory.ContentFields{Behavior: "bad commit", Action: "use /commit skill"},
			FilePath:  "mem/commit.toml",
		},
		{
			Situation: "build context",
			Content:   memory.ContentFields{Behavior: "bad build", Action: "use targ build"},
			FilePath:  "mem/build.toml",
		},
		{
			Situation: "test context",
			Content:   memory.ContentFields{Behavior: "bad test", Action: "use targ test"},
			FilePath:  "mem/test.toml",
		},
		{
			Situation: "lint context",
			Content:   memory.ContentFields{Behavior: "bad lint", Action: "run linter"},
			FilePath:  "mem/lint.toml",
		},
	}

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:             surface.ModePrompt,
		DataDir:          "/data",
		Message:          "I want to commit and build",
		TranscriptWindow: "I already use /commit skill for my commits",
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).NotTo(ContainSubstring("commit.toml"))
}

func TestRun_WithRecordSurfacing(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "alpha context",
			Content:   memory.ContentFields{Behavior: "alpha bad", Action: "alpha good"},
			FilePath:  "mem/alpha.toml",
		},
		{
			Situation: "beta context",
			Content:   memory.ContentFields{Behavior: "beta bad", Action: "beta good"},
			FilePath:  "mem/beta.toml",
		},
		{
			Situation: "gamma context",
			Content:   memory.ContentFields{Behavior: "gamma bad", Action: "gamma good"},
			FilePath:  "mem/gamma.toml",
		},
		{
			Situation: "delta context",
			Content:   memory.ContentFields{Behavior: "delta bad", Action: "delta good"},
			FilePath:  "mem/delta.toml",
		},
	}

	var recorded []string

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever, surface.WithSurfacingRecorder(func(path string) error {
		recorded = append(recorded, path)
		return nil
	}))

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

	g.Expect(recorded).To(HaveLen(2))
}

func TestSortPromptMatchesByScore_ProjectScopedPenalty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// A project-scoped memory in a different project gets a penalty (GenFactor < 1).
	// A global memory with lower raw score may outrank it after penalty.
	global := &memory.Stored{FilePath: "mem/global.toml", ProjectScoped: false}
	scoped := &memory.Stored{
		FilePath:      "mem/scoped.toml",
		ProjectScoped: true,
		ProjectSlug:   "other-project",
	}

	matches := []surface.PromptMatch{
		{Mem: scoped, BM25Score: 5.0},
		{Mem: global, BM25Score: 3.0},
	}

	surface.ExportSortPromptMatchesByScore(matches, "current-project")

	// global should outrank scoped due to cross-project penalty.
	g.Expect(matches[0].Mem.FilePath).To(Equal("mem/global.toml"))
}

func TestSortPromptMatchesByScore_SortsByScoreDescending(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	alpha := &memory.Stored{FilePath: "mem/alpha.toml"}
	beta := &memory.Stored{FilePath: "mem/beta.toml"}
	gamma := &memory.Stored{FilePath: "mem/gamma.toml"}

	matches := []surface.PromptMatch{
		{Mem: alpha, BM25Score: 1.5},
		{Mem: gamma, BM25Score: 3.2},
		{Mem: beta, BM25Score: 2.1},
	}

	surface.ExportSortPromptMatchesByScore(matches, "")

	g.Expect(matches[0].Mem.FilePath).To(Equal("mem/gamma.toml"))
	g.Expect(matches[1].Mem.FilePath).To(Equal("mem/beta.toml"))
	g.Expect(matches[2].Mem.FilePath).To(Equal("mem/alpha.toml"))
}

func TestSuppressByTranscript_EmptyAction(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{Content: memory.ContentFields{}, FilePath: "empty.toml"},
	}
	filtered, events := surface.ExportSuppressByTranscript(
		candidates, "some transcript",
	)
	g.Expect(filtered).To(HaveLen(1))
	g.Expect(events).To(BeEmpty())
}

func TestSuppressByTranscript_EmptyCandidates(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	filtered, events := surface.ExportSuppressByTranscript(
		nil, "some transcript",
	)
	g.Expect(filtered).To(BeEmpty())
	g.Expect(events).To(BeEmpty())
}

func TestSuppressByTranscript_MatchesAction(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{Content: memory.ContentFields{Action: "use targ test"}, FilePath: "targ.toml"},
		{Content: memory.ContentFields{Action: "run linter"}, FilePath: "lint.toml"},
	}
	filtered, events := surface.ExportSuppressByTranscript(
		candidates, "I already use targ test in my workflow",
	)
	g.Expect(filtered).To(HaveLen(1))

	if len(filtered) == 0 || len(events) == 0 {
		return
	}

	g.Expect(filtered[0].FilePath).To(Equal("lint.toml"))
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0].Reason).To(Equal(surface.SuppressionReasonTranscript))
}

func TestSuppressByTranscript_NoMatch(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{Content: memory.ContentFields{Action: "use targ test"}, FilePath: "targ.toml"},
	}
	filtered, events := surface.ExportSuppressByTranscript(
		candidates, "unrelated transcript text",
	)
	g.Expect(filtered).To(HaveLen(1))
	g.Expect(events).To(BeEmpty())
}

// TestSurfacingLogger verifies surfacing log callback.
func TestSurfacingLogger(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "alpha context",
			Content:   memory.ContentFields{Behavior: "alpha bad", Action: "alpha good"},
			FilePath:  "mem/alpha.toml",
		},
		{
			Situation: "beta context",
			Content:   memory.ContentFields{Behavior: "beta bad", Action: "beta good"},
			FilePath:  "mem/beta.toml",
		},
		{
			Situation: "gamma context",
			Content:   memory.ContentFields{Behavior: "gamma bad", Action: "gamma good"},
			FilePath:  "mem/gamma.toml",
		},
		{
			Situation: "delta context",
			Content:   memory.ContentFields{Behavior: "delta bad", Action: "delta good"},
			FilePath:  "mem/delta.toml",
		},
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

func TestWithInvocationTokenLogger_LogsTokens(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "alpha context",
			Content:   memory.ContentFields{Behavior: "alpha bad", Action: "alpha good"},
			FilePath:  "mem/alpha.toml",
		},
		{
			Situation: "beta context",
			Content:   memory.ContentFields{Behavior: "beta bad", Action: "beta good"},
			FilePath:  "mem/beta.toml",
		},
		{
			Situation: "gamma context",
			Content:   memory.ContentFields{Behavior: "gamma bad", Action: "gamma good"},
			FilePath:  "mem/gamma.toml",
		},
		{
			Situation: "delta context",
			Content:   memory.ContentFields{Behavior: "delta bad", Action: "delta good"},
			FilePath:  "mem/delta.toml",
		},
	}

	logger := &fakeTokenLogger{}
	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever, surface.WithInvocationTokenLogger(logger))

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

	g.Expect(logger.called).To(BeTrue())
	g.Expect(logger.mode).To(Equal(surface.ModePrompt))
	g.Expect(logger.tokenCount).To(BeNumerically(">", 0))
}

func TestWithInvocationTokenLogger_SetterCoversOption(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	logger := &fakeTokenLogger{}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	surfacer := surface.New(retriever, surface.WithInvocationTokenLogger(logger))

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/data",
		Message: "no match query zzzqqq",
	})

	g.Expect(err).NotTo(HaveOccurred())
}

func TestWithSurfacingLogger_SetterCoversOption(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	logger := &fakeSurfacingLogger{}
	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	surfacer := surface.New(retriever, surface.WithSurfacingLogger(logger))

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/data",
		Message: "no match query zzzqqq",
	})

	g.Expect(err).NotTo(HaveOccurred())
}

func TestWithSurfacingRecorder_SetterCoversOption(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var recorded []string

	retriever := &fakeRetriever{memories: []*memory.Stored{}}
	surfacer := surface.New(retriever, surface.WithSurfacingRecorder(func(path string) error {
		recorded = append(recorded, path)
		return nil
	}))

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/data",
		Message: "no match query zzzqqq",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(recorded).To(BeEmpty())
}

func TestWithTracker_RecordsSurfacing(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "commit context",
			Content:   memory.ContentFields{Behavior: "bad commit", Action: "good commit"},
			FilePath:  "mem/commit.toml",
		},
		{
			Situation: "build context",
			Content:   memory.ContentFields{Behavior: "bad build", Action: "good build"},
			FilePath:  "mem/build.toml",
		},
		{
			Situation: "review context",
			Content:   memory.ContentFields{Behavior: "bad review", Action: "good review"},
			FilePath:  "mem/review.toml",
		},
		{
			Situation: "deploy context",
			Content:   memory.ContentFields{Behavior: "bad deploy", Action: "good deploy"},
			FilePath:  "mem/deploy.toml",
		},
	}

	tracker := &fakeTracker{}
	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever, surface.WithTracker(tracker))

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/data",
		Message: "commit build",
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(tracker.called).To(BeTrue())
	g.Expect(tracker.mode).To(Equal(surface.ModePrompt))
}

func TestWithTracker_TrackerError_IsNonFatal(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := []*memory.Stored{
		{
			Situation: "commit context",
			Content:   memory.ContentFields{Behavior: "bad commit", Action: "good commit"},
			FilePath:  "mem/commit.toml",
		},
		{
			Situation: "build context",
			Content:   memory.ContentFields{Behavior: "bad build", Action: "good build"},
			FilePath:  "mem/build.toml",
		},
		{
			Situation: "review context",
			Content:   memory.ContentFields{Behavior: "bad review", Action: "good review"},
			FilePath:  "mem/review.toml",
		},
		{
			Situation: "deploy context",
			Content:   memory.ContentFields{Behavior: "bad deploy", Action: "good deploy"},
			FilePath:  "mem/deploy.toml",
		},
	}

	tracker := &fakeTracker{err: errors.New("tracker failure")}
	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever, surface.WithTracker(tracker))

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/data",
		Message: "commit build",
	})

	// Tracker errors are logged to stderr but do not fail the Run call.
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tracker.called).To(BeTrue())
}

func TestWriteResult_EmptyContext_WritesNothing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	retriever := &fakeRetriever{}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surface.ExportWriteResult(surfacer, &buf, surface.Result{Summary: "s", Context: ""}, "")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).To(BeEmpty())
}

func TestWriteResult_JSONFormat_WritesJSON(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	retriever := &fakeRetriever{}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surface.ExportWriteResult(surfacer, &buf,
		surface.Result{Summary: "the summary", Context: "the context"}, surface.FormatJSON)

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

	g.Expect(result.Summary).To(Equal("the summary"))
	g.Expect(result.Context).To(Equal("the context"))
}

func TestWriteResult_PlainFormat_WritesContext(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	retriever := &fakeRetriever{}
	surfacer := surface.New(retriever)

	var buf bytes.Buffer

	err := surface.ExportWriteResult(surfacer, &buf,
		surface.Result{Summary: "summary", Context: "some context text"}, "")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).To(Equal("some context text"))
}

// fakeRetriever is a test double for surface.MemoryRetriever.
type fakeRetriever struct {
	memories []*memory.Stored
	err      error
}

func (f *fakeRetriever) ListAllMemories(_ string) ([]*memory.Stored, error) {
	return f.memories, f.err
}

func (f *fakeRetriever) ListStored(_ string) ([]*memory.Stored, error) {
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

type fakeTokenLogger struct {
	called     bool
	mode       string
	tokenCount int
}

func (f *fakeTokenLogger) LogInvocationTokens(mode string, tokenCount int, _ time.Time) error {
	f.called = true
	f.mode = mode
	f.tokenCount = tokenCount

	return nil
}

type fakeTracker struct {
	called bool
	mode   string
	err    error
}

func (f *fakeTracker) RecordSurfacing(_ context.Context, _ []*memory.Stored, mode string) error {
	f.called = true
	f.mode = mode

	return f.err
}

type surfacingLogCall struct {
	memoryPath string
	mode       string
}

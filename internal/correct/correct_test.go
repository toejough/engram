package correct_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/anthropic"
	"engram/internal/correct"
	"engram/internal/memory"
	"engram/internal/policy"
)

// mustJSON marshals a value to JSON or fails the test.
func TestParseExtractionResponse_ArrayResponse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	response := `[{"situation":"s","behavior":"b","impact":"i","action":"a","filename_slug":"test"}]`

	result, err := correct.ExportParseExtractionResponse(response)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Situation).To(Equal("s"))
	g.Expect(result.Action).To(Equal("a"))
}

func TestParseExtractionResponse_EmptyArray(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := correct.ExportParseExtractionResponse("[]")
	g.Expect(err).To(MatchError(correct.ErrEmptyResponse))
}

func TestParseExtractionResponse_FencedArray(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	response := "```json\n[{\"situation\":\"s\",\"behavior\":\"b\",\"impact\":\"i\",\"action\":\"a\"}]\n```"

	result, err := correct.ExportParseExtractionResponse(response)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Situation).To(Equal("s"))
}

func TestRun_FastPathCorrection_StoresMemory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{returnPath: "/data/memories/use-targ.toml"}
	extraction := correct.ExtractionResult{
		Situation:    "user runs tests",
		Behavior:     "always use targ",
		Impact:       "prevents direct go test usage",
		Action:       "use targ test",
		FilenameSlug: "use-targ",
		Candidates:   []correct.CandidateResult{},
	}

	corrector := correct.New(
		correct.WithCaller(fakeCaller(map[string]string{
			anthropic.SonnetModel: mustJSON(t, extraction),
		})),
		correct.WithTranscriptReader(
			func(_ string, _ int) (string, int, error) {
				return "some transcript context", 22, nil
			},
		),
		correct.WithMemoryRetriever(
			func(_ string) ([]*memory.Stored, error) {
				return nil, nil
			},
		),
		correct.WithWriter(writer),
		correct.WithModifier(&fakeModifier{}),
		correct.WithPolicy(policy.Defaults()),
	)

	result, err := corrector.Run(
		context.Background(),
		"Remember to always use targ",
		"/transcript.jsonl",
		"/data",
		"myproject",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("stored"))
	g.Expect(writer.writtenRecord).NotTo(BeNil())

	if writer.writtenRecord == nil {
		return
	}

	g.Expect(writer.writtenRecord.Situation).To(Equal("user runs tests"))
}

func TestRun_HaikuCorrection_StoresMemory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{returnPath: "/data/memories/use-tabs.toml"}
	extraction := correct.ExtractionResult{
		Situation:    "formatting code",
		Behavior:     "use spaces",
		Impact:       "inconsistent formatting",
		Action:       "use tabs instead",
		FilenameSlug: "use-tabs",
		Candidates:   []correct.CandidateResult{},
	}

	corrector := correct.New(
		correct.WithCaller(fakeCaller(map[string]string{
			"claude-haiku-4-5-20251001": "CORRECTION",
			anthropic.SonnetModel:       mustJSON(t, extraction),
		})),
		correct.WithMemoryRetriever(
			func(_ string) ([]*memory.Stored, error) {
				return nil, nil
			},
		),
		correct.WithWriter(writer),
		correct.WithModifier(&fakeModifier{}),
		correct.WithPolicy(policy.Defaults()),
	)

	// Message has no fast-path keywords — Haiku decides.
	result, err := corrector.Run(
		context.Background(),
		"I prefer tabs over spaces for indentation",
		"",
		"/data",
		"myproject",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("stored"))
	g.Expect(writer.writtenRecord).NotTo(BeNil())

	if writer.writtenRecord == nil {
		return
	}

	g.Expect(writer.writtenRecord.Behavior).To(Equal("use spaces"))
}

func TestRun_NoTranscript_StillWorks(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{returnPath: "/data/memories/no-transcript.toml"}
	extraction := correct.ExtractionResult{
		Situation:    "general coding",
		Behavior:     "test first",
		Impact:       "catches bugs early",
		Action:       "write tests before code",
		FilenameSlug: "no-transcript",
		Candidates:   []correct.CandidateResult{},
	}

	corrector := correct.New(
		correct.WithCaller(fakeCaller(map[string]string{
			anthropic.SonnetModel: mustJSON(t, extraction),
		})),
		correct.WithMemoryRetriever(
			func(_ string) ([]*memory.Stored, error) {
				return nil, nil
			},
		),
		correct.WithWriter(writer),
		correct.WithModifier(&fakeModifier{}),
		correct.WithPolicy(policy.Defaults()),
	)

	// Empty transcript path — should still work.
	result, err := corrector.Run(
		context.Background(),
		"Always write tests before code",
		"",
		"/data",
		"myproject",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("stored"))
	g.Expect(writer.writtenRecord).NotTo(BeNil())
}

func TestRun_NotCorrection_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	corrector := correct.New(
		correct.WithCaller(fakeCaller(map[string]string{
			"claude-haiku-4-5-20251001": "NOT_CORRECTION",
		})),
		correct.WithWriter(&fakeWriter{}),
		correct.WithModifier(&fakeModifier{}),
		correct.WithPolicy(policy.Defaults()),
	)

	result, err := corrector.Run(
		context.Background(),
		"What is the weather today?",
		"",
		"/data",
		"myproject",
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

func TestRun_TranscriptReaderError_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	readErr := errors.New("transcript not found")

	corrector := correct.New(
		correct.WithCaller(fakeCaller(map[string]string{})),
		correct.WithTranscriptReader(
			func(_ string, _ int) (string, int, error) {
				return "", 0, readErr
			},
		),
		correct.WithWriter(&fakeWriter{}),
		correct.WithModifier(&fakeModifier{}),
		correct.WithPolicy(policy.Defaults()),
	)

	_, err := corrector.Run(
		context.Background(),
		"Remember to always use targ",
		"/missing/transcript.jsonl",
		"/data",
		"myproject",
	)

	g.Expect(err).To(MatchError(readErr))
}

func TestRun_WithCandidates_Duplicate(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{}
	extraction := correct.ExtractionResult{
		Situation:    "user runs tests",
		Behavior:     "always use targ",
		Impact:       "prevents direct go test usage",
		Action:       "use targ test",
		FilenameSlug: "use-targ",
		Candidates: []correct.CandidateResult{
			{
				Name:        "use-targ",
				Disposition: correct.DispositionDuplicate,
				Reason:      "same behavior already recorded",
			},
		},
	}

	// Need 5+ docs for BM25 IDF to produce positive scores on discriminating terms.
	memories := []*memory.Stored{
		{
			FilePath:  "/data/memories/use-targ.toml",
			Situation: "targ testing verification",
		},
		{
			FilePath:  "/data/memories/build.toml",
			Situation: "building compilation artifacts",
		},
		{
			FilePath:  "/data/memories/docs.toml",
			Situation: "documentation markdown headers",
		},
		{
			FilePath:  "/data/memories/deploy.toml",
			Situation: "deployment containers kubernetes",
		},
		{
			FilePath:  "/data/memories/monitor.toml",
			Situation: "monitoring alerts dashboards",
		},
	}

	corrector := correct.New(
		correct.WithCaller(fakeCaller(map[string]string{
			anthropic.SonnetModel: mustJSON(t, extraction),
		})),
		correct.WithMemoryRetriever(
			func(_ string) ([]*memory.Stored, error) {
				return memories, nil
			},
		),
		correct.WithWriter(writer),
		correct.WithModifier(&fakeModifier{}),
		correct.WithPolicy(policy.Defaults()),
	)

	result, err := corrector.Run(
		context.Background(),
		"Remember to always use targ for tests",
		"",
		"/data",
		"myproject",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("duplicate_skipped"))
	g.Expect(writer.writtenRecord).To(BeNil(), "should not write on duplicate")
}

func TestRun_WithEmptyMemories_StoresNew(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{returnPath: "/data/memories/new-mem.toml"}
	extraction := correct.ExtractionResult{
		Situation:    "coding",
		Behavior:     "test first",
		Impact:       "catches bugs",
		Action:       "write tests",
		FilenameSlug: "new-mem",
		Candidates:   []correct.CandidateResult{},
	}

	corrector := correct.New(
		correct.WithCaller(fakeCaller(map[string]string{
			anthropic.SonnetModel: mustJSON(t, extraction),
		})),
		correct.WithMemoryRetriever(
			func(_ string) ([]*memory.Stored, error) {
				return []*memory.Stored{}, nil
			},
		),
		correct.WithWriter(writer),
		correct.WithModifier(&fakeModifier{}),
		correct.WithPolicy(policy.Defaults()),
	)

	result, err := corrector.Run(
		context.Background(),
		"Always write tests first",
		"",
		"/data",
		"myproject",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("stored"))
}

func TestRun_WithLowScoreCandidates_FiltersBelow(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{returnPath: "/data/memories/new-correction.toml"}
	extraction := correct.ExtractionResult{
		Situation:    "deploying apps",
		Behavior:     "use docker",
		Impact:       "consistent environments",
		Action:       "containerize everything",
		FilenameSlug: "new-correction",
		Candidates:   []correct.CandidateResult{},
	}

	// Memory with completely unrelated text — BM25 should score below threshold.
	unrelatedMemory := &memory.Stored{
		FilePath:  "/data/memories/unrelated.toml",
		Situation: "writing documentation",
		Behavior:  "use markdown format",
		Impact:    "consistent docs",
		Action:    "format with headers",
	}

	corrector := correct.New(
		correct.WithCaller(fakeCaller(map[string]string{
			anthropic.SonnetModel: mustJSON(t, extraction),
		})),
		correct.WithMemoryRetriever(
			func(_ string) ([]*memory.Stored, error) {
				return []*memory.Stored{unrelatedMemory}, nil
			},
		),
		correct.WithWriter(writer),
		correct.WithModifier(&fakeModifier{}),
		correct.WithPolicy(policy.Defaults()),
	)

	result, err := corrector.Run(
		context.Background(),
		"Remember to always containerize deployments with docker",
		"",
		"/data",
		"myproject",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("stored"))
}

func TestRun_WithMaxCandidatesExceeded_LimitsCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := &fakeWriter{returnPath: "/data/memories/limited.toml"}
	extraction := correct.ExtractionResult{
		Situation:    "testing code",
		Behavior:     "use targ",
		Impact:       "consistent builds",
		Action:       "run targ test",
		FilenameSlug: "limited",
		Candidates:   []correct.CandidateResult{},
	}

	// Need 5+ docs for BM25 IDF to be positive on discriminating terms.
	// "targ" and "testing" appear in few docs, giving them positive IDF.
	memories := []*memory.Stored{
		{
			FilePath:  "/data/memories/targ-test.toml",
			Situation: "targ testing verification",
		},
		{
			FilePath:  "/data/memories/targ-build.toml",
			Situation: "targ building compilation",
		},
		{
			FilePath:  "/data/memories/docs.toml",
			Situation: "documentation markdown headers",
		},
		{
			FilePath:  "/data/memories/deploy.toml",
			Situation: "deployment containers kubernetes",
		},
		{
			FilePath:  "/data/memories/monitor.toml",
			Situation: "monitoring alerts dashboards",
		},
	}

	// Policy with max 1 candidate to trigger the limit branch.
	pol := policy.Defaults()
	pol.ExtractCandidateCountMax = 1
	pol.ExtractBM25Threshold = 0.01

	corrector := correct.New(
		correct.WithCaller(fakeCaller(map[string]string{
			anthropic.SonnetModel: mustJSON(t, extraction),
		})),
		correct.WithMemoryRetriever(
			func(_ string) ([]*memory.Stored, error) {
				return memories, nil
			},
		),
		correct.WithWriter(writer),
		correct.WithModifier(&fakeModifier{}),
		correct.WithPolicy(pol),
	)

	result, err := corrector.Run(
		context.Background(),
		"Remember to always use targ testing",
		"",
		"/data",
		"myproject",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("stored"))
}

func TestRun_WithRetrieverError_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	retrieveErr := errors.New("database unavailable")

	corrector := correct.New(
		correct.WithCaller(fakeCaller(map[string]string{})),
		correct.WithMemoryRetriever(
			func(_ string) ([]*memory.Stored, error) {
				return nil, retrieveErr
			},
		),
		correct.WithWriter(&fakeWriter{}),
		correct.WithModifier(&fakeModifier{}),
		correct.WithPolicy(policy.Defaults()),
	)

	_, err := corrector.Run(
		context.Background(),
		"Remember to always use targ",
		"",
		"/data",
		"myproject",
	)

	g.Expect(err).To(MatchError(retrieveErr))
}

// fakeCaller returns a CallerFunc that routes by model name.
func fakeCaller(responses map[string]string) correct.CallerFunc {
	return func(_ context.Context, model, _, _ string) (string, error) {
		if resp, ok := responses[model]; ok {
			return resp, nil
		}

		return "", nil
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshaling JSON: %v", err)
	}

	return string(data)
}

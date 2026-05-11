package cycle

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"engram/internal/debuglog"
	"engram/internal/memory"
)

// Cycle orchestrates a single per-turn evaluation cycle: it extracts
// learnings from the recent transcript, persists them, then proposes
// recall queries and runs them.
type Cycle struct {
	runner      Runner
	transcripts TranscriptReader
	persister   Persister
	recaller    Recaller
	budget      int
}

// New wires a Cycle. Any of persister/recaller may be nil for partial use.
func New(runner Runner, transcripts TranscriptReader, persister Persister, recaller Recaller) *Cycle {
	return &Cycle{
		runner:      runner,
		transcripts: transcripts,
		persister:   persister,
		recaller:    recaller,
		budget:      defaultTranscriptBudget,
	}
}

// Run executes one cycle: extract → persist → propose queries → per-query recall.
func (c *Cycle) Run(ctx context.Context, projectDir string) (*Output, error) {
	cycleID := debuglog.NewCycleID()
	ctx = debuglog.WithCycleID(ctx, cycleID)

	debuglog.Log(ctx, "cycle.start", "cycle=%s projectDir=%s", cycleID, projectDir)

	cycleStart := time.Now()

	out := NewOutput()

	transcriptStart := time.Now()
	transcript, err := c.transcripts.Read(projectDir, c.budget)

	debuglog.Log(ctx, "cycle.transcript", "cycle=%s bytes=%d err=%v took=%s",
		cycleID, len(transcript), err, time.Since(transcriptStart))

	if err != nil {
		return out, fmt.Errorf("reading transcript: %w", err)
	}

	c.runLearningStep(ctx, transcript, out)
	c.runRecallStep(ctx, transcript, projectDir, out)

	debuglog.Log(ctx, "cycle.end", "cycle=%s learned=%d recalled=%d took=%s",
		cycleID, len(out.Learned), len(out.Recalled), time.Since(cycleStart))

	return out, nil
}

func (c *Cycle) persistOne(ctx context.Context, cand learnCandidate, out *Output) {
	switch cand.Type {
	case "feedback":
		name, ok, err := c.persister.WriteFeedback(
			ctx, cand.Situation, cand.Behavior, cand.Impact, cand.Action,
		)

		debuglog.Log(ctx, "persistOne", "type=feedback situation=%q name=%s persisted=%v err=%v",
			cand.Situation, name, ok, err)

		if err != nil || !ok {
			return
		}

		out.Learned = append(out.Learned, LearnedMemory{
			MemoryRecord: memory.MemoryRecord{
				Type:      "feedback",
				Situation: cand.Situation,
				Source:    "agent",
				Content: memory.ContentFields{
					Behavior: cand.Behavior,
					Impact:   cand.Impact,
					Action:   cand.Action,
				},
			},
			Name: name,
		})

	case "fact":
		name, ok, err := c.persister.WriteFact(
			ctx, cand.Situation, cand.Subject, cand.Predicate, cand.Object,
		)

		debuglog.Log(ctx, "persistOne", "type=fact situation=%q name=%s persisted=%v err=%v",
			cand.Situation, name, ok, err)

		if err != nil || !ok {
			return
		}

		out.Learned = append(out.Learned, LearnedMemory{
			MemoryRecord: memory.MemoryRecord{
				Type:      "fact",
				Situation: cand.Situation,
				Source:    "agent",
				Content: memory.ContentFields{
					Subject:   cand.Subject,
					Predicate: cand.Predicate,
					Object:    cand.Object,
				},
			},
			Name: name,
		})
	}
}

func (c *Cycle) runLearningStep(ctx context.Context, transcript string, out *Output) {
	if c.persister == nil {
		return
	}

	cycleID := debuglog.CycleIDFromContext(ctx)
	learnCtx := debuglog.WithPhase(ctx, "cycle.learn")

	debuglog.Log(ctx, "cycle.learn.start", "cycle=%s", cycleID)

	start := time.Now()

	resp, err := c.runner.Run(learnCtx, LearnExtractionPrompt(transcript))
	if err != nil {
		debuglog.Log(ctx, "cycle.learn.end", "cycle=%s outcome=llm_error err=%v took=%s",
			cycleID, err, time.Since(start))

		return
	}

	candidates, parseErr := parseLearnCandidates(resp)
	if parseErr != nil {
		debuglog.Log(ctx, "cycle.learn.end", "cycle=%s outcome=parse_error err=%v took=%s",
			cycleID, parseErr, time.Since(start))

		return
	}

	debuglog.Log(ctx, "cycle.learn.end", "cycle=%s outcome=ok candidates=%d took=%s",
		cycleID, len(candidates), time.Since(start))

	for _, cand := range candidates {
		c.persistOne(ctx, cand, out)
	}
}

func (c *Cycle) runRecallStep(ctx context.Context, transcript, projectDir string, out *Output) {
	if c.recaller == nil {
		return
	}

	cycleID := debuglog.CycleIDFromContext(ctx)
	proposeCtx := debuglog.WithPhase(ctx, "cycle.propose_queries")

	debuglog.Log(ctx, "cycle.propose_queries.start", "cycle=%s", cycleID)

	start := time.Now()

	resp, err := c.runner.Run(proposeCtx, QueryProposalPrompt(transcript))
	if err != nil {
		debuglog.Log(ctx, "cycle.propose_queries.end", "cycle=%s outcome=llm_error err=%v took=%s",
			cycleID, err, time.Since(start))

		return
	}

	queries := parseQueries(resp)

	debuglog.Log(ctx, "cycle.propose_queries.end", "cycle=%s outcome=ok count=%d took=%s",
		cycleID, len(queries), time.Since(start))

	for idx, query := range queries {
		queryCtx := debuglog.WithPhase(ctx, fmt.Sprintf("cycle.recall.q%d", idx))

		debuglog.Log(ctx, "cycle.recall.start", "cycle=%s q=%d query=%q", cycleID, idx, query)

		recallStart := time.Now()
		report, recErr := c.recaller.Recall(queryCtx, projectDir, query)

		debuglog.Log(ctx, "cycle.recall.end", "cycle=%s q=%d report_bytes=%d err=%v took=%s",
			cycleID, idx, len(report), recErr, time.Since(recallStart))

		if recErr != nil || report == "" {
			continue
		}

		out.Recalled = append(out.Recalled, RecalledReport{
			Query:  query,
			Report: report,
		})
	}
}

// Persister persists a candidate learning. Returns the slug-name written
// (post auto-increment) and whether dedup skipped it.
type Persister interface {
	WriteFeedback(
		ctx context.Context,
		situation, behavior, impact, action string,
	) (name string, persisted bool, err error)
	WriteFact(
		ctx context.Context,
		situation, subject, predicate, object string,
	) (name string, persisted bool, err error)
}

// Recaller runs the existing recall pipeline for a single query.
type Recaller interface {
	Recall(ctx context.Context, projectDir, query string) (report string, err error)
}

// Runner runs a single LLM prompt and returns the response text.
type Runner interface {
	Run(ctx context.Context, prompt string) (string, error)
}

// TranscriptReader returns the recent project transcript under a budget.
type TranscriptReader interface {
	Read(projectDir string, budget int) (string, error)
}

// unexported constants.
const (
	defaultTranscriptBudget = 15 * 1024
	maxQueries              = 5
	noQueriesSentinel       = "NO QUERIES"
)

type learnCandidate struct {
	Type      string `json:"type"`
	Situation string `json:"situation"`
	Behavior  string `json:"behavior,omitempty"`
	Impact    string `json:"impact,omitempty"`
	Action    string `json:"action,omitempty"`
	Subject   string `json:"subject,omitempty"`
	Predicate string `json:"predicate,omitempty"`
	Object    string `json:"object,omitempty"`
}

func parseLearnCandidates(input string) ([]learnCandidate, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || trimmed == "[]" {
		return nil, nil
	}

	var out []learnCandidate

	err := json.Unmarshal([]byte(trimmed), &out)
	if err != nil {
		return nil, fmt.Errorf("parsing learn candidates: %w", err)
	}

	return out, nil
}

func parseQueries(input string) []string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || trimmed == noQueriesSentinel {
		return nil
	}

	lines := strings.Split(trimmed, "\n")
	queries := make([]string, 0, len(lines))

	for _, line := range lines {
		query := strings.TrimSpace(line)
		if query == "" || query == noQueriesSentinel {
			continue
		}

		queries = append(queries, query)

		if len(queries) >= maxQueries {
			break
		}
	}

	return queries
}

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"engram/internal/cycle"
	"engram/internal/debuglog"
	"engram/internal/llmcmd"
	"engram/internal/memory"
	"engram/internal/recall"
)

// CycleArgs holds the flag values for `engram cycle`.
type CycleArgs struct {
	LLMCmd           string `targ:"flag,name=llm-cmd,desc=LLM command (overrides ENGRAM_LLM_CMD)"`
	ProjectDir       string `targ:"flag,name=project-dir,desc=project working directory"`
	DataDir          string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=engram data directory"`
	TranscriptBudget int    `targ:"flag,name=transcript-budget,desc=max bytes of transcript fed to LLM"`
}

// RunCycle executes one engram cycle and writes the JSON output to stdout.
func RunCycle(ctx context.Context, args CycleArgs, stdout io.Writer) error {
	debuglog.Log("engram_cycle.invoke", "projectDir=%s dataDir=%s", args.ProjectDir, args.DataDir)

	requireErr := requireLLMCmd(args.LLMCmd)
	if requireErr != nil {
		return fmt.Errorf("cycle: %w", requireErr)
	}

	cmdString := resolveLLMCmd(args.LLMCmd)
	runner := llmcmd.New(cmdString)

	dataDir := args.DataDir

	defaultErr := applyDataDirDefault(&dataDir)
	if defaultErr != nil {
		return fmt.Errorf("cycle: %w", defaultErr)
	}

	transcripts := newTranscriptReaderAdapter()
	persister := &cyclePersisterAdapter{
		dataDir: dataDir,
		caller:  llmcmd.CallerFunc(runner),
		lister:  memory.NewLister(),
		stdout:  io.Discard,
	}
	recaller := &cycleRecallerAdapter{
		dataDir:    dataDir,
		summarizer: llmcmd.NewExtractor(runner),
	}

	cycler := cycle.New(runner, transcripts, persister, recaller)

	out, runErr := cycler.Run(ctx, args.ProjectDir)
	if runErr != nil {
		return fmt.Errorf("cycle: %w", runErr)
	}

	encoded, marshalErr := json.MarshalIndent(out, "", "  ")
	if marshalErr != nil {
		return fmt.Errorf("cycle: marshalling output: %w", marshalErr)
	}

	debuglog.Log("engram_cycle.done", "err=%v", runErr)

	_, writeErr := stdout.Write(append(encoded, '\n'))
	if writeErr != nil {
		return fmt.Errorf("cycle: writing output: %w", writeErr)
	}

	return nil
}

// cyclePersisterAdapter implements cycle.Persister by reusing writeMemory.
type cyclePersisterAdapter struct {
	dataDir string
	caller  llmCaller
	lister  memoryLister
	stdout  io.Writer
}

func (a *cyclePersisterAdapter) WriteFact(
	ctx context.Context,
	situation, subject, predicate, object string,
) (string, bool, error) {
	rec := &memory.MemoryRecord{
		SchemaVersion: memorySchemaVersion,
		Source:        "agent",
		Situation:     situation,
		Type:          typeFact,
		Content: memory.ContentFields{
			Subject:   subject,
			Predicate: predicate,
			Object:    object,
		},
	}

	dataDir := a.dataDir

	return writeMemory(ctx, rec, situation, &dataDir, false, a.stdout, "cycle", a.caller, a.lister)
}

func (a *cyclePersisterAdapter) WriteFeedback(
	ctx context.Context,
	situation, behavior, impact, action string,
) (string, bool, error) {
	rec := &memory.MemoryRecord{
		SchemaVersion: memorySchemaVersion,
		Source:        "agent",
		Situation:     situation,
		Type:          typeFeedback,
		Content: memory.ContentFields{
			Behavior: behavior,
			Impact:   impact,
			Action:   action,
		},
	}

	dataDir := a.dataDir

	return writeMemory(ctx, rec, situation, &dataDir, false, a.stdout, "cycle", a.caller, a.lister)
}

// cycleRecallerAdapter implements cycle.Recaller by running the existing
// recall pipeline scoped to the project dir, with --llm-cmd as the backend.
type cycleRecallerAdapter struct {
	dataDir    string
	summarizer recall.SummarizerI
}

func (a *cycleRecallerAdapter) Recall(ctx context.Context, projectDir, query string) (string, error) {
	finder := recall.NewCompositeSessionFinder(
		recall.NewSessionFinder(&osDirLister{}),
		recall.NewOpencodeSessionFinder(recall.DefaultOpencodeDBPath(), ""),
	)
	reader := recall.NewCompositeTranscriptReader(
		recall.NewTranscriptReader(&osFileReader{}),
		recall.NewOpencodeTranscriptReader(recall.DefaultOpencodeDBPath()),
	)

	orch := recall.NewOrchestrator(finder, reader, a.summarizer, memory.NewLister(), a.dataDir)

	result, err := orch.Recall(ctx, query, projectDir)
	if err != nil {
		return "", fmt.Errorf("recalling for query %q: %w", query, err)
	}

	return result.Report, nil
}

// transcriptReaderAdapter implements cycle.TranscriptReader by reusing
// the existing recall finder + reader composites.
type transcriptReaderAdapter struct {
	finder recall.Finder
	reader recall.Reader
}

func (a *transcriptReaderAdapter) Read(projectDir string, budget int) (string, error) {
	sessions, findErr := a.finder.Find(projectDir)
	if findErr != nil {
		return "", fmt.Errorf("finding sessions: %w", findErr)
	}

	var builder strings.Builder

	used := 0

	for _, session := range sessions {
		remaining := budget - used
		if remaining <= 0 {
			break
		}

		content, _, readErr := a.reader.Read(session.Path, remaining)
		if readErr != nil {
			continue
		}

		builder.WriteString(content)

		used = builder.Len()
	}

	return builder.String(), nil
}

func newTranscriptReaderAdapter() *transcriptReaderAdapter {
	return &transcriptReaderAdapter{
		finder: recall.NewCompositeSessionFinder(
			recall.NewSessionFinder(&osDirLister{}),
			recall.NewOpencodeSessionFinder(recall.DefaultOpencodeDBPath(), ""),
		),
		reader: recall.NewCompositeTranscriptReader(
			recall.NewTranscriptReader(&osFileReader{}),
			recall.NewOpencodeTranscriptReader(recall.DefaultOpencodeDBPath()),
		),
	}
}

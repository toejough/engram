package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

// unexported constants.
const (
	conflictDetectionSystemPrompt = `You are a memory deduplication checker. ` +
		`Given an index of existing memories and a new memory, determine if the new memory ` +
		`is a duplicate or contradiction of any existing one.

Respond with one of:
- "NONE" if no duplicates or contradictions found
- "DUPLICATE: <name>" if it duplicates an existing memory (one per line)
- "CONTRADICTION: <name>" if it contradicts an existing memory (one per line)

Only output the result lines, nothing else.`
	conflictDetectionUserPrompt = `Existing memories:
%s
New memory:
%s

Is this new memory a duplicate or contradiction of any existing one?`
	memorySchemaVersion = 2
	splitNParts         = 2
	typeFact            = "fact"
	typeFeedback        = "feedback"
)

// unexported variables.
var (
	errInvalidSource = errors.New("source must be \"human\" or \"agent\"")
)

// llmCaller calls an LLM with a model, system prompt, and user prompt.
type llmCaller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)

// memoryLister lists all stored memories.
type memoryLister interface {
	ListAllMemories(dataDir string) ([]*memory.Stored, error)
}

func callHaikuForConflicts(
	ctx context.Context,
	caller llmCaller,
	index, description string,
) (string, error) {
	systemPrompt := conflictDetectionSystemPrompt
	userPrompt := fmt.Sprintf(conflictDetectionUserPrompt, index, description)

	response, err := caller(ctx, "claude-haiku-4-5-20251001", systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("calling Haiku: %w", err)
	}

	return response, nil
}

func checkForConflicts(
	ctx context.Context,
	record *memory.MemoryRecord,
	dataDir string,
	stdout io.Writer,
	caller llmCaller,
	lister memoryLister,
) (bool, error) {
	if caller == nil || lister == nil {
		return false, nil
	}

	memories, err := lister.ListAllMemories(dataDir)
	if err != nil {
		// No existing memories is not an error -- nothing to conflict with.
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("listing memories: %w", err)
	}

	if len(memories) == 0 {
		return false, nil
	}

	index := memory.BuildIndex(memories)
	description := describeNewMemory(record)

	response, callErr := callHaikuForConflicts(ctx, caller, index, description)
	if callErr != nil {
		// API errors are non-fatal for dedup: fall through and write anyway.
		return false, nil //nolint:nilerr // intentional: API failure is non-fatal
	}

	return parseConflictResponse(response, dataDir, stdout), nil
}

func describeNewMemory(record *memory.MemoryRecord) string {
	var builder strings.Builder

	_, _ = fmt.Fprintf(&builder, "Type: %s\n", record.Type)
	_, _ = fmt.Fprintf(&builder, "Situation: %s\n", record.Situation)

	if record.Type == typeFact {
		_, _ = fmt.Fprintf(&builder, "Subject: %s\n", record.Content.Subject)
		_, _ = fmt.Fprintf(&builder, "Predicate: %s\n", record.Content.Predicate)
		_, _ = fmt.Fprintf(&builder, "Object: %s\n", record.Content.Object)
	} else {
		_, _ = fmt.Fprintf(&builder, "Behavior: %s\n", record.Content.Behavior)
		_, _ = fmt.Fprintf(&builder, "Impact: %s\n", record.Content.Impact)
		_, _ = fmt.Fprintf(&builder, "Action: %s\n", record.Content.Action)
	}

	_, _ = fmt.Fprintf(&builder, "Source: %s\n", record.Source)

	return builder.String()
}

// unexported functions.

// makeConflictDeps wires real I/O deps for conflict detection.
// Returns nil caller when no API token is available (skips dedup).
func makeConflictDeps(ctx context.Context) (llmCaller, memoryLister) {
	token := resolveToken(ctx)

	var caller llmCaller
	if token != "" {
		caller = makeAnthropicCaller(token)
	}

	return caller, memory.NewLister()
}

func parseConflictLine(line, dataDir string, stdout io.Writer) {
	parts := strings.SplitN(line, ":", splitNParts)
	if len(parts) < splitNParts {
		return
	}

	conflictType := parts[0]
	name := strings.TrimSpace(parts[1])

	_, _ = fmt.Fprintf(stdout, "%s: %s\n", conflictType, name)

	memPath := memory.ResolveMemoryPath(dataDir, name, fileExists)

	mem, loadErr := loadMemoryTOML(memPath)
	if loadErr != nil {
		return
	}

	renderConflictContent(stdout, mem)
}

func parseConflictResponse(response, dataDir string, stdout io.Writer) bool {
	trimmed := strings.TrimSpace(response)

	if trimmed == "NONE" {
		return false
	}

	lines := strings.Split(trimmed, "\n")
	foundConflict := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "DUPLICATE:") || strings.HasPrefix(line, "CONTRADICTION:") {
			foundConflict = true

			parseConflictLine(line, dataDir, stdout)
		}
	}

	return foundConflict
}

func renderConflictContent(writer io.Writer, mem *memory.MemoryRecord) {
	if mem.Situation != "" {
		_, _ = fmt.Fprintf(writer, "situation: %s\n", mem.Situation)
	}

	if mem.Type == typeFact {
		renderConflictFactFields(writer, mem)
	} else {
		renderConflictFeedbackFields(writer, mem)
	}
}

func renderConflictFactFields(writer io.Writer, mem *memory.MemoryRecord) {
	if mem.Content.Subject != "" {
		_, _ = fmt.Fprintf(writer, "subject: %s\n", mem.Content.Subject)
	}

	if mem.Content.Predicate != "" {
		_, _ = fmt.Fprintf(writer, "predicate: %s\n", mem.Content.Predicate)
	}

	if mem.Content.Object != "" {
		_, _ = fmt.Fprintf(writer, "object: %s\n", mem.Content.Object)
	}
}

func renderConflictFeedbackFields(writer io.Writer, mem *memory.MemoryRecord) {
	if mem.Content.Behavior != "" {
		_, _ = fmt.Fprintf(writer, "behavior: %s\n", mem.Content.Behavior)
	}

	if mem.Content.Impact != "" {
		_, _ = fmt.Fprintf(writer, "impact: %s\n", mem.Content.Impact)
	}

	if mem.Content.Action != "" {
		_, _ = fmt.Fprintf(writer, "action: %s\n", mem.Content.Action)
	}
}

func runLearnFact(ctx context.Context, args LearnFactArgs, stdout io.Writer) error {
	srcErr := validateSource(args.Source)
	if srcErr != nil {
		return fmt.Errorf("learn fact: %w", srcErr)
	}

	record := &memory.MemoryRecord{
		SchemaVersion: memorySchemaVersion,
		Source:        args.Source,
		Situation:     args.Situation,
		Type:          typeFact,
		Content: memory.ContentFields{
			Subject:   args.Subject,
			Predicate: args.Predicate,
			Object:    args.Object,
		},
	}

	dataDir := args.DataDir
	caller, lister := makeConflictDeps(ctx)

	return writeMemory(ctx, record, args.Situation, &dataDir, args.NoDupCheck, stdout, "learn fact", caller, lister)
}

func runLearnFeedback(ctx context.Context, args LearnFeedbackArgs, stdout io.Writer) error {
	srcErr := validateSource(args.Source)
	if srcErr != nil {
		return fmt.Errorf("learn feedback: %w", srcErr)
	}

	record := &memory.MemoryRecord{
		SchemaVersion: memorySchemaVersion,
		Source:        args.Source,
		Situation:     args.Situation,
		Type:          typeFeedback,
		Content: memory.ContentFields{
			Behavior: args.Behavior,
			Impact:   args.Impact,
			Action:   args.Action,
		},
	}

	dataDir := args.DataDir
	caller, lister := makeConflictDeps(ctx)

	return writeMemory(ctx, record, args.Situation, &dataDir, args.NoDupCheck, stdout, "learn feedback", caller, lister)
}

func validateSource(source string) error {
	if source != "human" && source != "agent" {
		return errInvalidSource
	}

	return nil
}

func writeMemory(
	ctx context.Context,
	record *memory.MemoryRecord,
	situation string,
	dataDir *string,
	noDupCheck bool,
	stdout io.Writer,
	cmdName string,
	caller llmCaller,
	lister memoryLister,
) error {
	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("%s: %w", cmdName, defaultErr)
	}

	slug := tomlwriter.Slugify(situation)

	if !noDupCheck {
		conflict, checkErr := checkForConflicts(ctx, record, *dataDir, stdout, caller, lister)
		if checkErr != nil {
			return fmt.Errorf("%s: %w", cmdName, checkErr)
		}

		if conflict {
			return nil
		}
	}

	writer := tomlwriter.New()

	filePath, writeErr := writer.Write(record, slug, *dataDir)
	if writeErr != nil {
		return fmt.Errorf("%s: %w", cmdName, writeErr)
	}

	name := memory.NameFromPath(filePath)

	_, printErr := fmt.Fprintf(stdout, "CREATED: %s\n", name)
	if printErr != nil {
		return fmt.Errorf("%s: %w", cmdName, printErr)
	}

	return nil
}

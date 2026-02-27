package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunDecide logs a decision to memory.
func RunDecide(args DecideArgs, homeDir string) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	var alternatives []string
	if args.Alternatives != "" {
		alternatives = strings.Split(args.Alternatives, ",")
		for i := range alternatives {
			alternatives[i] = strings.TrimSpace(alternatives[i])
		}
	}

	opts := DecideOpts{
		Context:      args.Context,
		Choice:       args.Choice,
		Reason:       args.Reason,
		Alternatives: alternatives,
		Project:      args.Project,
		MemoryRoot:   memoryRoot,
	}

	result, err := Decide(opts)
	if err != nil {
		return err
	}

	fmt.Printf("Decision logged to: %s\n", result.FilePath)

	return nil
}

// RunDigest computes and displays a memory digest.
func RunDigest(args DigestArgs, homeDir string) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	sinceStr := args.Since
	if sinceStr == "" {
		sinceStr = "168h"
	}

	since, err := time.ParseDuration(sinceStr)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", sinceStr, err)
	}

	opts := DigestOptions{
		Since:      since,
		Tier:       args.Tier,
		FlagsOnly:  args.FlagsOnly,
		MaxEntries: args.MaxEntries,
	}

	digest, err := ComputeDigest(opts, memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to compute digest: %w", err)
	}

	if args.FlagsOnly {
		if len(digest.Flags) == 0 {
			fmt.Println("No flags detected")
			return nil
		}

		fmt.Println("Flags:")

		for _, flag := range digest.Flags {
			fmt.Printf("  \u26a0 %s\n", flag)
		}

		return nil
	}

	output := FormatDigest(digest)
	fmt.Print(output)

	return nil
}

// RunGrep searches memory for a pattern.
func RunGrep(args GrepArgs, homeDir string) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	opts := GrepOpts{
		Pattern:          args.Pattern,
		Project:          args.Project,
		IncludeDecisions: args.IncludeDecisions,
		MemoryRoot:       memoryRoot,
	}

	result, err := Grep(opts)
	if err != nil {
		return err
	}

	if len(result.Matches) == 0 {
		fmt.Println("No matches found")
		return nil
	}

	for _, m := range result.Matches {
		fmt.Printf("%s:%d: %s\n", m.File, m.LineNum, m.Line)
	}

	return nil
}

// RunLearn stores a learning in memory.
func RunLearn(args LearnArgs, homeDir string) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	opts := LearnOpts{
		Message:    args.Message,
		Project:    args.Project,
		Source:     args.Source,
		Type:       args.Type,
		MemoryRoot: memoryRoot,
	}

	if !args.NoLLM {
		opts.Extractor = NewLLMExtractor()
		if opts.Extractor == nil {
			return errors.New("LLM extractor unavailable (keychain auth failed); use --no-llm to store without enrichment")
		}
	}

	if err := Learn(opts); err != nil {
		return err
	}

	fmt.Println("Learned: " + args.Message)

	return nil
}

// RunQuery queries memory for relevant entries.
func RunQuery(args QueryArgs, homeDir string, stdin io.Reader) error {
	stdinCount := 0
	if args.StdinProject {
		stdinCount++
	}

	if args.StdinPrompt {
		stdinCount++
	}

	if args.StdinTool {
		stdinCount++
	}

	if stdinCount > 1 {
		return errors.New("only one of --stdin-project, --stdin-prompt, --stdin-tool may be set")
	}

	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	limit := args.Limit
	if limit == 0 {
		limit = 10
	}

	queryText := args.Text
	project := args.Project

	var hookInput *HookInput
	if args.StdinProject || args.StdinPrompt || args.StdinTool {
		hookInput, _ = ParseHookInput(stdin)
	}

	if hookInput != nil {
		if project == "" {
			project = DeriveProjectName(hookInput.Cwd)
		}

		if args.StdinPrompt {
			if hookInput.Prompt != "" {
				queryText = hookInput.Prompt
			}
		} else if args.StdinTool {
			queryText = hookInput.ExtractToolQuery()
		}
	}

	if queryText == "" {
		return nil
	}

	searchText := queryText
	if project != "" {
		searchText = "[" + project + "] " + queryText
	}

	minScore := args.SimilarityThreshold
	if minScore == 0 {
		minScore = DefaultSimilarityThreshold
	}

	opts := QueryOpts{
		Text:       searchText,
		Limit:      limit * 2,
		Project:    project,
		MemoryRoot: memoryRoot,
		MinScore:   minScore,
	}

	result, err := Query(opts)
	if err != nil {
		return nil
	}

	if hookInput != nil {
		var retrievalResults []RetrievalResult
		for _, r := range result.Results {
			retrievalResults = append(retrievalResults, RetrievalResult{
				ID:      r.ID,
				Content: r.Content,
				Score:   r.Score,
				Tier:    "embedding",
			})
		}

		metadata := map[string]string{
			"project": project,
		}
		if hookInput.ToolName != "" {
			metadata["tool_name"] = hookInput.ToolName
		}

		logEntry := RetrievalLogEntry{
			Timestamp:     time.Now().Format(time.RFC3339),
			Hook:          hookInput.HookEventName,
			Query:         queryText,
			Results:       retrievalResults,
			FilteredCount: result.FilteredCount,
			SessionID:     hookInput.SessionID,
			Metadata:      metadata,
		}
		_ = LogRetrieval(memoryRoot, logEntry)
	}

	if args.Verbose {
		method := "vector"
		if result.UsedHybridSearch {
			method = "hybrid (vector+BM25)"
		}

		fmt.Fprintf(os.Stderr, "Search method: %s\n", method)
		fmt.Fprintf(os.Stderr, "BM25 available: %v\n", result.BM25Enabled)

		if args.StdinPrompt {
			fmt.Fprintf(os.Stderr, "Query source: stdin-prompt\n")
		} else if args.StdinTool {
			fmt.Fprintf(os.Stderr, "Query source: stdin-tool\n")
		} else if args.StdinProject {
			fmt.Fprintf(os.Stderr, "Query source: stdin-project\n")
		}

		fmt.Fprintln(os.Stderr)
	}

	tier := TierCompact
	if args.Rich {
		tier = TierFull
	} else if args.Curate {
		tier = TierCurated
	}

	if args.StdinPrompt && !args.Curate && !args.Rich {
		tier = TierCurated
	}

	tier = ResolveTier(tier, hookInput)

	if hookInput != nil && hookInput.SupportsCuration() {
		ext := NewLLMExtractor()
		if ext != nil {
			dbPath := filepath.Join(memoryRoot, "embeddings.db")

			db, dbErr := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
			if dbErr == nil {
				defer func() { _ = db.Close() }()

				pipelineOutput := RunFilterPipeline(context.Background(), RunFilterPipelineOpts{
					DB:           db,
					Extractor:    ext,
					QueryResults: result.Results,
					QueryText:    queryText,
					HookEvent:    hookInput.HookEventName,
					SessionID:    hookInput.SessionID,
				})
				if pipelineOutput != "" {
					fmt.Print(pipelineOutput)
					return nil
				}
			}
		}
	}

	var extractor LLMExtractor

	if tier == TierCurated {
		ext := NewLLMExtractor()
		if ext == nil {
			tier = TierCompact
		} else {
			extractor = ext
		}
	}

	output := FormatMarkdown(FormatMarkdownOpts{
		Results:       result.Results,
		MinConfidence: float64(args.MinConfidence) / 100.0,
		MaxEntries:    limit,
		MaxTokens:     args.MaxTokens,
		Primacy:       args.Primacy,
		Tier:          tier,
		Query:         queryText,
		Extractor:     extractor,
	})

	fmt.Print(output)

	return nil
}

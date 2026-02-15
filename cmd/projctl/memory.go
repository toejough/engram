package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/toejough/projctl/internal/memory"
)

type memoryLearnArgs struct {
	Message    string `targ:"flag,short=m,required,desc=Learning message to store"`
	Project    string `targ:"flag,short=p,desc=Project to tag the learning with"`
	Source     string `targ:"flag,short=s,desc=Source type: internal or external (default: internal)"`
	Type       string `targ:"flag,short=t,desc=Memory type: correction or reflection (default: empty)"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
	NoLLM      bool   `targ:"flag,desc=Disable LLM-based knowledge extraction"`
}

func memoryLearn(args memoryLearnArgs) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = home + "/.claude/memory"
	}

	opts := memory.LearnOpts{
		Message:    args.Message,
		Project:    args.Project,
		Source:     args.Source,
		Type:       args.Type,
		MemoryRoot: memoryRoot,
	}

	// Wire LLM extractor (unless --no-llm is set)
	if !args.NoLLM {
		opts.Extractor = memory.NewLLMExtractor()
	}

	if err := memory.Learn(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Learned: " + args.Message)
	return nil
}

type memoryDecideArgs struct {
	Context      string `targ:"flag,short=c,required,desc=Decision context"`
	Choice       string `targ:"flag,required,desc=The choice made"`
	Reason       string `targ:"flag,short=r,required,desc=Reason for the decision"`
	Alternatives string `targ:"flag,short=a,desc=Comma-separated alternatives considered"`
	Project      string `targ:"flag,short=p,required,desc=Project name"`
	MemoryRoot   string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

func memoryDecide(args memoryDecideArgs) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = home + "/.claude/memory"
	}

	var alternatives []string
	if args.Alternatives != "" {
		alternatives = strings.Split(args.Alternatives, ",")
		for i := range alternatives {
			alternatives[i] = strings.TrimSpace(alternatives[i])
		}
	}

	opts := memory.DecideOpts{
		Context:      args.Context,
		Choice:       args.Choice,
		Reason:       args.Reason,
		Alternatives: alternatives,
		Project:      args.Project,
		MemoryRoot:   memoryRoot,
	}

	result, err := memory.Decide(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Decision logged to: %s\n", result.FilePath)
	return nil
}

type memoryGrepArgs struct {
	Pattern          string `targ:"positional,required,desc=Pattern to search for"`
	Project          string `targ:"flag,short=p,desc=Limit search to specific project"`
	IncludeDecisions bool   `targ:"flag,short=d,desc=Also search decisions files"`
	MemoryRoot       string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

func memoryGrep(args memoryGrepArgs) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = home + "/.claude/memory"
	}

	opts := memory.GrepOpts{
		Pattern:          args.Pattern,
		Project:          args.Project,
		IncludeDecisions: args.IncludeDecisions,
		MemoryRoot:       memoryRoot,
	}

	result, err := memory.Grep(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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

type memoryQueryArgs struct {
	Text                string  `targ:"positional,desc=Text to search for"`
	Limit               int     `targ:"flag,short=n,desc=Maximum number of results (default 10)"`
	Project             string  `targ:"flag,short=p,desc=Project name for retrieval tracking"`
	Verbose             bool    `targ:"flag,short=v,desc=Show detailed scoring info"`
	MemoryRoot          string  `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
	MinConfidence       int     `targ:"flag,name=min-confidence,desc=Minimum confidence threshold 0-100 (default: 0)"`
	SimilarityThreshold float64 `targ:"flag,name=similarity-threshold,desc=Minimum similarity score 0.0-1.0 (default: 0.7)"`
	MaxTokens           int     `targ:"flag,name=max-tokens,desc=Max token count for output (default: 2000)"`
	Primacy             bool    `targ:"flag,desc=Sort corrections first (primacy ordering)"`
	Rich                bool    `targ:"flag,desc=Show full metadata (confidence/retrieval count/match type/projects)"`
	Curate              bool    `targ:"flag,desc=Use LLM curation for result selection and relevance annotations"`
	StdinProject        bool    `targ:"flag,name=stdin-project,desc=Derive project from stdin hook JSON cwd"`
	StdinPrompt         bool    `targ:"flag,name=stdin-prompt,desc=Read query text and project from stdin hook JSON prompt field"`
	StdinTool           bool    `targ:"flag,name=stdin-tool,desc=Read query from stdin hook JSON tool_name + tool_input fields"`
}

func memoryQuery(args memoryQueryArgs) error {
	// Mutual exclusivity check for stdin modes
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
		return fmt.Errorf("only one of --stdin-project, --stdin-prompt, --stdin-tool may be set")
	}

	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = home + "/.claude/memory"
	}

	limit := args.Limit
	if limit == 0 {
		limit = 10
	}

	queryText := args.Text
	project := args.Project

	// Parse stdin for hook-based modes
	var hookInput *memory.HookInput
	if args.StdinProject || args.StdinPrompt || args.StdinTool {
		hookInput, _ = memory.ParseHookInput(os.Stdin)
	}

	if hookInput != nil {
		// Derive project from cwd if not set explicitly
		if project == "" {
			project = memory.DeriveProjectName(hookInput.Cwd)
		}

		if args.StdinPrompt {
			if hookInput.Prompt != "" {
				queryText = hookInput.Prompt
			}
		} else if args.StdinTool {
			queryText = hookInput.ExtractToolQuery()
		}
	}

	// Graceful degradation: if no query text, exit silently
	if queryText == "" {
		return nil
	}

	// If project is set, prepend it to the query for retrieval boosting
	searchText := queryText
	if project != "" {
		searchText = "[" + project + "] " + queryText
	}

	// Apply similarity threshold (default to DefaultSimilarityThreshold)
	minScore := args.SimilarityThreshold
	if minScore == 0 {
		minScore = memory.DefaultSimilarityThreshold
	}

	opts := memory.QueryOpts{
		Text:       searchText,
		Limit:      limit * 2, // Query more than needed for confidence filtering
		Project:    project,
		MemoryRoot: memoryRoot,
		MinScore:   minScore,
	}

	result, err := memory.Query(opts)
	if err != nil {
		// Graceful degradation for hook usage
		return nil
	}

	// Log retrieval for relevance measurement (Task 2: self-reinforcing learning)
	if hookInput != nil {
		var retrievalResults []memory.RetrievalResult
		for _, r := range result.Results {
			retrievalResults = append(retrievalResults, memory.RetrievalResult{
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
		logEntry := memory.RetrievalLogEntry{
			Timestamp:     time.Now().Format(time.RFC3339),
			Hook:          hookInput.HookEventName,
			Query:         queryText,
			Results:       retrievalResults,
			FilteredCount: result.FilteredCount,
			SessionID:     hookInput.SessionID,
			Metadata:      metadata,
		}
		// Best-effort logging - don't fail the query
		_ = memory.LogRetrieval(memoryRoot, logEntry)
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

	// Determine output tier
	tier := memory.TierCompact
	if args.Rich {
		tier = memory.TierFull
	} else if args.Curate {
		tier = memory.TierCurated
	}

	// Auto-enable curation for stdin-prompt when not explicitly set
	if args.StdinPrompt && !args.Curate && !args.Rich {
		tier = memory.TierCurated
	}

	// Downgrade TierCurated for PreToolUse hooks (must stay ONNX-only for speed)
	tier = memory.ResolveTier(tier, hookInput)

	// Create LLM extractor for curated tier
	var extractor memory.LLMExtractor
	if tier == memory.TierCurated {
		extractor = memory.NewLLMExtractor()
	}

	// Always markdown output
	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
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

type memoryDigestArgs struct {
	Since      string `targ:"flag,short=s,desc=Time window like 7d or 24h or 168h,default=168h"`
	Tier       string `targ:"flag,short=t,desc=Filter by tier: skill or embedding or claude_md"`
	FlagsOnly  bool   `targ:"flag,short=f,desc=Show only flags not full digest"`
	MaxEntries int    `targ:"flag,short=n,desc=Maximum number of entries to show"`
	MemoryRoot string `targ:"flag,desc=Memory root directory"`
}

func memoryDigest(args memoryDigestArgs) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = home + "/.claude/memory"
	}

	// Parse since duration
	sinceStr := args.Since
	if sinceStr == "" {
		sinceStr = "168h" // 7 days
	}
	since, err := time.ParseDuration(sinceStr)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", sinceStr, err)
	}

	opts := memory.DigestOptions{
		Since:      since,
		Tier:       args.Tier,
		FlagsOnly:  args.FlagsOnly,
		MaxEntries: args.MaxEntries,
	}

	digest, err := memory.ComputeDigest(opts, memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to compute digest: %w", err)
	}

	// If flags-only mode, only show flags
	if args.FlagsOnly {
		if len(digest.Flags) == 0 {
			fmt.Println("No flags detected")
			return nil
		}
		fmt.Println("Flags:")
		for _, flag := range digest.Flags {
			fmt.Printf("  ⚠ %s\n", flag)
		}
		return nil
	}

	// Show full digest
	output := memory.FormatDigest(digest)
	fmt.Print(output)
	return nil
}


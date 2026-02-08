package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/toejough/projctl/internal/memory"
)

type memoryLearnArgs struct {
	Message    string `targ:"flag,short=m,required,desc=Learning message to store"`
	Project    string `targ:"flag,short=p,desc=Project to tag the learning with"`
	Source     string `targ:"flag,short=s,desc=Source type: internal or external (default: internal)"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
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
		MemoryRoot: memoryRoot,
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

type memorySessionEndArgs struct {
	Project    string `targ:"flag,short=p,required,desc=Project name"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

func memorySessionEnd(args memorySessionEndArgs) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = home + "/.claude/memory"
	}

	opts := memory.SessionEndOpts{
		Project:    args.Project,
		MemoryRoot: memoryRoot,
	}

	result, err := memory.SessionEnd(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Session summary saved to: %s\n", result.FilePath)
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
	Text       string `targ:"positional,required,desc=Text to search for"`
	Limit      int    `targ:"flag,short=n,desc=Maximum number of results (default 5)"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

func memoryQuery(args memoryQueryArgs) error {
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
		limit = 5
	}

	opts := memory.QueryOpts{
		Text:       args.Text,
		Limit:      limit,
		MemoryRoot: memoryRoot,
	}

	result, err := memory.Query(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(result.Results) == 0 {
		fmt.Println("No similar memories found")
		return nil
	}

	for i, r := range result.Results {
		fmt.Printf("%d. (%.2f) %s\n", i+1, r.Score, r.Content)
	}

	return nil
}

type memoryPromoteArgs struct {
	MinRetrievals int    `targ:"flag,name=min-retrievals,desc=Minimum retrieval count (default 3)"`
	MinProjects   int    `targ:"flag,name=min-projects,desc=Minimum unique projects (default 2)"`
	MemoryRoot    string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

func memoryPromote(args memoryPromoteArgs) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = home + "/.claude/memory"
	}

	opts := memory.PromoteOpts{
		MemoryRoot:    memoryRoot,
		MinRetrievals: args.MinRetrievals,
		MinProjects:   args.MinProjects,
	}

	result, err := memory.Promote(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(result.Candidates) == 0 {
		fmt.Println("No candidates for promotion")
		return nil
	}

	fmt.Printf("Found %d candidates for promotion:\n", len(result.Candidates))
	for i, c := range result.Candidates {
		fmt.Printf("%d. [%d retrievals, %d projects] %s\n", i+1, c.RetrievalCount, c.UniqueProjects, c.Content)
	}

	return nil
}

type memoryDecayArgs struct {
	Factor     string `targ:"flag,desc=Decay factor (default 0.9)"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

func memoryDecay(args memoryDecayArgs) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = home + "/.claude/memory"
	}

	var factor float64
	if args.Factor != "" {
		var err error
		factor, err = strconv.ParseFloat(args.Factor, 64)
		if err != nil {
			return fmt.Errorf("invalid factor: %w", err)
		}
	}

	opts := memory.DecayOpts{
		MemoryRoot: memoryRoot,
		Factor:     factor,
	}

	result, err := memory.Decay(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Decayed %d entries (factor: %.2f)\n", result.EntriesAffected, result.Factor)
	return nil
}

type memoryPruneArgs struct {
	Threshold  string `targ:"flag,desc=Confidence threshold (default 0.1)"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

func memoryPrune(args memoryPruneArgs) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = home + "/.claude/memory"
	}

	var threshold float64
	if args.Threshold != "" {
		var err error
		threshold, err = strconv.ParseFloat(args.Threshold, 64)
		if err != nil {
			return fmt.Errorf("invalid threshold: %w", err)
		}
	}

	opts := memory.PruneOpts{
		MemoryRoot: memoryRoot,
		Threshold:  threshold,
	}

	result, err := memory.Prune(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Pruned: removed %d, retained %d (threshold: %.2f)\n", result.EntriesRemoved, result.EntriesRetained, result.Threshold)
	return nil
}

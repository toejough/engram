package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/toejough/projctl/internal/memory"
)

type memoryLearnArgs struct {
	Message    string `targ:"flag,short=m,required,desc=Learning message to store"`
	Project    string `targ:"flag,short=p,desc=Project to tag the learning with"`
	Source     string `targ:"flag,short=s,desc=Source type: internal or external (default: internal)"`
	Type       string `targ:"flag,short=t,desc=Memory type: correction or reflection (default: empty)"`
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
		Type:       args.Type,
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
	Project    string `targ:"flag,short=p,desc=Project name for retrieval tracking"`
	Verbose    bool   `targ:"flag,short=v,desc=Show detailed scoring info (method and memory type)"`
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
		Project:    args.Project,
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

	if args.Verbose {
		method := "vector"
		if result.UsedHybridSearch {
			method = "hybrid (vector+BM25)"
		}
		fmt.Printf("Search method: %s\n", method)
		fmt.Printf("BM25 available: %v\n\n", result.BM25Enabled)
	}

	for i, r := range result.Results {
		if args.Verbose && r.MemoryType != "" {
			fmt.Printf("%d. (%.2f) [%s] %s\n", i+1, r.Score, r.MemoryType, r.Content)
		} else {
			fmt.Printf("%d. (%.2f) %s\n", i+1, r.Score, r.Content)
		}
	}

	return nil
}


package main

import (
	"fmt"
	"os"
	"time"

	"github.com/toejough/projctl/internal/issue"
	"github.com/toejough/projctl/internal/retro"
)

type retroExtractArgs struct {
	Dir         string `targ:"flag,short=d,desc=Project directory containing retro.md"`
	RepoDir     string `targ:"flag,short=r,desc=Repository directory for issues.md (default: current)"`
	MinPriority string `targ:"flag,short=p,desc=Minimum priority to extract (High, Medium, Low)"`
	DryRun      bool   `targ:"flag,desc=Print what would be created without creating"`
}

func retroExtract(args retroExtractArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	repoDir := args.RepoDir
	if repoDir == "" {
		var err error
		repoDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	minPriority := args.MinPriority
	if minPriority == "" {
		minPriority = "Medium" // Default to Medium and above
	}

	// Extract recommendations
	recommendations, err := retro.ExtractRecommendations(dir)
	if err != nil {
		return fmt.Errorf("failed to extract recommendations: %w", err)
	}

	// Filter by priority
	filtered := retro.FilterByPriority(recommendations, minPriority)

	// Extract open questions
	questions, err := retro.ExtractOpenQuestions(dir)
	if err != nil {
		return fmt.Errorf("failed to extract open questions: %w", err)
	}

	if len(filtered) == 0 && len(questions) == 0 {
		fmt.Println("No items to extract from retrospective")
		return nil
	}

	fmt.Printf("Found %d recommendations (%s+ priority) and %d open questions\n\n",
		len(filtered), minPriority, len(questions))

	// Create issues for recommendations
	for _, rec := range filtered {
		body := fmt.Sprintf("**From retrospective:** %s\n\n**Action:** %s\n\n**Rationale:** %s\n\n**Traces to:** Retrospective %s",
			rec.ID, rec.Action, rec.Rationale, rec.ID)

		if args.DryRun {
			fmt.Printf("[DRY RUN] Would create issue:\n")
			fmt.Printf("  Title: %s\n", rec.Title)
			fmt.Printf("  Priority: %s\n", rec.Priority)
			fmt.Printf("  Body: %s...\n\n", truncate(body, 100))
		} else {
			created, err := issue.Create(repoDir, issue.CreateOpts{
				Title:    rec.Title,
				Priority: rec.Priority,
				Body:     body,
			}, time.Now)
			if err != nil {
				return fmt.Errorf("failed to create issue for %s: %w", rec.ID, err)
			}
			fmt.Printf("Created %s: %s (from %s)\n", created.ID, created.Title, rec.ID)
		}
	}

	// Create issues for open questions
	for _, q := range questions {
		body := fmt.Sprintf("**From retrospective:** %s\n\n**Context:** %s\n\n**Decision needed before:** %s\n\n**Traces to:** Retrospective %s",
			q.ID, q.Context, q.DecisionNeeded, q.ID)

		title := fmt.Sprintf("[Decision needed] %s", q.Title)

		if args.DryRun {
			fmt.Printf("[DRY RUN] Would create issue:\n")
			fmt.Printf("  Title: %s\n", title)
			fmt.Printf("  Priority: Medium\n")
			fmt.Printf("  Body: %s...\n\n", truncate(body, 100))
		} else {
			created, err := issue.Create(repoDir, issue.CreateOpts{
				Title:    title,
				Priority: "Medium",
				Body:     body,
			}, time.Now)
			if err != nil {
				return fmt.Errorf("failed to create issue for %s: %w", q.ID, err)
			}
			fmt.Printf("Created %s: %s (from %s)\n", created.ID, created.Title, q.ID)
		}
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

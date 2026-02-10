package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/toejough/projctl/internal/memory"
)

type memorySkillListArgs struct {
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

func memorySkillList(args memorySkillListArgs) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	db, err := memory.OpenSkillDB(memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	skills, err := memory.ListSkillsPublic(db)
	if err != nil {
		return fmt.Errorf("failed to list skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No generated skills found")
		return nil
	}

	fmt.Printf("Generated Skills (%d):\n\n", len(skills))
	for _, s := range skills {
		confidence := s.Alpha / (s.Alpha + s.Beta) * 100
		fmt.Printf("  %s\n", s.Slug)
		fmt.Printf("    Theme:       %s\n", s.Theme)
		fmt.Printf("    Confidence:  %.0f%%\n", confidence)
		fmt.Printf("    Utility:     %.2f\n", s.Utility)
		fmt.Printf("    Retrievals:  %d\n", s.RetrievalCount)
		fmt.Println()
	}

	return nil
}

type memorySkillFeedbackArgs struct {
	Skill      string `targ:"flag,required,desc=Skill slug to provide feedback for"`
	Success    bool   `targ:"flag,desc=Record positive feedback (increases confidence)"`
	Failure    bool   `targ:"flag,desc=Record negative feedback (decreases confidence)"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

func memorySkillFeedback(args memorySkillFeedbackArgs) error {
	if !args.Success && !args.Failure {
		return fmt.Errorf("must specify --success or --failure")
	}
	if args.Success && args.Failure {
		return fmt.Errorf("cannot specify both --success and --failure")
	}

	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	db, err := memory.OpenSkillDB(memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	err = memory.RecordSkillFeedback(db, args.Skill, args.Success)
	if err != nil {
		return fmt.Errorf("failed to record feedback: %w", err)
	}

	action := "negative"
	if args.Success {
		action = "positive"
	}
	fmt.Printf("Recorded %s feedback for skill %q\n", action, args.Skill)
	return nil
}

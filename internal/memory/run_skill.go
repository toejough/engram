package memory

import (
	"errors"
	"fmt"
	"path/filepath"
)

// RunSkillFeedback records feedback for a generated skill.
func RunSkillFeedback(args SkillFeedbackArgs, homeDir string) error {
	if !args.Success && !args.Failure {
		return errors.New("must specify --success or --failure")
	}

	if args.Success && args.Failure {
		return errors.New("cannot specify both --success and --failure")
	}

	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	db, err := OpenSkillDB(memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	if err := RecordSkillFeedback(db, args.Skill, args.Success); err != nil {
		return fmt.Errorf("failed to record feedback: %w", err)
	}

	action := "negative"
	if args.Success {
		action = "positive"
	}

	fmt.Printf("Recorded %s feedback for skill %q\n", action, args.Skill)

	return nil
}

// RunSkillList lists all generated skills.
func RunSkillList(args SkillListArgs, homeDir string) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	db, err := OpenSkillDB(memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	skills, err := ListSkillsPublic(db)
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

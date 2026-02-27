package main

import (
	"os"

	"github.com/toejough/projctl/internal/memory"
)

func memorySkillFeedback(args memory.SkillFeedbackArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunSkillFeedback(args, home)
}

func memorySkillList(args memory.SkillListArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunSkillList(args, home)
}

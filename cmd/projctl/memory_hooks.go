package main

import (
	"os"

	"github.com/toejough/projctl/internal/memory"
)

func memoryHooksCheckClaudeMD(args memory.HooksCheckClaudeMDArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunHooksCheckClaudeMD(args, home)
}

func memoryHooksCheckEmbedding(args memory.HooksCheckEmbeddingArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunHooksCheckEmbedding(args, home, os.Stdin)
}

func memoryHooksCheckSkill(args memory.HooksCheckSkillArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunHooksCheckSkill(args, home)
}

func memoryHooksInstall(args memory.HooksInstallArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunHooksInstall(args, home)
}

func memoryHooksShow(args memory.HooksShowArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunHooksShow(args, home)
}

func memoryHooksStats(args memory.HooksStatsArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunHooksStats(args, home)
}

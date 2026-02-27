package main

import (
	"os"

	"github.com/toejough/projctl/internal/memory"
)

func memoryDecide(args memory.DecideArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunDecide(args, home)
}

func memoryDigest(args memory.DigestArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunDigest(args, home)
}

func memoryGrep(args memory.GrepArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunGrep(args, home)
}

func memoryLearn(args memory.LearnArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunLearn(args, home)
}

func memoryQuery(args memory.QueryArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunQuery(args, home, os.Stdin)
}

package main

import (
	"os"

	"github.com/toejough/projctl/internal/memory"
)

func memoryDiagnose(args memory.DiagnoseArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunDiagnose(args, home)
}

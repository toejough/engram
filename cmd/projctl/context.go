package main

import (
	"fmt"
	"os"

	"github.com/toejough/projctl/internal/config"
	"github.com/toejough/projctl/internal/context"
	"github.com/toejough/projctl/internal/log"
)

type contextCheckArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func contextCheck(args contextCheckArgs) error {
	// Load config for thresholds
	homeDir, _ := os.UserHomeDir()
	cfg, err := config.Load(args.Dir, homeDir, &osConfigFS{})
	if err != nil {
		cfg = config.Default()
	}

	// Use config thresholds or defaults
	thresholds := context.BudgetThresholds{
		Warning: cfg.Budget.WarningTokens,
		Limit:   cfg.Budget.LimitTokens,
	}
	if thresholds.Warning == 0 {
		thresholds.Warning = 80000 // Default
	}
	if thresholds.Limit == 0 {
		thresholds.Limit = 90000 // Default
	}

	result, err := context.CheckBudget(args.Dir, thresholds, log.RealFS{})
	if err != nil {
		return err
	}

	fmt.Println(result.Message)

	// Return with appropriate exit code
	if result.ExitCode != 0 {
		os.Exit(result.ExitCode)
	}

	return nil
}

// osConfigFS implements config.ConfigFS using real filesystem operations.
type osConfigFS struct{}

func (f *osConfigFS) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (f *osConfigFS) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

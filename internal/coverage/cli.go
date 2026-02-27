package coverage

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/toejough/projctl/internal/config"
)

// AnalyzeArgs holds arguments for the coverage analyze command.
type AnalyzeArgs struct {
	Dir string `targ:"flag,short=d,desc=Project directory (default: current)"`
}

// RealCoverageFS implements CoverageFS using the real file system.
type RealCoverageFS struct{}

func (r *RealCoverageFS) DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info == nil {
		return false
	}

	return info.IsDir()
}

func (r *RealCoverageFS) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *RealCoverageFS) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (r *RealCoverageFS) Walk(root string, fn func(path string, isDir bool) error) error {
	return walkDir(root, fn)
}

// ReportArgs holds arguments for the coverage report command.
type ReportArgs struct {
	Dir string `targ:"flag,short=d,desc=Project directory (default: current)"`
}

// RunAnalyze analyzes test coverage data.
func RunAnalyze(args AnalyzeArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	cfgFS := &realConfigFS{}

	cfg, err := config.Load(dir, homeDir, cfgFS)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	covFS := &RealCoverageFS{}

	result, err := Analyze(dir, cfg, covFS)
	if err != nil {
		return fmt.Errorf("coverage analysis failed: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}

	fmt.Println(string(data))

	return nil
}

// RunReport generates a coverage report.
func RunReport(args ReportArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	cfgFS := &realConfigFS{}

	cfg, err := config.Load(dir, homeDir, cfgFS)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	covFS := &RealCoverageFS{}

	result, err := Analyze(dir, cfg, covFS)
	if err != nil {
		return fmt.Errorf("coverage analysis failed: %w", err)
	}

	fmt.Println("Coverage Analysis Report")
	fmt.Println("========================")
	fmt.Printf("Documented items:  %d\n", result.DocumentedCount)
	fmt.Printf("Inferred items:    %d\n", result.InferredCount)
	fmt.Printf("Coverage ratio:    %.1f%%\n", result.CoverageRatio*100)
	fmt.Printf("Recommendation:    %s\n", result.Recommendation)
	fmt.Println()

	switch result.Recommendation {
	case "preserve":
		fmt.Println("This codebase has good documentation coverage.")
		fmt.Println("Use /project adopt to analyze and enhance existing docs.")
	case "migrate":
		fmt.Println("This codebase has limited documentation.")
		fmt.Println("Consider using /project new to start fresh with structured documentation.")
	case "evaluate":
		fmt.Println("This codebase has partial documentation.")
		fmt.Println("Review the existing docs to decide between adopt and new modes.")
	}

	return nil
}

// realConfigFS implements config.ConfigFS using the real file system.
type realConfigFS struct{}

func (r *realConfigFS) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *realConfigFS) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func walkDir(dir string, fn func(path string, isDir bool) error) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := dir + "/" + entry.Name()
		isDir := entry.IsDir()

		err := fn(path, isDir)
		if err != nil {
			if err.Error() == "skip" {
				continue
			}

			return err
		}

		if isDir {
			err := walkDir(path, fn)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

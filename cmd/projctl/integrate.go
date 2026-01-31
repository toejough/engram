package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/toejough/projctl/internal/integrate"
)

// realMergeFS implements integrate.MergeFS using the real file system.
type realMergeFS struct{}

func (r *realMergeFS) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *realMergeFS) WriteFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func (r *realMergeFS) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *realMergeFS) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (r *realMergeFS) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

type integrateMergeArgs struct {
	Dir     string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Project string `targ:"flag,short=p,desc=Per-project name to merge,required"`
	JSON    bool   `targ:"flag,short=j,desc=Output result as JSON"`
}

func integrateMerge(args integrateMergeArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	fs := &realMergeFS{}
	result, err := integrate.Merge(dir, args.Project, fs)
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	if args.JSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode result: %w", err)
		}
		fmt.Println(string(data))
	} else {
		fmt.Println("Integration Merge Complete")
		fmt.Println("==========================")
		fmt.Println(result.Summary)
		fmt.Println()
		if result.RequirementsAdded > 0 {
			fmt.Printf("  Requirements: +%d\n", result.RequirementsAdded)
		}
		if result.DesignAdded > 0 {
			fmt.Printf("  Design:       +%d\n", result.DesignAdded)
		}
		if result.ArchitectureAdded > 0 {
			fmt.Printf("  Architecture: +%d\n", result.ArchitectureAdded)
		}
		if result.TasksAdded > 0 {
			fmt.Printf("  Tasks:        +%d\n", result.TasksAdded)
		}
		if result.IDsRenumbered > 0 {
			fmt.Printf("\n  IDs renumbered: %d\n", result.IDsRenumbered)
		}
		if result.LinksUpdated > 0 {
			fmt.Printf("  Links updated:  %d\n", result.LinksUpdated)
		}
	}

	return nil
}

type integrateFeaturesArgs struct {
	Dir  string `targ:"flag,short=d,desc=Project directory (default: current)"`
	JSON bool   `targ:"flag,short=j,desc=Output result as JSON"`
}

func integrateFeatures(args integrateFeaturesArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	docsDir := filepath.Join(dir, "docs")

	fs := &realMergeFS{}
	result, err := integrate.MergeFeatureFiles(docsDir, fs)
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	if args.JSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode result: %w", err)
		}
		fmt.Println(string(data))
	} else {
		if result.RequirementsAdded == 0 && result.DesignAdded == 0 && result.ArchitectureAdded == 0 {
			fmt.Println("No feature files to merge")
			return nil
		}

		fmt.Println("Feature Files Merged")
		fmt.Println("====================")
		fmt.Println(result.Summary)
		if result.IDsRenumbered > 0 {
			fmt.Printf("\n  IDs renumbered: %d\n", result.IDsRenumbered)
		}
	}

	return nil
}

type integrateCleanupArgs struct {
	Dir     string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Project string `targ:"flag,short=p,desc=Per-project name to clean up,required"`
}

func integrateCleanup(args integrateCleanupArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	perProjectDir := filepath.Join(dir, "docs", "projects", args.Project)

	fs := &realMergeFS{}
	if !fs.FileExists(perProjectDir) {
		return fmt.Errorf("per-project directory does not exist: %s", perProjectDir)
	}

	if err := fs.RemoveAll(perProjectDir); err != nil {
		return fmt.Errorf("failed to remove per-project directory: %w", err)
	}

	fmt.Printf("Cleaned up per-project directory: %s\n", perProjectDir)
	return nil
}

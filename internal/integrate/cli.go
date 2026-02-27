package integrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CleanupArgs holds arguments for the integrate cleanup command.
type CleanupArgs struct {
	Dir     string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Project string `targ:"flag,short=p,desc=Per-project name to clean up,required"`
}

// FeaturesArgs holds arguments for the integrate features command.
type FeaturesArgs struct {
	Dir  string `targ:"flag,short=d,desc=Project directory (default: current)"`
	JSON bool   `targ:"flag,short=j,desc=Output result as JSON"`
}

// MergeArgs holds arguments for the integrate merge command.
type MergeArgs struct {
	Dir     string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Project string `targ:"flag,short=p,desc=Per-project name to merge,required"`
	JSON    bool   `targ:"flag,short=j,desc=Output result as JSON"`
}

// RealMergeFS implements MergeFS using the real file system.
type RealMergeFS struct{}

func (r *RealMergeFS) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *RealMergeFS) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

func (r *RealMergeFS) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (r *RealMergeFS) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (r *RealMergeFS) WriteFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// RunCleanup cleans up after integration.
func RunCleanup(args CleanupArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	perProjectDir := filepath.Join(dir, ".claude", "projects", args.Project)

	fs := &RealMergeFS{}
	if !fs.FileExists(perProjectDir) {
		return fmt.Errorf("per-project directory does not exist: %s", perProjectDir)
	}

	err := fs.RemoveAll(perProjectDir)
	if err != nil {
		return fmt.Errorf("failed to remove per-project directory: %w", err)
	}

	fmt.Printf("Cleaned up per-project directory: %s\n", perProjectDir)

	return nil
}

// RunFeatures merges feature-specific docs into top-level.
func RunFeatures(args FeaturesArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	docsDir := filepath.Join(dir, "docs")

	fs := &RealMergeFS{}

	result, err := MergeFeatureFiles(docsDir, fs)
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

// RunMerge merges per-project docs into top-level.
func RunMerge(args MergeArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	fs := &RealMergeFS{}

	result, err := Merge(dir, args.Project, fs)
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

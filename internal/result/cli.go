package result

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// CollectArgs holds arguments for the result collect command.
type CollectArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Project directory"`
	Tasks  string `targ:"flag,short=t,required,desc=Comma-separated task IDs (e.g. TASK-001;TASK-002)"`
	Skill  string `targ:"flag,short=s,desc=Skill name (default: tdd-red)"`
	Format string `targ:"flag,short=f,desc=Output format: text (default) or json"`
}

// ValidateArgs holds arguments for the result validate command.
type ValidateArgs struct {
	File   string `targ:"flag,short=f,required,desc=Path to result.toml file"`
	Format string `targ:"flag,desc=Output format: text (default) or json"`
}

// ValidationOutput is the JSON output format for validation results.
type ValidationOutput struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors,omitempty"`
	File    string   `json:"file"`
	Status  bool     `json:"status,omitempty"`
	Outputs int      `json:"outputs_count,omitempty"`
}

// RunCollect collects and merges results from parallel tasks.
func RunCollect(args CollectArgs) error {
	skill := args.Skill
	if skill == "" {
		skill = "tdd-red"
	}

	tasks := strings.Split(args.Tasks, ",")
	for i := range tasks {
		tasks[i] = strings.TrimSpace(tasks[i])
	}

	collected, err := Collect(args.Dir, tasks, skill, RealFS{})
	if err != nil {
		return err
	}

	if args.Format == "json" {
		output := struct {
			Succeeded      int      `json:"succeeded"`
			Failed         int      `json:"failed"`
			Total          int      `json:"total"`
			FailedTasks    []string `json:"failed_tasks,omitempty"`
			LearningsCount int      `json:"learnings_count"`
		}{
			Succeeded:      collected.Succeeded,
			Failed:         collected.Failed,
			Total:          collected.Total,
			FailedTasks:    collected.FailedTasks,
			LearningsCount: len(collected.Learnings),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(output)
	}

	fmt.Printf("%d/%d tasks complete", collected.Succeeded, collected.Total)

	if collected.Failed > 0 {
		fmt.Printf(", %d failed", collected.Failed)
	}

	fmt.Println()

	if len(collected.FailedTasks) > 0 {
		fmt.Printf("Failed tasks: %s\n", strings.Join(collected.FailedTasks, ", "))
	}

	if len(collected.Learnings) > 0 {
		fmt.Printf("Learnings: %d collected\n", len(collected.Learnings))
	}

	if collected.Failed > 0 {
		os.Exit(1)
	}

	return nil
}

// RunValidate validates a result.toml file.
func RunValidate(args ValidateArgs) error {
	return RunValidateCore(args, os.Exit)
}

// RunValidateCore is the testable core of RunValidate, accepting an injectable exit function.
func RunValidateCore(args ValidateArgs, exit func(int)) error {
	if args.File == "" {
		return errors.New("--file is required")
	}

	data, err := os.ReadFile(args.File)
	if err != nil {
		if args.Format == "json" {
			out := ValidationOutput{
				Valid:  false,
				Errors: []string{fmt.Sprintf("failed to read file: %v", err)},
				File:   args.File,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(out)

			exit(1)

			return nil
		}

		return fmt.Errorf("failed to read file: %w", err)
	}

	r, err := Parse(data)
	if err != nil {
		if args.Format == "json" {
			out := ValidationOutput{
				Valid:  false,
				Errors: []string{err.Error()},
				File:   args.File,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(out)

			exit(1)

			return nil
		}

		fmt.Fprintf(os.Stderr, "Validation failed: %s\n", err)
		exit(1)

		return nil
	}

	if args.Format == "json" {
		out := ValidationOutput{
			Valid:   true,
			File:    args.File,
			Status:  r.Status.Success,
			Outputs: len(r.Outputs.FilesModified),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(out)
	}

	fmt.Printf("Valid result file: %s\n", args.File)
	fmt.Printf("  Status: success=%v\n", r.Status.Success)
	fmt.Printf("  Outputs: %d files modified\n", len(r.Outputs.FilesModified))
	fmt.Printf("  Decisions: %d\n", len(r.Decisions))
	fmt.Printf("  Learnings: %d\n", len(r.Learnings))

	return nil
}

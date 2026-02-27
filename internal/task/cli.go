package task

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DepsArgs holds arguments for the tasks deps command.
type DepsArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Project directory"`
	Format string `targ:"flag,short=f,desc=Output format: json (default) or dot"`
}

// ParallelArgs holds arguments for the tasks parallel command.
type ParallelArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Project directory"`
	Format string `targ:"flag,short=f,desc=Output format: text (default) or json"`
}

// ValidateArgs holds arguments for the task validate command.
type ValidateArgs struct {
	Dir                  string `targ:"flag,short=d,required,desc=Project directory"`
	Task                 string `targ:"flag,short=t,required,desc=Task ID (e.g. TASK-001)"`
	ManualVisualVerified bool   `targ:"flag,desc=I manually verified visual correctness (bypass MCP requirement)"`
}

// RunDeps parses task dependencies and outputs graph.
func RunDeps(args DepsArgs) error {
	graph, err := ParseDependencies(args.Dir)
	if err != nil {
		return err
	}

	if graph.HasCycle() {
		return fmt.Errorf("cycle detected: %s", strings.Join(graph.CyclePath(), " → "))
	}

	if args.Format == "dot" {
		fmt.Println("digraph tasks {")
		fmt.Println("  rankdir=LR;")

		for _, t := range graph.Tasks {
			fmt.Printf("  \"%s\";\n", t)
		}

		for t, deps := range graph.Deps {
			for _, dep := range deps {
				fmt.Printf("  \"%s\" -> \"%s\";\n", t, dep)
			}
		}

		fmt.Println("}")

		return nil
	}

	output := struct {
		Tasks []string            `json:"tasks"`
		Deps  map[string][]string `json:"dependencies"`
		Roots []string            `json:"roots"`
	}{
		Tasks: graph.Tasks,
		Deps:  graph.Deps,
		Roots: graph.Roots(),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))

	return nil
}

// RunParallel lists tasks that can run in parallel.
func RunParallel(args ParallelArgs) error {
	parallel, err := Parallel(args.Dir)
	if err != nil {
		return err
	}

	if args.Format == "json" {
		data, err := json.MarshalIndent(parallel, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(data))

		return nil
	}

	if len(parallel) == 0 {
		fmt.Println("No parallel tasks available")
		return nil
	}

	fmt.Println("Tasks that can run in parallel:")

	for _, t := range parallel {
		fmt.Printf("  %s\n", t)
	}

	return nil
}

// RunValidate validates a task against requirements.
func RunValidate(args ValidateArgs) error {
	result := ValidateWithOpts(args.Dir, args.Task, ValidateOpts{
		ManualVisualVerified: args.ManualVisualVerified,
	})

	if !result.Valid {
		return fmt.Errorf("validation failed: %s", result.Error)
	}

	if result.Warning != "" {
		fmt.Printf("Warning: %s\n", result.Warning)
	}

	fmt.Println("Task validation passed")

	return nil
}

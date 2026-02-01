package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/toejough/projctl/internal/task"
)

type tasksDepsArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Project directory"`
	Format string `targ:"flag,short=f,desc=Output format: json (default) or dot"`
}

func tasksDeps(args tasksDepsArgs) error {
	graph, err := task.ParseDependencies(args.Dir)
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

	// JSON output (default)
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

type tasksParallelArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Project directory"`
	Format string `targ:"flag,short=f,desc=Output format: text (default) or json"`
}

func tasksParallel(args tasksParallelArgs) error {
	parallel, err := task.Parallel(args.Dir)
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

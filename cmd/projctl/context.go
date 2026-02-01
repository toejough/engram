package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/toejough/projctl/internal/config"
	"github.com/toejough/projctl/internal/context"
)

type contextWriteArgs struct {
	Dir          string `targ:"flag,short=d,required,desc=Project directory"`
	Task         string `targ:"flag,short=t,required,desc=Task ID (e.g. TASK-004)"`
	Skill        string `targ:"flag,short=s,required,desc=Skill name (e.g. tdd-red)"`
	File         string `targ:"flag,short=f,required,desc=Path to TOML context file"`
	NoRouting    bool   `targ:"flag,desc=Skip adding routing section"`
	InjectMemory string `targ:"flag,desc=Query memory and inject top results into context"`
}

func contextWrite(args contextWriteArgs) error {
	var path string
	var err error

	if args.NoRouting {
		path, err = context.Write(args.Dir, args.Task, args.Skill, args.File)
	} else {
		// Load config for routing
		homeDir, _ := os.UserHomeDir()
		cfg, cfgErr := config.Load(args.Dir, homeDir, &osConfigFS{})
		if cfgErr != nil {
			// Fall back to defaults if config loading fails
			cfg = config.Default()
		}

		routing := context.RoutingConfig{
			Simple:  cfg.Routing.Simple,
			Medium:  cfg.Routing.Medium,
			Complex: cfg.Routing.Complex,
		}

		// Use memory injection if --inject-memory is specified
		if args.InjectMemory != "" {
			memoryRoot := fmt.Sprintf("%s/.claude/memory", homeDir)
			path, err = context.WriteWithRoutingAndMemory(args.Dir, args.Task, args.Skill, args.File, routing, cfg.Routing.SkillComplexity, memoryRoot, args.InjectMemory)
		} else {
			// WriteWithRoutingAndMemory handles auto-injection for specific skills
			memoryRoot := fmt.Sprintf("%s/.claude/memory", homeDir)
			path, err = context.WriteWithRoutingAndMemory(args.Dir, args.Task, args.Skill, args.File, routing, cfg.Routing.SkillComplexity, memoryRoot, "")
		}
	}

	if err != nil {
		return err
	}

	fmt.Printf("Context written: %s\n", path)

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

type contextReadArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Project directory"`
	Task   string `targ:"flag,short=t,required,desc=Task ID (e.g. TASK-004)"`
	Skill  string `targ:"flag,short=s,required,desc=Skill name (e.g. tdd-red)"`
	Result bool   `targ:"flag,short=r,desc=Read result file instead of context file"`
	Format string `targ:"flag,desc=Output format: toml (default) or json"`
}

func contextRead(args contextReadArgs) error {
	content, err := context.Read(args.Dir, args.Task, args.Skill, args.Result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if args.Format == "json" {
		var data any
		if _, err := toml.Decode(content, &data); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing TOML: %v\n", err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	fmt.Print(content)
	return nil
}

type contextWriteParallelArgs struct {
	Dir      string `targ:"flag,short=d,required,desc=Project directory"`
	Tasks    string `targ:"flag,short=t,required,desc=Comma-separated task IDs (e.g. TASK-001,TASK-002)"`
	Skill    string `targ:"flag,short=s,desc=Skill name (default: tdd-red)"`
	Template string `targ:"flag,short=f,required,desc=Path to template TOML context file"`
}

func contextWriteParallel(args contextWriteParallelArgs) error {
	skill := args.Skill
	if skill == "" {
		skill = "tdd-red" // Default skill for pending implementation tasks
	}

	tasks := strings.Split(args.Tasks, ",")
	for i := range tasks {
		tasks[i] = strings.TrimSpace(tasks[i])
	}

	paths, err := context.WriteParallel(args.Dir, tasks, skill, args.Template)
	if err != nil {
		return err
	}

	fmt.Printf("Created %d context files:\n", len(paths))
	for _, path := range paths {
		fmt.Printf("  %s\n", path)
	}

	return nil
}

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

	result, err := context.CheckBudget(args.Dir, thresholds)
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

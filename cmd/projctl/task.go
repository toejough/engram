package main

import (
	"fmt"

	"github.com/toejough/projctl/internal/task"
)

type taskValidateArgs struct {
	Dir                  string `targ:"flag,short=d,required,desc=Project directory"`
	Task                 string `targ:"flag,short=t,required,desc=Task ID (e.g. TASK-001)"`
	ManualVisualVerified bool   `targ:"flag,desc=I manually verified visual correctness (bypass MCP requirement)"`
}

func taskValidate(args taskValidateArgs) error {
	result := task.ValidateWithOpts(args.Dir, args.Task, task.ValidateOpts{
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

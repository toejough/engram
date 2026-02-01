package main

import (
	"fmt"

	"github.com/toejough/projctl/internal/task"
)

type taskValidateArgs struct {
	Dir  string `targ:"flag,short=d,required,desc=Project directory"`
	Task string `targ:"flag,short=t,required,desc=Task ID (e.g. TASK-001)"`
}

func taskValidate(args taskValidateArgs) error {
	result := task.Validate(args.Dir, args.Task)

	if !result.Valid {
		return fmt.Errorf("validation failed: %s", result.Error)
	}

	fmt.Println("Task validation passed")

	return nil
}

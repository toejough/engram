package main

import "github.com/toejough/projctl/internal/task"

func taskValidate(args task.ValidateArgs) error {
	return task.RunValidate(args)
}

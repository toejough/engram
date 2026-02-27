package main

import "github.com/toejough/projctl/internal/task"

func tasksDeps(args task.DepsArgs) error {
	return task.RunDeps(args)
}

func tasksParallel(args task.ParallelArgs) error {
	return task.RunParallel(args)
}

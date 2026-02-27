package main

import "github.com/toejough/projctl/internal/conflict"

func conflictCheck(args conflict.CheckArgs) error {
	return conflict.RunCheck(args)
}

func conflictCreate(args conflict.CreateArgs) error {
	return conflict.RunCreate(args)
}

func conflictList(args conflict.ListArgs) error {
	return conflict.RunList(args)
}

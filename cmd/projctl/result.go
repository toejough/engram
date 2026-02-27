package main

import "github.com/toejough/projctl/internal/result"

func resultCollect(args result.CollectArgs) error {
	return result.RunCollect(args)
}

func resultValidate(args result.ValidateArgs) error {
	return result.RunValidate(args)
}

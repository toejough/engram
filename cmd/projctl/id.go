package main

import "github.com/toejough/projctl/internal/id"

func idNext(args id.NextArgs) error {
	return id.RunNext(args)
}

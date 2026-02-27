package main

import "github.com/toejough/projctl/internal/territory"

func territoryMap(args territory.MapArgs) error {
	return territory.RunMap(args)
}

func territoryShow(args territory.ShowArgs) error {
	return territory.RunShow(args)
}

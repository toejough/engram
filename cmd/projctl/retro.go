package main

import "github.com/toejough/projctl/internal/retro"

func retroExtract(args retro.ExtractArgs) error {
	return retro.RunExtract(args)
}

package main

import "github.com/toejough/projctl/internal/config"

func configGet(args config.GetArgs) error {
	return config.RunGet(args)
}

func configInit(args config.InitArgs) error {
	return config.RunInit(args)
}

func configPath(args config.PathArgs) error {
	return config.RunPath(args)
}

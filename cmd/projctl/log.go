package main

import "github.com/toejough/projctl/internal/log"

func logRead(args log.ReadArgs) error {
	return log.RunRead(args)
}

func logWrite(args log.WriteArgs) error {
	return log.RunWrite(args)
}

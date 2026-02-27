package main

import ctx "github.com/toejough/projctl/internal/context"

func contextCheck(args ctx.CheckArgs) error {
	return ctx.RunCheck(args)
}

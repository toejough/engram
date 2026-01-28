package main

import "github.com/toejough/targ"

func main() {
	targ.Main(
		targ.Group("state",
			targ.Targ(stateInit).Name("init"),
			targ.Targ(stateGet).Name("get"),
			targ.Targ(stateTransition).Name("transition"),
		),
		targ.Group("log",
			targ.Targ(logWrite).Name("write"),
		),
		targ.Group("trace",
			targ.Targ(traceAdd).Name("add"),
			targ.Targ(traceValidate).Name("validate"),
			targ.Targ(traceValidateV2).Name("validate-v2"),
			targ.Targ(traceImpact).Name("impact"),
		),
		targ.Group("conflict",
			targ.Targ(conflictCreate).Name("create"),
			targ.Targ(conflictCheck).Name("check"),
			targ.Targ(conflictList).Name("list"),
		),
		targ.Group("context",
			targ.Targ(contextWrite).Name("write"),
			targ.Targ(contextRead).Name("read"),
		),
		targ.Group("screenshot",
			targ.Targ(screenshotDiff).Name("diff"),
		),
	)
}

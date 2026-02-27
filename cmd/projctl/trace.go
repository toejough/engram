package main

import "github.com/toejough/projctl/internal/trace"

func tracePromote(args trace.PromoteArgs) error {
	return trace.RunPromote(args)
}

func traceRepair(args trace.RepairArgs) error {
	return trace.RunRepair(args)
}

func traceShow(args trace.ShowArgs) error {
	return trace.RunShow(args)
}

func traceValidateArtifacts(args trace.ValidateArtifactsArgs) error {
	return trace.RunValidateArtifacts(args)
}

package main

import "github.com/toejough/projctl/internal/corrections"

func correctionsAnalyze(args corrections.AnalyzeArgs) error {
	return corrections.RunAnalyze(args)
}

func correctionsCount(args corrections.CountArgs) error {
	return corrections.RunCount(args)
}

func correctionsLog(args corrections.LogArgs) error {
	return corrections.RunLog(args)
}

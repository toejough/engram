package main

import "github.com/toejough/projctl/internal/coverage"

func coverageAnalyze(args coverage.AnalyzeArgs) error {
	return coverage.RunAnalyze(args)
}

func coverageReport(args coverage.ReportArgs) error {
	return coverage.RunReport(args)
}

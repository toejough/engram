package main

import "github.com/toejough/projctl/internal/usage"

func usageCheck(args usage.CheckArgs) error {
	return usage.RunCheck(args)
}

func usageReport(args usage.ReportArgs) error {
	return usage.RunReport(args)
}

package main

import "github.com/toejough/projctl/internal/integrate"

func integrateCleanup(args integrate.CleanupArgs) error {
	return integrate.RunCleanup(args)
}

func integrateFeatures(args integrate.FeaturesArgs) error {
	return integrate.RunFeatures(args)
}

func integrateMerge(args integrate.MergeArgs) error {
	return integrate.RunMerge(args)
}

package main

import "github.com/toejough/projctl/internal/refactor"

func refactorCapabilities(args refactor.CapabilitiesArgs) error {
	return refactor.RunCapabilities(args)
}

func refactorExtractFunction(args refactor.ExtractFunctionArgs) error {
	return refactor.RunExtractFunction(args)
}

func refactorRename(args refactor.RenameArgs) error {
	return refactor.RunRename(args)
}

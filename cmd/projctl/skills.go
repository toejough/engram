package main

import "github.com/toejough/projctl/internal/skills"

func skillsDocs(args skills.DocsArgs) error {
	return skills.RunDocs(args)
}

func skillsInstall(args skills.InstallArgs) error {
	return skills.RunInstall(args)
}

func skillsList(args skills.ListArgs) error {
	return skills.RunList(args)
}

func skillsStatus(args skills.StatusArgs) error {
	return skills.RunStatus(args)
}

func skillsUninstall(args skills.UninstallArgs) error {
	return skills.RunUninstall(args)
}

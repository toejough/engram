//go:build targ

package projctl

import "github.com/toejough/targ"

func init() {
	targ.Register(InstallSkills, InstallProjctl, Install, Status)
}

// InstallSkills runs: projctl skills install
var InstallSkills = targ.Targ("projctl skills install").Name("install-skills")

// InstallProjctl runs: go install ./cmd/projctl
var InstallProjctl = targ.Targ("go install ./cmd/projctl").Name("install-projctl")

// Install runs: targ install-skills install-projctl
var Install = targ.Targ("targ install-skills install-projctl").Name("install")

// Status runs: git status
var Status = targ.Targ("git status").Name("status")

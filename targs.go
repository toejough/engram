//go:build targ

package projctl

import "github.com/toejough/targ"

func init() {
	targ.Register(InstallSkills, InstallProjctl, InstallHooks, InstallBinary, Install, Status)
}

// InstallSkills runs: projctl skills install
var InstallSkills = targ.Targ("projctl skills install").Name("install-skills")

// InstallProjctl runs: go install -tags sqlite_fts5 ./cmd/projctl
var InstallProjctl = targ.Targ("go install -tags sqlite_fts5 ./cmd/projctl").Name("install-projctl")

// InstallHooks runs: projctl memory hooks install
var InstallHooks = targ.Targ("projctl memory hooks install").Name("install-hooks")

// InstallBinary runs: targ install-projctl install-hooks (build + reinstall hooks)
var InstallBinary = targ.Targ("targ install-projctl install-hooks").Name("install-binary")

// Install runs: targ install-projctl install-skills install-hooks
var Install = targ.Targ("targ install-projctl install-skills install-hooks").Name("install")

// Status runs: git status
var Status = targ.Targ("git status").Name("status")

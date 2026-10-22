package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// Test792a11a1a_Update_ALocalCloneIsInstalledInPlace verifies Scenario 792a11a1a: from a directory inside a checkout
// whose go.mod names the engram module, the binary is built from that checkout and no clone is made.
func Test792a11a1a_Update_ALocalCloneIsInstalledInPlace(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo+"/internal/cli")

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	installs := machine.cmd.callsOf("go install")
	g.Expect(installs).To(HaveLen(1))

	if len(installs) != 1 {
		return
	}

	g.Expect(installs[0].dir).To(Equal(iu1Repo), "the binary is built from the checkout's root")
	g.Expect(machine.cmd.callsOf("git clone")).To(BeEmpty(), "no clone is made")
	g.Expect(outcome.stdout).To(ContainSubstring("source: local clone at " + iu1Repo))
}

// Test792a11a1c_Update_TheInstallLocationFollowsTheGoToolchain verifies Scenario 792a11a1c: with GOBIN unset, GOPATH
// /gp and the home directory /home/u the report names /gp/bin/engram, and with GOBIN also set to /gb it names
// /gb/engram.
func Test792a11a1c_Update_TheInstallLocationFollowsTheGoToolchain(t *testing.T) {
	t.Parallel()

	table := map[string]struct {
		vars map[string]string
		want string
	}{
		"GOBIN unset":    {vars: map[string]string{"GOPATH": "/gp"}, want: "/gp/bin/engram"},
		"GOBIN also set": {vars: map[string]string{"GOPATH": "/gp", "GOBIN": "/gb"}, want: "/gb/engram"},
	}

	for name, testCase := range table {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)
			machine.env.vars = testCase.vars

			outcome := machine.run(t, cli.UpdateArgs{})
			g.Expect(outcome.err).NotTo(HaveOccurred())

			if outcome.err != nil {
				return
			}

			g.Expect(iu1BinaryLine(outcome.stdout)).To(HaveSuffix("→ " + testCase.want + ")"))
			g.Expect(machine.spawn.spawned()).To(HaveLen(1))
			g.Expect(machine.spawn.names()).To(Equal([]string{testCase.want}), "the binary that runs is the one installed")
		})
	}
}

// Test792a11a1d_Update_TheFirstGoModEndsTheWalk verifies Scenario 792a11a1d: from a directory inside a checkout of a
// different Go module, even when an enclosing directory holds the engram module, the binary is built from a fresh
// clone and not from a checkout.
func Test792a11a1d_Update_TheFirstGoModEndsTheWalk(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1RemoteMachine("/outer/other/service")
	machine.fileSystem.files["/outer/go.mod"] = []byte("module github.com/toejough/engram\n")
	machine.fileSystem.files["/outer/other/go.mod"] = []byte("module example.com/other\n")

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	installs := machine.cmd.callsOf("go install")
	g.Expect(installs).To(HaveLen(1))

	if len(installs) != 1 {
		return
	}

	g.Expect(installs[0].dir).To(Equal(iu1CloneDir), "the binary is built from the fresh clone")
	g.Expect(machine.cmd.callsOf("git clone")).To(HaveLen(1))
	g.Expect(outcome.stdout).To(ContainSubstring("source: remote module "))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("local clone"))
}

// Test792a11a1e_Update_GoIsMissing verifies Scenario 792a11a1e: from a local clone with no go on PATH the update
// fails with an error that the go binary is not on PATH, and nothing is handed off.
func Test792a11a1e_Update_GoIsMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)
	machine.cmd.answers = append(machine.cmd.answers, iu1Answer{prefix: "go install", err: update.ErrCommandNotFound})

	outcome := machine.run(t, cli.UpdateArgs{})

	g.Expect(outcome.err).To(MatchError(update.ErrGoNotFound))
	g.Expect(outcome.err).To(MatchError(ContainSubstring("go binary not found on PATH")))
	g.Expect(machine.spawn.spawned()).To(BeEmpty())
	g.Expect(outcome.stdout).To(BeEmpty())
}

// Test792a11a1g_Update_TheFirstGOPATHEntryIsUsed verifies Scenario 792a11a1g: with GOBIN unset and GOPATH
// /gp1:/gp2 the report names /gp1/bin/engram as the install location.
func Test792a11a1g_Update_TheFirstGOPATHEntryIsUsed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)
	machine.env.vars = map[string]string{"GOPATH": "/gp1:/gp2"}

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	g.Expect(iu1BinaryLine(outcome.stdout)).To(HaveSuffix("→ /gp1/bin/engram)"))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("/gp2"))
}

// Test792a11a1h_Update_GitIsMissing verifies Scenario 792a11a1h: outside a checkout with no git on PATH the update
// fails with an error that the git binary is not on PATH, and nothing is installed.
func Test792a11a1h_Update_GitIsMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1RemoteMachine(iu1Work)
	machine.cmd.answers = append(machine.cmd.answers, iu1Answer{prefix: "git clone", err: update.ErrCommandNotFound})

	outcome := machine.run(t, cli.UpdateArgs{})

	g.Expect(outcome.err).To(MatchError(update.ErrGitNotFound))
	g.Expect(outcome.err).To(MatchError(ContainSubstring("git binary not found on PATH")))
	g.Expect(machine.cmd.callsOf("go install")).To(BeEmpty())
	g.Expect(machine.spawn.spawned()).To(BeEmpty())
}

// Test792a11b1a_Update_APointerFileStopsTheInstall verifies Scenario 792a11b1a: when the clone's model.onnx is a
// git-lfs pointer file the update fails with an error naming git-lfs and the install command, and no binary is
// installed.
func Test792a11b1a_Update_APointerFileStopsTheInstall(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1RemoteMachine(iu1Work)
	machine.cmd.cloneModel = []byte("version https://git-lfs.github.com/spec/v1\noid sha256:0f\nsize 90405214\n")

	outcome := machine.run(t, cli.UpdateArgs{})

	g.Expect(outcome.err).To(MatchError(update.ErrModelLFSStub))
	g.Expect(outcome.err).To(MatchError(ContainSubstring("git-lfs")))
	g.Expect(outcome.err).To(MatchError(ContainSubstring("brew install git-lfs && git lfs install")))
	g.Expect(outcome.err).To(MatchError(ContainSubstring("rerun `engram update`")))
	g.Expect(machine.cmd.callsOf("go install")).To(BeEmpty(), "no binary is installed")
	g.Expect(machine.spawn.spawned()).To(BeEmpty())
}

// Test792a11b1b_Update_ARealModelInstalls verifies Scenario 792a11b1b: when the clone's model.onnx holds the real
// model the binary is built from the clone.
func Test792a11b1b_Update_ARealModelInstalls(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1RemoteMachine(iu1Work)

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	if outcome.err != nil {
		return
	}

	installs := machine.cmd.callsOf("go install")
	g.Expect(installs).To(HaveLen(1))

	if len(installs) != 1 {
		return
	}

	g.Expect(installs[0].dir).To(Equal(iu1CloneDir), "the binary is built from the clone")
	g.Expect(machine.spawn.spawned()).To(HaveLen(1))
}

// Test792a11b1c_Update_AnUnreadableModelIsAnError verifies Scenario 792a11b1c: when the clone has no model.onnx the
// update fails with an error that the cloned model could not be read, and nothing is installed.
func Test792a11b1c_Update_AnUnreadableModelIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1RemoteMachine(iu1Work)
	machine.cmd.cloneModel = nil

	outcome := machine.run(t, cli.UpdateArgs{})

	g.Expect(outcome.err).To(MatchError(ContainSubstring("reading cloned model " + iu1ModelPath)))
	g.Expect(machine.cmd.callsOf("go install")).To(BeEmpty())
	g.Expect(machine.spawn.spawned()).To(BeEmpty())
}

// Test792a11c1a_Update_AProvableDowngradeIsRefused verifies Scenario 792a11c1a: when the installed binary was built
// at a revision the clone's revision does not descend from, the install is refused with an error that says it would
// downgrade the installed binary and names --allow-downgrade, and the run fails.
func Test792a11c1a_Update_AProvableDowngradeIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)
	iu1InstallBinaryWithProvableDowngrade(machine)

	outcome := machine.run(t, cli.UpdateArgs{})

	g.Expect(outcome.err).To(MatchError(update.ErrLocalDowngrade))
	g.Expect(outcome.err).To(MatchError(ContainSubstring("would downgrade the installed binary")))
	g.Expect(outcome.err).To(MatchError(ContainSubstring("--allow-downgrade")))
	g.Expect(machine.cmd.callsOf("go install")).To(BeEmpty(), "the install is refused")
	g.Expect(machine.spawn.spawned()).To(BeEmpty())
}

// Test792a11c1b_Update_TheFlagBypassesTheCheck verifies Scenario 792a11c1b: with the same provable downgrade,
// --allow-downgrade installs the binary.
func Test792a11c1b_Update_TheFlagBypassesTheCheck(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)
	iu1InstallBinaryWithProvableDowngrade(machine)

	outcome := machine.run(t, cli.UpdateArgs{AllowDowngrade: true})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(machine.cmd.callsOf("go install")).To(HaveLen(1))
	g.Expect(machine.spawn.spawned()).To(HaveLen(1))
}

// Test792a11c1c_Update_AFirstInstallIsNotChecked verifies Scenario 792a11c1c: with no binary at the install
// location the binary is installed without a downgrade check.
func Test792a11c1c_Update_AFirstInstallIsNotChecked(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)
	machine.cmd.answers = append(machine.cmd.answers, iu1NonAncestorAnswer())

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(machine.cmd.callsOf("go install")).To(HaveLen(1))
	g.Expect(machine.cmd.callsOf("go version")).To(BeEmpty(), "no installed binary is read")
	g.Expect(machine.cmd.callsOf("git merge-base")).To(BeEmpty(), "no ancestry is asked")
}

// Test792a11c1d_Update_AnUnknownInstalledRevisionDoesNotBlock verifies Scenario 792a11c1d: when the installed
// binary's revision is not a commit the clone knows, the binary is installed and the report does not claim a
// downgrade.
func Test792a11c1d_Update_AnUnknownInstalledRevisionDoesNotBlock(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)
	iu1InstallBinaryAtRevision(machine, iu1InstalledRevision)
	machine.cmd.answers = append(machine.cmd.answers, iu1Answer{
		prefix: "git merge-base",
		stderr: "fatal: Not a valid commit name " + iu1InstalledRevision,
		err:    errors.New("exit status 128"),
	})

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(machine.cmd.callsOf("go install")).To(HaveLen(1))
	g.Expect(strings.ToLower(outcome.stdout)).NotTo(ContainSubstring("downgrade"))
}

// Test792a11c1e_Update_AnUnreadableRevisionDoesNotBlock verifies Scenario 792a11c1e: when the installed binary
// carries no revision, or one that does not parse, the binary is installed.
func Test792a11c1e_Update_AnUnreadableRevisionDoesNotBlock(t *testing.T) {
	t.Parallel()

	table := map[string]iu1Answer{
		"the binary lists no revision": {
			prefix: "go version", stdout: iu1BinaryInfo + "\tbuild\tvcs=git\n",
		},
		"the revision is empty": {
			prefix: "go version", stdout: iu1BinaryInfo + "\tbuild\tvcs.revision=\n",
		},
		"the binary's build info cannot be read": {
			prefix: "go version", err: errors.New("could not read Go build info"),
		},
	}

	for name, versionAnswer := range table {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)
			machine.fileSystem.files[iu1HomeBinary] = []byte("binary")
			machine.cmd.answers = append(machine.cmd.answers, versionAnswer, iu1NonAncestorAnswer())

			outcome := machine.run(t, cli.UpdateArgs{})
			g.Expect(outcome.err).NotTo(HaveOccurred())
			g.Expect(machine.cmd.callsOf("go install")).To(HaveLen(1))
			g.Expect(machine.cmd.callsOf("git merge-base")).To(BeEmpty(), "no revision, so no ancestry is asked")
		})
	}
}

// Test792a11d1a_Update_ALocalInstallReportsItsRevision verifies Scenario 792a11d1a: for a local clone at
// ~/src/engram at commit abc1234 the report begins with the engram update line and its source line is
// "local clone at ~/src/engram (rev abc1234)".
func Test792a11d1a_Update_ALocalInstallReportsItsRevision(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const home = "/home/joe"

	machine := iu1LocalMachine(home, home+"/src/engram", home+"/src/engram")

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	lines := strings.Split(outcome.stdout, "\n")
	g.Expect(len(lines)).To(BeNumerically(">=", iu1HeaderLines))

	if len(lines) < iu1HeaderLines {
		return
	}

	g.Expect(lines[0]).To(Equal("engram update"))
	g.Expect(lines[1]).To(Equal("  source: local clone at ~/src/engram (rev abc1234)"))
}

// Test792a11d1b_Update_ARealInstallReportsWhereTheBinaryWent verifies Scenario 792a11d1b: after an install to
// /home/u/go/bin/engram the binary line ends with "ok (engram <short SHA> → ~/go/bin/engram)".
func Test792a11d1b_Update_ARealInstallReportsWhereTheBinaryWent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(machine.spawn.names()).To(Equal([]string{iu1HomeBinary}))
	g.Expect(iu1BinaryLine(outcome.stdout)).
		To(HaveSuffix(" ... ok (engram " + iu1Revision + " → ~/go/bin/engram)"))
}

// Test792a11d1c_Update_ARemoteSourceReportsItsRevision verifies Scenario 792a11d1c: for a binary built from a fresh
// clone at commit abc1234 the source line is "remote module github.com/toejough/engram abc1234".
func Test792a11d1c_Update_ARemoteSourceReportsItsRevision(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1RemoteMachine(iu1Work)

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	lines := strings.Split(outcome.stdout, "\n")
	g.Expect(len(lines)).To(BeNumerically(">=", iu1HeaderLines))

	if len(lines) < iu1HeaderLines {
		return
	}

	g.Expect(lines[1]).To(Equal("  source: remote module github.com/toejough/engram abc1234"))

	revisions := machine.cmd.callsOf("git rev-parse")
	g.Expect(revisions).To(HaveLen(1))

	if len(revisions) != 1 {
		return
	}

	g.Expect(revisions[0].dir).To(Equal(iu1CloneDir), "the revision is the clone's")
}

// Test792a11e1a_Update_TheDeployRunsInTheNewBinary verifies Scenario 792a11e1a: after a successful install the
// install result is printed once, the installed binary is started once with the update subcommand and the loop
// guard, the deploy report is printed once by that binary, and the exit status is that binary's.
func Test792a11e1a_Update_TheDeployRunsInTheNewBinary(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)
	machine.spawn.run = func(string, []string, []string) (int, error) {
		machine.stdout.WriteString("installed: Claude Code\n") // the deploy report the installed binary prints

		return iu1ChildExitCode, nil
	}

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	spawned := machine.spawn.spawned()
	g.Expect(spawned).To(HaveLen(1))

	if len(spawned) != 1 {
		return
	}

	g.Expect(spawned[0].name).To(Equal(iu1HomeBinary))
	g.Expect(spawned[0].args).To(Equal([]string{"update"}))
	g.Expect(spawned[0].env).To(Equal([]string{"ENGRAM_UPDATE_REEXEC=1"}))
	g.Expect(outcome.exits).To(Equal([]int{iu1ChildExitCode}), "this process exits with the installed binary's status")
	g.Expect(strings.Count(outcome.stdout, "engram update\n")).To(Equal(1))
	g.Expect(strings.Count(outcome.stdout, "binary: go install")).To(Equal(1))
	g.Expect(strings.Count(outcome.stdout, "installed: Claude Code")).To(Equal(1))
	g.Expect(machine.cmd.callsOf("go install")).To(HaveLen(1))
	g.Expect(machine.fileSystem.symlinks).To(BeEmpty(), "this process deploys nothing itself")
}

// Test792a11e1b_Update_TheInstallResultComesFirst verifies Scenario 792a11e1b: the output begins with the
// "engram update", source and binary lines, followed by the deploy report of the handed-off run with no second
// header. The installed binary is the production run again, with the loop guard it was given.
func Test792a11e1b_Update_TheInstallResultComesFirst(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)
	machine.spawn.run = machine.childRun(t)

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	lines := strings.Split(outcome.stdout, "\n")
	g.Expect(len(lines)).To(BeNumerically(">", iu1HeaderLines))

	if len(lines) <= iu1HeaderLines {
		return
	}

	g.Expect(lines[0]).To(Equal("engram update"))
	g.Expect(lines[1]).To(HavePrefix("  source: "))
	g.Expect(lines[2]).To(HavePrefix("  binary: "))

	deployReport := strings.Join(lines[iu1HeaderLines:], "\n")
	g.Expect(deployReport).To(ContainSubstring("installed: Claude Code"))
	g.Expect(deployReport).NotTo(ContainSubstring("\nengram update\n"), "the deploy report has no second header")
	g.Expect(strings.HasPrefix(deployReport, "engram update\n")).To(BeFalse())
	g.Expect(deployReport).NotTo(ContainSubstring("source:"))
	g.Expect(deployReport).NotTo(ContainSubstring("binary:"))
}

// Test792a11e1c_Update_AFailedHandOffFallsBack verifies Scenario 792a11e1c: when the installed binary cannot be
// started the update still deploys to the harnesses, and its report says re-exec failed, completed with pre-update
// logic, and why.
func Test792a11e1c_Update_AFailedHandOffFallsBack(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)
	machine.spawn.run = func(string, []string, []string) (int, error) {
		return 0, errors.New("fork/exec " + iu1HomeBinary + ": no such file or directory")
	}

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.exits).To(BeEmpty(), "there was no child whose status to take")
	g.Expect(outcome.stdout).To(ContainSubstring(
		"re-exec failed, completed with pre-update logic: fork/exec " + iu1HomeBinary + ": no such file or directory"))
	g.Expect(outcome.stdout).To(ContainSubstring("agent-instructions/skills/learn/ → ~/.claude/skills/learn/  (1 file)"))
	g.Expect(outcome.stdout).To(ContainSubstring("installed: Claude Code"))
	g.Expect(machine.fileSystem.symlinks).To(HaveKey(iu1Home+"/.claude/skills/learn"), "the skill is linked")
}

// Test792a11e1d_Update_AHandedOffRunDoesNotClaimAnInstall verifies Scenario 792a11e1d: an update running as the
// handed-off binary installs nothing, and its report does not say that go install ran.
func Test792a11e1d_Update_AHandedOffRunDoesNotClaimAnInstall(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu1LocalMachine(iu1Home, iu1Repo, iu1Repo)
	machine.env.vars = map[string]string{update.ReexecSentinelEnvVar: "1"}

	outcome := machine.run(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(machine.cmd.callsOf("go install")).To(BeEmpty())
	g.Expect(machine.spawn.spawned()).To(BeEmpty(), "a handed-off run hands off no further")
	g.Expect(outcome.stdout).To(ContainSubstring("installed: Claude Code"), "the deploy report is printed")
	g.Expect(outcome.stdout).NotTo(ContainSubstring("go install"))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("binary:"))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("engram update\n"))
}

// Test792a11e1e_Update_ADryRunNeverHandsOff verifies Scenario 792a11e1e: with --dry-run, from a local clone and from
// outside a checkout, nothing is installed and no second process is started.
func Test792a11e1e_Update_ADryRunNeverHandsOff(t *testing.T) {
	t.Parallel()

	table := map[string]*iu1Machine{
		"from a local clone":         iu1LocalMachine(iu1Home, iu1Repo, iu1Repo),
		"from outside of a checkout": iu1RemoteMachine(iu1Work),
	}

	for name, machine := range table {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			outcome := machine.run(t, cli.UpdateArgs{DryRun: true})
			g.Expect(outcome.err).NotTo(HaveOccurred())
			g.Expect(outcome.stdout).To(HavePrefix("[dry-run] engram update\n"))
			g.Expect(machine.cmd.callsOf("go install")).To(BeEmpty(), "nothing is installed")
			g.Expect(machine.spawn.spawned()).To(BeEmpty(), "no second process is started")
			g.Expect(outcome.exits).To(BeEmpty())
			g.Expect(machine.fileSystem.symlinks).To(BeEmpty(), "nothing is deployed")
		})
	}
}

// unexported constants.
const (
	// iu1BinaryInfo is the first lines of what go version -m prints for an installed binary.
	iu1BinaryInfo = iu1HomeBinary + ": go1.26.0\n\tpath\tgithub.com/toejough/engram/cmd/engram\n"
	// iu1ChildExitCode is the status of the handed-off binary in the hand-off scenario.
	iu1ChildExitCode = 7
	iu1CloneDir      = "/tmp/engram-update-clone"
	// iu1HeaderLines is the count of lines of the three-line header: "engram update", source and binary.
	iu1HeaderLines       = 3
	iu1Home              = "/home/u"
	iu1HomeBinary        = iu1Home + "/go/bin/engram"
	iu1InstalledRevision = "aaaa111"
	iu1MebibyteBytes     = 1 << 20
	iu1ModelPath         = iu1CloneDir + "/internal/embed/assets/model/model.onnx"
	iu1Repo              = "/repo"
	iu1Revision          = "abc1234"
	iu1Work              = "/work/elsewhere"
)

// unexported variables.
var (
	_ update.Commander = (*iu1Commander)(nil)
	_ update.Spawner   = (*iu1Spawner)(nil)
)

// iu1Answer is what the fake commander answers to a command whose line starts with prefix.
type iu1Answer struct {
	prefix string
	stdout string
	stderr string
	err    error
}

// iu1Call is one command the update asked the commander to run.
type iu1Call struct {
	dir  string
	name string
	args []string
}

func (c iu1Call) line() string { return strings.TrimSpace(c.name + " " + strings.Join(c.args, " ")) }

// iu1Commander is an update.Commander that records every call and answers by the prefix of the command line.
// Without an answer it prints the revision for git rev-parse and succeeds silently otherwise. A git clone seeds the
// clone's model file into fileSystem when cloneModel is set, as a real clone would.
type iu1Commander struct {
	fileSystem *iu1FS
	cloneModel []byte
	answers    []iu1Answer
	calls      []iu1Call
}

func (c *iu1Commander) Run(_ context.Context, dir, name string, args ...string) ([]byte, []byte, error) {
	call := iu1Call{dir: dir, name: name, args: slices.Clone(args)}
	c.calls = append(c.calls, call)

	line := call.line()

	if strings.HasPrefix(line, "git clone") && c.cloneModel != nil {
		c.fileSystem.files[iu1ModelPath] = c.cloneModel
	}

	for _, answer := range c.answers {
		if strings.HasPrefix(line, answer.prefix) {
			return []byte(answer.stdout), []byte(answer.stderr), answer.err
		}
	}

	if strings.HasPrefix(line, "git rev-parse") {
		return []byte(iu1Revision + "\n"), nil, nil
	}

	return nil, nil, nil
}

func (c *iu1Commander) callsOf(prefix string) []iu1Call {
	matching := []iu1Call{}

	for _, call := range c.calls {
		if strings.HasPrefix(call.line(), prefix) {
			matching = append(matching, call)
		}
	}

	return matching
}

// iu1Env is an update.Env over fixed values.
type iu1Env struct {
	home string
	cwd  string
	vars map[string]string
}

func (e iu1Env) Getenv(key string) string { return e.vars[key] }

func (e iu1Env) Getwd() (string, error) { return e.cwd, nil }

func (e iu1Env) UserHomeDir() (string, error) { return e.home, nil }

// iu1FS is the in-memory filesystem of the tests with directories that exist once a file is written under them, as on
// a disk.
type iu1FS struct{ *fakeUpdateFS }

func (f *iu1FS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	for dir := filepath.Dir(path); dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		f.dirs[dir] = true
	}

	return f.fakeUpdateFS.WriteFile(path, data, perm)
}

// iu1Machine is one machine an update runs on: its filesystem, commander, environment and spawner.
type iu1Machine struct {
	fileSystem *iu1FS
	cmd        *iu1Commander
	env        iu1Env
	spawn      *iu1Spawner
	stdout     bytes.Buffer
}

// childRun is a spawner body that runs the update again as the installed binary would: in the same machine, with the
// environment the parent handed it, writing to the same stdout.
func (m *iu1Machine) childRun(t *testing.T) func(string, []string, []string) (int, error) {
	t.Helper()

	return func(_ string, _ []string, env []string) (int, error) {
		vars := map[string]string{}

		for _, entry := range env {
			key, value, _ := strings.Cut(entry, "=")
			vars[key] = value
		}

		childEnv := iu1Env{home: m.env.home, cwd: m.env.cwd, vars: vars}
		childSpawn := &iu1Spawner{}
		deps := cli.ExportNewUpdateDepsWithSpawn(m.fileSystem, m.cmd, childEnv, childSpawn, func(int) {})

		exitCode := 0

		err := cli.ExportRunUpdate(t.Context(), cli.UpdateArgs{}, deps, &m.stdout)
		if err != nil || len(childSpawn.spawned()) > 0 {
			exitCode = 1
		}

		return exitCode, nil
	}
}

// run drives one update through the command's own composition and collects what it did.
func (m *iu1Machine) run(t *testing.T, args cli.UpdateArgs) iu1Outcome {
	t.Helper()

	var exits []int

	deps := cli.ExportNewUpdateDepsWithSpawn(
		m.fileSystem, m.cmd, m.env, m.spawn, func(code int) { exits = append(exits, code) },
	)

	err := cli.ExportRunUpdate(t.Context(), args, deps, &m.stdout)

	return iu1Outcome{err: err, stdout: m.stdout.String(), exits: exits}
}

// iu1Outcome is what one update returned, wrote and exited with.
type iu1Outcome struct {
	err    error
	stdout string
	exits  []int
}

// iu1Spawn is one start of the installed binary.
type iu1Spawn struct {
	name string
	args []string
	env  []string
}

// iu1Spawner is an update.Spawner that records every start and runs run, when set, as the started binary.
type iu1Spawner struct {
	run   func(name string, args, env []string) (int, error)
	calls []iu1Spawn
}

func (s *iu1Spawner) Run(name string, args, env []string) (int, error) {
	s.calls = append(s.calls, iu1Spawn{name: name, args: slices.Clone(args), env: slices.Clone(env)})

	if s.run == nil {
		return 0, nil
	}

	return s.run(name, args, env)
}

func (s *iu1Spawner) names() []string {
	names := make([]string, 0, len(s.calls))

	for _, call := range s.calls {
		names = append(names, call.name)
	}

	return names
}

func (s *iu1Spawner) spawned() []iu1Spawn { return slices.Clone(s.calls) }

// iu1BinaryLine returns the report's binary line.
func iu1BinaryLine(stdout string) string {
	for line := range strings.SplitSeq(stdout, "\n") {
		if strings.HasPrefix(line, "  binary: ") {
			return line
		}
	}

	return ""
}

// iu1InstallBinaryAtRevision puts a binary that carries revision at the default install location.
func iu1InstallBinaryAtRevision(machine *iu1Machine, revision string) {
	machine.fileSystem.files[iu1HomeBinary] = []byte("binary")
	machine.cmd.answers = append(machine.cmd.answers, iu1Answer{
		prefix: "go version", stdout: iu1BinaryInfo + "\tbuild\tvcs.revision=" + revision + "\n",
	})
}

// iu1InstallBinaryWithProvableDowngrade installs a binary whose revision the clone's revision does not descend from:
// git answers the ancestry question with a plain "no".
func iu1InstallBinaryWithProvableDowngrade(machine *iu1Machine) {
	iu1InstallBinaryAtRevision(machine, iu1InstalledRevision)
	machine.cmd.answers = append(machine.cmd.answers, iu1NonAncestorAnswer())
}

// iu1LocalMachine is a machine with a checkout of the engram module at repo, one Claude Code harness under home and
// one skill to deploy, running from cwd.
func iu1LocalMachine(home, repo, cwd string) *iu1Machine {
	fileSystem := &iu1FS{fakeUpdateFS: newFakeUpdateFS()}
	fileSystem.dirs[home+"/.claude"] = true
	fileSystem.files[repo+"/go.mod"] = []byte("module github.com/toejough/engram\n")
	fileSystem.files[repo+"/agent-instructions/skills/learn/SKILL.md"] = []byte("learn")

	return &iu1Machine{
		fileSystem: fileSystem,
		cmd:        &iu1Commander{fileSystem: fileSystem},
		env:        iu1Env{home: home, cwd: cwd, vars: map[string]string{}},
		spawn:      &iu1Spawner{},
	}
}

// iu1NonAncestorAnswer is git's plain "no" to the question whether the installed revision is an ancestor.
func iu1NonAncestorAnswer() iu1Answer {
	return iu1Answer{prefix: "git merge-base", err: errors.New("exit status 1")}
}

// iu1RemoteMachine is a machine with no go.mod above cwd, whose clone holds the real model.
func iu1RemoteMachine(cwd string) *iu1Machine {
	machine := iu1LocalMachine(iu1Home, iu1Repo, cwd)
	delete(machine.fileSystem.files, iu1Repo+"/go.mod")

	machine.cmd.cloneModel = bytes.Repeat([]byte("m"), iu1MebibyteBytes)
	machine.fileSystem.files[iu1CloneDir+"/agent-instructions/skills/learn/SKILL.md"] = []byte("learn")

	return machine
}

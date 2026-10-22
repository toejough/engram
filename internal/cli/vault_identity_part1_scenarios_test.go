package cli_test

// Scenario tests for the vault-identity product capability, part 1 (861c2zzz): vault location (792a2a), vault name
// (792a2b), flat vault layout (792a2c), existing vaults left alone (792a2f), stable vault ID (792a2h) and the
// uncommitted vault ID notice (792a2i). Spec: docs/specs/product/vault-identity.md.
//
// The commands run through the real command wiring (cli.Targets, flag parsing, first-use gating) over an in-memory
// vault (nr1FS of note_revision_part1_scenarios_test.go, a complete cli.EdgeFS) with the environment, home and
// working directory pinned by the Deps seams, so no temporary directory, no real home, no git and no network is
// involved. The update scenarios run engram update on the in-memory machine of install_and_update_part2 (iu2Machine)
// with a git that answers from the vault's files (vi1GitAnswers).
//
// One read the seams cannot pin: targ reads the env= tags of --vault and --vault-name from the process environment,
// so the process must not export ENGRAM_VAULT_PATH or ENGRAM_VAULT_NAME (the isolated runner exports neither).

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792a2a1a_Targets_TheFlagWinsOverTheEnvironment verifies 792a2a1a (The flag wins over the environment):
// ENGRAM_VAULT_PATH is /env/vault and the command is given --vault /flag/vault; the command creates and uses
// /flag/vault and touches nothing under /env/vault.
func Test792a2a1a_Targets_TheFlagWinsOverTheEnvironment(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := vi1NewFS()

	stdout, stderr := vi1Run(t, fsys, map[string]string{"ENGRAM_VAULT_PATH": "/env/vault"},
		"vault-id", "--vault", "/flag/vault")

	vi1ExpectCommandUsed(g, fsys, "/flag/vault", stdout, stderr)
}

// Test792a2a1b_Targets_TheEnvironmentVariableIsUsedWithoutAFlag verifies 792a2a1b (The environment variable is used
// without a flag): ENGRAM_VAULT_PATH is /env/vault and the command has no --vault; the command uses /env/vault.
func Test792a2a1b_Targets_TheEnvironmentVariableIsUsedWithoutAFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := vi1NewFS()

	stdout, stderr := vi1Run(t, fsys, map[string]string{
		"ENGRAM_VAULT_PATH": "/env/vault", "XDG_DATA_HOME": "/data",
	}, "vault-id")

	vi1ExpectCommandUsed(g, fsys, "/env/vault", stdout, stderr)
}

// Test792a2a1c_Targets_TheXDGDataDirectoryIsTheDefault verifies 792a2a1c (The XDG data directory is the default):
// neither --vault nor ENGRAM_VAULT_PATH is set and XDG_DATA_HOME is /data; the command uses /data/engram/vault and
// not the vault under the home directory.
func Test792a2a1c_Targets_TheXDGDataDirectoryIsTheDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := vi1NewFS()

	stdout, stderr := vi1Run(t, fsys, map[string]string{"XDG_DATA_HOME": "/data"}, "vault-id")

	vi1ExpectCommandUsed(g, fsys, "/data/engram/vault", stdout, stderr)
}

// Test792a2a1d_Targets_TheHomeDirectoryIsTheLastFallback verifies 792a2a1d (The home directory is the last
// fallback): none of --vault, ENGRAM_VAULT_PATH and XDG_DATA_HOME is set; the command uses
// ~/.local/share/engram/vault (the home is /home/u).
func Test792a2a1d_Targets_TheHomeDirectoryIsTheLastFallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := vi1NewFS()

	stdout, stderr := vi1Run(t, fsys, map[string]string{}, "vault-id")

	vi1ExpectCommandUsed(g, fsys, vi1Home+"/.local/share/engram/vault", stdout, stderr)
}

// Test792a2b1b_Learn_TheEnvironmentNamesTheVault verifies 792a2b1b (The environment names the vault):
// ENGRAM_VAULT_NAME is team and engram learn is run without --vault-name; the note's vault: field is team.
func Test792a2b1b_Learn_TheEnvironmentNamesTheVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, vi1EnvNamed("team"))
	rig.learnFact()

	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("vault", "team"))
}

// Test792a2b1c_Learn_TheFlagWinsOverTheEnvironmentForTheVaultName verifies 792a2b1c (The flag wins over the
// environment): ENGRAM_VAULT_NAME is team and engram learn is run with --vault-name org; the note's vault: field
// is org.
func Test792a2b1c_Learn_TheFlagWinsOverTheEnvironmentForTheVaultName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := nr1NewRig(t, vi1EnvNamed("team"))
	rig.learn("fact", "--situation", "testing", "--subject", "s", "--predicate", "p", "--object", "o",
		"--vault-name", "org")

	g.Expect(rig.fields(nr1BasenameA)).To(HaveKeyWithValue("vault", "org"))
}

// Test792a2c1a_EmbedStatus_AMarkdownFileInASubdirectoryIsNotANote verifies 792a2c1a (A Markdown file in a
// subdirectory is not a note): a vault with one note at its root and one Markdown file in a subdirectory reports a
// total of 1 note under engram embed status. The second case names the subdirectory like a note file, which a
// listing that took directories for notes would count.
func Test792a2c1a_EmbedStatus_AMarkdownFileInASubdirectoryIsNotANote(t *testing.T) {
	t.Parallel()

	const (
		rootNote   = vi1Vault + "/1.2026-10-09.fact.md"
		nestedNote = "/2.2026-10-09.fact.md"
	)

	for name, subdirectory := range map[string]string{
		"a subdirectory":                   vi1Vault + "/archive",
		"a subdirectory named like a note": vi1Vault + "/3.2026-10-09.fact.md",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := vi1NewFS(vi1Vault)
			fsys.files[rootNote] = []byte("---\ntype: fact\n---\n\nroot note\n")
			fsys.files[subdirectory+nestedNote] = []byte("---\ntype: fact\n---\n\nnested note\n")

			stdout, stderr := vi1Run(t, fsys, map[string]string{}, "embed", "status", "--vault", vi1Vault)

			g.Expect(stderr).To(BeEmpty())
			g.Expect(stdout).To(MatchRegexp(`total: +1\n`))
		})
	}
}

// Test792a2f1a_Targets_AnExistingVaultIsUntouchedByReadOnlyUse verifies 792a2f1a (An existing vault is untouched by
// read-only use): an existing vault with a note, no vault ID, no .engram/ directory and no parent configured is
// read by check, count, embed status, vault-id and show; no file is created or modified, nothing takes the vault
// lock, and no creation line is printed.
func Test792a2f1a_Targets_AnExistingVaultIsUntouchedByReadOnlyUse(t *testing.T) {
	t.Parallel()

	commands := map[string]struct {
		args []string
		want string
	}{
		"check":        {[]string{"check", "--vault", vi1Vault}, "PASS  G0 graph-resolution"},
		"count":        {[]string{"count", "--vault", vi1Vault, "--group-by", "type"}, "fact\t1\ntotal: 1\n"},
		"embed status": {[]string{"embed", "status", "--vault", vi1Vault}, "total:           1\n"},
		"vault-id":     {[]string{"vault-id", "--vault", vi1Vault}, "vault_id: none\n"},
		"show":         {[]string{"show", "1", "--vault", vi1Vault}, "root note"},
	}

	for name, command := range commands {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := vi1NewFS(vi1Vault)
			fsys.files[vi1Vault+"/1.2026-10-09.fact.md"] = []byte("---\ntype: fact\nsituation: s\n---\n\nroot note\n")
			before := maps.Clone(fsys.files)
			dirsBefore := maps.Clone(fsys.dirs)

			locks := []string{}
			stdout, stderr := vi1RunLocked(t, fsys, &locks, map[string]string{}, command.args...)

			g.Expect(stdout).To(ContainSubstring(command.want), "the command ran")
			g.Expect(stderr).To(BeEmpty(), "no creation line, no error")
			g.Expect(fsys.writes).To(BeEmpty(), "no file is created or modified")
			g.Expect(fsys.files).To(Equal(before))
			g.Expect(fsys.dirs).To(Equal(dirsBefore), "no vault directory is created")
			g.Expect(locks).To(BeEmpty(), "the lock file is a vault file too")
		})
	}
}

// Test792a2h1c_StampVaultID_ConcurrentCreationConverges verifies 792a2h1c (Concurrent creation converges): two
// processes create the ID of a vault that has none at the same moment (each holds its own create until the other has
// reached its own, and each mints a different ID); exactly one ID file is written, and both processes return the ID
// that file holds.
func Test792a2h1c_StampVaultID_ConcurrentCreationConverges(t *testing.T) {
	t.Parallel()

	const rounds = 8

	for round := range rounds {
		t.Run(fmt.Sprintf("round-%d", round), func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := vi1NewFS(vi1Vault)
			rendezvous := &vi1RendezvousFS{EdgeFS: fsys, met: make(chan struct{})}
			ids := make([]string, vi1Processes)
			errs := make([]error, vi1Processes)

			var group sync.WaitGroup

			for process := range vi1Processes {
				group.Go(func() {
					state := cli.ExportNewExchangeState(rendezvous, seqRand(byte(1+process*100)), identityPath,
						func() (string, error) { return "/work", nil })
					ids[process], errs[process] = cli.ExportStampVaultID(state, vi1Vault)
				})
			}

			group.Wait()

			g.Expect(errs).To(HaveEach(Not(HaveOccurred())))
			g.Expect(rendezvous.arrivals()).To(Equal(vi1Processes), "both processes reached their create")
			g.Expect(ids[0]).To(Equal(ids[1]), "both processes use the same ID")
			g.Expect(ids[0]).To(BeElementOf(seqID(1), seqID(1+100)))

			idFile := vi1Vault + "/.engram-vault-id"
			g.Expect(string(fsys.files[idFile])).To(Equal(ids[0] + "\n"))
			g.Expect(slices.DeleteFunc(slices.Clone(fsys.writes), func(written string) bool {
				return written != idFile
			})).To(HaveLen(1), "exactly one ID file is written")
		})
	}
}

// Test792a2i1a_Update_AnUncommittedVaultIDIsFlagged verifies 792a2i1a (Uncommitted ID is flagged): in a git-backed
// vault whose .engram-vault-id is untracked, the report of engram update includes the notice asking for a one-time
// vault commit.
func Test792a2i1a_Update_AnUncommittedVaultIDIsFlagged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := vi1UpdateMachine(vi1Repo{inside: true})
	outcome := machine.update(t, cli.UpdateArgs{})

	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.stdout).To(ContainSubstring(vi1CommitNotice))
	g.Expect(machine.cmd.callsOf("git ls-files --error-unmatch")).To(HaveLen(1), "the ID was not in the index")
}

// Test792a2i1b_Update_AnIgnoredVaultIDIsFlagged verifies 792a2i1b (An ignored ID is flagged): in a git-backed vault
// whose .gitignore ignores .engram-vault-id, the report includes the commit notice, though git status shows nothing
// for the file.
func Test792a2i1b_Update_AnIgnoredVaultIDIsFlagged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := vi1UpdateMachine(vi1Repo{inside: true, ignored: true})
	outcome := machine.update(t, cli.UpdateArgs{})

	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.stdout).To(ContainSubstring(vi1CommitNotice))
	g.Expect(machine.cmd.callsOf("git status --porcelain")).To(BeEmpty(), "the ID was flagged before git status")
}

// Test792a2i1c_Update_ACommittedVaultIDIsNotFlagged verifies 792a2i1c (A committed ID is not flagged): in a git-backed
// vault whose .engram-vault-id is committed and unmodified, the report includes no commit notice; the same vault
// with the ID modified after the commit is flagged.
func Test792a2i1c_Update_ACommittedVaultIDIsNotFlagged(t *testing.T) {
	t.Parallel()

	t.Run("committed and unmodified", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		machine := vi1UpdateMachine(vi1Repo{inside: true, committed: seqID(1) + "\n"})
		outcome := machine.update(t, cli.UpdateArgs{})

		g.Expect(outcome.err).NotTo(HaveOccurred())
		g.Expect(machine.cmd.callsOf("git status --porcelain")).To(HaveLen(1), "the detector ran to its last check")
		g.Expect(outcome.stdout).NotTo(ContainSubstring(vi1CommitNoticePrefix))
	})

	t.Run("committed and then modified", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		machine := vi1UpdateMachine(vi1Repo{inside: true, committed: seqID(9) + "\n"})
		outcome := machine.update(t, cli.UpdateArgs{})

		g.Expect(outcome.err).NotTo(HaveOccurred())
		g.Expect(outcome.stdout).To(ContainSubstring(vi1CommitNotice))
	})
}

// Test792a2i1d_Update_AVaultOutsideGitIsNotFlagged verifies 792a2i1d (A vault outside git is not flagged): a vault
// that is not a git work tree and has a .engram-vault-id gets no commit notice in the report of engram update.
func Test792a2i1d_Update_AVaultOutsideGitIsNotFlagged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := vi1UpdateMachine(vi1Repo{})
	outcome := machine.update(t, cli.UpdateArgs{})

	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(machine.cmd.callsOf("git rev-parse --is-inside-work-tree")).To(HaveLen(1), "the detector asked git")
	g.Expect(outcome.stdout).NotTo(ContainSubstring(vi1CommitNoticePrefix))
}

// unexported constants.
const (
	vi1CommitNotice       = "vault ID file .engram-vault-id is not committed — commit it once in the vault "
	vi1CommitNoticePrefix = "vault ID file .engram-vault-id is not committed"
	vi1GitAnswerCount     = 3
	vi1Home               = "/home/u"
	vi1Processes          = 2
	vi1RendezvousWait     = 2 * time.Second
	vi1Vault              = "/vault"
)

// unexported variables.
var (
	errVi1GitExit = errors.New("git: exit status 1")
)

// vi1RendezvousFS holds each create of the vault ID file until the other process has reached its own, or
// vi1RendezvousWait has passed, so two processes that both found no ID both try to create it.
type vi1RendezvousFS struct {
	cli.EdgeFS

	mutex   sync.Mutex
	arrived int
	met     chan struct{}
}

// WriteFileExcl waits for the other process when path is the vault ID file, then creates the file.
func (f *vi1RendezvousFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	if strings.HasSuffix(path, "/.engram-vault-id") {
		f.arrive()
	}

	return f.EdgeFS.WriteFileExcl(path, data, perm)
}

func (f *vi1RendezvousFS) arrivals() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.arrived
}

func (f *vi1RendezvousFS) arrive() {
	f.mutex.Lock()

	f.arrived++
	if f.arrived == vi1Processes {
		close(f.met)
	}

	f.mutex.Unlock()

	timer := time.NewTimer(vi1RendezvousWait)
	defer timer.Stop()

	select {
	case <-f.met:
	case <-timer.C:
	}
}

// vi1Repo is the git state of the vault of an update scenario: whether the vault is a work tree, whether its
// .gitignore ignores the ID file (the file is then written into the vault), and the content the last commit holds of
// the ID file ("" = the ID is not in the index).
type vi1Repo struct {
	inside    bool
	ignored   bool
	committed string
}

// vi1EnvNamed is the environment of alice's machine with ENGRAM_VAULT_NAME set to vaultName.
func vi1EnvNamed(vaultName string) nr1Env {
	env := nr1Alice
	env.vaultName = vaultName

	return env
}

// vi1ExpectCommandUsed asserts that a vault-id run used vault: it created exactly that vault (one creation line
// naming it), reported the ID it minted, and every file is under it.
func vi1ExpectCommandUsed(g *WithT, fsys *nr1FS, vault, stdout, stderr string) {
	g.Expect(stderr).To(Equal("engram: created new vault at " + vault + "\n"))
	g.Expect(stdout).To(ContainSubstring("vault_id: " + seqID(1) + "\n"))
	g.Expect(string(fsys.files[vault+"/.engram-vault-id"])).To(Equal(seqID(1) + "\n"))

	for file := range fsys.files {
		g.Expect(file).To(HavePrefix(vault+"/"), "no vault file outside the vault the command used")
	}

	for dir := range fsys.dirs {
		inVault := dir == vault || strings.HasPrefix(dir, vault+"/") || strings.HasPrefix(vault, dir+"/")
		g.Expect(inVault).To(BeTrue(), "no directory outside the vault the command used: "+dir)
	}
}

// vi1GitAnswers returns the answers a git gives the detector for the vault: whether it is a work tree, what ls-files
// says of the ID file (an error when it is not in the index), and what status --porcelain says of it (untracked
// shows as ??, a changed committed file as M, an ignored file not at all).
func vi1GitAnswers(repo vi1Repo, fileSystem *iu2FS) []iu1Answer {
	notRepo := errVi1GitExit
	idContent := fileSystem.text(iu3Vault + "/.engram-vault-id")
	ignored := strings.Contains(fileSystem.text(iu3Vault+"/.gitignore"), ".engram-vault-id")
	answers := make([]iu1Answer, 0, vi1GitAnswerCount)

	if !repo.inside {
		const fatal = "fatal: not a git repository (or any of the parent directories): .git"

		return append(answers,
			iu1Answer{prefix: "git rev-parse --is-inside-work-tree", stderr: fatal, err: notRepo},
			iu1Answer{prefix: "git ls-files", stderr: fatal, err: notRepo},
			iu1Answer{prefix: "git status", stderr: fatal, err: notRepo},
		)
	}

	answers = append(answers, iu1Answer{prefix: "git rev-parse --is-inside-work-tree", stdout: "true\n"})

	if repo.committed == "" {
		porcelain := "?? .engram-vault-id\n"
		if ignored {
			porcelain = ""
		}

		return append(answers,
			iu1Answer{prefix: "git ls-files --error-unmatch", stderr: "error: pathspec did not match", err: notRepo},
			iu1Answer{prefix: "git status --porcelain", stdout: porcelain},
		)
	}

	porcelain := ""
	if repo.committed != idContent {
		porcelain = " M .engram-vault-id\n"
	}

	return append(answers,
		iu1Answer{prefix: "git ls-files --error-unmatch", stdout: ".engram-vault-id\n"},
		iu1Answer{prefix: "git status --porcelain", stdout: porcelain},
	)
}

// vi1NewFS is an in-memory vault filesystem holding the given directories.
func vi1NewFS(dirs ...string) *nr1FS {
	known := map[string]bool{}
	for _, dir := range dirs {
		known[dir] = true
	}

	return &nr1FS{files: map[string][]byte{}, dirs: known}
}

// vi1Run runs `engram <args>` over fsys with the environment env, the home /home/u and no parent.
func vi1Run(t *testing.T, fsys *nr1FS, env map[string]string, args ...string) (string, string) {
	t.Helper()

	locks := []string{}

	return vi1RunLocked(t, fsys, &locks, env, args...)
}

// vi1RunLocked is vi1Run that also records every path the command takes the vault lock on.
func vi1RunLocked(
	t *testing.T, fsys *nr1FS, locks *[]string, env map[string]string, args ...string,
) (string, string) {
	t.Helper()

	return executeCapturingBoth(t, append([]string{"engram"}, args...), func(deps *cli.Deps) {
		deps.FS = fsys
		deps.Lock = fakeLocker{lock: func(path string) (func() error, error) {
			*locks = append(*locks, path)

			return func() error { return nil }, nil
		}}
		deps.Embed = stubEmbedder{modelID: "m@4", dims: 4}
		deps.RandRead = seqRand(1)
		deps.EvalSymlinks = identityPath
		deps.Getenv = func(key string) string { return env[key] }
		deps.UserHomeDir = func() (string, error) { return vi1Home, nil }
		deps.Getwd = func() (string, error) { return "/work", nil }
		deps.Commander = stubCommander{}
	})
}

// vi1UpdateMachine is the machine of install_and_update_part3 (Claude Code, a checkout, one skill) with a vault at
// the default location (the handed-off binary inherits no ENGRAM_VAULT_PATH) that holds a vault ID, and a
// git that answers for repo.
func vi1UpdateMachine(repo vi1Repo) *iu2Machine {
	machine := iu3NewMachine()
	machine.fileSystem.seedDir(iu3Vault)

	machine.fileSystem.seedFile(iu3Vault+"/.engram-vault-id", seqID(1)+"\n")

	if repo.ignored {
		machine.fileSystem.seedFile(iu3Vault+"/.gitignore", ".engram-vault-id\n")
	}

	machine.cmd.answers = append(machine.cmd.answers, vi1GitAnswers(repo, machine.fileSystem)...)

	return machine
}

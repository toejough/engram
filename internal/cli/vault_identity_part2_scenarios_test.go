package cli_test

// Scenario tests for the vault-identity product capability, part 2 (861c2zzza): a corrupt vault ID (792a2j), exchange
// state outside tracked files (792a2k), inspecting (792a2l), regenerating (792a2m) and claiming (792a2n) the vault ID,
// the exclusive flags (792a2o), the pause for copied and cloned vaults (792a2p) and the vault that is its own parent
// (792a2q). Spec: docs/specs/product/vault-identity.md.
//
// The vault-id journeys and the serve start run over vi2FS, an in-memory cli.EdgeFS that follows an alias for a
// symlink and can copy or move a tree, behind the production Targets: no file system, home or environment of the
// machine is read. A vault is created by the command itself (the first use of a missing path), and a copied or
// moved vault is that vault's tree under another path, so every Given is a state the system produces. The three
// scenarios that need a real query scan, a real note write or real git (792a2k1a, 792a2p1b and 792a2q1b) are CLI
// journeys over a wiringEnv: a vault in the test's own t.TempDir() (the recorded deviation, #860) and a fake parent
// behind Deps.Fetch, with no network and no real home; 792a2q1b drives Deps.Fetch through MockPvc1Fetch, so that a
// request the command makes holds it and the run reports it.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// Test792a2j1c_Targets_VaultID_AJunkIDFileFailsAsInvalid verifies 792a2j1c (vault-id reports the corrupt file): in a
// vault whose .engram-vault-id holds junk, engram vault-id fails with an error saying the vault ID is invalid, and
// the file is left as it was.
func Test792a2j1c_Targets_VaultID_AJunkIDFileFailsAsInvalid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault whose .engram-vault-id holds junk
	host := newVi2Host()
	host.fsys.put(filepath.Join(vi2Home, vi2IDFile), "junk\n")

	before := host.fsys.snapshot()

	// When the user runs engram vault-id
	result := host.run("vault-id", "--vault", vi2Home)

	// Then it fails with an error saying the vault ID is invalid, and the file is unchanged
	g.Expect(result.exits).To(Equal([]int{vi2FailureExit}))
	g.Expect(result.stderr).To(ContainSubstring("invalid vault id"))
	g.Expect(result.stdout).To(BeEmpty())
	g.Expect(host.fsys.snapshot()).To(Equal(before))
}

// Test792a2k1a_Targets_Learn_AQueuedOfferDoesNotDirtyTheVaultsGit verifies 792a2k1a (State does not dirty the
// vault's git): in a git-backed vault with a parent that is down, a learn queues an offer, and git status
// --porcelain then lists only the note the learn wrote and its sidecar: nothing under .engram/, and no exchange
// state anywhere else.
func Test792a2k1a_Targets_Learn_AQueuedOfferDoesNotDirtyTheVaultsGit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a git-backed vault whose first learn stamped the vault ID and created .engram/, all committed
	env := newWiringEnv(t)
	env.parent.setDown(true)

	commander := env.deps().Commander
	// the running user's global git config must not reach the test: no signing, no hooks, no global excludes
	isolation := []string{
		"-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "-c", "core.excludesFile=/dev/null",
	}
	git := func(args ...string) string {
		t.Helper()

		stdout, stderr, err := commander.Run(t.Context(), env.vault, "git", append(slices.Clone(isolation), args...)...)
		g.Expect(err).NotTo(HaveOccurred(), string(stderr))

		if err != nil {
			return ""
		}

		return string(stdout)
	}

	git("init", "-q")
	env.learnFact("baseline")
	git("add", "-A")
	git("-c", "user.email=vi2@example.invalid", "-c", "user.name=vi2", "commit", "-q", "-m", "baseline")

	// When an offer is queued in it
	queuedNote := env.learnFact("queued")

	// Then git status --porcelain shows no change caused by exchange state
	g.Expect(env.outboxEntries()).To(HaveLen(vi2QueuedOffers), "the second offer was queued")

	sidecar := strings.TrimSuffix(filepath.Base(queuedNote), ".md") + ".vec.json"
	status := strings.Split(strings.TrimSpace(git("status", "--porcelain")), "\n")
	g.Expect(status).To(ConsistOf("?? "+filepath.Base(queuedNote), "?? "+sidecar),
		"only the note and the sidecar the learn wrote are reported")

	for tracked := range strings.FieldsSeq(git("ls-files")) {
		g.Expect(tracked).NotTo(HavePrefix(vi2StateDir+"/"), "the state directory is never tracked")
	}
}

// Test792a2k1b_RunServe_StampingTheIDLeavesTheRootGitignoreUntouched verifies 792a2k1b (The root .gitignore is
// untouched): in an existing vault with its own root .gitignore, engram serve stamps the vault ID and creates
// .engram/, with its own .gitignore holding only *, and the root .gitignore is byte-for-byte unchanged.
func Test792a2k1b_RunServe_StampingTheIDLeavesTheRootGitignoreUntouched(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given an existing vault with its own root .gitignore
	const rootIgnore = "private/\n.luhmann.lock\n.obsidian/workspace*\n"

	host := newVi2Host()
	host.fsys.put(filepath.Join(vi2Home, ".gitignore"), rootIgnore)
	host.fsys.put(filepath.Join(vi2Home, "1.2026-10-01.note.md"), "a note\n")

	// When engram stamps the vault's ID and creates .engram/ (the start of engram serve)
	var stderr bytes.Buffer

	served := false
	deps := host.deps(io.Discard, &stderr, func(int) {})
	deps.NewServeMux = func() cli.RawServeMux { return "vi2-mux" }
	deps.RegisterRoute = func(cli.RawServeMux, string, string, cli.ServeHandler) {}
	deps.ListenAndServe = func(context.Context, cli.RawServeMux, string) error {
		served = true

		return nil
	}

	args := cli.ServeArgs{Addr: "127.0.0.1:0", Vault: vi2Home, VaultName: "personal", ChunksDir: "/chunks"}
	g.Expect(cli.RunServe(t.Context(), args, deps)).To(Succeed())

	// Then the root .gitignore is byte-for-byte unchanged
	g.Expect(served).To(BeTrue())
	g.Expect(host.fsys.text(filepath.Join(vi2Home, vi2IDFile))).To(Equal(seqID(1)+"\n"), "the ID was stamped")
	g.Expect(host.fsys.text(filepath.Join(vi2Home, vi2StateDir, ".gitignore"))).To(Equal("*\n"),
		".engram/ was created with its own .gitignore")
	g.Expect(host.fsys.text(filepath.Join(vi2Home, ".gitignore"))).To(Equal(rootIgnore))
	g.Expect(host.fsys.writes).NotTo(ContainElement("atomic:" + filepath.Join(vi2Home, ".gitignore")))
	g.Expect(host.fsys.writes).NotTo(ContainElement("write:" + filepath.Join(vi2Home, ".gitignore")))
}

// Test792a2l1b_Targets_VaultID_AMovedVaultReportsAMismatchWithBothRemedies verifies 792a2l1b (A moved vault reports
// a mismatch with both remedies): for a vault engram created and that was moved afterwards, engram vault-id prints
// location: mismatch and names both engram vault-id --claim and engram vault-id --regenerate, and writes nothing.
func Test792a2l1b_Targets_VaultID_AMovedVaultReportsAMismatchWithBothRemedies(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault that was moved after engram created it
	host := newVi2Host()
	vaultID := host.create()
	host.fsys.move(vi2Home, vi2Moved)

	before := host.fsys.snapshot()

	// When the user runs engram vault-id
	result := host.run("vault-id", "--vault", vi2Moved)

	// Then stdout shows location: mismatch and names both remedies
	g.Expect(vaultID).To(Equal(seqID(1)))
	g.Expect(result.stdout).To(ContainSubstring("vault_id: " + vaultID + "\nlocation: mismatch\n"))
	g.Expect(result.stdout).To(ContainSubstring("`engram vault-id --claim`"))
	g.Expect(result.stdout).To(ContainSubstring("`engram vault-id --regenerate`"))
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.exits).To(BeEmpty())
	g.Expect(host.fsys.snapshot()).To(Equal(before), "the report writes nothing")
}

// Test792a2l1c_Targets_VaultID_AVaultWithoutAnIDIsReportedWithoutWriting verifies 792a2l1c (A vault without an ID
// is reported without writing): for an existing vault with no vault ID, engram vault-id prints location: no vault
// id, and no file in the vault is created or changed.
func Test792a2l1c_Targets_VaultID_AVaultWithoutAnIDIsReportedWithoutWriting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given an existing vault with no vault ID
	host := newVi2Host()
	host.fsys.put(filepath.Join(vi2Home, "1.2026-10-01.note.md"), "a note\n")
	host.fsys.put(filepath.Join(vi2Home, ".gitignore"), ".luhmann.lock\n")

	before := host.fsys.snapshot()

	// When the user runs engram vault-id
	result := host.run("vault-id", "--vault", vi2Home)

	// Then stdout shows location: no vault id, and no file in the vault is created or changed
	g.Expect(result.stdout).To(Equal("vault_id: none\nlocation: no vault id\n"))
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.exits).To(BeEmpty())
	g.Expect(host.fsys.writes).To(BeEmpty(), "nothing is written")
	g.Expect(host.fsys.snapshot()).To(Equal(before))
}

// Test792a2m1a_Targets_VaultID_RegenerateGivesACopyItsOwnIdentity verifies 792a2m1a (Regenerate gives a copy its
// own identity): in a copy of a vault, engram vault-id --regenerate gives the copy an ID that differs from the
// original's, the location check then passes so that exchange resumes, and no note file changed.
func Test792a2m1a_Targets_VaultID_RegenerateGivesACopyItsOwnIdentity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a copied vault: the tree of a vault engram created, with notes, under another path
	host := newVi2Host()
	originalID := host.create()
	host.fsys.put(filepath.Join(vi2Home, "1.2026-10-01.first.md"), "---\ntype: fact\n---\n\nfirst body\n")
	host.fsys.put(filepath.Join(vi2Home, "2.2026-10-01.second.md"), "---\ntype: fact\n---\n\nsecond body\n")
	host.fsys.copyTree(vi2Home, vi2Copy)

	notesBefore := host.fsys.notes(vi2Copy)
	paused, _ := host.exchangeMayProceed(vi2Copy)
	g.Expect(originalID).To(Equal(seqID(1)))
	g.Expect(paused).To(BeFalse(), "the copy's exchange is paused until it is told what it is")
	g.Expect(notesBefore).To(HaveLen(vi2NotesWithStarter))

	// When the user runs engram vault-id --regenerate in it
	host.randStart = vi2RegenerateSeed
	result := host.run("vault-id", "--vault", vi2Copy, "--regenerate")

	// Then its ID differs from the original's
	copyID := strings.TrimSpace(host.fsys.text(filepath.Join(vi2Copy, vi2IDFile)))
	g.Expect(result.exits).To(BeEmpty())
	g.Expect(result.stdout).To(ContainSubstring("vault_id: " + seqID(vi2RegenerateSeed)))
	g.Expect(copyID).To(Equal(seqID(vi2RegenerateSeed)))
	g.Expect(copyID).NotTo(Equal(originalID))
	g.Expect(host.fsys.text(filepath.Join(vi2Home, vi2IDFile))).To(Equal(originalID+"\n"), "the original keeps its ID")

	// and exchange resumes
	resumed, warning := host.exchangeMayProceed(vi2Copy)
	g.Expect(resumed).To(BeTrue())
	g.Expect(warning).To(BeEmpty())
	g.Expect(host.run("vault-id", "--vault", vi2Copy).stdout).To(Equal("vault_id: " + copyID + "\nlocation: ok\n"))

	// and no note file changed
	g.Expect(host.fsys.notes(vi2Copy)).To(Equal(notesBefore))
}

// Test792a2n1a_Targets_VaultID_ClaimAcceptsAMovedVault verifies 792a2n1a (Claim accepts a moved vault): in a vault
// that was moved to a new path, engram vault-id --claim leaves the ID unchanged, engram vault-id then reports
// location: ok, and exchange resumes.
func Test792a2n1a_Targets_VaultID_ClaimAcceptsAMovedVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault that was moved to a new path
	host := newVi2Host()
	vaultID := host.create()
	host.fsys.move(vi2Home, vi2Moved)

	paused, _ := host.exchangeMayProceed(vi2Moved)
	g.Expect(vaultID).To(Equal(seqID(1)))
	g.Expect(paused).To(BeFalse(), "the moved vault's exchange is paused")

	// When the user runs engram vault-id --claim
	host.randStart = vi2RegenerateSeed
	result := host.run("vault-id", "--vault", vi2Moved, "--claim")

	// Then the ID is unchanged
	g.Expect(result.exits).To(BeEmpty())
	g.Expect(result.stdout).To(ContainSubstring("vault_id: " + vaultID))
	g.Expect(host.fsys.text(filepath.Join(vi2Moved, vi2IDFile))).To(Equal(vaultID + "\n"))

	// engram vault-id then reports location: ok
	g.Expect(host.run("vault-id", "--vault", vi2Moved).stdout).To(Equal("vault_id: " + vaultID + "\nlocation: ok\n"))

	// and exchange resumes
	resumed, warning := host.exchangeMayProceed(vi2Moved)
	g.Expect(resumed).To(BeTrue())
	g.Expect(warning).To(BeEmpty())
}

// Test792a2n1b_Targets_VaultID_ClaimNeedsAnID verifies 792a2n1b (Claim needs an ID): in an existing vault with no
// vault ID, engram vault-id --claim fails with an error saying the vault has no vault ID, and writes nothing.
func Test792a2n1b_Targets_VaultID_ClaimNeedsAnID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given an existing vault with no vault ID
	host := newVi2Host()
	host.fsys.put(filepath.Join(vi2Home, "1.2026-10-01.note.md"), "a note\n")

	before := host.fsys.snapshot()

	// When the user runs engram vault-id --claim
	result := host.run("vault-id", "--vault", vi2Home, "--claim")

	// Then it fails with an error saying the vault has no vault ID
	g.Expect(result.exits).To(Equal([]int{vi2FailureExit}))
	g.Expect(result.stderr).To(ContainSubstring("this vault has no vault ID"))
	g.Expect(result.stdout).To(BeEmpty())
	g.Expect(host.fsys.snapshot()).To(Equal(before), "nothing is written")
}

// Test792a2o1a_Targets_VaultID_BothFlagsAreRefused verifies 792a2o1a (Both flags are refused): engram vault-id
// --regenerate --claim fails naming --regenerate, and the vault ID is unchanged.
func Test792a2o1a_Targets_VaultID_BothFlagsAreRefused(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault with a vault ID
	host := newVi2Host()
	vaultID := host.create()
	before := host.fsys.snapshot()

	// When the user runs engram vault-id --regenerate --claim
	host.randStart = vi2RegenerateSeed
	result := host.run("vault-id", "--vault", vi2Home, "--regenerate", "--claim")

	// Then it fails naming --regenerate, and the vault ID is unchanged
	g.Expect(vaultID).To(Equal(seqID(1)))
	g.Expect(result.exits).To(Equal([]int{vi2FailureExit}))
	g.Expect(result.stderr).To(ContainSubstring("--regenerate"))
	g.Expect(result.stdout).To(BeEmpty())
	g.Expect(host.fsys.text(filepath.Join(vi2Home, vi2IDFile))).To(Equal(vaultID + "\n"))
	g.Expect(host.fsys.writes).To(BeEmpty(), "nothing is written")
	g.Expect(host.fsys.snapshot()).To(Equal(before))
}

// Test792a2p1b_Targets_Query_AClonedChildsMergedQueryStillRuns verifies 792a2p1b (A cloned child's merged query
// still runs): in a vault with ENGRAM_PARENT set whose location record is missing (a git clone carries the vault ID
// and the notes, not the untracked .engram/), engram query returns its results, and stderr carries one warning
// naming both engram vault-id remedies.
func Test792a2p1b_Targets_Query_AClonedChildsMergedQueryStillRuns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault with ENGRAM_PARENT set whose location record is missing
	env := ps2Env(t)
	env.plant(vi2IDFile, []byte(seqID(vi2CloneSeed)+"\n"))
	plantRealVaultNote(t, env.vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nlocal body\n", []float32{1, 0, 0, 0})
	g.Expect(filepath.Join(env.vault, vi2StateDir)).NotTo(BeADirectory())

	// When the user runs engram query
	stdout, stderr := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	// Then the query returns its results
	var parsed queryParsed

	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())
	g.Expect(parsed.Items).To(HaveLen(1))

	if len(parsed.Items) == 1 {
		g.Expect(parsed.Items[0].Path).To(Equal("1.fact.md"))
	}

	g.Expect(env.parent.requests()).NotTo(BeEmpty(), "the merged query ran")

	// and stderr carries one warning naming both remedies
	g.Expect(nonEmptyLines(stderr)).To(HaveLen(1), stderr)
	g.Expect(stderr).To(ContainSubstring("engram vault-id --regenerate"))
	g.Expect(stderr).To(ContainSubstring("engram vault-id --claim"))
}

// Test792a2p1d_Targets_VaultID_ASymlinkedPathPassesTheLocationCheck verifies 792a2p1d (A symlinked path is not a
// false positive): a vault reached through a symlink to its recorded path passes the location check, with no
// warning.
func Test792a2p1d_Targets_VaultID_ASymlinkedPathPassesTheLocationCheck(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault reached through a symlink to its recorded path
	host := newVi2Host()
	vaultID := host.create()
	host.fsys.aliases[vi2Link] = vi2Home

	record := host.fsys.text(filepath.Join(vi2Home, vi2StateDir, "home.json"))
	g.Expect(record).To(ContainSubstring(`"path":"`+vi2Home+`"`), "the record holds the path itself")

	// When the location check runs
	result := host.run("vault-id", "--vault", vi2Link)
	passed, warning := host.exchangeMayProceed(vi2Link)

	// Then the check passes
	g.Expect(result.stdout).To(Equal("vault_id: " + vaultID + "\nlocation: ok\n"))
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(passed).To(BeTrue())
	g.Expect(warning).To(BeEmpty())
}

// Test792a2p1e_Targets_VaultID_AHostnameChangePassesTheLocationCheck verifies 792a2p1e (A hostname change is not a
// false positive): a vault created while the host was called one name passes the location check once the host
// has another name and the vault path did not change; the location record holds the vault ID and the path, and no
// host name.
func Test792a2p1e_Targets_VaultID_AHostnameChangePassesTheLocationCheck(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault created on a host with one name, and the same host under another name
	host := newVi2Host()
	host.env["HOSTNAME"] = "before-rename.local"
	host.env["HOST"] = "before-rename.local"
	vaultID := host.create()

	host.env["HOSTNAME"] = "after-rename.example.net"
	host.env["HOST"] = "after-rename.example.net"

	// When the location check runs
	result := host.run("vault-id", "--vault", vi2Home)
	passed, warning := host.exchangeMayProceed(vi2Home)

	// Then the check passes
	g.Expect(result.stdout).To(Equal("vault_id: " + vaultID + "\nlocation: ok\n"))
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(passed).To(BeTrue())
	g.Expect(warning).To(BeEmpty())

	// and the record names no host: only the vault ID and the path
	var record map[string]any

	g.Expect(json.Unmarshal([]byte(host.fsys.text(filepath.Join(vi2Home, vi2StateDir, "home.json"))), &record)).
		To(Succeed())

	keys := slices.Sorted(maps.Keys(record))
	g.Expect(keys).To(Equal([]string{"path", "vault_id"}))
}

// Test792a2q1b_Targets_Learn_AVaultThatIsItsOwnParentQueuesNothing verifies 792a2q1b (A write in a vault that is
// its own parent queues nothing): in a vault whose parent reports the vault's own ID, engram learn writes the note,
// queues no offer, sends nothing to the parent (Deps.Fetch is a mock that is never given an expectation, so a
// request would hold the command), and prints one warning naming engram vault-id --regenerate.
func Test792a2q1b_Targets_Learn_AVaultThatIsItsOwnParentQueuesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault whose parent reports the vault's own ID: a first query reached the parent and cached its ID
	env := ps2Env(t)
	env.stampVault("")
	env.parent.vaultID = env.localID()

	_, warmup := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())
	g.Expect(warmup).To(ContainSubstring("the parent reports this vault's own ID"))
	g.Expect(env.parentCache().VaultID).To(Equal(env.localID()), "the parent's ID is cached")

	fetch, _ := MockPvc1Fetch(t)
	identity := ps2Identity(ps2User)
	env.wrap = func(deps *cli.Deps) {
		identity(deps)
		deps.Fetch = fetch
	}

	// When the user runs engram learn
	stdout, stderr := pvc1RunWithin(t, env, nil, "learn", "fact", "--slug", "selfparent", "--situation", "s",
		"--subject", "a", "--predicate", "b", "--object", "c", "--source", "t", "--position", "top")

	// Then the note is written, no offer is queued, nothing is sent (the run finished without a request), and one
	// warning names engram vault-id --regenerate
	g.Expect(strings.TrimSpace(stdout)).To(BeAnExistingFile(), "the note is written")
	g.Expect(env.outboxEntries()).To(BeEmpty(), "no offer is queued")
	g.Expect(nonEmptyLines(stderr)).To(HaveLen(1), stderr)
	g.Expect(stderr).To(ContainSubstring("engram vault-id --regenerate"))
}

// unexported constants.
const (
	// vi2CloneSeed starts the random source of the vault ID a clone carries.
	vi2CloneSeed = 70
	vi2Copy      = "/vaults/copy"
	// vi2FailureExit is the exit status of a command that fails.
	vi2FailureExit = 1
	vi2Home        = "/vaults/home"
	vi2IDFile      = ".engram-vault-id"
	vi2Link        = "/links/home"
	vi2Moved       = "/vaults/moved"
	// vi2NotesWithStarter is the number of note files in the created vault the copy tests plant two notes in: the
	// two notes and the starter README.
	vi2NotesWithStarter = 3
	// vi2QueuedOffers is the number of offers queued after the baseline learn and the learn under test.
	vi2QueuedOffers = 2
	// vi2RegenerateSeed starts the random source of the vault ID a regenerate mints.
	vi2RegenerateSeed = 40
	vi2StateDir       = ".engram"
)

// unexported variables.
var (
	errVi2Unsupported = errors.New("vi2FS: operation not modelled")
	vi2Clock          = time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
)

// vi2FS is an in-memory cli.EdgeFS for the vault-id journeys. It keeps files and directories by absolute path,
// refuses a write whose directory is missing, follows an alias as a symlink to a directory, copies and moves a tree
// as a test setup step, and logs every write it was asked for as "mkdir:", "write:", "atomic:" or "excl:" and the
// path. It is not safe for concurrent use: each test builds its own.
type vi2FS struct {
	files   map[string][]byte
	dirs    map[string]bool
	aliases map[string]string
	writes  []string
}

func (f *vi2FS) Lstat(name string) (fs.FileInfo, error) { return f.Stat(name) }

func (f *vi2FS) MkdirAll(name string, _ fs.FileMode) error {
	for current := f.resolve(name); !f.dirs[current]; current = filepath.Dir(current) {
		f.dirs[current] = true
		f.writes = append(f.writes, "mkdir:"+current)
	}

	return nil
}

func (f *vi2FS) MkdirTemp(string, string) (string, error) { return "", errVi2Unsupported }

func (f *vi2FS) ReadDir(string) ([]fs.DirEntry, error) { return nil, errVi2Unsupported }

func (f *vi2FS) ReadFile(name string) ([]byte, error) {
	data, found := f.files[f.resolve(name)]
	if !found {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	return slices.Clone(data), nil
}

func (f *vi2FS) Readlink(string) (string, error) { return "", errVi2Unsupported }

func (f *vi2FS) Remove(string) error { return errVi2Unsupported }

func (f *vi2FS) RemoveAll(string) error { return errVi2Unsupported }

func (f *vi2FS) Rename(string, string) error { return errVi2Unsupported }

func (f *vi2FS) Stat(name string) (fs.FileInfo, error) {
	resolved := f.resolve(name)
	if f.dirs[resolved] {
		return fakeFileInfo{dir: true}, nil
	}

	data, found := f.files[resolved]
	if !found {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}

	return fakeFileInfo{size: int64(len(data))}, nil
}

func (f *vi2FS) Symlink(string, string) error { return errVi2Unsupported }

func (f *vi2FS) WalkDir(string, fs.WalkDirFunc) error { return errVi2Unsupported }

func (f *vi2FS) WriteFile(name string, data []byte, _ fs.FileMode) error {
	return f.store("write", name, data)
}

func (f *vi2FS) WriteFileAtomic(name string, data []byte, _ fs.FileMode) error {
	return f.store("atomic", name, data)
}

func (f *vi2FS) WriteFileExcl(name string, data []byte, _ fs.FileMode) error {
	_, found := f.files[f.resolve(name)]
	if found || f.dirs[f.resolve(name)] {
		return &fs.PathError{Op: "open", Path: name, Err: fs.ErrExist}
	}

	return f.store("excl", name, data)
}

// copyTree copies the directory from, with everything under it, to the path target (as cp -R does).
func (f *vi2FS) copyTree(from, target string) {
	f.mkdirs(filepath.Dir(target))

	for dir := range maps.Clone(f.dirs) {
		if dir == from || strings.HasPrefix(dir, from+string(filepath.Separator)) {
			f.dirs[target+strings.TrimPrefix(dir, from)] = true
		}
	}

	for name, data := range maps.Clone(f.files) {
		if strings.HasPrefix(name, from+string(filepath.Separator)) {
			f.files[target+strings.TrimPrefix(name, from)] = slices.Clone(data)
		}
	}
}

// evalSymlinks is the Deps.EvalSymlinks of the host: the path an alias leads to, which must exist.
func (f *vi2FS) evalSymlinks(name string) (string, error) {
	resolved := f.resolve(name)
	_, statErr := f.Stat(resolved)
	if statErr != nil {
		return "", statErr
	}

	return resolved, nil
}

// mkdirs makes the directory name and its parents without logging a write: a test setup step.
func (f *vi2FS) mkdirs(name string) {
	for current := f.resolve(name); !f.dirs[current]; current = filepath.Dir(current) {
		f.dirs[current] = true
	}
}

// move moves the directory from, with everything under it, to the path target (as mv does).
func (f *vi2FS) move(from, target string) {
	f.copyTree(from, target)

	for dir := range maps.Clone(f.dirs) {
		if dir == from || strings.HasPrefix(dir, from+string(filepath.Separator)) {
			delete(f.dirs, dir)
		}
	}

	for name := range maps.Clone(f.files) {
		if strings.HasPrefix(name, from+string(filepath.Separator)) {
			delete(f.files, name)
		}
	}
}

// notes maps the name of each .md file directly in dir to its content.
func (f *vi2FS) notes(dir string) map[string]string {
	notes := map[string]string{}

	for name, data := range f.files {
		if filepath.Dir(name) == dir && strings.HasSuffix(name, ".md") {
			notes[filepath.Base(name)] = string(data)
		}
	}

	return notes
}

// put plants a file, and its directory, without logging a write: a test setup step.
func (f *vi2FS) put(name, content string) {
	f.mkdirs(filepath.Dir(name))
	f.files[f.resolve(name)] = []byte(content)
}

func (f *vi2FS) resetWrites() { f.writes = nil }

func (f *vi2FS) resolve(name string) string {
	cleaned := filepath.Clean(name)

	for alias, target := range f.aliases {
		if cleaned == alias || strings.HasPrefix(cleaned, alias+string(filepath.Separator)) {
			return target + strings.TrimPrefix(cleaned, alias)
		}
	}

	return cleaned
}

// snapshot maps every file to its content and every directory (with a trailing slash) to "".
func (f *vi2FS) snapshot() map[string]string {
	snapshot := make(map[string]string, len(f.files)+len(f.dirs))

	for name, data := range f.files {
		snapshot[name] = string(data)
	}

	for dir := range f.dirs {
		snapshot[dir+"/"] = ""
	}

	return snapshot
}

func (f *vi2FS) store(operation, name string, data []byte) error {
	resolved := f.resolve(name)
	if !f.dirs[filepath.Dir(resolved)] {
		return &fs.PathError{Op: operation, Path: name, Err: fs.ErrNotExist}
	}

	f.files[resolved] = slices.Clone(data)
	f.writes = append(f.writes, operation+":"+resolved)

	return nil
}

// text is the content of the file name, or "" when there is none.
func (f *vi2FS) text(name string) string { return string(f.files[f.resolve(name)]) }

// vi2Host is one machine for the vault-id journeys: the in-memory file system, the environment variables it
// reports and the seed of its random source. Everything the production Targets reads comes from here.
type vi2Host struct {
	fsys      *vi2FS
	env       map[string]string
	randStart byte
}

// create makes the vault at vi2Home by the first use of a missing path, as engram does, and returns its vault ID.
func (h *vi2Host) create() string {
	h.run("vault-id", "--vault", vi2Home)
	h.fsys.resetWrites()

	return strings.TrimSpace(h.fsys.text(filepath.Join(vi2Home, vi2IDFile)))
}

// deps composes the production carrier over the host.
func (h *vi2Host) deps(stdout, stderr io.Writer, exit func(int)) cli.Deps {
	return cli.Deps{
		Stdout:       stdout,
		Stderr:       stderr,
		Exit:         exit,
		Getenv:       func(key string) string { return h.env[key] },
		Now:          func() time.Time { return vi2Clock },
		Getwd:        func() (string, error) { return "/", nil },
		UserHomeDir:  func() (string, error) { return "/home/vi2", nil },
		FS:           h.fsys,
		RandRead:     seqRand(h.randStart),
		EvalSymlinks: h.fsys.evalSymlinks,
	}
}

// exchangeMayProceed runs the location check a command makes before it sends an offer or pulls a note down: it
// reports whether the check passed and what it warned.
func (h *vi2Host) exchangeMayProceed(vault string) (bool, string) {
	var stderr bytes.Buffer

	state := cli.ExportExchangeStateFromDeps(h.deps(io.Discard, &stderr, func(int) {}))
	passed := cli.ExportWarnVaultLocation(state, vault, &stderr)

	return passed, stderr.String()
}

// run executes `engram <args>` through the production Targets over the host.
func (h *vi2Host) run(args ...string) vi2Result {
	var stdout, stderr bytes.Buffer

	exits := make([]int, 0, 1)
	deps := h.deps(&stdout, &stderr, func(code int) { exits = append(exits, code) })

	_, err := targ.Execute(append([]string{"engram"}, args...), cli.Targets(deps)...)
	if err != nil {
		stderr.WriteString(err.Error())
		stderr.WriteString("\n")
	}

	return vi2Result{stdout: stdout.String(), stderr: stderr.String(), exits: slices.Clip(exits)}
}

// vi2Result is what one command printed, and the exit statuses it asked for.
type vi2Result struct {
	stdout string
	stderr string
	exits  []int
}

func newVi2FS() *vi2FS {
	return &vi2FS{files: map[string][]byte{}, dirs: map[string]bool{"/": true}, aliases: map[string]string{}}
}

func newVi2Host() *vi2Host {
	return &vi2Host{fsys: newVi2FS(), env: map[string]string{}, randStart: 1}
}

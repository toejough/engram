package cli_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestCheckVaultLocation_CopyIsMismatch: a `cp -R` copy carries the same ID
// and home.json, but its canonical path differs from the record, so the
// check fails (design D2 copy detection) while the original still passes.
func TestCheckVaultLocation_CopyIsMismatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := createdVault(t)
	copyVault := filepath.Join(t.TempDir(), "copy")
	copyTree(t, vault, copyVault)

	state := cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, 9))

	copyStatus, copyErr := cli.ExportCheckVaultLocation(state, copyVault)
	g.Expect(copyErr).NotTo(HaveOccurred())
	g.Expect(copyStatus).To(Equal(cli.ExportLocationMismatch))

	origStatus, origErr := cli.ExportCheckVaultLocation(state, vault)
	g.Expect(origErr).NotTo(HaveOccurred())
	g.Expect(origStatus).To(Equal(cli.ExportLocationOK))
}

// TestCheckVaultLocation_MissingRecordIsMissing: a fresh `git clone` has the
// tracked ID file but no (untracked) .engram/home.json.
func TestCheckVaultLocation_MissingRecordIsMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := createdVault(t)
	g.Expect(os.Remove(filepath.Join(vault, ".engram", "home.json"))).To(Succeed())

	status, err := cli.ExportCheckVaultLocation(cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, 1)), vault)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(status).To(Equal(cli.ExportLocationMissing))
}

// TestCheckVaultLocation_NoIDIsReported: a vault that never got an ID
// reports that, rather than missing/mismatch.
func TestCheckVaultLocation_NoIDIsReported(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	status, err := cli.ExportCheckVaultLocation(
		cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, 1)), t.TempDir())
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(status).To(Equal(cli.ExportLocationNoID))
}

// TestCheckVaultLocation_RecordForOtherIDIsMismatch: a record naming a
// different ID than .engram-vault-id fails the check.
func TestCheckVaultLocation_RecordForOtherIDIsMismatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := createdVault(t)
	g.Expect(os.WriteFile(filepath.Join(vault, ".engram-vault-id"),
		[]byte(strings.Repeat("ab", 16)+"\n"), 0o600)).To(Succeed())

	status, err := cli.ExportCheckVaultLocation(cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, 1)), vault)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(status).To(Equal(cli.ExportLocationMismatch))
}

// TestCheckVaultLocation_SymlinkedPathPasses: reaching the same vault
// through a symlink is not a false positive (EvalSymlinks canonicalization).
func TestCheckVaultLocation_SymlinkedPathPasses(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := createdVault(t)
	link := filepath.Join(t.TempDir(), "link")
	g.Expect(os.Symlink(vault, link)).To(Succeed())

	status, err := cli.ExportCheckVaultLocation(cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, 1)), link)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(status).To(Equal(cli.ExportLocationOK))
}

// TestClaimVaultLocation_MovedVaultKeepsIDAndPasses: `--claim` rewrites only
// the location record; the ID and every other file are unchanged.
func TestClaimVaultLocation_MovedVaultKeepsIDAndPasses(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := createdVault(t)
	moved := filepath.Join(t.TempDir(), "moved")
	g.Expect(os.Rename(vault, moved)).To(Succeed())

	state := cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, 50))

	before := treeSnapshot(t, moved)

	status, checkErr := cli.ExportCheckVaultLocation(state, moved)
	g.Expect(checkErr).NotTo(HaveOccurred())
	g.Expect(status).To(Equal(cli.ExportLocationMismatch))

	id, claimErr := cli.ExportClaimVaultLocation(state, moved)
	g.Expect(claimErr).NotTo(HaveOccurred())
	g.Expect(id).To(Equal(seqID(1)))

	after := treeSnapshot(t, moved)
	homeKey := filepath.Join(".engram", "home.json")
	g.Expect(after[homeKey]).NotTo(Equal(before[homeKey]))

	delete(before, homeKey)
	delete(after, homeKey)
	g.Expect(after).To(Equal(before))

	status, checkErr = cli.ExportCheckVaultLocation(state, moved)
	g.Expect(checkErr).NotTo(HaveOccurred())
	g.Expect(status).To(Equal(cli.ExportLocationOK))
}

// TestClaimVaultLocation_NoIDIsError: there is nothing to claim in a vault
// that never got an ID.
func TestClaimVaultLocation_NoIDIsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := cli.ExportClaimVaultLocation(cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, 1)), t.TempDir())
	g.Expect(err).To(HaveOccurred())
}

// TestEnsureVault_CreatesMissingVaultWithIDAndStateDir pins design D2's
// first-use creation: starters, a 32-hex ID from the injected RandRead, a
// self-ignoring .engram/, a {vault_id, path} home.json with the canonical
// path and no hostname, and exactly one stderr creation line.
func TestEnsureVault_CreatesMissingVaultWithIDAndStateDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "vault")

	var stderr bytes.Buffer

	g.Expect(cli.ExportEnsureVault(exchangeDeps(&stderr, 1), vault)).To(Succeed())

	g.Expect(stderr.String()).To(Equal("engram: created new vault at " + vault + "\n"))
	g.Expect(readFileString(t, filepath.Join(vault, ".obsidian", "app.json"))).To(Equal("{}\n"))
	g.Expect(filepath.Join(vault, "README.md")).To(BeARegularFile())
	g.Expect(readFileString(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(seqID(1) + "\n"))
	g.Expect(readFileString(t, filepath.Join(vault, ".engram", ".gitignore"))).To(Equal("*\n"))

	record := readHomeRecord(t, vault)
	g.Expect(record).To(HaveLen(2))
	g.Expect(record).To(HaveKeyWithValue("vault_id", seqID(1)))
	g.Expect(record).To(HaveKeyWithValue("path", canonicalPath(t, vault)))
}

// TestEnsureVault_ExistingVaultWithoutIDWritesNothing: read-only use of an
// existing vault never creates the ID, state dir, or anything else.
func TestEnsureVault_ExistingVaultWithoutIDWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(vault, ".gitignore"), []byte("custom\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "1.fact.md"), []byte("note\n"), 0o600)).To(Succeed())

	before := treeSnapshot(t, vault)

	var stderr bytes.Buffer

	g.Expect(cli.ExportEnsureVault(exchangeDeps(&stderr, 1), vault)).To(Succeed())
	g.Expect(stderr.String()).To(BeEmpty())
	g.Expect(treeSnapshot(t, vault)).To(Equal(before))
}

// TestEnsureVault_RandFailureIsError: a failing random source surfaces as
// an error and leaves no ID file behind.
func TestEnsureVault_RandFailureIsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "vault")
	deps := exchangeDeps(io.Discard, 1)
	deps.RandRead = func([]byte) (int, error) { return 0, errInjected }

	g.Expect(cli.ExportEnsureVault(deps, vault)).To(MatchError(ContainSubstring("vault id")))
	g.Expect(filepath.Join(vault, ".engram-vault-id")).NotTo(BeAnExistingFile())
}

// TestEnsureVault_RegularFileIsNotAVault: a vault path naming a regular file
// is refused, not treated as an existing vault.
func TestEnsureVault_RegularFileIsNotAVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	file := filepath.Join(t.TempDir(), "file")
	g.Expect(os.WriteFile(file, []byte("x"), 0o600)).To(Succeed())
	g.Expect(cli.ExportEnsureVault(exchangeDeps(io.Discard, 1), file)).To(MatchError(ContainSubstring("not a directory")))
}

// TestEnsureVault_RepairsHalfCreatedVault: a first-use stamp that failed
// (random source down) leaves .engram/ but no ID; the next run stamps it
// with one line, and later runs print nothing.
func TestEnsureVault_RepairsHalfCreatedVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "vault")
	broken := exchangeDeps(io.Discard, 1)
	broken.RandRead = func([]byte) (int, error) { return 0, errInjected }
	g.Expect(cli.ExportEnsureVault(broken, vault)).To(HaveOccurred())
	g.Expect(filepath.Join(vault, ".engram")).To(BeADirectory())

	var stderr bytes.Buffer

	g.Expect(cli.ExportEnsureVault(exchangeDeps(&stderr, 6), vault)).To(Succeed())
	g.Expect(readFileString(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(seqID(6) + "\n"))
	g.Expect(nonEmptyLines(stderr.String())).To(HaveLen(1))
	g.Expect(readHomeRecord(t, vault)).To(HaveKeyWithValue("vault_id", seqID(6)))

	var quiet bytes.Buffer

	g.Expect(cli.ExportEnsureVault(exchangeDeps(&quiet, 9), vault)).To(Succeed())
	g.Expect(quiet.String()).To(BeEmpty())
}

// TestEnsureVault_SecondRunCreatesAndPrintsNothing: idempotence — the second
// call leaves the tree byte-identical and prints nothing.
func TestEnsureVault_SecondRunCreatesAndPrintsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "vault")
	g.Expect(cli.ExportEnsureVault(exchangeDeps(io.Discard, 1), vault)).To(Succeed())

	before := treeSnapshot(t, vault)

	var stderr bytes.Buffer

	g.Expect(cli.ExportEnsureVault(exchangeDeps(&stderr, 77), vault)).To(Succeed())
	g.Expect(stderr.String()).To(BeEmpty())
	g.Expect(treeSnapshot(t, vault)).To(Equal(before))
}

// TestEnsureVault_StatFailureIsError: a vault path that cannot be stat'd for
// a reason other than absence (a path under a regular file) is an error, and
// the command never runs.
func TestEnsureVault_StatFailureIsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	file := filepath.Join(t.TempDir(), "file")
	g.Expect(os.WriteFile(file, []byte("x"), 0o600)).To(Succeed())

	vault := filepath.Join(file, "vault")
	g.Expect(cli.ExportEnsureVault(exchangeDeps(io.Discard, 1), vault)).To(HaveOccurred())

	_, stderr := runWithRand(t, []string{"engram", "count", "--vault", vault, "--group-by", "type"}, 1)
	g.Expect(stderr).To(ContainSubstring(vault))
	g.Expect(stderr).NotTo(ContainSubstring(createdVaultLine))
}

// TestExchangeState_ErrorPaths drives the adapter's failure branches over a
// scripted filesystem: every I/O failure surfaces as an error (or, for the
// location check's unparseable record, as a mismatch) and never panics.
func TestExchangeState_ErrorPaths(t *testing.T) {
	t.Parallel()

	const (
		vault   = "/v"
		idPath  = "/v/.engram-vault-id"
		recPath = "/v/.engram/home.json"
		ignPath = "/v/.engram/.gitignore"
	)

	validID := seqID(1) + "\n"

	t.Run("read-id-failure", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := newScriptedStateFS()
		fsys.failReads[idPath] = errInjected
		state := scriptedState(fsys)

		_, checkErr := cli.ExportCheckVaultLocation(state, vault)
		g.Expect(checkErr).To(MatchError(errInjected))

		_, claimErr := cli.ExportClaimVaultLocation(state, vault)
		g.Expect(claimErr).To(MatchError(errInjected))

		_, stampErr := cli.ExportStampVaultID(state, vault)
		g.Expect(stampErr).To(MatchError(errInjected))
	})

	t.Run("invalid-id-content", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := newScriptedStateFS()
		fsys.files[idPath] = "NOT-A-VALID-ID\n"

		_, checkErr := cli.ExportCheckVaultLocation(scriptedState(fsys), vault)
		g.Expect(checkErr).To(MatchError(ContainSubstring("invalid vault id")))

		_, stampErr := cli.ExportStampVaultID(scriptedState(fsys), vault)
		g.Expect(stampErr).To(MatchError(ContainSubstring("engram vault-id --regenerate")))
	})

	t.Run("record-read-failure", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := newScriptedStateFS()
		fsys.files[idPath] = validID
		fsys.failReads[recPath] = errInjected

		_, err := cli.ExportCheckVaultLocation(scriptedState(fsys), vault)
		g.Expect(err).To(MatchError(errInjected))

		var stderr bytes.Buffer

		g.Expect(cli.ExportWarnVaultLocation(scriptedState(fsys), vault, &stderr)).To(BeFalse())
		g.Expect(stderr.String()).To(ContainSubstring("engram vault-id --claim"))
	})

	t.Run("unparseable-record-is-mismatch", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := newScriptedStateFS()
		fsys.files[idPath] = validID
		fsys.files[recPath] = "{not json"

		status, err := cli.ExportCheckVaultLocation(scriptedState(fsys), vault)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(status).To(Equal(cli.ExportLocationMismatch))
	})

	t.Run("canonical-path-failures", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := newScriptedStateFS()
		fsys.files[idPath] = validID
		fsys.files[recPath] = `{"vault_id":"x","path":"/v"}`

		evalFails := cli.ExportNewExchangeState(fsys, seqRand(1),
			func(string) (string, error) { return "", errInjected }, func() (string, error) { return "/", nil })
		_, evalErr := cli.ExportCheckVaultLocation(evalFails, vault)
		g.Expect(evalErr).To(MatchError(errInjected))

		fsys.files["relative/.engram-vault-id"] = validID
		cwdFails := cli.ExportNewExchangeState(fsys, seqRand(1), identityPath,
			func() (string, error) { return "", errInjected })
		_, cwdErr := cli.ExportClaimVaultLocation(cwdFails, "relative")
		g.Expect(cwdErr).To(MatchError(errInjected))
	})

	t.Run("relative-path-is-made-absolute", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := newScriptedStateFS()
		state := cli.ExportNewExchangeState(fsys, seqRand(1), identityPath, func() (string, error) { return "/cwd", nil })

		_, err := cli.ExportStampVaultID(state, "rel/./vault")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(fsys.files["rel/vault/.engram/home.json"]).To(ContainSubstring(`"path":"/cwd/rel/vault"`))
	})

	t.Run("state-dir-failures", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		mkdirFails := newScriptedStateFS()
		mkdirFails.files[idPath] = validID
		mkdirFails.mkdirErr = errInjected
		_, claimErr := cli.ExportClaimVaultLocation(scriptedState(mkdirFails), vault)
		g.Expect(claimErr).To(MatchError(errInjected))

		ignoreFails := newScriptedStateFS()
		ignoreFails.files[idPath] = validID
		ignoreFails.failExcl[ignPath] = errInjected
		_, ignoreErr := cli.ExportClaimVaultLocation(scriptedState(ignoreFails), vault)
		g.Expect(ignoreErr).To(MatchError(errInjected))

		recordFails := newScriptedStateFS()
		recordFails.files[idPath] = validID
		recordFails.failAtomic[recPath] = errInjected
		_, recordErr := cli.ExportClaimVaultLocation(scriptedState(recordFails), vault)
		g.Expect(recordErr).To(MatchError(errInjected))

		_, stampErr := cli.ExportStampVaultID(scriptedState(newScriptedStateFSWith(func(f *scriptedStateFS) {
			f.failAtomic[recPath] = errInjected
		})), vault)
		g.Expect(stampErr).To(MatchError(errInjected))
	})

	t.Run("id-create-failures", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		exclFails := newScriptedStateFS()
		exclFails.failExcl[idPath] = errInjected
		_, exclErr := cli.ExportStampVaultID(scriptedState(exclFails), vault)
		g.Expect(exclErr).To(MatchError(errInjected))

		shortRead := cli.ExportNewExchangeState(newScriptedStateFS(),
			func([]byte) (int, error) { return 3, nil }, identityPath, func() (string, error) { return "/", nil })
		_, shortErr := cli.ExportStampVaultID(shortRead, vault)
		g.Expect(shortErr).To(MatchError(ContainSubstring("short random read")))

		rereadFails := newScriptedStateFS()
		rereadFails.failExcl[idPath] = fmt.Errorf("lost: %w", fs.ErrExist)
		rereadFails.readSequence = []error{fs.ErrNotExist, errInjected}
		_, rereadErr := cli.ExportStampVaultID(scriptedState(rereadFails), vault)
		g.Expect(rereadErr).To(MatchError(errInjected))
	})

	t.Run("regenerate-failures", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		randFails := cli.ExportNewExchangeState(newScriptedStateFS(),
			func([]byte) (int, error) { return 0, errInjected }, identityPath, func() (string, error) { return "/", nil })
		_, randErr := cli.ExportRegenerateVaultID(randFails, vault)
		g.Expect(randErr).To(MatchError(errInjected))

		idWriteFails := newScriptedStateFS()
		idWriteFails.failAtomic[idPath] = errInjected
		_, idErr := cli.ExportRegenerateVaultID(scriptedState(idWriteFails), vault)
		g.Expect(idErr).To(MatchError(errInjected))

		recordFails := newScriptedStateFS()
		recordFails.failAtomic[recPath] = errInjected
		_, recordErr := cli.ExportRegenerateVaultID(scriptedState(recordFails), vault)
		g.Expect(recordErr).To(MatchError(errInjected))
	})
}

// TestRegenerateVaultID_CopyGetsOwnIdentity: `--regenerate` in a copy mints
// a new ID, rewrites the ID file and record, passes the check, and changes
// no note file.
func TestRegenerateVaultID_CopyGetsOwnIdentity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := createdVault(t)
	g.Expect(os.WriteFile(filepath.Join(vault, "1.fact.md"), []byte("note body\n"), 0o600)).To(Succeed())

	copyVault := filepath.Join(t.TempDir(), "copy")
	copyTree(t, vault, copyVault)

	state := cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, 60))

	newID, regenErr := cli.ExportRegenerateVaultID(state, copyVault)
	g.Expect(regenErr).NotTo(HaveOccurred())
	g.Expect(newID).To(Equal(seqID(60)))
	g.Expect(newID).NotTo(Equal(seqID(1)))
	g.Expect(readFileString(t, filepath.Join(copyVault, ".engram-vault-id"))).To(Equal(newID + "\n"))
	g.Expect(readFileString(t, filepath.Join(copyVault, "1.fact.md"))).To(Equal("note body\n"))
	g.Expect(readHomeRecord(t, copyVault)).To(HaveKeyWithValue("vault_id", newID))

	status, checkErr := cli.ExportCheckVaultLocation(state, copyVault)
	g.Expect(checkErr).NotTo(HaveOccurred())
	g.Expect(status).To(Equal(cli.ExportLocationOK))

	g.Expect(readFileString(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(seqID(1) + "\n"))
}

// TestSelfParentGuard_Property: the guard fires iff the parent reports the
// local vault's own (non-empty) ID; firing prints exactly one warning naming
// `engram vault-id --regenerate`, not firing prints nothing.
func TestSelfParentGuard_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		hexID := rapid.StringMatching(`[0-9a-f]{32}`)
		local := rapid.OneOf(hexID, rapid.Just("")).Draw(rt, "local")
		parent := rapid.OneOf(hexID, rapid.Just(local)).Draw(rt, "parent")

		var stderr bytes.Buffer

		fired := cli.ExportSelfParentGuard(local, parent, &stderr)
		want := local != "" && local == parent

		if fired != want {
			rt.Fatalf("guard(%q, %q) = %v, want %v", local, parent, fired, want)
		}

		lines := nonEmptyLines(stderr.String())

		if want && (len(lines) != 1 || !strings.Contains(lines[0], "engram vault-id --regenerate")) {
			rt.Fatalf("want one warning naming --regenerate, got %q", stderr.String())
		}

		if !want && len(lines) != 0 {
			rt.Fatalf("want no output, got %q", stderr.String())
		}
	})
}

// TestStampVaultID_ConcurrentCreatorsConverge: two processes racing to stamp
// the same vault both end up with the single ID that won the exclusive
// create, and that ID is the file's content.
func TestStampVaultID_ConcurrentCreatorsConverge(t *testing.T) {
	t.Parallel()

	const rounds = 20

	for round := range rounds {
		t.Run(fmt.Sprintf("round-%d", round), func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := t.TempDir()
			ids := make([]string, 2)
			errs := make([]error, 2)

			var wg sync.WaitGroup

			for i := range 2 {
				wg.Go(func() {
					state := cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, byte(1+i*100)))
					ids[i], errs[i] = cli.ExportStampVaultID(state, vault)
				})
			}

			wg.Wait()

			g.Expect(errs).To(HaveEach(Not(HaveOccurred())))
			g.Expect(ids[0]).To(Equal(ids[1]))
			g.Expect(readFileString(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(ids[0] + "\n"))
		})
	}
}

// TestStampVaultID_ExistingVaultKeepsRootGitignoreAndReusesID: serve's lazy
// stamp creates the ID and .engram/ in an existing vault, never touches the
// tracked root .gitignore, and later stamps reuse the ID unchanged.
func TestStampVaultID_ExistingVaultKeepsRootGitignoreAndReusesID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(vault, ".gitignore"), []byte("custom\n"), 0o600)).To(Succeed())

	id, err := cli.ExportStampVaultID(cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, 5)), vault)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(id).To(Equal(seqID(5)))
	g.Expect(readFileString(t, filepath.Join(vault, ".gitignore"))).To(Equal("custom\n"))
	g.Expect(readFileString(t, filepath.Join(vault, ".engram", ".gitignore"))).To(Equal("*\n"))
	g.Expect(readHomeRecord(t, vault)).To(HaveKeyWithValue("vault_id", id))

	again, againErr := cli.ExportStampVaultID(cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, 99)), vault)
	g.Expect(againErr).NotTo(HaveOccurred())
	g.Expect(again).To(Equal(id))
}

// TestStampVaultID_LoserAdoptsWinnersID drives the race deterministically:
// the exclusive create reports the file already exists, and the loser
// re-reads it (retrying past a torn, still-empty read) and adopts the
// winner's ID without writing a location record of its own.
func TestStampVaultID_LoserAdoptsWinnersID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	winner := strings.Repeat("c3", 16)
	fakeFS := &raceLoserFS{reads: []string{absentRead, "", winner + "\n"}}
	state := cli.ExportNewExchangeState(fakeFS, seqRand(1), identityPath, func() (string, error) { return "/", nil })

	id, err := cli.ExportStampVaultID(state, "/vault")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(id).To(Equal(winner))
	g.Expect(fakeFS.exclAttempts).To(Equal([]string{"/vault/.engram-vault-id"}))
	g.Expect(fakeFS.atomicWrites).To(BeEmpty())
}

// TestStampVaultID_UsesInjectedRand: the ID is the hex of the injected
// random source's bytes (deterministic under a fake).
func TestStampVaultID_UsesInjectedRand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	id, err := cli.ExportStampVaultID(cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, 7)), t.TempDir())
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(id).To(Equal(seqID(7)))
	g.Expect(id).To(MatchRegexp(`^[0-9a-f]{32}$`))
}

// TestWarnVaultLocation_OneWarningNamingBothRemedies: a failing location
// check prints exactly one warning naming both `engram vault-id` remedies;
// a passing one prints nothing.
func TestWarnVaultLocation_OneWarningNamingBothRemedies(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := createdVault(t)
	state := cli.ExportExchangeStateFromDeps(exchangeDeps(io.Discard, 1))

	var quiet bytes.Buffer

	g.Expect(cli.ExportWarnVaultLocation(state, vault, &quiet)).To(BeTrue())
	g.Expect(quiet.String()).To(BeEmpty())

	g.Expect(os.Remove(filepath.Join(vault, ".engram", "home.json"))).To(Succeed())

	var loud bytes.Buffer

	g.Expect(cli.ExportWarnVaultLocation(state, vault, &loud)).To(BeFalse())

	lines := nonEmptyLines(loud.String())
	g.Expect(lines).To(HaveLen(1))
	g.Expect(lines[0]).To(ContainSubstring("engram vault-id --regenerate"))
	g.Expect(lines[0]).To(ContainSubstring("engram vault-id --claim"))
}

// unexported constants.
const (
	absentRead = "<absent>"
)

// raceLoserFS is an exchange-state filesystem whose exclusive create always
// loses (fs.ErrExist) and whose ID reads replay a scripted sequence.
type raceLoserFS struct {
	mu           sync.Mutex
	reads        []string
	exclAttempts []string
	atomicWrites []string
}

func (r *raceLoserFS) MkdirAll(string, fs.FileMode) error { return nil }

func (r *raceLoserFS) ReadFile(path string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !strings.HasSuffix(path, ".engram-vault-id") || len(r.reads) == 0 {
		return nil, fs.ErrNotExist
	}

	next := r.reads[0]
	if len(r.reads) > 1 {
		r.reads = r.reads[1:]
	}

	if next == absentRead {
		return nil, fs.ErrNotExist
	}

	return []byte(next), nil
}

func (r *raceLoserFS) WriteFileAtomic(path string, _ []byte, _ fs.FileMode) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.atomicWrites = append(r.atomicWrites, path)

	return nil
}

func (r *raceLoserFS) WriteFileExcl(path string, _ []byte, _ fs.FileMode) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.exclAttempts = append(r.exclAttempts, path)

	return fmt.Errorf("exclusive create: %w", fs.ErrExist)
}

// scriptedStateFS is an in-memory exchange-state filesystem with per-path
// scripted failures.
type scriptedStateFS struct {
	mu           sync.Mutex
	files        map[string]string
	failReads    map[string]error
	failExcl     map[string]error
	failAtomic   map[string]error
	mkdirErr     error
	readSequence []error
}

func (s *scriptedStateFS) MkdirAll(string, fs.FileMode) error { return s.mkdirErr }

func (s *scriptedStateFS) ReadFile(path string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.readSequence) > 0 {
		next := s.readSequence[0]
		s.readSequence = s.readSequence[1:]

		return nil, next
	}

	if err, failing := s.failReads[path]; failing {
		return nil, err
	}

	content, found := s.files[path]
	if !found {
		return nil, fs.ErrNotExist
	}

	return []byte(content), nil
}

func (s *scriptedStateFS) WriteFileAtomic(path string, data []byte, _ fs.FileMode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err, failing := s.failAtomic[path]; failing {
		return err
	}

	s.files[path] = string(data)

	return nil
}

func (s *scriptedStateFS) WriteFileExcl(path string, data []byte, _ fs.FileMode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err, failing := s.failExcl[path]; failing {
		return err
	}

	if _, exists := s.files[path]; exists {
		return fmt.Errorf("exclusive create %s: %w", path, fs.ErrExist)
	}

	s.files[path] = string(data)

	return nil
}

// canonicalPath mirrors design D2's EvalSymlinks(Abs(Clean(path))).
func canonicalPath(t *testing.T, path string) string {
	t.Helper()

	abs, absErr := filepath.Abs(filepath.Clean(path))
	if absErr != nil {
		t.Fatal(absErr)
	}

	resolved, evalErr := filepath.EvalSymlinks(abs)
	if evalErr != nil {
		t.Fatal(evalErr)
	}

	return resolved
}

// copyTree is a `cp -R` of src into dst (regular files and dirs only).
func copyTree(t *testing.T, src, dst string) {
	t.Helper()

	walkErr := filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}

		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o750)
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		return os.WriteFile(target, data, 0o600)
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
}

// createdVault returns a vault freshly created by ensureVault with an ID
// drawn from seqRand(1).
func createdVault(t *testing.T) string {
	t.Helper()

	vault := filepath.Join(t.TempDir(), "vault")

	ensureErr := cli.ExportEnsureVault(exchangeDeps(io.Discard, 1), vault)
	if ensureErr != nil {
		t.Fatal(ensureErr)
	}

	return vault
}

// exchangeDeps returns real-OS test deps whose RandRead is the deterministic
// seqRand(start) and whose Stderr is stderr.
func exchangeDeps(stderr io.Writer, start byte) cli.Deps {
	deps := newTestDeps(io.Discard, stderr)
	deps.RandRead = seqRand(start)

	return deps
}

// identityPath is an EvalSymlinks stand-in for fake filesystems.
func identityPath(path string) (string, error) { return path, nil }

func newScriptedStateFS() *scriptedStateFS {
	return &scriptedStateFS{
		files:      map[string]string{},
		failReads:  map[string]error{},
		failExcl:   map[string]error{},
		failAtomic: map[string]error{},
	}
}

func newScriptedStateFSWith(configure func(*scriptedStateFS)) *scriptedStateFS {
	fsys := newScriptedStateFS()
	configure(fsys)

	return fsys
}

func nonEmptyLines(text string) []string {
	lines := make([]string, 0)

	for line := range strings.SplitSeq(text, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

func readFileString(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func readHomeRecord(t *testing.T, vault string) map[string]string {
	t.Helper()

	record := map[string]string{}

	unmarshalErr := json.Unmarshal([]byte(readFileString(t, filepath.Join(vault, ".engram", "home.json"))), &record)
	if unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}

	return record
}

// scriptedState composes the adapter over fsys with seqRand(1), identity
// symlink resolution, and "/" as the working directory.
func scriptedState(fsys *scriptedStateFS) cli.ExchangeStateForTest {
	return cli.ExportNewExchangeState(fsys, seqRand(1), identityPath, func() (string, error) { return "/", nil })
}

// seqID is the ID seqRand(start) produces: hex of 16 bytes start, start+1, ...
func seqID(start byte) string {
	raw := make([]byte, 16)
	for i := range raw {
		raw[i] = start + byte(i)
	}

	return hex.EncodeToString(raw)
}

// seqRand is a deterministic RandRead fake filling bytes start, start+1, ...
func seqRand(start byte) func([]byte) (int, error) {
	return func(buf []byte) (int, error) {
		for i := range buf {
			buf[i] = start + byte(i)
		}

		return len(buf), nil
	}
}

// treeSnapshot maps every regular file under root (relative path) to its
// content, for byte-level "nothing changed" assertions.
func treeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()

	snapshot := map[string]string{}

	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}

		if entry.IsDir() {
			snapshot[rel+"/"] = ""

			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		snapshot[rel] = string(data)

		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, fs.ErrNotExist) {
		t.Fatal(walkErr)
	}

	return snapshot
}

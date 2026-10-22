package cli_test

// Scenario-ID tests of the Vault Identity Registry technical component
// (docs/specs/technical/vault-identity-registry.md): 792b5c4 Vault ID stamping, 792b5c5 Vault ID validation,
// 792b5c6 State directory, 792b5c11 vault-id report, 792b5c15 Exchange identifiers and 792b5c16 The
// declined-pull record. They drive the exchange-state adapter and RunVaultID over the in-memory scripted
// exchange-state filesystem of exchangestate_test.go, with a recording wrapper that logs every write.

import (
	"bytes"
	"io"
	"io/fs"
	"maps"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792b5c11a1_RunVaultID_TheReportPrintsTheIDAndTheCheckAndWritesNothing verifies 792b5c11a1
// (docs/specs/technical/vault-identity-registry.md): with no flag the report prints vault_id: with the ID
// and location: ok, and writes nothing.
func Test792b5c11a1_RunVaultID_TheReportPrintsTheIDAndTheCheckAndWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault with an ID at its recorded path
	fsys := newVir1RecordingFS()
	vaultID := seqID(1)
	fsys.files[vir1IDPath] = vaultID + "\n"
	fsys.files[vir1HomePath] = `{"vault_id":"` + vaultID + `","path":"` + vir1Vault + `"}` + "\n"
	before := maps.Clone(fsys.files)
	rand := &vir1RandCounter{inner: seqRand(1)}

	// When the vault-id report runs with no flag
	var stdout bytes.Buffer

	err := cli.RunVaultID(cli.VaultIDArgs{Vault: vir1Vault}, vir1Deps(fsys, rand, &stdout), &stdout)

	// Then it prints vault_id: with the ID and location: ok, and writes nothing
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(Equal("vault_id: " + vaultID + "\nlocation: ok\n"))
	g.Expect(fsys.writes).To(BeEmpty())
	g.Expect(fsys.files).To(Equal(before))
	g.Expect(rand.calls).To(BeZero())
}

// Test792b5c11a2_RunVaultID_BothFlagsAreRefusedAsMutuallyExclusiveAndNothingChanges verifies 792b5c11a2
// (docs/specs/technical/vault-identity-registry.md): --regenerate and --claim together fail as mutually
// exclusive and change nothing.
func Test792b5c11a2_RunVaultID_BothFlagsAreRefusedAsMutuallyExclusiveAndNothingChanges(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given both --regenerate and --claim, over a vault that holds an ID
	fsys := newVir1RecordingFS()
	fsys.files[vir1IDPath] = seqID(1) + "\n"
	before := maps.Clone(fsys.files)
	rand := &vir1RandCounter{inner: seqRand(50)}

	// When the vault-id command runs
	var stdout bytes.Buffer

	err := cli.RunVaultID(
		cli.VaultIDArgs{Vault: vir1Vault, Regenerate: true, Claim: true}, vir1Deps(fsys, rand, &stdout), &stdout)

	// Then it fails as mutually exclusive and changes nothing
	g.Expect(err).To(MatchError(ContainSubstring("mutually exclusive")))
	g.Expect(stdout.String()).To(BeEmpty())
	g.Expect(fsys.writes).To(BeEmpty())
	g.Expect(fsys.files).To(Equal(before))
	g.Expect(rand.calls).To(BeZero())
}

// Test792b5c15a1_IsExchangeID_OnlyA32CharacterLowercaseHexStringIsAnIdentifier verifies 792b5c15a1
// (docs/specs/technical/vault-identity-registry.md): of strings of the wrong length, with uppercase letters
// or with other characters, only a 32-character lowercase hexadecimal string is accepted.
func Test792b5c15a1_IsExchangeID_OnlyA32CharacterLowercaseHexStringIsAnIdentifier(t *testing.T) {
	t.Parallel()

	const hexLen = 32

	halfOfID := strings.Repeat("a", hexLen/2)

	cases := map[string]struct {
		candidate string
		accepted  bool
	}{
		"32 lowercase hex characters":    {candidate: seqID(1), accepted: true},
		"32 zeros":                       {candidate: strings.Repeat("0", hexLen), accepted: true},
		"empty":                          {candidate: "", accepted: false},
		"one character short":            {candidate: strings.Repeat("a", hexLen-1), accepted: false},
		"one character long":             {candidate: strings.Repeat("a", hexLen+1), accepted: false},
		"a trailing newline":             {candidate: seqID(1) + "\n", accepted: false},
		"32 uppercase hex characters":    {candidate: strings.Repeat("A", hexLen), accepted: false},
		"one uppercase hex character":    {candidate: "A" + strings.Repeat("a", hexLen-1), accepted: false},
		"a lowercase letter beyond f":    {candidate: "g" + strings.Repeat("a", hexLen-1), accepted: false},
		"a dash in place of a hex digit": {candidate: "-" + strings.Repeat("a", hexLen-1), accepted: false},
		"a path separator in the middle": {candidate: halfOfID + "/" + halfOfID[1:], accepted: false},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a string, When it is checked as an exchange identifier
			accepted := cli.ExportIsExchangeID(testCase.candidate)

			// Then only a 32-character lowercase hexadecimal string is accepted
			g.Expect(accepted).To(Equal(testCase.accepted))
		})
	}
}

// Test792b5c15a2_IsExchangeBasename_ABasenameWithASeparatorIsRefused verifies 792b5c15a2
// (docs/specs/technical/vault-identity-registry.md): a Luhmann basename is accepted, and the same
// basename containing /, \ or | is refused.
func Test792b5c15a2_IsExchangeBasename_ABasenameWithASeparatorIsRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		basename string
		accepted bool
	}{
		"the Luhmann basename":        {basename: vir1Basename, accepted: true},
		"a forward slash":             {basename: "12a3.some/slug", accepted: false},
		"a backslash":                 {basename: `12a3.some\slug`, accepted: false},
		"a pipe":                      {basename: "12a3.some|slug", accepted: false},
		"a pipe after a valid prefix": {basename: vir1Basename + "|supersedes|claim", accepted: false},
		"a backslash after the slug":  {basename: vir1Basename + `\`, accepted: false},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a Luhmann basename and the same basename containing a separator, When each is checked
			accepted := cli.ExportIsExchangeBasename(testCase.basename)

			// Then only the one without a separator is accepted
			g.Expect(accepted).To(Equal(testCase.accepted))
		})
	}
}

// Test792b5c16a1_RecordDeclinedPull_ADeclineIsRecordedOnceAndWrittenAtomically verifies 792b5c16a1
// (docs/specs/technical/vault-identity-registry.md): the same vault, basename and hash recorded twice
// leave one entry in declined.json, and the file is written atomically.
func Test792b5c16a1_RecordDeclinedPull_ADeclineIsRecordedOnceAndWrittenAtomically(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a declined-pull record (none yet: the first decline creates it)
	fsys := newVir1RecordingFS()
	state := vir1State(fsys, seqRand(1))
	entry := cli.ExportDeclinedPull{Vault: vir1ParentVault, Basename: vir1Basename, Hash: vir1Hash("a")}

	// When the same vault, basename and hash are recorded twice
	firstErr := cli.ExportRecordDeclinedPull(state, vir1Vault, entry)
	secondErr := cli.ExportRecordDeclinedPull(state, vir1Vault, entry)

	// Then the file holds one entry, and it is written atomically
	g.Expect(firstErr).NotTo(HaveOccurred())
	g.Expect(secondErr).NotTo(HaveOccurred())

	loaded, loadErr := cli.ExportLoadDeclined(state, vir1Vault)
	g.Expect(loadErr).NotTo(HaveOccurred())
	if loadErr != nil {
		return
	}
	g.Expect(loaded.Entries).To(Equal([]cli.ExportDeclinedPull{entry}))
	g.Expect(vir1Count(fsys.writes, "atomic:"+vir1DeclinedPath)).To(Equal(1))
}

// Test792b5c16a2_DeclineMatches_ADeclineMatchesByVaultNameAndHash verifies 792b5c16a2
// (docs/specs/technical/vault-identity-registry.md): of a decline recorded under one parent vault, a note
// with the same vault ID, a listed name and the same hash matches, and one whose vault ID, name or hash
// differs does not.
func Test792b5c16a2_DeclineMatches_ADeclineMatchesByVaultNameAndHash(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		vaultID string
		names   []string
		hash    string
		matches bool
	}{
		"the same vault ID, name and hash": {
			vaultID: vir1ParentVault, names: []string{vir1Basename}, hash: vir1Hash("a"), matches: true,
		},
		"the name is one of several the note answers to": {
			vaultID: vir1ParentVault, names: []string{"12a3.renamed", vir1Basename}, hash: vir1Hash("a"), matches: true,
		},
		"a different vault ID": {
			vaultID: "other-vault-id", names: []string{vir1Basename}, hash: vir1Hash("a"), matches: false,
		},
		"a different name": {
			vaultID: vir1ParentVault, names: []string{"12a4.other-slug"}, hash: vir1Hash("a"), matches: false,
		},
		"no name": {
			vaultID: vir1ParentVault, names: []string{}, hash: vir1Hash("a"), matches: false,
		},
		"a changed hash": {
			vaultID: vir1ParentVault, names: []string{vir1Basename}, hash: vir1Hash("b"), matches: false,
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a record of a decline under one parent vault
			record := cli.ExportDeclinedFile{Entries: []cli.ExportDeclinedPull{
				{Vault: vir1ParentVault, Basename: vir1Basename, Hash: vir1Hash("a")},
			}}

			// When a note is looked up, with the same or a different vault ID, name or hash
			matches := cli.ExportDeclineMatches(record, testCase.vaultID, testCase.names, testCase.hash)

			// Then only the one with the same vault ID, a listed name and the same hash matches
			g.Expect(matches).To(Equal(testCase.matches))
		})
	}
}

// Test792b5c16a3_DeclineMatches_AnUnknownHashComparisonStillMatches verifies 792b5c16a3
// (docs/specs/technical/vault-identity-registry.md): a decline whose recorded hash has an older version
// prefix matches the note's current hash, because unknown counts as not changed.
func Test792b5c16a3_DeclineMatches_AnUnknownHashComparisonStillMatches(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a decline whose recorded hash has an older version prefix
	olderHash := "xh0:" + strings.Repeat("a", vir1HashDigestLen)
	record := cli.ExportDeclinedFile{Entries: []cli.ExportDeclinedPull{
		{Vault: vir1ParentVault, Basename: vir1Basename, Hash: olderHash},
	}}

	// When the note's current hash is looked up
	matches := cli.ExportDeclineMatches(record, vir1ParentVault, []string{vir1Basename}, vir1Hash("b"))

	// Then the decline matches, because unknown counts as not changed
	g.Expect(matches).To(BeTrue())
}

// Test792b5c16a4_LoadDeclined_AMissingRecordIsEmptyAndACorruptOneIsAnError verifies 792b5c16a4
// (docs/specs/technical/vault-identity-registry.md): a missing declined.json reads as empty, and one that
// is not JSON fails.
func Test792b5c16a4_LoadDeclined_AMissingRecordIsEmptyAndACorruptOneIsAnError(t *testing.T) {
	t.Parallel()

	t.Run("a missing declined.json reads as empty", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given no declined.json
		state := vir1State(newVir1RecordingFS(), seqRand(1))

		// When it is read
		loaded, err := cli.ExportLoadDeclined(state, vir1Vault)

		// Then it reads as empty
		g.Expect(err).NotTo(HaveOccurred())
		if err != nil {
			return
		}
		g.Expect(loaded.Entries).To(BeEmpty())
	})

	t.Run("a declined.json that is not JSON fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given a declined.json that is not JSON
		fsys := newVir1RecordingFS()
		fsys.files[vir1DeclinedPath] = "{not json"

		// When it is read
		_, err := cli.ExportLoadDeclined(vir1State(fsys, seqRand(1)), vir1Vault)

		// Then it fails
		g.Expect(err).To(MatchError(ContainSubstring("declined.json")))
	})

	t.Run("a declined.json that cannot be read fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given a declined.json that cannot be read
		fsys := newVir1RecordingFS()
		fsys.failReads[vir1DeclinedPath] = errInjected

		// When it is read
		_, err := cli.ExportLoadDeclined(vir1State(fsys, seqRand(1)), vir1Vault)

		// Then it fails
		g.Expect(err).To(MatchError(errInjected))
	})
}

// Test792b5c4a4_StampVaultID_AnExistingIDIsReusedAndTheFileIsUnchanged verifies 792b5c4a4
// (docs/specs/technical/vault-identity-registry.md): stamping a vault with a valid ID file returns the
// existing ID and leaves the file unchanged.
func Test792b5c4a4_StampVaultID_AnExistingIDIsReusedAndTheFileIsUnchanged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault with a valid ID file
	fsys := newVir1RecordingFS()
	existing := seqID(9)
	fsys.files[vir1IDPath] = existing + "\n"
	before := maps.Clone(fsys.files)
	rand := &vir1RandCounter{inner: seqRand(70)}

	// When the vault ID is stamped
	stamped, err := cli.ExportStampVaultID(vir1State(fsys, rand.read), vir1Vault)

	// Then the existing ID is returned and the file is unchanged
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(stamped).To(Equal(existing))
	g.Expect(fsys.files).To(Equal(before))
	g.Expect(fsys.writes).To(BeEmpty())
	g.Expect(rand.calls).To(BeZero())
}

// Test792b5c5a2_ReadVaultID_AReadFailureIsReturned verifies 792b5c5a2
// (docs/specs/technical/vault-identity-registry.md): an ID file that cannot be read makes every reader of
// the vault ID return the read error.
func Test792b5c5a2_ReadVaultID_AReadFailureIsReturned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given an ID file that cannot be read
	fsys := newVir1RecordingFS()
	fsys.failReads[vir1IDPath] = errInjected
	state := vir1State(fsys, seqRand(1))

	// When the vault ID is read (by the location check, a claim, a stamp and the report)
	_, checkErr := cli.ExportCheckVaultLocation(state, vir1Vault)
	_, claimErr := cli.ExportClaimVaultLocation(state, vir1Vault)
	_, stampErr := cli.ExportStampVaultID(state, vir1Vault)

	var stdout bytes.Buffer

	rand := &vir1RandCounter{inner: seqRand(1)}
	reportErr := cli.RunVaultID(cli.VaultIDArgs{Vault: vir1Vault}, vir1Deps(fsys, rand, &stdout), &stdout)

	// Then the read error is returned
	for _, err := range []error{checkErr, claimErr, stampErr, reportErr} {
		g.Expect(err).To(MatchError(errInjected))
		g.Expect(err).To(MatchError(ContainSubstring("reading .engram-vault-id")))
	}

	g.Expect(stdout.String()).To(BeEmpty())
	g.Expect(fsys.writes).To(BeEmpty())
}

// Test792b5c6a2_EnsureStateDir_StateDirectoryFailuresAreReturned verifies 792b5c6a2
// (docs/specs/technical/vault-identity-registry.md): when .engram/ or its .gitignore cannot be created,
// ensuring the state directory returns the failure.
func Test792b5c6a2_EnsureStateDir_StateDirectoryFailuresAreReturned(t *testing.T) {
	t.Parallel()

	t.Run(".engram/ cannot be created", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given a filesystem where .engram/ cannot be created
		fsys := newVir1RecordingFS()
		fsys.mkdirErr = errInjected

		// When the state directory is ensured
		err := cli.ExportEnsureStateDir(vir1State(fsys, seqRand(1)), vir1Vault)

		// Then the failure is returned, and no .gitignore is written
		g.Expect(err).To(MatchError(errInjected))
		g.Expect(err).To(MatchError(ContainSubstring("vault state dir")))
		g.Expect(fsys.files).To(BeEmpty())
	})

	t.Run("its .gitignore cannot be created", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given a filesystem where .engram/.gitignore cannot be created
		fsys := newVir1RecordingFS()
		fsys.failExcl[vir1GitignorePath] = errInjected

		// When the state directory is ensured
		err := cli.ExportEnsureStateDir(vir1State(fsys, seqRand(1)), vir1Vault)

		// Then the failure is returned
		g.Expect(err).To(MatchError(errInjected))
		g.Expect(err).To(MatchError(ContainSubstring(".gitignore")))
	})
}

// unexported constants.
const (
	vir1Basename      = "12a3.some-slug"
	vir1DeclinedPath  = "/v/.engram/declined.json"
	vir1GitignorePath = "/v/.engram/.gitignore"
	vir1HashDigestLen = 64
	vir1HomePath      = "/v/.engram/home.json"
	vir1IDPath        = "/v/.engram-vault-id"
	vir1ParentVault   = "parent-vault-id"
	vir1Vault         = "/v"
)

// vir1RandCounter wraps a random source and counts its reads.
type vir1RandCounter struct {
	inner func([]byte) (int, error)
	calls int
}

func (c *vir1RandCounter) read(buf []byte) (int, error) {
	c.calls++

	return c.inner(buf)
}

// vir1RecordingFS is the scripted exchange-state filesystem with a log of every write it was asked for,
// each entry "mkdir:<path>", "excl:<path>" or "atomic:<path>".
type vir1RecordingFS struct {
	*scriptedStateFS

	writes []string
}

func (r *vir1RecordingFS) MkdirAll(path string, perm fs.FileMode) error {
	r.writes = append(r.writes, "mkdir:"+path)

	return r.scriptedStateFS.MkdirAll(path, perm)
}

func (r *vir1RecordingFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	r.writes = append(r.writes, "atomic:"+path)

	return r.scriptedStateFS.WriteFileAtomic(path, data, perm)
}

func (r *vir1RecordingFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	r.writes = append(r.writes, "excl:"+path)

	return r.scriptedStateFS.WriteFileExcl(path, data, perm)
}

func newVir1RecordingFS() *vir1RecordingFS {
	return &vir1RecordingFS{scriptedStateFS: newScriptedStateFS()}
}

// vir1Count is how many entries of log equal entry.
func vir1Count(log []string, entry string) int {
	count := 0

	for _, logged := range log {
		if logged == entry {
			count++
		}
	}

	return count
}

// vir1Deps composes the production carrier over fsys: every filesystem call RunVaultID makes goes to fsys.
func vir1Deps(fsys *vir1RecordingFS, rand *vir1RandCounter, stdout io.Writer) cli.Deps {
	return cli.Deps{
		Stdout: stdout,
		Stderr: io.Discard,
		FS: fakeEdgeFS{
			readFile:        fsys.ReadFile,
			writeFileAtomic: fsys.WriteFileAtomic,
			writeFileExcl:   fsys.WriteFileExcl,
			mkdirAll:        fsys.MkdirAll,
		},
		RandRead:     rand.read,
		EvalSymlinks: identityPath,
		Getwd:        func() (string, error) { return "/", nil },
	}
}

// vir1Hash is a well-formed exchange hash of the current version whose digest is 64 copies of digit.
func vir1Hash(digit string) string {
	return cli.ExportExchangeHashPrefix + strings.Repeat(digit, vir1HashDigestLen)
}

// vir1State composes the exchange-state adapter over fsys with the given random source, identity symlink
// resolution and "/" as the working directory.
func vir1State(fsys *vir1RecordingFS, rand func([]byte) (int, error)) cli.ExchangeStateForTest {
	return cli.ExportNewExchangeState(fsys, rand, identityPath, func() (string, error) { return "/", nil })
}

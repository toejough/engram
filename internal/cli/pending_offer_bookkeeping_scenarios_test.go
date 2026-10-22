package cli_test

// Scenario tests of the pending-offer-bookkeeping technical component (792b34 Pending Offer
// Bookkeeping): the vault scan, the write-path nudge, the judged-version check, the parent-link
// merge of a fold and the show header. Scenarios that already have a test elsewhere are not repeated.

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// Test792b34c3a2_VaultHasPendingOffers_NormalNotesNotFlagged: a vault holding only normal
// notes (a fact, and a skill runbook that is not pending) is not reported as holding a
// pending offer.
func Test792b34c3a2_VaultHasPendingOffers_NormalNotesNotFlagged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fileSystem := newU1FS()
	fileSystem.files["/vault/2.2026-01-02.a.md"] = []byte(normalFactNote)
	fileSystem.files["/vault/6.2026-01-06.b.md"] = []byte(nonPendingSkillRunbookNote)

	g.Expect(cli.ExportVaultHasPendingOffers("/vault", fileSystem)).To(BeFalse())
}

// Test792b34c3a3_VaultHasPendingOffers_MissingDirectoryIsSilent: a vault directory that does
// not exist, or cannot be read, reports no pending offer and the scan raises no error (the
// scan returns only the flag, so nothing can surface; an unreadable directory must not panic).
func Test792b34c3a3_VaultHasPendingOffers_MissingDirectoryIsSilent(t *testing.T) {
	t.Parallel()

	t.Run("directory does not exist", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		g.Expect(cli.ExportVaultHasPendingOffers("/vault/absent", newU1FS())).To(BeFalse())
	})

	t.Run("directory cannot be read", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fileSystem := &pobUnreadableFS{u1FS: newU1FS()}
		fileSystem.files["/vault/1.2026-01-01.a.md"] = []byte(pendingFactNote)

		g.Expect(cli.ExportVaultHasPendingOffers("/vault", fileSystem)).To(BeFalse())
	})
}

// Test792b34c3a5_WarnIfPendingOffers_SilentWithoutPendingOffers: a vault whose notes are all
// normal logs nothing at the write-path nudge, while the notes were scanned.
func Test792b34c3a5_WarnIfPendingOffers_SilentWithoutPendingOffers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := map[string][]byte{
		"/vault/2.2026-01-02.a.md": []byte(normalFactNote),
		"/vault/6.2026-01-06.b.md": []byte(nonPendingSkillRunbookNote),
	}
	listMD := func(string) ([]string, error) { return []string{"2.2026-01-02.a.md", "6.2026-01-06.b.md"}, nil }

	var (
		reads    []string
		warnings []string
	)

	read := func(path string) ([]byte, error) {
		reads = append(reads, path)

		return files[path], nil
	}
	logWarn := func(format string, args ...any) { warnings = append(warnings, fmt.Sprintf(format, args...)) }

	cli.ExportWarnIfPendingOffers("/vault", listMD, read, logWarn)

	g.Expect(warnings).To(BeEmpty())
	g.Expect(reads).To(ConsistOf("/vault/2.2026-01-02.a.md", "/vault/6.2026-01-06.b.md"),
		"the silence comes from scanning every note, not from skipping the scan")
}

// Test792b34c3a6_WarnIfPendingOffers_MissingDependencyIsNoOp: with the note lister missing
// (and, by the same requirement, the reader or the logger), the nudge logs nothing and does
// not panic, even though the vault holds a pending offer.
func Test792b34c3a6_WarnIfPendingOffers_MissingDependencyIsNoOp(t *testing.T) {
	t.Parallel()

	files := map[string][]byte{"/vault/1.2026-01-01.a.md": []byte(pendingFactNote)}
	listMD := func(vault string) ([]string, error) {
		if vault == pobUnlistableVault {
			return nil, errPobUnreadableDir
		}

		return []string{"1.2026-01-01.a.md"}, nil
	}
	read := func(path string) ([]byte, error) {
		data, found := files[path]
		if !found {
			return nil, fs.ErrNotExist
		}

		return data, nil
	}

	t.Run("note lister fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var warnings []string

		logWarn := func(format string, args ...any) { warnings = append(warnings, fmt.Sprintf(format, args...)) }

		cli.ExportWarnIfPendingOffers(pobUnlistableVault, listMD, read, logWarn)
		g.Expect(warnings).To(BeEmpty())
	})

	cases := []struct {
		name    string
		missing string
	}{
		{name: "note lister missing", missing: "listMD"},
		{name: "note reader missing", missing: "read"},
		{name: "logger missing", missing: "logWarn"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			var warnings []string

			logWarn := func(format string, args ...any) { warnings = append(warnings, fmt.Sprintf(format, args...)) }

			switch tc.missing {
			case "listMD":
				g.Expect(func() { cli.ExportWarnIfPendingOffers("/vault", nil, read, logWarn) }).NotTo(Panic())
			case "read":
				g.Expect(func() { cli.ExportWarnIfPendingOffers("/vault", listMD, nil, logWarn) }).NotTo(Panic())
			default:
				g.Expect(func() { cli.ExportWarnIfPendingOffers("/vault", listMD, read, nil) }).NotTo(Panic())
			}

			g.Expect(warnings).To(BeEmpty())
		})
	}
}

// Test792b34c5a4_RunAmend_UncomputableHashFails: an expected hash on a note whose current
// exchange hash cannot be computed (its frontmatter does not parse) fails the amend in every
// bookkeeping mode, changes no file and records no decline.
func Test792b34c5a4_RunAmend_UncomputableHashFails(t *testing.T) {
	t.Parallel()

	const unparseable = "---\ntype: fact\nsituation: [unclosed\n---\n\nbody\n"

	expected := "xh1:" + strings.Repeat("0", 64)

	modes := map[string]cli.AmendArgs{
		"clear-pending": {ClearPending: true, Pending: new(bool)},
		"discard":       {Discard: true},
		"discard-into":  {Discard: true, Into: foldExistingBase},
		"activate":      {Activate: true},
	}

	for mode, base := range modes {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newFoldVault()
			vault.put(foldExistingName, foldNote(foldExchange{}, false))
			vault.put(foldOfferName, unparseable)
			vault.put(strings.TrimSuffix(foldOfferName, ".md")+".vec.json", validFoldSidecar())
			before := maps.Clone(vault.files)

			args := base
			args.Target = foldOfferBase
			args.ExpectHash = expected

			err := vault.amend(t.Context(), args)

			g.Expect(err).To(MatchError(ContainSubstring("exchange hash: parsing frontmatter")))
			g.Expect(vault.files).To(Equal(before), "a failed check changes nothing")
			g.Expect(vault.declines).To(BeEmpty())
		})
	}
}

// Test792b34c7a1_RunAmend_FoldKeepsLinkRoleAndTakesOfferHash: E holds its own primary link to
// Q and a covered link to P; O's primary link is to P under the same vault. After the fold E's
// primary is still Q, and its link to P is still covered and carries O's hash for P.
func Test792b34c7a1_RunAmend_FoldKeepsLinkRoleAndTakesOfferHash(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newFoldVault()
	vault.put(foldExistingName, foldNote(foldExchange{
		vault: foldParentVault,
		links: []foldLink{{"9.2026-09-01.q", "offered", "xh1:q"}, {"7.2026-09-01.p", "covered", "xh1:p-old"}},
	}, false))
	vault.put(foldOfferName, foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault,
		links: []foldLink{{"7.2026-09-01.p", "pulled", "xh1:p-new"}},
	}, true))

	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	folded := decodeFoldExchange(t, vault.files[foldExistingName])
	g.Expect(folded.Parent.Vault).To(Equal(foldParentVault))
	g.Expect(folded.Parent.Links).To(Equal([]foldLinkYAML{
		{Note: "9.2026-09-01.q", Via: "offered", Hash: "xh1:q"},
		{Note: "7.2026-09-01.p", Via: "covered", Hash: "xh1:p-new"},
	}))
}

// Test792b34c7a2_RunAmend_FoldVaultSwitchDropsOldVaultLinks: E holds links under vault V1 and
// O's links are all under vault V2. After the fold E's parent.vault is V2, none of its V1 links
// remain and its links are O's links.
func Test792b34c7a2_RunAmend_FoldVaultSwitchDropsOldVaultLinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newFoldVault()
	vault.put(foldExistingName, foldNote(foldExchange{
		vault: foldOtherVault,
		links: []foldLink{{"9.2026-09-01.old", "offered", "xh1:old"}, {"8.2026-09-01.near", "covered", "xh1:near-old"}},
	}, false))
	vault.put(foldOfferName, foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault,
		links: []foldLink{{"7.2026-09-01.p", "pulled", "xh1:p"}, {"8.2026-09-01.near", "covered", "xh1:near-new"}},
	}, true))

	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	folded := decodeFoldExchange(t, vault.files[foldExistingName])
	g.Expect(folded.Parent.Vault).To(Equal(foldParentVault))
	g.Expect(folded.Parent.Links).To(Equal([]foldLinkYAML{
		{Note: "7.2026-09-01.p", Via: "pulled", Hash: "xh1:p"},
		{Note: "8.2026-09-01.near", Via: "covered", Hash: "xh1:near-new"},
	}))
}

// Test792b34c8a2_Show_NoHeaderWithoutXID: a note that carries no xid is shown exactly as
// before the header existed (the note, then the outbound-links tail) and holds no
// exchange_hash line.
func Test792b34c8a2_Show_NoHeaderWithoutXID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	plain := foldNote(foldExchange{}, false)

	memFS := newInMemoryFS()
	memFS.files["/vault/"+foldExistingName] = []byte(plain)

	var shown bytes.Buffer

	err := cli.RunShow(t.Context(), cli.ShowArgs{Ref: "2", VaultPath: "/vault"}, newShowDeps(memFS), &shown)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(shown.String()).To(Equal(plain + "\n# outbound links (fetch with: engram show <basename>)\n(none)\n"))
	g.Expect(shown.String()).NotTo(ContainSubstring("exchange_hash"))
}

// unexported constants.
const (
	// pobUnlistableVault is a vault path whose note listing fails.
	pobUnlistableVault = "/vault/unlistable"
)

// unexported variables.
var (
	errPobUnreadableDir = errors.New("pending_offer_bookkeeping_scenarios_test: directory unreadable")
)

// pobUnreadableFS is a filesystem whose directories cannot be read, though they exist.
type pobUnreadableFS struct {
	*u1FS
}

func (f *pobUnreadableFS) ReadDir(string) ([]update.DirEntry, error) {
	return nil, fmt.Errorf("read dir: %w", errors.Join(errPobUnreadableDir, fs.ErrPermission))
}

package cli_test

// Curation fold and the judged-version check (design D10; tasks 8.1, 8.2;
// spec vault-offer-curation "Bookkeeping on a served offer SHALL verify the
// judged version" and "Curation judges pending offers the same way recall
// judges candidates"; spec vault-note-identity D4 link roles).

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestRunAmend_DiscardIntoFoldsPulledCopy is the covered/near fold of a
// pulled-down copy into a local note that already has a primary: O and its
// sidecar are deleted; E gains O's basename and aliases; O's pulled primary
// becomes a covered link on E and O's covered link rides along; nothing is
// re-embedded, re-stamped (ruling S7), queued or declined; every other byte
// of E is unchanged.
func TestRunAmend_DiscardIntoFoldsPulledCopy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newFoldVault()
	existing := foldNote(foldExchange{
		xid: foldXIDE, vault: foldParentVault, aliases: []string{"2.2026-01-01.old-name"},
		links: []foldLink{{"9.2026-09-01.other", "offered", "xh1:other"}},
	}, false)
	vault.put(foldExistingName, existing)
	vault.put(foldOfferName, foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault, aliases: []string{"5.2026-01-01.prev-name"},
		links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}, {"8.2026-09-01.near", "covered", "xh1:near"}},
	}, true))
	vault.put(strings.TrimSuffix(foldOfferName, ".md")+".vec.json", []byte(`{}`))

	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(vault.files).NotTo(HaveKey(foldOfferName))
	g.Expect(vault.files).NotTo(HaveKey(strings.TrimSuffix(foldOfferName, ".md") + ".vec.json"))

	folded := decodeFoldExchange(t, vault.files[foldExistingName])
	g.Expect(folded.Aliases).To(Equal([]string{"2.2026-01-01.old-name", foldOfferBase, "5.2026-01-01.prev-name"}))
	g.Expect(folded.Parent.Vault).To(Equal(foldParentVault))
	g.Expect(folded.Parent.Links).To(Equal([]foldLinkYAML{
		{Note: "9.2026-09-01.other", Via: "offered", Hash: "xh1:other"},
		{Note: "7.2026-09-01.parent", Via: "covered", Hash: "xh1:parent"},
		{Note: "8.2026-09-01.near", Via: "covered", Hash: "xh1:near"},
	}))
	g.Expect(folded.XID).To(Equal(foldXIDE), "E keeps its own xid")

	expectOnlyExchangeKeysChanged(g, existing, vault.files[foldExistingName])
	g.Expect(vault.embeds).To(BeZero(), "a fold never re-embeds E")
	g.Expect(vault.declines).To(BeEmpty(), "a fold is not a decline")
	g.Expect(vault.detected).To(BeZero(), "a fold never runs identity detection (S7)")
}

// TestRunAmend_DiscardIntoMakesOfferPrimaryWhenTargetHasNone: E with no
// parent block adopts O's parent vault, and O's primary stays primary.
func TestRunAmend_DiscardIntoMakesOfferPrimaryWhenTargetHasNone(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newFoldVault()
	vault.put(foldExistingName, foldNote(foldExchange{}, false))
	vault.put(foldOfferName, foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault,
		links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}},
	}, true))

	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
	g.Expect(err).NotTo(HaveOccurred())

	folded := decodeFoldExchange(t, vault.files[foldExistingName])
	g.Expect(folded.Aliases).To(Equal([]string{foldOfferBase}))
	g.Expect(folded.Parent.Vault).To(Equal(foldParentVault))
	g.Expect(folded.Parent.Links).To(Equal([]foldLinkYAML{
		{Note: "7.2026-09-01.parent", Via: "pulled", Hash: "xh1:parent"},
	}))
	g.Expect(folded.XID).To(BeEmpty(), "a fold stamps no xid on E")
}

// TestRunAmend_DiscardIntoPropertyUnionsIdentity is 8.1's rule over random
// alias and link sets (postcondition oracles derived from D10/D4, not from
// the implementation): E ends with every alias it had plus O's basename and
// aliases, without duplicates or its own basename; every link O held is on
// E under O's parent vault; E keeps its own primary when it had one under
// that vault, and never holds two primaries; O is gone; and nothing outside
// E's parent/aliases keys changes.
func TestRunAmend_DiscardIntoPropertyUnionsIdentity(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		existing := drawFoldExchange(rt, "existing")
		offer := drawFoldExchange(rt, "offer")
		offer.xid = foldXIDO

		vault := newFoldVault()
		before := foldNote(existing, false)
		vault.put(foldExistingName, before)
		vault.put(foldOfferName, foldNote(offer, true))

		err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
		if err != nil {
			rt.Fatalf("fold: %v", err)
		}

		if _, present := vault.files[foldOfferName]; present {
			rt.Fatalf("O survived the fold")
		}

		folded := decodeFoldExchange(rt, vault.files[foldExistingName])
		assertFoldAliases(rt, existing, offer, folded.Aliases)
		assertFoldLinks(rt, existing, offer, folded.Parent)

		probe := NewGomega(func(message string, _ ...int) { rt.Fatalf("%s", message) })
		expectOnlyExchangeKeysChanged(probe, before, vault.files[foldExistingName])
	})
}

// TestRunAmend_DiscardIntoRejectsBadTargets: --into without --discard, an
// --into naming no note, and an --into naming the offer itself all fail and
// change nothing.
func TestRunAmend_DiscardIntoRejectsBadTargets(t *testing.T) {
	t.Parallel()

	for name, args := range map[string]cli.AmendArgs{
		"into without discard": {Target: foldOfferBase, Into: foldExistingBase, Activate: true},
		"into a missing note":  {Target: foldOfferBase, Discard: true, Into: "99.2026-01-01.none"},
		"into itself":          {Target: foldOfferBase, Discard: true, Into: foldOfferBase},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newFoldVault()
			vault.put(foldExistingName, foldNote(foldExchange{}, false))
			vault.put(foldOfferName, foldNote(foldExchange{
				xid: foldXIDO, vault: foldParentVault, links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:p"}},
			}, true))
			before := maps.Clone(vault.files)

			err := vault.amend(t.Context(), args)
			g.Expect(err).To(HaveOccurred())
			g.Expect(vault.files).To(Equal(before))
			g.Expect(vault.declines).To(BeEmpty())
		})
	}
}

// TestRunAmend_ExpectHashTruthTable is 8.2's rule checked over its whole
// input domain (every bookkeeping mode × origin × pending × expected-hash
// kind): the amend succeeds exactly when the expected hash equals the
// note's current exchange hash, or when none is given and the note does not
// carry offer.origin under a mode that requires one (--clear-pending,
// --discard, --discard --into). A failure changes no file and records no
// decline. An expected hash of another version is not the judged version,
// so it fails like a changed one (spec: fail "when the note's current
// exchange hash differs from that value").
func TestRunAmend_ExpectHashTruthTable(t *testing.T) {
	t.Parallel()

	modes := map[string]cli.AmendArgs{
		"clear-pending": {ClearPending: true},
		"discard":       {Discard: true},
		"discard-into":  {Discard: true, Into: foldExistingBase},
		"activate":      {Activate: true},
	}
	requiring := []string{"clear-pending", "discard", "discard-into"}

	for mode, base := range modes {
		for _, origin := range []bool{false, true} {
			for _, pending := range []bool{false, true} {
				for _, expect := range []string{"none", "correct", "changed", "other-version"} {
					t.Run(strings.Join([]string{mode, boolName("origin", origin), boolName("pending", pending), expect}, "/"),
						func(t *testing.T) {
							t.Parallel()
							g := NewWithT(t)

							vault := newFoldVault()
							vault.put(foldExistingName, foldNote(foldExchange{}, false))
							offer := foldNote(foldExchange{
								xid: foldXIDO, vault: foldParentVault, origin: origin,
								links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:p"}},
							}, pending)
							vault.put(foldOfferName, offer)
							vault.put(strings.TrimSuffix(foldOfferName, ".md")+".vec.json", []byte(`{"last_used":"2026-01-01"}`))
							before := maps.Clone(vault.files)

							args := base
							args.Target = foldOfferBase
							args.ExpectHash = expectedHashFor(t, offer, expect)

							err := vault.amend(t.Context(), args)

							succeeds := expect == "correct" || (expect == "none" && (!origin || !slices.Contains(requiring, mode)))
							if succeeds {
								g.Expect(err).NotTo(HaveOccurred())

								return
							}

							g.Expect(err).To(HaveOccurred())
							g.Expect(vault.files).To(Equal(before), "a failed check writes nothing")
							g.Expect(vault.declines).To(BeEmpty())
						})
				}
			}
		}
	}
}

// TestRunAmend_OfferUpdatedAfterJudgmentIsNotAccepted is the spec scenario
// "An offer updated after judgment is not silently accepted": judged at H1,
// updated in place to H2, clear-pending with H1 fails and N stays pending
// with the H2 content.
func TestRunAmend_OfferUpdatedAfterJudgmentIsNotAccepted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	judged := foldNote(foldExchange{xid: foldXIDO, origin: true}, true)
	updated := strings.Replace(judged, "object: n", "object: updated", 1)

	vault := newFoldVault()
	vault.put(foldOfferName, updated)

	hash, hashErr := cli.ExportExchangeHash([]byte(judged))
	g.Expect(hashErr).NotTo(HaveOccurred())

	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, ClearPending: true, ExpectHash: hash})
	g.Expect(err).To(HaveOccurred())
	g.Expect(string(vault.files[foldOfferName])).To(Equal(updated))
	g.Expect(string(vault.files[foldOfferName])).To(ContainSubstring("pending: true"))
}

// TestRunAmend_UnparseableNoteNeedsNoHash: a note whose frontmatter does
// not parse has no readable offer.origin, so a bare discard needs no
// --expect-hash.
func TestRunAmend_UnparseableNoteNeedsNoHash(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newFoldVault()
	vault.put(foldOfferName, "---\ntype: fact\nsituation: [unclosed\n---\n\nbody\n")

	g.Expect(vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true})).To(Succeed())
	g.Expect(vault.files).NotTo(HaveKey(foldOfferName))
}

// TestServeShow_ExchangeHashHeaderMatchesLocal: the served non-raw /show
// returns exactly the local output, header included.
func TestServeShow_ExchangeHashHeaderMatchesLocal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	writeFoldFile(t, filepath.Join(vault, foldOfferName), foldNote(foldExchange{xid: foldXIDO}, true))

	deps := newTestDeps(&strings.Builder{}, &strings.Builder{})

	var local bytes.Buffer

	g.Expect(cli.RunShow(t.Context(), cli.ShowArgs{Ref: foldOfferBase, VaultPath: vault},
		cli.ExportNewShowDeps(deps.FS), &local)).To(Succeed())

	resp := routeFor(t, cli.ServeRoutes(deps, vault, "personal", t.TempDir()), "/show").
		Serve(t.Context(), cli.ServeRequest{Query: map[string][]string{"note": {foldOfferBase}}})

	g.Expect(resp.Status).To(Equal(200))
	g.Expect(string(resp.Body)).To(Equal(local.String()))
	g.Expect(local.String()).To(HavePrefix("# exchange_hash: xh1:"))
}

// TestShowFallback_LabelPrecedesParentExchangeHash is the spec scenario
// "Label order on the parent fallback", end to end: the child's local miss
// reaches a real served /show over the parent's vault.
func TestShowFallback_LabelPrecedesParentExchangeHash(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	parentVault := t.TempDir()
	writeFoldFile(t, filepath.Join(parentVault, foldOfferName), foldNote(foldExchange{xid: foldXIDO}, false))

	parentDeps := newTestDeps(&strings.Builder{}, &strings.Builder{})
	show := routeFor(t, cli.ServeRoutes(parentDeps, parentVault, "personal", t.TempDir()), "/show")

	stdout, stderr := executeCapturingBoth(t, []string{"engram", "show", foldOfferBase, "--vault", t.TempDir()},
		func(d *cli.Deps) {
			d.Fetch = func(ctx context.Context, _, _ string, _ []byte) (cli.FetchResponse, error) {
				resp := show.Serve(ctx, cli.ServeRequest{Query: map[string][]string{"note": {foldOfferBase}}})

				return cli.FetchResponse(resp), nil
			}
		})

	g.Expect(stderr).To(BeEmpty())

	lines := strings.SplitN(stdout, "\n", 3)
	g.Expect(lines).To(HaveLen(3))

	if len(lines) < 3 {
		return
	}

	g.Expect(lines[0]).To(Equal("# from_parent: true"))
	g.Expect(lines[1]).To(HavePrefix("# exchange_hash: xh1:"))
}

// TestShow_ExchangeHashHeaderOnlyForXIDNotes covers the spec scenarios
// "Header on an exchanged note" and "No header on an unexchanged note".
func TestShow_ExchangeHashHeaderOnlyForXIDNotes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	exchanged := foldNote(foldExchange{xid: foldXIDO, origin: true}, true)
	plain := foldNote(foldExchange{}, false)

	memFS := newInMemoryFS()
	memFS.files["/vault/"+foldOfferName] = []byte(exchanged)
	memFS.files["/vault/"+foldExistingName] = []byte(plain)

	hash, hashErr := cli.ExportExchangeHash([]byte(exchanged))
	g.Expect(hashErr).NotTo(HaveOccurred())

	var withXID, without bytes.Buffer

	g.Expect(cli.RunShow(t.Context(), cli.ShowArgs{Ref: "5", VaultPath: "/vault"}, newShowDeps(memFS), &withXID)).
		To(Succeed())
	g.Expect(cli.RunShow(t.Context(), cli.ShowArgs{Ref: "2", VaultPath: "/vault"}, newShowDeps(memFS), &without)).
		To(Succeed())

	lines := strings.SplitN(withXID.String(), "\n", 3)
	g.Expect(lines).To(HaveLen(3))

	if len(lines) < 3 {
		return
	}

	g.Expect(lines[0]).To(Equal("# exchange_hash: " + hash))
	g.Expect(lines[1]).To(Equal("---"))
	g.Expect(strings.TrimPrefix(withXID.String(), lines[0]+"\n")).To(HavePrefix(exchanged))

	g.Expect(without.String()).To(HavePrefix(plain), "a note without xid shows exactly as before")
	g.Expect(without.String()).NotTo(ContainSubstring("exchange_hash"))
}

// unexported constants.
const (
	foldExistingBase = "2.2026-09-20.existing"
	foldExistingName = foldExistingBase + ".md"
	foldOfferBase    = "5.2026-09-28.offer"
	foldOfferName    = foldOfferBase + ".md"
	foldOtherVault   = "0123456789abcdef0123456789abcdef"
	foldParentVault  = "9a1e2b3c4d5e6f708192a3b4c5d6e7f8"
	foldXIDE         = "1111111111111111111111111111111e"
	foldXIDO         = "0000000000000000000000000000000f"
)

// foldExchange is a fixture note's exchange frontmatter.
type foldExchange struct {
	xid     string
	vault   string
	aliases []string
	links   []foldLink
	origin  bool
}

// foldExchangeYAML decodes a note's exchange frontmatter.
type foldExchangeYAML struct {
	XID     string         `yaml:"xid"`
	Aliases []string       `yaml:"aliases"`
	Parent  foldParentYAML `yaml:"parent"`
}

type foldLink struct {
	note, via, hash string
}

type foldLinkYAML struct {
	Note string `yaml:"note"`
	Via  string `yaml:"via"`
	Hash string `yaml:"hash"`
}

// foldParentYAML decodes a note's parent block.
type foldParentYAML struct {
	Vault string         `yaml:"vault"`
	Links []foldLinkYAML `yaml:"links"`
}

// foldVault is an in-memory vault driving RunAmend.
type foldVault struct {
	files    map[string][]byte
	embeds   int
	detected int
	declines []cli.ExportDeclinedPull
}

func (v *foldVault) amend(ctx context.Context, args cli.AmendArgs) error {
	args.Vault = "/vault"

	var out bytes.Buffer

	return cli.RunAmend(ctx, args, v.deps(), &out)
}

func (v *foldVault) deps() cli.AmendDeps {
	return cli.AmendDeps{
		DetectRepo: func(context.Context) string { v.detected++; return "github.com/bob/gadgets" },
		DetectUser: func(context.Context) string { v.detected++; return "bob@example.com" },
		Scan: func(string) ([]vaultgraph.Note, error) {
			names := slices.Sorted(maps.Keys(v.files))
			notes := make([]vaultgraph.Note, 0, len(names))

			for _, name := range names {
				if base, isNote := strings.CutSuffix(name, ".md"); isNote {
					id, _, _ := strings.Cut(base, ".")
					notes = append(notes, vaultgraph.Note{Basename: base, LuhmannID: id})
				}
			}

			return notes, nil
		},
		Read: func(path string) ([]byte, error) {
			data, ok := v.files[filepath.Base(path)]
			if !ok {
				return nil, fs.ErrNotExist
			}

			return data, nil
		},
		Write: func(path string, data []byte) error {
			if strings.HasSuffix(path, ".vec.json") {
				v.embeds++
			}

			v.files[filepath.Base(path)] = data

			return nil
		},
		Remove: func(path string) error {
			if _, ok := v.files[filepath.Base(path)]; !ok {
				return fs.ErrNotExist
			}

			delete(v.files, filepath.Base(path))

			return nil
		},
		RecordDecline: func(_ string, entry cli.ExportDeclinedPull) error {
			v.declines = append(v.declines, entry)

			return nil
		},
		Now: func() time.Time { return time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC) },
	}
}

func (v *foldVault) put(name string, content any) {
	switch typed := content.(type) {
	case string:
		v.files[name] = []byte(typed)
	case []byte:
		v.files[name] = typed
	}
}

// assertFoldAliases checks E's aliases after the fold: every alias E had,
// then O's basename and aliases, with no duplicate and never E's own name.
func assertFoldAliases(rt *rapid.T, existing, offer foldExchange, got []string) {
	want := make([]string, 0, len(existing.aliases)+len(offer.aliases)+1)
	want = append(want, existing.aliases...)
	want = append(want, foldOfferBase)
	want = append(want, offer.aliases...)

	for _, name := range want {
		if name != foldExistingBase && !slices.Contains(got, name) {
			rt.Fatalf("alias %s missing from %v", name, got)
		}
	}

	if slices.Contains(got, foldExistingBase) {
		rt.Fatalf("E answers to its own name as an alias: %v", got)
	}

	sorted := slices.Sorted(slices.Values(got))
	if len(slices.Compact(sorted)) != len(got) {
		rt.Fatalf("duplicate aliases: %v", got)
	}
}

// assertFoldLinks checks E's parent block after the fold.
func assertFoldLinks(rt *rapid.T, existing, offer foldExchange, got foldParentYAML) {
	primaries := 0

	for _, link := range got.Links {
		if isPrimaryVia(link.Via) {
			primaries++
		}
	}

	if primaries > 1 {
		rt.Fatalf("two primaries: %v", got.Links)
	}

	if len(offer.links) == 0 {
		if !linksEqual(existing, got) {
			rt.Fatalf("O had no links, yet E's parent changed: %v", got)
		}

		return
	}

	if got.Vault != offer.vault {
		rt.Fatalf("E's parent vault %q, want O's %q", got.Vault, offer.vault)
	}

	assertFoldLinksCarried(rt, existing, offer, got)
}

// assertFoldLinksCarried: every link O held is on E, and — under a shared
// parent vault — every link E held survives with its primary intact.
func assertFoldLinksCarried(rt *rapid.T, existing, offer foldExchange, got foldParentYAML) {
	names := make([]string, 0, len(got.Links))
	for _, link := range got.Links {
		names = append(names, link.Note)
	}

	for _, link := range offer.links {
		if !slices.Contains(names, link.note) {
			rt.Fatalf("O's link %s missing from E: %v", link.note, got.Links)
		}
	}

	if existing.vault != offer.vault {
		return
	}

	for _, link := range existing.links {
		index := slices.Index(names, link.note)
		if index < 0 {
			rt.Fatalf("E's own link %s lost: %v", link.note, got.Links)
		}

		if isPrimaryVia(link.via) && got.Links[index].Via != link.via {
			rt.Fatalf("E's primary %s demoted: %v", link.note, got.Links)
		}
	}
}

func boolName(name string, value bool) string {
	if value {
		return name
	}

	return "no-" + name
}

func decodeFoldExchange(t failer, raw []byte) foldExchangeYAML {
	t.Helper()

	var decoded foldExchangeYAML

	if len(raw) == 0 {
		t.Fatal(errors.New("no note"))

		return decoded
	}

	frontmatter, _, found := embed.SplitFrontmatter(raw)
	if !found {
		t.Fatal(errors.New("no frontmatter"))

		return decoded
	}

	err := yaml.Unmarshal(frontmatter, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	return decoded
}

// drawFoldExchange draws a note's exchange frontmatter from small pools so
// E and O overlap: aliases, a parent vault, at most one primary link, and
// covered links.
func drawFoldExchange(rt *rapid.T, label string) foldExchange {
	namePool := []string{
		"7.2026-09-01.p", "8.2026-09-01.q", "9.2026-09-01.r", foldOfferBase, foldExistingBase, "6.2026-01-01.s",
	}
	names := rapid.SliceOfNDistinct(rapid.SampledFrom(namePool), 0, 3, func(name string) string { return name }).
		Draw(rt, label+"Aliases")

	drawn := foldExchange{
		xid:     rapid.SampledFrom([]string{"", foldXIDE}).Draw(rt, label+"XID"),
		aliases: slices.DeleteFunc(names, func(name string) bool { return name == foldExistingBase && label == "existing" }),
	}

	linkNames := rapid.SliceOfNDistinct(rapid.SampledFrom(namePool[:3]), 0, 3, func(name string) string { return name }).
		Draw(rt, label+"Links")
	if len(linkNames) == 0 {
		return drawn
	}

	drawn.vault = rapid.SampledFrom([]string{foldParentVault, foldOtherVault}).Draw(rt, label+"Vault")
	primary := rapid.SampledFrom([]string{"", "offered", "pulled"}).Draw(rt, label+"Primary")

	for index, name := range linkNames {
		via := "covered"
		if index == 0 && primary != "" {
			via = primary
		}

		drawn.links = append(drawn.links, foldLink{note: name, via: via, hash: "xh1:" + label + name})
	}

	return drawn
}

// expectOnlyExchangeKeysChanged: after removing the parent and aliases keys,
// the frontmatter decodes to the same map and the body is byte-identical —
// so identity, xid, pending and content are untouched.
func expectOnlyExchangeKeysChanged(g Gomega, before string, after []byte) {
	beforeRaw := []byte(before)

	strip := func(raw []byte) (map[string]any, string) {
		frontmatter, body, found := embed.SplitFrontmatter(raw)
		g.Expect(found).To(BeTrue())

		decoded := map[string]any{}
		g.Expect(yaml.Unmarshal(frontmatter, &decoded)).To(Succeed())
		delete(decoded, "parent")
		delete(decoded, "aliases")

		return decoded, string(body)
	}

	beforeFields, beforeBody := strip(beforeRaw)
	afterFields, afterBody := strip(after)
	g.Expect(afterFields).To(Equal(beforeFields))
	g.Expect(afterBody).To(Equal(beforeBody))
}

func expectedHashFor(t *testing.T, note, kind string) string {
	t.Helper()

	hash, err := cli.ExportExchangeHash([]byte(note))
	if err != nil {
		t.Fatal(err)
	}

	switch kind {
	case "correct":
		return hash
	case "changed":
		return "xh1:" + strings.Repeat("0", 64)
	case "other-version":
		return "xh0:" + strings.TrimPrefix(hash, "xh1:")
	default:
		return ""
	}
}

// foldNote renders a live (or pending) local fact carrying exchange.
func foldNote(exchange foldExchange, pending bool) string {
	var builder strings.Builder

	builder.WriteString("---\ntype: fact\ntier: L2\nsituation: when local\nsubject: l\npredicate: m\nobject: n\n" +
		"luhmann: \"2\"\ncreated: \"2026-09-20\"\nsource: test\nrepo: github.com/alice/widgets\n" +
		"user: alice@example.com\nvault: personal\n")

	if pending {
		builder.WriteString("pending: true\n")
	}

	if exchange.xid != "" {
		builder.WriteString("xid: " + exchange.xid + "\n")
	}

	if len(exchange.links) > 0 {
		builder.WriteString("parent:\n    vault: " + exchange.vault + "\n    links:\n")

		for _, link := range exchange.links {
			builder.WriteString("        - note: " + link.note + "\n          via: " + link.via +
				"\n          hash: " + link.hash + "\n")
		}
	}

	if len(exchange.aliases) > 0 {
		builder.WriteString("aliases:\n")

		for _, alias := range exchange.aliases {
			builder.WriteString("    - " + alias + "\n")
		}
	}

	if exchange.origin {
		builder.WriteString("offer:\n    origin: " + foldOtherVault + ":" + foldXIDO + "\n    key: k1\n")
	}

	builder.WriteString("---\n\nInformation learned: when local, l m n.\n")

	return builder.String()
}

func isPrimaryVia(via string) bool {
	return via == "offered" || via == "pulled"
}

func linksEqual(existing foldExchange, got foldParentYAML) bool {
	if got.Vault != existing.vault || len(got.Links) != len(existing.links) {
		return false
	}

	for index, link := range existing.links {
		if got.Links[index] != (foldLinkYAML{Note: link.note, Via: link.via, Hash: link.hash}) {
			return false
		}
	}

	return true
}

func newFoldVault() *foldVault {
	return &foldVault{files: map[string][]byte{}}
}

func writeFoldFile(t *testing.T, path, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatal(err)
	}
}

package cli_test

// Scenario tests for the vault-serving product capability, part 3 (861c2zzh): a caller cannot set link, identity or
// registration fields (792a20j), a served show returns a note and a raw envelope on request (792a20k), a served
// activate takes note names only (792a20l), a served query takes the message and can return dedupe keys (792a20m)
// and an oversized parent response is refused (792a20n1c). Spec: docs/specs/product/vault-serving.md. The request
// body caps (792a20n1a, 792a20n1b) are set by the real net/http edge and live in
// cmd/engram/vault_serving_part3_scenarios_test.go.
//
// The scenarios run over the registered route handlers with the deps the neighbouring serve tests use: a vault in
// the test's own t.TempDir() (the recorded deviation, #860: the vault lock is a real flock and no in-memory EdgeFS
// models the exclusive create and the directory listing a learn needs), a deterministic stub embedder, a fixed clock
// and a scripted git. A pending offer or a renamed note is produced by the commands that make them (a served learn,
// the rename pass, engram amend); the raw-show notes and the activate decoy are written directly because their bytes
// are the point of the scenario. No network and no real home.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a20j1a_ServeLearn_RemoteSetLinkAndRegistrationFieldsAreIgnored verifies 792a20j1a (Remote-set link and
// registration fields are ignored): a served learn body that carries parent, aliases, xid, skillHash and skillKey
// keys, in camelCase, PascalCase and the other spellings of the keys, writes a pending note with no parent, aliases
// or skill field and none of the values sent. The note's xid is the one the server minted: every pending offer
// carries one, so the scenario's "none of those fields" reads as none of the caller's (861a4zzh1).
func Test792a20j1a_ServeLearn_RemoteSetLinkAndRegistrationFieldsAreIgnored(t *testing.T) {
	t.Parallel()

	offer := cli.LearnOffer{Origin: childOrigin, Key: vs2Key("k1"), Path: []string{childVaultID}}

	// The skill fields belong to a runbook note, the link fields to every note, so both kinds are offered.
	for name, args := range map[string]cli.LearnArgs{
		"a fact":    vs2Fact("forged", "object", "k1"),
		"a runbook": vs3RunbookOffer("forged", offer),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			deps, vault := vs3Server(t)

			resp := vs3PostLearn(t, deps, vault, vs3WithForged(t, args, vs3ForgedSpellings))
			g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

			notePath := filepath.Join(vault, decodeReceipt(t, resp).Basename+".md")
			fields := vs3Frontmatter(t, notePath)

			for _, key := range vs3LinkAndRegistrationKeys {
				g.Expect(fields).NotTo(HaveKey(key), "the pending note carries no %s", key)
			}

			g.Expect(fields).To(HaveKeyWithValue("xid", seqID(vs3EntropyStart)), "its xid is the one the server minted")
			g.Expect(readFileString(t, notePath)).NotTo(ContainSubstring("forged"), "none of the values sent lands")
		})
	}
}

// Test792a20j1b_ServeLearn_ACallerCannotForgeALink verifies 792a20j1b (A caller cannot forge a link): a served
// learn body that carries parent, aliases and xid values, as a new offer and as a rewrite of the caller's pending
// offer, writes a pending note with none of those values; its xid is the one the server minted.
func Test792a20j1b_ServeLearn_ACallerCannotForgeALink(t *testing.T) {
	t.Parallel()

	for name, rewrite := range map[string]bool{"a new offer": false, "a rewrite of the pending offer": true} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			deps, vault := vs3Server(t)

			if rewrite {
				first := vs3PostLearn(t, deps, vault, vs3WithForged(t, vs2Fact("forged", "first object", "k1"), ""))
				g.Expect(first.Status).To(Equal(vs1StatusOK), string(first.Body))
			}

			args := vs2Fact("forged", "second object", "k2")
			resp := vs3PostLearn(t, deps, vault, vs3WithForged(t, args, vs3ForgedValues))
			g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

			receipt := decodeReceipt(t, resp)
			g.Expect(vs2PendingNotes(t, vault)).To(HaveLen(1), "one pending note, rewritten in place when it was one")

			notePath := filepath.Join(vault, receipt.Basename+".md")
			note := readFileString(t, notePath)

			for _, forged := range []string{vs3ForgedVault, vs3ForgedLink, vs3ForgedAlias, vs3ForgedXID} {
				g.Expect(note).NotTo(ContainSubstring(forged), "the note carries none of the forged values")
			}

			fields := vs3Frontmatter(t, notePath)
			g.Expect(fields).NotTo(HaveKey("parent"))
			g.Expect(fields).NotTo(HaveKey("aliases"))
			g.Expect(fields).To(HaveKeyWithValue("xid", seqID(vs3EntropyStart)), "its xid is the server's own")
		})
	}
}

// Test792a20k1a_ServeShow_WithoutRawMatchesTheLocalOutput verifies 792a20k1a (Show without raw matches the local
// output): for a note that carries an xid, GET /show without raw answers exactly what engram show prints, the
// exchange-hash header line and the outbound-links section included.
func Test792a20k1a_ServeShow_WithoutRawMatchesTheLocalOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	const basename = "5.2026-09-20.exchanged"

	notePath := writeVaultNote(t, vault, basename+".md", vs3ExchangedNote())

	var local bytes.Buffer

	showErr := cli.RunShow(t.Context(), cli.ShowArgs{Ref: basename, VaultPath: vault},
		cli.ExportNewShowDeps(deps.FS), &local)
	g.Expect(showErr).NotTo(HaveOccurred())

	if showErr != nil {
		return
	}

	g.Expect(local.String()).To(HavePrefix("# exchange_hash: "+fileExchangeHash(t, notePath)+"\n"),
		"the local output opens with the header line, so equality is not trivial")
	g.Expect(local.String()).To(ContainSubstring("2026-09-21.next-hop"), "and ends with the outbound links")

	resp := vs3Get(t, deps, vault, "/show", map[string][]string{"note": {basename}})
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(string(resp.Body)).To(Equal(local.String()), "the served response is the local output")
}

// Test792a20k1b_ServeShow_RawShowIsExact verifies 792a20k1b (Raw show is exact): GET /show?note=<basename>&raw=1 for
// a live note answers a JSON envelope whose content is byte-identical to the note file, with no red_flags preview cap
// (the note has more red flags than the preview shows) and no appended links section.
func Test792a20k1b_ServeShow_RawShowIsExact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	const basename = "6.2026-09-20.exact"

	notePath := writeVaultNote(t, vault, basename+".md", vs3RawNote)

	resp := vs3Get(t, deps, vault, "/show", map[string][]string{"note": {basename}, "raw": {"1"}})
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

	envelope := vs3Envelope(t, resp)
	g.Expect(envelope["content"]).To(Equal(string(readFileBytes(t, notePath))), "content is the file's bytes")
	g.Expect(envelope["content"]).To(Equal(vs3RawNote))
	g.Expect(envelope["content"]).NotTo(ContainSubstring("# outbound links"))
	g.Expect(envelope["basename"]).To(Equal(basename))
	g.Expect(envelope["vault_id"]).To(Equal(serverVaultID))
	g.Expect(envelope["exchange_hash"]).To(Equal(fileExchangeHash(t, notePath)))
}

// Test792a20k1c_ServeShow_RawShowFollowsARename verifies 792a20k1c (Raw show follows a rename): a note renamed by
// the rename pass, which lists its old basename in aliases, is served raw under the old basename as an envelope
// that carries the note's current basename and current content.
func Test792a20k1c_ServeShow_RawShowFollowsARename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	vs2LearnLive(t, deps, vault, "renamed")

	oldName := vs2NoteBasename(t, vault, "renamed")
	newName := vs2Rename(t, vault, oldName)
	g.Expect(newName).NotTo(Equal(oldName))

	current := readFileString(t, filepath.Join(vault, newName+".md"))
	g.Expect(current).To(ContainSubstring(oldName), "the renamed note lists its old basename in aliases")

	resp := vs3Get(t, deps, vault, "/show", map[string][]string{"note": {oldName}, "raw": {"1"}})
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

	envelope := vs3Envelope(t, resp)
	g.Expect(envelope["basename"]).To(Equal(newName), "the envelope carries the note's current basename")
	g.Expect(envelope["content"]).To(Equal(current), "and its current content")
}

// Test792a20k1d_ServeShow_AMissingNoteIsNotFound verifies 792a20k1d (A missing note is not found): GET
// /show?note=<missing>&raw=1 for a name that no note has, as a basename or an alias, answers 404 with an error body.
func Test792a20k1d_ServeShow_AMissingNoteIsNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	vs2LearnLive(t, deps, vault, "bystander")

	resp := vs3Get(t, deps, vault, "/show", map[string][]string{"note": {"404.2026-01-01.missing"}, "raw": {"1"}})
	g.Expect(resp.Status).To(Equal(vs3StatusNotFound), string(resp.Body))

	var body map[string]string

	g.Expect(json.Unmarshal(resp.Body, &body)).To(Succeed())
	g.Expect(body).To(HaveKey("error"))
	g.Expect(body["error"]).To(ContainSubstring("not found"))
}

// Test792a20k1e_ServeShow_APendingNoteIsNotServedRaw verifies 792a20k1e (A pending note is not served raw): GET
// /show?note=<its basename or an alias>&raw=1 for a pending offer answers 404 with an error body; once curation
// accepts the same note (the pending marker cleared) the same request is served, so the marker alone decides.
func Test792a20k1e_ServeShow_APendingNoteIsNotServedRaw(t *testing.T) {
	t.Parallel()

	for name, byAlias := range map[string]bool{"its basename": false, "an alias": true} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			deps, vault := vs3Server(t)

			resp, receipt := vs2Offer(t, deps, vault, vs2Fact("pending", "object", "k1"))
			g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

			ref := receipt.Basename

			if byAlias {
				renamed := vs2Rename(t, vault, receipt.Basename)
				g.Expect(renamed).NotTo(Equal(receipt.Basename))
				g.Expect(readFileString(t, filepath.Join(vault, renamed+".md"))).To(ContainSubstring(receipt.Basename),
					"the renamed pending note lists its old basename in aliases")

				receipt.Basename = renamed
				receipt.Luhmann, _, _ = strings.Cut(renamed, ".")
			}

			raw := func() cli.ServeResponse {
				return vs3Get(t, deps, vault, "/show", map[string][]string{"note": {ref}, "raw": {"1"}})
			}

			refused := raw()
			g.Expect(refused.Status).To(Equal(vs3StatusNotFound), string(refused.Body))
			g.Expect(string(refused.Body)).To(ContainSubstring("error"))
			g.Expect(string(refused.Body)).NotTo(ContainSubstring("an offered fact"), "no pending content is served")

			vs2Accept(t, deps, vault, receipt)

			accepted := raw()
			g.Expect(accepted.Status).To(Equal(vs1StatusOK), string(accepted.Body))
			g.Expect(vs3Envelope(t, accepted)["basename"]).To(Equal(receipt.Basename))
		})
	}
}

// Test792a20k1f_ServeShow_AMissingRefIsABadRequest verifies 792a20k1f (A missing ref is a bad request): GET /show
// with no note parameter, with raw and without, answers 400.
func Test792a20k1f_ServeShow_AMissingRefIsABadRequest(t *testing.T) {
	t.Parallel()

	for name, query := range map[string]map[string][]string{
		"without raw": {},
		"with raw":    {"raw": {"1"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			deps := vs1ServerDeps()
			vault := newServeVault(t)

			vs2LearnLive(t, deps, vault, "bystander")

			resp := vs3Get(t, deps, vault, "/show", query)
			g.Expect(resp.Status).To(Equal(vs1StatusBadRequest), string(resp.Body))
		})
	}
}

// Test792a20l1a_ServeActivate_APathRefIsRejected verifies 792a20l1a (A path ref is rejected): POST /activate naming
// /etc/x, ../x.md or an absolute path to a note outside the vault answers 400, reads and writes nothing, and changes
// no sidecar inside or outside the vault (a note of each place has a sidecar the path would reach).
func Test792a20l1a_ServeActivate_APathRefIsRejected(t *testing.T) {
	t.Parallel()

	for name, ref := range map[string]string{
		"/etc/x":                      "/etc/x",
		"../x.md":                     "../x.md",
		"an absolute path outside":    "",
		"a path inside the vault dir": "sub/inside",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			base := t.TempDir()
			vault := filepath.Join(base, "vault")
			outside := filepath.Join(base, "outside")
			g.Expect(os.MkdirAll(vault, vs3DirPerm)).To(Succeed())
			g.Expect(os.MkdirAll(outside, vs3DirPerm)).To(Succeed())

			deps := vs1ServerDeps()
			vs3LearnAt(t, deps, vault, "inside")
			outsideNote := vs3LearnAt(t, deps, outside, "outside")
			vs3PlantSidecarOnly(t, filepath.Join(base, "x.md"))

			if ref == "" {
				ref = outsideNote
			}

			before := snapshotVault(t, base)
			log := &eventLog{}
			deps.FS = recordingFS{EdgeFS: deps.FS, log: log}

			resp := vs3PostActivate(t, deps, vault, ref)
			g.Expect(resp.Status).To(Equal(vs1StatusBadRequest), string(resp.Body))
			g.Expect(string(resp.Body)).To(ContainSubstring("error"))
			g.Expect(snapshotVault(t, base)).To(Equal(before), "no sidecar inside or outside the vault changed")
			g.Expect(log.snapshot()).To(BeEmpty(), "nothing was read or written")
		})
	}
}

// Test792a20l1b_ServeActivate_AListedNoteNameIsActivated verifies 792a20l1b (A listed note name is activated): POST
// /activate naming a vault note's basename, with or without .md, bumps that note's sidecar last-used date to the
// day of the request and answers 200 with {"status":"ok"}.
func Test792a20l1b_ServeActivate_AListedNoteNameIsActivated(t *testing.T) {
	t.Parallel()

	for name, suffix := range map[string]string{"without .md": "", "with .md": ".md"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newServeVault(t)
			held, bystander := vs3LearnEarlier(t, vault, "held"), vs3LearnEarlier(t, vault, "bystander")
			g.Expect(vs3LastUsed(t, vault, held)).NotTo(Equal(vs1Date), "the sidecar starts on an earlier day")

			resp := vs3PostActivate(t, vs1ServerDeps(), vault, held+suffix)
			g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
			g.Expect(string(resp.Body)).To(MatchJSON(`{"status":"ok"}`))
			g.Expect(vs3LastUsed(t, vault, held)).To(Equal(vs1Date), "that note's last-used date is bumped")
			g.Expect(vs3LastUsed(t, vault, bystander)).NotTo(Equal(vs1Date), "and no other note's")
		})
	}
}

// Test792a20l1c_ServeActivate_NoRefFoundIsA404 verifies 792a20l1c (No ref found is a 404): POST /activate naming
// only notes the vault does not hold answers 404 and lists each ref as not found; nothing in the vault changes.
func Test792a20l1c_ServeActivate_NoRefFoundIsA404(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	vs3LearnEarlier(t, vault, "bystander")

	before := snapshotVault(t, vault)
	missing := []string{"404.2026-01-01.first-missing", "405.2026-01-02.second-missing.md"}

	resp := vs3PostActivate(t, vs1ServerDeps(), vault, missing...)
	g.Expect(resp.Status).To(Equal(vs3StatusNotFound), string(resp.Body))

	result := vs3ActivateResult(t, resp)
	g.Expect(result.Activated).To(BeEmpty())
	g.Expect(result.Failed).To(HaveLen(len(missing)))

	for index, ref := range missing {
		if index >= len(result.Failed) {
			break
		}

		g.Expect(result.Failed[index].Ref).To(Equal(ref))
		g.Expect(result.Failed[index].Error).To(ContainSubstring("not found"), "each ref is listed as not found")
	}

	g.Expect(snapshotVault(t, vault)).To(Equal(before), "nothing changed")
}

// Test792a20l1d_ServeActivate_APartialResultIsA200 verifies 792a20l1d (A partial result is a 200): POST /activate
// naming one note the vault holds and one it does not answers 200 with the per-ref result: the held note is
// activated (its sidecar bumped) and the other is listed as not found.
func Test792a20l1d_ServeActivate_APartialResultIsA200(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	held := vs3LearnEarlier(t, vault, "held")

	const missing = "404.2026-01-01.missing"

	resp := vs3PostActivate(t, vs1ServerDeps(), vault, held, missing)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

	result := vs3ActivateResult(t, resp)
	g.Expect(result.Activated).To(Equal([]string{held}))
	g.Expect(result.Failed).To(HaveLen(1))

	if len(result.Failed) == 1 {
		g.Expect(result.Failed[0].Ref).To(Equal(missing))
		g.Expect(result.Failed[0].Error).To(ContainSubstring("not found"))
	}

	g.Expect(vs3LastUsed(t, vault, held)).To(Equal(vs1Date), "the held note was activated")
}

// Test792a20l1e_ServeActivate_AFailedActivationIsA500 verifies 792a20l1e (A failed activation is a 500): POST
// /activate naming a note whose sidecar cannot be written answers 500 with the per-ref result that names the
// failure, and the sidecar keeps its earlier last-used date.
func Test792a20l1e_ServeActivate_AFailedActivationIsA500(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	held := vs3LearnEarlier(t, vault, "held")
	earlier := vs3LastUsed(t, vault, held)

	deps := vs1ServerDeps()
	deps.FS = vs3SidecarWriteFailFS{EdgeFS: deps.FS}

	resp := vs3PostActivate(t, deps, vault, held)
	g.Expect(resp.Status).To(Equal(vs3StatusServerError), string(resp.Body))

	result := vs3ActivateResult(t, resp)
	g.Expect(result.Activated).To(BeEmpty())
	g.Expect(result.Failed).To(HaveLen(1))

	if len(result.Failed) == 1 {
		g.Expect(result.Failed[0].Ref).To(Equal(held))
		g.Expect(result.Failed[0].Error).To(MatchRegexp(`writing sidecar .*: `+errVs3SidecarDenied.Error()),
			"the failure is the sidecar write, not a missing note")
	}

	g.Expect(vs3LastUsed(t, vault, held)).To(Equal(earlier), "the sidecar was not rewritten")
}

// Test792a20l1f_Activate_LocalActivateKeepsPaths verifies 792a20l1f (Local activate keeps paths): the local
// engram activate with an absolute path to a note activates it, whether the note sits in the vault or in another
// directory the path reaches.
func Test792a20l1f_Activate_LocalActivateKeepsPaths(t *testing.T) {
	t.Parallel()

	for name, inVault := range map[string]bool{"a note in the vault": true, "a note outside the vault": false} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newServeVault(t)
			noteDir := vault

			if !inVault {
				noteDir = t.TempDir()
			}

			earlier := vs3LearnerDeps()
			notePath := vs3LearnAt(t, earlier, noteDir, "pathed")
			g.Expect(vs3LastUsedAt(t, notePath)).NotTo(Equal(vs1Date), "the sidecar starts on an earlier day")
			g.Expect(filepath.IsAbs(notePath)).To(BeTrue())

			stderr := executeForTestWithDeps(t,
				[]string{"engram", "activate", "--vault", vault, "--note", notePath}, vs1Customize)
			g.Expect(stderr).To(BeEmpty())
			g.Expect(vs3LastUsedAt(t, notePath)).To(Equal(vs1Date), "the note is activated")
		})
	}
}

// Test792a20m1a_ServeQuery_ShortTextRoundTripsUnchanged verifies 792a20m1a (Short text round-trips unchanged): a
// text of 2048 bytes or fewer with quotes, newlines, & and non-ASCII characters is queried as the identical string
// (each runbook trigger below matches only when its characters arrive exactly) and produces the same payload as
// the local path; a text of exactly 2048 bytes is not cut (its last letter keeps the cue from being a whole word).
func Test792a20m1a_ServeQuery_ShortTextRoundTripsUnchanged(t *testing.T) {
	t.Parallel()

	special := "Please run: say \"hello there\" now.\nfish & chips with crème brûlée?\nfirst line\nsecond line"
	atCap := strings.Repeat(" ", vs3TextCap-len(vs3Cue)-len("é")) + vs3Cue + "é"

	cases := map[string]struct {
		text     string
		triggers map[string]string
		hits     []string
		misses   []string
	}{
		"quotes, newlines, & and non-ASCII": {
			text: special,
			triggers: map[string]string{
				"quoted": `say "hello there"`, "ampersand": "fish & chips", "accented": "crème brûlée",
				"multiline": "first line second line", "unrelated": "no such words",
			},
			hits:   []string{"quoted", "ampersand", "accented", "multiline"},
			misses: []string{"unrelated"},
		},
		"exactly 2048 bytes": {
			text:     atCap,
			triggers: map[string]string{"whole": vs3Cue + "é", "cut": vs3Cue},
			hits:     []string{"whole"},
			misses:   []string{"cut"},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(len(testCase.text)).To(BeNumerically("<=", vs3TextCap))

			deps := vs1ServerDeps()
			vault := newServeVault(t)

			for slug, trigger := range testCase.triggers {
				vs3Runbook(t, deps, vault, slug, trigger)
			}

			resp := vs3Get(t, deps, vault, "/query", map[string][]string{"text": {testCase.text}})
			g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

			for _, slug := range testCase.hits {
				g.Expect(string(resp.Body)).To(ContainSubstring("."+slug+".md"), "the %s trigger matches", slug)
			}

			for _, slug := range testCase.misses {
				g.Expect(string(resp.Body)).NotTo(ContainSubstring("."+slug+".md"), "the %s trigger does not", slug)
			}

			var local bytes.Buffer

			localErr := cli.RunQuery(t.Context(), cli.QueryArgs{Text: testCase.text, VaultPath: vault},
				cli.ExportNewQueryDeps(deps), &local)
			g.Expect(localErr).NotTo(HaveOccurred())
			g.Expect(string(resp.Body)).To(Equal(local.String()), "the payload is the local path's")
		})
	}
}

// Test792a20m1b_ServeQuery_LongTextIsTruncatedOnARuneBoundary verifies 792a20m1b (Long text is truncated on a rune
// boundary): a text longer than 2048 bytes whose byte 2048 falls inside a multi-byte character (2, 3 and 4 bytes
// wide, each cut at every inside position) is queried cut back to the last complete character at or before byte
// 2048, and the request answers 200 with no error. A cue that ends where the character begins matches only when
// the cut is made there; a cue that carries the replacement character matches only when a broken byte was left in
// the text (the text is lower-cased with an invalid byte read as U+FFFD); a cue after the character is cut off.
func Test792a20m1b_ServeQuery_LongTextIsTruncatedOnARuneBoundary(t *testing.T) {
	t.Parallel()

	for _, character := range []string{"\u00e9", "\u65e5", "\U00020000"} {
		for inside := 1; inside < len(character); inside++ {
			t.Run(fmt.Sprintf("%s cut after byte %d", character, inside), func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)

				text := strings.Repeat(" ", vs3TextCap-inside-len(vs3Cue)) + vs3Cue + character + " beyondcue"
				g.Expect(len(text)).To(BeNumerically(">", vs3TextCap))
				g.Expect(utf8.RuneStart(text[vs3TextCap])).To(BeFalse(), "byte 2048 falls inside the character")

				deps := vs1ServerDeps()
				vault := newServeVault(t)

				vs3Runbook(t, deps, vault, "kept", vs3Cue)
				vs3Runbook(t, deps, vault, "broken", vs3Cue+"\ufffd")
				vs3Runbook(t, deps, vault, "beyond", "beyondcue")

				resp := vs3Get(t, deps, vault, "/query", map[string][]string{"text": {text}})
				g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
				g.Expect(string(resp.Body)).NotTo(ContainSubstring("error"), "no error is returned")
				g.Expect(string(resp.Body)).To(ContainSubstring(".kept.md"), "the cut falls where the character begins")
				g.Expect(string(resp.Body)).NotTo(ContainSubstring(".broken.md"), "no broken byte is left in the text")
				g.Expect(string(resp.Body)).NotTo(ContainSubstring(".beyond.md"), "what follows byte 2048 is cut off")
			})
		}
	}
}

// Test792a20m1c_ServeQuery_KeysArePresentOnlyOnRequest verifies 792a20m1c (Keys are present only on request): GET
// /query served once with dedupe-keys=1 and once without: only the first carries a top-level vault_id and each
// note item's exchange_hash, and the second is byte-identical to the local payload.
func Test792a20m1c_ServeQuery_KeysArePresentOnlyOnRequest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	vs1PlantNotes(t, deps, vault)

	plain := vs3Get(t, deps, vault, "/query", map[string][]string{"phrase": {vs1Phrase}})
	keyed := vs3Get(t, deps, vault, "/query", map[string][]string{"phrase": {vs1Phrase}, "dedupe-keys": {"1"}})
	g.Expect(plain.Status).To(Equal(vs1StatusOK), string(plain.Body))
	g.Expect(keyed.Status).To(Equal(vs1StatusOK), string(keyed.Body))

	var local bytes.Buffer

	localErr := cli.RunQuery(t.Context(), cli.QueryArgs{Phrases: []string{vs1Phrase}, VaultPath: vault},
		cli.ExportNewQueryDeps(deps), &local)
	g.Expect(localErr).NotTo(HaveOccurred())
	g.Expect(string(plain.Body)).To(Equal(local.String()), "without the parameter the payload is the local one")
	g.Expect(string(plain.Body)).NotTo(ContainSubstring("vault_id"))
	g.Expect(string(plain.Body)).NotTo(ContainSubstring("exchange_hash"))

	payload := vs3KeyedPayload(t, keyed)
	g.Expect(payload.VaultID).To(Equal(serverVaultID))
	g.Expect(payload.Items).To(HaveLen(len(noteFiles(t, vault))), "every note is in the payload")

	for _, item := range payload.Items {
		g.Expect(item.ExchangeHash).To(Equal(fileExchangeHash(t, filepath.Join(vault, item.Path))), item.Path)
	}
}

// Test792a20m1d_ServeQuery_ANoteWhoseKeysCannotBeComputedLosesOnlyItsOwn verifies 792a20m1d (A note whose keys
// cannot be computed loses only its own): a dedupe-keys query whose payload has a note that cannot be read (the
// read the keys need fails after the query itself read the note) keeps the other notes' keys, succeeds, and warns
// once, naming the skipped note.
func Test792a20m1d_ServeQuery_ANoteWhoseKeysCannotBeComputedLosesOnlyItsOwn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	vs1PlantNotes(t, deps, vault)

	unreadable := vs1NoteWithSlug(t, vault, "served-beta")
	query := map[string][]string{"phrase": {vs1Phrase}, "dedupe-keys": {"1"}}

	// The query itself reads the unreadable note; count those reads without a failure, so the keys' own read is
	// the first one that fails.
	realFS := deps.FS
	counting := &vs3ReadFS{EdgeFS: realFS, path: unreadable}
	deps.FS = counting

	g.Expect(vs3Get(t, deps, vault, "/query", map[string][]string{"phrase": {vs1Phrase}}).Status).
		To(Equal(vs1StatusOK))

	queryReads := counting.count()
	g.Expect(queryReads).To(BeNumerically(">", 0), "the query reads the note")

	var stderr bytes.Buffer

	deps.Stderr = &stderr
	deps.FS = &vs3ReadFS{EdgeFS: realFS, path: unreadable, failFrom: queryReads + 1}

	resp := vs3Get(t, deps, vault, "/query", query)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

	payload := vs3KeyedPayload(t, resp)
	g.Expect(payload.Items).To(HaveLen(len(noteFiles(t, vault))), "the query still lists every note")

	keyed, skipped := 0, 0

	for _, item := range payload.Items {
		if item.Path == filepath.Base(unreadable) {
			skipped++

			g.Expect(item.ExchangeHash).To(BeEmpty(), "the unreadable note loses its key")

			continue
		}

		keyed++

		g.Expect(item.ExchangeHash).To(HavePrefix("xh1:"), "%s keeps its key", item.Path)
	}

	g.Expect(skipped).To(Equal(1))
	g.Expect(keyed).To(Equal(len(noteFiles(t, vault)) - 1))

	warnings := vs3Lines(stderr.String(), "skipping")
	g.Expect(warnings).To(HaveLen(1), "one warning")
	g.Expect(warnings).To(ContainElement(ContainSubstring(filepath.Base(unreadable))))
}

// Test792a20n1c_Activate_AnOversizedParentResponseIsRefused verifies 792a20n1c (An oversized parent response is
// refused): a parent that answers a pull-down with a response over 16 MiB, as the HTTP edge hands it to the child
// (the first FetchResponseReadLimit bytes, one over the largest response accepted), fails the child's request as
// the parent's failure (it is counted for backoff and named as exceeding the size limit) and nothing from it is
// stored; the same parent's smaller note is pulled in the same run, so the size alone decides.
func Test792a20n1c_Activate_AnOversizedParentResponseIsRefused(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)

	huge := pulledFact("7.2026-09-01.huge", "c", "")
	huge.content += strings.Repeat("x", cli.FetchResponseReadLimit)
	env.parent.addNote(huge)
	env.parent.addNote(pulledFact("8.2026-09-02.small", "c", ""))

	env.wrap = func(deps *cli.Deps) {
		parent := deps.Fetch
		edge := func(ctx context.Context, method, url string, body []byte) (cli.FetchResponse, error) {
			resp, err := parent(ctx, method, url, body)
			if err == nil && len(resp.Body) > cli.FetchResponseReadLimit {
				resp.Body = resp.Body[:cli.FetchResponseReadLimit]
			}

			return resp, err
		}

		deps.Fetch = cli.NewDeps(cli.Primitives{HTTP: cli.HTTPPrims{Fetch: edge}}, nil, io.Discard, io.Discard,
			func(int) {}).Fetch
	}

	_, stderr := env.run("activate", "--note", "7.2026-09-01.huge.md")
	g.Expect(stderr).To(ContainSubstring("exceeds the size limit"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
	g.Expect(env.noteFiles()).To(BeEmpty(), "nothing from the response is stored")
	g.Expect(env.parentCache().Failures).To(Equal(1), "the parent's failure is counted for backoff")
	g.Expect(env.parentCache().VaultID).To(BeEmpty(), "nothing the response said is cached")
	g.Expect(env.parent.activated()).To(BeEmpty(), "and the parent is not told the note was used")

	// The failure paused the exchange for the backoff window, so the control runs after the window.
	env.advance(vs3BackoffWindow)

	_, controlErr := env.run("activate", "--note", "8.2026-09-02.small.md")
	g.Expect(controlErr).NotTo(ContainSubstring("exceeds"))
	g.Expect(env.exitCodes()).To(BeEmpty(), "the control did not fail")
	g.Expect(env.noteFiles()).To(HaveLen(1), "a response within the limit is stored")
}

// unexported constants.
const (
	vs3BackoffWindow     = 24 * time.Hour
	vs3Cue               = "zzcue"
	vs3DirPerm           = 0o700
	vs3EntropyStart      = byte(1)
	vs3FilePerm          = 0o600
	vs3ForgedAlias       = "8.2026-01-08.forged-alias"
	vs3ForgedLink        = "9.2026-01-09.someone-elses-note"
	vs3ForgedVault       = "b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0"
	vs3ForgedXID         = "dddddddddddddddddddddddddddddddd"
	vs3MonthsEarlier     = 8
	vs3StatusNotFound    = 404
	vs3StatusServerError = 500
	vs3TextCap           = 2048
)

// unexported variables.
var (
	errVs3ReadDenied    = errors.New("vs3: read denied")
	errVs3SidecarDenied = errors.New("vs3: sidecar write denied")
	// vs3ForgedSpellings are the keys of the scenario in every spelling a caller may try.
	vs3ForgedSpellings = `"parent":{"vault":"forgedparent"},"Parent":{"Vault":"forgedparent"},` +
		`"aliases":["forgedalias"],"Aliases":["forgedalias"],` +
		`"xid":"forgedxid","Xid":"forgedxid","XID":"forgedxid",` +
		`"skillHash":"forgedhash","SkillHash":"forgedhash","skillKey":"forgedkey","SkillKey":"forgedkey",` +
		`"skillSource":"forgedsource","SkillSource":"forgedsource",` +
		`"skill_hash":"forgedhash","skill_key":"forgedkey","skill_source":"forgedsource"`
	// vs3ForgedValues are well-formed link, alias and exchange ID values of someone else's notes.
	vs3ForgedValues = `"parent":{"vault":"` + vs3ForgedVault + `","links":[{"note":"` + vs3ForgedLink +
		`","via":"offered"}]},"aliases":["` + vs3ForgedAlias + `"],"xid":"` + vs3ForgedXID + `"`
	vs3LinkAndRegistrationKeys = []string{"parent", "aliases", "skill_hash", "skill_key", "skill_source"}
	// vs3RawNote has more red flags than the show preview shows, a wikilink, a tab, trailing spaces, non-ASCII
	// text and no final newline, so only a verbatim copy of the file equals it.
	vs3RawNote = "---\ntype: runbook\nsituation: s\ndone_when: d\nred_flags:\n" +
		"  - flag one\n  - flag two\n  - flag three\n  - flag four\n  - flag five\n  - flag six\n" +
		"  - flag seven\n  - flag eight\n  - flag nine\n  - flag ten\n" +
		"luhmann: \"6\"\ncreated: 2026-01-06\nsource: agent\nuser: u\nvault: personal\n---\n\n" +
		"1. see [[7.2026-01-07.other]]\t\n2. caf\u00e9   \n3. last line, no newline"
)

// vs3ActivateFailure is one ref a served activate could not activate, with the reason.
type vs3ActivateFailure struct {
	Ref   string `json:"ref"`
	Error string `json:"error"`
}

// vs3ActivateOutcome is the per-ref result a served activate answers when some ref did not activate.
type vs3ActivateOutcome struct {
	Activated []string             `json:"activated"`
	Failed    []vs3ActivateFailure `json:"failed"`
}

// vs3KeyedBody is a dedupe-keys query payload.
type vs3KeyedBody struct {
	VaultID string         `yaml:"vault_id"`
	Items   []vs3KeyedItem `yaml:"items"`
}

// vs3KeyedItem is one item of a dedupe-keys query payload.
type vs3KeyedItem struct {
	Path         string `yaml:"path"`
	ExchangeHash string `yaml:"exchange_hash"`
}

// vs3ReadFS wraps an EdgeFS, counting the reads of one path and failing every read of it from the failFrom-th on
// (0 never fails).
type vs3ReadFS struct {
	cli.EdgeFS

	path     string
	failFrom int32
	reads    atomic.Int32
}

// ReadFile counts the read of the watched path and fails it from the failFrom-th read on.
func (f *vs3ReadFS) ReadFile(path string) ([]byte, error) {
	if path == f.path {
		reads := f.reads.Add(1)
		if f.failFrom > 0 && reads >= f.failFrom {
			return nil, errVs3ReadDenied
		}
	}

	data, err := f.EdgeFS.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// count is how many times the watched path was read.
func (f *vs3ReadFS) count() int32 {
	return f.reads.Load()
}

// vs3SidecarWriteFailFS fails every atomic write of a sidecar and passes every other call through.
type vs3SidecarWriteFailFS struct {
	cli.EdgeFS
}

// WriteFileAtomic fails for a sidecar path.
func (f vs3SidecarWriteFailFS) WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if strings.HasSuffix(path, ".vec.json") {
		return errVs3SidecarDenied
	}

	return f.EdgeFS.WriteFileAtomic(path, data, perm)
}

// vs3ActivateResult decodes the per-ref result of a served activate.
func vs3ActivateResult(t *testing.T, resp cli.ServeResponse) vs3ActivateOutcome {
	t.Helper()

	var outcome vs3ActivateOutcome

	err := json.Unmarshal(resp.Body, &outcome)
	if err != nil {
		t.Fatalf("decoding the activate result %s: %v", resp.Body, err)
	}

	return outcome
}

// vs3Envelope decodes the raw show envelope of resp: every value of it is a string.
func vs3Envelope(t *testing.T, resp cli.ServeResponse) map[string]string {
	t.Helper()

	var envelope map[string]string

	err := json.Unmarshal(resp.Body, &envelope)
	if err != nil {
		t.Fatalf("decoding the envelope %s: %v", resp.Body, err)
	}

	return envelope
}

// vs3ExchangedNote is a live fact note that carries an xid and a wikilink, so engram show prints the exchange-hash
// header line and an outbound link.
func vs3ExchangedNote() string {
	return liveFactNote("5", "xid: "+seqID(vs3EntropyStart+1)+"\n") + "\nSee [[2026-09-21.next-hop]].\n"
}

// vs3Frontmatter parses the frontmatter of the note at path.
func vs3Frontmatter(t *testing.T, path string) map[string]any {
	t.Helper()

	frontmatter, _, found := strings.Cut(strings.TrimPrefix(readFileString(t, path), "---\n"), "\n---\n")
	if !found {
		t.Fatalf("%s has no frontmatter", path)
	}

	var fields map[string]any

	unmarshalErr := yaml.Unmarshal([]byte(frontmatter), &fields)
	if unmarshalErr != nil {
		t.Fatalf("parsing the frontmatter of %s: %v", path, unmarshalErr)
	}

	return fields
}

// vs3Get serves GET pattern with query over vault.
func vs3Get(t *testing.T, deps cli.Deps, vault, pattern string, query map[string][]string) cli.ServeResponse {
	t.Helper()

	routes := cli.ServeRoutes(deps, vault, vs1ServerName, t.TempDir())

	return routeFor(t, routes, pattern).Serve(t.Context(), cli.ServeRequest{Query: query})
}

// vs3KeyedPayload decodes a dedupe-keys query response.
func vs3KeyedPayload(t *testing.T, resp cli.ServeResponse) vs3KeyedBody {
	t.Helper()

	var payload vs3KeyedBody

	err := yaml.Unmarshal(resp.Body, &payload)
	if err != nil {
		t.Fatalf("decoding the payload %s: %v", resp.Body, err)
	}

	return payload
}

// vs3LastUsed is the last-used date of the sidecar of the note basename in vault.
func vs3LastUsed(t *testing.T, vault, basename string) string {
	t.Helper()

	return vs3LastUsedAt(t, filepath.Join(vault, basename+".md"))
}

// vs3LastUsedAt is the last-used date of the sidecar of the note at notePath.
func vs3LastUsedAt(t *testing.T, notePath string) string {
	t.Helper()

	sidecar, err := embed.UnmarshalSidecar(readFileBytes(t, embed.SidecarPath(notePath)))
	if err != nil {
		t.Fatalf("reading the sidecar of %s: %v", notePath, err)
	}

	return sidecar.LastUsed
}

// vs3LearnAt learns a live fact into dir and returns the note's path.
func vs3LearnAt(t *testing.T, deps cli.Deps, dir, slug string) string {
	t.Helper()

	vs2LearnLive(t, deps, dir, slug)

	return vs1NoteWithSlug(t, dir, slug)
}

// vs3LearnEarlier learns a live fact into vault on an earlier day than the scenarios' clock and returns its
// basename.
func vs3LearnEarlier(t *testing.T, vault, slug string) string {
	t.Helper()

	return strings.TrimSuffix(filepath.Base(vs3LearnAt(t, vs3LearnerDeps(), vault, slug)), ".md")
}

// vs3LearnerDeps is the served node's deps on an earlier day than the scenarios' clock, so a sidecar written by
// them is not already on the day a later activate stamps.
func vs3LearnerDeps() cli.Deps {
	deps := vs1ServerDeps()
	deps.Now = func() time.Time { return vs1Now().AddDate(0, -vs3MonthsEarlier, 0) }

	return deps
}

// vs3Lines returns the lines of text that contain marker.
func vs3Lines(text, marker string) []string {
	var lines []string

	for line := range strings.SplitSeq(text, "\n") {
		if strings.Contains(line, marker) {
			lines = append(lines, line)
		}
	}

	return lines
}

// vs3PlantSidecarOnly writes the note at path and a stale sidecar beside it, as a decoy the activate must not reach.
func vs3PlantSidecarOnly(t *testing.T, path string) {
	t.Helper()

	const staleDay = "2020-01-01"

	writeVaultNote(t, filepath.Dir(path), filepath.Base(path), liveFactNote("1", ""))

	stale := embed.MarshalSidecar(embed.Sidecar{SchemaVersion: embed.SidecarSchemaVersion, LastUsed: staleDay})
	writeErr := os.WriteFile(embed.SidecarPath(path), stale, vs3FilePerm)

	if writeErr != nil {
		t.Fatalf("writing the decoy sidecar: %v", writeErr)
	}
}

// vs3PostActivate serves POST /activate naming refs over vault.
func vs3PostActivate(t *testing.T, deps cli.Deps, vault string, refs ...string) cli.ServeResponse {
	t.Helper()

	body, err := json.Marshal(map[string][]string{"notes": refs})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	routes := cli.ServeRoutes(deps, vault, vs1ServerName, t.TempDir())

	return routeFor(t, routes, "/activate").Serve(t.Context(), cli.ServeRequest{Body: body})
}

// vs3PostLearn serves POST /learn with the raw JSON body over vault.
func vs3PostLearn(t *testing.T, deps cli.Deps, vault, body string) cli.ServeResponse {
	t.Helper()

	routes := cli.ServeRoutes(deps, vault, vs1ServerName, t.TempDir())

	return routeFor(t, routes, "/learn").Serve(t.Context(), cli.ServeRequest{Body: []byte(body)})
}

// vs3Runbook learns a live runbook with one trigger into vault.
func vs3Runbook(t *testing.T, deps cli.Deps, vault, slug, trigger string) {
	t.Helper()

	learnLocal(t, deps, vault, cli.LearnArgs{
		Type: "runbook", Slug: slug, Position: "top", Source: "local",
		Situation: "a runbook for " + slug, DoneWhen: "done", Body: "1. step", Triggers: []string{trigger},
	})
}

// vs3RunbookOffer is a served learn body for a runbook offered by the child vault of childOrigin.
func vs3RunbookOffer(slug string, offer cli.LearnOffer) cli.LearnArgs {
	return cli.LearnArgs{
		Type: "runbook", Slug: slug, Position: "top", Source: "child",
		Situation: "an offered runbook", DoneWhen: "done", Body: "1. step",
		Repo: "git@github.com:example/child.git", User: "child@example.com", Offer: offer,
	}
}

// vs3Server is the served node's deps, whose entropy is deterministic so the xid the server mints is known, and a
// vault with its ID.
func vs3Server(t *testing.T) (cli.Deps, string) {
	t.Helper()

	deps := vs1ServerDeps()
	deps.RandRead = seqRand(vs3EntropyStart)

	return deps, newServeVault(t)
}

// vs3WithForged is the served learn body of args with the raw JSON members forged added.
func vs3WithForged(t *testing.T, args cli.LearnArgs, forged string) string {
	t.Helper()

	body, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if forged == "" {
		return string(body)
	}

	return strings.TrimSuffix(string(body), "}") + "," + forged + "}"
}

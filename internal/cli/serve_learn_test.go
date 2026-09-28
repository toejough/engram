package cli_test

// Served learn (design D7, vault-serve-api "Served learn SHALL update a
// pending offer in place only for the same origin", "Served learn SHALL
// return an offer receipt naming the pending note and the vault"; tasks
// 4.3): input validation (400/409), origin matching, offer.for resolution,
// top-level placement, the receipt, and the one locked section.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestServeLearn_AcceptedOfferRetry: a live note still carrying
// offer.origin (an accepted offer) answers a same-key retry with its own
// receipt and no write; a different key writes a new pending note whose
// offer.for names the live note.
func TestServeLearn_AcceptedOfferRetry(t *testing.T) {
	t.Parallel()

	t.Run("same key writes nothing", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newServeVault(t)
		live := writeVaultNote(t, vault, "4.2026-01-04.accepted.md",
			liveFactNote("4", "offer:\n  origin: "+childOrigin+"\n  key: key-one\n"))

		before := snapshotVault(t, vault)
		resp := serveLearnRequest(t, vault, offeredFact("accepted", "object", offerOf(childOrigin, "key-one", "")))

		g.Expect(resp.Status).To(Equal(200))
		g.Expect(snapshotVault(t, vault)).To(Equal(before), "a retry after acceptance writes nothing")

		receipt := decodeReceipt(t, resp)
		g.Expect(receipt.Basename).To(Equal("4.2026-01-04.accepted"))
		g.Expect(receipt.Luhmann).To(Equal("4"))
		g.Expect(receipt.Pending).To(BeFalse())
		g.Expect(receipt.VaultID).To(Equal(serverVaultID))
		g.Expect(receipt.StoredHash).To(Equal(fileExchangeHash(t, live)))
	})

	t.Run("different key offers an amend to the live note", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newServeVault(t)
		live := writeVaultNote(t, vault, "4.2026-01-04.accepted.md",
			liveFactNote("4", "offer:\n  origin: "+childOrigin+"\n  key: key-one\n"))
		liveBefore := readFileString(t, live)

		resp := serveLearnRequest(t, vault, offeredFact("accepted", "changed", offerOf(childOrigin, "key-two", "")))

		g.Expect(resp.Status).To(Equal(200))
		g.Expect(readFileString(t, live)).To(Equal(liveBefore), "the live note is unchanged")

		receipt := decodeReceipt(t, resp)
		g.Expect(receipt.Basename).NotTo(Equal("4.2026-01-04.accepted"))
		g.Expect(receipt.Pending).To(BeTrue())
		g.Expect(receipt.For).To(Equal("4.2026-01-04.accepted"))

		written := readFileString(t, filepath.Join(vault, receipt.Basename+".md"))
		g.Expect(written).To(ContainSubstring("pending: true"))
		g.Expect(written).To(ContainSubstring("for: 4.2026-01-04.accepted"))
		g.Expect(written).To(ContainSubstring("key: key-two"))
	})
}

// TestServeLearn_AmendBeforeCurationUpdatesInPlace (H3): a second offer from
// the same origin with a new key rewrites the pending note in place — same
// basename, new content, key and path, re-embedded — and leaves exactly one
// pending note.
func TestServeLearn_AmendBeforeCurationUpdatesInPlace(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	deps := serveTestDeps()

	first := serveLearnWith(t, deps, vault,
		offeredFact("offered", "first-object", offerOf(childOrigin, "key-one", "", childVaultID)))
	g.Expect(first.Status).To(Equal(200))

	firstReceipt := decodeReceipt(t, first)
	notePath := filepath.Join(vault, firstReceipt.Basename+".md")
	sidecarBefore := readFileString(t, embed.SidecarPath(notePath))

	amended := offeredFact("offered", "amended-object", offerOf(childOrigin, "key-two", "", childVaultID))
	amended.User = "second-declared@example.com"

	second := serveLearnWith(t, deps, vault, amended)
	g.Expect(second.Status).To(Equal(200))

	secondReceipt := decodeReceipt(t, second)
	g.Expect(secondReceipt.Basename).To(Equal(firstReceipt.Basename))
	g.Expect(secondReceipt.Luhmann).To(Equal(firstReceipt.Luhmann))
	g.Expect(secondReceipt.Pending).To(BeTrue())

	written := readFileString(t, notePath)
	g.Expect(written).To(ContainSubstring("object: amended-object"))
	g.Expect(written).NotTo(ContainSubstring("first-object"))
	g.Expect(written).To(ContainSubstring("key: key-two"))
	g.Expect(written).To(ContainSubstring("pending: true"))
	g.Expect(written).To(ContainSubstring("user: second-declared@example.com"))
	g.Expect(written).To(ContainSubstring("origin: " + childOrigin))
	g.Expect(secondReceipt.StoredHash).To(Equal(fileExchangeHash(t, notePath)))
	g.Expect(readFileString(t, embed.SidecarPath(notePath))).NotTo(Equal(sidecarBefore), "re-embedded")
	g.Expect(noteFiles(t, vault)).To(HaveLen(1), "no second pending note")
}

// TestServeLearn_ConcurrentSameOriginLeavesOnePending: same-origin offers
// racing each other through the served learn leave exactly one pending note
// (the lookup and the write share one locked section).
func TestServeLearn_ConcurrentSameOriginLeavesOnePending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	deps := serveTestDeps()

	const racers = 6

	var wg sync.WaitGroup

	statuses := make([]int, racers)

	for index := range racers {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()

			args := offeredFact("raced", fmt.Sprintf("object-%d", index),
				offerOf(childOrigin, fmt.Sprintf("key-%d", index), "", childVaultID))
			statuses[index] = serveLearnWith(t, deps, vault, args).Status
		}(index)
	}

	wg.Wait()

	for _, status := range statuses {
		g.Expect(status).To(Equal(200))
	}

	g.Expect(noteFiles(t, vault)).To(HaveLen(1), "exactly one pending note for one origin")
}

// TestServeLearn_CycleRefusalReportsVaultID (ruling S12): the 409 loop
// refusal carries the server's vault ID, so a child whose own ID it is can
// recognize itself as its own parent and cache the parent's ID.
func TestServeLearn_CycleRefusalReportsVaultID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	resp := serveLearnRequest(t, vault,
		offeredFact("looped", "object", offerOf(childOrigin, "k", "", childVaultID, serverVaultID)))

	g.Expect(resp.Status).To(Equal(409))

	var body struct {
		Error   string `json:"error"`
		Reason  string `json:"reason"`
		VaultID string `json:"vault_id"` //nolint:tagliatelle // the D7 wire key
	}
	g.Expect(json.Unmarshal(resp.Body, &body)).To(Succeed())
	g.Expect(body.Error).NotTo(BeEmpty())
	g.Expect(body.Reason).To(Equal("loop"))
	g.Expect(body.VaultID).To(Equal(serverVaultID))
}

// TestServeLearn_DuplicateKeyWritesNothing: the same origin and key as an
// existing pending note return that note's receipt and change no file.
func TestServeLearn_DuplicateKeyWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	deps := serveTestDeps()
	args := offeredFact("offered", "object", offerOf(childOrigin, "key-one", "", childVaultID))

	first := serveLearnWith(t, deps, vault, args)
	g.Expect(first.Status).To(Equal(200))

	before := snapshotVault(t, vault)

	second := serveLearnWith(t, deps, vault, args)
	g.Expect(second.Status).To(Equal(200))
	g.Expect(snapshotVault(t, vault)).To(Equal(before), "no vault file changes")
	g.Expect(decodeReceipt(t, second)).To(Equal(decodeReceipt(t, first)))
}

// TestServeLearn_ForIgnoresNonExchangeNotes: offer.for resolves only to
// fact, feedback and runbook notes that parse — a target that is a vocab
// note, a file with no frontmatter, or a note whose frontmatter does not
// parse is unresolved, so it is dropped.
func TestServeLearn_ForIgnoresNonExchangeNotes(t *testing.T) {
	t.Parallel()

	targets := map[string]string{
		"20.2026-01-20.term":     "---\ntype: term\nterm: t\ndescription: d\n---\n\nbody\n",
		"21.2026-01-21.plain":    "no frontmatter at all\n",
		"22.2026-01-22.bad-yaml": "---\ntype: fact\nsituation: [unclosed\n---\n\nbody\n",
	}

	for target, content := range targets {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newServeVault(t)
			writeVaultNote(t, vault, target+".md", content)

			resp := serveLearnRequest(t, vault, offeredFact("offer", "object", offerOf(childOrigin, "k", target)))
			g.Expect(resp.Status).To(Equal(200), string(resp.Body))

			receipt := decodeReceipt(t, resp)
			g.Expect(receipt.For).To(BeEmpty())
			g.Expect(readFileString(t, filepath.Join(vault, receipt.Basename+".md"))).NotTo(ContainSubstring("for:"))
		})
	}
}

// TestServeLearn_ForResolution: offer.for resolves against live basenames,
// then live aliases, then pending notes; a live target is left unchanged
// and reported as `for`; a pending target of another origin is never
// overwritten; an unresolvable target is dropped.
func TestServeLearn_ForResolution(t *testing.T) {
	t.Parallel()

	t.Run("live target", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newServeVault(t)
		live := writeVaultNote(t, vault, "7.2026-01-07.existing.md", liveFactNote("7", ""))
		liveBefore := readFileString(t, live)

		resp := serveLearnRequest(t, vault,
			offeredFact("amend-offer", "object", offerOf(childOrigin, "k", "7.2026-01-07.existing")))
		g.Expect(resp.Status).To(Equal(200))

		receipt := decodeReceipt(t, resp)
		g.Expect(receipt.For).To(Equal("7.2026-01-07.existing"))
		g.Expect(readFileString(t, live)).To(Equal(liveBefore), "the live target is unchanged and live")
		g.Expect(readFileString(t, filepath.Join(vault, receipt.Basename+".md"))).
			To(ContainSubstring("for: 7.2026-01-07.existing"))
	})

	t.Run("renamed target resolves through aliases", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newServeVault(t)
		writeVaultNote(t, vault, "8.2026-01-08.renamed.md", liveFactNote("8", "aliases:\n  - 3.2026-01-03.old-name\n"))

		resp := serveLearnRequest(t, vault,
			offeredFact("amend-offer", "object", offerOf(childOrigin, "k", "3.2026-01-03.old-name")))
		g.Expect(resp.Status).To(Equal(200))

		receipt := decodeReceipt(t, resp)
		g.Expect(receipt.For).To(Equal("8.2026-01-08.renamed"))
		g.Expect(readFileString(t, filepath.Join(vault, receipt.Basename+".md"))).
			To(ContainSubstring("for: 8.2026-01-08.renamed"))
	})

	t.Run("no cross-origin overwrite", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newServeVault(t)
		other := writeVaultNote(t, vault, "5.2026-01-05.from-a.md",
			pendingOfferFactNote("5", "offer:\n  origin: "+otherOrigin+"\n  key: key-a\n"))
		otherBefore := readFileString(t, other)

		resp := serveLearnRequest(t, vault,
			offeredFact("from-b", "object", offerOf(childOrigin, "key-b", "5.2026-01-05.from-a")))
		g.Expect(resp.Status).To(Equal(200))

		receipt := decodeReceipt(t, resp)
		g.Expect(receipt.Basename).NotTo(Equal("5.2026-01-05.from-a"))
		g.Expect(receipt.For).To(BeEmpty(), "for names only a live target")
		g.Expect(readFileString(t, other)).To(Equal(otherBefore), "origin A's pending note is unchanged")
		g.Expect(readFileString(t, filepath.Join(vault, receipt.Basename+".md"))).
			To(ContainSubstring("for: 5.2026-01-05.from-a"))
	})

	t.Run("unresolvable target is dropped", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newServeVault(t)

		resp := serveLearnRequest(t, vault,
			offeredFact("amend-offer", "object", offerOf(childOrigin, "k", "99.2026-01-01.nowhere")))
		g.Expect(resp.Status).To(Equal(200))
		g.Expect(string(resp.Body)).NotTo(ContainSubstring(`"for"`))

		receipt := decodeReceipt(t, resp)
		written := readFileString(t, filepath.Join(vault, receipt.Basename+".md"))
		g.Expect(written).NotTo(ContainSubstring("for:"))
		g.Expect(written).NotTo(ContainSubstring("nowhere"))
	})
}

// TestServeLearn_InvalidOfferIsRejected (r4 L-C): an oversized or malformed
// offer.path, or a malformed offer.origin, answers 400 and writes nothing;
// a path already holding the server's own vault ID answers 409 and writes
// nothing (r3-7).
func TestServeLearn_InvalidOfferIsRejected(t *testing.T) {
	t.Parallel()

	seventeen := make([]string, 0, maxOfferPathEntries+1)
	for i := range maxOfferPathEntries + 1 {
		seventeen = append(seventeen, fmt.Sprintf("%032x", i+1))
	}

	cases := []struct {
		name   string
		offer  cli.LearnOffer
		status int
	}{
		{"path of 17 entries", cli.LearnOffer{Origin: childOrigin, Key: "k", Path: seventeen}, 400},
		{"uppercase hex entry", offerOf(childOrigin, "k", "", strings.ToUpper(childVaultID)), 400},
		{"31-character entry", offerOf(childOrigin, "k", "", childVaultID[:31]), 400},
		{"non-hex entry", offerOf(childOrigin, "k", "", strings.Repeat("g", 32)), 400},
		{"origin without separator", offerOf(childVaultID+childXID, "k", ""), 400},
		{"origin with a short half", offerOf(childVaultID+":abc", "k", ""), 400},
		{"origin with three parts", offerOf(childOrigin+":"+childXID, "k", ""), 400},
		{"key without origin", offerOf("", "k", ""), 400},
		{"path without origin", offerOf("", "", "", childVaultID), 400},
		{"cycle through this server", offerOf(childOrigin, "k", "", childVaultID, serverVaultID), 409},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newServeVault(t)
			before := snapshotVault(t, vault)

			resp := serveLearnRequest(t, vault, offeredFact("rejected", "object", testCase.offer))

			g.Expect(resp.Status).To(Equal(testCase.status))
			g.Expect(snapshotVault(t, vault)).To(Equal(before), "nothing is written")

			var errBody struct {
				Error string `json:"error"`
			}
			g.Expect(json.Unmarshal(resp.Body, &errBody)).To(Succeed())
			g.Expect(errBody.Error).NotTo(BeEmpty())
		})
	}
}

// TestServeLearn_LateRetryOfAcceptedKeyWritesNothing (idempotency first):
// a late retry whose key was already recorded — here on an accepted live
// note — returns that note's receipt and writes nothing, even though a newer
// same-origin pending amend exists; it must never rewrite that amend back to
// the old content.
func TestServeLearn_LateRetryOfAcceptedKeyWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	writeVaultNote(t, vault, "4.2026-01-04.accepted.md",
		liveFactNote("4", "offer:\n  origin: "+childOrigin+"\n  key: key-one\n"))
	writeVaultNote(t, vault, "5.2026-01-05.newer-amend.md",
		pendingOfferFactNote("5", "offer:\n  origin: "+childOrigin+"\n  key: key-two\n"))

	before := snapshotVault(t, vault)

	resp := serveLearnRequest(t, vault, offeredFact("accepted", "old-content", offerOf(childOrigin, "key-one", "")))
	g.Expect(resp.Status).To(Equal(200))
	g.Expect(snapshotVault(t, vault)).To(Equal(before), "the newer pending amend is untouched")

	receipt := decodeReceipt(t, resp)
	g.Expect(receipt.Basename).To(Equal("4.2026-01-04.accepted"))
	g.Expect(receipt.Pending).To(BeFalse())
}

// TestServeLearn_LateRetryOfSupersededKeyWritesNothing (ruling S10): a
// pending offer rewritten from K1 to K2 keeps K1 in offer.prior_keys, so a
// late K1 retry returns the current receipt and never reverts the note —
// and the history survives acceptance, so the same holds for the live note.
func TestServeLearn_LateRetryOfSupersededKeyWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	deps := serveTestDeps()

	first := offeredFact("offered", "first-object", offerOf(childOrigin, "key-one", "", childVaultID))
	g.Expect(serveLearnWith(t, deps, vault, first).Status).To(Equal(200))

	second := serveLearnWith(t, deps, vault,
		offeredFact("offered", "second-object", offerOf(childOrigin, "key-two", "", childVaultID)))
	g.Expect(second.Status).To(Equal(200))

	current := decodeReceipt(t, second)
	notePath := filepath.Join(vault, current.Basename+".md")
	g.Expect(readFileString(t, notePath)).To(MatchRegexp(`prior_keys:\s*\n\s*- key-one`))

	before := snapshotVault(t, vault)

	late := serveLearnWith(t, deps, vault, first)
	g.Expect(late.Status).To(Equal(200))
	g.Expect(snapshotVault(t, vault)).To(Equal(before), "a late K1 retry never reverts the K2 content")
	g.Expect(decodeReceipt(t, late)).To(Equal(current))

	// Accept the offer: the history survives, so a late K1 retry after
	// acceptance is still a no-op answered by the live note.
	cleared := false

	amendErr := cli.ExportRunAmend(context.Background(),
		cli.AmendArgs{Vault: vault, Target: current.Luhmann, Pending: &cleared},
		cli.ExportNewAmendDeps(deps), io.Discard)
	g.Expect(amendErr).NotTo(HaveOccurred())

	accepted := readFileString(t, notePath)
	g.Expect(accepted).NotTo(ContainSubstring("pending: true"))
	g.Expect(accepted).To(MatchRegexp(`prior_keys:\s*\n\s*- key-one`))

	afterAccept := snapshotVault(t, vault)

	lateAfterAccept := serveLearnWith(t, deps, vault, first)
	g.Expect(lateAfterAccept.Status).To(Equal(200))
	g.Expect(snapshotVault(t, vault)).To(Equal(afterAccept))
	g.Expect(decodeReceipt(t, lateAfterAccept).Basename).To(Equal(current.Basename))
	g.Expect(decodeReceipt(t, lateAfterAccept).Pending).To(BeFalse())
}

// TestServeLearn_LockSpansLookupWriteEmbedReceipt: every note read (the
// origin/for lookup), the note write and the sidecar re-embed happen inside
// the one vault-lock section — for a new pending note and for an in-place
// update alike.
func TestServeLearn_LockSpansLookupWriteEmbedReceipt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	writeVaultNote(t, vault, "7.2026-01-07.existing.md", liveFactNote("7", ""))

	log := &eventLog{}
	deps := serveTestDeps()
	deps.FS = recordingFS{EdgeFS: deps.FS, log: log}
	deps.Lock = recordingLocker{inner: deps.Lock, log: log}

	for _, key := range []string{"key-one", "key-two"} {
		log.reset()

		resp := serveLearnWith(t, deps, vault,
			offeredFact("locked", "object-"+key, offerOf(childOrigin, key, "7.2026-01-07.existing")))
		g.Expect(resp.Status).To(Equal(200))

		events := log.snapshot()
		lockAt := slices.Index(events, "lock")
		unlockAt := slices.Index(events, "unlock")
		g.Expect(lockAt).To(BeNumerically(">=", 0), "events: %v", events)
		g.Expect(unlockAt).To(BeNumerically(">", lockAt), "events: %v", events)

		for index, event := range events {
			if !strings.Contains(event, ".md") && !strings.Contains(event, ".vec.json") {
				continue
			}

			g.Expect(index).To(And(BeNumerically(">", lockAt), BeNumerically("<", unlockAt)),
				"%s happened outside the lock (%s): %v", event, key, events)
		}

		locked := events[lockAt:unlockAt]
		g.Expect(slices.ContainsFunc(locked, func(event string) bool {
			return strings.HasPrefix(event, "read ") && strings.Contains(event, "7.2026-01-07.existing.md")
		})).To(BeTrue(), "the lookup reads notes under the lock: %v", events)
		g.Expect(slices.ContainsFunc(locked, func(event string) bool {
			return strings.HasPrefix(event, "write ") && strings.HasSuffix(event, ".vec.json")
		})).To(BeTrue(), "the re-embed happens under the lock: %v", events)
	}
}

// TestServeLearn_NewOfferRecordsOfferAndXID: a new pending note gets a
// server xid, records offer.{origin,key,for,path}, lands at top level
// whatever target/position the caller sent, and its receipt names it,
// the vault, and the stored exchange hash.
func TestServeLearn_NewOfferRecordsOfferAndXID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	writeVaultNote(t, vault, "12.2026-01-12.existing.md", liveFactNote("12", ""))

	args := offeredFact("placed", "object", offerOf(childOrigin, "key-one", "12.2026-01-12.existing", childVaultID))
	args.Target = "12"
	args.Position = "continuation"

	resp := serveLearnRequest(t, vault, args)
	g.Expect(resp.Status).To(Equal(200))

	receipt := decodeReceipt(t, resp)
	g.Expect(receipt.Status).To(Equal("offer received"))
	g.Expect(receipt.Luhmann).To(MatchRegexp(`^[0-9]+$`), "top-level placement")
	g.Expect(receipt.Luhmann).NotTo(Equal("12a"))
	g.Expect(receipt.Basename).To(HavePrefix(receipt.Luhmann + "."))
	g.Expect(receipt.Pending).To(BeTrue())
	g.Expect(receipt.VaultID).To(Equal(serverVaultID))
	g.Expect(receipt.For).To(Equal("12.2026-01-12.existing"))

	notePath := filepath.Join(vault, receipt.Basename+".md")
	written := readFileString(t, notePath)
	g.Expect(receipt.StoredHash).To(Equal(fileExchangeHash(t, notePath)))
	g.Expect(written).To(ContainSubstring("pending: true"))
	g.Expect(written).To(MatchRegexp(`(?m)^xid: [0-9a-f]{32}$`))
	g.Expect(written).NotTo(ContainSubstring("xid: "+childXID), "the server mints its own xid")
	g.Expect(written).To(ContainSubstring("origin: " + childOrigin))
	g.Expect(written).To(ContainSubstring("key: key-one"))
	g.Expect(written).To(ContainSubstring("for: 12.2026-01-12.existing"))
	g.Expect(written).To(ContainSubstring("- " + childVaultID))
}

// TestServeLearn_PathValidationProperty: for any offer.path, the served
// learn answers 400 (and writes nothing) exactly when the path has more
// than 16 entries or any entry is not 32 lowercase hex characters.
func TestServeLearn_PathValidationProperty(t *testing.T) {
	t.Parallel()

	deps := serveTestDeps()
	validEntry := rapid.StringMatching(`[0-9a-f]{32}`)
	anyEntry := rapid.OneOf(validEntry, rapid.String(), rapid.StringMatching(`[0-9a-fA-F]{30,34}`))

	rapid.Check(t, func(rt *rapid.T) {
		path := rapid.SliceOfN(anyEntry, 0, maxOfferPathEntries+3).Draw(rt, "path")
		if slices.Contains(path, serverVaultID) {
			rt.Skip("the server's own ID is the 409 case")
		}

		vault := t.TempDir()
		writeVaultID(rt, vault)

		resp := serveLearnWith(rt, deps, vault,
			offeredFact("property", "object", cli.LearnOffer{Origin: childOrigin, Key: "k", Path: path}))

		wantValid := len(path) <= maxOfferPathEntries && !slices.ContainsFunc(path, func(entry string) bool {
			return !exchangeIDPattern.MatchString(entry)
		})

		switch {
		case wantValid && resp.Status != 200:
			rt.Fatalf("valid path %q: status %d: %s", path, resp.Status, resp.Body)
		case !wantValid && resp.Status != 400:
			rt.Fatalf("invalid path %q: status %d, want 400", path, resp.Status)
		case !wantValid && len(noteFiles(rt, vault)) != 0:
			rt.Fatalf("invalid path %q wrote a note", path)
		}
	})
}

// TestServeLearn_PriorKeysAreBounded: the superseded-key history keeps only
// the most recent keys, oldest dropped, and the wire cannot set it.
func TestServeLearn_PriorKeysAreBounded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	deps := serveTestDeps()

	const rewrites = 12

	var receipt receiptBody

	for index := range rewrites {
		resp := serveLearnWith(t, deps, vault, offeredFact("offered", fmt.Sprintf("object-%d", index),
			offerOf(childOrigin, fmt.Sprintf("key-%02d", index), "", childVaultID)))
		g.Expect(resp.Status).To(Equal(200))

		receipt = decodeReceipt(t, resp)
	}

	written := readFileString(t, filepath.Join(vault, receipt.Basename+".md"))
	g.Expect(written).To(ContainSubstring("key: key-11"))
	g.Expect(strings.Count(written, "- key-")).To(Equal(maxPriorOfferKeys))
	g.Expect(written).To(ContainSubstring("- key-10"))
	g.Expect(written).To(ContainSubstring("- key-03"))
	g.Expect(written).NotTo(ContainSubstring("- key-02"), "the oldest keys are dropped")

	forged := serveLearnRequestBody(t, vault, `{"type":"fact","slug":"forged","situation":"s","subject":"a",`+
		`"predicate":"b","object":"c","user":"u@example.com","offer":{"origin":"`+childOrigin+`","key":"k",`+
		`"prior_keys":["forged-prior"],"priorKeys":["forged-prior"],"PriorKeys":["forged-prior"]}}`)
	g.Expect(forged.Status).To(Equal(200))
	g.Expect(readFileString(t, filepath.Join(vault, decodeReceipt(t, forged).Basename+".md"))).
		NotTo(ContainSubstring("forged-prior"), "prior keys are never settable from the wire")
}

// TestServeLearn_StoredHashMatchesChildHash (golden property, design D3
// "Stored hash"): for any offered content, the stored_hash the parent
// returns equals the exchange hash the child computes over its own note
// with the same content — identity, luhmann, pending and the exchange
// fields never enter it.
func TestServeLearn_StoredHashMatchesChildHash(t *testing.T) {
	t.Parallel()

	deps := serveTestDeps()
	text := rapid.StringMatching(`[A-Za-z]{3}[A-Za-z0-9 .,:#'"!?()-]{0,37}`)

	rapid.Check(t, func(rt *rapid.T) {
		args := cli.LearnArgs{
			Type: rapid.SampledFrom([]string{"fact", "feedback", "runbook"}).Draw(rt, "type"),
			Slug: "golden", Position: "top", Source: "child",
			Situation: text.Draw(rt, "situation"),
			Subject:   text.Draw(rt, "subject"), Predicate: text.Draw(rt, "predicate"), Object: text.Draw(rt, "object"),
			Behavior: text.Draw(rt, "behavior"), Impact: text.Draw(rt, "impact"), Action: text.Draw(rt, "action"),
			DoneWhen: text.Draw(rt, "doneWhen"),
			Body:     "1. " + text.Draw(rt, "step") + "\n2. " + text.Draw(rt, "step2"),
			RedFlags: rapid.SliceOfN(text, 0, 3).Draw(rt, "redFlags"),
			Triggers: rapid.SliceOfN(text, 0, 2).Draw(rt, "triggers"),
		}

		childVault := t.TempDir()
		parentVault := t.TempDir()
		writeVaultID(rt, parentVault)

		childArgs := args
		childArgs.Vault = childVault
		childArgs.VaultName = "child-vault"

		var childOut strings.Builder

		runErr := cli.ExportRunLearn(context.Background(), childArgs, cli.ExportNewLearnDeps(deps), &childOut)
		if runErr != nil {
			rt.Fatalf("child learn: %v", runErr)
		}

		childHash, hashErr := cli.ExportExchangeHash(readFileBytes(rt, strings.TrimSpace(childOut.String())))
		if hashErr != nil {
			rt.Fatalf("child hash: %v", hashErr)
		}

		offered := args
		offered.User = "child@example.com"
		offered.Offer = offerOf(childOrigin, "key", "", childVaultID)

		resp := serveLearnWith(rt, deps, parentVault, offered)
		if resp.Status != 200 {
			rt.Fatalf("served learn: status %d: %s", resp.Status, resp.Body)
		}

		receipt := decodeReceipt(rt, resp)
		if receipt.StoredHash != childHash {
			rt.Fatalf("stored_hash %q != child hash %q", receipt.StoredHash, childHash)
		}
	})
}

// TestServeLearn_UnreadableNoteFailsTheRequest: a note the lookup cannot
// read (other than one deleted meanwhile) fails the request with a 5xx the
// child retries — skipping it could miss a same-origin pending offer and
// write a second one.
func TestServeLearn_UnreadableNoteFailsTheRequest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	pending := writeVaultNote(t, vault, "5.2026-01-05.pending.md",
		pendingOfferFactNote("5", "offer:\n  origin: "+childOrigin+"\n  key: key-one\n"))
	g.Expect(os.Chmod(pending, 0o000)).To(Succeed())

	resp := serveLearnRequest(t, vault, offeredFact("second", "object", offerOf(childOrigin, "key-two", "")))

	g.Expect(resp.Status).To(Equal(500))
	g.Expect(noteFiles(t, vault)).To(HaveLen(1), "no second pending note")
}

// unexported constants.
const (
	childOrigin         = childVaultID + ":" + childXID
	childVaultID        = "c0ffee00c0ffee00c0ffee00c0ffee00"
	childXID            = "0123456789abcdef0123456789abcdef"
	maxOfferPathEntries = 16
	maxPriorOfferKeys   = 8
	otherOrigin         = "a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0:b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1"
	serverVaultID       = "9a1e9a1e9a1e9a1e9a1e9a1e9a1e9a1e"
)

// unexported variables.
var (
	exchangeIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)
)

// eventLog is an ordered, concurrency-safe record of FS and lock events.
type eventLog struct {
	mu     sync.Mutex
	events []string
}

func (l *eventLog) add(event string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = append(l.events, event)
}

func (l *eventLog) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = nil
}

func (l *eventLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return append([]string{}, l.events...)
}

// receiptBody is the served learn receipt's wire shape (design D7).
type receiptBody struct {
	Status     string `json:"status"`
	Luhmann    string `json:"luhmann"`
	Basename   string `json:"basename"`
	Pending    bool   `json:"pending"`
	VaultID    string `json:"vault_id"`    //nolint:tagliatelle // design D7 fixes the receipt's snake_case keys
	StoredHash string `json:"stored_hash"` //nolint:tagliatelle // design D7 fixes the receipt's snake_case keys
	For        string `json:"for"`
}

// recordingFS wraps an EdgeFS, logging every read and write in order.
type recordingFS struct {
	cli.EdgeFS

	log *eventLog
}

func (r recordingFS) ReadFile(path string) ([]byte, error) {
	r.log.add("read " + path)

	return r.EdgeFS.ReadFile(path)
}

func (r recordingFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	r.log.add("write " + path)

	return r.EdgeFS.WriteFile(path, data, perm)
}

func (r recordingFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	r.log.add("write " + path)

	return r.EdgeFS.WriteFileAtomic(path, data, perm)
}

func (r recordingFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	r.log.add("write " + path)

	return r.EdgeFS.WriteFileExcl(path, data, perm)
}

// recordingLocker wraps a FileLocker, logging acquire and release.
type recordingLocker struct {
	inner cli.FileLocker
	log   *eventLog
}

func (r recordingLocker) Lock(path string) (func() error, error) {
	unlock, err := r.inner.Lock(path)
	if err != nil {
		return nil, err
	}

	r.log.add("lock")

	return func() error {
		r.log.add("unlock")

		return unlock()
	}, nil
}

// testFataler is the failure surface shared by *testing.T and *rapid.T.
type testFataler interface {
	Helper()
	Fatalf(format string, args ...any)
}

// decodeReceipt decodes a served learn's receipt body.
func decodeReceipt(t testFataler, resp cli.ServeResponse) receiptBody {
	t.Helper()

	var receipt receiptBody

	err := json.Unmarshal(resp.Body, &receipt)
	if err != nil {
		t.Fatalf("decoding receipt %s: %v", resp.Body, err)
	}

	return receipt
}

// fileExchangeHash is the child's exchange hash of the file at path.
func fileExchangeHash(t *testing.T, path string) string {
	t.Helper()

	hash, err := cli.ExportExchangeHash(readFileBytes(t, path))
	if err != nil {
		t.Fatalf("hashing %s: %v", path, err)
	}

	return hash
}

// liveFactNote is a live (non-pending) fact note with extra frontmatter
// lines appended before the closing delimiter.
func liveFactNote(luhmannID, extra string) string {
	return "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\nluhmann: \"" + luhmannID +
		"\"\ncreated: 2026-01-01\nsource: agent\nuser: u\nvault: personal\n" + extra + "---\n\nbody\n"
}

// newServeVault is a temp vault whose ID is serverVaultID.
func newServeVault(t *testing.T) string {
	t.Helper()

	vault := t.TempDir()
	writeVaultID(t, vault)

	return vault
}

// noteFiles lists the vault's .md files.
func noteFiles(t testFataler, vault string) []string {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(vault, "*.md"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	return matches
}

// offerOf builds a wire offer.
func offerOf(origin, key, target string, path ...string) cli.LearnOffer {
	return cli.LearnOffer{Origin: origin, Key: key, For: target, Path: path}
}

// offeredFact is a served learn body for a fact offer from a child.
func offeredFact(slug, object string, offer cli.LearnOffer) cli.LearnArgs {
	return cli.LearnArgs{
		Type: "fact", Slug: slug, Position: "top", Source: "child",
		Situation: "an offered fact", Subject: "engram", Predicate: "offers", Object: object,
		Repo: "git@github.com:example/child.git", User: "child@example.com", Offer: offer,
	}
}

// pendingOfferFactNote is a pending fact note with extra frontmatter lines.
func pendingOfferFactNote(luhmannID, extra string) string {
	return liveFactNote(luhmannID, "pending: true\n"+extra)
}

func readFileBytes(t testFataler, path string) []byte {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	return raw
}

// serveLearnRequest posts args to a fresh route set over vault.
func serveLearnRequest(t *testing.T, vault string, args cli.LearnArgs) cli.ServeResponse {
	t.Helper()

	return serveLearnWith(t, serveTestDeps(), vault, args)
}

// serveLearnRequestBody posts a raw JSON body to /learn.
func serveLearnRequestBody(t *testing.T, vault, body string) cli.ServeResponse {
	t.Helper()

	return routeFor(t, cli.ServeRoutes(serveTestDeps(), vault, "personal", ""), "/learn").
		Serve(context.Background(), cli.ServeRequest{Body: []byte(body)})
}

// serveLearnWith posts args to /learn on the route set built from deps.
func serveLearnWith(t testFataler, deps cli.Deps, vault string, args cli.LearnArgs) cli.ServeResponse {
	t.Helper()

	body, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	for _, route := range cli.ServeRoutes(deps, vault, "personal", "") {
		if route.Pattern == "/learn" {
			return route.Handler.Serve(context.Background(), cli.ServeRequest{Body: body})
		}
	}

	t.Fatalf("no /learn route")

	return cli.ServeResponse{}
}

// serveTestDeps is the real-FS deps with a deterministic stub embedder, so
// learn writes sidecars.
func serveTestDeps() cli.Deps {
	deps := newTestDeps(io.Discard, io.Discard)
	deps.Embed = stubEmbedder{modelID: "test-model@4", dims: 4}

	return deps
}

// snapshotVault maps every file under vault (notes, sidecars, state) to
// its content. The vault lock file is left out: taking the lock is not a
// vault write.
func snapshotVault(t *testing.T, vault string) map[string]string {
	t.Helper()

	files := map[string]string{}

	err := filepath.WalkDir(vault, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || entry.Name() == ".luhmann.lock" {
			return walkErr
		}

		raw, readErr := os.ReadFile(path)
		files[path] = string(raw)

		return readErr
	})
	if err != nil {
		t.Fatalf("walking %s: %v", vault, err)
	}

	return files
}

// writeVaultID writes serverVaultID as vault's ID file.
func writeVaultID(t testFataler, vault string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(vault, ".engram-vault-id"), []byte(serverVaultID+"\n"), 0o600)
	if err != nil {
		t.Fatalf("writing vault id: %v", err)
	}
}

// writeVaultNote writes content as name under vault and returns its path.
func writeVaultNote(t *testing.T, vault, name, content string) string {
	t.Helper()

	path := filepath.Join(vault, name)

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}

	return path
}

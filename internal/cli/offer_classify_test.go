package cli_test

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestClassifyOffer covers every row of design D5's table (spec
// vault-parent-offers, "Local writes SHALL be offered to the parent").
func TestClassifyOffer(t *testing.T) {
	t.Parallel()

	childID := seqID(40)
	plain := offerTestNote{}
	contentAmend := cli.AmendArgs{Object: "new object"}
	clearPending := cli.AmendArgs{ClearPending: true}

	cases := []struct {
		name    string
		command cli.OfferCommandForTest
		amend   cli.AmendArgs
		note    offerTestNote
		want    bool
	}{
		{"learn fact", cli.ExportOfferCmdLearn, cli.AmendArgs{}, plain, true},
		{"learn runbook", cli.ExportOfferCmdLearn, cli.AmendArgs{}, offerTestNote{noteType: "runbook"}, true},
		{"learn qa", cli.ExportOfferCmdLearnQA, cli.AmendArgs{}, plain, false},
		{"identity backfill", cli.ExportOfferCmdIdentityBackfill, cli.AmendArgs{}, plain, false},
		{"registration", cli.ExportOfferCmdRegistration, cli.AmendArgs{}, plain, false},
		{"skill_hash runbook", cli.ExportOfferCmdLearn, cli.AmendArgs{},
			offerTestNote{noteType: "runbook", skillHash: "abc"}, false},
		{"pending note", cli.ExportOfferCmdAmend, contentAmend, offerTestNote{pending: true}, false},
		{"resituate", cli.ExportOfferCmdResituate, cli.AmendArgs{}, plain, true},
		{"content amend, unlinked", cli.ExportOfferCmdAmend, contentAmend, plain, true},
		{"--activate", cli.ExportOfferCmdAmend, cli.AmendArgs{Activate: true}, plain, false},
		{"--supersedes only", cli.ExportOfferCmdAmend,
			cli.AmendArgs{Supersedes: []string{"1.x|updates|c"}}, plain, false},
		{"--chunk-source only", cli.ExportOfferCmdAmend, cli.AmendArgs{ChunkSources: []string{"s#a"}}, plain, false},
		{"--discard", cli.ExportOfferCmdAmend, cli.AmendArgs{Discard: true, Object: "x"}, plain, false},
		{"--clear-pending, not a served offer", cli.ExportOfferCmdAmend, clearPending, plain, false},
		{"--clear-pending (Pending=false), not a served offer", cli.ExportOfferCmdAmend,
			cli.AmendArgs{Pending: new(false)}, plain, false},
		{"--clear-pending of a child's served offer propagates (M14)", cli.ExportOfferCmdAmend, clearPending,
			offerTestNote{origin: childID + ":" + xidA, path: []string{childID}}, true},
		{"--clear-pending of an offer whose origin is the parent", cli.ExportOfferCmdAmend, clearPending,
			offerTestNote{origin: parentVaultID + ":" + xidA, path: []string{parentVaultID}}, false},
		{"--activate of an accepted served offer", cli.ExportOfferCmdAmend, cli.AmendArgs{Activate: true},
			offerTestNote{origin: childID + ":" + xidA}, false},
		{"content amend back to the primary link's hash", cli.ExportOfferCmdAmend, contentAmend,
			offerTestNote{parentVault: parentVaultID, linkVia: "offered", linkMatches: true}, false},
		{"content amend of a pulled note back to its hash", cli.ExportOfferCmdAmend, contentAmend,
			offerTestNote{parentVault: parentVaultID, linkVia: "pulled", linkMatches: true}, false},
		{"content amend away from the primary link's hash", cli.ExportOfferCmdAmend, contentAmend,
			offerTestNote{parentVault: parentVaultID, linkVia: "offered", linkHash: "xh1:other"}, true},
		{"a covered link never suppresses", cli.ExportOfferCmdAmend, contentAmend,
			offerTestNote{parentVault: parentVaultID, linkVia: "covered", linkMatches: true}, true},
		{"an unknown-version link hash never suppresses", cli.ExportOfferCmdAmend, contentAmend,
			offerTestNote{parentVault: parentVaultID, linkVia: "offered", linkHash: "xh0:old"}, true},
		{"a link under another parent's vault never suppresses", cli.ExportOfferCmdAmend, contentAmend,
			offerTestNote{parentVault: seqID(77), linkVia: "offered", linkMatches: true}, true},
		{"parent ID unknown: an equal link hash never suppresses", cli.ExportOfferCmdAmend, contentAmend,
			offerTestNote{parentVault: parentVaultID, linkVia: "offered", linkMatches: true, unknownParent: true}, true},
		{"parent ID unknown: a served offer's clear-pending is queued for the send-time check",
			cli.ExportOfferCmdAmend, clearPending,
			offerTestNote{origin: parentVaultID + ":" + xidA, path: []string{parentVaultID}, unknownParent: true}, true},
		{"near-fold of a pulled note into L bounces up once (B1)", cli.ExportOfferCmdAmend, contentAmend,
			offerTestNote{
				parentVault: parentVaultID, linkVia: "offered", linkHash: "xh1:before-fold",
				coveredNote: "3.2026-09-27.pulled",
			}, true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			known := parentVaultID
			if testCase.note.unknownParent {
				known = ""
			}

			got := cli.ExportClassifyOffer(testCase.command, testCase.amend, testCase.note.render(t), known)
			g.Expect(got).To(Equal(testCase.want))
		})
	}
}

// TestClassifyOffer_BookkeepingNeverOfferedProperty: an amend with no
// content flag is never offered unless it clears the pending marker on a
// served offer from a vault other than the parent (D5, D12).
func TestClassifyOffer_BookkeepingNeverOfferedProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		note := drawOfferTestNote(rt)
		amend := cli.AmendArgs{
			Activate:     rapid.Bool().Draw(rt, "activate"),
			ClearPending: rapid.Bool().Draw(rt, "clear"),
			Discard:      rapid.Bool().Draw(rt, "discard"),
		}

		if rapid.Bool().Draw(rt, "supersedes") {
			amend.Supersedes = []string{"1.x|updates|c"}
		}

		if rapid.Bool().Draw(rt, "chunk") {
			amend.ChunkSources = []string{"s#a"}
		}

		propagates := amend.ClearPending && !amend.Discard && note.origin != "" &&
			!strings.HasPrefix(note.origin, parentVaultID+":")
		got := cli.ExportClassifyOffer(cli.ExportOfferCmdAmend, amend, note.render(rt), parentVaultID)

		if got != propagates {
			rt.Fatalf("bookkeeping amend %+v on %+v: offered=%v, want %v", amend, note, got, propagates)
		}
	})
}

// TestClassifyOffer_ContentFlagsAreOffered: each content flag on its own
// makes an amend an offer (spec: "any content flag").
func TestClassifyOffer_ContentFlagsAreOffered(t *testing.T) {
	t.Parallel()

	flags := map[string]cli.AmendArgs{
		"situation": {Situation: "x"}, "subject": {Subject: "x"}, "predicate": {Predicate: "x"},
		"object": {Object: "x"}, "behavior": {Behavior: "x"}, "impact": {Impact: "x"},
		"action": {Action: "x"}, "done-when": {DoneWhen: "x"}, "body": {Body: "x"},
		"red-flag": {RedFlags: []string{"x"}}, "trigger": {Triggers: []string{"xyz"}},
	}

	for name, amend := range flags {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(cli.ExportClassifyOffer(cli.ExportOfferCmdAmend, amend, offerTestNote{}.render(t), "")).
				To(BeTrue())
		})
	}
}

// TestClassifyOffer_LoopSuppressionProperty (metamorphic): for any offered
// write, a primary link (under the parent) whose hash equals the note's
// exchange hash suppresses it, and a different current-version hash does
// not.
func TestClassifyOffer_LoopSuppressionProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		note := drawOfferTestNote(rt)
		note.parentVault = parentVaultID
		note.linkVia = rapid.SampledFrom([]string{"offered", "pulled"}).Draw(rt, "via")
		command := rapid.SampledFrom([]cli.OfferCommandForTest{
			cli.ExportOfferCmdLearn, cli.ExportOfferCmdAmend, cli.ExportOfferCmdResituate,
		}).Draw(rt, "command")
		amend := cli.AmendArgs{Object: "o"}

		note.linkMatches = true
		if cli.ExportClassifyOffer(command, amend, note.render(rt), parentVaultID) {
			rt.Fatalf("equal primary-link hash was offered: %+v", note)
		}

		note.linkMatches = false
		note.linkHash = "xh1:" + strings.Repeat("0", 64)

		if !cli.ExportClassifyOffer(command, amend, note.render(rt), parentVaultID) {
			rt.Fatalf("changed primary-link hash was not offered: %+v", note)
		}
	})
}

// TestClassifyOffer_NeverOfferedProperty: a pending note or a note carrying
// skill_hash is never offered, whatever the command and flags (D5).
func TestClassifyOffer_NeverOfferedProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		note := drawOfferTestNote(rt)
		if rapid.Bool().Draw(rt, "pending-not-skill") {
			note.pending = true
		} else {
			note.noteType, note.skillHash = "runbook", "h"
		}

		command := drawOfferCommand(rt)
		amend := drawAmendFlags(rt)

		if cli.ExportClassifyOffer(command, amend, note.render(rt), parentVaultID) {
			rt.Fatalf("offered a never-offered note: %+v, command %v, amend %+v", note, command, amend)
		}
	})
}

// offerTestNote describes a note for the classification and payload
// tests; render builds its bytes.
type offerTestNote struct {
	noteType    string
	object      string
	pending     bool
	skillHash   string
	origin      string
	path        []string
	parentVault string
	linkVia     string
	linkHash    string
	linkMatches bool
	coveredNote string
	// unknownParent classifies with no cached parent vault ID.
	unknownParent bool
}

// render builds the note. A link whose hash must match is filled in with
// the note's own exchange hash, which never depends on the parent field.
func (n offerTestNote) render(tester failer) []byte {
	tester.Helper()

	raw := n.renderWithLinkHash(n.linkHash)
	if !n.linkMatches {
		return raw
	}

	return n.renderWithLinkHash(mustHash(tester, raw))
}

func (n offerTestNote) renderWithLinkHash(linkHash string) []byte {
	noteType := n.noteType
	if noteType == "" {
		noteType = "fact"
	}

	var builder strings.Builder

	builder.WriteString("---\ntype: " + noteType + "\ntier: L2\nsituation: when testing\n")

	if noteType == "runbook" {
		builder.WriteString("done_when: done\n")
	} else {
		object := n.object
		if object == "" {
			object = "c"
		}

		builder.WriteString("subject: a\npredicate: b\nobject: " + object + "\n")
	}

	builder.WriteString("luhmann: \"1\"\ncreated: \"2026-09-27\"\nsource: test\nuser: alice\nvault: personal\n")

	if n.skillHash != "" {
		builder.WriteString("skill_hash: " + n.skillHash + "\n")
	}

	if n.pending {
		builder.WriteString("pending: true\n")
	}

	builder.WriteString("xid: " + xidA + "\n")
	n.writeExchange(&builder, linkHash)
	builder.WriteString("---\n\nInformation learned: body.\n")

	return []byte(builder.String())
}

// writeExchange renders the parent and offer fields.
func (n offerTestNote) writeExchange(builder *strings.Builder, linkHash string) {
	if n.parentVault != "" {
		builder.WriteString("parent:\n  vault: " + n.parentVault + "\n  links:\n")
		fmt.Fprintf(builder, "    - note: 5.2026-09-27.p\n      via: %s\n      hash: %s\n", n.linkVia, linkHash)

		if n.coveredNote != "" {
			builder.WriteString("    - note: " + n.coveredNote + "\n      via: covered\n      hash: xh1:c\n")
		}
	}

	if n.origin == "" {
		return
	}

	builder.WriteString("offer:\n  origin: " + n.origin + "\n")

	if len(n.path) > 0 {
		builder.WriteString("  path:\n")

		for _, entry := range n.path {
			builder.WriteString("    - " + entry + "\n")
		}
	}
}

func drawAmendFlags(rt *rapid.T) cli.AmendArgs {
	return cli.AmendArgs{
		Object:       rapid.SampledFrom([]string{"", "o"}).Draw(rt, "object"),
		Activate:     rapid.Bool().Draw(rt, "activate"),
		ClearPending: rapid.Bool().Draw(rt, "clear-pending"),
		Discard:      rapid.Bool().Draw(rt, "discard"),
	}
}

func drawOfferCommand(rt *rapid.T) cli.OfferCommandForTest {
	return rapid.SampledFrom([]cli.OfferCommandForTest{
		cli.ExportOfferCmdLearn, cli.ExportOfferCmdLearnQA, cli.ExportOfferCmdAmend,
		cli.ExportOfferCmdResituate, cli.ExportOfferCmdIdentityBackfill, cli.ExportOfferCmdRegistration,
	}).Draw(rt, "command")
}

func drawOfferTestNote(rt *rapid.T) offerTestNote {
	note := offerTestNote{
		noteType: rapid.SampledFrom([]string{"fact", "runbook"}).Draw(rt, "type"),
		object:   rapid.SampledFrom([]string{"c", "d", "e"}).Draw(rt, "object"),
	}

	switch rapid.IntRange(0, 2).Draw(rt, "origin") {
	case 1:
		note.origin, note.path = seqID(40)+":"+xidB, []string{seqID(40)}
	case 2:
		note.origin, note.path = parentVaultID+":"+xidB, []string{parentVaultID}
	}

	return note
}

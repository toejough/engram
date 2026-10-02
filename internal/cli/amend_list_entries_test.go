package cli_test

// Amend keeps unknown sub-keys in list entries it keeps (#789 follow-up,
// design D5; spec vault-note-identity "Exchange fields SHALL survive every
// frontmatter rewrite"): when amend replaces supersedes:, an entry that
// survives — matched by its note's basename — keeps the keys the typed
// entry does not define; only new entries are written fresh.

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestRunAmend_SupersedesKeepsUnknownSubKeysOfKeptEntries: A survives
// (re-typed and re-claimed, matched by basename although the old entry
// names it with .md) and keeps x_reason; B is dropped; C is new and
// carries only the modeled keys. An old entry naming no note matches
// nothing and is dropped.
func TestRunAmend_SupersedesKeepsUnknownSubKeysOfKeptEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	note := supersedesNote([]supersedesFixture{
		{note: "5.2026-01-01.a", typ: "narrows", claim: "old a", extra: "      x_reason: kept\n"},
		{note: "6.2026-01-01.b", typ: "updates", claim: "old b", extra: "      x_meta:\n          k: v\n"},
	})

	note = strings.Replace(note, "supersedes:\n", "supersedes:\n    - type: narrows\n      claim: no note named\n", 1)

	vault := newCRLFAmendVault(note)
	err := vault.amend(t.Context(), cli.AmendArgs{Supersedes: []string{
		"5.2026-01-01.a|refutes|new a", "7.2026-01-01.c|narrows|new c",
	}})
	g.Expect(err).NotTo(HaveOccurred())

	entries := decodeSupersedesEntries(t, string(vault.files[crlfAmendNoteName]))
	g.Expect(entries).To(Equal([]map[string]any{
		{"note": "5.2026-01-01.a", "type": "refutes", "claim": "new a", "x_reason": "kept"},
		{"note": "7.2026-01-01.c", "type": "narrows", "claim": "new c"},
	}))
}

// TestRunAmend_SupersedesSubKeysProperty: for any existing supersedes list
// with optional unknown sub-keys and any replacement list, every kept entry
// keeps its unknown sub-keys, every entry carries the new modeled values in
// the new order, and a new entry carries only the modeled keys.
func TestRunAmend_SupersedesSubKeysProperty(t *testing.T) {
	t.Parallel()

	pool := []string{"5.2026-01-01.a", "6.2026-01-01.b", "7.2026-01-01.c", "8.2026-01-01.d"}
	types := []string{"updates", "narrows", "refutes"}

	rapid.Check(t, func(rt *rapid.T) {
		oldNotes := rapid.SliceOfNDistinct(rapid.SampledFrom(pool), 1, len(pool), rapid.ID[string]).Draw(rt, "old")
		existing := make([]supersedesFixture, 0, len(oldNotes))
		extras := map[string]string{}

		for _, name := range oldNotes {
			fixture := supersedesFixture{note: name, typ: rapid.SampledFrom(types).Draw(rt, name+"-type"), claim: "old"}
			if rapid.Bool().Draw(rt, name+"-extra") {
				extras[name+".md"] = rapid.StringMatching(`[a-z]{1,8}`).Draw(rt, name+"-value")
				fixture.extra = "      x_unknown: " + extras[name+".md"] + "\n"
			}

			existing = append(existing, fixture)
		}

		newNotes := rapid.SliceOfNDistinct(rapid.SampledFrom(pool), 1, len(pool), rapid.ID[string]).Draw(rt, "new")
		flags := make([]string, 0, len(newNotes))

		for _, name := range newNotes {
			flags = append(flags, name+"|"+rapid.SampledFrom(types).Draw(rt, name+"-newtype")+"|claim "+name)
		}

		vault := newCRLFAmendVault(supersedesNote(existing))

		err := vault.amend(context.Background(), cli.AmendArgs{Supersedes: flags})
		if err != nil {
			rt.Fatalf("amend: %v", err)
		}

		written := decodeSupersedesEntries(rt, string(vault.files[crlfAmendNoteName]))
		checkSupersedesEntries(rt, written, flags, oldNotes, extras)
	})
}

// supersedesFixture is one existing supersedes entry, with extra lines
// (unknown sub-keys) appended under it.
type supersedesFixture struct {
	note, typ, claim, extra string
}

func checkSupersedesEntries(
	rt *rapid.T, got []map[string]any, flags, oldNotes []string, extras map[string]string,
) {
	if len(got) != len(flags) {
		rt.Fatalf("got %d entries, want %d: %v", len(got), len(flags), got)
	}

	for index, flag := range flags {
		checkSupersedesEntry(rt, got[index], strings.Split(flag, "|"), oldNotes, extras)
	}
}

// checkSupersedesEntry checks one written entry against its flag's parts
// (note, type, claim): the modeled values, and the unknown sub-key kept
// exactly when the entry survived from one that carried it.
func checkSupersedesEntry(
	rt *rapid.T, entry map[string]any, parts, oldNotes []string, extras map[string]string,
) {
	if entry["note"] != parts[0] || entry["type"] != parts[1] || entry["claim"] != parts[2] {
		rt.Fatalf("entry %v, want %v", entry, parts)
	}

	value, hadExtra := extras[parts[0]+".md"]
	if slices.Contains(oldNotes, parts[0]) && hadExtra {
		if entry["x_unknown"] != value {
			rt.Fatalf("entry %s lost x_unknown %q: %v", parts[0], value, entry)
		}

		return
	}

	if len(entry) != len(parts) {
		rt.Fatalf("entry %s carries extra keys: %v", parts[0], entry)
	}
}

func decodeSupersedesEntries(tester failer, content string) []map[string]any {
	tester.Helper()

	decoded := decodeBackfillFrontmatter(tester, content)

	list, ok := decoded["supersedes"].([]any)
	if !ok {
		tester.Fatalf("no supersedes list in %s", content)

		return nil
	}

	entries := make([]map[string]any, 0, len(list))

	for _, item := range list {
		entry, isMap := item.(map[string]any)
		if !isMap {
			tester.Fatalf("supersedes entry %v is not a map", item)

			return nil
		}

		entries = append(entries, entry)
	}

	return entries
}

// supersedesNote is the minimal fact note with the given supersedes list.
func supersedesNote(entries []supersedesFixture) string {
	var block strings.Builder

	block.WriteString("supersedes:\n")

	for _, entry := range entries {
		fmt.Fprintf(&block, "    - note: %s.md\n      type: %s\n      claim: %s\n%s", entry.note, entry.typ, entry.claim,
			entry.extra)
	}

	return strings.Replace(amendParityMinimalNote("fact"), "source: test\n", "source: test\n"+block.String(), 1)
}

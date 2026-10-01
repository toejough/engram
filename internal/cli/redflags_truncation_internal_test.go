package cli

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/vaultgraph"
)

// TestCapRedFlagsForPreview_KeepsNewestEntryWhenOverBudget reproduces
// engram#763: a red_flags list whose combined size exceeds the preview
// budget silently dropped its most-recently-appended (last) entry under
// Claude Code's own ~2KB Bash-tool-output truncation. Entries are appended
// by callers (engram learn runbook / amend --red-flag), so the newest entry
// is conventionally last — capRedFlagsForPreview must keep entries from the
// END of the list, not the start, and must leave an explicit in-band marker
// naming the omission rather than silently dropping the rest.
func TestCapRedFlagsForPreview_KeepsNewestEntryWhenOverBudget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var b strings.Builder

	b.WriteString("---\ntype: runbook\nsituation: x\ndone_when: y\nred_flags:\n")

	for range 20 {
		b.WriteString("    - " + strings.Repeat("filler ", 20) + "\n")
	}

	b.WriteString("    - the newest entry added to fix a specific defect\n")
	b.WriteString("luhmann: \"1\"\n---\n\nbody\n")

	content := b.String()

	got := capRedFlagsForPreview(content, "1.query-preview-fixture")

	g.Expect(len(got)).To(BeNumerically("<", len(content)),
		"expected the oversized red_flags block to shrink")
	g.Expect(got).To(ContainSubstring("the newest entry added to fix a specific defect"),
		"the most-recently-appended red_flags entry must survive truncation")
	g.Expect(got).To(ContainSubstring("EARLIER RED_FLAGS OMITTED"),
		"the omission must be visible in-band, not silent")
	g.Expect(got).To(HaveSuffix("luhmann: \"1\"\n---\n\nbody\n"),
		"content after the red_flags block must be untouched")
	g.Expect(got).To(HavePrefix("---\ntype: runbook\nsituation: x\ndone_when: y\nred_flags:\n"),
		"content before the red_flags block must be untouched")
}

// TestCapRedFlagsForPreview_MarkerCommandReturnsFullListViaShow is the guard
// test D1 requires: an agent reading only the query preview's marker must be
// able to parse out `engram show <basename>`, run it, and get back every
// entry — including the ones the preview itself omitted. engram show never
// truncates (#772), so the full original note (not the capped preview) is
// what the vault actually holds and what RunShow must return in full.
func TestCapRedFlagsForPreview_MarkerCommandReturnsFullListViaShow(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		basename   = "1.guard-fixture"
		totalCount = 15
	)

	entries := fixedWidthRedFlagEntries(totalCount)
	fullContent := "---\ntype: runbook\nsituation: x\ndone_when: y\nred_flags:\n" +
		strings.Join(entries, "") + "luhmann: \"1\"\n---\n\nbody\n"

	preview := capRedFlagsForPreview(fullContent, basename)

	commandPattern := regexp.MustCompile(`run engram show (\S+) for all`)

	match := commandPattern.FindStringSubmatch(preview)
	g.Expect(match).To(HaveLen(2), "marker must contain a parseable `engram show <basename>` command")

	if match == nil {
		return
	}

	parsedBasename := match[1]
	g.Expect(parsedBasename).To(Equal(basename), "the parsed basename must be the note's real basename")

	deps := ShowDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: parsedBasename}}, nil
		},
		// The vault always holds the FULL, untruncated note — only the
		// query preview is capped.
		Read: func(string) ([]byte, error) { return []byte(fullContent), nil },
	}

	var out bytes.Buffer

	err := RunShow(context.Background(), ShowArgs{Ref: parsedBasename, VaultPath: "/vault"}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	for _, entry := range entries {
		g.Expect(out.String()).To(ContainSubstring(strings.TrimSpace(entry)),
			"engram show must return every entry, including ones the query preview omitted")
	}
}

// TestCapRedFlagsForPreview_MarkerNamesCountTotalAndRealBasename pins #772's
// fix: the omission marker is no longer the fixed, countless, literal
// "<basename>" placeholder — it names how many entries were dropped, the
// list's true total, and the note's REAL basename, so the command it prints
// actually resolves (D1, recall-runbook-surfacing "Oversized red_flags list
// keeps its newest entry in query").
//
// The fixture uses 15 entries of a fixed, known byte width so the expected
// drop count is a hand-computed oracle, not a re-derivation of the function
// under test: at 100 bytes/entry and the 1200-byte budget, the marker (81
// bytes for this basename/total) leaves room for exactly 11 entries, so 4
// are dropped.
func TestCapRedFlagsForPreview_MarkerNamesCountTotalAndRealBasename(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		basename   = "1.test-note"
		totalCount = 15
		wantDrop   = 4
		wantKeep   = totalCount - wantDrop
	)

	entries := fixedWidthRedFlagEntries(totalCount)
	content := "---\ntype: runbook\nsituation: x\ndone_when: y\nred_flags:\n" +
		strings.Join(entries, "") + "luhmann: \"1\"\n---\n\nbody\n"

	got := capRedFlagsForPreview(content, basename)

	wantMarker := fmt.Sprintf(
		"    - \"[%d EARLIER RED_FLAGS OMITTED — run engram show %s for all %d]\"\n",
		wantDrop, basename, totalCount,
	)

	g.Expect(got).To(ContainSubstring(wantMarker), "marker must name the drop count, basename and total exactly")
	g.Expect(got).To(ContainSubstring(entries[totalCount-1]), "the newest entry must survive")

	for _, dropped := range entries[:wantDrop] {
		g.Expect(got).NotTo(ContainSubstring(dropped), "a dropped entry must not still appear in the preview")
	}

	for _, kept := range entries[totalCount-wantKeep:] {
		g.Expect(got).To(ContainSubstring(kept), "every kept entry must survive")
	}
}

// TestCapRedFlagsForPreview_NoRedFlagsBlockUnchanged proves a note without a
// red_flags field at all (fact/feedback kinds, or a runbook with none set)
// passes through unmodified.
func TestCapRedFlagsForPreview_NoRedFlagsBlockUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\nsituation: x\n---\n\nbody\n"

	g.Expect(capRedFlagsForPreview(content, "1.no-red-flags")).To(Equal(content))
}

// TestCapRedFlagsForPreview_UnderBudgetUnchanged proves a red_flags block
// already within the preview budget is returned byte-identical — this
// function must not touch notes that were never at truncation risk.
func TestCapRedFlagsForPreview_UnderBudgetUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: runbook\nsituation: x\ndone_when: y\n" +
		"red_flags:\n    - short one\n    - short two\n" +
		"luhmann: \"1\"\n---\n\nbody\n"

	g.Expect(capRedFlagsForPreview(content, "1.under-budget")).To(Equal(content))
}

// TestProperty_CapRedFlagsForPreview_SizeAndContiguity: for any red_flags
// list, capRedFlagsForPreview either leaves content byte-identical (when the
// list already fits the budget) or produces a red_flags block — marker plus
// kept entries — that is at most redFlagsPreviewBudget bytes and is a
// CONTIGUOUS run of original entries ending at the newest (last) one,
// preceded by the exact marker naming that run's drop count.
func TestProperty_CapRedFlagsForPreview_SizeAndContiguity(t *testing.T) {
	t.Parallel()

	const (
		header = "---\ntype: runbook\nsituation: x\ndone_when: y\nred_flags:\n"
		footer = "luhmann: \"1\"\n---\n\nbody\n"
		maxN   = 25
	)

	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(0, maxN).Draw(rt, "n")
		basename := rapid.StringMatching(`[0-9]\.[a-z][a-z-]{0,20}`).Draw(rt, "basename")

		entries := make([]string, n)
		consumed := 0

		for i := range n {
			text := rapid.StringMatching(`[a-zA-Z0-9 ]{1,120}`).Draw(rt, fmt.Sprintf("entry%d", i))
			entries[i] = "    - " + text + "\n"
			consumed += len(entries[i])
		}

		content := header + strings.Join(entries, "") + footer

		got := capRedFlagsForPreview(content, basename)

		if consumed <= redFlagsPreviewBudget {
			if got != content {
				rt.Fatalf("under-budget input must round-trip byte-identical:\nwant %q\ngot  %q", content, got)
			}

			return
		}

		listPart := got[len(header) : len(got)-len(footer)]

		if len(listPart) > redFlagsPreviewBudget {
			rt.Fatalf("over-budget red_flags block exceeds the preview budget: %d > %d bytes",
				len(listPart), redFlagsPreviewBudget)
		}

		if !contiguousSuffixWithMarker(listPart, entries, basename) {
			rt.Fatalf("over-budget result is not <exact marker>+<contiguous run of entries ending at newest>: %q",
				listPart)
		}
	})
}

// contiguousSuffixWithMarker reports whether listPart is exactly
// redFlagsOmittedMarker(dropped, len(entries), basename) followed by
// entries[dropped:], for some dropped in [0, len(entries)].
func contiguousSuffixWithMarker(listPart string, entries []string, basename string) bool {
	total := len(entries)

	for dropped := range total + 1 {
		want := redFlagsOmittedMarker(dropped, total, basename) + strings.Join(entries[dropped:], "")
		if listPart == want {
			return true
		}
	}

	return false
}

// fixedWidthRedFlagEntries returns n distinguishable red_flags list-item
// lines, each exactly 100 bytes (6-byte "    - " prefix + a unique 2-digit
// "entry-NN-" tag + filler + trailing newline), so callers can hand-compute
// expected drop counts against the budget instead of re-deriving the
// function under test. n must be <= 99 (2-digit tags).
func fixedWidthRedFlagEntries(n int) []string {
	const lineWidth = 100

	entries := make([]string, n)

	for i := range n {
		prefix := fmt.Sprintf("    - entry-%02d-", i+1)
		pad := lineWidth - len(prefix) - 1 // -1 for the trailing newline
		entries[i] = prefix + strings.Repeat("x", pad) + "\n"
	}

	return entries
}

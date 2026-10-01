package cli_test

// Amend keeps frontmatter keys the typed note model does not define, and
// anchors on keys it does not edit (ruling V6, final review F4;
// vault-note-identity "Exchange fields SHALL survive every frontmatter
// rewrite").

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestRunAmend_KeepsAnchorOnUneditedKey: an anchor on a key amend does not
// edit (source:), aliased by an unknown key, survives a content amend.
func TestRunAmend_KeepsAnchorOnUneditedKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := strings.Replace(amendParityMinimalNote("fact"), "source: test\n", "source: &src test\nx_src: *src\n", 1)

	after, err := runAmendParityCase(t.Context(), amendParityCase{input: input, args: cli.AmendArgs{Object: "new"}})
	g.Expect(err).NotTo(HaveOccurred())

	fields := frontmatterOf(after)
	g.Expect(fields["object"]).To(Equal("new"))
	g.Expect(fields["source"]).To(Equal("test"))
	g.Expect(fields["x_src"]).To(Equal("test"))
}

// TestRunAmend_KeepsUnknownKeysAndAnchorsProperty: for every amend kind on
// every note type, 0-3 unknown keys (scalar, list, nested map) and an
// optional anchor on an unedited key survive with their decoded values; an
// anchor on the edited key refuses the amend unwritten.
func TestRunAmend_KeepsUnknownKeysAndAnchorsProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		tc := rapid.SampledFrom(amendKindCases()).Draw(rt, "case")
		unknown := skillUnknownKeysGen().Draw(rt, "unknown")

		if rapid.Bool().Draw(rt, "anchoredUnedited") {
			tc.input = strings.Replace(tc.input, "source: test\n", "source: &src test\n", 1)
			unknown += "x_src: *src\n"
		}

		tc.input = withFrontmatterLines(tc.input, unknown)

		editedAnchor := tc.edited != "" && rapid.Bool().Draw(rt, "anchoredEdited")
		if editedAnchor {
			tc.input = anchorKey(tc.input, tc.edited)
		}

		after, err := runAmendParityCase(context.Background(), tc.amendParityCase)

		if editedAnchor {
			if !errors.Is(err, cli.ErrFrontmatterAnchoredKeyForTest) || after != "" {
				rt.Fatalf("%s with an anchored %s: err = %v, wrote %d bytes", tc.name, tc.edited, err, len(after))
			}

			return
		}

		if err != nil {
			rt.Fatalf("%s: %v", tc.name, err)
		}

		assertUneditedKeysKept(rt, tc.name, tc.input, after)
	})
}

// TestRunAmend_KeepsUnmodeledKeys: every amend kind on a fact, feedback and
// runbook note keeps `luhmann_old: "12"` and a nested unknown map.
func TestRunAmend_KeepsUnmodeledKeys(t *testing.T) {
	t.Parallel()

	const unknown = "luhmann_old: \"12\"\nprovenance:\n    origin: import\n    steps: [a, b]\n"

	for _, tc := range amendKindCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			tc.input = withFrontmatterLines(tc.input, unknown)

			after, err := runAmendParityCase(t.Context(), tc.amendParityCase)
			g.Expect(err).NotTo(HaveOccurred())

			fields := frontmatterOf(after)
			g.Expect(fields["luhmann_old"]).To(Equal("12"))
			g.Expect(fields["provenance"]).To(Equal(map[string]any{"origin": "import", "steps": []any{"a", "b"}}))
		})
	}
}

// TestRunAmend_RefusesAnchoredEditedKey: an anchor on the key a content
// amend replaces refuses the amend unwritten (ruling V3).
func TestRunAmend_RefusesAnchoredEditedKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := strings.Replace(amendParityMinimalNote("fact"), "object: old\n", "object: &o old\nx_obj: *o\n", 1)

	after, err := runAmendParityCase(t.Context(), amendParityCase{input: input, args: cli.AmendArgs{Object: "new"}})
	g.Expect(err).To(MatchError(cli.ErrFrontmatterAnchoredKeyForTest))
	g.Expect(err).To(MatchError(ContainSubstring("object")))
	g.Expect(after).To(BeEmpty(), "the note must not be written")
}

// amendKindCase is an amend parity case plus the key it edits.
type amendKindCase struct {
	amendParityCase

	edited string
}

// amendKindCases is every amend kind that writes frontmatter, on the
// minimal fact, feedback and runbook notes; edited names the frontmatter
// key the amend replaces ("" for an amend that only appends or clears).
func amendKindCases() []amendKindCase {
	notPending := false
	pending := "pending: true\n"

	return []amendKindCase{
		{amendParityCase{"fact-object", amendParityMinimalNote("fact"), cli.AmendArgs{Object: "new"}}, "object"},
		{amendParityCase{"fact-supersedes", amendParityMinimalNote("fact"),
			cli.AmendArgs{Supersedes: []string{"5.2026-01-01.x|narrows|old"}}}, ""},
		{amendParityCase{"fact-chunk-source", amendParityMinimalNote("fact"),
			cli.AmendArgs{ChunkSources: []string{"s.jsonl#a1"}}}, ""},
		{amendParityCase{"fact-clear-pending", withFrontmatterLines(amendParityMinimalNote("fact"), pending),
			cli.AmendArgs{Pending: &notPending}}, ""},
		{amendParityCase{"feedback-impact", amendParityMinimalNote("feedback"), cli.AmendArgs{Impact: "new"}}, "impact"},
		{amendParityCase{"feedback-chunk-source", amendParityMinimalNote("feedback"),
			cli.AmendArgs{ChunkSources: []string{"s.jsonl#a1"}}}, ""},
		{amendParityCase{"runbook-done-when", amendParityMinimalNote("runbook"),
			cli.AmendArgs{DoneWhen: "new"}}, "done_when"},
		{amendParityCase{"runbook-red-flags", amendParityMinimalNote("runbook"),
			cli.AmendArgs{RedFlags: []string{"x", "y"}}}, "red_flags"},
		{amendParityCase{"runbook-supersedes", amendParityMinimalNote("runbook"),
			cli.AmendArgs{Supersedes: []string{"5.2026-01-01.x|narrows|old"}}}, ""},
	}
}

// anchorKey puts anchor &e on key's value in content's frontmatter and adds
// another key aliasing it.
func anchorKey(content, key string) string {
	content = strings.Replace(content, "\n"+key+": ", "\n"+key+": &e ", 1)
	content = strings.Replace(content, "\n"+key+":\n", "\n"+key+": &e\n", 1)

	return withFrontmatterLines(content, "x_edit_ref: *e\n")
}

// assertUneditedKeysKept checks every unknown (x_) key, and source:, decodes
// in after exactly as in before.
func assertUneditedKeysKept(rt *rapid.T, name, before, after string) {
	got := frontmatterOf(after)

	for key, value := range frontmatterOf(before) {
		if !strings.HasPrefix(key, "x_") && key != "source" {
			continue
		}

		if !reflect.DeepEqual(got[key], value) {
			rt.Fatalf("%s changed %s: %v -> %v\n%s", name, key, value, got[key], after)
		}
	}
}

// withFrontmatterLines appends lines to content's frontmatter.
func withFrontmatterLines(content, lines string) string {
	return strings.Replace(content, "\n---\n", "\n"+lines+"---\n", 1)
}

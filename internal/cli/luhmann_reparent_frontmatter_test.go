package cli_test

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestRenameAndRewriteReferences_FrontmatterWikilinksRewritten asserts a
// [[old-basename]] wikilink inside ANY frontmatter string field — a scalar
// (action:, situation:, object:, ...) or a list item (red_flags:, done_when:)
// in any YAML quoting style — is rewritten to the new basename, the frontmatter
// stays valid YAML with only the link changed, and the referrer is reported
// for a sidecar rebuild.
func TestRenameAndRewriteReferences_FrontmatterWikilinksRewritten(t *testing.T) {
	t.Parallel()

	const (
		oldBasename  = "9a.2026-01-01.old-topic"
		newBasename  = "9b1.2026-01-01.old-topic"
		referrerName = "9c.2026-01-01.referrer.md"
	)

	fields := []string{"action", "impact", "behavior", "object", "situation", "subject", "red_flags", "done_when"}
	styles := []string{"plain", "single", "double"}

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		field := rapid.SampledFrom(fields).Draw(rt, "field")
		style := rapid.SampledFrom(styles).Draw(rt, "style")
		asList := rapid.Bool().Draw(rt, "asList")
		legacyMD := rapid.Bool().Draw(rt, "legacyMD")
		prefix := rapid.SampledFrom([]string{"see", "follow the rule in", "per note"}).Draw(rt, "prefix")

		target := oldBasename
		if legacyMD {
			target += ".md"
		}

		value := prefix + " [[" + target + "]] then act"
		wantValue := prefix + " [[" + newBasename + "]] then act"

		rendered := renderYAMLScalar(style, value)

		entry := field + ": " + rendered + "\n"
		if asList {
			entry = field + ":\n    - " + rendered + "\n"
		}

		referrerBody := "---\ntype: feedback\n" + entry + "luhmann: \"9c\"\n---\n\nNo body links.\n"

		fixture := newReparentFixture(map[string]string{referrerName: referrerBody})

		rewritten, err := cli.RenameAndRewriteReferences(
			fixture.deps(), "/vault", map[string]string{oldBasename: newBasename})
		g.Expect(err).NotTo(HaveOccurred())

		updated := string(fixture.written["/vault/"+referrerName])
		g.Expect(updated).To(ContainSubstring("[[" + newBasename + "]]"))
		g.Expect(updated).NotTo(ContainSubstring("[[" + oldBasename))
		g.Expect(rewritten).To(ConsistOf("/vault/" + referrerName))

		g.Expect(frontmatterFieldValue(rt, updated, field, asList)).To(Equal(wantValue))
	})
}

// TestRenameAndRewriteReferences_LuhmannValueAnyShapeRewritten asserts the
// renamed note's own luhmann: field is rewritten whatever shape its value
// takes — plain, quoted, a trailing comment, a plain value on a continuation
// line, a block scalar, or a multi-line quoted scalar: the key's entire value
// (continuation lines included) becomes one `luhmann: "<new id>"` line, the
// frontmatter still decodes, and every neighboring key keeps its value.
func TestRenameAndRewriteReferences_LuhmannValueAnyShapeRewritten(t *testing.T) {
	t.Parallel()

	const (
		oldBasename = "9a.2026-01-01.note"
		newBasename = "9b1.2026-01-01.note"
	)

	shapes := []string{
		"luhmann: 9a\n",
		"luhmann: \"9a\"\n",
		"luhmann: '9a'\n",
		"luhmann: \"9a\" # moved\n",
		"luhmann:\n  9a\n",
		"luhmann:\n    9a\n\n",
		"luhmann: >-\n  9a\n",
		"luhmann: |\n  9a\n",
		"luhmann: |-\n  9\n\n  a\n",
		"luhmann: >\n    9a\n    more\n",
		"luhmann: \"9\n  a\"\n",
		"luhmann: \"9\\\"\n  a\"\n",
		"luhmann: '9\n  a'\n",
		"luhmann: '9''\n  a'\n",
	}

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		shape := rapid.SampledFrom(shapes).Draw(rt, "shape")
		last := rapid.Bool().Draw(rt, "luhmannLast")

		frontmatter := "type: fact\n" + shape + "situation: s\ntags:\n    - a\n"
		if last {
			frontmatter = "type: fact\nsituation: s\ntags:\n    - a\n" + shape
		}

		precondition := map[string]any{}
		g.Expect(yaml.Unmarshal([]byte(frontmatter), &precondition)).To(Succeed(), "fixture must decode")

		fixture := newReparentFixture(map[string]string{
			oldBasename + ".md": "---\n" + frontmatter + "---\n\nBody.\n",
		})

		_, err := cli.RenameAndRewriteReferences(
			fixture.deps(), "/vault", map[string]string{oldBasename: newBasename})
		g.Expect(err).NotTo(HaveOccurred())

		updated := string(fixture.written["/vault/"+newBasename+".md"])
		g.Expect(updated).To(ContainSubstring("luhmann: \"9b1\"\n"))
		g.Expect(updated).To(HaveSuffix("---\n\nBody.\n"))

		g.Expect(frontmatterFieldValue(rt, updated, "luhmann", false)).To(Equal("9b1"))
		g.Expect(frontmatterFieldValue(rt, updated, "type", false)).To(Equal("fact"))
		g.Expect(frontmatterFieldValue(rt, updated, "situation", false)).To(Equal("s"))
		g.Expect(frontmatterFieldValue(rt, updated, "tags", true)).To(Equal("a"))
	})
}

// frontmatterFieldValue parses content's frontmatter as YAML and returns the
// named field's string value (or its single list item's value).
func frontmatterFieldValue(rt *rapid.T, content, field string, asList bool) string {
	rest, found := strings.CutPrefix(content, "---\n")
	if !found {
		rt.Fatalf("no frontmatter in %q", content)
	}

	frontmatter, _, found := strings.Cut(rest, "\n---\n")
	if !found {
		rt.Fatalf("unterminated frontmatter in %q", content)
	}

	parsed := map[string]any{}

	err := yaml.Unmarshal([]byte(frontmatter), &parsed)
	if err != nil {
		rt.Fatalf("frontmatter no longer valid YAML: %v\n%s", err, frontmatter)
	}

	if !asList {
		return fmt.Sprint(parsed[field])
	}

	items, ok := parsed[field].([]any)
	if !ok || len(items) != 1 {
		rt.Fatalf("field %s not a one-item list: %#v", field, parsed[field])
	}

	return fmt.Sprint(items[0])
}

// renderYAMLScalar renders value as a YAML scalar in the given quoting style.
func renderYAMLScalar(style, value string) string {
	switch style {
	case "single":
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	case "double":
		return fmt.Sprintf("%q", value)
	default:
		return value
	}
}

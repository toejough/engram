package cli_test

// Tests for the exchange hash (design D3, vault-parent-offers "The exchange
// hash SHALL cover every offered content field"; task 3.1).

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestExchangeHash_BodyLineEndingsDoNotMoveTheHashProperty (ruling S6): the
// canonical body normalizes CRLF to LF and ends in exactly one newline, so the
// LF, CRLF, and missing-final-newline spellings of the same body — a CRLF
// body's blank separator line included — hash equally on both sides of an
// exchange.
func TestExchangeHash_BodyLineEndingsDoNotMoveTheHashProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		note := exchangeHashNoteGen().Draw(rt, "note")
		lines := rapid.SliceOfN(exchangeBodyGen(), 1, 4).Draw(rt, "lines")
		frontmatter, _ := yaml.Marshal(note.fields)
		head := "---\n" + string(frontmatter) + "---\n"
		// The blank separator line after the closing delimiter is part of
		// the body, so a CRLF body may carry it as CRLF too (ruling V2).
		separator := rapid.SampledFrom([]string{"\n", "\r\n"}).Draw(rt, "separator")

		lf := strings.Join(lines, "\n")
		crlf := strings.Join(lines, "\r\n")
		want := mustExchangeHash(rt, head+"\n"+lf+"\n")

		for name, body := range map[string]string{
			"LF, no final newline":   "\n" + lf,
			"CRLF":                   separator + crlf + "\r\n",
			"CRLF, no final newline": separator + crlf,
		} {
			if got := mustExchangeHash(rt, head+body); got != want {
				rt.Fatalf("%s body hashed %s, LF body hashed %s", name, got, want)
			}
		}
	})
}

// TestExchangeHash_ClassificationCoversEveryFrontmatterKey is the r3-5
// reflection test: every YAML key of the fact, feedback and runbook
// frontmatter structs is classified as offered or not offered, and the table
// names no key that no struct has.
func TestExchangeHash_ClassificationCoversEveryFrontmatterKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	classification := cli.ExportExchangeKeyClassification()
	structKeys := map[string]bool{}

	for _, docType := range cli.ExportFrontmatterDocTypes() {
		for _, key := range yamlKeysOf(docType) {
			structKeys[key] = true
			_, classified := classification[key]
			g.Expect(classified).To(BeTrue(),
				"%s key %q is not classified as offered or not offered (exchangehash.go)", docType.Name(), key)
		}
	}

	for key := range classification {
		g.Expect(structKeys).To(HaveKey(key), "classified key %q is on no frontmatter struct", key)
	}
}

// TestExchangeHash_ClassifiesTheDesignFields pins design D3's split: the
// offered keys are exactly the hashed content fields, and the exchange's own
// bookkeeping keys are not offered.
func TestExchangeHash_ClassifiesTheDesignFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	classification := cli.ExportExchangeKeyClassification()

	offered := make([]string, 0, len(classification))

	for key, isOffered := range classification {
		if isOffered {
			offered = append(offered, key)
		}
	}

	g.Expect(offered).To(ConsistOf(exchangeOfferedKeys))

	for _, key := range []string{
		"repo", "user", "vault", "pending", "tags", "sources", "supersedes",
		"xid", "parent", "aliases", "offer", "skill_hash", "skill_key", "skill_source",
	} {
		g.Expect(classification).To(HaveKeyWithValue(key, false), key+" must be classified not offered")
	}
}

// TestExchangeHash_CompareIsThreeWay covers the versioned three-way
// comparison: equal and changed only when both sides carry the current
// version prefix; unknown whenever either side's prefix is different or
// missing.
func TestExchangeHash_CompareIsThreeWay(t *testing.T) {
	t.Parallel()

	current := cli.ExportExchangeHashPrefix + strings.Repeat("ab", 32)
	other := cli.ExportExchangeHashPrefix + strings.Repeat("cd", 32)

	cases := map[string]struct {
		a, b string
		want cli.ExportHashComparison
	}{
		"equal":                {current, current, cli.ExportHashEqual},
		"changed":              {current, other, cli.ExportHashChanged},
		"older version":        {"xh0:" + strings.Repeat("ab", 32), current, cli.ExportHashUnknown},
		"newer version":        {current, "xh2:" + strings.Repeat("ab", 32), cli.ExportHashUnknown},
		"missing prefix":       {strings.Repeat("ab", 32), current, cli.ExportHashUnknown},
		"sidecar content hash": {"sha256:" + strings.Repeat("ab", 32), current, cli.ExportHashUnknown},
		"empty side":           {"", current, cli.ExportHashUnknown},
		"both empty":           {"", "", cli.ExportHashUnknown},
		"same old version":     {"xh0:aa", "xh0:aa", cli.ExportHashUnknown},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(cli.ExportCompareExchangeHashes(tc.a, tc.b)).To(Equal(tc.want))
			g.Expect(cli.ExportCompareExchangeHashes(tc.b, tc.a)).To(Equal(tc.want), "comparison is symmetric")
		})
	}
}

// TestExchangeHash_GoldenMatchesCanonicalJSON is the golden test: the hash
// of a fixed note file is "xh1:" + sha256 of the canonical (sorted-key) JSON
// of every offered field plus the body text, with absent fields as empty
// strings or lists. The server and the child both call this one function, so
// pinning its output pins what both sides compute for the same file.
func TestExchangeHash_GoldenMatchesCanonicalJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	note := "---\n" +
		"type: feedback\n" +
		"situation: releasing a widget\n" +
		"behavior: skipped the checklist\n" +
		"impact: the release broke\n" +
		"action: run the checklist\n" +
		"luhmann: \"12a\"\n" +
		"created: \"2026-09-27\"\n" +
		"source: test\n" +
		"user: alice@example.com\n" +
		"vault: personal\n" +
		"tags:\n    - vocab/release\n" +
		"---\n\n" +
		"Lesson learned: when releasing a widget, run the checklist.\n\n" +
		"Supersedes: [[9.2026-01-01.old]] — narrows: older claim\n"

	canonical := `{"action":"run the checklist","behavior":"skipped the checklist",` +
		`"body":"Lesson learned: when releasing a widget, run the checklist.\n",` +
		`"done_when":"","impact":"the release broke","object":"","predicate":"","red_flags":[],` +
		`"situation":"releasing a widget","subject":"","triggers":[],"type":"feedback"}`
	sum := sha256.Sum256([]byte(canonical))

	got, err := cli.ExportExchangeHash([]byte(note))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(Equal("xh1:" + hex.EncodeToString(sum[:])))
}

// TestExchangeHash_NonOfferedFieldsDoNotMoveTheHashProperty: changing any
// one non-offered frontmatter key, on any note type, leaves the hash
// unchanged.
func TestExchangeHash_NonOfferedFieldsDoNotMoveTheHashProperty(t *testing.T) {
	t.Parallel()

	notOffered := exchangeClassifiedKeys(false)

	rapid.Check(t, func(rt *rapid.T) {
		note := exchangeHashNoteGen().Draw(rt, "note")
		key := rapid.SampledFrom(notOffered).Draw(rt, "key")
		mutated := note.with(key, exchangeKeyValueGen(key).Filter(func(value any) bool {
			return !sameFrontmatterValue(value, note.fields[key])
		}).Draw(rt, "value"))

		before := mustExchangeHash(rt, note.render())
		if after := mustExchangeHash(rt, mutated.render()); after != before {
			rt.Fatalf("changing non-offered %q moved the hash", key)
		}
	})
}

// TestExchangeHash_OfferedFieldsMoveTheHashProperty: changing any one
// offered field (frontmatter or body), on any note type, changes the hash.
func TestExchangeHash_OfferedFieldsMoveTheHashProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		note := exchangeHashNoteGen().Draw(rt, "note")
		key := rapid.SampledFrom(exchangeOfferedMutableKeys).Draw(rt, "key")

		var mutated exchangeHashNote

		if key == "body" {
			mutated = note
			mutated.body = exchangeBodyGen().Filter(func(body string) bool { return body != note.body }).Draw(rt, "body")
		} else {
			mutated = note.with(key, exchangeKeyValueGen(key).Filter(func(value any) bool {
				return !sameFrontmatterValue(value, note.fields[key])
			}).Draw(rt, "value"))
		}

		before := mustExchangeHash(rt, note.render())
		if after := mustExchangeHash(rt, mutated.render()); after == before {
			rt.Fatalf("changing offered %q left the hash unchanged", key)
		}
	})
}

// TestExchangeHash_OfferedFieldsOfEachTypeMoveTheHash is the scenario
// "Every offered field moves the hash": feedback impact, and runbook
// done_when, red_flags and triggers.
func TestExchangeHash_OfferedFieldsOfEachTypeMoveTheHash(t *testing.T) {
	t.Parallel()

	base := map[string]exchangeHashNote{
		"feedback": {fields: map[string]any{
			"type": "feedback", "situation": "s", "behavior": "b", "impact": "i", "action": "a",
		}, body: "Lesson learned: when s, a."},
		"runbook": {fields: map[string]any{
			"type": "runbook", "situation": "s", "done_when": "d",
			"red_flags": []string{"r"}, "triggers": []string{"t"},
		}, body: "1. step"},
	}

	cases := []struct {
		noteType, key string
		value         any
	}{
		{"feedback", "impact", "a different impact"},
		{"runbook", "done_when", "a different done_when"},
		{"runbook", "red_flags", []string{"r", "another"}},
		{"runbook", "triggers", []string{"another"}},
	}

	for _, tc := range cases {
		t.Run(tc.noteType+"/"+tc.key, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			note := base[tc.noteType]

			before, err := cli.ExportExchangeHash([]byte(note.render()))
			g.Expect(err).NotTo(HaveOccurred())

			after, err := cli.ExportExchangeHash([]byte(note.with(tc.key, tc.value).render()))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(after).NotTo(Equal(before))
		})
	}
}

// TestExchangeHash_RejectsUnparseableFrontmatter: a note whose frontmatter is
// not YAML has no exchange hash.
func TestExchangeHash_RejectsUnparseableFrontmatter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := cli.ExportExchangeHash([]byte("---\nsituation: [unclosed\n---\n\nbody\n"))
	g.Expect(err).To(HaveOccurred())
}

// TestExchangeHash_UnknownPerConsumer covers what each exchange consumer does
// with an unknown comparison (design D3): loop suppression and dedupe rule 2
// fire only on equal; the pull-down skip, decline match, rejected-entry
// re-arm and send/apply change check treat unknown as not changed.
func TestExchangeHash_UnknownPerConsumer(t *testing.T) {
	t.Parallel()

	current := cli.ExportExchangeHashPrefix + strings.Repeat("ab", 32)
	other := cli.ExportExchangeHashPrefix + strings.Repeat("cd", 32)
	stale := "xh0:" + strings.Repeat("ab", 32)

	type outcome struct{ equal, changed, unknown bool }

	consumers := map[string]struct {
		decide func(recorded, current string) bool
		want   outcome
	}{
		"loop suppression fires":        {cli.ExportExchangeHashesMatch, outcome{true, false, false}},
		"dedupe rule 2 matches":         {cli.ExportExchangeHashesMatch, outcome{true, false, false}},
		"pull-down re-pulls":            {cli.ExportExchangeHashChanged, outcome{false, true, false}},
		"decline no longer matches":     {cli.ExportExchangeHashChanged, outcome{false, true, false}},
		"rejected entry re-arms":        {cli.ExportExchangeHashChanged, outcome{false, true, false}},
		"send/apply sees a change":      {cli.ExportExchangeHashChanged, outcome{false, true, false}},
		"dedupe ignores missing hashes": {cli.ExportExchangeHashesMatch, outcome{true, false, false}},
	}

	for name, consumer := range consumers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(consumer.decide(current, current)).To(Equal(consumer.want.equal), "on equal")
			g.Expect(consumer.decide(current, other)).To(Equal(consumer.want.changed), "on changed")
			g.Expect(consumer.decide(stale, current)).To(Equal(consumer.want.unknown), "on unknown (version)")
			g.Expect(consumer.decide("", "")).To(Equal(consumer.want.unknown), "on unknown (missing)")
		})
	}
}

// TestExchangeHash_UnterminatedFrontmatterIsWholeTextBody pins the hash of a
// note whose leading "---" block never closes: it is not an error, and the
// whole text is the body with every offered frontmatter field empty. This is
// the reading every other engram path already takes (embed.SplitFrontmatter
// finds no frontmatter, so ExtractBody, ContentHash and show all treat the
// text as body), and an error would let one malformed note stall an outbox
// drain or a dedupe pass. Two such notes still hash differently whenever
// their text differs.
func TestExchangeHash_UnterminatedFrontmatterIsWholeTextBody(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	note := "---\ntype: fact\nsituation: never closed\n\nbody text\n"

	canonical := `{"action":"","behavior":"",` +
		`"body":"---\ntype: fact\nsituation: never closed\n\nbody text\n",` +
		`"done_when":"","impact":"","object":"","predicate":"","red_flags":[],` +
		`"situation":"","subject":"","triggers":[],"type":""}`
	sum := sha256.Sum256([]byte(canonical))

	got, err := cli.ExportExchangeHash([]byte(note))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(Equal("xh1:" + hex.EncodeToString(sum[:])))
}

// unexported variables.
var (
	exchangeOfferedKeys = []string{
		"type", "situation", "subject", "predicate", "object", "behavior", "impact", "action",
		"done_when", "red_flags", "triggers",
	}
	// exchangeOfferedMutableKeys is every offered field a property may
	// change: the offered frontmatter keys plus the body.
	exchangeOfferedMutableKeys = append(slices.Clone(exchangeOfferedKeys), "body")
)

// exchangeHashNote is a note under construction: frontmatter key values and
// a body.
type exchangeHashNote struct {
	fields map[string]any
	body   string
}

func (n exchangeHashNote) render() string {
	frontmatter, _ := yaml.Marshal(n.fields)

	return "---\n" + string(frontmatter) + "---\n\n" + n.body + "\n"
}

func (n exchangeHashNote) with(key string, value any) exchangeHashNote {
	fields := make(map[string]any, len(n.fields)+1)
	maps.Copy(fields, n.fields)
	fields[key] = value

	return exchangeHashNote{fields: fields, body: n.body}
}

// exchangeBodyGen draws body text with no machine-written channel lines and
// no leading or trailing whitespace.
func exchangeBodyGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z0-9][a-zA-Z0-9 .]{0,30}[a-zA-Z0-9.]`)
}

// exchangeClassifiedKeys returns the classified keys with the given
// offered-ness, sorted.
func exchangeClassifiedKeys(offered bool) []string {
	keys := []string{}

	for key, isOffered := range cli.ExportExchangeKeyClassification() {
		if isOffered == offered {
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)

	return keys
}

// exchangeHashNoteGen draws a note of any of the three types with a value for
// every classified key.
func exchangeHashNoteGen() *rapid.Generator[exchangeHashNote] {
	return rapid.Custom(func(rt *rapid.T) exchangeHashNote {
		fields := map[string]any{}

		for _, key := range exchangeClassifiedKeys(true) {
			fields[key] = exchangeKeyValueGen(key).Draw(rt, key)
		}

		for _, key := range exchangeClassifiedKeys(false) {
			fields[key] = exchangeKeyValueGen(key).Draw(rt, key)
		}

		fields["type"] = rapid.SampledFrom([]string{"fact", "feedback", "runbook"}).Draw(rt, "noteType")

		return exchangeHashNote{fields: fields, body: exchangeBodyGen().Draw(rt, "body")}
	})
}

// exchangeKeyValueGen draws a value of the shape key has in real frontmatter.
func exchangeKeyValueGen(key string) *rapid.Generator[any] {
	text := rapid.StringMatching(`[a-zA-Z0-9][a-zA-Z0-9 .,'-]{0,20}`)
	list := rapid.SliceOfN(text, 0, 3)

	switch key {
	case "pending":
		return rapid.Map(rapid.Bool(), func(value bool) any { return value })
	case "red_flags", "triggers", "tags", "sources", "aliases":
		return rapid.Map(list, func(value []string) any { return value })
	case "supersedes":
		return rapid.Map(text, func(claim string) any {
			return []map[string]string{{"note": "9.2026-01-01.old.md", "type": "narrows", "claim": claim}}
		})
	case "parent":
		return rapid.Map(text, func(note string) any {
			return map[string]any{
				"vault": strings.Repeat("9a", 16),
				"links": []map[string]string{{"note": note, "via": "offered", "hash": "xh1:" + strings.Repeat("ab", 32)}},
			}
		})
	case "offer":
		return rapid.Map(text, func(forNote string) any {
			return map[string]any{"origin": "a:b", "key": "k", "for": forNote, "path": []string{"a"}}
		})
	default:
		return rapid.Map(text, func(value string) any { return value })
	}
}

func mustExchangeHash(rt failer, note string) string {
	rt.Helper()

	hash, err := cli.ExportExchangeHash([]byte(note))
	if err != nil {
		rt.Fatalf("exchange hash of\n%s\nfailed: %v", note, err)
	}

	return hash
}

// sameFrontmatterValue reports whether two drawn values render identically
// (a nil and an empty list both render as []).
func sameFrontmatterValue(a, b any) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// yamlKeysOf returns the top-level YAML keys a struct type marshals,
// descending into `yaml:",inline"` embedded structs.
func yamlKeysOf(structType reflect.Type) []string {
	keys := make([]string, 0, structType.NumField())

	for field := range structType.Fields() {
		name, options, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		if name == "-" {
			continue
		}

		if slices.Contains(strings.Split(options, ","), "inline") {
			keys = append(keys, yamlKeysOf(field.Type)...)

			continue
		}

		if name == "" {
			name = strings.ToLower(field.Name)
		}

		keys = append(keys, name)
	}

	return keys
}

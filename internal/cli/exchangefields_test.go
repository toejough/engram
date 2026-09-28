package cli_test

// Tests for the exchange frontmatter fields themselves (design D4,
// vault-note-identity "Exchanged notes SHALL carry a stable exchange ID";
// task 3.3): every frontmatter struct declares them, a note without them
// serializes exactly as before, nothing stamps them by backfill, and JSON
// decoding can never populate them.

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// TestExchangeFields_AbsentFieldsSerializeAsToday pins every frontmatter
// renderer's output for a note with every pre-exchange field set and no
// exchange field: the omitempty exchange fields add nothing.
func TestExchangeFields_AbsentFieldsSerializeAsToday(t *testing.T) {
	t.Parallel()

	when := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	supersedes := []cli.ExportSupersedesEntry{{Note: "9.2026-01-01.old.md", Type: "narrows", Claim: "old claim"}}
	provenance := "luhmann: \"1a\"\ncreated: \"2026-01-02\"\nsource: src\nproject: proj\nrepo: r\nuser: u\nvault: v\n"
	tail := "pending: true\nissue: \"12\"\nsources:\n    - c#a\n"
	listTail := "tags:\n    - vocab/t\n" +
		"supersedes:\n    - note: 9.2026-01-01.old.md\n      type: narrows\n      claim: old claim\n"

	cases := map[string]struct {
		rendered, want string
	}{
		"fact": {
			rendered: cli.ExportRenderFactFrontmatter(cli.ExportFactFields{
				Situation: "s", Subject: "sub", Predicate: "p", Object: "o", Luhmann: "1a", Source: "src",
				Project: "proj", Repo: "r", User: "u", Vault: "v", Pending: true, Issue: "12", Tier: "L2",
				ChunkSources: []string{"c#a"}, Tags: []string{"vocab/t"}, Supersedes: supersedes, VocabVersion: "1.0",
			}, when),
			want: "---\ntype: fact\ntier: L2\nsituation: s\nsubject: sub\npredicate: p\nobject: o\n" +
				provenance + tail + "vocab_version: \"1.0\"\n" + listTail + "---\n\n",
		},
		"feedback": {
			rendered: cli.ExportRenderFeedbackFrontmatter(cli.ExportFeedbackFields{
				Situation: "s", Behavior: "b", Impact: "i", Action: "a", Luhmann: "1a", Source: "src",
				Project: "proj", Repo: "r", User: "u", Vault: "v", Pending: true, Issue: "12", Tier: "L2",
				ChunkSources: []string{"c#a"}, Tags: []string{"vocab/t"}, Supersedes: supersedes,
			}, when),
			want: "---\ntype: feedback\ntier: L2\nsituation: s\nbehavior: b\nimpact: i\naction: a\n" +
				provenance + tail + listTail + "---\n\n",
		},
		"runbook": {
			rendered: cli.ExportRenderRunbookFrontmatter(cli.ExportRunbookFields{
				Situation: "s", DoneWhen: "d", RedFlags: []string{"rf"}, Triggers: []string{"tr"}, Luhmann: "1a",
				Source: "src", Project: "proj", Repo: "r", User: "u", Vault: "v", SkillHash: "h", SkillKey: "k:x",
				SkillSource: "~/s/SKILL.md", Pending: true, Issue: "12", Tier: "L2",
				ChunkSources: []string{"c#a"}, Tags: []string{"vocab/t"}, Supersedes: supersedes,
			}, when),
			want: "---\ntype: runbook\ntier: L2\nsituation: s\ndone_when: d\nred_flags:\n    - rf\ntriggers:\n    - tr\n" +
				provenance + "skill_hash: h\nskill_key: k:x\nskill_source: ~/s/SKILL.md\n" + tail + listTail + "---\n\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(tc.rendered).To(Equal(tc.want))
		})
	}
}

// TestExchangeFields_BackfillStampsNoXID is "No backfill": identity backfill
// (the only `engram update` step that rewrites note frontmatter) stamps its
// identity fields and adds no exchange field to any note.
func TestExchangeFields_BackfillStampsNoXID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fact := "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"1\"\ncreated: \"2026-01-01\"\nsource: t\n---\n\nInformation learned: when s, a b c.\n"
	feedback := "---\ntype: feedback\nsituation: s\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"2\"\ncreated: \"2026-01-01\"\nsource: t\n---\n\nLesson learned: when s, a.\n"
	runbook := "---\ntype: runbook\nsituation: s\ndone_when: d\n" +
		"luhmann: \"3\"\ncreated: \"2026-01-01\"\nsource: t\n---\n\n1. step\n"

	fs := newSkillSurvivalFS(map[string]string{
		"1.2026-01-01.fact.md": fact, "2.2026-01-01.feedback.md": feedback, "3.2026-01-01.runbook.md": runbook,
	})

	stamped, err := cli.ExportBackfillIdentity(t.Context(), "/vault", cli.IdentityDeps{
		Lock:       func(string) (func(), error) { return func() {}, nil },
		ListMD:     fs.listMD,
		ReadFile:   fs.readFile,
		WriteFile:  fs.writeFile,
		DetectRepo: func(context.Context) string { return "" },
		DetectUser: func(context.Context) string { return "agent@example.com" },
		Getenv:     func(string) string { return "" },
	}, false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stamped).To(Equal(2), "the backfill must actually have rewritten the fact and feedback notes")

	for _, name := range fs.names {
		for _, key := range exchangeFieldKeys {
			g.Expect(yamlKeyBlock(fs.content(name), key)).To(BeEmpty(), name+" must not gain "+key)
		}
	}
}

// TestExchangeFields_DeclaredOnEveryFrontmatterStruct: the fact, feedback and
// runbook frontmatter structs all carry xid, parent, aliases and offer.
func TestExchangeFields_DeclaredOnEveryFrontmatterStruct(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	for _, docType := range cli.ExportFrontmatterDocTypes() {
		g.Expect(yamlKeysOf(docType)).To(ContainElements(exchangeFieldKeys), docType.Name())
	}
}

// TestExchangeFields_JSONNeverPopulatesThem: decoding JSON into a
// frontmatter struct, under any key spelling, never sets an exchange field
// (they are tagged json:"-"; the served-learn wire itself is task 3.7).
func TestExchangeFields_JSONNeverPopulatesThem(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"xid": "x", "XID": "x", "Xid": "x",
		"parent": {"vault": "v", "links": [{"note": "n", "via": "offered", "hash": "h"}]},
		"Parent": {"Vault": "v"},
		"aliases": ["a"], "Aliases": ["a"],
		"offer": {"origin": "o", "key": "k", "for": "f", "path": ["p"]},
		"Offer": {"Origin": "o"},
		"ExchangeFrontmatter": {"xid": "x"}, "exchangeFrontmatter": {"xid": "x"}
	}`)

	for _, docType := range cli.ExportFrontmatterDocTypes() {
		t.Run(docType.Name(), func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			doc := reflect.New(docType)
			g.Expect(json.Unmarshal(payload, doc.Interface())).To(Succeed())

			rendered, err := yaml.Marshal(doc.Elem().Interface())
			g.Expect(err).NotTo(HaveOccurred())

			for _, key := range exchangeFieldKeys {
				g.Expect(strings.Contains(string(rendered), "\n"+key+":")).To(BeFalse(),
					key+" must not be decodable from JSON:\n"+string(rendered))
			}
		})
	}
}

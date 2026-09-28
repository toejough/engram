package cli_test

// resituate preserves every field it does not change (design D10 M7,
// vault-note-identity "Resituate preserves every untouched field"; task
// 3.4): only situation: and the body opener differ.

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"
)

// TestRunResituate_PreservesEveryUntouchedField is the spec scenario: a
// pending note carrying tags, sources, supersedes, vocab_version, issue,
// project, parent and aliases comes out of resituate with all of them and
// the pending marker unchanged; only situation and the body opener differ.
func TestRunResituate_PreservesEveryUntouchedField(t *testing.T) {
	t.Parallel()

	for _, noteType := range []string{"fact", "feedback"} {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			note := resituateFixture{
				Type: noteType, Tier: "L2", Situation: "working on the old widget",
				Subject: "the widget", Predicate: "uses", Object: "a gear",
				Behavior: "skipped it", Impact: "it broke", Action: "do it",
				Luhmann: "1aa", Created: "2026-01-01", Source: "test", Project: "widgets",
				Repo: "github.com/acme/widgets", User: "alice@example.com", Vault: "team", Pending: true,
				Issue: "#12", Sources: []string{"session.jsonl#a1"}, Tags: []string{"vocab/widgets"},
				Supersedes: []resituateSupersedes{{Note: "9.2026-01-01.old.md", Type: "narrows", Claim: "old"}},
				Exchange:   exchangeSurvivalFixture(),
			}
			if noteType == "fact" {
				note.VocabVersion = "6.1"
			}

			before := note.render("Supersedes: [[9.2026-01-01.old]] — narrows: old\n")

			after, writes, err := runExchangeSurvivalResituate(t.Context(), before)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(writes).To(Equal(1))

			expectOnlySituationAndOpenerChanged(g, noteType, before, after)
		})
	}
}

// TestRunResituate_PreservesEveryUntouchedFieldProperty is the property form:
// whichever optional fields a fact or feedback note carries, resituate
// changes only situation: and the body opener.
func TestRunResituate_PreservesEveryUntouchedFieldProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		noteType := rapid.SampledFrom([]string{"fact", "feedback"}).Draw(rt, "type")
		text := rapid.StringMatching(`[a-z][a-z0-9 ]{0,15}[a-z0-9]`)
		optional := func(label string) string {
			if rapid.Bool().Draw(rt, label+"Present") {
				return text.Draw(rt, label)
			}

			return ""
		}

		note := resituateFixture{
			Type: noteType, Tier: optional("tier"), Situation: text.Draw(rt, "situation"),
			Subject: text.Draw(rt, "subject"), Predicate: text.Draw(rt, "predicate"), Object: text.Draw(rt, "object"),
			Behavior: text.Draw(rt, "behavior"), Impact: text.Draw(rt, "impact"), Action: text.Draw(rt, "action"),
			Luhmann: "1aa", Created: "2026-01-01", Source: text.Draw(rt, "source"), Project: optional("project"),
			Repo: optional("repo"), User: text.Draw(rt, "user"), Vault: text.Draw(rt, "vault"),
			Pending: rapid.Bool().Draw(rt, "pending"), Issue: optional("issue"),
			Sources: rapid.SliceOfN(text, 0, 3).Draw(rt, "sources"),
			Tags:    rapid.SliceOfN(text, 0, 3).Draw(rt, "tags"),
		}

		if noteType == "fact" {
			note.VocabVersion = optional("vocabVersion")
		}

		if rapid.Bool().Draw(rt, "exchange") {
			note.Exchange = exchangeFixtureGen().Draw(rt, "exchangeFields")
		}

		tail := ""

		if rapid.Bool().Draw(rt, "supersedes") {
			claim := text.Draw(rt, "claim")
			note.Supersedes = []resituateSupersedes{{Note: "9.2026-01-01.old.md", Type: "narrows", Claim: claim}}
			tail = "Supersedes: [[9.2026-01-01.old]] — narrows: " + claim + "\n"
		}

		before := note.render(tail)

		after, writes, err := runExchangeSurvivalResituate(t.Context(), before)
		if err != nil {
			rt.Fatalf("resituate: %v", err)
		}

		if writes != 1 {
			rt.Fatalf("resituate wrote %d times, want 1", writes)
		}

		expectOnlySituationAndOpenerChanged(NewWithT(rt), noteType, before, after)
	})
}

// resituateFixture renders a fact or feedback note with every frontmatter
// field either type can carry, in the frontmatter writer's key order.
type resituateFixture struct {
	Type         string                `yaml:"type"`
	Tier         string                `yaml:"tier,omitempty"`
	Situation    string                `yaml:"situation"`
	Subject      string                `yaml:"subject,omitempty"`
	Predicate    string                `yaml:"predicate,omitempty"`
	Object       string                `yaml:"object,omitempty"`
	Behavior     string                `yaml:"behavior,omitempty"`
	Impact       string                `yaml:"impact,omitempty"`
	Action       string                `yaml:"action,omitempty"`
	Luhmann      string                `yaml:"luhmann"`
	Created      string                `yaml:"created"`
	Source       string                `yaml:"source"`
	Project      string                `yaml:"project,omitempty"`
	Repo         string                `yaml:"repo,omitempty"`
	User         string                `yaml:"user"`
	Vault        string                `yaml:"vault"`
	Pending      bool                  `yaml:"pending,omitempty"`
	Issue        string                `yaml:"issue,omitempty"`
	Sources      []string              `yaml:"sources,omitempty"`
	VocabVersion string                `yaml:"vocab_version,omitempty"`
	Tags         []string              `yaml:"tags,omitempty"`
	Supersedes   []resituateSupersedes `yaml:"supersedes,omitempty"`
	Exchange     exchangeFixture       `yaml:",inline"`
}

// render returns the note: frontmatter, the type's body opener, and tail.
func (f resituateFixture) render(tail string) string {
	if f.Type == "fact" {
		f.Behavior, f.Impact, f.Action = "", "", ""
	} else {
		f.Subject, f.Predicate, f.Object = "", "", ""
	}

	frontmatter, _ := yaml.Marshal(f)

	opener := resituateOpener(f.Type, f.Situation, frontmatterOf(string(frontmatter)))

	return "---\n" + string(frontmatter) + "---\n\n" + opener + "\n\n" + tail
}

type resituateSupersedes struct {
	Note  string `yaml:"note"`
	Type  string `yaml:"type"`
	Claim string `yaml:"claim"`
}

// expectOnlySituationAndOpenerChanged asserts after equals before except
// for the situation: value (now "resituated context") and the body's first
// line (the opener rebuilt around it).
func expectOnlySituationAndOpenerChanged(g Gomega, noteType, before, after string) {
	beforeFields := frontmatterOf(before)
	afterFields := frontmatterOf(after)

	g.Expect(afterFields["situation"]).To(Equal("resituated context"))
	delete(beforeFields, "situation")
	delete(afterFields, "situation")
	g.Expect(afterFields).To(Equal(beforeFields), "every field but situation must be unchanged")

	_, beforeBody, _ := strings.Cut(strings.TrimPrefix(before, "---\n"), "\n---\n")
	_, afterBody, _ := strings.Cut(strings.TrimPrefix(after, "---\n"), "\n---\n")
	beforeOpener, beforeRest, _ := strings.Cut(strings.TrimPrefix(beforeBody, "\n"), "\n")
	afterOpener, afterRest, _ := strings.Cut(strings.TrimPrefix(afterBody, "\n"), "\n")

	g.Expect(afterRest).To(Equal(beforeRest), "the body after the opener must be byte-unchanged")
	g.Expect(afterOpener).NotTo(Equal(beforeOpener))
	g.Expect(afterOpener).To(Equal(resituateOpener(noteType, "resituated context", beforeFields)))
}

// frontmatterOf decodes a note's frontmatter (or a bare frontmatter
// document) into a generic map.
func frontmatterOf(content string) map[string]any {
	block := content
	if rest, ok := strings.CutPrefix(content, "---\n"); ok {
		block, _, _ = strings.Cut(rest, "\n---\n")
	}

	fields := map[string]any{}
	_ = yaml.Unmarshal([]byte(block), &fields)

	return fields
}

// resituateOpener is the body's first line for noteType with situation.
func resituateOpener(noteType, situation string, fields map[string]any) string {
	field := func(key string) string {
		value, _ := fields[key].(string)

		return value
	}

	if noteType == "fact" {
		return "Information learned: when in " + situation + ", " +
			field("subject") + " " + field("predicate") + " " + field("object") + "."
	}

	return "Lesson learned: when " + situation + ", " + field("action") + "."
}

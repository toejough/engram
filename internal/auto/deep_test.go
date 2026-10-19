package auto_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test855d4j1_ReadDeep_ALessonNamesTheNoteItCorrectsAsAFileNameAnObjectOrAList(t *testing.T) {
	t.Parallel()

	const lesson = `{"kind":"fact","slug":"a-fact","situation":"s","covered":null,` +
		`"subject":"x","predicate":"y","object":"z"`

	carry := auto.Carry{Depth: auto.DepthDeep, Candidates: []auto.Candidate{{ID: "1.old.md", Kind: auto.KindNote}}}
	want := auto.Ties{{Note: "1.old.md", Type: "updates", Claim: "moved"}}

	for name, tie := range map[string]string{
		"a file name with the type and the claim of the lesson": `,"supersedes":"1.old.md","type":"updates","claim":"moved"`,
		"an object": `,"supersedes":{"note":"1.old.md","type":"updates","claim":"moved"}`,
		"a list":    `,"supersedes":[{"note":"1.old.md","type":"updates","claim":"moved"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deep := auto.ReadDeep(`{"candidates":[],"lessons":[`+lesson+tie+`}]}`, carry)

			g := NewWithT(t)
			g.Expect(deep.Lessons).To(HaveLen(1))

			if len(deep.Lessons) == 1 {
				g.Expect(deep.Lessons[0].Supersedes).To(Equal(want))
			}
		})
	}
}

func Test855d4j1_ReadDeep_ANearNoteIsRevisedByTheFieldsOfItsKind(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		kind, field, other string
	}{
		"a fact":     {kind: "fact", field: "object", other: "action"},
		"a feedback": {kind: "feedback", field: "action", other: "object"},
		"a runbook":  {kind: "runbook", field: "done_when", other: "object"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			carry := auto.Carry{Candidates: []auto.Candidate{{ID: "1.a.md", Kind: auto.KindNote, Type: test.kind}}}
			deep := auto.ReadDeep(`{"candidates":[{"id":"1.a.md","verdict":"near","fields":{"`+test.field+
				`":"new","`+test.other+`":"x"}}],"lessons":[]}`, carry)

			g.Expect(deep.Amendments).To(Equal([]auto.Amendment{
				{Note: "1.a.md", Fields: map[string]string{test.field: "new"}},
			}))
			g.Expect(deep.Skipped).To(HaveLen(1), "the field the kind has not")
		})
	}
}

func Test855d4j1_ReadDeep_AReplyThatIsNoObjectWritesNothingAndSaysWhy(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	deep := auto.ReadDeep(`["not", "the object"]`, auto.Carry{})

	g.Expect(deep.Lessons).To(BeEmpty())
	g.Expect(deep.Amendments).To(BeEmpty())
	g.Expect(deep.Skipped).To(HaveLen(1))
}

func Test855d4j1_ReadDeep_ARunbookLessonIsLeftOutAndSaidSo(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	deep := auto.ReadDeep(`{"candidates":[],"lessons":[{"kind":"runbook","slug":"a-run","situation":"s",`+
		`"done_when":"d","body":"1. a 2. b"}]}`, auto.Carry{})

	g.Expect(deep.Lessons).To(BeEmpty())
	g.Expect(deep.Skipped).To(ConsistOf(auto.Skipped{
		Slug: "a-run", Reason: "a deep delivery writes facts and feedback only",
	}))
}

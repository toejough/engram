package auto_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test855d4j1g_ReadDeep_ALessonOfAnUnexpectedFieldTypeIsLeftOutAndTheGoodLessonIsKept(t *testing.T) {
	t.Parallel()

	const good = `{"kind":"fact","slug":"good-one","situation":"s","covered":null,` +
		`"subject":"x","predicate":"y","object":"z"}`

	carry := auto.Carry{Depth: auto.DepthDeep, Candidates: []auto.Candidate{{ID: "1.old.md", Kind: auto.KindNote}}}

	for name, test := range map[string]struct {
		fault   string
		skipped []auto.Skipped
		ties    auto.Ties
	}{
		"chunks as a string": {fault: `,"chunks":"c#1"`, skipped: []auto.Skipped{{Slug: "bad-one"}}},
		"a list of names":    {fault: `,"supersedes":["1.old.md"]`},
		"a number":           {fault: `,"supersedes":3`},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			bad := `{"kind":"fact","slug":"bad-one","situation":"s","covered":null,"subject":"x","predicate":"y",` +
				`"object":"z"` + test.fault + `}`
			deep := auto.ReadDeep(`{"candidates":[],"lessons":[`+bad+`,`+good+`]}`, carry)

			slugs := make([]string, 0, len(deep.Lessons))
			for _, lesson := range deep.Lessons {
				slugs = append(slugs, lesson.Slug)
			}

			g.Expect(slugs).To(ContainElement("good-one"))

			if len(test.skipped) > 0 {
				g.Expect(deep.Skipped).To(HaveLen(1))

				if len(deep.Skipped) == 1 {
					g.Expect(deep.Skipped[0].Slug).To(Equal("bad-one"))
				}
			}
		})
	}
}

func Test861b4zzze1a_ReadDeep_ASupersessionIsReadAsAFileNameAnObjectOrAList(t *testing.T) {
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

func Test861b4zzze1b_ReadDeep_ALessonThatCannotBeReadAndHasNoSlugIsLeftOutByItsPlace(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	deep := auto.ReadDeep(`{"candidates":[],"lessons":[{"kind":"fact","chunks":"c#1"},`+
		`{"kind":"fact","slug":"good-one","situation":"s","covered":null,"subject":"x","predicate":"y","object":"z"}]}`,
		auto.Carry{})

	g.Expect(deep.Lessons).To(HaveLen(1))
	g.Expect(deep.Skipped).To(HaveLen(1))

	if len(deep.Skipped) == 1 {
		g.Expect(deep.Skipped[0].Reason).To(ContainSubstring("lesson 1"))
	}
}

func Test861b4zzze1c_ReadDeep_AListOfNamesIsTheTiesOfThoseNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	carry := auto.Carry{Candidates: []auto.Candidate{{ID: "1.old.md", Kind: auto.KindNote}}}
	deep := auto.ReadDeep(`{"candidates":[],"lessons":[{"kind":"fact","slug":"good-one","situation":"s","covered":null,`+
		`"subject":"x","predicate":"y","object":"z","supersedes":["1.old.md"],"type":"updates","claim":"c"}]}`, carry)

	g.Expect(deep.Lessons).To(HaveLen(1))

	if len(deep.Lessons) == 1 {
		g.Expect(deep.Lessons[0].Supersedes).To(Equal(auto.Ties{{Note: "1.old.md", Type: "updates", Claim: "c"}}))
	}
}

func Test861b4zzze1d_ReadDeep_ANearNoteIsRevisedByTheFieldsOfItsKind(t *testing.T) {
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

func Test861b4zzze1e_ReadDeep_AReplyThatIsNotAnObjectWritesNothingAndSaysWhy(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	deep := auto.ReadDeep(`["not", "the object"]`, auto.Carry{})

	g.Expect(deep.Lessons).To(BeEmpty())
	g.Expect(deep.Amendments).To(BeEmpty())
	g.Expect(deep.Skipped).To(HaveLen(1))

	if len(deep.Skipped) == 1 {
		g.Expect(deep.Skipped[0].Reason).To(ContainSubstring(notDeepReplyReason))
	}
}

func Test861b4zzze1f_ReadDeep_AReplyWithNoCandidatesOrCutShortIsNoDeepReply(t *testing.T) {
	t.Parallel()

	for name, reply := range map[string]string{
		"no candidates key": `{"lessons":[]}`,
		"a cut fragment":    `{"candidates":[{"id":"1.a.md","verd`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			deep := auto.ReadDeep(reply, auto.Carry{})

			g.Expect(deep.Lessons).To(BeEmpty())
			g.Expect(deep.Amendments).To(BeEmpty())
			g.Expect(deep.Skipped).To(HaveLen(1))

			if len(deep.Skipped) == 1 {
				g.Expect(deep.Skipped[0].Reason).To(ContainSubstring(notDeepReplyReason))
			}
		})
	}
}

func Test861b4zzze1g_ReadDeep_ARunbookLessonIsLeftOutAndSaidSo(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	deep := auto.ReadDeep(`{"candidates":[],"lessons":[{"kind":"runbook","slug":"a-run","situation":"s",`+
		`"done_when":"d","body":"1. a 2. b"}]}`, auto.Carry{})

	g.Expect(deep.Lessons).To(BeEmpty())
	g.Expect(deep.Skipped).To(ConsistOf(auto.Skipped{
		Slug: "a-run", Reason: "a deep delivery writes facts and feedback only",
	}))
}

func Test861b4zzze1h_ReadDeep_ASupersessionNamedWithoutItsExtensionOrWithSpacesIsStoredUnderTheNoteID(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	carry := auto.Carry{Candidates: []auto.Candidate{{ID: "1.old.md", Kind: auto.KindNote, Type: "fact"}}}
	deep := auto.ReadDeep(`{"candidates":[],"lessons":[{"kind":"fact","slug":"good-one","situation":"s",`+
		`"covered":null,"subject":"x","predicate":"y","object":"z","supersedes":" 1.old ","type":"updates",`+
		`"claim":"c"}]}`, carry)

	g.Expect(deep.Lessons).To(HaveLen(1))

	if len(deep.Lessons) == 1 {
		g.Expect(deep.Lessons[0].Supersedes).To(Equal(auto.Ties{{Note: "1.old.md", Type: "updates", Claim: "c"}}))
	}
}

// unexported constants.
const (
	notDeepReplyReason = "the reply is not the object of the candidates and the lessons"
)

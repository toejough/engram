package auto_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test793d4h1c_ParseLessons_AReplyThatIsNotAListOfLessonsWritesNothing(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct{ reply, slug, reason string }{
		"not JSON":      {reply: "I think there is one lesson here.", reason: "not a list of lessons"},
		"a JSON object": {reply: `{"kind":"fact"}`, reason: "not a list of lessons"},
		"no reply":      {reply: ``, reason: "not a list of lessons"},
		"a lesson of unknown kind": {reply: `[{"kind":"runbook","slug":"a-b","situation":"s","subject":"x",` +
			`"predicate":"y","object":"z"}]`, slug: "a-b", reason: "kind"},
		"a lesson with no slug": {reply: `[{"kind":"fact","slug":"","situation":"s","subject":"x",` +
			`"predicate":"y","object":"z"}]`, reason: "slug"},
		"a lesson with a slug that is not kebab-case": {reply: `[{"kind":"fact","slug":"Not Kebab","situation":"s",` +
			`"subject":"x","predicate":"y","object":"z"}]`, slug: "Not Kebab", reason: "slug"},
		"a lesson with no situation": {reply: `[{"kind":"fact","slug":"a-b","situation":" ","subject":"x",` +
			`"predicate":"y","object":"z"}]`, slug: "a-b", reason: "situation"},
		"a feedback lesson with an empty field": {reply: `[{"kind":"feedback","slug":"a-b","situation":"s",` +
			`"behavior":"x","impact":"","action":"z"}]`, slug: "a-b", reason: "impact"},
		"a fact lesson with a missing field": {reply: `[{"kind":"fact","slug":"a-b","situation":"s",` +
			`"subject":"x","predicate":"y"}]`, slug: "a-b", reason: "object"},
		"a fact lesson that holds the fields of a feedback lesson": {reply: `[{"kind":"fact","slug":"a-b",` +
			`"situation":"s","behavior":"x","impact":"y","action":"z"}]`, slug: "a-b", reason: "subject"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			lessons, skipped := auto.ParseLessons(testCase.reply)

			g.Expect(lessons).To(BeEmpty())
			g.Expect(skipped).To(HaveLen(1))

			if len(skipped) == 1 {
				g.Expect(skipped[0].Slug).To(Equal(testCase.slug))
				g.Expect(skipped[0].Reason).To(ContainSubstring(testCase.reason))
			}
		})
	}

	t.Run("an empty list holds no lesson and leaves none out", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		lessons, skipped := auto.ParseLessons(`[]`)

		g.Expect(lessons).To(BeEmpty())
		g.Expect(skipped).To(BeEmpty())
	})

	t.Run("a whole lesson beside a lesson that is not whole", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		lessons, skipped := auto.ParseLessons(`[{"kind":"fact","slug":"","situation":"s","subject":"x",` +
			`"predicate":"y","object":"z"},` +
			`{"kind":"fact","slug":"a-b","situation":"s","subject":"x","predicate":"y","object":"z"}]`)

		g.Expect(lessons).To(HaveLen(1))
		g.Expect(skipped).To(HaveLen(1))

		if len(lessons) == 1 {
			g.Expect(lessons[0].Slug).To(Equal("a-b"))
		}
	})
}

func Test793d4h1d_ParseLessons_TheCodeFenceAroundTheListIsNotPartOfIt(t *testing.T) {
	t.Parallel()

	list := `[{"kind":"fact","slug":"staging-reset","situation":"testing on the staging database",` +
		`"covered":null,"subject":"the staging database","predicate":"is reset every","object":"Sunday night"}]`

	for name, reply := range map[string]string{
		"a fence that names json":     "```json\n" + list + "\n```",
		"a fence that names nothing":  "```\n" + list + "\n```\n",
		"white space around the list": "\n  " + list + "\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			lessons, skipped := auto.ParseLessons(reply)

			g.Expect(skipped).To(BeEmpty())
			g.Expect(lessons).To(HaveLen(1))

			if len(lessons) == 1 {
				g.Expect(lessons[0].Kind).To(Equal("fact"))
				g.Expect(lessons[0].Object).To(Equal("Sunday night"))
			}
		})
	}
}

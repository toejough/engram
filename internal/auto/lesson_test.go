package auto_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test793d4h1c_ParseLessons_AReplyThatIsNotAListOfLessonsWritesNothing(t *testing.T) {
	t.Parallel()

	for name, reply := range map[string]string{
		"not JSON":      "I think there is one lesson here.",
		"a JSON object": `{"kind":"fact"}`,
		"an empty list": `[]`,
		"no reply":      ``,
		"a lesson of unknown kind": `[{"kind":"runbook","slug":"a-b","situation":"s","subject":"x",` +
			`"predicate":"y","object":"z"}]`,
		"a lesson with no slug": `[{"kind":"fact","slug":"","situation":"s","subject":"x","predicate":"y","object":"z"}]`,
		"a lesson with a slug that is not kebab-case": `[{"kind":"fact","slug":"Not Kebab","situation":"s",` +
			`"subject":"x","predicate":"y","object":"z"}]`,
		"a lesson with no situation": `[{"kind":"fact","slug":"a-b","situation":" ","subject":"x",` +
			`"predicate":"y","object":"z"}]`,
		"a feedback lesson with an empty field": `[{"kind":"feedback","slug":"a-b","situation":"s","behavior":"x",` +
			`"impact":"","action":"z"}]`,
		"a fact lesson with a missing field": `[{"kind":"fact","slug":"a-b","situation":"s","subject":"x","predicate":"y"}]`,
		"a fact lesson that holds the fields of a feedback lesson": `[{"kind":"fact","slug":"a-b","situation":"s",` +
			`"behavior":"x","impact":"y","action":"z"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			lessons, _ := auto.ParseLessons(reply)

			g.Expect(lessons).To(BeEmpty())
		})
	}

	t.Run("a whole lesson beside a lesson that is not whole", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		lessons, err := auto.ParseLessons(`[{"kind":"fact","slug":"","situation":"s","subject":"x",` +
			`"predicate":"y","object":"z"},` +
			`{"kind":"fact","slug":"a-b","situation":"s","subject":"x","predicate":"y","object":"z"}]`)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(lessons).To(HaveLen(1))
		g.Expect(lessons[0].Slug).To(Equal("a-b"))
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

			lessons, err := auto.ParseLessons(reply)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(lessons).To(HaveLen(1))
			g.Expect(lessons[0].Kind).To(Equal("fact"))
			g.Expect(lessons[0].Object).To(Equal("Sunday night"))
		})
	}
}

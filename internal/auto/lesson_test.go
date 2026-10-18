package auto_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

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

func Test793d4k1j_RecordedReason_WhateverTextSurroundsACredentialNoPartOfItIsKept(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		found := credentialShape().Draw(rt, "credential")
		// letters that spell none of the names of a secret, so that the text around the credential changes none of
		// its shape: the place where the reason is cut is the only thing that varies
		before := rapid.StringMatching(`[xyz .,\n]{0,400}`).Draw(rt, "before")
		after := rapid.StringMatching(`[xyz .,\n]{0,400}`).Draw(rt, "after")

		reason := auto.RecordedReason(before + " " + found.text + " " + after)

		switch {
		case strings.Contains(reason, "\n"):
			rt.Fatalf("the reason is not one line: %q", reason)
		case utf8.RuneCountInString(reason) > recordedReasonLimit:
			rt.Fatalf("the reason is %d characters long: %q", utf8.RuneCountInString(reason), reason)
		case holdsPartOf(reason, found.secret):
			rt.Fatalf("a part of the secret %q is in %q", found.secret, reason)
		}
	})
}

// unexported constants.
const (
	// partLength is how many characters of a secret make a part of it that must not be kept.
	partLength = 8
	// recordedReasonLimit is how many characters of a reason a record keeps at most (793d4k).
	recordedReasonLimit = 300
)

// holdsPartOf reports whether the text holds the secret or, when it is longer than partLength characters, a part of it
// of partLength characters.
func holdsPartOf(text, secret string) bool {
	runes := []rune(strings.TrimSpace(secret))

	if len(runes) <= partLength {
		return strings.Contains(text, string(runes))
	}

	for start := 0; start+partLength <= len(runes); start++ {
		if strings.Contains(text, string(runes[start:start+partLength])) {
			return true
		}
	}

	return false
}

package reminders_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/reminders"
)

func TestGet_EmptyKind_ReturnsErrUnknownKind(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	got, err := reminders.Get("")
	g.Expect(got).To(BeEmpty())
	g.Expect(errors.Is(err, reminders.ErrUnknownKind)).To(BeTrue())
}

func TestGet_UnknownKind_ReturnsErrUnknownKind(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	got, err := reminders.Get("not-a-kind")
	g.Expect(got).To(BeEmpty())
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, reminders.ErrUnknownKind)).To(BeTrue())
}

func TestGet_ValidKinds_ReturnExpectedConstants(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind string
		want string
	}{
		{name: "session-start", kind: "session-start", want: reminders.SessionStart},
		{name: "user-prompt", kind: "user-prompt", want: reminders.UserPrompt},
		{name: "post-tool", kind: "post-tool", want: reminders.PostTool},
		{name: "system", kind: "system", want: reminders.System},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got, err := reminders.Get(tc.kind)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(got).NotTo(BeEmpty())
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

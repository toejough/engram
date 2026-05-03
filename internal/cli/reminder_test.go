package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/reminders"
)

func TestReminder_KnownKind_PrintsCanonicalText(t *testing.T) {
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

			stdout, stderr := executeForTest(t, []string{"engram", "reminder", tc.kind})
			g.Expect(stderr).To(BeEmpty())
			g.Expect(stdout).To(Equal(tc.want))
		})
	}
}

func TestReminder_UnknownKind_WritesErrorToStderr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stdout, stderr := executeForTest(t, []string{"engram", "reminder", "bogus"})
	g.Expect(stdout).To(BeEmpty())
	g.Expect(stderr).NotTo(BeEmpty())
	g.Expect(stderr).To(ContainSubstring("unknown reminder kind"))
}

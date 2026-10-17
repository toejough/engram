package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func Test793d4h1a_RunAutoWrite_AFeedbackLessonBecomesAFeedbackNoteMarkedAsWrittenAutomatically(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	written := map[string]string{}

	var out bytes.Buffer

	err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", VaultName: "personal"},
		cli.AutoWriteDeps{
			Stdin: strings.NewReader(`{"session":"` + sessionID + `","reply":"[{\"kind\":\"feedback\",` +
				`\"slug\":\"format-before-commit\",\"situation\":\"committing a Python file\",\"covered\":null,` +
				`\"behavior\":\"left trailing whitespace\",\"impact\":\"the lint check failed\",` +
				`\"action\":\"run the formatter before committing\"}]"}`),
			Learn: learnInto(written),
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(written).To(HaveLen(1))

	for path, note := range written {
		g.Expect(path).To(HaveSuffix(".format-before-commit.md"))
		g.Expect(note).To(ContainSubstring("type: feedback"))
		g.Expect(note).To(ContainSubstring("situation: committing a Python file"))
		g.Expect(note).To(ContainSubstring("left trailing whitespace"))
		g.Expect(note).To(ContainSubstring("the lint check failed"))
		g.Expect(note).To(ContainSubstring("run the formatter before committing"))
		g.Expect(note).To(ContainSubstring("source: written automatically (session " + sessionID + ")"))
		g.Expect(out.String()).To(ContainSubstring(path))
	}
}

func Test793d4h1b_RunAutoWrite_AFactLessonBecomesAFactNoteMarkedAsWrittenAutomatically(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	written := map[string]string{}

	var out bytes.Buffer

	err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault", VaultName: "personal"},
		cli.AutoWriteDeps{
			Stdin: strings.NewReader(`{"session":"` + sessionID + `","reply":"[{\"kind\":\"fact\",` +
				`\"slug\":\"staging-reset\",\"situation\":\"testing on the staging database\",\"covered\":null,` +
				`\"subject\":\"the staging database\",\"predicate\":\"is reset every\",\"object\":\"Sunday night\"}]"}`),
			Learn: learnInto(written),
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(written).To(HaveLen(1))

	for path, note := range written {
		g.Expect(path).To(HaveSuffix(".staging-reset.md"))
		g.Expect(note).To(ContainSubstring("type: fact"))
		g.Expect(note).To(ContainSubstring("situation: testing on the staging database"))
		g.Expect(note).To(ContainSubstring("the staging database"))
		g.Expect(note).To(ContainSubstring("Sunday night"))
		g.Expect(note).To(ContainSubstring("source: written automatically (session " + sessionID + ")"))
	}
}

func Test793d4h1c_RunAutoWrite_AReplyThatWritesNothingCallsNoLearn(t *testing.T) {
	t.Parallel()

	for name, reply := range map[string]string{
		"not JSON":      `"I think there is one lesson here."`,
		"an empty list": `"[]"`,
		"a lesson that is not whole": `"[{\"kind\":\"fact\",\"slug\":\"\",\"situation\":\"s\",\"subject\":\"x\",` +
			`\"predicate\":\"y\",\"object\":\"z\"}]"`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			written := map[string]string{}

			var out bytes.Buffer

			err := cli.RunAutoWrite(context.Background(), cli.AutoWriteArgs{VaultPath: "/vault"},
				cli.AutoWriteDeps{
					Stdin: strings.NewReader(`{"session":"` + sessionID + `","reply":` + reply + `}`),
					Learn: learnInto(written),
				}, &out)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(written).To(BeEmpty())
		})
	}
}

// learnInto is the learn path over an in-memory vault: it writes each note into the map, by its path, and returns the
// path as `engram learn` prints it.
func learnInto(written map[string]string) cli.LearnRun {
	return func(ctx context.Context, args cli.LearnArgs) (string, error) {
		deps := cli.LearnDeps{
			DetectRepo:    func(context.Context) string { return "" },
			DetectUser:    func(context.Context) string { return "" },
			Now:           func() time.Time { return time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC) },
			Getenv:        func(string) string { return "" },
			ListIDs:       func(string) ([]string, error) { return nil, nil },
			ListBasenames: func(string) ([]string, error) { return nil, nil },
			Lock:          func(string) (func(), error) { return func() {}, nil },
			WriteNew: func(path string, data []byte) error {
				written[path] = string(data)

				return nil
			},
		}

		var printed bytes.Buffer

		err := cli.RunLearn(ctx, args, deps, &printed)

		return strings.TrimSpace(printed.String()), err
	}
}

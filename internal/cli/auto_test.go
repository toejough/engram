package cli_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

func Test793d22b2_AutoRunIsATargetOfTheCommand(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	var stdout, stderr bytes.Buffer

	deps := newTestDeps(&stdout, &stderr)
	deps.Stdin = strings.NewReader(`{"kind":"summary","tally":[{"added":1}]}`)

	_, err := targ.Execute([]string{"engram", "auto", "run"}, cli.Targets(deps)...)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(stderr.String()).To(gomega.BeEmpty())
	g.Expect(stdout.String()).To(gomega.ContainSubstring("Memory activity: 1 note added"))
}

func Test793d22b2_AutoRunPrintsTheResponseOfAJob(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	out, err := autoRun(t, &autoCommander{}, `{"kind":"summary","tally":[{"delivered":2}]}`)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(out).To(gomega.ContainSubstring(`"line":"Memory activity: 2 items delivered"`))
}

func Test793d22b2_AutoRunRefusesAJobItCannotRead(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	_, err := autoRun(t, &autoCommander{}, `{"kind":`)

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("auto run")))
}

func Test793d22b3_AutoRunClassifiesTheFailureOfACommandByWhatItLeft(t *testing.T) {
	t.Parallel()

	failed := errors.New("exit status 1")
	missing := errors.Join(update.ErrCommandNotFound, errors.New("executable file not found"))
	cases := map[string]struct {
		commander *autoCommander
		want      string
	}{
		"a command that exits with an error": {
			&autoCommander{err: failed, stderr: "permission denied"}, "the vault is unavailable",
		},
		"a command that cannot start":        {&autoCommander{err: missing}, "the vault is unavailable"},
		"a command that fails for no reason": {&autoCommander{err: failed}, "of an error"},
	}

	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := gomega.NewWithT(t)
			out, err := autoRun(t, one.commander, `{"kind":"housekeeping","step":"the housekeeping"}`)

			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(out).To(gomega.ContainSubstring(one.want))
			g.Expect(one.commander.argv).To(gomega.Equal([]string{"engram", "ingest", "--auto"}))
		})
	}
}

func Test793d22b3_AutoRunCutsACommandAtItsDeadline(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	commander := &autoCommander{err: errors.New("signal: killed"), wait: true}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)

	defer cancel()

	out, err := cli.AutoCommandForTest(ctx, commander, []string{"engram", "ingest"}, time.Hour)

	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(out.Stdout).To(gomega.BeEmpty())
	g.Expect(err.Error()).To(gomega.ContainSubstring("still running after"))
}

func Test793d28b3_AutoEnvAndClockReadTheInjectedCapabilities(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	env := cli.NewAutoEnvForTest(func(key string) string { return "value of " + key })
	now := time.UnixMilli(5)

	g.Expect(env.Get("HOME")).To(gomega.Equal("value of HOME"))
	g.Expect(cli.NewAutoClockForTest(func() time.Time { return now }).Now()).To(gomega.Equal(now))
}

func Test793d28b3_AutoFilesReadWriteWalkAndTellWhatIsThere(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	root := t.TempDir()
	files := cli.NewAutoFilesForTest(newTestDeps(&bytes.Buffer{}, &bytes.Buffer{}).FS)

	g.Expect(files.Write(root+"/vault/sub/2.b.md", "x")).To(gomega.Succeed())
	g.Expect(files.Write(root+"/vault/1.a.md", "pending: true\n")).To(gomega.Succeed())

	text, err := files.Read(root + "/vault/1.a.md")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(text).To(gomega.Equal("pending: true\n"))

	_, err = files.Read(root + "/vault/missing.md")
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("no such file")))

	walked, err := files.Walk(root + "/vault")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(walked).To(gomega.Equal([]string{root + "/vault/1.a.md", root + "/vault/sub/2.b.md"}))

	_, err = files.Walk(root + "/nowhere")
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(files.Exists(root + "/vault")).To(gomega.BeTrue())
	g.Expect(files.Exists(root + "/nowhere")).To(gomega.BeFalse())
}

// The adapters of `engram auto run` (793e8m): the commands, the files, the environment and the clock a job runs in.

// autoCommander answers a command with what the test gives.
type autoCommander struct {
	stdout, stderr string
	err            error
	wait           bool
	argv           []string
}

func (c *autoCommander) Run(ctx context.Context, _, name string, args ...string) ([]byte, []byte, error) {
	c.argv = append([]string{name}, args...)

	if c.wait {
		<-ctx.Done()
	}

	return []byte(c.stdout), []byte(c.stderr), c.err
}

// autoRun runs the job `summary` with the given commander and returns what it printed.
func autoRun(t *testing.T, commander update.Commander, stdin string) (string, error) {
	t.Helper()

	var out bytes.Buffer

	deps := newTestDeps(&out, &bytes.Buffer{})
	deps.Stdin = strings.NewReader(stdin)
	deps.Commander = commander
	deps.Now = func() time.Time { return time.UnixMilli(1_700_000_000_000) }

	err := cli.RunAutoForTest(context.Background(), deps)

	return out.String(), err
}

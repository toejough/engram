package cli_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
)

func Test793d28b3_TheChunksDirectoryFollowsTheVariablesOfTheEngramCommands(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		env  map[string]string
		home string
		dir  string
	}{
		"ENGRAM_CHUNKS_DIR": {map[string]string{"ENGRAM_CHUNKS_DIR": "/custom/chunks"}, "/home/me", "/custom/chunks"},
		"XDG_DATA_HOME":     {map[string]string{"XDG_DATA_HOME": "/xdg"}, "/home/me", "/xdg/engram/chunks"},
		"the home, no variable": {
			map[string]string{}, "/home/me", "/home/me/.local/share/engram/chunks",
		},
	}

	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := gomega.NewWithT(t)
			_, chunks := cli.AutoDirectoriesForTest(one.home, one.env)

			g.Expect(chunks).To(gomega.Equal(one.dir))
		})
	}
}

func Test793d29b1_AutoOffersAreTheNotesThePendingOfferBookkeepingCallsPending(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	root := t.TempDir()
	deps := newTestDeps(&bytes.Buffer{}, &bytes.Buffer{})
	files := cli.NewAutoFilesForTest(deps.FS)
	pending := "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\npending: true\n---\n\nbody\n"
	live := strings.Replace(pending, "pending: true\n", "", 1)
	vocabulary := strings.Replace(pending, "type: fact", "type: vocab", 1)

	for name, text := range map[string]string{
		"2.2026-10-12.offer-b.md": pending, "1.2026-10-12.offer-a.md": pending, "3.2026-10-12.live.md": live,
		"4.2026-10-12.vocab.md": vocabulary,
	} {
		g.Expect(files.Write(root+"/vault/"+name, text)).To(gomega.Succeed())
	}

	g.Expect(files.Write(root+"/vault/sub/6.2026-10-12.nested.md", pending)).To(gomega.Succeed())

	names, err := cli.NewAutoOffersForTest(deps.FS).Pending(root + "/vault")

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(names).To(gomega.Equal([]string{"1.2026-10-12.offer-a.md", "2.2026-10-12.offer-b.md"}))

	_, err = cli.NewAutoOffersForTest(deps.FS).Pending(root + "/nowhere")
	g.Expect(err).To(gomega.HaveOccurred())
}

func Test793d29b1_TheVaultIsFoundUnderTheEngramDataDirectoryWhenNoPathIsGiven(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		env   map[string]string
		vault string
	}{
		"ENGRAM_VAULT_PATH": {map[string]string{"ENGRAM_VAULT_PATH": "/v"}, "/v"},
		"the data home":     {map[string]string{"XDG_DATA_HOME": "/data"}, "/data/engram/vault"},
		"the home":          {map[string]string{}, "/home/t/.local/share/engram/vault"},
	}

	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := gomega.NewWithT(t)
			vault, _ := cli.AutoDirectoriesForTest("/home/t", one.env)

			g.Expect(vault).To(gomega.Equal(one.vault))
		})
	}
}

func Test793d36b1_AutoClockReadsTheInjectedCapability(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	now := time.UnixMilli(5)

	g.Expect(cli.NewAutoClockForTest(func() time.Time { return now }).Now()).To(gomega.Equal(now))
}

func Test793d36b1_AutoFilesAreReadableByTheUserAlone(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	root := t.TempDir()
	files := cli.NewAutoFilesForTest(newTestDeps(&bytes.Buffer{}, &bytes.Buffer{}).FS)

	g.Expect(files.Write(root+"/chunks/backlog-marks.json", "[]")).To(gomega.Succeed())

	info, err := os.Stat(root + "/chunks/backlog-marks.json")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(info.Mode().Perm()).To(gomega.Equal(os.FileMode(0o600)))
}

func Test793d36b1_AutoFilesReadWriteAndTellWhatIsThere(t *testing.T) {
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

	g.Expect(files.Exists(root + "/vault")).To(gomega.BeTrue())
	g.Expect(files.Exists(root + "/nowhere")).To(gomega.BeFalse())
}

func Test793d36b1_AutoRunIsATargetOfTheCommand(t *testing.T) {
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

func Test793d36b1_AutoRunPrintsTheResponseOfAJob(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	out, err := autoRun(t, `{"kind":"summary","tally":[{"delivered":2}]}`)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(out).To(gomega.ContainSubstring(`"line":"Memory activity: 2 items delivered"`))
}

func Test793d36b1_AutoRunRefusesAJobItCannotRead(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	_, err := autoRun(t, `{"kind":`)

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("auto run")))
}

func Test793d36b4_AutoRunWritesTheLogInTheDataDirectoryForItsOwnerAlone(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	home := t.TempDir()

	var out bytes.Buffer

	deps := newTestDeps(&out, &bytes.Buffer{})
	deps.Stdin = strings.NewReader(`{"kind":"failure","step":"indexing","text":"boom","sessionId":"s1"}`)
	deps.Now = func() time.Time { return time.UnixMilli(1_700_000_000_000) }
	deps.UserHomeDir = func() (string, error) { return home, nil }
	deps.Getenv = func(string) string { return "" }

	g.Expect(cli.RunAutoForTest(context.Background(), deps)).To(gomega.Succeed())

	log := home + "/.local/share/engram/auto.log"
	text, err := os.ReadFile(log)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(string(text)).To(gomega.Equal(
		`2023-11-14T22:13:20Z session="s1" operation="indexing" class=error reason="boom"` + "\n"))
	g.Expect(out.String()).To(gomega.ContainSubstring(
		`Automatic memory skipped: indexing failed: boom; nothing lost, retried next turn `+
			`(details: ~/.local/share/engram/auto.log)`))

	info, err := os.Stat(log)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(info.Mode().Perm()).To(gomega.Equal(os.FileMode(0o600)))
}

// The adapters of `engram auto run` (793e8m): the commands, the files, the environment and the clock a job runs in.

// autoRun runs the job `summary` with the given commander and returns what it printed.
func autoRun(t *testing.T, stdin string) (string, error) {
	t.Helper()

	var out bytes.Buffer

	deps := newTestDeps(&out, &bytes.Buffer{})
	deps.Stdin = strings.NewReader(stdin)
	deps.Now = func() time.Time { return time.UnixMilli(1_700_000_000_000) }

	err := cli.RunAutoForTest(context.Background(), deps)

	return out.String(), err
}

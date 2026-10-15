package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/auto"
	"github.com/toejough/engram/internal/cli"
)

func Test793d24b4_AutoVaultAQueryWithTheChunkIndexReturnsItsChunksAsStubsAndOneWithoutLeavesThemOut(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	vault, deps := autoVaultWithNote(t, "")
	source := filepath.Join(t.TempDir(), "chunk-source.md")
	text := "# Heading\n\nthe staging credentials rotate monthly (CHUNK-TEXT).\n"
	g.Expect(os.WriteFile(source, []byte(text), 0o600)).To(gomega.Succeed())

	_, err := targ.Execute([]string{"engram", "ingest", "--markdown", source}, cli.Targets(deps)...)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	request := auto.QueryRequest{Text: "staging", Phrases: []string{"staging"}, Limit: 10}

	notesOnly, err := vault.Query(request, time.Minute)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(notesOnly).NotTo(gomega.ContainSubstring("kind: chunk"), "the chunk index was searched")

	request.WithChunks = true
	stubs, err := vault.Query(request, time.Minute)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(stubs).To(gomega.ContainSubstring("path: " + source + "#Heading"))
	g.Expect(stubs).To(gomega.ContainSubstring("kind: chunk"))
	g.Expect(stubs).NotTo(gomega.ContainSubstring("CHUNK-TEXT"), "a chunk came with its content, not as a stub")
}

func Test793d29b2_AutoVaultAnAmendCheckedAgainstTheNoteAsItWasReadIsRefusedWhenTheNoteChanged(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	vault, deps := autoVaultWithNote(t, "")
	file := learnedNote(t, deps)
	name := filepath.Base(file)

	read, err := os.ReadFile(file)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// the note as `engram show` prints it, less what is not the note: the form the engine keeps it in
	shown, err := vault.Show(name, time.Minute)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	printed, _, _ := strings.Cut(shown, "\n# outbound links")
	g.Expect(printed).To(gomega.HavePrefix("---\ntype: fact"))

	for _, asRead := range []string{string(read), printed} {
		amend := auto.AmendRequest{
			Target: name, ExpectNote: asRead, Provenance: "amended automatically from offer 9.x.md",
			Fields: [][2]string{{"object", "changed"}},
		}
		g.Expect(vault.Amend(amend, time.Minute)).To(gomega.Succeed())

		// the note has changed since it was read: the same version no longer verifies it, and nothing is written
		stale := auto.AmendRequest{Target: name, ExpectNote: asRead, Fields: [][2]string{{"object", "later"}}}
		g.Expect(vault.Amend(stale, time.Minute)).To(
			gomega.MatchError(gomega.ContainSubstring("changed since it was judged")))

		current, readErr := os.ReadFile(file)
		g.Expect(readErr).NotTo(gomega.HaveOccurred())
		g.Expect(string(current)).To(gomega.ContainSubstring("object: changed"))
		g.Expect(string(current)).To(gomega.ContainSubstring("source: test; amended automatically from offer 9.x.md"))

		restore := auto.AmendRequest{Target: name, Fields: [][2]string{{"object", "v1"}}}
		g.Expect(vault.Amend(restore, time.Minute)).To(gomega.Succeed())
	}
}

// The vault of a job (793b417): the code of the commands, called in the process.

func Test793d36b1_AutoVaultQueryReturnsThePayloadOfTheQueryCommand(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	vault, _ := autoVaultWithNote(t, "")

	payload, err := vault.Query(auto.QueryRequest{
		Text: "tagging a release", Phrases: []string{"tagging a release"}, Limit: 5,
	}, time.Minute)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(payload).To(gomega.HavePrefix("version: 1"))
	g.Expect(payload).To(gomega.ContainSubstring("tagging a release"))
}

func Test793d36b3_AutoVaultAFailureCarriesWhatTheCodeOfTheCommandSaid(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	vault, deps := autoVaultWithNote(t, "")
	home := deps.Getenv("ENGRAM_VAULT_PATH")

	g.Expect(os.Chmod(home, 0o000)).To(gomega.Succeed())
	t.Cleanup(func() { _ = os.Chmod(home, 0o755) })

	_, err := vault.Query(auto.QueryRequest{Text: "x", Phrases: []string{"x"}, Limit: 5}, time.Minute)

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("permission denied")))
}

func Test793d36b3_AutoVaultAnOperationPastItsDeadlineIsCutAndSaysItWasStillRunning(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	vault, _ := autoVaultWithNote(t, "hang")

	_, err := vault.Query(auto.QueryRequest{Text: "x", Phrases: []string{"x"}, Limit: 5}, 20*time.Millisecond)

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("still running after")))
}

func Test793d36b3_AutoVaultTheHarnessFaultDelayWaitsOnlyBeforeTheOperationsThatAreNotAQuery(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	vault, deps := autoVaultWithNote(t, "delay:0")
	log := deps.AutoFaultLog

	_, queryErr := vault.Query(auto.QueryRequest{Text: "x", Phrases: []string{"x"}, Limit: 5}, time.Minute)
	g.Expect(queryErr).NotTo(gomega.HaveOccurred())

	_, noteErr := os.ReadFile(log)
	g.Expect(noteErr).To(gomega.HaveOccurred(), "a query was delayed")

	g.Expect(vault.Index(time.Minute)).To(gomega.Succeed())

	_, showErr := vault.Show("1.2026-01-01.nothing.md", time.Minute)
	g.Expect(showErr).To(gomega.MatchError(gomega.ContainSubstring("note not found")))

	noted, err := os.ReadFile(log)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(strings.Split(strings.TrimSpace(string(noted)), "\n")).To(gomega.HaveLen(2))
	g.Expect(string(noted)).To(gomega.MatchRegexp(`(?m)^\d+ slept-start ingest$`))
	g.Expect(string(noted)).To(gomega.MatchRegexp(`(?m)^\d+ slept-start show$`))
}

func Test793d36b3_AutoVaultTheHarnessFaultErrorFailsEveryOperation(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	vault, _ := autoVaultWithNote(t, "error")

	_, queryErr := vault.Query(auto.QueryRequest{Text: "x", Phrases: []string{"x"}, Limit: 5}, time.Minute)

	g.Expect(queryErr).To(gomega.MatchError(gomega.ContainSubstring("injected failure")))
	g.Expect(vault.Index(time.Minute)).To(gomega.MatchError(gomega.ContainSubstring("injected failure")))
}

// autoVaultWithNote is a vault of a job on a temporary directory, holding one fact note, with the fault given.
func autoVaultWithNote(t *testing.T, fault string) (auto.Vault, cli.Deps) {
	t.Helper()

	root := t.TempDir()
	env := map[string]string{
		"ENGRAM_VAULT_PATH": filepath.Join(root, "vault"), "ENGRAM_CHUNKS_DIR": filepath.Join(root, "chunks"),
		"ENGRAM_VAULT_NAME": "personal", "HOME": root,
	}
	deps := newTestDeps(&strings.Builder{}, &strings.Builder{})
	deps.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
	// the sweep of the index looks under the home and the working directory: here, nothing of the user's
	deps.UserHomeDir = func() (string, error) { return root, nil }
	deps.Getwd = func() (string, error) { return root, nil }
	deps.Getenv = func(key string) string { return env[key] }
	deps.AutoFaultLog = filepath.Join(root, "fault.log")

	_, err := cli.NewAutoVaultForTest(context.Background(), deps).Learn(auto.LearnRequest{
		Kind: "fact", Slug: "a-note", Source: "test", Situation: "tagging a release",
		Values: [][2]string{{"subject", "x"}, {"predicate", "is"}, {"object", "v1"}},
	}, time.Minute)

	if err != nil {
		t.Fatalf("seeding the note: %v", err)
	}

	// the fault stands only after the note is there
	deps.AutoFault = fault

	return cli.NewAutoVaultForTest(context.Background(), deps), deps
}

// learnedNote is the path of the one note of the vault.
func learnedNote(t *testing.T, deps cli.Deps) string {
	t.Helper()

	notes, err := filepath.Glob(filepath.Join(deps.Getenv("ENGRAM_VAULT_PATH"), "*.a-note.md"))
	if err != nil || len(notes) != 1 {
		t.Fatalf("the note is not there: %v %v", notes, err)
	}

	return notes[0]
}

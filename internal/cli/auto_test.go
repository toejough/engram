package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

func Test793d4a1a_RunAutoDeliver_PrintsTheNotesThatBearOnThePromptAsOneBlock(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.2026-10-17.zorblax-staging-reset.md", zorblaxResetNote)

	var out bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: vault},
		cli.AutoDeliverDeps{
			Stdin: strings.NewReader(`{"text":"When is the Zorblax staging database reset?"}`),
			Query: newQueryDeps(memFS),
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")

	g.Expect(lines[0]).To(HavePrefix("[engram memory]"))
	g.Expect(lines).To(ContainElement("- 1.2026-10-17.zorblax-staging-reset.md " +
		"(testing against the Zorblax staging database): " +
		"Information learned: when in testing against the Zorblax staging database, " +
		"the Zorblax staging database is reset every Sunday at 03:40 UTC."))
}

func Test793d4a1b_RunAutoDeliver_AFailedLookupFailsTheCommandAndPrintsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	query := newQueryDeps(newInMemoryFS())
	query.Embedder = errorEmbedder{}

	var out bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: t.TempDir()},
		cli.AutoDeliverDeps{Stdin: strings.NewReader(`{"text":"anything"}`), Query: query}, &out)

	g.Expect(err).To(MatchError(ContainSubstring("embedder down")))
	g.Expect(out.String()).To(BeEmpty())
}

func Test793d4a1e_RunAutoDeliver_NoNoteToDeliverPrintsNothing(t *testing.T) {
	t.Parallel()

	t.Run("no note bears on the prompt", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var out bytes.Buffer

		err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: t.TempDir()},
			cli.AutoDeliverDeps{
				Stdin: strings.NewReader(`{"text":"anything","session":"` + sessionID + `"}`),
				Query: newQueryDeps(newInMemoryFS()), ReadDir: os.ReadDir, ReadFile: os.ReadFile,
			}, &out)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out.String()).To(BeEmpty())
	})

	t.Run("the only note that bears on it is held", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := t.TempDir()
		memFS := newInMemoryFS()
		plantNoteWithSidecar(t, memFS, vault, "1.2026-10-17.zorblax-staging-reset.md", zorblaxResetNote)

		sessions := t.TempDir()
		record := `{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
			`"content":["[engram memory] The lines below are notes.\n- 1.2026-10-17.zorblax-staging-reset.md (s): b"]}}` + "\n"
		g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

		var out bytes.Buffer

		err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: vault, SessionsDir: sessions},
			cli.AutoDeliverDeps{
				Stdin: strings.NewReader(`{"text":"When is the Zorblax staging database reset?","session":"` + sessionID + `"}`),
				Query: newQueryDeps(memFS), ReadDir: os.ReadDir, ReadFile: os.ReadFile,
			}, &out)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out.String()).To(BeEmpty())
	})
}

func Test793d4a1d_RunAutoDeliver_ANoteTheSessionHoldsIsLeftOut(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.2026-10-17.zorblax-staging-reset.md", zorblaxResetNote)
	plantNoteWithSidecar(t, memFS, vault, "2.2026-10-17.zorblax-backup-window.md",
		strings.NewReplacer("reset", "backed up", "1.", "2.").Replace(zorblaxResetNote))

	sessions := t.TempDir()
	record := `{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
		`"content":["[engram memory] The lines below are notes.\n- 1.2026-10-17.zorblax-staging-reset.md (s): b"]}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: vault, SessionsDir: sessions},
		cli.AutoDeliverDeps{
			Stdin:    strings.NewReader(`{"text":"When is the Zorblax staging database reset?","session":"` + sessionID + `"}`),
			Query:    newQueryDeps(memFS),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(out.String()).To(ContainSubstring("2.2026-10-17.zorblax-backup-window.md"))
	g.Expect(out.String()).NotTo(ContainSubstring("1.2026-10-17.zorblax-staging-reset.md"))
}

func Test793d4a1c_NotesOf_ANoteOnlyTheExploreHalfPickedDoesNotBearOnThePrompt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes, err := cli.ExportNotesOf([]byte(`version: 1
phrases:
  - When is the Zorblax staging database reset?
items:
  - path: 1.2026-10-17.zorblax-staging-reset.md
    kind: fact
    score: 0.92
    provenances:
      - direct
    content: |
      ---
      type: fact
      situation: testing against the Zorblax staging database
      ---

      The Zorblax staging database is reset every Sunday at 03:40 UTC.
  - path: 2.2026-10-17.format-python-before-commit.md
    kind: feedback
    score: 0.41
    provenances:
      - explore
    source_term: formatting
    content: |
      ---
      type: feedback
      situation: committing a Python file
      ---

      Run black on the file before committing.
clusters: []
budget: {}
model_id: m@4
`))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(notes).To(HaveLen(1))

	if len(notes) == 1 {
		g.Expect(notes[0].Path).To(Equal("1.2026-10-17.zorblax-staging-reset.md"))
	}
}

func Test793d4d1b_HeldText_TheRecordIsFoundUnderItsProjectDirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	project := filepath.Join(sessions, "-home-joe-a-project")
	g.Expect(os.MkdirAll(project, 0o750)).To(Succeed())

	record := `{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
		`"content":["[engram memory] the block of a session"]}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(project, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(project, "another-session.jsonl"),
		[]byte(strings.ReplaceAll(record, "the block of a session", "the block of another")), 0o600)).To(Succeed())

	held, err := cli.ExportHeldText(cli.AutoDeliverDeps{ReadDir: os.ReadDir, ReadFile: os.ReadFile}, sessions, sessionID)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(held).To(ContainSubstring("the block of a session"))
	g.Expect(held).NotTo(ContainSubstring("the block of another"))
}

func Test793d4d1c_HeldText_ASessionWithNoRecordOrAnIDThatIsNotAFileNameHoldsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	parent := t.TempDir()
	sessions := filepath.Join(parent, "sessions")
	g.Expect(os.MkdirAll(sessions, 0o750)).To(Succeed())

	// a record outside the session-log directory, which an ID that leaves the directory would reach
	record := `{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
		`"content":["[engram memory] a block outside the directory"]}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(parent, "outside.jsonl"), []byte(record), 0o600)).To(Succeed())

	deps := cli.AutoDeliverDeps{ReadDir: os.ReadDir, ReadFile: os.ReadFile}

	for _, id := range []string{sessionID, "../outside", "nested/" + sessionID, "..", "."} {
		held, err := cli.ExportHeldText(deps, sessions, id)
		g.Expect(err).NotTo(HaveOccurred(), id)
		g.Expect(held).To(BeEmpty(), id)
	}

	held, err := cli.ExportHeldText(deps, filepath.Join(parent, "no-such-directory"), sessionID)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(held).To(BeEmpty())
}

// TestTargets_AutoDeliver reaches the command the way the mod does, as `engram auto deliver` with the prompt on
// standard input, so the wiring of the group, the flag and the dependencies is run.
func TestTargets_AutoDeliverPrintsTheNotesThatBearOnThePromptOnStandardInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	plantNoteOnDisk(t, vault, "1.2026-10-17.zorblax-staging-reset.md", zorblaxResetNote)

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t, []string{"engram", "auto", "deliver", "--vault", vault}, func(d *cli.Deps) {
		d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
		d.Stdin = strings.NewReader(`{"text":"When is the Zorblax staging database reset?"}`)
		d.Stdout = &stdout
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout.String()).To(ContainSubstring("- 1.2026-10-17.zorblax-staging-reset.md (testing against"))
}

// unexported constants.
const (
	// sessionID is the ID of a session in the tests.
	sessionID        = "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11"
	zorblaxResetNote = "---\ntype: fact\ntier: L2\nsituation: testing against the Zorblax staging database\n" +
		"subject: the Zorblax staging database\npredicate: is reset every\nobject: Sunday at 03:40 UTC\n" +
		"luhmann: \"1\"\ncreated: \"2026-10-17\"\nsource: test\nuser: u\nvault: personal\n---\n\n" +
		"Information learned: when in testing against the Zorblax staging database, " +
		"the Zorblax staging database is reset every Sunday at 03:40 UTC.\n"
)

// plantNoteOnDisk writes a note and its embedding sidecar into a vault directory.
func plantNoteOnDisk(t *testing.T, vault, name, body string) {
	t.Helper()

	sidecar, err := embed.BuildSidecar(context.Background(), stubEmbedder{modelID: "m@4", dims: 4}, []byte(body))
	if err != nil {
		t.Fatalf("building the sidecar: %v", err)
	}

	for path, content := range map[string][]byte{
		filepath.Join(vault, name):                    []byte(body),
		filepath.Join(vault, embed.SidecarPath(name)): embed.MarshalSidecar(sidecar),
	} {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
}

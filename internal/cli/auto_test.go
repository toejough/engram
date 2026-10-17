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

func Test793d4a1f_RunAutoDeliver_TheChunkIndexIsNotSearched(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.2026-10-17.zorblax-staging-reset.md", zorblaxResetNote)

	query := newQueryDeps(memFS)
	query.ListChunkIndexes = func(string) ([]string, error) {
		t.Error("the chunk index was listed")

		return nil, nil
	}

	var out bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: vault},
		cli.AutoDeliverDeps{
			Stdin: strings.NewReader(`{"text":"When is the Zorblax staging database reset?"}`), Query: query,
		}, &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("1.2026-10-17.zorblax-staging-reset.md"))
}

func Test793d4a1g_RunAutoDeliver_ANoteTheAgentFetchedByItsBasenameIsLeftOut(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.2026-10-17.zorblax-staging-reset.md", zorblaxResetNote)
	plantNoteWithSidecar(t, memFS, vault, "2.2026-10-17.zorblax-backup-window.md",
		strings.NewReplacer("reset", "backed up", "1.", "2.").Replace(zorblaxResetNote))

	// engram show prints the note without its file name, so only the command names it, and by its basename
	sessions := t.TempDir()
	record := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Bash",` +
		`"input":{"command":"engram show 1.2026-10-17.zorblax-staging-reset"}}]}}` + "\n" +
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1",` +
		`"content":"---\ntype: fact\n---\n\nThe Zorblax staging database is reset every Sunday at 03:40 UTC.\n"}]}}` + "\n"
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

func Test793d4d1b_HeldText_TheRecordIsFoundUnderItsProjectDirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	project := filepath.Join(sessions, "-home-joe-a-project")
	g.Expect(os.MkdirAll(project, 0o750)).To(Succeed())

	record := `{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
		`"content":["[engram memory] The lines below are notes.\n- 1.2026-10-17.of-a-session.md (s): b"]}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(project, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(project, "another-session.jsonl"),
		[]byte(strings.ReplaceAll(record, "of-a-session", "of-another")), 0o600)).To(Succeed())

	held := cli.ExportHeldText(cli.AutoDeliverDeps{ReadDir: os.ReadDir, ReadFile: os.ReadFile}, sessions, sessionID)
	g.Expect(held).To(ContainSubstring("1.2026-10-17.of-a-session.md"))
	g.Expect(held).NotTo(ContainSubstring("of-another"))
}

func Test793d4d1c_HeldText_ASessionWithNoRecordOrAnIDThatIsNotAFileNameHoldsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	parent := t.TempDir()
	sessions := filepath.Join(parent, "sessions")
	g.Expect(os.MkdirAll(sessions, 0o750)).To(Succeed())

	// a record outside the session-log directory, which an ID that leaves the directory would reach
	record := `{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
		`"content":["[engram memory] The lines below are notes.\n- 1.2026-10-17.outside-the-directory.md (s): b"]}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(parent, "outside.jsonl"), []byte(record), 0o600)).To(Succeed())

	deps := cli.AutoDeliverDeps{ReadDir: os.ReadDir, ReadFile: os.ReadFile}

	for _, id := range []string{sessionID, "../outside", "nested/" + sessionID, "..", "."} {
		g.Expect(cli.ExportHeldText(deps, sessions, id)).To(BeEmpty(), id)
	}

	g.Expect(cli.ExportHeldText(deps, filepath.Join(parent, "no-such-directory"), sessionID)).To(BeEmpty())
}

func Test793d4f1a_Targets_ENGRAMTRANSCRIPTDIRNamesTheRecordsFirst(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	transcripts, config, home := t.TempDir(), t.TempDir(), t.TempDir()
	vault := plantTwoNotes(t)

	plantRecord(t, transcripts, resetNoteName)
	plantRecord(t, filepath.Join(config, "projects", "-a-project"), backupNoteName)

	out := deliverBy(t, vault, home, map[string]string{"ENGRAM_TRANSCRIPT_DIR": transcripts, "CLAUDE_CONFIG_DIR": config})

	g.Expect(out).To(ContainSubstring(backupNoteName))
	g.Expect(out).NotTo(ContainSubstring(resetNoteName))
}

func Test793d4f1b_Targets_CLAUDECONFIGDIRNamesTheRecordsWhenENGRAMTRANSCRIPTDIRDoesNot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	config, home := t.TempDir(), t.TempDir()
	vault := plantTwoNotes(t)

	plantRecord(t, filepath.Join(config, "projects", "-a-project"), resetNoteName)
	plantRecord(t, filepath.Join(home, ".claude", "projects", "-a-project"), backupNoteName)

	out := deliverBy(t, vault, home, map[string]string{"CLAUDE_CONFIG_DIR": config})

	g.Expect(out).To(ContainSubstring(backupNoteName))
	g.Expect(out).NotTo(ContainSubstring(resetNoteName))
}

func Test793d4f1c_Targets_TheHomesClaudeProjectsHoldTheRecordsByDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	home := t.TempDir()
	vault := plantTwoNotes(t)

	plantRecord(t, filepath.Join(home, ".claude", "projects", "-a-project"), resetNoteName)

	out := deliverBy(t, vault, home, nil)

	g.Expect(out).To(ContainSubstring(backupNoteName))
	g.Expect(out).NotTo(ContainSubstring(resetNoteName))
}

func Test793d4e1a_Targets_AutoDeliverCreatesNoVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "no-such-vault")

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t, []string{"engram", "auto", "deliver", "--vault", vault}, func(d *cli.Deps) {
		d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
		d.Stdin = strings.NewReader(`{"text":"anything"}`)
		d.Stdout = &stdout
	})

	_, statErr := os.Stat(vault)

	g.Expect(os.IsNotExist(statErr)).To(BeTrue(), "no vault was created at "+vault)
	g.Expect(stdout.String()).To(BeEmpty())
	g.Expect(stderr).NotTo(ContainSubstring("created new vault"))
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
	resetNoteName    = "1.2026-10-17.zorblax-staging-reset.md"
	backupNoteName   = "2.2026-10-17.zorblax-backup-window.md"
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

// deliverBy dispatches `engram auto deliver` for a prompt of the session, without --sessions, with the environment and
// the home that are given and nothing else of the machine's, and returns what it prints.
func deliverBy(t *testing.T, vault, home string, environment map[string]string) string {
	t.Helper()

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t, []string{"engram", "auto", "deliver", "--vault", vault}, func(d *cli.Deps) {
		d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
		d.Getenv = func(name string) string { return environment[name] }
		d.UserHomeDir = func() (string, error) { return home, nil }
		d.Stdin = strings.NewReader(`{"text":"When is the Zorblax staging database reset?","session":"` + sessionID + `"}`)
		d.Stdout = &stdout
	})
	if stderr != "" {
		t.Fatalf("the dispatch printed to standard error: %s", stderr)
	}

	return stdout.String()
}

// plantRecord writes the record of the session in a directory: a block delivered beside a prompt, whose one note line
// names the note.
func plantRecord(t *testing.T, dir, note string) {
	t.Helper()

	mkdirErr := os.MkdirAll(dir, 0o750)
	if mkdirErr != nil {
		t.Fatalf("creating %s: %v", dir, mkdirErr)
	}

	record := `{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
		`"content":["[engram memory] The lines below are notes.\n- ` + note + ` (s): b"]}}` + "\n"

	writeErr := os.WriteFile(filepath.Join(dir, sessionID+".jsonl"), []byte(record), 0o600)
	if writeErr != nil {
		t.Fatalf("writing the record: %v", writeErr)
	}
}

// plantTwoNotes makes a vault, on disk, whose two notes both bear on the prompt about the Zorblax staging database.
func plantTwoNotes(t *testing.T) string {
	t.Helper()

	vault := t.TempDir()

	plantNoteOnDisk(t, vault, resetNoteName, zorblaxResetNote)
	plantNoteOnDisk(t, vault, backupNoteName, strings.NewReplacer("reset", "backed up", "1.", "2.").Replace(zorblaxResetNote))

	return vault
}

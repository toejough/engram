package cli_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

//go:generate impgen cli.QueryRun --dependency --import-path github.com/toejough/engram/internal/cli

func Test793d4a1a_RunAutoDeliver_PrintsTheNotesThatBearOnThePromptAsOneBlock(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var out bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: t.TempDir()},
		cli.AutoDeliverDeps{
			Stdin:    deliveryRequest(sessionID, resetCandidate(), freezePassage()),
			Activate: noActivation,
		}, &out, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")

	g.Expect(lines).To(HaveLen(3), "the framing line, the note and the passage")
	g.Expect(lines[0]).To(HavePrefix("[engram memory]"))
	g.Expect(lines).To(ContainElement("- 1.2026-10-17.zorblax-staging-reset.md " +
		"(testing against the Zorblax staging database): " +
		"Information learned: when in testing against the Zorblax staging database, " +
		"the Zorblax staging database is reset every Sunday at 03:40 UTC."))
	g.Expect(lines).To(ContainElement("- /chunks/an-earlier-session.jsonl#c3 (passage): " +
		"the user said the production cutover is frozen on the last Friday of every month"))
}

func Test793d4a1b_RunAutoDeliver_AReplyThatIsNotAReplyFailsTheCommandAndPrintsNothing(t *testing.T) {
	t.Parallel()

	// a word that no redaction by shape takes for a secret, in the prompt the delivery is for
	const prompt = "how do I deploy zorblax staging, the db pass hunter2-abc"

	for name, reply := range map[string]string{
		"text that is not JSON":        "I think the first note applies.",
		"a JSON array":                 `[{"id":"1.2026-10-17.zorblax-staging-reset.md","verdict":"covered"}]`,
		"an object with no candidates": `{"verdicts":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			var out, shown bytes.Buffer

			request := deliveryRequestWith(sessionID, reply, resetCandidate())
			request["text"] = prompt

			err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: t.TempDir()},
				cli.AutoDeliverDeps{Stdin: requestJSON(request), Activate: noActivation}, &out, &shown)

			g.Expect(err).To(MatchError(And(ContainSubstring("candidate judgement"), Not(ContainSubstring("hunter2-abc")),
				Not(ContainSubstring("zorblax staging")))), "the judgement, and none of the prompt")
			g.Expect(out.String()).To(BeEmpty())
			g.Expect(shown.String()).To(BeEmpty())
		})
	}
}

func Test793d4a1d_RunAutoDeliver_ANoteTheSessionHoldsIsLeftOut(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	record := `{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
		`"content":["[engram memory] The lines below are notes.\n- 1.2026-10-17.zorblax-staging-reset.md (s): b"]}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
		cli.AutoDeliverDeps{
			Stdin:    deliveryRequest(sessionID, resetCandidate(), backupCandidate()),
			Activate: noActivation,
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(out.String()).To(ContainSubstring("2.2026-10-17.zorblax-backup-window.md"))
	g.Expect(out.String()).NotTo(ContainSubstring("1.2026-10-17.zorblax-staging-reset.md"))
}

func Test793d4a1e_RunAutoDeliver_NoNoteToDeliverPrintsNothing(t *testing.T) {
	t.Parallel()

	sessions := t.TempDir()
	record := `{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
		`"content":["[engram memory] The lines below are notes.\n- 1.2026-10-17.zorblax-staging-reset.md (s): b"]}}` + "\n"
	g := NewWithT(t)
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	for name, request := range map[string]map[string]any{
		"every verdict is unrelated": deliveryRequestWith(sessionID,
			`{"candidates":[{"id":"1.2026-10-17.zorblax-staging-reset.md","verdict":"unrelated"}]}`, resetCandidate()),
		"the only note delivered is held": deliveryFields(sessionID, resetCandidate()),
		"no reply":                        withoutReply(deliveryFields(sessionID, resetCandidate())),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			var out bytes.Buffer

			err := cli.RunAutoDeliver(context.Background(),
				cli.AutoDeliverArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
				cli.AutoDeliverDeps{
					Stdin: requestJSON(request), Activate: noActivation, ReadDir: os.ReadDir, ReadFile: os.ReadFile,
				}, &out, io.Discard)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(out.String()).To(BeEmpty())
		})
	}
}

func Test793d4a1g_RunAutoDeliver_ANoteTheAgentFetchedByItsBasenameIsLeftOut(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// engram show prints the note without its file name, so only the command names it, and by its basename
	sessions := t.TempDir()
	record := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Bash",` +
		`"input":{"command":"engram show 1.2026-10-17.zorblax-staging-reset"}}]}}` + "\n" +
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1",` +
		`"content":"---\ntype: fact\n---\n\nThe Zorblax staging database is reset every Sunday at 03:40 UTC.\n"}]}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
		cli.AutoDeliverDeps{
			Stdin:    deliveryRequest(sessionID, resetCandidate(), backupCandidate()),
			Activate: noActivation,
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(out.String()).To(ContainSubstring("2.2026-10-17.zorblax-backup-window.md"))
	g.Expect(out.String()).NotTo(ContainSubstring("1.2026-10-17.zorblax-staging-reset.md"))
}

func Test793d4c1c_RunAutoDeliver_ALineSaysWhenItsNoteWasWrittenAutomatically(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		source string
		marked bool
	}{
		"a note written automatically": {source: "written automatically (session " + sessionID + ")", marked: true},
		"a note the earlier mod wrote": {
			source: "captured from the conversation; written automatically (session " + sessionID + ", turn 3)", marked: true,
		},
		"a note the agent saved": {source: "agent"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			note := resetCandidate()
			note.Source = test.source

			var out bytes.Buffer

			err := cli.RunAutoDeliver(context.Background(), cli.AutoDeliverArgs{VaultPath: t.TempDir()},
				cli.AutoDeliverDeps{Stdin: deliveryRequest("", note), Activate: noActivation}, &out, io.Discard)
			g.Expect(err).NotTo(HaveOccurred())

			lines := strings.Split(strings.TrimSpace(out.String()), "\n")
			g.Expect(lines).To(HaveLen(2), "the framing line and the line of the note")

			if len(lines) != 2 {
				return
			}

			if test.marked {
				g.Expect(lines[1]).To(HaveSuffix("the Zorblax staging database is reset every Sunday at 03:40 UTC. " +
					"(written automatically)"))
			} else {
				g.Expect(lines[1]).NotTo(ContainSubstring("written automatically"))
			}
		})
	}
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

func Test793d4e1a_Targets_AutoDeliverCreatesNoVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "no-such-vault")
	sessions, home := t.TempDir(), t.TempDir()

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t,
		[]string{"engram", "auto", "deliver", "--vault", vault, "--sessions", sessions},
		func(d *cli.Deps) {
			d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
			d.Getenv = func(string) string { return "" }
			d.UserHomeDir = func() (string, error) { return home, nil }
			d.Stdin = requestJSON(withoutReply(deliveryFields("", resetCandidate())))
			d.Stdout = &stdout
		})

	_, statErr := os.Stat(vault)

	g.Expect(os.IsNotExist(statErr)).To(BeTrue(), "no vault was created at "+vault)
	g.Expect(stdout.String()).To(BeEmpty())
	g.Expect(stderr).NotTo(ContainSubstring("created new vault"))
}

func Test793d4e1b_Targets_AutoJudgeCreatesNoVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "no-such-vault")
	sessions, home := t.TempDir(), t.TempDir()
	record := `{"type":"user","message":{"role":"user","content":"No - build with make zorblax-build."}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t,
		[]string{"engram", "auto", "judge", "--vault", vault, "--sessions", sessions},
		func(d *cli.Deps) {
			d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
			d.Getenv = func(string) string { return "" }
			d.UserHomeDir = func() (string, error) { return home, nil }
			d.Stdin = strings.NewReader(`{"origin":"composer","session":"` + sessionID +
				`","reply":"{\"learn\":true}","answer":"Understood."}`)
			d.Stdout = &stdout
		})

	_, statErr := os.Stat(vault)

	g.Expect(os.IsNotExist(statErr)).To(BeTrue(), "no vault was created at "+vault)
	g.Expect(stdout.String()).To(BeEmpty(), "the judgement printed no request")
	g.Expect(stderr).To(ContainSubstring("no vault at "+vault), "the judgement failed, naming the path")
	g.Expect(stderr).NotTo(ContainSubstring("created new vault"))
}

func Test793d4e1c_Targets_AutoWriteCreatesNoVaultAndWritesNoNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "no-such-vault")
	records, home := t.TempDir(), t.TempDir()

	var stdout bytes.Buffer

	stderr := executeForTestWithDeps(t,
		[]string{"engram", "auto", "write", "--vault", vault, "--records", records},
		func(d *cli.Deps) {
			d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
			d.Getenv = func(string) string { return "" }
			d.UserHomeDir = func() (string, error) { return home, nil }
			d.Stdin = strings.NewReader(`{"session":"` + sessionID + `","turn":"t","reply":"[{\"kind\":\"fact\",` +
				`\"slug\":\"staging-reset\",\"situation\":\"testing on the staging database\",\"covered\":null,` +
				`\"subject\":\"the staging database\",\"predicate\":\"is reset every\",\"object\":\"Sunday night\"}]"}`)
			d.Stdout = &stdout
		})

	_, statErr := os.Stat(vault)

	recorded, readErr := os.ReadDir(filepath.Join(records, sessionID))

	g.Expect(os.IsNotExist(statErr)).To(BeTrue(), "no vault was created at "+vault)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(recorded).NotTo(BeEmpty(), "the writing ran and recorded what became of the lesson")
	g.Expect(stdout.String()).To(BeEmpty(), "no path of a note was printed")
	g.Expect(stderr).NotTo(ContainSubstring("created new vault"))
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

func Test793d4f1d_Targets_NoRecordIsReadWhenNoDirectoryCanBeFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := plantTwoNotes(t)

	var (
		stdout bytes.Buffer
		reads  []string
	)

	stderr := executeForTestWithDeps(t, []string{"engram", "auto", "deliver", "--vault", vault}, func(d *cli.Deps) {
		d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
		d.Getenv = func(string) string { return "" }
		d.UserHomeDir = func() (string, error) { return "", errNoHome }
		d.Stdin = deliveryRequest(sessionID, resetCandidate(), backupCandidate())
		d.Stdout = &stdout
		d.FS = recordsReadFS{EdgeFS: d.FS, reads: &reads}
	})

	g.Expect(notOnlyDelivered(stderr)).To(BeEmpty(), "what the delivery printed to standard error, but the note lines")
	g.Expect(reads).To(BeEmpty(), "records read")
	g.Expect(stdout.String()).To(ContainSubstring(resetNoteName))
	g.Expect(stdout.String()).To(ContainSubstring(backupNoteName))
}

func Test793d4m1c_Targets_TheTurnOfAHeadlessPromptIsJudgedOnlyWhenTheUserAsks(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		flags  []string
		judged bool
	}{
		"without the flag": {},
		"with the flag":    {flags: []string{"--capture-sdk"}, judged: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			home := t.TempDir()
			sessions := recordOfAPrompt(t)

			var stdout bytes.Buffer

			stderr := executeForTestWithDeps(t,
				slices.Concat([]string{"engram", "auto", "moment", "--sessions", sessions}, test.flags),
				func(d *cli.Deps) {
					d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
					d.Getenv = func(string) string { return "" }
					d.UserHomeDir = func() (string, error) { return home, nil }
					d.Stdin = momentOf("sdk", "turn-end")
					d.Stdout = &stdout
				})

			g.Expect(stderr).To(BeEmpty())

			if test.judged {
				g.Expect(stdout.String()).To(HavePrefix(`{"request":`), "the request for the model")
			} else {
				g.Expect(stdout.String()).To(BeEmpty(), "the turn of a headless prompt is judged nothing")
			}

			// the headless prompt itself is judged with the cue prompt either way
			var prompted bytes.Buffer

			executeForTestWithDeps(t, slices.Concat([]string{"engram", "auto", "moment", "--sessions", sessions}, test.flags),
				func(d *cli.Deps) {
					d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
					d.Getenv = func(string) string { return "" }
					d.UserHomeDir = func() (string, error) { return home, nil }
					d.Stdin = momentOf("sdk", "prompt")
					d.Stdout = &prompted
				})

			g.Expect(prompted.String()).To(HavePrefix(`{"request":`), "the headless prompt is judged either way")
		})
	}
}

func Test855d4c1d_Targets_AutoDeliverMarksTheNotesItDeliversAsUsed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault, home := t.TempDir(), t.TempDir()
	plantNoteOnDisk(t, vault, resetNoteName, zorblaxResetNote)
	plantNoteOnDisk(t, vault, backupNoteName, strings.NewReplacer("reset", "backed up", "1.", "2.").
		Replace(zorblaxResetNote))

	var stdout bytes.Buffer

	executeForTestWithDeps(t,
		[]string{"engram", "auto", "deliver", "--vault", vault, "--records", t.TempDir()},
		func(d *cli.Deps) {
			d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
			d.Getenv = func(string) string { return "" }
			d.UserHomeDir = func() (string, error) { return home, nil }
			d.Stdin = deliveryRequest(sessionID, resetCandidate())
			d.Stdout = &stdout
		})

	g.Expect(stdout.String()).To(ContainSubstring(resetNoteName), "the block holds the note")
	g.Expect(lastUsed(t, vault, resetNoteName)).NotTo(BeEmpty(), "the note delivered was marked as used")
	g.Expect(lastUsed(t, vault, backupNoteName)).To(BeEmpty(), "a note that was not delivered was not")
}

func Test855d4c1f_Deliver_TheDeliveryLooksNothingUp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions, records, vault, chunks := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()

	var touched []string

	var out bytes.Buffer

	err := cli.RunAutoDeliver(context.Background(),
		cli.AutoDeliverArgs{VaultPath: vault, SessionsDir: sessions, RecordsDir: records},
		cli.AutoDeliverDeps{
			Stdin:       deliveryRequest(sessionID, resetCandidate(), freezePassage()),
			Activate:    noActivation,
			WriteRecord: writeFileExclusively,
			Remove:      os.Remove,
			ReadDir: func(path string) ([]fs.DirEntry, error) {
				touched = append(touched, path)

				return os.ReadDir(path)
			},
			ReadFile: func(path string) ([]byte, error) {
				touched = append(touched, path)

				return os.ReadFile(path)
			},
		}, &out, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring(resetNoteName), "the note of the carry is delivered")

	for _, path := range touched {
		g.Expect(path).NotTo(HavePrefix(vault), "the vault was read")
		g.Expect(path).NotTo(HavePrefix(chunks), "the chunk index was read")
	}
}

// unexported constants.
const (
	backupNoteName = "2.2026-10-17.zorblax-backup-window.md"
	// mockTimeout is how long a test waits for the code under it before it fails, so that an expectation that is never
	// met does not hold the run.
	mockTimeout   = 5 * time.Second
	resetNoteName = "1.2026-10-17.zorblax-staging-reset.md"
	// sessionID is the ID of a session in the tests.
	sessionID        = "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11"
	zorblaxResetNote = "---\ntype: fact\ntier: L2\nsituation: testing against the Zorblax staging database\n" +
		"subject: the Zorblax staging database\npredicate: is reset every\nobject: Sunday at 03:40 UTC\n" +
		"luhmann: \"1\"\ncreated: \"2026-10-17\"\nsource: test\nuser: u\nvault: personal\n---\n\n" +
		"Information learned: when in testing against the Zorblax staging database, " +
		"the Zorblax staging database is reset every Sunday at 03:40 UTC.\n"
)

// unexported variables.
var (
	errNoHome = errors.New("no home directory")
)

// recordsReadFS reads through the file system it wraps and notes the reads of records of sessions.
type recordsReadFS struct {
	cli.EdgeFS

	reads *[]string
}

func (r recordsReadFS) ReadFile(path string) ([]byte, error) {
	if strings.HasSuffix(path, ".jsonl") {
		*r.reads = append(*r.reads, path)
	}

	data, err := r.EdgeFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return data, nil
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
		d.Stdin = deliveryRequest(sessionID, resetCandidate(), backupCandidate())
		d.Stdout = &stdout
	})

	if problem := notOnlyDelivered(stderr); problem != "" {
		t.Fatalf("the dispatch printed to standard error: %s", problem)
	}

	return stdout.String()
}

// notOnlyDelivered returns the first line of what a delivery printed to standard error that is not the line of a note
// delivered, or nothing when all are.
func notOnlyDelivered(stderr string) string {
	for line := range strings.SplitSeq(stderr, "\n") {
		if line != "" && !strings.HasPrefix(line, "delivered the note") {
			return line
		}
	}

	return ""
}

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
	backup := strings.NewReplacer("reset", "backed up", "1.", "2.").Replace(zorblaxResetNote)
	plantNoteOnDisk(t, vault, backupNoteName, backup)

	return vault
}

// queryOver runs the lookup of engram query over the given dependencies.
func queryOver(deps cli.QueryDeps) cli.QueryRun {
	return func(ctx context.Context, args cli.QueryArgs, stdout io.Writer) error {
		return cli.RunQuery(ctx, args, deps, stdout)
	}
}

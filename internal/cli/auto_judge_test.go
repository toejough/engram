package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/debuglog"
)

func Test793d4g1a_RunAutoJudge_PrintsTheRequestForTheTurnTheRecordEndsWith(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	project := filepath.Join(sessions, "-home-joe-zorblax")
	g.Expect(os.MkdirAll(project, 0o750)).To(Succeed())

	record := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":"How do I build the Zorblax project?"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Try make all."}]}}`,
		`{"type":"user","message":{"role":"user","content":"No - build with make zorblax-build."}}`,
	}, "\n") + "\n"
	g.Expect(os.WriteFile(filepath.Join(project, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
		cli.AutoJudgeDeps{
			Stdin:    turnInput(sessionID, "Understood."),
			Query:    queryOver(newQueryDeps(newInMemoryFS())),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	var step struct {
		Request struct {
			Model     string `json:"model"`
			Effort    string `json:"effort"`
			Prompt    string `json:"prompt"`
			TimeoutMs int    `json:"timeoutMs"`
		} `json:"request"`
		Carry map[string]any `json:"carry"`
	}

	g.Expect(json.Unmarshal(out.Bytes(), &step)).To(Succeed(), "the output is one JSON object: %q", out.String())
	g.Expect(step.Request.Model).To(Equal("sonnet"))
	g.Expect(step.Request.Effort).To(BeEmpty(), "the effort is the cheap tier's")
	g.Expect(step.Request.TimeoutMs).To(Equal(30000), "the row of the lessons judgement")
	g.Expect(step.Request.Prompt).To(ContainSubstring("User: How do I build the Zorblax project?\nAgent: Try make all.\n" +
		"--- the turn ---\nUser: No - build with make zorblax-build.\nAgent: Understood."))
	g.Expect(step.Carry).NotTo(BeNil(), "a carry")
	g.Expect(out.String()).To(HavePrefix(`{"request":`), "the mod knows a request by its first characters")
}

func Test793d4g1c_RunAutoJudge_ASessionWithNoPromptInItsRecordIsJudgedNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	record := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Understood."}]}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
		cli.AutoJudgeDeps{
			Stdin:    turnInput(sessionID, "Understood."),
			Query:    queryOver(newQueryDeps(newInMemoryFS())),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(BeEmpty())
}

func Test793d4g1d_RunAutoJudge_AnInputThatIsNotAJSONObjectFailsTheCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var out bytes.Buffer

	err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: t.TempDir()},
		cli.AutoJudgeDeps{
			Stdin: strings.NewReader("not a JSON object"), Query: queryOver(newQueryDeps(newInMemoryFS())),
			ReadDir: os.ReadDir, ReadFile: os.ReadFile,
		}, &out)

	g.Expect(err).To(MatchError(ContainSubstring("auto judge: reading the turn")))
	g.Expect(out.String()).To(BeEmpty())
}

func Test793d4g1h_RunAutoJudge_TheNearestNotesAreInTheRequest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.2026-10-17.zorblax-staging-reset.md", zorblaxResetNote)

	sessions := t.TempDir()
	record := `{"type":"user","message":{"role":"user","content":"When is the Zorblax staging database reset?"}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: vault, SessionsDir: sessions},
		cli.AutoJudgeDeps{
			Stdin:    turnInput(sessionID, "On Sundays."),
			Query:    queryOver(newQueryDeps(memFS)),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	var step struct {
		Request struct {
			Prompt string `json:"prompt"`
		} `json:"request"`
	}

	g.Expect(json.Unmarshal(out.Bytes(), &step)).To(Succeed())
	g.Expect(step.Request.Prompt).To(ContainSubstring("\n<notes>\n- 1.2026-10-17.zorblax-staging-reset.md " +
		"(testing against the Zorblax staging database): " +
		"Information learned: when in testing against the Zorblax staging database, " +
		"the Zorblax staging database is reset every Sunday at 03:40 UTC.\n</notes>"))
}

func Test855d4g1j_RunAutoJudge_TheNotesTheSessionWroteAreInTheRequestOnceAndThoseOfOthersAreNot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	ownNote := strings.Replace(zorblaxResetNote, "source: test", "source: written automatically (session "+sessionID+")", 1)
	otherNote := strings.NewReplacer("1.", "2.", "source: test", "source: written automatically (session another)").
		Replace(zorblaxResetNote)

	plantNoteOnDisk(t, vault, "1.2026-10-17.zorblax-staging-reset.md", ownNote)
	plantNoteOnDisk(t, vault, "2.2026-10-17.zorblax-other-session.md", otherNote)

	sessions := t.TempDir()
	record := `{"type":"user","message":{"role":"user","content":"Run the tests."}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	judged := func(query cli.QueryRun) []string {
		var out bytes.Buffer

		err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: vault, SessionsDir: sessions},
			cli.AutoJudgeDeps{Stdin: turnInput(sessionID, "Done."), Query: query, ReadDir: os.ReadDir, ReadFile: os.ReadFile},
			&out)
		g.Expect(err).NotTo(HaveOccurred())

		var step struct {
			Request struct {
				Prompt string `json:"prompt"`
			} `json:"request"`
		}

		g.Expect(json.Unmarshal(out.Bytes(), &step)).To(Succeed())

		_, rest, _ := strings.Cut(step.Request.Prompt, "<notes>\n")
		section, _, _ := strings.Cut(rest, "\n</notes>")

		return strings.Split(section, "\n")
	}

	nothingNear := queryOver(newQueryDeps(newInMemoryFS()))

	lines := judged(nothingNear)

	g.Expect(lines).To(HaveLen(1), "the note this session wrote, and not the note another session wrote")
	g.Expect(lines[0]).To(HavePrefix("- 1.2026-10-17.zorblax-staging-reset.md (testing against the Zorblax staging database): "))

	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.2026-10-17.zorblax-staging-reset.md", ownNote)

	g.Expect(judged(queryOver(newQueryDeps(memFS)))).To(HaveLen(1), "a note both near and written in the session is one line")
}

func Test793d4g1i_RunAutoJudge_AFailedLookupFailsTheCommandAndPrintsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	record := `{"type":"user","message":{"role":"user","content":` +
		`"When is the Zorblax staging database reset? the db pass is hunter2-abc"}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	query := newQueryDeps(newInMemoryFS())
	query.Embedder = errorEmbedder{}

	var out bytes.Buffer

	err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
		cli.AutoJudgeDeps{
			Stdin:    turnInput(sessionID, "On Sundays."),
			Query:    queryOver(query),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)

	g.Expect(err).To(MatchError(And(ContainSubstring("embedder down"), Not(ContainSubstring("hunter2-abc")),
		Not(ContainSubstring("Zorblax staging")))), "the cause, and none of the turn's prompts")
	g.Expect(out.String()).To(BeEmpty())
}

func Test793d4g1q_RunAutoJudge_ALongRecordIsJudgedInTime(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// 350 messages of about 3,000 characters: a record of 1 MB
	const (
		messages    = 350
		messageSize = 3000
		bound       = 2 * time.Second
	)

	const sentence = "the build of the zorblax project takes a while "

	filler := strings.Repeat(sentence, messageSize/len(sentence))
	rows := make([]string, 0, messages+1)

	for index := range messages {
		text := strconv.Itoa(index) + " " + filler

		if index%2 == 0 {
			rows = append(rows, `{"type":"user","message":{"role":"user","content":"prompt `+text+`"}}`)
		} else {
			rows = append(rows, `{"type":"assistant","message":{"role":"assistant",`+
				`"content":[{"type":"text","text":"answer `+text+`"}]}}`)
		}
	}

	rows = append(rows, `{"type":"user","message":{"role":"user","content":"No - build with make zorblax-build."}}`)

	sessions := t.TempDir()
	record := strings.Join(rows, "\n") + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())
	g.Expect(len(record)).To(BeNumerically(">", 1_000_000), "a record of 1 MB")

	var out bytes.Buffer

	started := time.Now()
	err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
		cli.AutoJudgeDeps{
			Stdin:    turnInput(sessionID, "Understood."),
			Query:    queryOver(newQueryDeps(newInMemoryFS())),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)
	elapsed := time.Since(started)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("User: No - build with make zorblax-build."))
	g.Expect(elapsed).To(BeNumerically("<", bound), "the request of a record of 1 MB")
}

func Test793d4g1r_RunAutoJudge_APromptTypedAheadDoesNotBeginTheTurnOfTheOneBeforeIt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// the second prompt was typed while the first turn ran: its row is in the record when the first turn is judged
	record := strings.Join([]string{
		`{"type":"user","message":{"role":"user",` +
			`"content":"Which port does zorblax listen on? Guess, then read zorblax.conf."}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I guessed 8080, but that was ` +
			`wrong: zorblax.conf says it listens on 7731."}]}}`,
		`{"type":"user","message":{"role":"user","content":"Thanks. Now list the files."}}`,
	}, "\n") + "\n"

	window := judged(t, record, `{"origin":"composer","session":"`+sessionID+`",`+
		`"text":"Which port does zorblax listen on? Guess, then read zorblax.conf.",`+
		learnTrue+`"answer":"I guessed 8080, but that was wrong: zorblax.conf says it listens on 7731."}`).window()

	g.Expect(window).To(Equal([]string{
		"--- the turn ---",
		"User: Which port does zorblax listen on? Guess, then read zorblax.conf.",
		"Agent: I guessed 8080, but that was wrong: zorblax.conf says it listens on 7731.",
	}))
}

func Test793d4g1s_RunAutoJudge_ATurnThatANotificationStartedAfterItIsNotPartOfIt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// the rows of a real record that Claude Code 2.1.302 wrote: the agent starts a command in the background and says so,
	// the command ends, and its notification starts a turn of its own
	fixture, readErr := os.ReadFile(filepath.Join("..", "auto", "testdata", "session_with_harness_rows.jsonl"))
	g.Expect(readErr).NotTo(HaveOccurred())

	rows := strings.Split(strings.TrimSpace(string(fixture)), "\n")
	record := strings.Join(rows[5:11], "\n") + "\n"
	asked := "Run this exact Bash command with run_in_background set to true: sleep 6; echo finished-zorblax. " +
		"Then reply with the single word started."

	window := judged(t, record,
		`{"origin":"composer","session":"`+sessionID+`","text":"`+asked+`",`+learnTrue+`"answer":"started"}`).window()

	g.Expect(window).To(Equal([]string{
		"--- the turn ---",
		"User: " + asked,
		"Agent used Bash",
		"Agent: started",
	}))
}

func Test793d4g1t_RunAutoJudge_ATextThatNoPromptHasBeginsTheTurnAtTheLastPrompt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":"How do I build the Zorblax project?"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Try make all."}]}}`,
		`{"type":"user","message":{"role":"user","content":"No - build with make zorblax-build."}}`,
	}, "\n") + "\n"

	window := judged(t, record, `{"origin":"composer","session":"`+sessionID+`",`+
		`"text":"/learn remember zorblax",`+learnTrue+`"answer":"Understood."}`).window()

	g.Expect(window).To(Equal([]string{
		"User: How do I build the Zorblax project?",
		"Agent: Try make all.",
		"--- the turn ---",
		"User: No - build with make zorblax-build.",
		"Agent: Understood.",
	}))
}

func Test793d4g1u_RunAutoJudge_TheInstructionsAreTheSystemOfTheRequest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := `{"type":"user","message":{"role":"user","content":"No - build with make zorblax-build."}}` + "\n"

	request := judged(t, record, turnInputText(sessionID, "Understood."))

	g.Expect(request.System).To(HavePrefix("You keep the long-term memory of an AI coding agent."))
	g.Expect(request.System).NotTo(ContainSubstring("make zorblax-build"), "nothing of the record")
	g.Expect(request.Prompt).To(HavePrefix("<window>\n"))
	g.Expect(request.Prompt).To(HaveSuffix("\n</notes>"))
	g.Expect(request.Prompt).NotTo(ContainSubstring("You keep the long-term memory"), "none of the instructions")
}

func Test793d4g1w_RunAutoJudge_ASessionWithNoRecordFailsTheCommand(t *testing.T) {
	t.Parallel()

	for name, session := range map[string]string{"a session no record holds": sessionID, "no session": ""} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			sessions := t.TempDir()

			var out bytes.Buffer

			err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
				cli.AutoJudgeDeps{
					Stdin:    turnInput(session, "Understood."),
					Query:    queryOver(newQueryDeps(newInMemoryFS())),
					ReadDir:  os.ReadDir,
					ReadFile: os.ReadFile,
				}, &out)

			g.Expect(err).To(MatchError(ContainSubstring("no record of session " + session + " under " + sessions)))
			g.Expect(out.String()).To(BeEmpty())
		})
	}
}

func Test793d4g1x_RunAutoJudge_AVaultThatDoesNotExistFailsTheCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	gone := filepath.Join(t.TempDir(), "no-such-vault")
	sessions := t.TempDir()
	record := `{"type":"user","message":{"role":"user","content":"No - build with make zorblax-build."}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	looked := false

	var out bytes.Buffer

	err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: gone, SessionsDir: sessions},
		cli.AutoJudgeDeps{
			Stdin: turnInput(sessionID, "Understood."),
			Query: func(context.Context, cli.QueryArgs, io.Writer) error {
				looked = true

				return nil
			},
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)

	g.Expect(err).To(MatchError(ContainSubstring("no vault at " + gone)))
	g.Expect(out.String()).To(BeEmpty())
	g.Expect(looked).To(BeFalse(), "the lookup was not run")
}

func Test855d4g1d_Judge_AJudgementThatPrintsNothingLogsIt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessions := t.TempDir()
	record := `{"type":"user","message":{"role":"user","content":"Thanks, that settles it."}}` + "\n"
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	var logged, out bytes.Buffer

	ctx := debuglog.WithLogger(context.Background(),
		debuglog.New(&logged, "engram", func() time.Time { return recordedAt }))

	err := cli.RunAutoJudge(ctx, cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
		cli.AutoJudgeDeps{
			Stdin: strings.NewReader(`{"origin":"composer","session":"` + sessionID +
				`","reply":"{\"learn\":false}","answer":"You are welcome."}`),
			Query:    queryOver(newQueryDeps(newInMemoryFS())),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(BeEmpty())

	lines := strings.Split(strings.TrimSpace(logged.String()), "\n")

	g.Expect(lines).To(HaveLen(1), "a line for the judgement")

	if len(lines) == 1 {
		g.Expect(lines[0]).To(HaveSuffix("[engram] auto.judge: session=" + sessionID + " nothing"))
	}
}

// unexported constants.
const (
	learnTrue = `"reply":"{\"learn\":true}",`
)

// judgedRequest is the request that the judgement prints: the fields of it that the tests read.
type judgedRequest struct {
	System string `json:"system"`
	Prompt string `json:"prompt"`
}

// window returns the lines of the window of the request.
func (r judgedRequest) window() []string {
	_, rest, _ := strings.Cut(r.Prompt, "<window>\n")
	window, _, _ := strings.Cut(rest, "\n</window>")

	return strings.Split(window, "\n")
}

// judged runs the judgement over the record of a session with the input the mod gives it, and returns the request.
func judged(t *testing.T, record, input string) judgedRequest {
	t.Helper()

	g := NewWithT(t)
	sessions := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(sessions, sessionID+".jsonl"), []byte(record), 0o600)).To(Succeed())

	var out bytes.Buffer

	err := cli.RunAutoJudge(context.Background(), cli.AutoJudgeArgs{VaultPath: t.TempDir(), SessionsDir: sessions},
		cli.AutoJudgeDeps{
			Stdin:    strings.NewReader(input),
			Query:    queryOver(newQueryDeps(newInMemoryFS())),
			ReadDir:  os.ReadDir,
			ReadFile: os.ReadFile,
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	var step struct {
		Request judgedRequest `json:"request"`
	}

	g.Expect(json.Unmarshal(out.Bytes(), &step)).To(Succeed(), "the output is one JSON object: %q", out.String())

	return step.Request
}

// turnInput is the JSON object the memory mod hands to the judgement for a turn of a prompt of the composer that the
// learn-moment judgement found to be a learn moment.
func turnInput(session, answer string) *strings.Reader {
	return strings.NewReader(turnInputText(session, answer))
}

// turnInputText is turnInput as text.
func turnInputText(session, answer string) string {
	return `{"origin":"composer","session":"` + session + `",` + learnTrue + `"answer":"` + answer + `"}`
}

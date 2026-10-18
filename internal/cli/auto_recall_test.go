package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

func Test855d4b1a_Recall_ThePhrasesAreLookedUpOnceAsPhrasesOfTheirOwn(t *testing.T) {
	t.Parallel()

	for name, want := range map[string]struct {
		depth  string
		count  int
		phrase int
	}{
		"four phrases at glance": {depth: "glance", count: 4, phrase: 3},
		"twelve phrases at deep": {depth: "deep", count: 12, phrase: 10},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			phrases := make([]string, 0, want.count)
			for index := range want.count {
				phrases = append(phrases, "phrase "+strconv.Itoa(index))
			}

			query, queryCalls := MockQueryRun(t)
			done := make(chan error, 1)

			go func() {
				var out bytes.Buffer

				done <- cli.RunAutoRecall(context.Background(),
					cli.AutoRecallArgs{VaultPath: "/a/vault", ChunksDir: "/a/chunks"},
					cli.AutoRecallDeps{Stdin: recallInput(want.depth, phrases...), Query: query}, &out)
			}()

			// the expectation runs apart, so that a call that does not match it, which this version of imptest leaves
			// waiting, cannot hold the test: the test waits for the outcome with a time limit
			asked := make(chan struct{})

			go func() {
				queryCalls.ArgsShould(match.BeAny, Equal(cli.QueryArgs{
					Phrases: phrases[:want.phrase], VaultPath: "/a/vault", ChunksDir: "/a/chunks",
					LazyChunks: true, RecentFill: -1, Limit: 30,
				}), match.BeAny).Return(nil)
				close(asked)
			}()

			select {
			case err := <-done:
				g.Expect(err).NotTo(HaveOccurred())
			case <-time.After(mockTimeout):
				t.Fatal("the recall did not finish: its lookup was not run once with the first phrases and no message")
			}

			select {
			case <-asked:
			case <-time.After(mockTimeout):
				t.Fatal("the recall did not run its lookup once with the first phrases and no message")
			}
		})
	}
}

func Test855d4b1c_Recall_AReplyThatIsNotAPhrasesReplyFailsTheCommand(t *testing.T) {
	t.Parallel()

	t.Run("a fenced reply of one phrase is looked up", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var asked []cli.QueryArgs

		err := cli.RunAutoRecall(context.Background(), cli.AutoRecallArgs{VaultPath: "/a/vault"},
			cli.AutoRecallDeps{
				Stdin: recallInputOfReply("```json\n{\"phrases\":[\"staging resets\"]}\n```"),
				Query: func(_ context.Context, args cli.QueryArgs, _ io.Writer) error {
					asked = append(asked, args)

					return nil
				},
			}, io.Discard)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(asked).To(HaveLen(1))

		if len(asked) == 1 {
			g.Expect(asked[0].Phrases).To(Equal([]string{"staging resets"}))
		}
	})

	for name, reply := range map[string]string{
		"text that is not JSON": "the phrases are staging and resets", "an empty reply": "",
		"an object with no phrases": "{}", "an empty list": `{"phrases":[]}`,
		"a list of blanks": `{"phrases":["", "  "]}`, "a list": `["staging"]`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			var out bytes.Buffer

			err := cli.RunAutoRecall(context.Background(), cli.AutoRecallArgs{VaultPath: "/a/vault"},
				cli.AutoRecallDeps{
					Stdin: recallInputOfReply(reply),
					Query: func(context.Context, cli.QueryArgs, io.Writer) error {
						t.Error("the lookup was run for a reply that is no phrases reply")

						return nil
					},
				}, &out)

			g.Expect(err).To(MatchError(ContainSubstring("phrases judgement")))
			g.Expect(err).NotTo(MatchError(ContainSubstring("Prepare a test run")), "the cause holds no text of the prompt")
			g.Expect(out.String()).To(BeEmpty())
		})
	}
}

func Test855d4b1d_Recall_TheNotesAndPassagesThatMatchedBecomeCandidatesTheExplorePicksAndTheNewestChunks(
	t *testing.T,
) {
	t.Parallel()
	g := NewWithT(t)

	payload := payloadOf(
		noteItem("1.2026-10-17.zorblax-staging-reset.md", "direct", "2026-10-17", resetSituation, resetBody),
		noteItem("2.2026-10-10.zorblax-explore-pick.md", "explore", "2026-10-10", "an explore pick", "Picked by explore."),
		chunkItem("/sessions/earlier.jsonl#turn-3", "direct"),
		chunkItem("/sessions/newest.jsonl#turn-9", "recent"),
	)

	var out bytes.Buffer

	err := cli.RunAutoRecall(context.Background(), cli.AutoRecallArgs{VaultPath: "/a/vault"},
		cli.AutoRecallDeps{
			Stdin: recallInput("glance", "staging resets"), Query: payloadQuery(payload),
			ReadChunks: chunksRead(
				chunk.Record{Source: "/sessions/earlier.jsonl", Anchor: "turn-3", Text: "USER: the staging resets on Sunday."},
				chunk.Record{Source: "/sessions/newest.jsonl", Anchor: "turn-9", Text: "USER: the newest chunk."}),
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	lines := candidateLines(t, out.Bytes())

	g.Expect(lines).To(HaveLen(2), "the matched note and the matched chunk")

	if len(lines) == 2 {
		g.Expect(lines[0]).To(HavePrefix("- 1.2026-10-17.zorblax-staging-reset.md (2026-10-17; " + resetSituation + "): " +
			resetBody))
		g.Expect(lines[1]).To(Equal(
			"- /sessions/earlier.jsonl#turn-3 (undated; passage): USER: the staging resets on Sunday."))
	}
}

func Test855d4b1e_Recall_AChunkOfTheSessionsOwnTranscriptIsNotACandidate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	own := "/home/joe/.claude/projects/-home-joe-zorblax/" + sessionID + ".jsonl"
	payload := payloadOf(chunkItem(own+"#turn-2", "direct"), chunkItem("/sessions/another.jsonl#turn-3", "direct"))

	var out bytes.Buffer

	err := cli.RunAutoRecall(context.Background(), cli.AutoRecallArgs{VaultPath: "/a/vault"},
		cli.AutoRecallDeps{
			Stdin: recallInput("glance", "staging resets"), Query: payloadQuery(payload),
			ReadChunks: chunksRead(
				chunk.Record{Source: own, Anchor: "turn-2", Text: "USER: this very conversation."},
				chunk.Record{Source: "/sessions/another.jsonl", Anchor: "turn-3", Text: "USER: another conversation."}),
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(candidateLines(t, out.Bytes())).To(Equal([]string{
		"- /sessions/another.jsonl#turn-3 (undated; passage): USER: another conversation.",
	}))
}

func Test855d4b1f_Recall_NothingMatchedPrintsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var out bytes.Buffer

	err := cli.RunAutoRecall(context.Background(), cli.AutoRecallArgs{VaultPath: "/a/vault"},
		cli.AutoRecallDeps{
			Stdin: recallInput("deep", "staging resets"), Query: payloadQuery(payloadOf()),
			ReadChunks: chunksRead(),
		}, &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(BeEmpty())
}

func Test855d4b1g_Recall_TheRequestIsForTheMidTierAndHoldsTheVerdictsTheWindowAndTheDated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	payload := payloadOf(
		noteItem("1.2026-10-17.zorblax-staging-reset.md", "direct", "2026-10-17", resetSituation, resetBody),
		fromParent(noteItem("2.2026-10-12.zorblax-backup-window.md", "direct", "2026-10-12", "backing up Zorblax",
			"Back up at 02:00.")),
		chunkItem("/sessions/earlier.jsonl#turn-3", "direct"),
	)

	var out bytes.Buffer

	err := cli.RunAutoRecall(context.Background(), cli.AutoRecallArgs{VaultPath: "/a/vault"},
		cli.AutoRecallDeps{
			Stdin: recallInput("glance", "staging resets"), Query: payloadQuery(payload),
			ReadChunks: chunksRead(chunk.Record{
				Source: "/sessions/earlier.jsonl", Anchor: "turn-3", Text: "USER: the staging resets on Sunday.",
				IngestedAt: time.Date(2026, 10, 14, 9, 30, 0, 0, time.UTC),
			}),
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(HavePrefix(`{"request":`), "the mod knows a request by its first characters")
	g.Expect(out.String()).NotTo(ContainSubstring(`"timeoutMs"`), "no time limit")
	g.Expect(out.String()).NotTo(ContainSubstring(`"write"`), "a glance writes nothing")

	var step struct {
		Request struct {
			Model     string `json:"model"`
			Effort    string `json:"effort"`
			System    string `json:"system"`
			Prompt    string `json:"prompt"`
			MaxTokens int    `json:"maxTokens"`
		} `json:"request"`
		Carry struct {
			Cue        string           `json:"cue"`
			Depth      string           `json:"depth"`
			Window     string           `json:"window"`
			Candidates []map[string]any `json:"candidates"`
		} `json:"carry"`
	}

	g.Expect(json.Unmarshal(out.Bytes(), &step)).To(Succeed())
	g.Expect(step.Request.Model).To(Equal("sonnet"))
	g.Expect(step.Request.Effort).To(BeEmpty())
	g.Expect(step.Request.MaxTokens).To(BeNumerically(">", 0))
	g.Expect(step.Request.System).To(And(ContainSubstring("covered"), ContainSubstring("near"),
		ContainSubstring("evidence"), ContainSubstring("unrelated")), "the four verdicts")
	g.Expect(step.Request.System).To(ContainSubstring("later date"), "the newer as the later date that reverses")
	g.Expect(step.Request.System).To(ContainSubstring(`{"candidates":[`), "the reply shape")
	g.Expect(step.Request.System).To(ContainSubstring("only matches the superseded content is near, not covered"),
		"the recall skill's rule of Step 2.5C")
	g.Expect(step.Request.System).NotTo(Or(ContainSubstring("zorblax"), ContainSubstring("Zorblax"),
		ContainSubstring("USER:")), "none of the candidates")
	g.Expect(step.Request.System).NotTo(Or(ContainSubstring(`"lessons"`), ContainSubstring("fields")),
		"a glance asks for no lesson")
	g.Expect(step.Request.Prompt).To(HavePrefix("<window>\nUser: Prepare a test run.\n</window>\n<candidates>\n"))
	g.Expect(candidateLines(t, out.Bytes())).To(Equal([]string{
		"- 1.2026-10-17.zorblax-staging-reset.md (2026-10-17; " + resetSituation + "): " + resetBody,
		"- 2.2026-10-12.zorblax-backup-window.md (2026-10-12; backing up Zorblax): Back up at 02:00.",
		"- /sessions/earlier.jsonl#turn-3 (2026-10-14; passage): USER: the staging resets on Sunday.",
	}))
	g.Expect(step.Carry.Cue).To(Equal("prompt"))
	g.Expect(step.Carry.Depth).To(Equal("glance"))
	g.Expect(step.Carry.Window).To(Equal("User: Prepare a test run."))
	g.Expect(step.Carry.Candidates).To(Equal([]map[string]any{
		{"id": "1.2026-10-17.zorblax-staging-reset.md", "kind": "note", "date": "2026-10-17", "situation": resetSituation,
			"text": resetBody, "parent": false, "pending": false},
		{"id": "2.2026-10-12.zorblax-backup-window.md", "kind": "note", "date": "2026-10-12",
			"situation": "backing up Zorblax", "text": "Back up at 02:00.", "parent": true, "pending": false},
		{"id": "/sessions/earlier.jsonl#turn-3", "kind": "passage", "date": "2026-10-14", "situation": "",
			"text": "USER: the staging resets on Sunday.", "parent": false, "pending": false},
	}))
}

func Test855d4b1h_Recall_ADeepRequestAlsoAsksForTheLessonsAndTheRevisions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var out bytes.Buffer

	err := cli.RunAutoRecall(context.Background(), cli.AutoRecallArgs{VaultPath: "/a/vault"},
		cli.AutoRecallDeps{
			Stdin: recallInput("deep", "staging resets"), ReadChunks: chunksRead(),
			Query: payloadQuery(payloadOf(
				noteItem("1.2026-10-17.zorblax-staging-reset.md", "direct", "2026-10-17", resetSituation, resetBody))),
		}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	var step struct {
		Request struct {
			System string `json:"system"`
		} `json:"request"`
		Write bool `json:"write"`
	}

	g.Expect(json.Unmarshal(out.Bytes(), &step)).To(Succeed())
	g.Expect(step.Request.System).To(And(ContainSubstring(`"lessons"`), ContainSubstring(`"fields"`),
		ContainSubstring(`"chunks"`)), "the lessons the evidence shows and the fields to revise")
	g.Expect(step.Request.System).To(ContainSubstring("that a newer candidate reverses is near, with fields"),
		"a note that a newer candidate reverses is revised")
	g.Expect(step.Write).To(BeTrue(), "the output holds write true at deep")
}

func Test855d4b1j_Recall_AFailedLookupFailsTheCommandAndPrintsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var out bytes.Buffer

	err := cli.RunAutoRecall(context.Background(), cli.AutoRecallArgs{VaultPath: "/a/vault"},
		cli.AutoRecallDeps{
			Stdin: recallInput("glance", "Prepare a test run against the Zorblax staging database tonight"),
			Query: func(context.Context, cli.QueryArgs, io.Writer) error {
				return errors.New("the vault cannot be scanned")
			},
			ReadChunks: chunksRead(),
		}, &out)

	g.Expect(err).To(MatchError(ContainSubstring("auto recall: lookup: the vault cannot be scanned")))
	g.Expect(err).NotTo(MatchError(ContainSubstring("Zorblax")), "the cause holds no text of the prompt")
	g.Expect(out.String()).To(BeEmpty())
}

func Test855d4b1k_Recall_AWarningOfTheLookupIsNoLineForTheUser(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := plantTwoNotes(t)

	// the embedding of the second note is of another model: the lookup drops it, and warns
	backupPath := filepath.Join(vault, backupNoteName)
	backup, readErr := os.ReadFile(backupPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	stale, buildErr := embed.BuildSidecar(t.Context(), stubEmbedder{modelID: "other@4", dims: 4}, backup)
	g.Expect(buildErr).NotTo(HaveOccurred())
	g.Expect(os.WriteFile(filepath.Join(vault, embed.SidecarPath(backupNoteName)), embed.MarshalSidecar(stale),
		0o600)).To(Succeed())

	var stdout, debug bytes.Buffer

	stderr := executeForTestWithDeps(t,
		[]string{"engram", "auto", "recall", "--vault", vault, "--chunks-dir", t.TempDir()},
		func(d *cli.Deps) {
			d.Embed = stubEmbedder{modelID: "m@4", dims: 4}
			d.Getenv = func(string) string { return "" }
			d.UserHomeDir = func() (string, error) { return t.TempDir(), nil }
			d.Stdin = recallInput("glance", "When is the Zorblax staging database reset?")
			d.Stdout = &stdout
			d.DebugLog = &debug
		})

	g.Expect(stdout.String()).To(HavePrefix(`{"request":`), "the request of the candidate judgement")
	g.Expect(stdout.String()).To(ContainSubstring(resetNoteName), "the note that is fit is a candidate")
	g.Expect(stderr).To(BeEmpty(), "what the user is shown")
	g.Expect(debug.String()).To(ContainSubstring("dropped 1 sidecar"), "the debug log holds the warning of the lookup")
}

// unexported constants.
const (
	resetBody      = "The Zorblax staging database is reset every Sunday at 03:40 UTC."
	resetSituation = "testing against the Zorblax staging database"
)

// candidateLines returns the lines of the section of candidates of the prompt of the request that a step printed.
func candidateLines(tb testing.TB, printed []byte) []string {
	tb.Helper()

	var step struct {
		Request struct {
			Prompt string `json:"prompt"`
		} `json:"request"`
	}

	err := json.Unmarshal(printed, &step)
	if err != nil {
		tb.Fatalf("the output is not one JSON object: %q", printed)
	}

	_, rest, _ := strings.Cut(step.Request.Prompt, "<candidates>\n")
	section, _, _ := strings.Cut(rest, "\n</candidates>")

	return strings.Split(section, "\n")
}

// chunkItem is an item of the payload of the lookup for a chunk: under the lazy lookup it holds no text.
func chunkItem(path, provenance string) string {
	return "  - path: " + path + "\n    kind: chunk\n    score: 0.5\n    provenances: [" + provenance + "]\n"
}

// chunksRead returns a reading of the chunk index that holds the records.
func chunksRead(records ...chunk.Record) func(string) ([]chunk.Record, error) {
	return func(string) ([]chunk.Record, error) { return records, nil }
}

// fromParent marks an item of the payload of the lookup as one from the parent vault.
func fromParent(item string) string {
	return item + "    from_parent: true\n"
}

// noteItem is an item of the payload of the lookup for a note.
func noteItem(path, provenance, created, situation, body string) string {
	return "  - path: " + path + "\n    kind: fact\n    score: 0.9\n    provenances: [" + provenance + "]\n" +
		"    content: |\n      ---\n      type: fact\n      situation: " + situation + "\n      created: \"" + created +
		"\"\n      source: a session\n      ---\n      " + body + "\n"
}

// payloadOf is the payload that the lookup prints for the items.
func payloadOf(items ...string) string {
	return "version: 1\nphrases: [a]\nitems:\n" + strings.Join(items, "") + "clusters: []\nmodel_id: m@4\n"
}

// payloadQuery returns a lookup that prints the payload.
func payloadQuery(payload string) cli.QueryRun {
	return func(_ context.Context, _ cli.QueryArgs, stdout io.Writer) error {
		_, err := io.WriteString(stdout, payload)

		return err
	}
}

// recallInput is the JSON object that the memory mod hands to the recall: the reply of the phrases judgement, a list of
// phrases, and the carry of the step before, of a depth.
func recallInput(depth string, phrases ...string) io.Reader {
	reply, err := json.Marshal(map[string][]string{"phrases": phrases})
	if err != nil {
		panic(err)
	}

	input, err := json.Marshal(map[string]any{
		"session": sessionID, "cue": "prompt", "reply": string(reply),
		"carry": map[string]string{"cue": "prompt", "depth": depth, "window": "User: Prepare a test run."},
	})
	if err != nil {
		panic(err)
	}

	return bytes.NewReader(input)
}

// recallInputOfReply is the JSON object that the memory mod hands to the recall, with the reply of the phrases
// judgement as the model gave it, at glance.
func recallInputOfReply(reply string) io.Reader {
	input, err := json.Marshal(map[string]any{
		"session": sessionID, "cue": "prompt", "reply": reply,
		"carry": map[string]string{"cue": "prompt", "depth": "glance", "window": "User: Prepare a test run."},
	})
	if err != nil {
		panic(err)
	}

	return bytes.NewReader(input)
}

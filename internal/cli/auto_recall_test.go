package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"

	"github.com/toejough/engram/internal/cli"
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
		"text that is not JSON": "the phrases are staging and resets", "an empty reply": "", "an object with no phrases": "{}",
		"an empty list": `{"phrases":[]}`, "a list of blanks": `{"phrases":["", "  "]}`, "a list": `["staging"]`,
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

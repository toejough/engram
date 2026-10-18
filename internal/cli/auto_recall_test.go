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

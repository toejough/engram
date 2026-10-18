package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func Test855d4v1c_Phrases_ADepthOfNoneLooksNothingUp(t *testing.T) {
	t.Parallel()

	for depth, want := range map[string]string{"none": "", "glance": `{"request":`} {
		t.Run(depth, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			var out bytes.Buffer

			err := cli.RunAutoPhrases(context.Background(), cli.AutoPhrasesDeps{
				Stdin: strings.NewReader(`{"session":"` + sessionID + `","cue":"prompt","reply":"{\"depth\":\"` + depth +
					`\"}","carry":{"cue":"prompt","window":"User: Prepare a test run."}}`),
			}, &out)

			g.Expect(err).NotTo(HaveOccurred())

			if want == "" {
				g.Expect(out.String()).To(BeEmpty(), "a depth of none prints nothing")
			} else {
				g.Expect(out.String()).To(HavePrefix(want), "the mod knows a request by its first characters")
			}
		})
	}
}

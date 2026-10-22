package chunk_test

import (
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
)

func Test792b16c1a3_SourceChunking_TextBeforeFirstHeadingIsPreamble(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	raw := strings.Join([]string{
		"This sentence comes before any heading and is long enough to clear the noise floor.",
		"## Setup",
		"This sentence follows the Setup heading and is also long enough to clear the noise floor.",
	}, "\n")

	chunks := chunk.Markdown(raw, sc1MaxLen)

	g.Expect(chunks).To(gomega.HaveLen(2))

	if len(chunks) != 2 {
		return
	}

	g.Expect(chunks[0].Anchor).To(gomega.Equal("preamble"))
	g.Expect(chunks[0].Text).To(gomega.ContainSubstring("before any heading"))
	g.Expect(chunks[1].Anchor).To(gomega.Equal("Setup"))
	g.Expect(chunks[1].Text).To(gomega.ContainSubstring("follows the Setup heading"))
}

func Test792b16c4a2_SourceChunking_TinySectionYieldsNothing(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	chunks := chunk.Markdown("## A\ntiny", sc1MaxLen)

	g.Expect(chunks).To(gomega.BeEmpty())
}

// unexported constants.
const (
	sc1MaxLen = 1500
)

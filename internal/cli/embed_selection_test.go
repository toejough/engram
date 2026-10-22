package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792b10c11a1_SelectStates covers the no-flag selection: missing notes
// are embedded, and broken ones with them. The flag combinations are covered
// by Test792b10c11a2 to Test792b10c11a6 in embedding_sidecar_scenarios_test.go.
func Test792b10c11a1_SelectStates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	args := cli.EmbedApplyArgs{}

	// ExportSelectStates is exposed for documentation; the
	// real assertions are on shouldEmbed below.
	_ = cli.ExportSelectStates(args)

	expectByState := map[embed.State]bool{
		embed.StateOK:           false,
		embed.StateMissing:      true,
		embed.StateStale:        false,
		embed.StateIncompatible: false,
		embed.StateBroken:       true, // broken inherits missing's intent
	}

	for state, want := range expectByState {
		got := cli.ExportShouldEmbed(args, state)
		g.Expect(got).To(Equal(want),
			"args=%+v state=%s want=%v got=%v", args, state, want, got)
	}
}

// Test792b23c2a1_KindFromContent covers the frontmatter type-extraction's three
// outcomes: too-short content, missing type line, and a parsed kind.
func Test792b23c2a1_KindFromContent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"empty content", "", "unknown"},
		{"shorter than minViableLen", "---", "unknown"},
		{"no type line", "---\nsituation: x\n---\nbody", "unknown"},
		{"valid fact frontmatter", "---\ntype: fact\nluhmann: \"1\"\n---\nbody", "fact"},
		{"valid feedback frontmatter", "---\ntype: feedback\n---\nbody", "feedback"},
		{
			"type line beyond max scan returns unknown",
			"---\n" + repeat("a: b\n", 80) + "type: fact\n---\nbody",
			"unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			g.Expect(cli.ExportKindFromContent(tc.content)).To(Equal(tc.want))
		})
	}
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for range n {
		out = append(out, s...)
	}

	return string(out)
}

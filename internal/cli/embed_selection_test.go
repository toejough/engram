package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestKindFromContent covers the frontmatter type-extraction's three
// outcomes: too-short content, missing type line, and a parsed kind.
func TestKindFromContent(t *testing.T) {
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

// TestSelectStates exercises every flag combination so the
// shouldEmbed dispatcher covers every State branch.
func TestSelectStates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		args          cli.EmbedApplyArgs
		expectByState map[embed.State]bool
	}{
		{
			name: "empty selection defaults to missing only",
			args: cli.EmbedApplyArgs{},
			expectByState: map[embed.State]bool{
				embed.StateOK:           false,
				embed.StateMissing:      true,
				embed.StateStale:        false,
				embed.StateIncompatible: false,
				embed.StateBroken:       true, // broken inherits missing's intent
			},
		},
		{
			name: "--all enables every state",
			args: cli.EmbedApplyArgs{All: true},
			expectByState: map[embed.State]bool{
				embed.StateOK:           true,
				embed.StateMissing:      true,
				embed.StateStale:        true,
				embed.StateIncompatible: true,
				embed.StateBroken:       true,
			},
		},
		{
			name: "--stale targets stale and broken only",
			args: cli.EmbedApplyArgs{Stale: true},
			expectByState: map[embed.State]bool{
				embed.StateOK:           false,
				embed.StateMissing:      false,
				embed.StateStale:        true,
				embed.StateIncompatible: false,
				embed.StateBroken:       true,
			},
		},
		{
			name: "--force lifts incompatible (and broken)",
			args: cli.EmbedApplyArgs{Force: true},
			expectByState: map[embed.State]bool{
				embed.StateOK:           false,
				embed.StateMissing:      false,
				embed.StateStale:        false,
				embed.StateIncompatible: true,
				embed.StateBroken:       true,
			},
		},
		{
			name: "--missing alone is explicit-default",
			args: cli.EmbedApplyArgs{Missing: true},
			expectByState: map[embed.State]bool{
				embed.StateOK:           false,
				embed.StateMissing:      true,
				embed.StateStale:        false,
				embed.StateIncompatible: false,
				embed.StateBroken:       true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// ExportSelectStates is exposed for documentation; the
			// real assertions are on shouldEmbed below.
			_ = cli.ExportSelectStates(tc.args)

			for state, want := range tc.expectByState {
				got := cli.ExportShouldEmbed(tc.args, state)
				g.Expect(got).To(Equal(want),
					"args=%+v state=%s want=%v got=%v", tc.args, state, want, got)
			}
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

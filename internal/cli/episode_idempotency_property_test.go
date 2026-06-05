package cli_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestEpisodeBody_RenderParseIdempotencyProperty mechanizes the migration
// fixed-point over adversarial transcripts. It renders an episode body from a
// generated transcript (seeded with "## " headings, ``` runs, "Related to:"
// lines, and [[wikilinks]]) plus fixed summary / authored-relation / preceding
// inputs, then re-derives the body exactly as a second migrate pass would —
// parse the prior body, fold its ## Related back into the relation set, and
// re-render with the same preceding links. The two bodies must be byte-
// identical (migrate ∘ migrate == migrate), the transcript must round-trip
// verbatim, and no wikilink buried in the transcript may become a graph edge.
func TestEpisodeBody_RenderParseIdempotencyProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		transcript := genAdversarialTranscript(rt)

		preceding := []cli.ExportEpisodeLink{
			{Basename: "9p.2026-05-01.prior", Rationale: "immediately preceding episode"},
		}
		authored := []string{"9r.2026-05-02.rel|an authored relation"}

		first := cli.ExportRenderEpisodeBody(cli.ExportEpisodeFields{
			Summary:        "a fixed summary line",
			TranscriptText: transcript,
			Relations:      authored,
			Preceding:      preceding,
		})

		// The transcript round-trips verbatim (modulo the trailing newline the
		// renderer guarantees before the closing fence).
		_, gotTranscript, gotRelations := cli.ExportParseEpisodeBody(first)
		g.Expect(gotTranscript).To(Equal(ensureTrailingNewline(transcript)),
			"transcript not recovered verbatim; body:\n%s", first)

		// Second migrate pass: the prior ## Related (preceding + authored) is
		// re-parsed as the relation set, then re-rendered with the same
		// preceding links — dedup must collapse it back to identical output.
		second := cli.ExportRenderEpisodeBody(cli.ExportEpisodeFields{
			Summary:        "a fixed summary line",
			TranscriptText: gotTranscript,
			Relations:      gotRelations,
			Preceding:      preceding,
		})
		g.Expect(second).To(Equal(first), "render∘parse is not a fixed point; body:\n%s", first)

		// No wikilink inside the fenced transcript becomes a graph edge; the
		// authored + preceding links in ## Related still resolve. The buried
		// targets are exactly the links ParseWikilinks finds in the raw
		// (unfenced) transcript text.
		links := vaultgraph.ParseWikilinks([]byte(first))
		for _, buried := range vaultgraph.ParseWikilinks([]byte(transcript)) {
			g.Expect(links).NotTo(ContainElement(buried),
				"in-transcript link %q leaked into the graph; body:\n%s", buried, first)
		}

		g.Expect(links).To(ContainElement("9p.2026-05-01.prior"))
		g.Expect(links).To(ContainElement("9r.2026-05-02.rel"))
	})
}

// unexported constants.
const (
	buriedLinkTarget = "buried-in-transcript"
)

// ensureTrailingNewline appends a newline when s lacks one, matching the
// renderer's guarantee that the fenced transcript ends with a newline before
// its closing fence.
func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}

	return s + "\n"
}

// genAdversarialTranscript builds a verbatim transcript seeded with the exact
// structural tokens that have historically corrupted the migration parser:
// "## " headings (including literal ## Summary / ## Transcript / ## Related),
// ``` fenced runs, "Related to:" lines, and [[wikilinks]]. The returned text is
// what an episode "captured an agent editing episode notes" would contain.
func genAdversarialTranscript(rt *rapid.T) string {
	const maxLines = 12

	choices := []string{
		"USER: a plain line",
		"ASSISTANT: another plain line",
		"## Summary",
		"## Transcript",
		"## Related",
		"## Some Other Heading",
		"```",
		"````",
		"Related to:",
		"- a bullet that is not a wikilink",
		"- [[" + buriedLinkTarget + "]] — buried, must not be an edge",
		"some text mentioning Related to: inline",
	}

	lineCount := rapid.IntRange(1, maxLines).Draw(rt, "lineCount")

	lines := make([]string, 0, lineCount+1)
	for range lineCount {
		lines = append(lines, rapid.SampledFrom(choices).Draw(rt, "line"))
	}

	// Guarantee at least one buried wikilink so the no-edge assertion has teeth.
	lines = append(lines, "- [["+buriedLinkTarget+"]] — always buried")

	return strings.Join(lines, "\n") + "\n"
}

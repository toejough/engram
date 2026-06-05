package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestParseEpisodeBody_LegacyRelatedToFollowedByProse guards the legacy
// relation split: a "Related to:" line followed by non-bullet prose is NOT an
// authored relation block, so the whole body stays the transcript and no
// spurious relations are extracted.
func TestParseEpisodeBody_LegacyRelatedToFollowedByProse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := "USER: how is this Related to: the prior work\nASSISTANT: it builds on it\n"

	summary, transcript, relations := cli.ExportParseEpisodeBody(body)

	g.Expect(summary).To(BeEmpty())
	g.Expect(transcript).To(Equal("USER: how is this Related to: the prior work\nASSISTANT: it builds on it"))
	g.Expect(relations).To(BeNil())
}

// TestParseEpisodeBody_LegacyRoundTrip verifies the parser splits a legacy
// verbatim body (transcript + "Related to:" block) into transcript and
// relations.
func TestParseEpisodeBody_LegacyRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := "USER: do the thing\nASSISTANT: done\n\nRelated to:\n- [[3]] — earlier note.\n"

	summary, transcript, relations := cli.ExportParseEpisodeBody(body)

	g.Expect(summary).To(BeEmpty())
	g.Expect(transcript).To(Equal("USER: do the thing\nASSISTANT: done"))
	g.Expect(relations).To(Equal([]string{"3|earlier note"}))
}

// TestParseEpisodeBody_LegacyTranscriptContainingTranscriptHeading guards the
// legacy-vs-migrated detection: a legacy body whose verbatim transcript
// contains the literal line "## Transcript" must still be treated as legacy
// (no fenced structure), so the whole transcript is preserved.
func TestParseEpisodeBody_LegacyTranscriptContainingTranscriptHeading(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Legacy = no fenced ## Transcript SECTION, even though the text mentions it.
	body := "USER: I edited the ## Transcript section\nASSISTANT: done\n"

	summary, transcript, relations := cli.ExportParseEpisodeBody(body)

	g.Expect(summary).To(BeEmpty())
	g.Expect(transcript).To(Equal("USER: I edited the ## Transcript section\nASSISTANT: done"))
	g.Expect(relations).To(BeNil())
}

// TestParseEpisodeBody_LegacyTranscriptWithExactTranscriptHeadingAndFence is
// the detection-bug guard: a LEGACY transcript that contains the EXACT line
// "## Transcript" followed by a ``` fence must still parse as legacy (it does
// not begin with "## Summary"), so the whole verbatim transcript is preserved
// — never truncated by a fence found inside the transcript.
func TestParseEpisodeBody_LegacyTranscriptWithExactTranscriptHeadingAndFence(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := "USER: pasted a note\n## Transcript\n```\ninner\n```\nASSISTANT: ok\n"

	summary, transcript, relations := cli.ExportParseEpisodeBody(body)

	g.Expect(summary).To(BeEmpty())
	g.Expect(transcript).To(Equal("USER: pasted a note\n## Transcript\n```\ninner\n```\nASSISTANT: ok"))
	g.Expect(relations).To(BeNil())
}

// TestParseEpisodeBody_MigratedFencedTranscriptRoundTrip verifies a migrated
// body whose transcript itself contains a ``` block (wrapped in a longer
// fence) recovers the inner ``` verbatim.
func TestParseEpisodeBody_MigratedFencedTranscriptRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := "## Summary\ns\n\n## Transcript\n````\nUSER: paste\n```\ncode\n```\nASSISTANT: ok\n````\n"

	_, transcript, _ := cli.ExportParseEpisodeBody(body)

	g.Expect(transcript).To(Equal("USER: paste\n```\ncode\n```\nASSISTANT: ok\n"))
}

// TestParseEpisodeBody_MigratedRoundTrip verifies the parser pulls the three
// sections back out of an already-migrated D6 body, recovering the transcript
// verbatim from inside the fence.
func TestParseEpisodeBody_MigratedRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := "## Summary\nthe summary\n\n## Transcript\n```\nUSER: hi\nASSISTANT: bye\n```\n\n" +
		"## Related\n- [[3.2026-05-20.earlier]] — earlier note\n"

	summary, transcript, relations := cli.ExportParseEpisodeBody(body)

	g.Expect(summary).To(Equal("the summary"))
	g.Expect(transcript).To(Equal("USER: hi\nASSISTANT: bye\n"))
	g.Expect(relations).To(Equal([]string{"3.2026-05-20.earlier|earlier note"}))
}

// TestParseEpisodeBody_MigratedTranscriptWithHeadingsAndFences is the
// discriminating case: a migrated transcript that itself contains "## Related",
// a ``` block, and a [[wikilink]]. The fence (4 backticks, longer than the
// inner 3-run) must be honored — the parser may NOT break the ## Transcript
// section at the in-transcript "## Related" line, and must recover the whole
// transcript verbatim. The real ## Related (after the closing fence) is parsed
// separately.
func TestParseEpisodeBody_MigratedTranscriptWithHeadingsAndFences(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	transcript := "USER: look at this note\n## Related\n```\n- [[should-not-link]] — x\n```\nASSISTANT: ok\n"
	body := "## Summary\ns\n\n## Transcript\n````\n" + transcript + "````\n\n" +
		"## Related\n- [[3.2026-05-20.real]] — the real relation\n"

	summary, gotTranscript, relations := cli.ExportParseEpisodeBody(body)

	g.Expect(summary).To(Equal("s"))
	g.Expect(gotTranscript).To(Equal(transcript))
	g.Expect(relations).To(Equal([]string{"3.2026-05-20.real|the real relation"}))
}

// TestParseEpisodeBody_NonBacktickFenceFallsBackToLegacy verifies that a
// "## Transcript" heading NOT immediately followed by a backtick fence (e.g. a
// tilde "fence") is not recognized as a migrated transcript — backtickFenceRun
// rejects the non-backtick run — so the body is parsed as legacy (the whole
// verbatim text, trimmed, is the transcript and there is no summary).
func TestParseEpisodeBody_NonBacktickFenceFallsBackToLegacy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := "## Summary\ns\n\n## Transcript\n~~~\nUSER: hi\n~~~\n"

	summary, transcript, _ := cli.ExportParseEpisodeBody(body)

	g.Expect(summary).To(BeEmpty())
	g.Expect(transcript).To(Equal("## Summary\ns\n\n## Transcript\n~~~\nUSER: hi\n~~~"))
}

// TestRunMigrateEpisodes_ApplyRewritesToThreeSections verifies --apply
// rewrites a legacy episode into the D6 3-section format, seeding ## Summary
// from boundary_rationale, fencing the old body as ## Transcript, and
// migrating the authored relation to full-basename form under ## Related.
func TestRunMigrateEpisodes_ApplyRewritesToThreeSections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	wrote := map[string][]byte{}

	var out bytes.Buffer

	err := cli.RunMigrateEpisodes(
		context.Background(),
		cli.MigrateEpisodesArgs{VaultPath: "v", Apply: true},
		oneEpisodeMigrateDeps(wrote),
		&out,
	)

	g.Expect(err).NotTo(HaveOccurred())

	got := string(wrote["v/Permanent/5.2026-05-25.work.md"])
	g.Expect(got).NotTo(BeEmpty())

	// Frontmatter (incl. boundary_rationale) preserved.
	g.Expect(got).To(ContainSubstring("type: episode"))
	g.Expect(got).To(ContainSubstring("boundary_rationale: a discrete arc of work"))
	// Summary seeded from boundary_rationale.
	g.Expect(got).To(ContainSubstring("## Summary\na discrete arc of work"))
	// Transcript fenced and verbatim.
	g.Expect(got).To(ContainSubstring("## Transcript"))
	g.Expect(got).To(ContainSubstring("USER: do the thing"))
	// Related migrated to full basename, no "Related to:" marker.
	g.Expect(got).To(ContainSubstring("## Related"))
	g.Expect(got).To(ContainSubstring("[[3.2026-05-20.earlier]]"))
	g.Expect(got).NotTo(ContainSubstring("Related to:"))
	g.Expect(got).NotTo(ContainSubstring("[[3]]"))

	g.Expect(out.String()).To(ContainSubstring("applied"))
}

// TestRunMigrateEpisodes_DryRunDoesNotWrite verifies the default (no --apply)
// reports what would change without writing.
func TestRunMigrateEpisodes_DryRunDoesNotWrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	wrote := map[string][]byte{}

	var out bytes.Buffer

	err := cli.RunMigrateEpisodes(
		context.Background(),
		cli.MigrateEpisodesArgs{VaultPath: "v"},
		oneEpisodeMigrateDeps(wrote),
		&out,
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(wrote).To(BeEmpty())
	g.Expect(out.String()).To(ContainSubstring("would-rewrite"))
	g.Expect(out.String()).To(ContainSubstring("dry-run"))
}

// TestRunMigrateEpisodes_EmbedFailureSurfaces verifies a re-embed failure
// during --apply aborts the migration with a wrapped error.
func TestRunMigrateEpisodes_EmbedFailureSurfaces(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	wrote := map[string][]byte{}

	deps := oneEpisodeMigrateDeps(wrote)
	deps.Embedder = failingEmbedder{}

	var out bytes.Buffer

	err := cli.RunMigrateEpisodes(
		context.Background(),
		cli.MigrateEpisodesArgs{VaultPath: "v", Apply: true},
		deps,
		&out,
	)
	g.Expect(err).To(MatchError(ContainSubstring("embedding")))
}

// TestRunMigrateEpisodes_Idempotent verifies running the migration twice over
// a real temp vault changes nothing the second time.
func TestRunMigrateEpisodes_Idempotent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o700)).To(Succeed())

	episodePath := filepath.Join(vault, "Permanent", "5.2026-05-25.work.md")
	g.Expect(os.WriteFile(episodePath, []byte(legacyEpisodeNote), 0o600)).To(Succeed())

	// A second episode so the relation target ("3") resolves to a real basename.
	earlierPath := filepath.Join(vault, "Permanent", "3.2026-05-20.earlier.md")
	g.Expect(os.WriteFile(earlierPath, []byte(legacyEpisodeNote), 0o600)).To(Succeed())

	deps := cli.ExportNewOsMigrateEpisodesDeps(successEmbedder{})

	var out1 bytes.Buffer

	err := cli.RunMigrateEpisodes(
		context.Background(),
		cli.MigrateEpisodesArgs{VaultPath: vault, Apply: true},
		deps,
		&out1,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	afterFirst, rErr := os.ReadFile(episodePath)
	g.Expect(rErr).NotTo(HaveOccurred())
	g.Expect(string(afterFirst)).To(ContainSubstring("## Summary"))
	g.Expect(string(afterFirst)).To(ContainSubstring("## Transcript"))

	var out2 bytes.Buffer

	err = cli.RunMigrateEpisodes(
		context.Background(),
		cli.MigrateEpisodesArgs{VaultPath: vault, Apply: true},
		deps,
		&out2,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	afterSecond, r2Err := os.ReadFile(episodePath)
	g.Expect(r2Err).NotTo(HaveOccurred())

	// Idempotent: second run is byte-identical and reports zero changes.
	g.Expect(string(afterSecond)).To(Equal(string(afterFirst)))
	g.Expect(out2.String()).To(ContainSubstring("0 notes"))
}

// TestRunMigrateEpisodes_IdempotentWithHeadingAndRelatedProse is the
// detection-corruption guard: a legacy transcript containing an exact
// "## Transcript" line + fence AND a "Related to:" line followed by prose must
// migrate without losing transcript content and re-migrate byte-identically.
func TestRunMigrateEpisodes_IdempotentWithHeadingAndRelatedProse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o700)).To(Succeed())

	notePath := filepath.Join(vault, "Permanent", "8.2026-05-25.meta.md")
	g.Expect(os.WriteFile(notePath, []byte(legacyEpisodeWithHeadingAndRelatedProse), 0o600)).To(Succeed())

	deps := cli.ExportNewOsMigrateEpisodesDeps(successEmbedder{})

	var out1 bytes.Buffer

	err := cli.RunMigrateEpisodes(
		context.Background(),
		cli.MigrateEpisodesArgs{VaultPath: vault, Apply: true},
		deps,
		&out1,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	afterFirst, rErr := os.ReadFile(notePath)
	g.Expect(rErr).NotTo(HaveOccurred())

	if rErr != nil {
		return
	}

	first := string(afterFirst)
	// No transcript content lost: the in-transcript heading, fence, and the
	// trailing "Related to:" prose line all survive verbatim.
	g.Expect(first).To(ContainSubstring("USER: the body becomes\n## Transcript"))
	g.Expect(first).To(ContainSubstring("the fenced chunk"))
	g.Expect(first).To(ContainSubstring("ASSISTANT: and how is this Related to: the earlier episode? it builds on it."))
	// The transcript prose is not mistaken for an authored relation block: the
	// only ## Related section, if any, comes from preceding links (none here).
	g.Expect(first).NotTo(ContainSubstring("## Related"))

	var out2 bytes.Buffer

	err = cli.RunMigrateEpisodes(
		context.Background(),
		cli.MigrateEpisodesArgs{VaultPath: vault, Apply: true},
		deps,
		&out2,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	afterSecond, r2Err := os.ReadFile(notePath)
	g.Expect(r2Err).NotTo(HaveOccurred())

	g.Expect(string(afterSecond)).To(Equal(first))
	g.Expect(out2.String()).To(ContainSubstring("0 notes"))
}

// TestRunMigrateEpisodes_IdempotentWithTrickyTranscript is the blocking-bug
// guard: a legacy transcript containing "## Related", a ``` block, and a
// [[wikilink]] must migrate, then re-migrate to byte-identical output (no data
// loss, no second rewrite), and the in-transcript wikilink must NOT become a
// graph edge while the authored ## Related link does.
func TestRunMigrateEpisodes_IdempotentWithTrickyTranscript(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o700)).To(Succeed())

	notePath := filepath.Join(vault, "Permanent", "7.2026-05-25.tricky.md")
	g.Expect(os.WriteFile(notePath, []byte(legacyEpisodeWithTrickyTranscript), 0o600)).To(Succeed())

	deps := cli.ExportNewOsMigrateEpisodesDeps(successEmbedder{})

	var out1 bytes.Buffer

	err := cli.RunMigrateEpisodes(
		context.Background(),
		cli.MigrateEpisodesArgs{VaultPath: vault, Apply: true},
		deps,
		&out1,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	afterFirst, rErr := os.ReadFile(notePath)
	g.Expect(rErr).NotTo(HaveOccurred())

	if rErr != nil {
		return
	}

	first := string(afterFirst)
	// The in-transcript "## Related" line and everything after it survives.
	g.Expect(first).To(ContainSubstring("USER: here is a note body I pasted\n## Related"))
	g.Expect(first).To(ContainSubstring("- [[in-transcript-link]] — should not become an edge"))
	g.Expect(first).To(ContainSubstring("ASSISTANT: noted"))

	// The in-transcript wikilink is fenced → not a graph edge.
	links := vaultgraph.ParseWikilinks(afterFirst)
	g.Expect(links).NotTo(ContainElement("in-transcript-link"))

	var out2 bytes.Buffer

	err = cli.RunMigrateEpisodes(
		context.Background(),
		cli.MigrateEpisodesArgs{VaultPath: vault, Apply: true},
		deps,
		&out2,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	afterSecond, r2Err := os.ReadFile(notePath)
	g.Expect(r2Err).NotTo(HaveOccurred())

	g.Expect(string(afterSecond)).To(Equal(first))
	g.Expect(out2.String()).To(ContainSubstring("0 notes"))
}

// TestRunMigrateEpisodes_NilEmbedderSkipsSidecar verifies that with a nil
// Embedder, --apply still rewrites the note body but writes no sidecar.
func TestRunMigrateEpisodes_NilEmbedderSkipsSidecar(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	wrote := map[string][]byte{}

	deps := oneEpisodeMigrateDeps(wrote)
	deps.Embedder = nil

	var out bytes.Buffer

	err := cli.RunMigrateEpisodes(
		context.Background(),
		cli.MigrateEpisodesArgs{VaultPath: "v", Apply: true},
		deps,
		&out,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(wrote).To(HaveKey("v/Permanent/5.2026-05-25.work.md"))

	for path := range wrote {
		g.Expect(path).NotTo(HaveSuffix(".vec.json"))
	}
}

// unexported constants.
const (
	laterEpisodeNote = `---
type: episode
tier: L1
situation: the later work
boundary_rationale: a later arc of work
provenance:
  sessions:
    - sess-2
  transcript_range:
    start: "2026-05-25T23:30:00Z"
    end: "2026-05-25T23:45:00Z"
luhmann: "3"
created: "2026-05-20"
source: agent
---

USER: later thing
ASSISTANT: later done
`
	legacyEpisodeNote = `---
type: episode
tier: L1
situation: doing the work
boundary_rationale: a discrete arc of work
provenance:
  sessions:
    - sess-1
  transcript_range:
    start: "2026-05-25T22:00:00Z"
    end: "2026-05-25T23:00:00Z"
luhmann: "5"
created: "2026-05-25"
source: agent
---

USER: do the thing
ASSISTANT: done

Related to:
- [[3]] — earlier note.
`
	// legacyEpisodeWithHeadingAndRelatedProse is a legacy episode (it captured
	// this very D6 work) whose verbatim transcript contains the EXACT line
	// "## Transcript" + a ``` fence AND a "Related to:" line followed by prose
	// — the two detection-corruption cases combined.
	legacyEpisodeWithHeadingAndRelatedProse = `---
type: episode
tier: L1
situation: building the D6 format
boundary_rationale: a meta arc
provenance:
  sessions:
    - sess-meta
  transcript_range:
    start: "2026-05-25T22:00:00Z"
    end: "2026-05-25T23:00:00Z"
luhmann: "8"
created: "2026-05-25"
source: agent
---

USER: the body becomes
## Transcript
` + "```" + `
the fenced chunk
` + "```" + `
ASSISTANT: and how is this Related to: the earlier episode? it builds on it.
`
	legacyEpisodeWithTrickyTranscript = `---
type: episode
tier: L1
situation: editing an episode note
boundary_rationale: a tricky arc
provenance:
  sessions:
    - sess-9
  transcript_range:
    start: "2026-05-25T22:00:00Z"
    end: "2026-05-25T23:00:00Z"
luhmann: "7"
created: "2026-05-25"
source: agent
---

USER: here is a note body I pasted
## Related
` + "```" + `
- [[in-transcript-link]] — should not become an edge
` + "```" + `
ASSISTANT: noted
`
)

// oneEpisodeMigrateDeps returns MigrateEpisodesDeps with a single legacy
// episode plus a sibling note so the bare-id relation resolves.
func oneEpisodeMigrateDeps(wrote map[string][]byte) cli.MigrateEpisodesDeps {
	return cli.MigrateEpisodesDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{
				{Basename: "5.2026-05-25.work"},
				{Basename: "3.2026-05-20.earlier"},
			}, nil
		},
		Read: func(path string) ([]byte, error) {
			if filepath.Base(path) == "5.2026-05-25.work.md" {
				return []byte(legacyEpisodeNote), nil
			}

			return []byte(laterEpisodeNote), nil
		},
		Write:    func(path string, data []byte) error { wrote[path] = data; return nil },
		Embedder: successEmbedder{},
	}
}

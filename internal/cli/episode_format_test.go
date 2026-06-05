package cli_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestComputePrecedingLinks covers the four shapes of the preceding-link rule
// plus the boundary collapse where an immediate-prior episode is also
// active-at-start.
func TestComputePrecedingLinks(t *testing.T) {
	t.Parallel()

	const (
		activeRationale = "preceding episode (active at this episode's start)"
		priorRationale  = "immediately preceding episode"
	)

	newStart := "2026-05-25T12:00:00Z"

	cases := []struct {
		name     string
		existing []cli.ExportEpisodeRange
		want     []cli.ExportEpisodeLink
	}{
		{
			name: "active-at-start: F spans E.start",
			existing: []cli.ExportEpisodeRange{
				{Basename: "a.f.active", Start: "2026-05-25T11:00:00Z", End: "2026-05-25T13:00:00Z"},
			},
			want: []cli.ExportEpisodeLink{{Basename: "a.f.active", Rationale: activeRationale}},
		},
		{
			name: "immediate-prior: F ends strictly before E.start",
			existing: []cli.ExportEpisodeRange{
				{Basename: "a.f.prior", Start: "2026-05-25T09:00:00Z", End: "2026-05-25T10:00:00Z"},
			},
			want: []cli.ExportEpisodeLink{{Basename: "a.f.prior", Rationale: priorRationale}},
		},
		{
			name: "both distinct: one active, one strictly-prior",
			existing: []cli.ExportEpisodeRange{
				{Basename: "a.f.active", Start: "2026-05-25T11:30:00Z", End: "2026-05-25T12:30:00Z"},
				{Basename: "a.f.prior", Start: "2026-05-25T08:00:00Z", End: "2026-05-25T09:00:00Z"},
				{Basename: "a.f.older", Start: "2026-05-25T06:00:00Z", End: "2026-05-25T07:00:00Z"},
			},
			want: []cli.ExportEpisodeLink{
				{Basename: "a.f.active", Rationale: activeRationale},
				{Basename: "a.f.prior", Rationale: priorRationale},
			},
		},
		{
			name: "neither: only later episodes exist",
			existing: []cli.ExportEpisodeRange{
				{Basename: "a.f.later", Start: "2026-05-25T13:00:00Z", End: "2026-05-25T14:00:00Z"},
			},
			want: nil,
		},
		{
			name: "boundary collapse: F.end == E.start is active AND prior, one bullet preferring active",
			existing: []cli.ExportEpisodeRange{
				{Basename: "a.f.boundary", Start: "2026-05-25T11:00:00Z", End: "2026-05-25T12:00:00Z"},
			},
			want: []cli.ExportEpisodeLink{{Basename: "a.f.boundary", Rationale: activeRationale}},
		},
		{
			name: "prior tie on equal end: smaller basename wins the single immediate-prior bullet",
			existing: []cli.ExportEpisodeRange{
				{Basename: "a.f.zeta", Start: "2026-05-25T08:00:00Z", End: "2026-05-25T10:00:00Z"},
				{Basename: "a.f.alpha", Start: "2026-05-25T09:00:00Z", End: "2026-05-25T10:00:00Z"},
			},
			want: []cli.ExportEpisodeLink{{Basename: "a.f.alpha", Rationale: priorRationale}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got := cli.ExportComputePrecedingLinks(tc.existing, newStart)
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

// TestComputePrecedingLinks_ActiveSetSortedByStart verifies the active set is
// emitted in F.start-ascending order and is stable.
func TestComputePrecedingLinks_ActiveSetSortedByStart(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const activeRationale = "preceding episode (active at this episode's start)"

	existing := []cli.ExportEpisodeRange{
		{Basename: "a.late", Start: "2026-05-25T11:50:00Z", End: "2026-05-25T13:00:00Z"},
		{Basename: "a.early", Start: "2026-05-25T11:00:00Z", End: "2026-05-25T13:00:00Z"},
		{Basename: "a.mid", Start: "2026-05-25T11:30:00Z", End: "2026-05-25T13:00:00Z"},
	}

	got := cli.ExportComputePrecedingLinks(existing, "2026-05-25T12:00:00Z")

	g.Expect(got).To(Equal([]cli.ExportEpisodeLink{
		{Basename: "a.early", Rationale: activeRationale},
		{Basename: "a.mid", Rationale: activeRationale},
		{Basename: "a.late", Rationale: activeRationale},
	}))
}

// TestComputePrecedingLinks_SkipsUnparseableRanges verifies an existing
// episode with an unparseable or empty range is silently skipped (it cannot
// be ordered against E.start).
func TestComputePrecedingLinks_SkipsUnparseableRanges(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const priorRationale = "immediately preceding episode"

	existing := []cli.ExportEpisodeRange{
		{Basename: "a.broken", Start: "not-a-time", End: "also-bad"},
		{Basename: "a.empty", Start: "", End: ""},
		{Basename: "a.good", Start: "2026-05-25T09:00:00Z", End: "2026-05-25T10:00:00Z"},
	}

	got := cli.ExportComputePrecedingLinks(existing, "2026-05-25T12:00:00Z")
	g.Expect(got).To(Equal([]cli.ExportEpisodeLink{{Basename: "a.good", Rationale: priorRationale}}))
}

// TestEngramLearn_Episode_BodyIsThreeSections drives the full write path and
// asserts the persisted episode note body has the three D6 sections.
func TestEngramLearn_Episode_BodyIsThreeSections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var written []byte

	deps := newCapturingLearnDeps(&written)

	args := newEpisodeLearnArgsForTest()
	args.Summary = "We did the work and learned a lesson."
	args.Relations = []string{"157|applied subtraction"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	body := string(written)
	g.Expect(body).To(ContainSubstring("## Summary\nWe did the work and learned a lesson."))
	g.Expect(body).To(ContainSubstring("## Transcript"))
	g.Expect(body).To(ContainSubstring("USER: hi"))
	g.Expect(body).To(ContainSubstring("## Related"))
	g.Expect(body).To(ContainSubstring("- [[157]] — applied subtraction"))
	g.Expect(body).NotTo(ContainSubstring("Related to:"))
}

// TestEngramLearn_Episode_ListEpisodesErrorSurfaces verifies a ListEpisodes
// scan failure aborts the write with a wrapped error.
func TestEngramLearn_Episode_ListEpisodesErrorSurfaces(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var written []byte

	deps := newCapturingLearnDeps(&written)
	deps.ListEpisodes = func(string) ([]cli.ExportEpisodeRange, error) {
		return nil, errStubListEpisodes
	}

	args := newEpisodeLearnArgsForTest()
	args.Summary = "Summary."

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(MatchError(ContainSubstring("listing episodes")))
}

// TestEngramLearn_Episode_PrecedingLinksFromScan verifies that when
// ListEpisodes reports an episode whose range spans the new episode's start,
// the new episode's ## Related section links it as active-at-start.
func TestEngramLearn_Episode_PrecedingLinksFromScan(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var written []byte

	deps := newCapturingLearnDeps(&written)
	deps.ListEpisodes = func(string) ([]cli.ExportEpisodeRange, error) {
		return []cli.ExportEpisodeRange{
			{
				Basename: "9p.2026-05-25.prior",
				Start:    "2026-05-25T21:00:00Z",
				End:      "2026-05-25T23:30:00Z",
			},
		}, nil
	}

	args := newEpisodeLearnArgsForTest()
	args.Summary = "Summary."
	// New episode range starts at 22:00, inside [21:00, 23:30].
	args.TranscriptRange = "2026-05-25T22:00:00Z..2026-05-25T22:30:00Z"

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	body := string(written)
	g.Expect(body).To(ContainSubstring(
		"- [[9p.2026-05-25.prior]] — preceding episode (active at this episode's start)",
	))
}

// TestEngramLearn_Episode_SummaryRequired verifies an empty/whitespace
// --summary is rejected at the runLearn layer.
func TestEngramLearn_Episode_SummaryRequired(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		summary string
	}{
		{name: "missing", summary: ""},
		{name: "whitespace", summary: "   "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			var written []byte

			deps := newCapturingLearnDeps(&written)

			args := newEpisodeLearnArgsForTest()
			args.Summary = tc.summary

			var stdout strings.Builder

			err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
			g.Expect(err).To(MatchError(ContainSubstring("summary")))
		})
	}
}

// TestRenderEpisodeBody_FencesTranscriptWithBackticks verifies a transcript
// containing a fenced code block (a ``` run) is wrapped in a LONGER backtick
// fence so it round-trips, AND that vaultgraph.ParseWikilinks (which skips
// fenced blocks) returns NO links from inside the transcript while still
// returning the ## Related links. This is the G5 goal: verbatim transcript
// text cannot manufacture graph edges.
func TestRenderEpisodeBody_FencesTranscriptWithBackticks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Transcript contains a triple-backtick code block AND a wikilink that
	// must NOT become a graph edge.
	transcript := "USER: paste\n```\ncode with [[should-not-link]] inside\n```\nASSISTANT: ok\n"

	fields := cli.ExportEpisodeFields{
		Summary:        "Summary.",
		TranscriptText: transcript,
		Preceding: []cli.ExportEpisodeLink{
			{Basename: "9a.2026-05-01.prior", Rationale: "immediately preceding episode"},
		},
	}

	body := cli.ExportRenderEpisodeBody(fields)

	// The fence must be longer than the longest backtick run inside (3),
	// i.e. at least 4 backticks, so the inner ``` does not close the block.
	g.Expect(body).To(ContainSubstring("````\nUSER: paste"))

	links := vaultgraph.ParseWikilinks([]byte(body))
	g.Expect(links).To(ContainElement("9a.2026-05-01.prior"))
	g.Expect(links).NotTo(ContainElement("should-not-link"))
}

// TestRenderEpisodeBody_OmitsEmptyRelated verifies that with no preceding
// links and no authored relations, the ## Related section is dropped
// entirely (no empty heading).
func TestRenderEpisodeBody_OmitsEmptyRelated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fields := cli.ExportEpisodeFields{
		Summary:        "Solo episode.",
		TranscriptText: "USER: hi\n",
	}

	body := cli.ExportRenderEpisodeBody(fields)

	g.Expect(body).To(ContainSubstring("## Summary"))
	g.Expect(body).To(ContainSubstring("## Transcript"))
	g.Expect(body).NotTo(ContainSubstring("## Related"))
}

// TestRenderEpisodeBody_PlainTranscriptUsesMinFence verifies a transcript with
// no backtick runs still gets a 3-backtick fence (the minimum).
func TestRenderEpisodeBody_PlainTranscriptUsesMinFence(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fields := cli.ExportEpisodeFields{
		Summary:        "Summary.",
		TranscriptText: "USER: plain text, no fences\n",
	}

	body := cli.ExportRenderEpisodeBody(fields)
	g.Expect(body).To(ContainSubstring("```\nUSER: plain text, no fences\n```"))
	g.Expect(body).NotTo(ContainSubstring("````"))
}

// TestRenderEpisodeBody_ThreeSections verifies the D6 episode body: a
// ## Summary section with the authored summary, a fenced ## Transcript
// section with the verbatim chunk, and a ## Related section listing the
// preceding-episode links then the authored relations.
func TestRenderEpisodeBody_ThreeSections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fields := cli.ExportEpisodeFields{
		Summary:        "We sharpened the spec and dispatched the work.",
		TranscriptText: "USER: do the thing\nASSISTANT: done\n",
		Preceding: []cli.ExportEpisodeLink{
			{Basename: "9a.2026-05-01.prior", Rationale: "immediately preceding episode"},
		},
		Relations: []string{"9b.2026-05-02.other|shared the design"},
	}

	body := cli.ExportRenderEpisodeBody(fields)

	g.Expect(body).To(ContainSubstring("## Summary\nWe sharpened the spec and dispatched the work."))
	g.Expect(body).To(ContainSubstring("## Transcript\n"))
	g.Expect(body).To(ContainSubstring("USER: do the thing"))
	g.Expect(body).To(ContainSubstring("## Related\n"))
	g.Expect(body).To(ContainSubstring("- [[9a.2026-05-01.prior]] — immediately preceding episode"))
	g.Expect(body).To(ContainSubstring("- [[9b.2026-05-02.other]] — shared the design"))
	// The episode body must NOT use the fact/feedback "Related to:" marker.
	g.Expect(body).NotTo(ContainSubstring("Related to:"))
}

// unexported variables.
var (
	errStubListEpisodes = errors.New("boom: list episodes")
)

// newCapturingLearnDeps returns LearnDeps wired to capture the written note
// content into dst, with no-op vault/lock/list deps and no embedder.
func newCapturingLearnDeps(dst *[]byte) cli.LearnDeps {
	return cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error { *dst = data; return nil },
	}
}

// newEpisodeLearnArgsForTest returns a minimal valid episode LearnArgs (no
// Summary set — callers set it per-case).
func newEpisodeLearnArgsForTest() cli.LearnArgs {
	return cli.LearnArgs{
		Type:              "episode",
		Slug:              "ep",
		Vault:             "/v",
		Position:          "top",
		Source:            "src",
		Situation:         "topic phrase",
		BoundaryRationale: "discrete arc",
		TranscriptText:    "USER: hi\nASSISTANT: hello\n",
		Sessions:          []string{"sess"},
		TranscriptRange:   "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
	}
}

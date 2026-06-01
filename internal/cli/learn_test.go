package cli_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestDefaultSessionPathResolver exercises the production resolver so
// coverage attributes a hit. The resolver concatenates HOME + projects/
// + cwd-slug + sessionID, returning a Claude Code JSONL path.
func TestDefaultSessionPathResolver(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path, err := cli.DefaultSessionPathResolverForTest("abc-123")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(path).To(HaveSuffix("/abc-123.jsonl"))
	g.Expect(path).To(ContainSubstring(".claude/projects/"))
}

// TestEngramLearn_Episode_AutoEmbedsSidecar verifies an episode write
// produces a `.vec.json` sidecar via the same auto-embed path
// facts/feedback use. Uses a fake embedder and captures the sidecar
// path/bytes that hit WriteSidecar.
func TestEngramLearn_Episode_AutoEmbedsSidecar(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var (
		sidecarPath  string
		sidecarBytes []byte
	)

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
		Embedder: successEmbedder{},
		WriteSidecar: func(path string, data []byte) error {
			sidecarPath = path
			sidecarBytes = data

			return nil
		},
		LogWarning: func(string, ...any) {
			t.Fatal("happy path should not warn")
		},
	}

	args := cli.LearnArgs{
		Type:              "episode",
		Slug:              "embed-shape",
		Vault:             "/v",
		Position:          "top",
		Source:            "src",
		Situation:         "embedding check",
		BoundaryRationale: "topic shift",
		TranscriptText:    "USER: hi\nASSISTANT: hello\n",
		Sessions:          []string{"sess"},
		TranscriptRange:   "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(sidecarPath).To(Equal("/v/Permanent/1.2026-05-25.embed-shape.vec.json"))

	var parsed embed.Sidecar
	g.Expect(json.Unmarshal(sidecarBytes, &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.EmbeddingModelID).To(Equal("m@4"))
	g.Expect(parsed.Dims).To(Equal(4))
	g.Expect(parsed.Vector).To(HaveLen(4))
	g.Expect(parsed.ContentHash).To(HavePrefix("sha256:"))
}

// TestEngramLearn_Episode_BoundaryRationaleRequired verifies the
// --boundary-rationale flag is required and rejects empty/whitespace.
func TestEngramLearn_Episode_BoundaryRationaleRequired(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
	}{
		{name: "empty", raw: ""},
		{name: "whitespace", raw: "   "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			deps := cli.LearnDeps{
				Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
				Getenv:   func(string) string { return "" },
				StatDir:  func(string) error { return nil },
				ListIDs:  func(string) ([]string, error) { return nil, nil },
				Lock:     func(string) (func(), error) { return func() {}, nil },
				WriteNew: func(string, []byte) error { return nil },
			}

			args := cli.LearnArgs{
				Type:              "episode",
				Slug:              "x",
				Vault:             "/v",
				Position:          "top",
				Source:            "src",
				Situation:         "s",
				BoundaryRationale: tc.raw,
				TranscriptText:    "USER: hi\n",
				Sessions:          []string{"sess"},
				TranscriptRange:   "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
			}

			var stdout strings.Builder

			err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
			g.Expect(err).To(MatchError(ContainSubstring("--boundary-rationale")))
		})
	}
}

// TestEngramLearn_Episode_ExactlyOneBodySource verifies the XOR contract:
// exactly one of --from-transcript-range or --transcript-text must be set.
// Neither errors; both errors. This lives at the args-to-args layer
// (runLearnFromEpisodeArgsWithReader), since the resolver enforces the
// XOR before runLearn sees a body string.
func TestEngramLearn_Episode_ExactlyOneBodySource(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		mutate      func(*cli.LearnEpisodeArgs)
		expectMatch string
	}{
		{
			name:        "neither set",
			mutate:      func(*cli.LearnEpisodeArgs) {},
			expectMatch: "exactly one of",
		},
		{
			name: "both set",
			mutate: func(a *cli.LearnEpisodeArgs) {
				a.TranscriptText = "USER: hi\n"
				a.FromTranscriptRange = []string{
					"sess-1:2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
				}
			},
			expectMatch: "mutually exclusive",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			args := cli.LearnEpisodeArgs{
				CommonLearnArgs: cli.CommonLearnArgs{
					Slug:     "x",
					Vault:    "/v",
					Position: "top",
					Source:   "src",
				},
				Situation:         "s",
				BoundaryRationale: "discrete arc",
				Sessions:          []string{"sess"},
				TranscriptRange:   "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
			}
			tc.mutate(&args)

			deps := cli.LearnDeps{
				Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
				Getenv:   func(string) string { return "" },
				StatDir:  func(string) error { return nil },
				ListIDs:  func(string) ([]string, error) { return nil, nil },
				Lock:     func(string) (func(), error) { return func() {}, nil },
				WriteNew: func(string, []byte) error { return nil },
			}

			var stdout strings.Builder

			err := cli.RunLearnFromEpisodeArgsWithReaderForTest(
				t.Context(), args,
				stubRangeReader{},
				func(string) (string, error) { return "/unused.jsonl", nil },
				"", deps, &stdout,
			)
			g.Expect(err).To(MatchError(ContainSubstring(tc.expectMatch)))
		})
	}
}

// TestEngramLearn_Episode_FromTranscriptRange_ReadsChunk verifies
// --from-transcript-range A:start..end uses the injected RangeReader to
// fetch the chunk and inlines it as the body.
func TestEngramLearn_Episode_FromTranscriptRange_ReadsChunk(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenContent []byte

	deps := cli.LearnDeps{
		Now:     func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(string) ([]string, error) { return nil, nil },
		Lock:    func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error {
			writtenContent = data

			return nil
		},
	}

	const fakeChunk = "USER: hello from range reader\nASSISTANT: chunk content here\n"

	reader := stubRangeReader{
		chunks: map[string]string{
			"/sessions/sess-1.jsonl": fakeChunk,
		},
	}

	args := cli.LearnEpisodeArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "range-read",
			Vault:    "/v",
			Position: "top",
			Source:   "src",
		},
		Situation:           "range read check",
		BoundaryRationale:   "discrete arc",
		FromTranscriptRange: []string{"sess-1:2026-05-25T22:00:00Z..2026-05-25T23:00:00Z"},
		Sessions:            []string{"sess-1"},
		TranscriptRange:     "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
	}

	var stdout strings.Builder

	sessionPath := func(id string) (string, error) {
		return "/sessions/" + id + ".jsonl", nil
	}

	err := cli.RunLearnFromEpisodeArgsWithReaderForTest(
		t.Context(), args, reader, sessionPath, "", deps, &stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(writtenContent)).To(ContainSubstring(fakeChunk))
}

// TestEngramLearn_Episode_FromTranscriptRange_RecordsTranscriptFile verifies
// that a --from-transcript-range episode records the resolved transcript file
// path in provenance.transcript_files, so the L1 note links back to its source.
func TestEngramLearn_Episode_FromTranscriptRange_RecordsTranscriptFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenContent []byte

	deps := cli.LearnDeps{
		Now:     func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(string) ([]string, error) { return nil, nil },
		Lock:    func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error {
			writtenContent = data

			return nil
		},
	}

	reader := stubRangeReader{
		chunks: map[string]string{
			"/sessions/sess-1.jsonl": "USER: hi\nASSISTANT: yo\n",
		},
	}

	args := cli.LearnEpisodeArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "range-file",
			Vault:    "/v",
			Position: "top",
			Source:   "src",
		},
		Situation:           "range file check",
		BoundaryRationale:   "discrete arc",
		FromTranscriptRange: []string{"sess-1:2026-05-25T22:00:00Z..2026-05-25T23:00:00Z"},
		Sessions:            []string{"sess-1"},
		TranscriptRange:     "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
	}

	var stdout strings.Builder

	sessionPath := func(id string) (string, error) {
		return "/sessions/" + id + ".jsonl", nil
	}

	err := cli.RunLearnFromEpisodeArgsWithReaderForTest(
		t.Context(), args, reader, sessionPath, "", deps, &stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(writtenContent)).To(ContainSubstring("transcript_files:"))
	g.Expect(string(writtenContent)).To(ContainSubstring("/sessions/sess-1.jsonl"))
}

// TestEngramLearn_Episode_FrontmatterShape verifies the rendered
// L1 episode frontmatter contains the spec-mandated keys
// (type=episode, situation, boundary_rationale, nested
// provenance.sessions and provenance.transcript_range, standard
// luhmann/created/source). The L1 spec drops the "outcomes" field.
func TestEngramLearn_Episode_FrontmatterShape(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenContent []byte

	deps := cli.LearnDeps{
		Now:     func() time.Time { return time.Date(2026, time.May, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(string) ([]string, error) { return nil, nil },
		Lock:    func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error {
			writtenContent = data

			return nil
		},
	}

	args := cli.LearnArgs{
		Type:              "episode",
		Slug:              "episode-shape",
		Vault:             "/vault",
		Position:          "top",
		Source:            "session log engram, 2026-05-25",
		Situation:         "Sharpening the F1 episode spec",
		BoundaryRationale: "Discrete sharpen-then-dispatch arc",
		TranscriptText:    "USER: please execute the spec\nASSISTANT: I'll set up the task list...\n",
		Sessions:          []string{"971fc252-8b44-4bd2-b44a-4f44464105eb"},
		TranscriptRange:   "2026-05-25T22:00:00Z..2026-05-25T23:30:00Z",
		Relations:         []string{"157|applied subtraction"},
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	body := string(writtenContent)
	// Frontmatter
	g.Expect(body).To(ContainSubstring("type: episode"))
	g.Expect(body).To(ContainSubstring("situation: Sharpening the F1 episode spec"))
	g.Expect(body).To(ContainSubstring("boundary_rationale: Discrete sharpen-then-dispatch arc"))
	g.Expect(body).To(ContainSubstring("provenance:"))
	g.Expect(body).To(ContainSubstring("sessions:"))
	g.Expect(body).To(ContainSubstring("- 971fc252-8b44-4bd2-b44a-4f44464105eb"))
	g.Expect(body).To(ContainSubstring("transcript_range:"))
	// --transcript-text episodes have no source file → transcript_files omitted.
	g.Expect(body).NotTo(ContainSubstring("transcript_files"))
	g.Expect(body).To(ContainSubstring(`start: "2026-05-25T22:00:00Z"`))
	g.Expect(body).To(ContainSubstring(`end: "2026-05-25T23:30:00Z"`))
	g.Expect(body).To(ContainSubstring(`luhmann: "1"`))
	g.Expect(body).To(ContainSubstring(`created: "2026-05-25"`))
	g.Expect(body).To(ContainSubstring("source: session log engram, 2026-05-25"))
	// L1 episode body: filtered transcript chunk verbatim + related block.
	// No auto-prefix lines, no Outcomes section, no narrative summary.
	g.Expect(body).NotTo(ContainSubstring("Information learned"))
	g.Expect(body).NotTo(ContainSubstring("Lesson learned"))
	g.Expect(body).NotTo(ContainSubstring("## Outcomes"))
	g.Expect(body).NotTo(ContainSubstring("outcomes:"))
	g.Expect(body).To(ContainSubstring("USER: please execute the spec"))
	g.Expect(body).To(ContainSubstring("ASSISTANT: I'll set up the task list..."))
	g.Expect(body).To(ContainSubstring("Related to:"))
	g.Expect(body).To(ContainSubstring("- [[157]] — applied subtraction."))
}

// TestEngramLearn_Episode_LuhmannPlacement exercises the three
// --position values (top, continuation, sibling) against a fixed
// existing-IDs list and verifies the computed filename has the correct
// Luhmann ID and the frontmatter's luhmann field matches.
func TestEngramLearn_Episode_LuhmannPlacement(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		target   string
		position string
		wantID   string
	}{
		{name: "top", position: "top", wantID: "11"},
		{name: "continuation", target: "1", position: "continuation", wantID: "1c"},
		{name: "sibling", target: "1b", position: "sibling", wantID: "1c"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			var (
				writtenPath    string
				writtenContent []byte
			)

			deps := cli.LearnDeps{
				Now:     func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
				Getenv:  func(string) string { return "" },
				StatDir: func(string) error { return nil },
				ListIDs: func(string) ([]string, error) {
					return []string{"1", "1a", "1b", "2", "10"}, nil
				},
				Lock: func(string) (func(), error) { return func() {}, nil },
				WriteNew: func(path string, data []byte) error {
					writtenPath = path
					writtenContent = data

					return nil
				},
			}

			args := cli.LearnArgs{
				Type:              "episode",
				Slug:              "placement",
				Vault:             "/v",
				Target:            tc.target,
				Position:          tc.position,
				Source:            "src",
				Situation:         "ordering",
				BoundaryRationale: "discrete arc",
				TranscriptText:    "USER: ping\nASSISTANT: pong\n",
				Sessions:          []string{"sess"},
				TranscriptRange:   "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
			}

			var stdout strings.Builder

			err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			expectedPath := "/v/Permanent/" + tc.wantID + ".2026-05-25.placement.md"
			g.Expect(writtenPath).To(Equal(expectedPath))
			g.Expect(string(writtenContent)).To(ContainSubstring(`luhmann: "` + tc.wantID + `"`))
		})
	}
}

// TestEngramLearn_Episode_Opencode_RecordsDBPath verifies an episode sourced
// from an opencode://<id> session reads via the dispatched reader (the URI as
// the read handle) and records the OpenCode DB file path — not a .jsonl path —
// in provenance.transcript_files. The session id stays the pk in sessions.
func TestEngramLearn_Episode_Opencode_RecordsDBPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenContent []byte

	deps := cli.LearnDeps{
		Now:     func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(string) ([]string, error) { return nil, nil },
		Lock:    func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error {
			writtenContent = data

			return nil
		},
	}

	reader := stubRangeReader{
		chunks: map[string]string{
			"opencode://ses_oc": "USER: opencode chunk\nASSISTANT: yes\n",
		},
	}

	args := cli.LearnEpisodeArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "oc-range",
			Vault:    "/v",
			Position: "top",
			Source:   "src",
		},
		Situation:           "opencode range check",
		BoundaryRationale:   "discrete arc",
		FromTranscriptRange: []string{"opencode://ses_oc:2026-05-25T22:00:00Z..2026-05-25T23:00:00Z"},
		Sessions:            []string{"ses_oc"},
		TranscriptRange:     "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
	}

	var stdout strings.Builder

	// Claude-only resolver — must NOT be used for an opencode:// session.
	sessionPath := func(id string) (string, error) {
		return "/claude/" + id + ".jsonl", nil
	}

	dbPath := "/home/u/.local/share/opencode/opencode.db"

	err := cli.RunLearnFromEpisodeArgsWithReaderForTest(
		t.Context(), args, reader, sessionPath, dbPath, deps, &stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	body := string(writtenContent)
	g.Expect(body).To(ContainSubstring("opencode chunk"))
	g.Expect(body).To(ContainSubstring("transcript_files:"))
	g.Expect(body).To(ContainSubstring(dbPath))
	g.Expect(body).NotTo(ContainSubstring("/claude/"))
}

// TestEngramLearn_Episode_RequiredFields covers required-field validation
// at the runLearn layer (situation, boundary-rationale, session,
// transcript-range — all enforced before the body source check kicks in,
// since runLearnFromEpisodeArgsWithReader's resolver enforces XOR
// separately). Pre-populates TranscriptText so the body-source XOR
// gate passes.
func TestEngramLearn_Episode_RequiredFields(t *testing.T) {
	t.Parallel()

	baseArgs := func() cli.LearnArgs {
		return cli.LearnArgs{
			Type:              "episode",
			Slug:              "x",
			Vault:             "/v",
			Position:          "top",
			Source:            "src",
			Situation:         "s",
			BoundaryRationale: "discrete arc",
			TranscriptText:    "USER: hi\nASSISTANT: hello\n",
			Sessions:          []string{"sess"},
			TranscriptRange:   "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
		}
	}

	cases := []struct {
		name      string
		mutate    func(*cli.LearnArgs)
		expectMsg string
	}{
		{
			name:      "missing --situation",
			mutate:    func(a *cli.LearnArgs) { a.Situation = "" },
			expectMsg: "--situation",
		},
		{
			name:      "whitespace --situation",
			mutate:    func(a *cli.LearnArgs) { a.Situation = "   " },
			expectMsg: "--situation",
		},
		{
			name:      "missing --boundary-rationale",
			mutate:    func(a *cli.LearnArgs) { a.BoundaryRationale = "" },
			expectMsg: "--boundary-rationale",
		},
		{
			name:      "missing --session",
			mutate:    func(a *cli.LearnArgs) { a.Sessions = nil },
			expectMsg: "--session",
		},
		{
			name:      "empty --session entry",
			mutate:    func(a *cli.LearnArgs) { a.Sessions = []string{""} },
			expectMsg: "--session",
		},
		{
			name:      "missing --transcript-range",
			mutate:    func(a *cli.LearnArgs) { a.TranscriptRange = "" },
			expectMsg: "transcript-range",
		},
		{
			name:      "malformed --transcript-range (no separator)",
			mutate:    func(a *cli.LearnArgs) { a.TranscriptRange = "2026-05-25T22:00:00Z" },
			expectMsg: "transcript-range",
		},
		{
			name:      "unparseable RFC3339 start",
			mutate:    func(a *cli.LearnArgs) { a.TranscriptRange = "yesterday..2026-05-25T23:00:00Z" },
			expectMsg: "transcript-range",
		},
		{
			name:      "unparseable RFC3339 end",
			mutate:    func(a *cli.LearnArgs) { a.TranscriptRange = "2026-05-25T22:00:00Z..nope" },
			expectMsg: "transcript-range",
		},
		{
			name:      "start > end",
			mutate:    func(a *cli.LearnArgs) { a.TranscriptRange = "2026-05-25T23:00:00Z..2026-05-25T22:00:00Z" },
			expectMsg: "transcript-range",
		},
		{
			name:      "start == end",
			mutate:    func(a *cli.LearnArgs) { a.TranscriptRange = "2026-05-25T22:00:00Z..2026-05-25T22:00:00Z" },
			expectMsg: "transcript-range",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			deps := cli.LearnDeps{
				Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
				Getenv:   func(string) string { return "" },
				StatDir:  func(string) error { return nil },
				ListIDs:  func(string) ([]string, error) { return nil, nil },
				Lock:     func(string) (func(), error) { return func() {}, nil },
				WriteNew: func(string, []byte) error { return nil },
			}

			args := baseArgs()
			tc.mutate(&args)

			var stdout strings.Builder

			err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
			g.Expect(err).To(MatchError(ContainSubstring(tc.expectMsg)))
		})
	}
}

// TestEngramLearn_Episode_TranscriptTextInlined verifies --transcript-text
// produces a body containing the literal text verbatim (no narrative
// rewrite, no Outcomes section).
func TestEngramLearn_Episode_TranscriptTextInlined(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenContent []byte

	deps := cli.LearnDeps{
		Now:     func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(string) ([]string, error) { return nil, nil },
		Lock:    func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error {
			writtenContent = data

			return nil
		},
	}

	const literal = "USER: build the spec\nASSISTANT: starting now\n[tool] Edit(...)\n"

	args := cli.LearnEpisodeArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "literal-body",
			Vault:    "/v",
			Position: "top",
			Source:   "src",
		},
		Situation:         "literal body check",
		BoundaryRationale: "verbatim chunk",
		TranscriptText:    literal,
		Sessions:          []string{"sess"},
		TranscriptRange:   "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
	}

	var stdout strings.Builder

	err := cli.RunLearnFromEpisodeArgsWithReaderForTest(
		t.Context(), args,
		stubRangeReader{},
		func(string) (string, error) { return "/unused.jsonl", nil },
		"", deps, &stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(writtenContent)).To(ContainSubstring(literal))
}

// TestEpisode_FromTranscriptRange_ParseErrorPropagates verifies that a
// malformed --from-transcript-range argument bubbles up cleanly through
// the args-to-args resolver layer.
func TestEpisode_FromTranscriptRange_ParseErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	args := cli.LearnEpisodeArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "bad-range",
			Vault:    "/v",
			Position: "top",
			Source:   "src",
		},
		Situation:           "x",
		BoundaryRationale:   "discrete arc",
		FromTranscriptRange: []string{"not-a-range"},
		Sessions:            []string{"sess"},
		TranscriptRange:     "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
	}

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
	}

	var stdout strings.Builder

	err := cli.RunLearnFromEpisodeArgsWithReaderForTest(
		t.Context(), args,
		stubRangeReader{},
		func(string) (string, error) { return "/unused.jsonl", nil },
		"", deps, &stdout,
	)
	g.Expect(err).To(MatchError(ContainSubstring("from-transcript-range")))
}

func TestExtractLuhmannFromFilename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got, ok := cli.ExportExtractLuhmannFromFilename("1a3.2026-05-09.subagent-recovery.md")
	g.Expect(ok).To(BeTrue())
	g.Expect(got).To(Equal("1a3"))
}

func TestExtractLuhmannFromFilename_RejectsBadFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	_, ok := cli.ExportExtractLuhmannFromFilename("README.md")
	g.Expect(ok).To(BeFalse())
}

func TestExtractLuhmannFromFilename_RejectsNonMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	_, ok := cli.ExportExtractLuhmannFromFilename("1a3.2026-05-09.subagent-recovery.txt")
	g.Expect(ok).To(BeFalse())
}

func TestLearnPath_Permanent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportLearnPath("/vault", "1a3", "subagent-driven-recovery", when)
	g.Expect(got).To(Equal("/vault/Permanent/1a3.2026-05-09.subagent-driven-recovery.md"))
}

// TestMarshalFrontmatter_WrapsValidValue verifies the helper produces the
// expected "---"-delimited block. Error returns are unreachable for the
// typed-string struct callers used in production, so only the happy path is
// covered here.
func TestMarshalFrontmatter_WrapsValidValue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportMarshalFrontmatter(map[string]string{"k": "v"})
	g.Expect(got).To(Equal("---\nk: v\n---\n\n"))
}

// TestParseFromTranscriptRange_ErrorBranches drives every malformed
// input branch so coverage attributes hits to each error path.
func TestParseFromTranscriptRange_ErrorBranches(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		raw       string
		expectMsg string
	}{
		{
			name:      "empty",
			raw:       "",
			expectMsg: "from-transcript-range",
		},
		{
			name:      "no separator",
			raw:       "sess-1:2026-05-25T22:00:00Z",
			expectMsg: "from-transcript-range",
		},
		{
			name:      "no end after ..",
			raw:       "sess-1:2026-05-25T22:00:00Z..",
			expectMsg: "from-transcript-range",
		},
		{
			name:      "no colon (missing session-id)",
			raw:       "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
			expectMsg: "from-transcript-range",
		},
		{
			name:      "empty session-id",
			raw:       ":2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
			expectMsg: "from-transcript-range",
		},
		{
			name:      "unparseable start",
			raw:       "sess-1:yesterday..2026-05-25T23:00:00Z",
			expectMsg: "from-transcript-range",
		},
		{
			name:      "unparseable end",
			raw:       "sess-1:2026-05-25T22:00:00Z..nope",
			expectMsg: "from-transcript-range",
		},
		{
			name:      "start equals end",
			raw:       "sess-1:2026-05-25T22:00:00Z..2026-05-25T22:00:00Z",
			expectMsg: "start must be before end",
		},
		{
			name:      "start after end",
			raw:       "sess-1:2026-05-25T23:00:00Z..2026-05-25T22:00:00Z",
			expectMsg: "start must be before end",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			_, _, _, err := cli.ParseFromTranscriptRangeForTest(tc.raw)
			g.Expect(err).To(MatchError(ContainSubstring(tc.expectMsg)))
		})
	}
}

// TestParseFromTranscriptRange_HappyPath verifies the parser splits
// "<session-id>:<RFC3339-start>..<RFC3339-end>" correctly. The session
// ID is the literal text before the first colon; the start/end are
// RFC3339 timestamps (which themselves contain colons).
func TestParseFromTranscriptRange_HappyPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := "971fc252-8b44-4bd2-b44a-4f44464105eb:2026-05-25T22:00:00Z..2026-05-25T23:30:00Z"

	sessionID, start, end, err := cli.ParseFromTranscriptRangeForTest(raw)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(sessionID).To(Equal("971fc252-8b44-4bd2-b44a-4f44464105eb"))
	g.Expect(start.Format(time.RFC3339)).To(Equal("2026-05-25T22:00:00Z"))
	g.Expect(end.Format(time.RFC3339)).To(Equal("2026-05-25T23:30:00Z"))
}

func TestRenderBody_Fact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderFactBody(cli.ExportFactFields{
		Situation: "reasoning about agent coordination",
		Subject:   "subagent dispatch",
		Predicate: "is fundamentally",
		Object:    "a verification problem dressed as coordination",
	}, "Related to:\n- [[X]] — adjacent.\n")
	g.Expect(got).To(Equal(
		"Information learned: when in reasoning about agent coordination, " +
			"subagent dispatch is fundamentally a verification problem dressed as coordination.\n" +
			"\n" +
			"Related to:\n- [[X]] — adjacent.\n"))
}

func TestRenderBody_Feedback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	action := "set up a task list with self-contained briefs and dispatch; " +
		"if a small model cannot finish a subtask, shrink the task"
	got := cli.ExportRenderFeedbackBody(cli.ExportFeedbackFields{
		Situation: "orchestrating multi-step work as the main LLM under context pressure",
		Action:    action,
	}, "Related to:\n- [[1a.foo]] — same shape.\n- [[5.bar]] — the MOC.\n")
	g.Expect(got).To(Equal(
		"Lesson learned: when orchestrating multi-step work as the main LLM under context pressure, " +
			action + ".\n" +
			"\n" +
			"Related to:\n- [[1a.foo]] — same shape.\n- [[5.bar]] — the MOC.\n",
	))
}

func TestRenderEpisodeFrontmatter_EmitsProjectAndIssueBelowSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportEpisodeFields{
		Situation:         "s",
		BoundaryRationale: "br",
		TranscriptText:    "tx",
		Sessions:          []string{"sess-1"},
		TranscriptStart:   "2026-05-26T00:00:00Z",
		TranscriptEnd:     "2026-05-26T01:00:00Z",
		Luhmann:           "1",
		Source:            "src",
		Project:           "engram",
		Issue:             "636",
	}
	got := cli.ExportRenderEpisodeFrontmatter(fields, when)
	g.Expect(got).To(ContainSubstring("project: engram\n"))
	g.Expect(got).To(ContainSubstring("issue: \"636\"\n"))

	srcIdx := strings.Index(got, "source:")
	projIdx := strings.Index(got, "project:")
	g.Expect(srcIdx).To(BeNumerically("<", projIdx))
}

func TestRenderEpisodeFrontmatter_OmitsProjectAndIssueWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportEpisodeFields{
		Situation:         "s",
		BoundaryRationale: "br",
		TranscriptText:    "tx",
		Sessions:          []string{"sess-1"},
		TranscriptStart:   "2026-05-26T00:00:00Z",
		TranscriptEnd:     "2026-05-26T01:00:00Z",
		Luhmann:           "1",
		Source:            "src",
	}
	got := cli.ExportRenderEpisodeFrontmatter(fields, when)
	g.Expect(got).NotTo(ContainSubstring("project:"))
	g.Expect(got).NotTo(ContainSubstring("issue:"))
}

// TestRenderFactBody_StripsLeadingWhenFromSituation is the fact-type variant of
// the double-"when" bug guard.
func TestRenderFactBody_StripsLeadingWhenFromSituation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderFactBody(cli.ExportFactFields{
		Situation: "When reasoning about agent coordination",
		Subject:   "subagent dispatch",
		Predicate: "is fundamentally",
		Object:    "a verification problem",
	}, "")
	g.Expect(got).
		To(HavePrefix("Information learned: when in reasoning about agent coordination, " +
			"subagent dispatch is fundamentally a verification problem."))
	g.Expect(got).NotTo(ContainSubstring("when in When"))
}

func TestRenderFactFrontmatter_EmitsProjectAndIssueBelowSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFactFields{
		Situation: "s", Subject: "subj", Predicate: "pred", Object: "obj",
		Luhmann: "1", Source: "src",
		Project: "engram", Issue: "636",
	}
	got := cli.ExportRenderFactFrontmatter(fields, when)
	g.Expect(got).To(ContainSubstring("source: src\n"))
	g.Expect(got).To(ContainSubstring("project: engram\n"))
	g.Expect(got).To(ContainSubstring("issue: \"636\"\n"))

	srcIdx := strings.Index(got, "source:")
	projIdx := strings.Index(got, "project:")
	issueIdx := strings.Index(got, "issue:")

	g.Expect(srcIdx).To(BeNumerically("<", projIdx))
	g.Expect(projIdx).To(BeNumerically("<", issueIdx))
}

func TestRenderFactFrontmatter_OmitsProjectAndIssueWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFactFields{
		Situation: "s", Subject: "subj", Predicate: "pred", Object: "obj",
		Luhmann: "1", Source: "src",
	}
	got := cli.ExportRenderFactFrontmatter(fields, when)
	g.Expect(got).NotTo(ContainSubstring("project:"))
	g.Expect(got).NotTo(ContainSubstring("issue:"))
}

// TestRenderFactFrontmatter_SafelyEncodesTrickyValues mirrors the feedback
// safety check for the fact frontmatter.
func TestRenderFactFrontmatter_SafelyEncodesTrickyValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFactFields{
		Situation: "context: tricky",
		Subject:   "- subj",
		Predicate: "is\nmultiline",
		Object:    "* obj",
		Luhmann:   "11",
		Source:    "src",
	}
	got := cli.ExportRenderFactFrontmatter(fields, when)
	parsed := parseFrontmatter(t, got)
	g.Expect(parsed["situation"]).To(Equal(fields.Situation))
	g.Expect(parsed["subject"]).To(Equal(fields.Subject))
	g.Expect(parsed["predicate"]).To(Equal(fields.Predicate))
	g.Expect(parsed["object"]).To(Equal(fields.Object))
}

// TestRenderFeedbackBody_StripsLeadingWhenFromSituation guards against the
// double-"when" bug where the body template prepended "when " to a situation
// that already started with "When" — producing "Lesson learned: when When ...".
func TestRenderFeedbackBody_StripsLeadingWhenFromSituation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderFeedbackBody(cli.ExportFeedbackFields{
		Situation: "When writing concurrent Go code",
		Action:    "check ctx.Done()",
	}, "")
	g.Expect(got).
		To(HavePrefix("Lesson learned: when writing concurrent Go code, check ctx.Done()."))
	g.Expect(got).NotTo(ContainSubstring("when When"))
}

func TestRenderFeedbackFrontmatter_EmitsProjectAndIssueBelowSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFeedbackFields{
		Situation: "s", Behavior: "b", Impact: "i", Action: "a",
		Luhmann: "1", Source: "src",
		Project: "engram", Issue: "636",
	}
	got := cli.ExportRenderFeedbackFrontmatter(fields, when)
	g.Expect(got).To(ContainSubstring("project: engram\n"))
	g.Expect(got).To(ContainSubstring("issue: \"636\"\n"))

	srcIdx := strings.Index(got, "source:")
	projIdx := strings.Index(got, "project:")
	g.Expect(srcIdx).To(BeNumerically("<", projIdx))
}

// TestRenderFeedbackFrontmatter_LuhmannIsQuoted guards against yaml.v3's
// default behavior of emitting alphanumeric scalars unquoted. The vault
// convention is luhmann: "<id>" (double-quoted) so reads stay consistent
// across hand-written, migrated, and engram-learn-written notes; the existing
// pre-migration vault and the 218 migrated notes all quote this field.
func TestRenderFeedbackFrontmatter_LuhmannIsQuoted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportRenderFeedbackFrontmatter(cli.ExportFeedbackFields{
		Situation: "x", Behavior: "x", Impact: "x", Action: "x",
		Luhmann: "9aa", Source: "src",
	}, when)
	g.Expect(got).To(ContainSubstring(`luhmann: "9aa"`))
}

func TestRenderFeedbackFrontmatter_OmitsProjectAndIssueWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 26, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFeedbackFields{
		Situation: "s", Behavior: "b", Impact: "i", Action: "a",
		Luhmann: "1", Source: "src",
	}
	got := cli.ExportRenderFeedbackFrontmatter(fields, when)
	g.Expect(got).NotTo(ContainSubstring("project:"))
	g.Expect(got).NotTo(ContainSubstring("issue:"))
}

// TestRenderFeedbackFrontmatter_RoundtripFidelity is a property test: for any
// printable string values, the rendered frontmatter parses back to the same
// values. This is the invariant the YAML library buys us — verify it holds
// across the input space, not just hand-picked examples.
func TestRenderFeedbackFrontmatter_RoundtripFidelity(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		// Restricted to printable ASCII plus newline. Tab is excluded because
		// yaml.v3's block-scalar emitter and parser disagree about indented
		// tabs; CLI flag values don't carry tabs in practice, so this is not
		// a meaningful gap for engram learn.
		gen := rapid.StringMatching(`[ -~\n]{0,40}`)
		fields := cli.ExportFeedbackFields{
			Situation: gen.Draw(rt, "situation"),
			Behavior:  gen.Draw(rt, "behavior"),
			Impact:    gen.Draw(rt, "impact"),
			Action:    gen.Draw(rt, "action"),
			Luhmann:   rapid.StringMatching(`[0-9][0-9a-z]{0,3}`).Draw(rt, "luhmann"),
			Source:    gen.Draw(rt, "source"),
		}
		when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
		got := cli.ExportRenderFeedbackFrontmatter(fields, when)

		// Use Unmarshal directly to surface decode errors as property failures.
		const delim = "---\n"

		body := strings.TrimPrefix(got, delim)
		end := strings.Index(body, "\n"+delim)

		if end < 0 {
			rt.Fatalf("no closing delimiter in %q", got)
		}

		parsed := map[string]string{}

		if err := yaml.Unmarshal([]byte(body[:end+1]), &parsed); err != nil {
			rt.Fatalf("unmarshal %q: %v", body[:end+1], err)
		}

		for key, want := range map[string]string{
			"situation": fields.Situation, "behavior": fields.Behavior,
			"impact": fields.Impact, "action": fields.Action,
			"luhmann": fields.Luhmann, "source": fields.Source,
		} {
			if parsed[key] != want {
				rt.Fatalf("%s: got %q want %q\nfull:\n%s", key, parsed[key], want, got)
			}
		}
	})
}

// TestRenderFeedbackFrontmatter_SafelyEncodesTrickyValues verifies that values
// containing YAML-significant characters (newlines, colons, leading dashes,
// asterisks) survive a roundtrip — the original bug was that raw string
// concatenation let a multi-line Behavior end the frontmatter mid-document.
func TestRenderFeedbackFrontmatter_SafelyEncodesTrickyValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFeedbackFields{
		Situation: "writing tests: a guide",
		Behavior:  "first line\nsecond line",
		Impact:    "- leading dash list marker",
		Action:    "* alias-looking marker",
		Luhmann:   "11",
		Source:    "src: with colon",
	}
	got := cli.ExportRenderFeedbackFrontmatter(fields, when)
	parsed := parseFrontmatter(t, got)
	g.Expect(parsed["situation"]).To(Equal(fields.Situation))
	g.Expect(parsed["behavior"]).To(Equal(fields.Behavior))
	g.Expect(parsed["impact"]).To(Equal(fields.Impact))
	g.Expect(parsed["action"]).To(Equal(fields.Action))
	g.Expect(parsed["luhmann"]).To(Equal(fields.Luhmann))
	g.Expect(parsed["source"]).To(Equal(fields.Source))
}

func TestRenderFrontmatter_Fact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportRenderFactFrontmatter(cli.ExportFactFields{
		Situation: "reasoning about agent coordination",
		Subject:   "subagent dispatch",
		Predicate: "is fundamentally",
		Object:    "a verification problem dressed as coordination",
		Luhmann:   "11",
		Source:    "session log bar, 2026-05-09 13:00 UTC",
	}, when)
	parsed := parseFrontmatter(t, got)
	g.Expect(parsed["type"]).To(Equal("fact"))
	g.Expect(parsed["subject"]).To(Equal("subagent dispatch"))
	g.Expect(parsed["luhmann"]).To(Equal("11"))
	g.Expect(parsed["created"]).To(Equal("2026-05-09"))
}

func TestRenderFrontmatter_Feedback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC)
	got := cli.ExportRenderFeedbackFrontmatter(cli.ExportFeedbackFields{
		Situation: "writing concurrent Go code with context",
		Behavior:  "ignoring context cancellation",
		Impact:    "leaks goroutines on shutdown",
		Action:    "always check ctx.Done() in select loops",
		Luhmann:   "9z",
		Source:    "session log foo, 2026-05-09 12:00 UTC",
	}, when)
	parsed := parseFrontmatter(t, got)
	g.Expect(parsed).To(Equal(map[string]string{
		"type":      "feedback",
		"situation": "writing concurrent Go code with context",
		"behavior":  "ignoring context cancellation",
		"impact":    "leaks goroutines on shutdown",
		"action":    "always check ctx.Done() in select loops",
		"luhmann":   "9z",
		"created":   "2026-05-09",
		"source":    "session log foo, 2026-05-09 12:00 UTC",
	}))
}

func TestRenderRelatedSection_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportRenderRelatedSection(nil)).To(Equal(""))
}

func TestRenderRelatedSection_MultipleEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderRelatedSection([]string{
		"1a.foo|same shape",
		"5.bar | the MOC",
	})
	g.Expect(got).To(Equal(
		"Related to:\n- [[1a.foo]] — same shape.\n- [[5.bar]] — the MOC.\n"))
}

func TestRenderRelatedSection_NoPipeMeansEmptyRationale(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	got := cli.ExportRenderRelatedSection([]string{"7"})
	g.Expect(got).To(Equal("Related to:\n- [[7]] — .\n"))
}

// TestResolveSessionPath_GetwdError covers the getwd error branch.
func TestResolveSessionPath_GetwdError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := cli.ResolveSessionPathForTest(
		"abc-123",
		func() (string, error) { return "", errors.New("cwd boom") },
		func() (string, error) { return "/Users/joe", nil },
	)
	g.Expect(err).To(MatchError(ContainSubstring("getwd")))
	g.Expect(err).To(MatchError(ContainSubstring("cwd boom")))
}

// TestResolveSessionPath_HappyPath exercises the injected resolver with
// fake getwd/homeDir that return successful values.
func TestResolveSessionPath_HappyPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path, err := cli.ResolveSessionPathForTest(
		"abc-123",
		func() (string, error) { return "/Users/joe/repos/x", nil },
		func() (string, error) { return "/Users/joe", nil },
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(path).To(Equal("/Users/joe/.claude/projects/-Users-joe-repos-x/abc-123.jsonl"))
}

// TestResolveSessionPath_HomeError covers the homeDir error branch.
func TestResolveSessionPath_HomeError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := cli.ResolveSessionPathForTest(
		"abc-123",
		func() (string, error) { return "/Users/joe", nil },
		func() (string, error) { return "", errors.New("home boom") },
	)
	g.Expect(err).To(MatchError(ContainSubstring("home dir")))
	g.Expect(err).To(MatchError(ContainSubstring("home boom")))
}

func TestRunLearn_BootstrapsVaultWhenMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	initCalled := false
	deps := cli.LearnDeps{
		Now:       func() time.Time { return time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC) },
		Getenv:    func(string) string { return "" },
		StatDir:   func(string) error { return fs.ErrNotExist },
		InitVault: func(string) error { initCalled = true; return nil },
		ListIDs:   func(string) ([]string, error) { return nil, nil },
		Lock:      func(string) (func(), error) { return func() {}, nil },
		WriteNew:  func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{
		Type:     "feedback",
		Slug:     "x",
		Vault:    "/v",
		Position: "top",
		Source:   "test",
	}

	var stdout strings.Builder

	g.Expect(cli.ExportRunLearn(t.Context(), args, deps, &stdout)).To(Succeed())
	g.Expect(initCalled).To(BeTrue())
}

func TestRunLearn_Fact_WritesExpectedFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var (
		writtenPath    string
		writtenContent []byte
	)

	deps := cli.LearnDeps{
		Now:     func() time.Time { return time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC) },
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(string) ([]string, error) { return nil, nil },
		Lock:    func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(path string, data []byte) error {
			writtenPath = path
			writtenContent = data

			return nil
		},
	}

	args := cli.LearnArgs{
		Type:      "fact",
		Slug:      "fact-slug",
		Vault:     "/vault",
		Position:  "top",
		Situation: "s",
		Subject:   "subj",
		Predicate: "is",
		Object:    "obj",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writtenPath).To(Equal("/vault/Permanent/1.2026-05-09.fact-slug.md"))
	g.Expect(string(writtenContent)).To(ContainSubstring("type: fact"))
	g.Expect(string(writtenContent)).To(ContainSubstring("Information learned"))
}

func TestRunLearn_Feedback_WritesExpectedFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var (
		lockAcquired, lockReleased bool
		writtenPath                string
		writtenContent             []byte
	)

	deps := cli.LearnDeps{
		Now:     func() time.Time { return time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC) },
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		ListIDs: func(string) ([]string, error) {
			return []string{"1", "2"}, nil
		},
		Lock: func(string) (func(), error) {
			lockAcquired = true

			return func() { lockReleased = true }, nil
		},
		WriteNew: func(path string, data []byte) error {
			writtenPath = path
			writtenContent = data

			return nil
		},
	}

	args := cli.LearnArgs{
		Type:      "feedback",
		Slug:      "ctx-cancellation-rule",
		Vault:     "/vault",
		Target:    "",
		Position:  "top",
		Source:    "session log foo, 2026-05-09 12:00 UTC",
		Situation: "writing concurrent Go code",
		Behavior:  "ignoring ctx.Done()",
		Impact:    "leaks goroutines",
		Action:    "always check ctx.Done() in select",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(lockAcquired).To(BeTrue())
	g.Expect(lockReleased).To(BeTrue())
	g.Expect(writtenPath).To(Equal("/vault/Permanent/3.2026-05-09.ctx-cancellation-rule.md"))
	g.Expect(string(writtenContent)).To(ContainSubstring("type: feedback"))
	g.Expect(string(writtenContent)).
		To(ContainSubstring("Lesson learned: when writing concurrent Go code"))
}

func TestRunLearn_PropagatesListIDsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, errors.New("io fail") },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{Type: "fact", Slug: "x", Vault: "/v", Position: "top"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(MatchError(ContainSubstring("listing existing IDs")))
}

func TestRunLearn_PropagatesLockError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, time.May, 9, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return nil, errors.New("locked") },
		WriteNew: func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{Type: "fact", Slug: "x", Vault: "/v", Position: "top"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(MatchError(ContainSubstring("acquiring lock")))
}

func TestRunLearn_PropagatesStatDirError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.LearnDeps{
		Now:      time.Now,
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return errors.New("nope") },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{Type: "fact", Slug: "x", Vault: "/v", Position: "top"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(MatchError(ContainSubstring("vault")))
}

func TestRunLearn_RejectsInvalidSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.LearnDeps{
		Now:      time.Now,
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{Type: "fact", Slug: "Bad Slug", Vault: "/v", Position: "top"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}

func TestRunLearn_RejectsUnknownType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	deps := cli.LearnDeps{
		Now:       time.Now,
		Getenv:    func(string) string { return "" },
		StatDir:   func(string) error { return nil },
		InitVault: func(string) error { return nil },
		ListIDs:   func(string) ([]string, error) { return nil, nil },
		Lock:      func(string) (func(), error) { return func() {}, nil },
		WriteNew:  func(string, []byte) error { return nil },
	}
	args := cli.LearnArgs{Type: "principle", Slug: "x", Vault: "/v", Position: "top"}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}

// TestTierFrontmatter_BadTierRejected verifies that an invalid --tier value
// returns errLearnBadTier.
func TestTierFrontmatter_BadTierRejected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(string, []byte) error { return nil },
	}

	args := cli.LearnArgs{
		Type:      "fact",
		Slug:      "tier-bad",
		Vault:     "/v",
		Position:  "top",
		Source:    "src",
		Situation: "tier bad check",
		Subject:   "subj",
		Predicate: "pred",
		Object:    "obj",
		Tier:      "L9",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).To(MatchError(cli.ErrLearnBadTierForTest))
}

// TestTierFrontmatter_EpisodeDefaultsToL1 verifies that a rendered episode
// note carries tier: L1 derived from its type.
func TestTierFrontmatter_EpisodeDefaultsToL1(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenContent []byte

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error { writtenContent = data; return nil },
	}

	args := cli.LearnArgs{
		Type:              "episode",
		Slug:              "tier-episode",
		Vault:             "/v",
		Position:          "top",
		Source:            "src",
		Situation:         "tier check",
		BoundaryRationale: "discrete arc",
		TranscriptText:    "USER: hi\nASSISTANT: hello\n",
		Sessions:          []string{"sess"},
		TranscriptRange:   "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(writtenContent)).To(ContainSubstring("tier: L1"))
}

// TestTierFrontmatter_FactDefaultsToL2 verifies that a rendered fact note
// carries tier: L2 derived from its type.
func TestTierFrontmatter_FactDefaultsToL2(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenContent []byte

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error { writtenContent = data; return nil },
	}

	args := cli.LearnArgs{
		Type:      "fact",
		Slug:      "tier-fact",
		Vault:     "/v",
		Position:  "top",
		Source:    "src",
		Situation: "tier check",
		Subject:   "subj",
		Predicate: "pred",
		Object:    "obj",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(writtenContent)).To(ContainSubstring("tier: L2"))
}

// TestTierFrontmatter_FeedbackDefaultsToL2 verifies that a rendered feedback
// note carries tier: L2 derived from its type.
func TestTierFrontmatter_FeedbackDefaultsToL2(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenContent []byte

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error { writtenContent = data; return nil },
	}

	args := cli.LearnArgs{
		Type:      "feedback",
		Slug:      "tier-feedback",
		Vault:     "/v",
		Position:  "top",
		Source:    "src",
		Situation: "tier check",
		Behavior:  "beh",
		Impact:    "imp",
		Action:    "act",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(writtenContent)).To(ContainSubstring("tier: L2"))
}

// TestTierFrontmatter_OverrideL3 verifies that --tier L3 on a fact note
// overrides the default L2 tier.
func TestTierFrontmatter_OverrideL3(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var writtenContent []byte

	deps := cli.LearnDeps{
		Now:      func() time.Time { return time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) },
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		ListIDs:  func(string) ([]string, error) { return nil, nil },
		Lock:     func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(_ string, data []byte) error { writtenContent = data; return nil },
	}

	args := cli.LearnArgs{
		Type:      "fact",
		Slug:      "tier-override",
		Vault:     "/v",
		Position:  "top",
		Source:    "src",
		Situation: "tier override check",
		Subject:   "subj",
		Predicate: "pred",
		Object:    "obj",
		Tier:      "L3",
	}

	var stdout strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(writtenContent)).To(ContainSubstring("tier: L3"))
}

func TestValidateIssueID_AcceptsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateIssueID("")).To(Succeed())
}

func TestValidateIssueID_AcceptsNonWhitespace(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateIssueID("636")).To(Succeed())
	g.Expect(cli.ExportValidateIssueID("#636")).To(Succeed())
	g.Expect(cli.ExportValidateIssueID("PROJ-1234")).To(Succeed())
	g.Expect(cli.ExportValidateIssueID("gh-636")).To(Succeed())
}

func TestValidateIssueID_RejectsWhitespace(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateIssueID("636 ")).To(HaveOccurred())
	g.Expect(cli.ExportValidateIssueID("two words")).To(HaveOccurred())
	g.Expect(cli.ExportValidateIssueID("with\ttab")).To(HaveOccurred())
}

func TestValidateProjectSlug_AcceptsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateProjectSlug("")).To(Succeed())
}

func TestValidateProjectSlug_AcceptsKebabCase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateProjectSlug("engram")).To(Succeed())
	g.Expect(cli.ExportValidateProjectSlug("opencode-plugin")).To(Succeed())
	g.Expect(cli.ExportValidateProjectSlug("proj-123")).To(Succeed())
}

func TestValidateProjectSlug_RejectsBadShape(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportValidateProjectSlug("Engram")).To(HaveOccurred())
	g.Expect(cli.ExportValidateProjectSlug("with spaces")).To(HaveOccurred())
	g.Expect(cli.ExportValidateProjectSlug("punct!")).To(HaveOccurred())
}

// stubRangeReader returns deterministic content per path.
type stubRangeReader struct {
	chunks map[string]string
}

func (s stubRangeReader) ReadRange(path string, _, _ time.Time) (string, error) {
	if chunk, ok := s.chunks[path]; ok {
		return chunk, nil
	}

	return "", nil
}

// parseFrontmatter strips the "---" delimiters from a rendered frontmatter
// block and decodes the inner YAML mapping into key→string pairs. Tests use
// it to assert frontmatter values survive a YAML roundtrip regardless of the
// quoting style the encoder happens to choose.
func parseFrontmatter(t *testing.T, rendered string) map[string]string {
	t.Helper()

	g := NewWithT(t)

	const delim = "---\n"

	g.Expect(strings.HasPrefix(rendered, delim)).To(BeTrue(), "missing opening ---")

	body := strings.TrimPrefix(rendered, delim)
	end := strings.Index(body, "\n"+delim)
	g.Expect(end).To(BeNumerically(">=", 0), "missing closing ---")

	parsed := map[string]string{}
	g.Expect(yaml.Unmarshal([]byte(body[:end+1]), &parsed)).To(Succeed())

	return parsed
}

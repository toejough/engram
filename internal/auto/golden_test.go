package auto_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test793d22b3_FailuresAreClassifiedByWhatTheyLeft(t *testing.T) {
	t.Parallel()
	each(t, "failure_kind", func(g *WithT, input map[string]any, want golden) {
		g.Expect(auto.FailureKind(field[string](input, "text"))).To(Equal(text(want)))
	})
	each(t, "warning", func(g *WithT, input map[string]any, want golden) {
		g.Expect(auto.Warning(field[string](input, "step"), field[string](input, "kind"))).To(Equal(text(want)))
	})
}

func Test793d23b1_ATaskNotificationIsLookedUpByItsSummary(t *testing.T) {
	t.Parallel()
	each(t, "moment_text", func(g *WithT, input map[string]any, want golden) {
		g.Expect(auto.MomentText(field[string](input, "text"))).To(Equal(text(want)))
	})
}

func Test793d23b2_TheRecentTextsAreRedactedAndCut(t *testing.T) {
	t.Parallel()
	each(t, "recent_texts", func(g *WithT, input map[string]any, want golden) {
		got := auto.RecentTexts(convert[[]auto.Message](input["messages"]), field[string](input, "answer"))
		g.Expect(got).To(Equal(text(want)))
	})
}

func Test793d24b1_QueryArgumentsAreFlagValueForms(t *testing.T) {
	t.Parallel()
	each(t, "query_argv", func(g *WithT, input map[string]any, want golden) {
		got := auto.QueryArgv(field[string](input, "kind"), field[string](input, "text"), convert[[]string](input["moments"]))
		g.Expect(asJSON(got)).To(MatchJSON(want.Out))
	})
}

func Test793d24b1_QueryOutputIsReadAsTheOriginalDid(t *testing.T) {
	t.Parallel()
	each(t, "parse_query", func(g *WithT, input map[string]any, want golden) {
		items, err := auto.ParseQuery(field[string](input, "output"))
		if want.Err != "" {
			g.Expect(err).To(HaveOccurred())

			return
		}

		g.Expect(err).NotTo(HaveOccurred())

		var expected struct {
			Items      []map[string]any `json:"items"`
			Candidates []map[string]any `json:"candidates"`
		}

		_ = json.Unmarshal(want.Out, &expected)

		g.Expect(items).To(HaveLen(len(expected.Items)))

		for index, found := range auto.Items(items) {
			for _, key := range []string{"path", "kind", "situation", "body", "score"} {
				g.Expect(found[key]).To(Equal(expected.Items[index][key]), key)
			}
		}

		paths := make([]string, 0, len(expected.Candidates))
		wanted := make([]string, 0, len(expected.Candidates))

		for _, note := range auto.CandidatesOf(items) {
			paths = append(paths, note.Path)
		}

		for _, candidate := range expected.Candidates {
			wanted = append(wanted, field[string](candidate, "path"))
		}

		g.Expect(paths).To(Equal(wanted))
	})
}

func Test793d24b2_TheContextSinceTheLastCompactionIsWhatTheAgentHolds(t *testing.T) {
	t.Parallel()
	each(t, "since_compaction", func(g *WithT, input map[string]any, want golden) {
		g.Expect(auto.SinceCompaction(field[string](input, "text"))).To(Equal(text(want)))
	})
}

func Test793d25b1_RequestsAreTheOnesTheGreenPhaseMeasured(t *testing.T) {
	t.Parallel()
	each(t, "requests", func(g *WithT, input map[string]any, want golden) {
		var (
			request auto.Request
			err     error
		)

		switch {
		case input["recent"] != nil:
			request, err = auto.ClassifyRequest(field[string](input, "recent"))
		case input["window"] != nil:
			request, err = auto.CaptureRequest(field[string](input, "window"), convert[[]auto.Note](input["near"]))
		case input["offer"] != nil:
			offer := field[map[string]any](input, "offer")
			request, err = auto.CurationRequest(
				auto.Note{Situation: field[string](offer, "situation"), Body: field[string](offer, "body")},
				convert[[]auto.Note](input["near"]),
			)
		default:
			request, err = auto.RelevanceRequest(
				field[string](input, "moment"), field[string](input, "text"), convert[[]auto.Note](input["candidates"]),
			)
		}

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(asJSON(request)).To(MatchJSON(want.Out))
	})
}

func Test793d25b3_CurationIsReadStrictly(t *testing.T) {
	t.Parallel()
	each(t, "parse_curation", func(g *WithT, input map[string]any, want golden) {
		got, err := auto.ParseCuration(field[string](input, "reply"), convert[[]auto.Note](input["near"]))
		out := map[string]any{"outcome": got.Outcome, "target": nil, "fields": nil}

		if got.Target != "" {
			out["target"] = got.Target
		}

		if got.Fields != nil {
			fields := map[string]string{}
			for _, pair := range got.Fields {
				fields[pair[0]] = pair[1]
			}

			out["fields"] = fields
		}

		answers(g, want, out, err)
	})
}

func Test793d25b3_LessonsAreReadStrictly(t *testing.T) {
	t.Parallel()
	each(t, "parse_lessons", func(g *WithT, input map[string]any, want golden) {
		got, err := auto.ParseLessons(field[string](input, "reply"), convert[[]auto.Note](input["near"]))
		out := make([]map[string]any, 0, len(got))

		for _, lesson := range got {
			values := map[string]string{}
			for _, pair := range lesson.Values {
				values[pair[0]] = pair[1]
			}

			var covered any
			if lesson.Covered != "" {
				covered = lesson.Covered
			}

			out = append(out, map[string]any{
				"kind": lesson.Kind, "slug": lesson.Slug, "situation": lesson.Situation, "covered": covered, "values": values,
			})
		}

		answers(g, want, out, err)
	})
}

func Test793d25b3_MomentsAreReadStrictly(t *testing.T) {
	t.Parallel()
	each(t, "parse_moments", func(g *WithT, input map[string]any, want golden) {
		got, err := auto.ParseMoments(field[string](input, "reply"))
		out := make([]map[string]string, 0, len(got))

		for _, cue := range got {
			out = append(out, map[string]string{"moment": cue.Moment, "subject": cue.Subject})
		}

		answers(g, want, out, err)
	})
}

func Test793d25b3_SelectionIsReadStrictly(t *testing.T) {
	t.Parallel()
	each(t, "parse_selection", func(g *WithT, input map[string]any, want golden) {
		got, err := auto.ParseSelection(field[string](input, "reply"), int(field[float64](input, "count")))
		answers(g, want, got, err)
	})
}

func Test793d26b1_RedactMatchesTheOriginalOnTheCorpus(t *testing.T) {
	t.Parallel()

	for index, one := range load(t, "redact") {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			t.Parallel()

			NewWithT(t).Expect(auto.Redact(convert[string](one.In))).To(Equal(text(one)))
		})
	}
}

func Test793d27b1_ASubagentTranscriptIsCompleteWhenItHoldsTheLastMessage(t *testing.T) {
	t.Parallel()
	each(t, "transcript_holds", func(g *WithT, input map[string]any, want golden) {
		holds := auto.TranscriptHolds(field[string](input, "text"), field[string](input, "last"))
		g.Expect(asJSON(holds)).To(MatchJSON(want.Out))
	})
	each(t, "window_from_transcript", func(g *WithT, input map[string]any, want golden) {
		g.Expect(auto.WindowFromFile(field[string](input, "text"), field[string](input, "last"))).To(Equal(text(want)))
	})
}

func Test793d27b1_AWindowIsTheTurn(t *testing.T) {
	t.Parallel()
	each(t, "window_from_messages", func(g *WithT, input map[string]any, want golden) {
		got := auto.WindowFromMessages(convert[[]auto.Message](input["messages"]), field[string](input, "answer"))
		g.Expect(got).To(Equal(text(want)))
	})
}

func Test793d27b2_SourcesNameTheTurnOrThePositionAndTheAutomaticWrite(t *testing.T) {
	t.Parallel()
	each(t, "sources", func(g *WithT, input map[string]any, want golden) {
		if input["session"] != nil {
			turn := int(field[float64](input, "turn"))
			g.Expect(auto.SourceFor(field[string](input, "session"), turn, field[string](input, "agent"))).To(Equal(text(want)))

			return
		}

		g.Expect(auto.BacklogSourceFor(field[string](input, "path"), field[string](input, "position"))).To(Equal(text(want)))
	})
}

func Test793d27b2_TheNearestNotesAreLookedUpWithWhatTheUserSaidFirst(t *testing.T) {
	t.Parallel()
	each(t, "query_text", func(g *WithT, input map[string]any, want golden) {
		g.Expect(auto.QueryText(field[string](input, "window"))).To(Equal(text(want)))
	})
}

func Test793d28b1_TheSourcesToCaptureAreTheQuietOnesAfterTheCutoff(t *testing.T) {
	t.Parallel()
	each(t, "pending", func(g *WithT, input map[string]any, want golden) {
		listed := convert[[]map[string]any](input["sources"])
		sources := make([]auto.Source, 0, len(listed))

		for _, one := range listed {
			sources = append(sources, auto.Source{
				Path: field[string](one, "path"), MtimeMs: field[float64](one, "mtimeMs"), Hash: field[string](one, "hash"),
			})
		}

		slices.SortStableFunc(sources, func(left, right auto.Source) int {
			switch {
			case left.MtimeMs > right.MtimeMs:
				return -1
			case left.MtimeMs < right.MtimeMs:
				return 1
			}

			return 0
		})

		marks, err := auto.ParseMarks(string(asJSON(input["marks"])))
		g.Expect(err).NotTo(HaveOccurred())

		var expected []map[string]any

		_ = json.Unmarshal(want.Out, &expected)

		paths := make([]string, 0, len(expected))
		for _, source := range auto.Pending(sources, marks, field[float64](input, "now"), field[string](input, "live")) {
			paths = append(paths, source.Path)
		}

		wanted := make([]string, 0, len(expected))
		for _, source := range expected {
			wanted = append(wanted, field[string](source, "path"))
		}

		g.Expect(paths).To(Equal(wanted))
	})
}

func Test793d28b2_AConversationIsWindowsThatOverlapByAMessage(t *testing.T) {
	t.Parallel()
	each(t, "windows_from_messages", func(g *WithT, input map[string]any, want golden) {
		got := auto.WindowsFromMessage(convert[[]auto.Message](input["messages"]))
		if got == nil {
			got = []string{}
		}

		g.Expect(asJSON(got)).To(MatchJSON(want.Out))
	})
}

func Test793d28b2_ATranscriptIsReadByTheRolesOfItsRows(t *testing.T) {
	t.Parallel()
	each(t, "messages_from_transcript", func(g *WithT, input map[string]any, want golden) {
		got := auto.MessagesFromFile(field[string](input, "text"))
		out := make([]map[string]any, 0, len(got))

		for _, message := range got {
			uses := make([]map[string]string, 0, len(message.ToolUses))
			for _, use := range message.ToolUses {
				uses = append(uses, map[string]string{"tool": use.Tool})
			}

			out = append(out, map[string]any{"role": message.Role, "text": message.Text, "toolUses": uses})
		}

		g.Expect(asJSON(out)).To(MatchJSON(want.Out))
	})
}

func Test793d28b3_TheManifestListsTheSessionTranscripts(t *testing.T) {
	t.Parallel()
	each(t, "parse_manifest", func(g *WithT, input map[string]any, want golden) {
		got, err := auto.ParseManifest(field[string](input, "text"))
		if want.Err != "" {
			g.Expect(err).To(HaveOccurred())

			return
		}

		g.Expect(err).NotTo(HaveOccurred())

		paths := make([]string, 0, len(got))
		for _, source := range got {
			paths = append(paths, source.Path)
		}

		var expected []map[string]any

		_ = json.Unmarshal(want.Out, &expected)

		wanted := make([]string, 0, len(expected))
		for _, source := range expected {
			wanted = append(wanted, field[string](source, "path"))
		}

		slices.Sort(paths)
		slices.Sort(wanted)
		g.Expect(paths).To(Equal(wanted))
	})
}

func Test793d28b3_TheMarksAreReadAsTheOriginalDid(t *testing.T) {
	t.Parallel()
	each(t, "parse_marks", func(g *WithT, input map[string]any, want golden) {
		got, err := auto.ParseMarks(field[string](input, "text"))
		answers(g, want, got, err)
	})
}

func Test793d29b1_AnOfferIsReadFromTheHeaderAndFrontMatterOfShow(t *testing.T) {
	t.Parallel()
	each(t, "parse_show", func(g *WithT, input map[string]any, want golden) {
		g.Expect(asJSON(auto.ShownMap(field[string](input, "output")))).To(MatchJSON(want.Out))
	})
}

func Test793d29b2_AnOutcomeSettlesWithTheCommandsOfCurate(t *testing.T) {
	t.Parallel()
	each(t, "settle_argvs", func(g *WithT, input map[string]any, want golden) {
		decision := field[map[string]any](input, "decision")
		out := auto.Decision{Outcome: field[string](decision, "outcome"), Target: field[string](decision, "target")}
		fields := field[map[string]any](decision, "fields")

		for _, key := range []string{"subject", "predicate", "object", "behavior", "impact", "action"} {
			if value, found := fields[key]; found {
				out.Fields = append(out.Fields, [2]string{key, convert[string](value)})
			}
		}

		argvs := auto.SettleArgvs(out, field[string](input, "offer"), field[string](input, "hash"))
		g.Expect(asJSON(argvs)).To(MatchJSON(want.Out))
	})
}

func Test793d30b1_TheLineIsBuiltFromConstantsAndCounts(t *testing.T) {
	t.Parallel()
	each(t, "status_line", func(g *WithT, input map[string]any, want golden) {
		job := map[string]any{"kind": "summary", "tally": []any{input["counts"]}, "warnings": input["warnings"]}
		output, err := auto.Run(auto.Deps{}, asJSON(job))
		g.Expect(err).NotTo(HaveOccurred())

		var response auto.Response

		_ = json.Unmarshal(output, &response)

		var expected *string

		_ = json.Unmarshal(want.Out, &expected)

		if expected == nil {
			g.Expect(response.Done.Line).To(BeEmpty())

			return
		}

		g.Expect(response.Done.Line).To(Equal(*expected))
	})
}

// The golden files are what the JavaScript the logic replaced answered for a corpus of inputs (793e8m): the Go port
// must answer the same, byte for byte, for every input of every file.
type golden struct {
	In  json.RawMessage `json:"in"`
	Out json.RawMessage `json:"out"`
	Err string          `json:"err"`
}

// answers checks a function that returns a value or an error against the golden answer.
func answers(g *WithT, want golden, got any, err error) {
	if want.Err != "" {
		g.Expect(err).To(HaveOccurred())

		return
	}

	g.Expect(err).NotTo(HaveOccurred())

	if err == nil {
		g.Expect(asJSON(got)).To(MatchJSON(want.Out))
	}
}

func asJSON(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return encoded
}

// convert reads a value of the input as a typed value.
func convert[T any](value any) T {
	var out T

	_ = json.Unmarshal(asJSON(value), &out)

	return out
}

// each runs one check per case of a golden file, with the input decoded into a map.
func each(t *testing.T, name string, check func(g *WithT, input map[string]any, want golden)) {
	t.Helper()

	for index, one := range load(t, name) {
		t.Run(name+"/"+strconv.Itoa(index), func(t *testing.T) {
			t.Parallel()

			var input map[string]any

			_ = json.Unmarshal(one.In, &input)
			check(NewWithT(t), input, one)
		})
	}
}

// field is a value of the input of a case, or the zero value when it is not of that type.
func field[T any](input map[string]any, key string) T {
	value, _ := input[key].(T)

	return value
}

func load(t *testing.T, name string) []golden {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "golden", name+".json"))
	if err != nil {
		t.Fatalf("reading the golden file %s: %v", name, err)
	}

	var cases []golden

	err = json.Unmarshal(data, &cases)
	if err != nil {
		t.Fatalf("decoding the golden file %s: %v", name, err)
	}

	return cases
}

// text is the string a golden answer holds.
func text(want golden) string { return convert[string](want.Out) }

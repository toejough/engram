package auto

// Exported constants.
const (
	DeadlineMomentQuery = deadlineMomentQuery
	DeadlineNearest     = deadlineNearest
)

// Exported variables.
var (
	BacklogSourceFor   = backlogSourceFor
	CandidatesOf       = candidates
	DecodeComponent    = decode
	FailureKind        = failureKind
	MessagesFromFile   = messagesFromTranscript
	MomentText         = momentText
	ParseShow          = parseShow
	Pending            = pending
	QueryRequestOf     = queryRequest
	QueryText          = queryText
	RecentTexts        = recentTexts
	Redact             = redact
	SettleRequests     = settleRequests
	SinceCompaction    = sinceCompaction
	SourceFor          = sourceFor
	TranscriptHolds    = transcriptHolds
	Warning            = warning
	WindowFromFile     = windowFromTranscript
	WindowFromMessages = windowFromMessages
	WindowsFromMessage = windowsFromMessages
)

// Request builds a request of a kind from the tier table given.
func (t Tiers) Request(kind, prompt string) (Request, error) {
	return Try(func() Request { return t.request(kind, prompt) })
}

// CaptureRequest builds the capture request with the default tiers.
func CaptureRequest(window string, near []Note) (Request, error) {
	return DefaultTiers().CaptureWith(window, near)
}

// ClassifyRequest builds the classification request with the default tiers.
func ClassifyRequest(recent string) (Request, error) { return DefaultTiers().ClassifyWith(recent) }

// CurationRequest builds the curation request with the default tiers.
func CurationRequest(offer Note, near []Note) (Request, error) {
	return DefaultTiers().CurationWith(offer, near)
}

// Items are the items of a query's output as JSON, for the golden comparison.
func Items(items []Note) []map[string]any {
	out := make([]map[string]any, 0, len(items))

	for _, found := range items {
		out = append(out, map[string]any{
			"path": found.Path, "kind": found.Kind, "score": found.Score, "counted": found.counted,
			"situation": found.Situation, "body": found.Body,
		})
	}

	return out
}

// NoteAsRead is a note with the text it was read as.
func NoteAsRead(path, raw string) Note { return Note{Path: path, raw: raw} }

// The parsers, which fail by panic, as functions that return the error.
func ParseCuration(text string, near []Note) (Decision, error) {
	return Try(func() Decision { return parseCuration(text, near) })
}

func ParseLessons(text string, near []Note) ([]Lesson, error) {
	return Try(func() []Lesson { return parseLessons(text, near) })
}

func ParseManifest(text string) ([]Source, error) {
	return Try(func() []Source { return parseManifest(text) })
}

func ParseMarks(text string) (Marks, error) { return Try(func() Marks { return parseMarks(text) }) }

func ParseMoments(text string) ([]Cue, error) { return Try(func() []Cue { return parseMoments(text) }) }

func ParseQuery(text string) ([]Note, error) { return Try(func() []Note { return parseQuery(text) }) }

func ParseSelection(text string, count int) ([]int, error) {
	return Try(func() []int { return parseSelection(text, count) })
}

// RelevanceRequest builds the relevance request with the default tiers.
func RelevanceRequest(moment, text string, candidates []Note) (Request, error) {
	return DefaultTiers().RelevanceWith(moment, text, candidates)
}

// ShownMap is a parsed `engram show` as the golden comparison reads it.
func ShownMap(output string) map[string]any {
	offer := parseShow(output)
	out := map[string]any{
		"hash": nil, "kind": offer.Kind, "situation": offer.Situation, "pending": offer.pending, "skill": offer.skill,
		"forNote": nil, "body": offer.Body,
	}

	if offer.hash != "" {
		out["hash"] = offer.hash
	}

	if offer.forNote != "" {
		out["forNote"] = offer.forNote
	}

	return out
}

// Try runs a function that fails by panic, and returns what it returned or the error it failed with.
func Try[T any](run func() T) (value T, err error) {
	defer func() {
		if fault := recover(); fault != nil {
			var failed bool

			// a panic that is not an error is a bug, and is not turned into a success with no error
			if err, failed = fault.(error); !failed {
				panic(fault)
			}
		}
	}()

	return run(), nil
}

package cycle

// LearnExtractionPrompt returns the LLM Call A prompt for the given transcript.
// The model is asked to identify learnings worth preserving and emit them as a JSON array.
func LearnExtractionPrompt(transcript string) string {
	return learnExtractionHeader + transcript
}

// QueryProposalPrompt returns the LLM Call B prompt for the given transcript.
// The model is asked to propose 1-5 recall queries, or "NO QUERIES" if none are warranted.
func QueryProposalPrompt(transcript string) string {
	return queryProposalHeader + transcript
}

// unexported constants.
const (
	//nolint:lll
	learnExtractionHeader = `You are reviewing a project session transcript to identify learnings worth preserving.

Examine the transcript and propose any new learnings: corrections you observe, completed work that taught a lesson, decisions made, or facts established.

Output a JSON array of objects, each with:
- "type": "feedback" or "fact"
- "situation": short context phrase identifying when this applies
- For feedback: "behavior", "impact", "action"
- For fact: "subject", "predicate", "object"

Return [] if there is nothing learnable.

Transcript:
`

	//nolint:lll
	queryProposalHeader = `You are reviewing a project session transcript to decide if memories should be recalled.

If the project is starting new research, taking new action, shifting approach, or otherwise embarking on something where prior memories could help, propose 1-5 targeted recall queries. Each query is 5-15 words capturing a specific facet to recall about.

Output one query per line, no numbering, no commentary.

If nothing in the transcript warrants recall, output exactly:
NO QUERIES

Transcript:
`
)

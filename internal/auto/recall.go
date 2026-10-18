package auto

import "encoding/json"

// RecallInput is the reply of the phrases judgement and its carry, as the memory mod hands them over (855d4b).
type RecallInput struct {
	Session string `json:"session"`
	Cue     string `json:"cue"`
	Reply   string `json:"reply"`
	Carry   Carry  `json:"carry"`
}

// PhrasesOf returns the phrases of the reply of the phrases judgement that the lookup is run for: the first 3 at
// glance and the first 10 at deep (855d4b).
func PhrasesOf(reply, depth string) []string {
	var parsed struct {
		Phrases []string `json:"phrases"`
	}

	_ = json.Unmarshal([]byte(unfenced(reply)), &parsed)

	return parsed.Phrases[:min(len(parsed.Phrases), phraseLimit(depth))]
}

// phraseLimit is how many phrases the lookup is run for at a depth.
func phraseLimit(depth string) int {
	if depth == DepthGlance {
		return glancePhrases
	}

	return deepPhrases
}

// unexported constants.
const (
	deepPhrases   = 10
	glancePhrases = 3
)

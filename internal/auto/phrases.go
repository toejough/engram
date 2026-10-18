package auto

// PhrasesInput is the reply of the moment judgement and its carry, as the memory mod hands them over (855d4v).
type PhrasesInput struct {
	Session string `json:"session"`
	Cue     string `json:"cue"`
	Reply   string `json:"reply"`
	Carry   Carry  `json:"carry"`
}

// Phrases returns the request that phrases the lookup for the depth the moment judgement gave.
func Phrases(input PhrasesInput) (Step, bool, error) {
	return Step{}, false, nil
}

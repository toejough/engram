package auto

// ModelRequest is the request of a call of the model that a step prints: the fields the host's model call takes
// (855d1b). The effort is the cheap tier's only, and the time limit a capture's.
type ModelRequest struct {
	Model     string `json:"model"`
	Effort    string `json:"effort,omitempty"`
	System    string `json:"system"`
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"maxTokens"`
	TimeoutMs int    `json:"timeoutMs,omitempty"`
}

// Carry is what a step hands the next one with a request, and the mod reads no part of (855d1b).
type Carry struct {
	Cue    string `json:"cue"`
	Window string `json:"window"`
}

// Step is what a step prints for a judgement: the request, and the carry for the next step.
type Step struct {
	Request ModelRequest `json:"request"`
	Carry   Carry        `json:"carry"`
}

// MomentInput is a moment, as the memory mod hands it over (855d4a).
type MomentInput struct {
	Session string `json:"session"`
	Cue     string `json:"cue"`
	Origin  string `json:"origin"`
	Text    string `json:"text"`
}

// Moment returns the request that judges whether a moment is one for recall.
func Moment(record []byte, input MomentInput) (Step, bool, error) {
	return Step{}, false, nil
}

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

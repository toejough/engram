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

// unexported constants.
const (
	// cheapTier and midTier are the aliases of the models the steps are asked of: the cheap tier judges a moment and a
	// turn, and the mid tier phrases the lookup and judges the candidates.
	cheapTier = "haiku"
	midTier   = "sonnet"
)

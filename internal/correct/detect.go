package correct

import (
	"context"
	"strings"

	"engram/internal/anthropic"
)

// CallerFunc is the DI interface for calling an LLM in the correct package.
type CallerFunc = anthropic.CallerFunc

// DetectFastPath returns true if message contains any keyword (case-insensitive).
func DetectFastPath(message string, keywords []string) bool {
	loweredMessage := strings.ToLower(message)

	for _, keyword := range keywords {
		if strings.Contains(loweredMessage, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// DetectHaiku calls Haiku to classify whether a message is a correction.
// It returns true when the trimmed response contains correctionResponse.
func DetectHaiku(ctx context.Context, caller CallerFunc, message, systemPrompt string) (bool, error) {
	response, err := caller(ctx, anthropic.HaikuModel, systemPrompt, message)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(response) == correctionResponse, nil
}

// unexported constants.
const (
	// correctionResponse is the token returned by Haiku when a message is a correction.
	correctionResponse = "CORRECTION"
)

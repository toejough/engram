package cli

import (
	"context"
	"io"
)

// AutoPhrasesDeps holds injected dependencies for RunAutoPhrases.
type AutoPhrasesDeps struct {
	// Stdin carries the reply of the moment judgement and its carry: one JSON object.
	Stdin io.Reader
}

// RunAutoPhrases reads the reply of the moment judgement from standard input and prints the request for the model that
// phrases the lookup.
func RunAutoPhrases(_ context.Context, _ AutoPhrasesDeps, _ io.Writer) error {
	return nil
}

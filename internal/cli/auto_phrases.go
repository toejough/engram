package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/toejough/engram/internal/auto"
)

// AutoPhrasesArgs holds parsed flags for `engram auto phrases`: none.
type AutoPhrasesArgs struct{}

// AutoPhrasesDeps holds injected dependencies for RunAutoPhrases.
type AutoPhrasesDeps struct {
	// Stdin carries the reply of the moment judgement and its carry: one JSON object.
	Stdin io.Reader
}

// RunAutoPhrases reads the reply of the moment judgement from standard input and prints the request for the model that
// phrases the lookup, or nothing when the depth is none (855d4v).
func RunAutoPhrases(_ context.Context, deps AutoPhrasesDeps, stdout io.Writer) error {
	var input auto.PhrasesInput

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&input)
	if decodeErr != nil {
		return fmt.Errorf("auto phrases: reading the reply: %w", decodeErr)
	}

	step, found, phrasesErr := auto.Phrases(input)
	if phrasesErr != nil {
		return fmt.Errorf("auto phrases: %w", phrasesErr)
	}

	if !found {
		return nil
	}

	return printStep(stdout, "auto phrases", step)
}

// newAutoPhrasesDeps wires RunAutoPhrases from the injected CLI capabilities - pure composition (#700).
func newAutoPhrasesDeps(d Deps) AutoPhrasesDeps {
	return AutoPhrasesDeps{Stdin: d.Stdin}
}

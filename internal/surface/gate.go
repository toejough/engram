package surface

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"engram/internal/anthropic"
	"engram/internal/memory"
)

// HaikuCallerFunc calls the Haiku API with model, system, and user prompts.
type HaikuCallerFunc func(ctx context.Context, model, system, user string) (string, error)

// GateMemories uses Haiku to filter candidates by situational relevance.
// Returns an error if the gate call fails — callers must surface the failure.
func GateMemories(
	ctx context.Context,
	candidates []*memory.Stored,
	userMessage string,
	caller HaikuCallerFunc,
	systemPrompt string,
) ([]*memory.Stored, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}

	userPrompt := buildGateUserPrompt(candidates, userMessage)

	response, callErr := caller(ctx, anthropic.HaikuModel, systemPrompt, userPrompt)
	if callErr != nil {
		return nil, fmt.Errorf("haiku gate: %w", callErr)
	}

	slugs, parseErr := parseGateResponse(response)
	if parseErr != nil {
		return nil, fmt.Errorf("haiku gate: %w", parseErr)
	}

	return filterBySlug(candidates, slugs), nil
}

// WithHaikuGate sets the Haiku gate caller on the Surfacer.
func WithHaikuGate(caller HaikuCallerFunc) SurfacerOption {
	return func(s *Surfacer) { s.haikuGate = caller }
}

// buildGateUserPrompt formats the user prompt for the Haiku gate call.
func buildGateUserPrompt(candidates []*memory.Stored, userMessage string) string {
	var buf strings.Builder

	fmt.Fprintf(&buf, "User context:\n%s\n\nCandidate memories:\n", userMessage)

	for _, candidate := range candidates {
		slug := memory.NameFromPath(candidate.FilePath)
		fmt.Fprintf(&buf, "- slug: %s\n  situation: %s\n  behavior: %s\n  impact: %s\n  action: %s\n",
			slug, candidate.Situation, candidate.Behavior, candidate.Impact, candidate.Action)
	}

	return buf.String()
}

// filterBySlug keeps candidates whose filename slug (sans .toml) is in the slug set.
func filterBySlug(candidates []*memory.Stored, slugs []string) []*memory.Stored {
	slugSet := make(map[string]bool, len(slugs))
	for _, slug := range slugs {
		slugSet[slug] = true
	}

	result := make([]*memory.Stored, 0, len(candidates))

	for _, candidate := range candidates {
		slug := memory.NameFromPath(candidate.FilePath)
		if slugSet[slug] {
			result = append(result, candidate)
		}
	}

	return result
}

// parseGateResponse unmarshals the Haiku response JSON into a slug list,
// stripping markdown code fences if present.
func parseGateResponse(response string) ([]string, error) {
	cleaned := anthropic.StripCodeFences(response)

	var slugs []string

	err := json.Unmarshal([]byte(cleaned), &slugs)
	if err != nil {
		return nil, fmt.Errorf("parsing gate response: %w", err)
	}

	return slugs, nil
}

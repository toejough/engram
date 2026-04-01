package correct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"engram/internal/anthropic"
	"engram/internal/memory"
)

// Exported variables.
var (
	ErrEmptyResponse = errors.New("extraction: empty response array")
)

// CandidateResult describes how an existing memory relates to the new one.
type CandidateResult struct {
	Name        string `json:"name"`
	Disposition string `json:"disposition"`
	Reason      string `json:"reason"`
}

// ExtractionResult holds the SBIA fields and dedup decisions returned by Sonnet.
//
//nolint:tagliatelle // JSON API uses snake_case per spec.
type ExtractionResult struct {
	Situation     string            `json:"situation"`
	Behavior      string            `json:"behavior"`
	Impact        string            `json:"impact"`
	Action        string            `json:"action"`
	FilenameSlug  string            `json:"filename_slug"`
	ProjectScoped bool              `json:"project_scoped"`
	Candidates    []CandidateResult `json:"candidates"`
}

// Extract calls Sonnet to extract SBIA fields from a correction message.
// It builds a prompt with the message, conversation context, and any candidate
// memories, then parses the JSON response.
func Extract(
	ctx context.Context,
	caller CallerFunc,
	message, transcriptContext string,
	candidates []*memory.Stored,
	systemPrompt string,
) (*ExtractionResult, error) {
	userPrompt := buildExtractionPrompt(message, transcriptContext, candidates)

	response, err := caller(ctx, anthropic.SonnetModel, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	return parseExtractionResponse(response)
}

// buildExtractionPrompt assembles the user prompt for Sonnet extraction.
func buildExtractionPrompt(message, transcriptContext string, candidates []*memory.Stored) string {
	var builder strings.Builder

	builder.WriteString("## Correction message\n\n")
	builder.WriteString(message)
	builder.WriteString("\n\n")

	builder.WriteString("## Conversation context\n\n")
	builder.WriteString(transcriptContext)
	builder.WriteString("\n\n")

	if len(candidates) > 0 {
		builder.WriteString("## Existing similar memories (check for duplicates)\n\n")

		for _, candidate := range candidates {
			name := strings.TrimSuffix(filepath.Base(candidate.FilePath), ".toml")

			builder.WriteString("### ")
			builder.WriteString(name)
			builder.WriteString("\n")

			if candidate.Situation != "" {
				builder.WriteString("- Situation: ")
				builder.WriteString(candidate.Situation)
				builder.WriteString("\n")
			}

			if candidate.Behavior != "" {
				builder.WriteString("- Behavior: ")
				builder.WriteString(candidate.Behavior)
				builder.WriteString("\n")
			}

			if candidate.Impact != "" {
				builder.WriteString("- Impact: ")
				builder.WriteString(candidate.Impact)
				builder.WriteString("\n")
			}

			if candidate.Action != "" {
				builder.WriteString("- Action: ")
				builder.WriteString(candidate.Action)
				builder.WriteString("\n")
			}

			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// parseExtractionResponse parses the JSON response from Sonnet, stripping
// markdown code fences if present. Handles both single object and array
// responses (the prompt may produce either).
func parseExtractionResponse(response string) (*ExtractionResult, error) {
	cleaned := anthropic.StripCodeFences(response)

	// Try single object first.
	var result ExtractionResult

	err := json.Unmarshal([]byte(cleaned), &result)
	if err == nil {
		return &result, nil
	}

	// Try array — take the first element.
	var results []ExtractionResult

	arrErr := json.Unmarshal([]byte(cleaned), &results)
	if arrErr != nil {
		const maxPreview = 200

		preview := cleaned
		if len(preview) > maxPreview {
			preview = preview[:maxPreview] + "..."
		}

		return nil, fmt.Errorf("extraction: parsing response JSON: %w\nraw response:\n%s", err, preview)
	}

	if len(results) == 0 {
		return nil, ErrEmptyResponse
	}

	return &results[0], nil
}

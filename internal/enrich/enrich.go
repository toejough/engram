// Package enrich enriches pattern-matched messages into structured memories via the Anthropic API.
// If no API key is provided, or if the API returns unparseable JSON, a degraded memory is
// returned without making any network call.
package enrich

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"engram/internal/memory"
)

// HTTPDoer is the interface for making HTTP requests. Wire http.DefaultClient in production.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// LLMEnricher uses the Anthropic API to enrich memories into structured form.
type LLMEnricher struct {
	apiKey string
	client HTTPDoer
}

// New creates an LLMEnricher. Pass an empty apiKey to always use the degraded path.
// Pass http.DefaultClient as client in production.
func New(apiKey string, client HTTPDoer) *LLMEnricher {
	return &LLMEnricher{
		apiKey: apiKey,
		client: client,
	}
}

const (
	anthropicAPIURL          = "https://api.anthropic.com/v1/messages"
	anthropicModel           = "claude-haiku-4-5-20251001"
	anthropicVersion         = "2023-06-01"
	maxResponseTokens        = 1024
	maxTitleLength           = 60
	filenameSummaryWordCount = 5
)

// systemPrompt returns the system prompt instructing the LLM to extract structured memory fields.
func systemPrompt() string {
	return strings.TrimSpace(`
You are a memory extraction assistant. Given a user correction message and its category,
extract structured information and return ONLY a JSON object — no markdown, no explanation.

Return a JSON object with these exact fields:
{
  "title": "Short title (5-10 words) summarizing the memory",
  "content": "The full original message verbatim",
  "observation_type": "The category label provided",
  "concepts": ["key", "concepts"],
  "keywords": ["searchable", "keywords"],
  "principle": "The positive rule or principle to follow",
  "anti_pattern": "The negative pattern or mistake to avoid",
  "rationale": "Why this principle matters",
  "filename_summary": "three to five words"
}`)
}

// anthropicRequest is the request body for the Anthropic messages API.
//
//nolint:tagliatelle // Anthropic API requires snake_case JSON field names.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

// anthropicMessage is a single message in the Anthropic messages API.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the response body from the Anthropic messages API.
type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
}

// anthropicContentBlock is a content block in an Anthropic API response.
type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// llmMemoryJSON is the JSON structure the LLM is instructed to return.
//
//nolint:tagliatelle // LLM prompt specifies snake_case JSON field names.
type llmMemoryJSON struct {
	Title           string   `json:"title"`
	Content         string   `json:"content"`
	ObservationType string   `json:"observation_type"`
	Concepts        []string `json:"concepts"`
	Keywords        []string `json:"keywords"`
	Principle       string   `json:"principle"`
	AntiPattern     string   `json:"anti_pattern"`
	Rationale       string   `json:"rationale"`
	FilenameSummary string   `json:"filename_summary"`
}

var errEmptyAPIResponse = errors.New("API response contained no content blocks")

// Enrich enriches a message into a structured memory.
// If apiKey is empty or the LLM response cannot be parsed, a degraded memory is returned.
// Degraded memories have no enrichment fields but never return an error.
func (e *LLMEnricher) Enrich(
	ctx context.Context,
	message string,
	match *memory.PatternMatch,
) (*memory.Enriched, error) {
	if e.apiKey == "" {
		return degradedMemory(message, match), nil
	}

	mem, err := e.callLLM(ctx, message, match)
	if err != nil {
		// Intentional: degrade gracefully on LLM failure (REQ-5, ARCH-3).
		return degradedMemory(message, match), nil //nolint:nilerr
	}

	return mem, nil
}

func (e *LLMEnricher) callLLM(
	ctx context.Context,
	message string,
	match *memory.PatternMatch,
) (*memory.Enriched, error) {
	resp, err := e.sendRequest(ctx, message, match)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	return parseLLMResponse(resp, match)
}

func (e *LLMEnricher) sendRequest(
	ctx context.Context,
	message string,
	match *memory.PatternMatch,
) (*http.Response, error) {
	reqBody := anthropicRequest{
		Model:     anthropicModel,
		MaxTokens: maxResponseTokens,
		System:    systemPrompt(),
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf("Category: %s\nMessage: %s", match.Label, message),
			},
		},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	req.Header.Set("X-Api-Key", e.apiKey)
	req.Header.Set("Anthropic-Version", anthropicVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic API: %w", err)
	}

	return resp, nil
}

func parseLLMResponse(resp *http.Response, match *memory.PatternMatch) (*memory.Enriched, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var apiResp anthropicResponse

	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return nil, fmt.Errorf("parsing API response JSON: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, errEmptyAPIResponse
	}

	var llmData llmMemoryJSON

	err = json.Unmarshal([]byte(apiResp.Content[0].Text), &llmData)
	if err != nil {
		return nil, fmt.Errorf("parsing LLM JSON output: %w", err)
	}

	now := time.Now()

	return &memory.Enriched{
		Title:           llmData.Title,
		Content:         llmData.Content,
		ObservationType: llmData.ObservationType,
		Concepts:        llmData.Concepts,
		Keywords:        llmData.Keywords,
		Principle:       llmData.Principle,
		AntiPattern:     llmData.AntiPattern,
		Rationale:       llmData.Rationale,
		FilenameSummary: llmData.FilenameSummary,
		Confidence:      match.Confidence,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// degradedMemory returns a minimal Enriched without making any API call.
func degradedMemory(message string, match *memory.PatternMatch) *memory.Enriched {
	now := time.Now()

	return &memory.Enriched{
		Title:           truncateAtWordBoundary(message, maxTitleLength),
		Content:         message,
		ObservationType: match.Label,
		FilenameSummary: firstNWords(message, filenameSummaryWordCount),
		Confidence:      match.Confidence,
		Degraded:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// truncateAtWordBoundary truncates text to at most maxLen characters, ending at a word boundary.
func truncateAtWordBoundary(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	truncated := text[:maxLen]

	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > 0 {
		return truncated[:lastSpace]
	}

	return truncated
}

// firstNWords returns the first n words of text joined by spaces.
func firstNWords(text string, count int) string {
	words := strings.Fields(text)
	if len(words) <= count {
		return strings.Join(words, " ")
	}

	return strings.Join(words[:count], " ")
}

// Package enrich enriches pattern-matched messages into structured memories via the Anthropic API.
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

// Exported variables.
var (
	// ErrNilResponse is returned when the HTTP client returns a nil response without error.
	ErrNilResponse = errors.New("calling Anthropic API: nil response")
	// ErrNoToken is returned when no API token is configured.
	ErrNoToken = errors.New("no API token configured")
)

// HTTPDoer is the interface for making HTTP requests. Wire http.DefaultClient in production.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// LLMEnricher uses the Anthropic API to enrich memories into structured form.
type LLMEnricher struct {
	token  string
	client HTTPDoer
}

// New creates an LLMEnricher. Pass http.DefaultClient as client in production.
func New(token string, client HTTPDoer) *LLMEnricher {
	return &LLMEnricher{
		token:  token,
		client: client,
	}
}

// Enrich enriches a message into a structured memory via the Anthropic API.
// Returns ErrNoAPIKey if no API key is configured.
func (e *LLMEnricher) Enrich(
	ctx context.Context,
	message string,
	match *memory.PatternMatch,
) (*memory.Enriched, error) {
	if e.token == "" {
		return nil, ErrNoToken
	}

	mem, err := e.callLLM(ctx, message, match)
	if err != nil {
		return nil, fmt.Errorf("enrichment: %w", err)
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

	if resp == nil {
		return nil, ErrNilResponse
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

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		anthropicAPIURL,
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Anthropic-Version", anthropicVersion)
	req.Header.Set("Anthropic-Beta", "oauth-2025-04-20")
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic API: %w", err)
	}

	return resp, nil
}

// unexported constants.
const (
	anthropicAPIURL   = "https://api.anthropic.com/v1/messages"
	anthropicModel    = "claude-haiku-4-5-20251001"
	anthropicVersion  = "2023-06-01"
	maxResponseTokens = 1024
)

// unexported variables.
var (
	errEmptyAPIResponse = errors.New("API response contained no content blocks")
)

// anthropicContentBlock is a content block in an Anthropic API response.
type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// anthropicMessage is a single message in the Anthropic messages API.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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

// anthropicResponse is the response body from the Anthropic messages API.
type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
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

	llmText := stripMarkdownFence(apiResp.Content[0].Text)

	err = json.Unmarshal([]byte(llmText), &llmData)
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

// stripMarkdownFence removes markdown code fences (```json ... ```) that LLMs
// sometimes wrap around JSON output despite being told not to.
func stripMarkdownFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return text
	}

	// Remove opening fence (```json or ```)
	firstNewline := strings.Index(trimmed, "\n")
	if firstNewline < 0 {
		return text
	}

	trimmed = trimmed[firstNewline+1:]

	// Remove closing fence
	if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
		trimmed = trimmed[:idx]
	}

	return strings.TrimSpace(trimmed)
}

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

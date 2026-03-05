// Package enforce implements LLM-based tool call judgment for PreToolUse enforcement (ARCH-11).
package enforce

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"engram/internal/memory"
)

// Exported variables.
var (
	ErrEmptyAPIResult = errors.New("empty API response")
	ErrNilResponse    = errors.New("calling Anthropic API: empty response")
	ErrNoToken        = errors.New("no API token configured")
)

// Enforcer judges whether tool calls violate memory anti-patterns via LLM.
type Enforcer struct {
	client HTTPDoer
}

// New creates an Enforcer. Pass http.DefaultClient in production.
func New(client HTTPDoer) *Enforcer {
	return &Enforcer{client: client}
}

// JudgeViolation calls the LLM to determine whether a tool call violates
// a memory's anti-pattern. Returns (false, error) on failure (graceful degradation).
func (e *Enforcer) JudgeViolation(
	ctx context.Context, toolName, toolInput string,
	mem *memory.Stored, token string,
) (bool, error) {
	if token == "" {
		return false, ErrNoToken
	}

	resp, err := e.callLLM(ctx, toolName, toolInput, mem, token)
	if err != nil {
		return false, fmt.Errorf("enforce: %w", err)
	}

	return resp.Violated, nil
}

func (e *Enforcer) callLLM(
	ctx context.Context, toolName, toolInput string,
	mem *memory.Stored, token string,
) (*judgmentResponse, error) {
	prompt := buildPrompt(toolName, toolInput, mem)

	reqBody := anthropicRequest{
		Model:     anthropicModel,
		MaxTokens: maxResponseTokens,
		System:    systemPrompt(),
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		anthropicAPIURL,
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Anthropic-Version", anthropicVersion)
	req.Header.Set("Anthropic-Beta", "oauth-2025-04-20")
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic API: %w", err)
	}

	if resp == nil {
		return nil, ErrNilResponse
	}

	defer func() { _ = resp.Body.Close() }()

	return parseResponse(resp)
}

// HTTPDoer is the interface for making HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// unexported constants.
const (
	anthropicAPIURL   = "https://api.anthropic.com/v1/messages"
	anthropicModel    = "claude-haiku-4-5-20251001"
	anthropicVersion  = "2023-06-01"
	maxResponseTokens = 256
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

type judgmentResponse struct {
	Violated bool `json:"violated"`
}

func buildPrompt(toolName, toolInput string, mem *memory.Stored) string {
	return fmt.Sprintf(
		"Tool call: %s with input %s\n\nMemory principle: %s\nMemory anti_pattern: %s\n\nIs the anti_pattern being violated?",
		toolName,
		toolInput,
		mem.Principle,
		mem.AntiPattern,
	)
}

func parseResponse(resp *http.Response) (*judgmentResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var apiResp anthropicResponse

	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return nil, fmt.Errorf("parsing API response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, ErrEmptyAPIResult
	}

	text := stripMarkdownFence(apiResp.Content[0].Text)

	var judgment judgmentResponse

	err = json.Unmarshal([]byte(text), &judgment)
	if err != nil {
		return nil, fmt.Errorf("parsing judgment JSON: %w", err)
	}

	return &judgment, nil
}

func stripMarkdownFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return text
	}

	firstNewline := strings.Index(trimmed, "\n")
	if firstNewline < 0 {
		return text
	}

	trimmed = trimmed[firstNewline+1:]

	if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
		trimmed = trimmed[:idx]
	}

	return strings.TrimSpace(trimmed)
}

func systemPrompt() string {
	return strings.TrimSpace(`
You are a tool call validator. Given a tool call and a memory's principle and anti_pattern,
determine whether the tool call violates the anti_pattern.

Return ONLY a JSON object: {"violated": true} or {"violated": false}
No explanation, no markdown fences.`)
}

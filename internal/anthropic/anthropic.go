// Package anthropic provides a shared client for the Anthropic Messages API.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Exported errors.
var (
	ErrNilResponse     = errors.New("anthropic: nil response")
	ErrNoContentBlocks = errors.New("anthropic: response contained no content blocks")
	ErrNoToken         = errors.New("anthropic: no API token configured")
)

// Model constants.
const (
	HaikuModel = "claude-haiku-4-5-20251001"
)

// API constants.
const (
	defaultAPIURL = "https://api.anthropic.com/v1/messages"
	apiVersion    = "2023-06-01"
	betaHeader    = "oauth-2025-04-20"
)

// HTTPDoer is the interface for making HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// contentBlock is a content block in an Anthropic API response.
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// message is a single message in the Anthropic messages API.
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// request is the request body for the Anthropic messages API.
//
//nolint:tagliatelle // Anthropic API requires snake_case JSON field names.
type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
}

// response is the response body from the Anthropic messages API.
type response struct {
	Content []contentBlock `json:"content"`
}

// Client calls the Anthropic Messages API.
type Client struct {
	token  string
	client HTTPDoer
	apiURL string
}

// NewClient creates a Client. Pass http.DefaultClient as doer in production.
func NewClient(token string, doer HTTPDoer) *Client {
	return &Client{
		token:  token,
		client: doer,
		apiURL: defaultAPIURL,
	}
}

// SetAPIURL overrides the API endpoint URL (for testing).
func (c *Client) SetAPIURL(url string) {
	c.apiURL = url
}

// CallerFunc is the function signature used by packages that receive an LLM
// caller via dependency injection (signal, maintain, instruct).
type CallerFunc func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)

// Caller returns a CallerFunc backed by this client with the given maxTokens.
func (c *Client) Caller(maxTokens int) CallerFunc {
	return func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		return c.Call(ctx, model, systemPrompt, userPrompt, maxTokens)
	}
}

// Call makes a single call to the Anthropic Messages API and returns the text response.
func (c *Client) Call(
	ctx context.Context,
	model, systemPrompt, userPrompt string,
	maxTokens int,
) (string, error) {
	if c.token == "" {
		return "", ErrNoToken
	}

	reqBody := request{
		Model:     model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  []message{{Role: "user", Content: userPrompt}},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("anthropic: marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.apiURL,
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		return "", fmt.Errorf("anthropic: creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Anthropic-Version", apiVersion)
	req.Header.Set("Anthropic-Beta", betaHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: calling API: %w", err)
	}

	if resp == nil {
		return "", ErrNilResponse
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("anthropic: reading response: %w", err)
	}

	var apiResp response

	if jsonErr := json.Unmarshal(body, &apiResp); jsonErr != nil {
		return "", fmt.Errorf("anthropic: parsing response: %w", jsonErr)
	}

	if len(apiResp.Content) == 0 {
		return "", ErrNoContentBlocks
	}

	return apiResp.Content[0].Text, nil
}

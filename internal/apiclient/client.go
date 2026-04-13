// Package apiclient provides a thin HTTP client for the engram API server.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// HTTPDoer abstracts http.Client for testing.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is a thin HTTP client for the engram API server.
type Client struct {
	baseURL string
	doer    HTTPDoer
}

// New creates a Client. Pass http.DefaultClient as doer in production.
func New(baseURL string, doer HTTPDoer) *Client {
	return &Client{baseURL: baseURL, doer: doer}
}

// PostMessageRequest is the request body for POST /message.
type PostMessageRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
}

// PostMessageResponse is the response from POST /message.
type PostMessageResponse struct {
	Cursor int    `json:"cursor"`
	Error  string `json:"error,omitempty"`
}

// PostMessage posts a message to the chat via the API server.
func (c *Client) PostMessage(
	ctx context.Context,
	req PostMessageRequest,
) (PostMessageResponse, error) {
	body, marshalErr := json.Marshal(req)
	if marshalErr != nil {
		return PostMessageResponse{}, fmt.Errorf("apiclient: marshaling request: %w", marshalErr)
	}

	httpReq, reqErr := http.NewRequestWithContext(
		ctx, http.MethodPost, c.baseURL+"/message", bytes.NewReader(body),
	)
	if reqErr != nil {
		return PostMessageResponse{}, fmt.Errorf("apiclient: building request: %w", reqErr)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	return c.doJSON(httpReq)
}

// doJSON executes a request and decodes the JSON response.
func (c *Client) doJSON(httpReq *http.Request) (PostMessageResponse, error) {
	httpResp, doErr := c.doer.Do(httpReq)
	if doErr != nil {
		return PostMessageResponse{}, fmt.Errorf("apiclient: %w", doErr)
	}
	defer httpResp.Body.Close()

	var resp PostMessageResponse

	decErr := json.NewDecoder(httpResp.Body).Decode(&resp)
	if decErr != nil {
		return PostMessageResponse{}, fmt.Errorf("apiclient: decoding response: %w", decErr)
	}

	if httpResp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf(
			"apiclient: server returned %d: %s", httpResp.StatusCode, resp.Error,
		)
	}

	return resp, nil
}

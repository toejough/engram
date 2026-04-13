// Package apiclient provides a thin HTTP client for the engram API server.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Exported variables.
var (
	// ErrNilResponse is returned when the HTTP client returns a nil response.
	ErrNilResponse = errors.New("apiclient: nil response from server")
	// ErrNonOK is returned when the server responds with a non-200 status code.
	ErrNonOK = errors.New("apiclient: server returned non-OK status")
)

// Client is a thin HTTP client for the engram API server.
type Client struct {
	baseURL string
	doer    HTTPDoer
}

// New creates a Client. Pass http.DefaultClient as doer in production.
func New(baseURL string, doer HTTPDoer) *Client {
	return &Client{baseURL: baseURL, doer: doer}
}

// PostMessage posts a message to the chat via the API server.
func (c *Client) PostMessage(
	ctx context.Context,
	req PostMessageRequest,
) (PostMessageResponse, error) {
	var resp PostMessageResponse

	err := c.doPost(ctx, "/message", req, &resp)
	if err != nil {
		return PostMessageResponse{}, err
	}

	return resp, nil
}

// WaitForResponse long-polls the server for a response message.
func (c *Client) WaitForResponse(
	ctx context.Context,
	req WaitRequest,
) (WaitResponse, error) {
	fullURL := fmt.Sprintf(
		"%s/wait-for-response?from=%s&to=%s&after-cursor=%d",
		c.baseURL, req.From, req.To, req.AfterCursor,
	)

	var resp WaitResponse

	err := c.doGet(ctx, fullURL, &resp)
	if err != nil {
		return WaitResponse{}, err
	}

	return resp, nil
}

// doAndDecode executes an HTTP request, decodes the JSON response into dest,
// and returns an error if the status code is not 200.
func (c *Client) doAndDecode(httpReq *http.Request, dest any) error {
	httpResp, doErr := c.doer.Do(httpReq)
	if doErr != nil {
		return fmt.Errorf("apiclient: %w", doErr)
	}

	if httpResp == nil {
		return ErrNilResponse
	}

	defer func() { _ = httpResp.Body.Close() }()

	decErr := json.NewDecoder(httpResp.Body).Decode(dest)
	if decErr != nil {
		return fmt.Errorf("apiclient: decoding response: %w", decErr)
	}

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d", ErrNonOK, httpResp.StatusCode)
	}

	return nil
}

// doGet builds a GET request, executes it, and decodes the JSON response
// into dest. The caller must supply the full URL including query params.
func (c *Client) doGet(ctx context.Context, fullURL string, dest any) error {
	httpReq, reqErr := http.NewRequestWithContext(
		ctx, http.MethodGet, fullURL, nil,
	)
	if reqErr != nil {
		return fmt.Errorf("apiclient: building request: %w", reqErr)
	}

	return c.doAndDecode(httpReq, dest)
}

// doPost builds a POST request with a JSON body, executes it, and decodes
// the JSON response into dest.
func (c *Client) doPost(
	ctx context.Context,
	path string,
	body any,
	dest any,
) error {
	encoded, marshalErr := json.Marshal(body)
	if marshalErr != nil {
		return fmt.Errorf("apiclient: marshaling request: %w", marshalErr)
	}

	httpReq, reqErr := http.NewRequestWithContext(
		ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(encoded),
	)
	if reqErr != nil {
		return fmt.Errorf("apiclient: building request: %w", reqErr)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	return c.doAndDecode(httpReq, dest)
}

// HTTPDoer abstracts http.Client for testing.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
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

// WaitRequest is the request for GET /wait-for-response.
type WaitRequest struct {
	From        string
	To          string
	AfterCursor int
}

// WaitResponse is the response from GET /wait-for-response.
type WaitResponse struct {
	Text   string `json:"text"`
	Cursor int    `json:"cursor"`
	From   string `json:"from"`
	To     string `json:"to"`
}

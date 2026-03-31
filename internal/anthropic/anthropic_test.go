package anthropic_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/anthropic"
)

func TestCall_APIErrorResponse(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Simulate a 401 error response from the Anthropic API.
	errorBody := `{"type":"error","error":{"type":"authentication_error","message":"invalid api key"}}`
	doer := &fakeDoer{
		response: &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(bytes.NewBufferString(errorBody)),
		},
	}

	client := anthropic.NewClient("bad-token", doer)
	_, err := client.Call(context.Background(), anthropic.HaikuModel, "sys", "usr", 1024)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(anthropic.ErrAPIError))
	g.Expect(err.Error()).To(ContainSubstring("401"))
	g.Expect(err.Error()).To(ContainSubstring("invalid api key"))
}

func TestCall_APIErrorResponse_NonJSONBody(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeDoer{
		response: &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(bytes.NewBufferString("upstream error")),
		},
	}

	client := anthropic.NewClient("token", doer)
	_, err := client.Call(context.Background(), anthropic.HaikuModel, "sys", "usr", 1024)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("502")))
	g.Expect(err).To(MatchError(ContainSubstring("upstream error")))
	g.Expect(err).To(MatchError(anthropic.ErrAPIError))
}

func TestCall_APIErrorResponse_RateLimit(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	errorBody := `{"type":"error","error":{"type":"rate_limit_error","message":"rate limited"}}`
	doer := &fakeDoer{
		response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Body:       io.NopCloser(bytes.NewBufferString(errorBody)),
		},
	}

	client := anthropic.NewClient("token", doer)
	_, err := client.Call(context.Background(), anthropic.HaikuModel, "sys", "usr", 1024)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(anthropic.ErrAPIError))
	g.Expect(err.Error()).To(ContainSubstring("429"))
	g.Expect(err.Error()).To(ContainSubstring("rate limited"))
}

func TestCall_EmptyContent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	emptyResp, err := json.Marshal(struct {
		Content []struct{} `json:"content"`
	}{Content: []struct{}{}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	doer := &fakeDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(emptyResp)),
		},
	}

	client := anthropic.NewClient("token", doer)
	_, callErr := client.Call(context.Background(), anthropic.HaikuModel, "sys", "usr", 1024)
	g.Expect(callErr).To(MatchError(anthropic.ErrNoContentBlocks))
}

func TestCall_NilResponse(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeDoer{response: nil}
	client := anthropic.NewClient("token", doer)
	_, err := client.Call(context.Background(), anthropic.HaikuModel, "sys", "usr", 1024)
	g.Expect(err).To(MatchError(anthropic.ErrNilResponse))
}

func TestCall_NoToken(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	client := anthropic.NewClient("", nil)
	_, err := client.Call(context.Background(), anthropic.HaikuModel, "sys", "usr", 1024)
	g.Expect(err).To(MatchError(anthropic.ErrNoToken))
}

func TestCall_ReturnsTextContent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	body := makeAPIResponse(t, g, "hello world")
	doer := &fakeDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
		},
	}

	client := anthropic.NewClient("test-token", doer)
	result, err := client.Call(context.Background(), anthropic.HaikuModel, "system", "user", 1024)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(Equal("hello world"))
}

func TestCall_SetsCorrectHeaders(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	body := makeAPIResponse(t, g, "ok")
	doer := &fakeDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
		},
	}

	client := anthropic.NewClient("my-token", doer)
	_, err := client.Call(context.Background(), anthropic.HaikuModel, "sys", "usr", 1024)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(doer.lastRequest).NotTo(BeNil())

	if doer.lastRequest == nil {
		return
	}

	g.Expect(doer.lastRequest.Header.Get("Authorization")).To(Equal("Bearer my-token"))
	g.Expect(doer.lastRequest.Header.Get("Anthropic-Version")).To(Equal("2023-06-01"))
	g.Expect(doer.lastRequest.Header.Get("Anthropic-Beta")).To(Equal("oauth-2025-04-20"))
	g.Expect(doer.lastRequest.Header.Get("Content-Type")).To(Equal("application/json"))
}

// fakeDoer is a test double for anthropic.HTTPDoer.
type fakeDoer struct {
	lastRequest *http.Request
	response    *http.Response
	err         error
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.lastRequest = req
	return f.response, f.err
}

func makeAPIResponse(t *testing.T, g Gomega, text string) []byte {
	t.Helper()

	resp := struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}{
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{{Type: "text", Text: text}},
	}

	data, err := json.Marshal(resp)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return nil
	}

	return data
}

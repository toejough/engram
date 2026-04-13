package apiclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"engram/internal/apiclient"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

func TestPostMessage_AlwaysReturnsCursorFromServer(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Cursor: cursor})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		resp, err := client.PostMessage(context.Background(), apiclient.PostMessageRequest{
			From: "a", To: "b", Text: "c",
		})
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(resp.Cursor).To(Equal(cursor))
	})
}

func TestPostMessage_AlwaysReturnsErrorForNon200(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		status := rapid.SampledFrom([]int{
			http.StatusBadRequest, http.StatusUnauthorized,
			http.StatusInternalServerError, http.StatusServiceUnavailable,
		}).Draw(rt, "status")
		errMsg := rapid.StringMatching(`[a-z ]{5,50}`).Draw(rt, "errMsg")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Error: errMsg})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		_, err := client.PostMessage(context.Background(), apiclient.PostMessageRequest{
			From: "a", To: "b", Text: "c",
		})
		g.Expect(err).To(HaveOccurred())
	})
}

func TestPostMessage_AlwaysSendsCorrectMethodPathAndBody(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{0,19}`).Draw(rt, "from")
		recipient := rapid.StringMatching(`[a-z][a-z0-9\-]{0,19}`).Draw(rt, "to")
		text := rapid.StringMatching(`.{1,200}`).Draw(rt, "text")

		var gotMethod, gotPath string

		var gotBody apiclient.PostMessageRequest

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			decErr := json.NewDecoder(r.Body).Decode(&gotBody)
			g.Expect(decErr).NotTo(HaveOccurred())

			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Cursor: 1})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		_, err := client.PostMessage(context.Background(), apiclient.PostMessageRequest{
			From: from, To: recipient, Text: text,
		})
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(gotMethod).To(Equal(http.MethodPost))
		g.Expect(gotPath).To(Equal("/message"))
		g.Expect(gotBody.From).To(Equal(from))
		g.Expect(gotBody.To).To(Equal(recipient))
		g.Expect(gotBody.Text).To(Equal(text))
	})
}

func TestPostMessage_ReturnsErrorForInvalidURL(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	client := apiclient.New("http://\x00invalid", http.DefaultClient)
	_, err := client.PostMessage(context.Background(), apiclient.PostMessageRequest{
		From: "a", To: "b", Text: "c",
	})
	g.Expect(err).To(HaveOccurred())
}

func TestWaitForResponse_AlwaysRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		},
	))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := apiclient.New(srv.URL, srv.Client())
	_, err := client.WaitForResponse(ctx, apiclient.WaitRequest{
		From: "a", To: "b", AfterCursor: 0,
	})
	g.Expect(err).To(HaveOccurred())
}

func TestWaitForResponse_AlwaysReturnsFaithfulText(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		text := rapid.StringMatching(`.{1,500}`).Draw(rt, "text")
		cursor := rapid.IntRange(1, 100000).Draw(rt, "cursor")

		srv := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				encErr := json.NewEncoder(w).Encode(apiclient.WaitResponse{
					Text: text, Cursor: cursor,
				})
				g.Expect(encErr).NotTo(HaveOccurred())
			},
		))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		resp, err := client.WaitForResponse(
			context.Background(),
			apiclient.WaitRequest{From: "a", To: "b", AfterCursor: 0},
		)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(resp.Text).To(Equal(text))
		g.Expect(resp.Cursor).To(Equal(cursor))
	})
}

func TestWaitForResponse_AlwaysSendsCorrectQueryParams(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{0,19}`).Draw(rt, "from")
		recipient := rapid.StringMatching(`[a-z][a-z0-9\-]{0,19}`).Draw(rt, "to")
		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		srv := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				g.Expect(r.Method).To(Equal(http.MethodGet))
				g.Expect(r.URL.Path).To(Equal("/wait-for-response"))
				g.Expect(r.URL.Query().Get("from")).To(Equal(from))
				g.Expect(r.URL.Query().Get("to")).To(Equal(recipient))
				g.Expect(r.URL.Query().Get("after-cursor")).To(
					Equal(strconv.Itoa(cursor)),
				)

				w.WriteHeader(http.StatusOK)
				encErr := json.NewEncoder(w).Encode(
					apiclient.WaitResponse{Cursor: cursor + 1},
				)
				g.Expect(encErr).NotTo(HaveOccurred())
			},
		))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		_, err := client.WaitForResponse(
			context.Background(),
			apiclient.WaitRequest{
				From: from, To: recipient, AfterCursor: cursor,
			},
		)
		g.Expect(err).NotTo(HaveOccurred())
	})
}

func TestWaitForResponse_ReturnsErrorForInvalidURL(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// A base URL with a control character makes NewRequestWithContext fail.
	client := apiclient.New("http://\x00invalid", http.DefaultClient)
	_, err := client.WaitForResponse(context.Background(), apiclient.WaitRequest{
		From: "a", To: "b", AfterCursor: 0,
	})
	g.Expect(err).To(HaveOccurred())
}

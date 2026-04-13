package apiclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/apiclient"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

func TestPostMessage_AlwaysSendsCorrectMethodPathAndBody(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{0,19}`).Draw(rt, "from")
		to := rapid.StringMatching(`[a-z][a-z0-9\-]{0,19}`).Draw(rt, "to")
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
			From: from, To: to, Text: text,
		})
		g.Expect(err).NotTo(HaveOccurred())
		if err != nil {
			return
		}

		g.Expect(gotMethod).To(Equal(http.MethodPost))
		g.Expect(gotPath).To(Equal("/message"))
		g.Expect(gotBody.From).To(Equal(from))
		g.Expect(gotBody.To).To(Equal(to))
		g.Expect(gotBody.Text).To(Equal(text))
	})
}

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

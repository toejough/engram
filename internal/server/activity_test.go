package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/server"
)

func TestActivity_DefaultPagination(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "Test default params",
				CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
				FilePath:  "/data/memories/default-test.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	// No page or limit params — should default to page=1, limit=50.
	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var events []struct {
		MemorySlug string `json:"memorySlug"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &events)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0].MemorySlug).To(Equal("default-test"))
}

func TestActivity_EmptyMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var events []any

	err := json.Unmarshal(rec.Body.Bytes(), &events)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(events).To(BeEmpty())
}

func TestActivity_IncludesSurfacedEvents(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	created := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	surfacedAt := "2026-04-04T16:00:00Z"

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "When debugging flaky tests",
				CreatedAt: created,
				UpdatedAt: created,
				FilePath:  "/data/memories/flaky-tests.toml",
				PendingEvaluations: []memory.PendingEvaluation{
					{
						SurfacedAt: surfacedAt,
						UserPrompt: "How do I fix this flaky test?",
						SessionID:  "sess-123",
					},
				},
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity?page=1&limit=50", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var events []struct {
		Type       string `json:"type"`
		MemorySlug string `json:"memorySlug"`
		Context    string `json:"context"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &events)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(events).To(HaveLen(2))

	// Surfaced event should be newest (2026-04-04 > 2026-04-01).
	g.Expect(events[0].Type).To(Equal("surfaced"))
	g.Expect(events[0].MemorySlug).To(Equal("flaky-tests"))
	g.Expect(events[0].Context).To(Equal("How do I fix this flaky test?"))
}

func TestActivity_IncludesUpdatedEvents(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	created := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "When reviewing PRs",
				CreatedAt: created,
				UpdatedAt: updated,
				FilePath:  "/data/memories/reviewing-prs.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity?page=1&limit=50", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var events []struct {
		Type       string `json:"type"`
		MemorySlug string `json:"memorySlug"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &events)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(events).To(HaveLen(2))

	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.Type)
	}

	g.Expect(types).To(ContainElement("created"))
	g.Expect(types).To(ContainElement("updated"))
}

func TestActivity_InvalidLimitParam(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity?limit=-1", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["error"]).To(Equal("invalid limit parameter"))
}

func TestActivity_InvalidPageParam(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity?page=abc", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["error"]).To(Equal("invalid page parameter"))
}

func TestActivity_LimitExceedsMaxReturnsBadRequest(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity?limit=999", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))
}

func TestActivity_ListerError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{err: errListFailed}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity?page=1", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

func TestActivity_NoUpdatedEventWhenTimestampsEqual(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ts := time.Date(2026, 4, 3, 8, 0, 0, 0, time.UTC)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "When pair programming",
				CreatedAt: ts,
				UpdatedAt: ts,
				FilePath:  "/data/memories/pair-programming.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity?page=1&limit=50", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var events []struct {
		Type string `json:"type"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &events)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0].Type).To(Equal("created"))
}

func TestActivity_PageBeyondEnd(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "Single memory",
				CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
				FilePath:  "/data/memories/single.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity?page=100&limit=50", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var events []any

	err := json.Unmarshal(rec.Body.Bytes(), &events)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(events).To(BeEmpty())
}

func TestActivity_PageZeroReturnsBadRequest(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity?page=0", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))
}

func TestActivity_Pagination(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memories := make([]*memory.Stored, 0, 5)
	for i := range 5 {
		ts := time.Date(2026, 4, 1+i, 10, 0, 0, 0, time.UTC)
		memories = append(memories, &memory.Stored{
			Situation: "Memory " + strconv.Itoa(i),
			CreatedAt: ts,
			UpdatedAt: ts,
			FilePath:  "/data/memories/mem-" + strconv.Itoa(i) + ".toml",
		})
	}

	lister := &mockLister{memories: memories}

	srv := server.NewServer(lister, "/data")

	// Page 1, limit 2.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity?page=1&limit=2", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var page1 []struct {
		MemorySlug string `json:"memorySlug"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &page1)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(page1).To(HaveLen(2))
	g.Expect(page1[0].MemorySlug).To(Equal("mem-4"))
	g.Expect(page1[1].MemorySlug).To(Equal("mem-3"))

	// Page 2, limit 2.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/activity?page=2&limit=2", nil)

	srv.Handler().ServeHTTP(rec2, req2)

	g.Expect(rec2.Code).To(Equal(http.StatusOK))

	var page2 []struct {
		MemorySlug string `json:"memorySlug"`
	}

	err = json.Unmarshal(rec2.Body.Bytes(), &page2)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(page2).To(HaveLen(2))
	g.Expect(page2[0].MemorySlug).To(Equal("mem-2"))
	g.Expect(page2[1].MemorySlug).To(Equal("mem-1"))
}

func TestActivity_ReturnsEventsNewestFirst(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	earlier := now.Add(-24 * time.Hour)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "When writing tests",
				CreatedAt: earlier,
				UpdatedAt: earlier,
				FilePath:  "/data/memories/writing-tests.toml",
			},
			{
				Situation: "When deploying code",
				CreatedAt: now,
				UpdatedAt: now,
				FilePath:  "/data/memories/deploying-code.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity?page=1&limit=50", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))

	var events []struct {
		Type       string `json:"type"`
		Timestamp  string `json:"timestamp"`
		MemorySlug string `json:"memorySlug"`
		Context    string `json:"context"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &events)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(events).To(HaveLen(2))
	g.Expect(events[0].MemorySlug).To(Equal("deploying-code"))
	g.Expect(events[0].Type).To(Equal("created"))
	g.Expect(events[1].MemorySlug).To(Equal("writing-tests"))
}

func TestActivity_SkipsInvalidSurfacedAtTimestamp(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	created := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "Memory with bad surfaced_at",
				CreatedAt: created,
				UpdatedAt: created,
				FilePath:  "/data/memories/bad-surfaced.toml",
				PendingEvaluations: []memory.PendingEvaluation{
					{
						SurfacedAt: "not-a-valid-timestamp",
						UserPrompt: "Some prompt",
					},
				},
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var events []struct {
		Type string `json:"type"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &events)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Only created event, surfaced event skipped due to parse error.
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0].Type).To(Equal("created"))
}

func TestActivity_SkipsMemoriesWithZeroCreatedAt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "No timestamps",
				FilePath:  "/data/memories/no-time.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var events []any

	err := json.Unmarshal(rec.Body.Bytes(), &events)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(events).To(BeEmpty())
}

func TestActivity_SurfacedEventFallsBackToSituation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	created := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "When deploying services",
				CreatedAt: created,
				UpdatedAt: created,
				FilePath:  "/data/memories/deploy-svc.toml",
				PendingEvaluations: []memory.PendingEvaluation{
					{
						SurfacedAt: "2026-04-05T10:00:00Z",
						UserPrompt: "",
					},
				},
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var events []struct {
		Type    string `json:"type"`
		Context string `json:"context"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &events)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Find the surfaced event.
	var surfacedContext string

	for _, event := range events {
		if event.Type == "surfaced" {
			surfacedContext = event.Context
		}
	}

	g.Expect(surfacedContext).To(Equal("When deploying services"))
}

func TestActivity_ValidPageAndLimitParams(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ts := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "Valid params test",
				CreatedAt: ts,
				UpdatedAt: ts,
				FilePath:  "/data/memories/valid.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity?page=1&limit=100", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var events []struct {
		MemorySlug string `json:"memorySlug"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &events)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(events).To(HaveLen(1))
}

package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/server"
)

func TestCORS_AllowedOrigin(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories", nil)
	req.Header.Set("Origin", "http://localhost:5173")

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Header().Get("Access-Control-Allow-Origin")).To(Equal("http://localhost:5173"))
	g.Expect(rec.Header().Get("Access-Control-Allow-Methods")).To(ContainSubstring("GET"))
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories", nil)
	req.Header.Set("Origin", "http://evil.com")

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Header().Get("Access-Control-Allow-Origin")).To(BeEmpty())
}

func TestCORS_PreflightRequest(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/memories", nil)
	req.Header.Set("Origin", "http://localhost:5173")

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusNoContent))
	g.Expect(rec.Header().Get("Access-Control-Allow-Origin")).To(Equal("http://localhost:5173"))
}

func TestGetMemory_BackslashSlugRejected(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories/test%5Cpath", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))
}

func TestGetMemory_ListerError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{err: errListFailed}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories/anything", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

func TestGetMemory_NoPendingEvaluations(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				FollowedCount: 5,
				SurfacedCount: 5,
				FilePath:      "/data/memories/clean.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories/clean", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	pending, ok := result["pendingEvaluations"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(pending).To(BeEmpty())
}

func TestGetMemory_NotFound(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{FilePath: "/data/memories/existing.toml"},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories/nonexistent", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusNotFound))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["error"]).To(Equal("memory not found"))
}

func TestGetMemory_PathTraversalRejected(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories/..%2F..%2Fetc%2Fpasswd", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["error"]).To(Equal("invalid slug"))
}

func TestGetMemory_ReturnsDetailWithPendingEvaluations(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation:        "When writing tests",
				Behavior:         "Skip t.Parallel()",
				Impact:           "Tests run slowly",
				Action:           "Add t.Parallel()",
				ProjectScoped:    true,
				ProjectSlug:      "engram",
				SurfacedCount:    10,
				FollowedCount:    8,
				NotFollowedCount: 1,
				IrrelevantCount:  1,
				UpdatedAt:        time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
				FilePath:         "/data/memories/add-parallel.toml",
				PendingEvaluations: []memory.PendingEvaluation{
					{
						SurfacedAt:  "2026-04-03T09:00:00Z",
						UserPrompt:  "run tests",
						SessionID:   "sess-abc",
						ProjectSlug: "engram",
					},
				},
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories/add-parallel", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["slug"]).To(Equal("add-parallel"))
	g.Expect(result["situation"]).To(Equal("When writing tests"))
	g.Expect(result["effectiveness"]).To(BeNumerically("==", 80.0))

	pending, ok := result["pendingEvaluations"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(pending).To(HaveLen(1))

	pe, ok := pending[0].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(pe["surfacedAt"]).To(Equal("2026-04-03T09:00:00Z"))
	g.Expect(pe["userPrompt"]).To(Equal("run tests"))
	g.Expect(pe["sessionId"]).To(Equal("sess-abc"))
	g.Expect(pe["projectSlug"]).To(Equal("engram"))
}

func TestGetMemory_SlashOnlySlugRejected(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories/a%2Fb", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))
}

func TestListMemories_ComputedEffectiveness(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				FollowedCount:    8,
				NotFollowedCount: 1,
				IrrelevantCount:  1,
				SurfacedCount:    10,
				FilePath:         "/data/memories/test.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories", nil)

	srv.Handler().ServeHTTP(rec, req)

	var results []map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &results)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	if results == nil {
		return
	}

	g.Expect(results[0]["effectiveness"]).To(BeNumerically("==", 80.0))
}

func TestListMemories_ComputedQuadrant(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				FollowedCount:    8,
				NotFollowedCount: 2,
				SurfacedCount:    15,
				FilePath:         "/data/memories/working.toml",
			},
			{
				FollowedCount:    1,
				NotFollowedCount: 8,
				IrrelevantCount:  1,
				SurfacedCount:    5,
				FilePath:         "/data/memories/noise.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories", nil)

	srv.Handler().ServeHTTP(rec, req)

	var results []map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &results)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(HaveLen(2))

	quadrants := map[string]string{}

	for _, r := range results {
		slug, ok := r["slug"].(string)
		g.Expect(ok).To(BeTrue())

		quadrant, ok := r["quadrant"].(string)
		g.Expect(ok).To(BeTrue())

		quadrants[slug] = quadrant
	}

	// Median surfaced = (5+15)/2 = 10
	g.Expect(quadrants["working"]).To(Equal(memory.QuadrantWorking))
	g.Expect(quadrants["noise"]).To(Equal(memory.QuadrantNoise))
}

func TestListMemories_EmptyList(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var results []map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &results)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(BeEmpty())
}

func TestListMemories_InsufficientDataQuadrant(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				FollowedCount: 2,
				SurfacedCount: 5,
				FilePath:      "/data/memories/new.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories", nil)

	srv.Handler().ServeHTTP(rec, req)

	var results []map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &results)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	if results == nil {
		return
	}

	g.Expect(results[0]["quadrant"]).To(Equal(memory.QuadrantInsufficientData))
}

func TestListMemories_ListerError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		err: errListFailed,
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

func TestListMemories_ReturnsJSONArray(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation:        "When writing tests",
				Behavior:         "Skip t.Parallel()",
				Impact:           "Tests run slowly",
				Action:           "Add t.Parallel() to every test",
				ProjectScoped:    true,
				ProjectSlug:      "engram",
				SurfacedCount:    10,
				FollowedCount:    8,
				NotFollowedCount: 1,
				IrrelevantCount:  1,
				UpdatedAt:        time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
				FilePath:         "/data/memories/add-parallel-tests.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/memories", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))

	var results []map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &results)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(HaveLen(1))

	if results == nil {
		return
	}

	mem := results[0]
	g.Expect(mem["slug"]).To(Equal("add-parallel-tests"))
	g.Expect(mem["situation"]).To(Equal("When writing tests"))
	g.Expect(mem["behavior"]).To(Equal("Skip t.Parallel()"))
	g.Expect(mem["impact"]).To(Equal("Tests run slowly"))
	g.Expect(mem["action"]).To(Equal("Add t.Parallel() to every test"))
	g.Expect(mem["projectScoped"]).To(BeTrue())
	g.Expect(mem["projectSlug"]).To(Equal("engram"))
	g.Expect(mem["surfacedCount"]).To(BeNumerically("==", 10))
	g.Expect(mem["followedCount"]).To(BeNumerically("==", 8))
	g.Expect(mem["notFollowedCount"]).To(BeNumerically("==", 1))
	g.Expect(mem["irrelevantCount"]).To(BeNumerically("==", 1))
	g.Expect(mem["totalEvaluations"]).To(BeNumerically("==", 10))
	g.Expect(mem["updatedAt"]).To(Equal("2026-04-01T12:00:00Z"))
}

func TestListenAddr_BindsLocalhost(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	addr := server.ListenAddr("3001")
	g.Expect(addr).To(Equal("127.0.0.1:3001"))
}

func TestProjects_EmptyMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var results []map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &results)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(BeEmpty())
}

func TestProjects_ListerError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{err: errListFailed}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

func TestProjects_ReturnsPerProjectStats(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				ProjectSlug:      "engram",
				FollowedCount:    8,
				NotFollowedCount: 2,
				SurfacedCount:    15,
				FilePath:         "/data/memories/a.toml",
			},
			{
				ProjectSlug:      "engram",
				FollowedCount:    6,
				NotFollowedCount: 4,
				SurfacedCount:    5,
				FilePath:         "/data/memories/b.toml",
			},
			{
				ProjectSlug:      "other",
				FollowedCount:    1,
				NotFollowedCount: 8,
				IrrelevantCount:  1,
				SurfacedCount:    3,
				FilePath:         "/data/memories/c.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))

	var results []map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &results)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(HaveLen(2))

	if results == nil {
		return
	}

	// Sorted alphabetically: engram, other
	g.Expect(results[0]["projectSlug"]).To(Equal("engram"))
	g.Expect(results[0]["memoryCount"]).To(BeNumerically("==", 2))

	// engram: a=8/10=0.8, b=6/10=0.6; avg = (0.8+0.6)/2 = 0.7 * 100 = 70.0
	g.Expect(results[0]["avgEffectiveness"]).To(BeNumerically("==", 70.0))

	g.Expect(results[1]["projectSlug"]).To(Equal("other"))
	g.Expect(results[1]["memoryCount"]).To(BeNumerically("==", 1))

	// other: 1/10 = 0.1 * 100 = 10.0
	g.Expect(results[1]["avgEffectiveness"]).To(BeNumerically("==", 10.0))
}

func TestStats_EmptyMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["totalMemories"]).To(BeNumerically("==", 0))
	g.Expect(result["avgEffectiveness"]).To(BeNumerically("==", 0.0))
}

func TestStats_ListerError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{err: errListFailed}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

func TestStats_ReturnsAggregates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				FollowedCount:    8,
				NotFollowedCount: 2,
				SurfacedCount:    15,
				FilePath:         "/data/memories/a.toml",
			},
			{
				FollowedCount:    1,
				NotFollowedCount: 8,
				IrrelevantCount:  1,
				SurfacedCount:    5,
				FilePath:         "/data/memories/b.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["totalMemories"]).To(BeNumerically("==", 2))

	// a: 8/10 = 0.8, b: 1/10 = 0.1; avg = (0.8+0.1)/2 = 0.45 * 100 = 45.0
	g.Expect(result["avgEffectiveness"]).To(BeNumerically("==", 45.0))

	dist, ok := result["quadrantDistribution"].(map[string]any)
	g.Expect(ok).To(BeTrue())

	// Median surfaced = (5+15)/2 = 10; a(80%, 15>=10) = working, b(10%, 5<10) = noise
	g.Expect(dist[memory.QuadrantWorking]).To(BeNumerically("==", 1))
	g.Expect(dist[memory.QuadrantNoise]).To(BeNumerically("==", 1))
}

// unexported variables.
var (
	errListFailed = errorf("list failed")
)

type constError string

func (e constError) Error() string { return string(e) }

type mockLister struct {
	memories []*memory.Stored
	err      error
}

func (m *mockLister) ListMemories(_ context.Context, _ string) ([]*memory.Stored, error) {
	return m.memories, m.err
}

func errorf(msg string) constError { return constError(msg) }

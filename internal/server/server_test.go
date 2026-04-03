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

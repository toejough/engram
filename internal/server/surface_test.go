package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/server"
)

func TestSurface_EmptyResultsForNoMatch(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "When cooking pasta in the kitchen",
				Content:   memory.ContentFields{Action: "Always salt the water before boiling"},
				FilePath:  "/data/memories/cooking-pasta.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/surface?q=quantum+physics+entanglement", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	matches, ok := result["matches"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(matches).To(BeEmpty())

	nearMisses, ok := result["nearMisses"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(nearMisses).To(BeEmpty())
}

func TestSurface_IrrelevancePenaltyReducesScore(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Need enough non-matching memories so that matching terms (df=2 out of N=6)
	// still produce positive IDF values in BM25 scoring.
	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation:       "When running parallel tests",
				Content:         memory.ContentFields{Action: "Always add t.Parallel()"},
				IrrelevantCount: 0,
				FilePath:        "/data/memories/clean.toml",
			},
			{
				Situation:       "When running parallel tests",
				Content:         memory.ContentFields{Action: "Always add t.Parallel()"},
				IrrelevantCount: 10,
				FilePath:        "/data/memories/noisy.toml",
			},
			{
				Situation: "When deploying containers to production",
				Content:   memory.ContentFields{Action: "Check server logs for errors"},
				FilePath:  "/data/memories/deploy.toml",
			},
			{
				Situation: "When cooking pasta in the kitchen",
				Content:   memory.ContentFields{Action: "Salt the water before boiling"},
				FilePath:  "/data/memories/cooking.toml",
			},
			{
				Situation: "When writing documentation for APIs",
				Content:   memory.ContentFields{Action: "Include request and response examples"},
				FilePath:  "/data/memories/docs.toml",
			},
			{
				Situation: "When configuring database connections",
				Content:   memory.ContentFields{Action: "Set connection pool limits"},
				FilePath:  "/data/memories/database.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data",
		server.WithSurfaceThreshold(0.001),
		server.WithIrrelevanceHalfLife(5),
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/surface?q=parallel+tests", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var result struct {
		Matches []struct {
			Slug      string  `json:"slug"`
			BM25Score float64 `json:"bm25Score"`
		} `json:"matches"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Matches).To(HaveLen(2))

	scoreBySlug := map[string]float64{}
	for _, match := range result.Matches {
		scoreBySlug[match.Slug] = match.BM25Score
	}

	g.Expect(scoreBySlug["clean"]).To(BeNumerically(">", scoreBySlug["noisy"]))
}

func TestSurface_ListerError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{err: errListFailed}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/surface?q=test", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

func TestSurface_MissingQueryReturns400(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{memories: []*memory.Stored{}}

	srv := server.NewServer(lister, "/data")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/surface", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))

	var result map[string]any

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result["error"]).To(Equal("missing required parameter: q"))
}

func TestSurface_NearMissesIncludeThreshold(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "When running parallel tests",
				Content:   memory.ContentFields{Action: "Always add t.Parallel()"},
				FilePath:  "/data/memories/parallel-tests.toml",
			},
			{
				Situation: "When deploying containers to production",
				Content:   memory.ContentFields{Action: "Check server logs for errors"},
				FilePath:  "/data/memories/deploy.toml",
			},
			{
				Situation: "When reviewing code changes",
				Content:   memory.ContentFields{Action: "Verify documentation coverage"},
				FilePath:  "/data/memories/code-review.toml",
			},
		},
	}

	// Threshold set above typical BM25 scores but low enough that
	// floor (threshold * 0.5) is below the actual BM25 score.
	srv := server.NewServer(lister, "/data",
		server.WithSurfaceThreshold(2.0),
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/surface?q=parallel+tests", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var result struct {
		Matches    []any `json:"matches"`
		NearMisses []struct {
			Slug       string  `json:"slug"`
			BM25Score  float64 `json:"bm25Score"`
			FinalScore float64 `json:"finalScore"`
			Situation  string  `json:"situation"`
			Action     string  `json:"action"`
			Threshold  float64 `json:"threshold"`
		} `json:"nearMisses"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Matches).To(BeEmpty())
	g.Expect(result.NearMisses).To(HaveLen(1))

	nearMiss := result.NearMisses[0]
	g.Expect(nearMiss.Slug).To(Equal("parallel-tests"))
	g.Expect(nearMiss.BM25Score).To(BeNumerically(">", 0))
	g.Expect(nearMiss.FinalScore).To(BeNumerically(">", 0))
	g.Expect(nearMiss.Situation).To(Equal("When running parallel tests"))
	g.Expect(nearMiss.Action).To(Equal("Always add t.Parallel()"))
	g.Expect(nearMiss.Threshold).To(BeNumerically("==", 2.0))
}

func TestSurface_ProjectScopeExcludesCrossProject(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation:     "When running parallel tests",
				Content:       memory.ContentFields{Action: "Always add t.Parallel()"},
				ProjectScoped: true,
				ProjectSlug:   "alpha",
				FilePath:      "/data/memories/parallel-tests.toml",
			},
			{
				Situation: "When deploying containers to production",
				Content:   memory.ContentFields{Action: "Check server logs for errors"},
				FilePath:  "/data/memories/deploy.toml",
			},
			{
				Situation: "When reviewing code changes",
				Content:   memory.ContentFields{Action: "Verify documentation coverage"},
				FilePath:  "/data/memories/code-review.toml",
			},
		},
	}

	srv := server.NewServer(lister, "/data",
		server.WithSurfaceThreshold(0.001),
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/surface?q=parallel+tests&project=beta", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var result struct {
		Matches []struct {
			Slug string `json:"slug"`
		} `json:"matches"`
		NearMisses []struct {
			Slug string `json:"slug"`
		} `json:"nearMisses"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Project-scoped memory from "alpha" should be excluded when querying project "beta".
	for _, match := range result.Matches {
		g.Expect(match.Slug).NotTo(Equal("parallel-tests"))
	}

	for _, nearMiss := range result.NearMisses {
		g.Expect(nearMiss.Slug).NotTo(Equal("parallel-tests"))
	}
}

func TestSurface_ReturnsMatchesAboveThreshold(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	lister := &mockLister{
		memories: []*memory.Stored{
			{
				Situation: "When running parallel tests",
				Content:   memory.ContentFields{Action: "Always add t.Parallel()"},
				FilePath:  "/data/memories/parallel-tests.toml",
			},
			{
				Situation: "When deploying containers to production",
				Content:   memory.ContentFields{Action: "Check server logs for errors"},
				FilePath:  "/data/memories/deploy.toml",
			},
			{
				Situation: "When reviewing code changes",
				Content:   memory.ContentFields{Action: "Verify documentation coverage"},
				FilePath:  "/data/memories/code-review.toml",
			},
		},
	}

	// Low threshold ensures any scored result becomes a match.
	srv := server.NewServer(lister, "/data",
		server.WithSurfaceThreshold(0.001),
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/surface?q=parallel+tests", nil)

	srv.Handler().ServeHTTP(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))

	var result struct {
		Matches []struct {
			Slug       string  `json:"slug"`
			BM25Score  float64 `json:"bm25Score"`
			FinalScore float64 `json:"finalScore"`
			Situation  string  `json:"situation"`
			Action     string  `json:"action"`
		} `json:"matches"`
		NearMisses []any `json:"nearMisses"`
	}

	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Matches).NotTo(BeEmpty())
	g.Expect(result.NearMisses).To(BeEmpty())

	match := result.Matches[0]
	g.Expect(match.Slug).To(Equal("parallel-tests"))
	g.Expect(match.BM25Score).To(BeNumerically(">", 0))
	g.Expect(match.FinalScore).To(BeNumerically(">", 0))
	g.Expect(match.Situation).To(Equal("When running parallel tests"))
	g.Expect(match.Action).To(Equal("Always add t.Parallel()"))
}

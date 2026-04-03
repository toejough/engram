// Package server implements the engram HTTP API server.
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"engram/internal/memory"
)

// Exported constants.
const (
	DefaultPort = "3001"
)

// MemoryLister retrieves all stored memories from the data directory.
type MemoryLister interface {
	ListMemories(ctx context.Context, dataDir string) ([]*memory.Stored, error)
}

// Server is the engram HTTP API server.
type Server struct {
	lister  MemoryLister
	dataDir string
	mux     *http.ServeMux
}

// NewServer creates a Server with the given dependencies and wires routes.
func NewServer(lister MemoryLister, dataDir string) *Server {
	s := &Server{
		lister:  lister,
		dataDir: dataDir,
		mux:     http.NewServeMux(),
	}

	s.mux.HandleFunc("GET /api/memories", s.handleListMemories)
	s.mux.HandleFunc("GET /api/memories/{slug}", s.handleGetMemory)
	s.mux.HandleFunc("GET /api/stats", s.handleStats)
	s.mux.HandleFunc("GET /api/projects", s.handleProjects)

	return s
}

// Handler returns the server's HTTP handler with CORS middleware applied.
func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux)
}

func (s *Server) handleGetMemory(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if !isValidSlug(slug) {
		writeJSONError(w, "invalid slug", http.StatusBadRequest)

		return
	}

	memories, err := s.lister.ListMemories(r.Context(), s.dataDir)
	if err != nil {
		writeJSONError(w, "failed to list memories", http.StatusInternalServerError)

		return
	}

	medianSurfaced := medianSurfacedCount(memories)

	for _, mem := range memories {
		if memory.NameFromPath(mem.FilePath) == slug {
			resp := toMemoryDetailResponse(mem, medianSurfaced)

			w.Header().Set("Content-Type", "application/json")

			encodeErr := json.NewEncoder(w).Encode(resp)
			if encodeErr != nil {
				writeJSONError(w, "failed to encode response", http.StatusInternalServerError)
			}

			return
		}
	}

	writeJSONError(w, "memory not found", http.StatusNotFound)
}

func (s *Server) handleListMemories(w http.ResponseWriter, r *http.Request) {
	memories, err := s.lister.ListMemories(r.Context(), s.dataDir)
	if err != nil {
		http.Error(w, `{"error":"failed to list memories"}`, http.StatusInternalServerError)

		return
	}

	medianSurfaced := medianSurfacedCount(memories)
	responses := make([]memoryResponse, 0, len(memories))

	for _, mem := range memories {
		responses = append(responses, toMemoryResponse(mem, medianSurfaced))
	}

	w.Header().Set("Content-Type", "application/json")

	encodeErr := json.NewEncoder(w).Encode(responses)
	if encodeErr != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
	}
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	memories, err := s.lister.ListMemories(r.Context(), s.dataDir)
	if err != nil {
		writeJSONError(w, "failed to list memories", http.StatusInternalServerError)

		return
	}

	medianSurfaced := medianSurfacedCount(memories)
	projectMap := map[string]*projectAccumulator{}

	for _, mem := range memories {
		slug := mem.ProjectSlug
		acc, exists := projectMap[slug]

		if !exists {
			acc = &projectAccumulator{quadrants: map[string]int{}}
			projectMap[slug] = acc
		}

		acc.count++
		acc.totalEffectiveness += mem.Effectiveness()
		quadrant := mem.Quadrant(medianSurfaced)
		acc.quadrants[quadrant]++
	}

	responses := make([]projectResponse, 0, len(projectMap))

	for slug, acc := range projectMap {
		avgEff := 0.0
		if acc.count > 0 {
			avgEff = (acc.totalEffectiveness / float64(acc.count)) * effectivenessPercentMultiplier
		}

		responses = append(responses, projectResponse{
			ProjectSlug:       slug,
			MemoryCount:       acc.count,
			AvgEffectiveness:  avgEff,
			QuadrantBreakdown: acc.quadrants,
		})
	}

	sort.Slice(responses, func(i, j int) bool {
		return responses[i].ProjectSlug < responses[j].ProjectSlug
	})

	w.Header().Set("Content-Type", "application/json")

	encodeErr := json.NewEncoder(w).Encode(responses)
	if encodeErr != nil {
		writeJSONError(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	memories, err := s.lister.ListMemories(r.Context(), s.dataDir)
	if err != nil {
		writeJSONError(w, "failed to list memories", http.StatusInternalServerError)

		return
	}

	medianSurfaced := medianSurfacedCount(memories)
	totalEffectiveness := 0.0
	quadrantDist := map[string]int{}

	for _, mem := range memories {
		totalEffectiveness += mem.Effectiveness()
		quadrant := mem.Quadrant(medianSurfaced)
		quadrantDist[quadrant]++
	}

	avgEffectiveness := 0.0
	if len(memories) > 0 {
		avgEffectiveness = (totalEffectiveness / float64(len(memories))) * effectivenessPercentMultiplier
	}

	resp := statsResponse{
		TotalMemories:        len(memories),
		AvgEffectiveness:     avgEffectiveness,
		QuadrantDistribution: quadrantDist,
	}

	w.Header().Set("Content-Type", "application/json")

	encodeErr := json.NewEncoder(w).Encode(resp)
	if encodeErr != nil {
		writeJSONError(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// ListenAddr returns a listen address bound to localhost only.
func ListenAddr(port string) string {
	return "127.0.0.1:" + port
}

// unexported constants.
const (
	allowedOrigin                  = "http://localhost:5173"
	effectivenessPercentMultiplier = 100.0
	medianDivisor                  = 2
)

type errorResponse struct {
	Error string `json:"error"`
}

type memoryDetailResponse struct {
	memoryResponse //nolint:unused // embedded for JSON flattening

	PendingEvaluations []pendingEvaluationResponse `json:"pendingEvaluations"`
}

type memoryResponse struct {
	Slug             string  `json:"slug"`
	Situation        string  `json:"situation"`
	Behavior         string  `json:"behavior"`
	Impact           string  `json:"impact"`
	Action           string  `json:"action"`
	ProjectScoped    bool    `json:"projectScoped"`
	ProjectSlug      string  `json:"projectSlug"`
	SurfacedCount    int     `json:"surfacedCount"`
	FollowedCount    int     `json:"followedCount"`
	NotFollowedCount int     `json:"notFollowedCount"`
	IrrelevantCount  int     `json:"irrelevantCount"`
	Effectiveness    float64 `json:"effectiveness"`
	Quadrant         string  `json:"quadrant"`
	TotalEvaluations int     `json:"totalEvaluations"`
	UpdatedAt        string  `json:"updatedAt"`
}

type pendingEvaluationResponse struct {
	SurfacedAt  string `json:"surfacedAt"`
	UserPrompt  string `json:"userPrompt"`
	SessionID   string `json:"sessionId"`
	ProjectSlug string `json:"projectSlug"`
}

type projectAccumulator struct {
	count              int
	totalEffectiveness float64
	quadrants          map[string]int
}

type projectResponse struct {
	ProjectSlug       string         `json:"projectSlug"`
	MemoryCount       int            `json:"memoryCount"`
	AvgEffectiveness  float64        `json:"avgEffectiveness"`
	QuadrantBreakdown map[string]int `json:"quadrantBreakdown"`
}

type statsResponse struct {
	TotalMemories        int            `json:"totalMemories"`
	AvgEffectiveness     float64        `json:"avgEffectiveness"`
	QuadrantDistribution map[string]int `json:"quadrantDistribution"`
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func isValidSlug(slug string) bool {
	if slug == "" {
		return false
	}

	if strings.Contains(slug, "..") || strings.Contains(slug, "/") || strings.Contains(slug, "\\") {
		return false
	}

	return true
}

func medianSurfacedCount(memories []*memory.Stored) int {
	if len(memories) == 0 {
		return 0
	}

	counts := make([]int, len(memories))
	for i, mem := range memories {
		counts[i] = mem.SurfacedCount
	}

	sort.Ints(counts)

	mid := len(counts) / medianDivisor

	if len(counts)%medianDivisor == 0 {
		return (counts[mid-1] + counts[mid]) / medianDivisor
	}

	return counts[mid]
}

func toMemoryDetailResponse(mem *memory.Stored, medianSurfaced int) memoryDetailResponse {
	base := toMemoryResponse(mem, medianSurfaced)

	pending := make([]pendingEvaluationResponse, 0, len(mem.PendingEvaluations))
	for _, pe := range mem.PendingEvaluations {
		pending = append(pending, pendingEvaluationResponse{
			SurfacedAt:  pe.SurfacedAt,
			UserPrompt:  pe.UserPrompt,
			SessionID:   pe.SessionID,
			ProjectSlug: pe.ProjectSlug,
		})
	}

	return memoryDetailResponse{
		memoryResponse:     base,
		PendingEvaluations: pending,
	}
}

func toMemoryResponse(mem *memory.Stored, medianSurfaced int) memoryResponse {
	effectiveness := mem.Effectiveness()

	return memoryResponse{
		Slug:             memory.NameFromPath(mem.FilePath),
		Situation:        mem.Situation,
		Behavior:         mem.Behavior,
		Impact:           mem.Impact,
		Action:           mem.Action,
		ProjectScoped:    mem.ProjectScoped,
		ProjectSlug:      mem.ProjectSlug,
		SurfacedCount:    mem.SurfacedCount,
		FollowedCount:    mem.FollowedCount,
		NotFollowedCount: mem.NotFollowedCount,
		IrrelevantCount:  mem.IrrelevantCount,
		Effectiveness:    effectiveness * effectivenessPercentMultiplier,
		Quadrant:         mem.Quadrant(medianSurfaced),
		TotalEvaluations: mem.TotalEvaluations(),
		UpdatedAt:        mem.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func writeJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	resp := errorResponse{Error: message}

	encodeErr := json.NewEncoder(w).Encode(resp)
	if encodeErr != nil {
		http.Error(w, message, code)
	}
}

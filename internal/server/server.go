// Package server implements the engram HTTP API server.
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

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

	return s
}

// Handler returns the server's HTTP handler with CORS middleware applied.
func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux)
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

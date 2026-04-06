package server

import (
	"encoding/json"
	"net/http"
	"sort"

	"engram/internal/bm25"
	"engram/internal/memory"
)

func (s *Server) classifyScoredResults(
	scored []bm25.ScoredDocument,
	memIndex map[string]*memory.Stored,
	project string,
) ([]surfaceMatchResponse, []surfaceNearMissResponse) {
	threshold := s.surfaceThreshold
	nearMissFloor := threshold * nearMissMinFraction

	matches := make([]surfaceMatchResponse, 0)
	nearMisses := make([]surfaceNearMissResponse, 0)

	for _, result := range scored {
		mem := memIndex[result.ID]
		if mem == nil {
			continue
		}

		penalizedScore := result.Score * surfaceIrrelevancePenalty(
			mem.IrrelevantCount, s.irrelevanceHalfLife,
		)

		genFac := surfaceGenFactor(mem.ProjectScoped, mem.ProjectSlug, project)
		finalScore := penalizedScore * genFac

		if finalScore <= 0 {
			continue
		}

		slug := memory.NameFromPath(mem.FilePath)

		matchResp := surfaceMatchResponse{
			Slug:       slug,
			BM25Score:  penalizedScore,
			FinalScore: finalScore,
			Situation:  mem.Situation,
			Action:     mem.Action,
		}

		if finalScore >= threshold {
			matches = append(matches, matchResp)
		} else if finalScore >= nearMissFloor {
			nearMisses = append(nearMisses, surfaceNearMissResponse{
				surfaceMatchResponse: matchResp,
				Threshold:            threshold,
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].FinalScore > matches[j].FinalScore
	})

	sort.Slice(nearMisses, func(i, j int) bool {
		return nearMisses[i].FinalScore > nearMisses[j].FinalScore
	})

	return matches, nearMisses
}

func (s *Server) handleSurface(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSONError(w, "missing required parameter: q", http.StatusBadRequest)

		return
	}

	project := r.URL.Query().Get("project")

	memories, err := s.lister.ListMemories(r.Context(), s.dataDir)
	if err != nil {
		writeJSONError(w, "failed to list memories", http.StatusInternalServerError)

		return
	}

	resp := s.scoreSurfaceResults(query, project, memories)

	w.Header().Set("Content-Type", "application/json")

	encodeErr := json.NewEncoder(w).Encode(resp)
	if encodeErr != nil {
		writeJSONError(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) scoreSurfaceResults(
	query, project string, memories []*memory.Stored,
) surfaceResponse {
	docs := make([]bm25.Document, 0, len(memories))
	memIndex := make(map[string]*memory.Stored, len(memories))

	for _, mem := range memories {
		docs = append(docs, bm25.Document{
			ID:   mem.FilePath,
			Text: mem.SearchText(),
		})

		memIndex[mem.FilePath] = mem
	}

	scorer := bm25.New()
	scored := scorer.Score(query, docs)

	matches, nearMisses := s.classifyScoredResults(scored, memIndex, project)

	return surfaceResponse{
		Matches:    matches,
		NearMisses: nearMisses,
	}
}

// WithIrrelevanceHalfLife sets the irrelevance half-life for the surface endpoint.
func WithIrrelevanceHalfLife(halfLife int) Option {
	return func(s *Server) { s.irrelevanceHalfLife = halfLife }
}

// WithSurfaceThreshold sets the BM25 threshold for the surface endpoint.
func WithSurfaceThreshold(threshold float64) Option {
	return func(s *Server) { s.surfaceThreshold = threshold }
}

// unexported constants.
const (
	defaultBM25Threshold       = 0.3
	defaultIrrelevanceHalfLife = 5
	nearMissMinFraction        = 0.5
)

type surfaceMatchResponse struct {
	Slug       string  `json:"slug"`
	BM25Score  float64 `json:"bm25Score"`
	FinalScore float64 `json:"finalScore"`
	Situation  string  `json:"situation"`
	Action     string  `json:"action"`
}

type surfaceNearMissResponse struct {
	surfaceMatchResponse

	Threshold float64 `json:"threshold"`
}

type surfaceResponse struct {
	Matches    []surfaceMatchResponse    `json:"matches"`
	NearMisses []surfaceNearMissResponse `json:"nearMisses"`
}

// surfaceGenFactor returns the BM25 relevance penalty for project-scoped memories
// in a different project (inlined from surface.GenFactor to avoid package dependency).
func surfaceGenFactor(projectScoped bool, memProject, currentProject string) float64 {
	if !projectScoped {
		return 1.0
	}

	if memProject == "" || currentProject == "" || memProject == currentProject {
		return 1.0
	}

	return 0.0
}

// surfaceIrrelevancePenalty computes a BM25 score multiplier based on irrelevant feedback
// (inlined from surface.irrelevancePenalty to avoid package dependency).
func surfaceIrrelevancePenalty(irrelevantCount, halfLife int) float64 {
	return float64(halfLife) / float64(halfLife+irrelevantCount)
}

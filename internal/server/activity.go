package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"time"

	"engram/internal/memory"
)

func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	page, limit, parseErr := parseActivityParams(r)
	if parseErr != "" {
		writeJSONError(w, parseErr, http.StatusBadRequest)

		return
	}

	memories, err := s.lister.ListMemories(r.Context(), s.dataDir)
	if err != nil {
		writeJSONError(w, "failed to list memories", http.StatusInternalServerError)

		return
	}

	events := synthesizeEvents(memories)

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})

	offset := min((page-1)*limit, len(events))
	end := min(offset+limit, len(events))

	pageEvents := events[offset:end]

	responses := make([]activityEventResponse, 0, len(pageEvents))
	for _, event := range pageEvents {
		responses = append(responses, activityEventResponse{
			Type:       event.Type,
			Timestamp:  event.Timestamp.Format(time.RFC3339),
			MemorySlug: event.MemorySlug,
			Context:    event.Context,
		})
	}

	w.Header().Set("Content-Type", "application/json")

	encodeErr := json.NewEncoder(w).Encode(responses)
	if encodeErr != nil {
		writeJSONError(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// unexported constants.
const (
	defaultActivityLimit = 50
	eventTypeCreated     = "created"
	eventTypeSurfaced    = "surfaced"
	eventTypeUpdated     = "updated"
	maxActivityLimit     = 200
)

// unexported types and helpers for activity.

type activityEvent struct {
	Type       string
	Timestamp  time.Time
	MemorySlug string
	Context    string
}

type activityEventResponse struct {
	Type       string `json:"type"`
	Timestamp  string `json:"timestamp"`
	MemorySlug string `json:"memorySlug"`
	Context    string `json:"context"`
}

func parseActivityParams(r *http.Request) (int, int, string) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	limit := defaultActivityLimit

	if pageStr != "" {
		parsed, err := strconv.Atoi(pageStr)
		if err != nil || parsed < 1 {
			return 0, 0, "invalid page parameter"
		}

		page = parsed
	}

	if limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 || parsed > maxActivityLimit {
			return 0, 0, "invalid limit parameter"
		}

		limit = parsed
	}

	return page, limit, ""
}

func synthesizeEvents(memories []*memory.Stored) []activityEvent {
	events := make([]activityEvent, 0)

	for _, mem := range memories {
		slug := memory.NameFromPath(mem.FilePath)

		if !mem.CreatedAt.IsZero() {
			events = append(events, activityEvent{
				Type:       eventTypeCreated,
				Timestamp:  mem.CreatedAt,
				MemorySlug: slug,
				Context:    mem.Situation,
			})
		}

		if !mem.UpdatedAt.IsZero() && !mem.CreatedAt.IsZero() && !mem.UpdatedAt.Equal(mem.CreatedAt) {
			events = append(events, activityEvent{
				Type:       eventTypeUpdated,
				Timestamp:  mem.UpdatedAt,
				MemorySlug: slug,
				Context:    mem.Situation,
			})
		}

		for _, pendingEval := range mem.PendingEvaluations {
			surfacedAt, parseErr := time.Parse(time.RFC3339, pendingEval.SurfacedAt)
			if parseErr != nil {
				continue
			}

			context := pendingEval.UserPrompt
			if context == "" {
				context = mem.Situation
			}

			events = append(events, activityEvent{
				Type:       eventTypeSurfaced,
				Timestamp:  surfacedAt,
				MemorySlug: slug,
				Context:    context,
			})
		}
	}

	return events
}

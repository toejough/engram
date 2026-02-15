// batch_extract_test.go
package memory_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/toejough/projctl/internal/memory"
)

func TestIdentifyEvents_ParsesHaikuResponse(t *testing.T) {
	// Mock Haiku returning events (note: response is WITHOUT the leading [
	// because the prefill provides it)
	responseEvents := `
		{"line_range": "1-20", "event_type": "root-cause-discovery", "what_happened": "Found type mismatch.", "why_it_matters": "Check types."},
		{"line_range": "30-40", "event_type": "user-correction", "what_happened": "User said no.", "why_it_matters": "Listen."}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"text": responseEvents},
			},
		})
	}))
	defer server.Close()

	ext := memory.NewDirectAPIExtractor("test-token",
		memory.WithBaseURL(server.URL),
	)

	chunk := memory.TextChunk{
		Text:      "[user] fix the bug\n[assistant] Found it.",
		StartLine: 1,
		EndLine:   20,
		Index:     0,
	}

	events, err := ext.IdentifyEvents(context.Background(), chunk, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].EventType != "root-cause-discovery" {
		t.Errorf("event 0 type: want root-cause-discovery, got %s", events[0].EventType)
	}
	if events[0].ChunkIndex != 0 {
		t.Errorf("event 0 chunk: want 0, got %d", events[0].ChunkIndex)
	}
}

func TestIdentifyEvents_HandlesEmptyArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"text": "]"},
			},
		})
	}))
	defer server.Close()

	ext := memory.NewDirectAPIExtractor("test-token",
		memory.WithBaseURL(server.URL),
	)

	chunk := memory.TextChunk{Text: "nothing here", StartLine: 1, EndLine: 1, Index: 0}
	events, err := ext.IdentifyEvents(context.Background(), chunk, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

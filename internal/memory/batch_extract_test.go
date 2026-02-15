// batch_extract_test.go
package memory_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestExtractPrinciples_ParsesSonnetResponse(t *testing.T) {
	responsePrinciples := `
		{"principle": "When X, do Y.", "evidence": "Session showed X failing.", "category": "debugging"}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Sonnet model is used
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		model, _ := body["model"].(string)
		if model != "claude-sonnet-4-5-20250929" {
			t.Errorf("expected sonnet model, got %s", model)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"text": responsePrinciples},
			},
		})
	}))
	defer server.Close()

	ext := memory.NewDirectAPIExtractor("test-token",
		memory.WithBaseURL(server.URL),
	)

	events := []memory.HaikuEvent{
		{EventType: "root-cause-discovery", WhatHappened: "Found bug.", WhyItMatters: "Check types."},
	}

	principles, err := ext.ExtractPrinciples(context.Background(), events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(principles) != 1 {
		t.Fatalf("expected 1 principle, got %d", len(principles))
	}
	if principles[0].Category != "debugging" {
		t.Errorf("category: want debugging, got %s", principles[0].Category)
	}
}

func TestExtractPrinciples_NoEvents(t *testing.T) {
	ext := memory.NewDirectAPIExtractor("test-token")
	principles, err := ext.ExtractPrinciples(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(principles) != 0 {
		t.Errorf("expected 0 principles for nil events, got %d", len(principles))
	}
}

func TestBatchExtractSession_EndToEnd(t *testing.T) {
	// Create a minimal session JSONL
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")
	content := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"fix the auth bug"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Found it: ExpiresAt was string, should be any."}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"auth.go","old_string":"ExpiresAt string","new_string":"ExpiresAt any"}}]}}`,
	}, "\n")
	os.WriteFile(sessionPath, []byte(content), 0644)

	// Mock server that responds to both Haiku and Sonnet calls
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		model, _ := body["model"].(string)

		callCount++
		if strings.Contains(model, "haiku") {
			// Return a single event
			json.NewEncoder(w).Encode(map[string]any{
				"content": []map[string]any{
					{"text": `{"event_type":"root-cause-discovery","what_happened":"Type mismatch found.","why_it_matters":"Check types.","line_range":"1-3"}]`},
				},
			})
		} else {
			// Sonnet: return a principle
			json.NewEncoder(w).Encode(map[string]any{
				"content": []map[string]any{
					{"text": `{"principle":"When integrating with external data, use flexible types.","evidence":"ExpiresAt was string but should be any.","category":"debugging"}]`},
				},
			})
		}
	}))
	defer server.Close()

	ext := memory.NewDirectAPIExtractor("test-token",
		memory.WithBaseURL(server.URL),
	)

	result, err := memory.BatchExtractSession(context.Background(), sessionPath, ext, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.StrippedSize == 0 {
		t.Error("stripped size should be > 0")
	}
	if result.ChunkCount == 0 {
		t.Error("chunk count should be > 0")
	}
	if len(result.Events) == 0 {
		t.Error("should have events")
	}
	if len(result.Principles) == 0 {
		t.Error("should have principles")
	}
	if result.EndOffset == 0 {
		t.Error("EndOffset should be > 0")
	}
}

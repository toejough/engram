package memory_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/toejough/projctl/internal/memory"
)

func TestCallAPIWithMessages_SendsSystemAndPrefill(t *testing.T) {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"text": `{"result": "ok"}`},
			},
		})
	}))
	defer server.Close()

	ext := memory.NewDirectAPIExtractor("test-token",
		memory.WithBaseURL(server.URL),
	)

	params := memory.APIMessageParams{
		System: "You are a test analyzer.",
		Messages: []memory.APIMessage{
			{Role: "user", Content: "Analyze this."},
			{Role: "assistant", Content: "["},
		},
		MaxTokens: 1024,
	}

	_, err := ext.CallAPIWithMessages(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify system prompt was sent
	sys, _ := receivedBody["system"].(string)
	if sys != "You are a test analyzer." {
		t.Errorf("system prompt not sent: got %q", sys)
	}

	// Verify messages include both user and assistant
	msgs, _ := receivedBody["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	msg1, _ := msgs[1].(map[string]any)
	if role, _ := msg1["role"].(string); role != "assistant" {
		t.Errorf("second message role: want assistant, got %s", role)
	}

	if content, _ := msg1["content"].(string); content != "[" {
		t.Errorf("assistant prefill: want [, got %s", content)
	}
}

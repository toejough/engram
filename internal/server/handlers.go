package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"engram/internal/chat"
)

// Deps holds injected dependencies for HTTP handlers.
type Deps struct {
	// PostMessage writes a message to the chat file and returns the new cursor.
	PostMessage PostFunc

	// WatchForMessage blocks until a message matching from/to appears after the cursor.
	// Independently watches the file (does not rely on goroutine cursors).
	WatchForMessage func(ctx context.Context, from, toAgent string, afterCursor int) (chat.Message, int, error)

	// SubscribeMessages blocks until new messages for the agent appear after cursor.
	SubscribeMessages func(ctx context.Context, agent string, afterCursor int) ([]chat.Message, int, error)

	// Logger is used for structured event logging.
	Logger *slog.Logger

	// ShutdownFn is called by POST /shutdown to initiate graceful shutdown.
	ShutdownFn context.CancelFunc
}

// logger returns deps.Logger if set, otherwise slog.Default().
func (d *Deps) logger() *slog.Logger {
	if d.Logger != nil {
		return d.Logger
	}

	return slog.Default()
}

// PostFunc writes a message to the chat file and returns the new cursor.
type PostFunc func(msg chat.Message) (int, error)

// HandlePostMessage returns an http.HandlerFunc for POST /message.
// Decodes JSON {from, to, text}, calls PostMessage, returns {cursor}.
func HandlePostMessage(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			From string `json:"from"`
			To   string `json:"to"`
			Text string `json:"text"`
		}

		decErr := json.NewDecoder(r.Body).Decode(&req)
		if decErr != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)

			return
		}

		cursor, postErr := deps.PostMessage(chat.Message{
			From: req.From,
			To:   req.To,
			Text: req.Text,
		})
		if postErr != nil {
			deps.logger().Error("posting message", "err", postErr)
			http.Error(w, `{"error":"failed to post"}`, http.StatusInternalServerError)

			return
		}

		deps.logger().Info("message posted",
			"from", req.From,
			"to", req.To,
			"text_len", len(req.Text),
			"cursor", cursor,
		)

		writeJSON(w, postMessageResponse{Cursor: cursor})
	}
}

// HandleShutdown returns an http.HandlerFunc for POST /shutdown.
// Calls ShutdownFn and returns an acknowledgment.
func HandleShutdown(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		deps.logger().Info("shutdown requested")

		writeJSON(w, shutdownResponse{Status: "shutting down"})

		if deps.ShutdownFn != nil {
			deps.ShutdownFn()
		}
	}
}

// HandleStatus returns an http.HandlerFunc for GET /status.
// Always returns {"running": true}.
func HandleStatus(_ *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, statusResponse{Running: true})
	}
}

// HandleSubscribe returns an http.HandlerFunc for GET /subscribe.
// Query params: agent (required), after-cursor (optional, defaults to 0).
// Returns {messages, cursor}.
func HandleSubscribe(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := r.URL.Query().Get("agent")
		afterCursorStr := r.URL.Query().Get("after-cursor")

		afterCursor, parseErr := strconv.Atoi(afterCursorStr)
		if parseErr != nil {
			afterCursor = 0
		}

		messages, newCursor, watchErr := deps.SubscribeMessages(r.Context(), agent, afterCursor)
		if watchErr != nil {
			deps.logger().Error("subscribing", "err", watchErr, "agent", agent)
			http.Error(w, `{"error":"subscribe failed"}`, http.StatusInternalServerError)

			return
		}

		writeJSON(w, subscribeResponse{Messages: messages, Cursor: newCursor})
	}
}

// HandleWaitForResponse returns an http.HandlerFunc for GET /wait-for-response.
// Query params: from, to, after-cursor (required integer).
// Returns {text, from, to, cursor} for the matched message.
func HandleWaitForResponse(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from := r.URL.Query().Get("from")
		toAgent := r.URL.Query().Get("to")
		afterCursorStr := r.URL.Query().Get("after-cursor")

		afterCursor, parseErr := strconv.Atoi(afterCursorStr)
		if parseErr != nil {
			http.Error(w, `{"error":"invalid after-cursor"}`, http.StatusBadRequest)

			return
		}

		msg, newCursor, watchErr := deps.WatchForMessage(r.Context(), from, toAgent, afterCursor)
		if watchErr != nil {
			deps.logger().Error("watching for response", "err", watchErr, "from", from, "to", toAgent)
			http.Error(w, `{"error":"watch failed"}`, http.StatusInternalServerError)

			return
		}

		writeJSON(w, waitForResponseResponse{
			Text:   msg.Text,
			From:   msg.From,
			To:     msg.To,
			Cursor: newCursor,
		})
	}
}

// postMessageResponse is the JSON response for POST /message.
type postMessageResponse struct {
	Cursor int `json:"cursor"`
}

// shutdownResponse is the JSON response for POST /shutdown.
type shutdownResponse struct {
	Status string `json:"status"`
}

// statusResponse is the JSON response for GET /status.
type statusResponse struct {
	Running bool `json:"running"`
}

// subscribeResponse is the JSON response for GET /subscribe.
type subscribeResponse struct {
	Messages []chat.Message `json:"messages"`
	Cursor   int            `json:"cursor"`
}

// waitForResponseResponse is the JSON response for GET /wait-for-response.
type waitForResponseResponse struct {
	Text   string `json:"text"`
	From   string `json:"from"`
	To     string `json:"to"`
	Cursor int    `json:"cursor"`
}

// writeJSON writes value as JSON to w with Content-Type application/json.
// All callers pass concrete typed response structs; encoding never fails for these types.
func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data) //nolint:errchkjson // callers always pass safe typed structs
}

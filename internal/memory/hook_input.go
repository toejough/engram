package memory

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
)

// HookInput represents the JSON payload Claude Code sends to hooks via stdin.
type HookInput struct {
	SessionID      string          `json:"session_id"`
	TranscriptPath string          `json:"transcript_path"`
	Cwd            string          `json:"cwd"`
	PermissionMode string          `json:"permission_mode"`
	HookEventName  string          `json:"hook_event_name"`
	Prompt         string          `json:"prompt"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
}

// ExtractToolQuery returns a human-readable query string from the tool input,
// suitable for use as a memory search query.
func (h *HookInput) ExtractToolQuery() string {
	if h == nil {
		return ""
	}

	var fields map[string]json.RawMessage
	if len(h.ToolInput) > 0 {
		_ = json.Unmarshal(h.ToolInput, &fields)
	}

	getString := func(key string) string {
		raw, ok := fields[key]
		if !ok {
			return ""
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return s
		}
		return ""
	}

	switch h.ToolName {
	case "Bash":
		if desc := getString("description"); desc != "" {
			return desc
		}
		if cmd := getString("command"); cmd != "" {
			return cmd
		}
		return h.ToolName
	case "Grep", "Glob":
		if p := getString("pattern"); p != "" {
			return p
		}
		return h.ToolName
	case "Read", "Write", "Edit":
		if fp := getString("file_path"); fp != "" {
			return fp
		}
		return h.ToolName
	case "WebSearch":
		if q := getString("query"); q != "" {
			return q
		}
		return h.ToolName
	case "WebFetch":
		if p := getString("prompt"); p != "" {
			return p
		}
		return h.ToolName
	case "Task":
		desc := getString("description")
		prompt := getString("prompt")
		var parts []string
		if desc != "" {
			parts = append(parts, desc)
		}
		if prompt != "" {
			parts = append(parts, prompt)
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
		return h.ToolName
	default:
		if h.ToolName != "" {
			return h.ToolName
		}
		return ""
	}
}

// ParseHookInput reads JSON from r and returns a HookInput.
// Returns nil, nil if the reader is empty (no data available).
func ParseHookInput(r io.Reader) (*HookInput, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var hi HookInput
	if err := json.Unmarshal(trimmed, &hi); err != nil {
		return nil, err
	}
	return &hi, nil
}

// DeriveProjectName returns filepath.Base(cwd), or "" if cwd is empty.
func DeriveProjectName(cwd string) string {
	if cwd == "" {
		return ""
	}
	return filepath.Base(cwd)
}

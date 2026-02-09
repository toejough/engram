package memory

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
)

// HookInput represents the JSON payload Claude Code sends to hooks via stdin.
type HookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	PermissionMode string `json:"permission_mode"`
	HookEventName  string `json:"hook_event_name"`
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

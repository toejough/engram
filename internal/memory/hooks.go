package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// InstallHooksOpts contains options for installing Claude Code hooks.
type InstallHooksOpts struct {
	// SettingsPath is the path to Claude Code settings.json
	SettingsPath string
}

// RunFilterPipelineOpts contains options for the filter pipeline.
type RunFilterPipelineOpts struct {
	DB           *sql.DB
	Extractor    LLMExtractor
	QueryResults []QueryResult
	QueryText    string
	HookEvent    string
	SessionID    string
}

// ShowHooksOpts contains options for showing hook configuration.
type ShowHooksOpts struct {
	// SettingsPath is the path to Claude Code settings.json
	SettingsPath string
}

// InstallHooks installs projctl memory hooks into Claude Code settings.json.
// It creates the file if it doesn't exist, merges with existing settings,
// and replaces any existing projctl memory hooks.
func InstallHooks(opts InstallHooksOpts) error {
	// Define the hooks to install (new format: { hooks: [...] })
	stopHook := hookEntry{
		Hooks: []hookCommand{
			{
				Type:    "command",
				Command: "projctl memory extract-session",
			},
			{
				Type:    "command",
				Command: "projctl memory score-session",
			},
			{
				Type:    "command",
				Command: "projctl memory hooks check-claudemd --max-lines=260",
			},
		},
	}

	preCompactHook := hookEntry{
		Hooks: []hookCommand{
			{
				Type:    "command",
				Command: "projctl memory extract-session",
			},
			{
				Type:    "command",
				Command: "projctl memory score-session",
			},
		},
	}

	sessionStartQueryHook := hookEntry{
		Hooks: []hookCommand{{
			Type:    "command",
			Command: "projctl memory query --primacy --stdin-project --min-confidence=30 --max-tokens=1000 -n 10 \"recent important learnings\"",
		}},
	}

	sessionStartScoreHook := hookEntry{
		Matcher: "clear",
		Hooks: []hookCommand{{
			Type:    "command",
			Command: "projctl memory score-session",
		}},
	}

	userPromptSubmitHook := hookEntry{
		Hooks: []hookCommand{{
			Type:    "command",
			Command: "projctl memory query --primacy --stdin-prompt --min-confidence=30 --max-tokens=2000 -n 10",
		}},
	}

	preToolUseHook := hookEntry{
		Hooks: []hookCommand{{
			Type:    "command",
			Command: "projctl memory query --stdin-tool --min-confidence=50 --max-tokens=1000 -n 5",
		}},
	}

	postToolUseHook := hookEntry{
		Matcher: "Bash",
		Hooks: []hookCommand{{
			Type:    "command",
			Command: "projctl memory hooks check-embedding",
		}},
	}

	teammateIdleHook := hookEntry{
		Hooks: []hookCommand{{
			Type:    "command",
			Command: "projctl memory hooks check-skill",
		}},
	}

	// Read existing settings or create new
	var settings map[string]any

	data, err := os.ReadFile(opts.SettingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new settings
			settings = make(map[string]any)
		} else {
			return fmt.Errorf("failed to read settings file: %w", err)
		}
	} else {
		// Parse existing settings
		err := json.Unmarshal(data, &settings)
		if err != nil {
			return fmt.Errorf("failed to parse settings file: %w", err)
		}

		if settings == nil {
			settings = make(map[string]any)
		}
	}

	// Ensure hooks section exists
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		hooks = make(map[string]any)
		settings["hooks"] = hooks
	}

	// Install/replace hooks
	hooks["Stop"] = []hookEntry{stopHook}
	hooks["PreCompact"] = []hookEntry{preCompactHook}
	hooks["SessionStart"] = []hookEntry{sessionStartQueryHook, sessionStartScoreHook}
	hooks["UserPromptSubmit"] = []hookEntry{userPromptSubmitHook}
	hooks["PreToolUse"] = []hookEntry{preToolUseHook}
	hooks["PostToolUse"] = []hookEntry{postToolUseHook}
	hooks["TeammateIdle"] = []hookEntry{teammateIdleHook}

	// Write updated settings
	data, err = json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(opts.SettingsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

// RunFilterPipeline executes the filter→log→synthesize→format pipeline.
// Returns formatted output string (empty string = no relevant memories).
func RunFilterPipeline(ctx context.Context, opts RunFilterPipelineOpts) string {
	if len(opts.QueryResults) == 0 {
		return ""
	}

	filterResults, err := opts.Extractor.Filter(ctx, opts.QueryText, opts.QueryResults)
	if err != nil {
		// Should not happen (Filter degrades internally), but handle anyway
		return FormatMarkdown(FormatMarkdownOpts{Results: opts.QueryResults})
	}

	// Calculate context precision (fraction of relevant results)
	relevantCount := 0

	for _, fr := range filterResults {
		if fr.Relevant {
			relevantCount++
		}
	}

	droppedCount := len(filterResults) - relevantCount
	fmt.Fprintf(os.Stderr, "[memory:filter] %d candidates → %d kept, %d dropped\n",
		len(filterResults), relevantCount, droppedCount)

	var contextPrecision float64
	if len(filterResults) > 0 {
		contextPrecision = float64(relevantCount) / float64(len(filterResults))
	}

	// Log ALL surfacing events (best-effort)
	for _, fr := range filterResults {
		event := SurfacingEvent{
			MemoryID:         fr.MemoryID,
			QueryText:        opts.QueryText,
			HookEvent:        opts.HookEvent,
			Timestamp:        time.Now(),
			SessionID:        opts.SessionID,
			E5Similarity:     findE5Score(fr.MemoryID, opts.QueryResults),
			ContextPrecision: contextPrecision,
		}
		if fr.RelevanceScore != -1.0 {
			event.HaikuRelevant = &fr.Relevant
			event.HaikuTag = fr.Tag
			event.HaikuRelevanceScore = &fr.RelevanceScore
			event.ShouldSynthesize = &fr.ShouldSynthesize
		}

		_, _ = LogSurfacingEvent(opts.DB, event)
	}

	// Collect synthesis candidates (relevant AND should_synthesize)
	var synthCandidates []string

	for _, fr := range filterResults {
		if fr.Relevant && fr.ShouldSynthesize {
			synthCandidates = append(synthCandidates, fr.Content)
		}
	}

	var synthesizedText string
	if len(synthCandidates) >= 2 {
		synthesizedText, _ = opts.Extractor.Synthesize(ctx, synthCandidates)
	}

	return FormatFiltered(filterResults, synthesizedText)
}

// ShowHooks returns the current hook configuration as formatted JSON.
// Returns "{}" if no hooks are configured or the file doesn't exist.
func ShowHooks(opts ShowHooksOpts) (string, error) {
	// Read settings file
	data, err := os.ReadFile(opts.SettingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "{}", nil
		}

		return "", fmt.Errorf("failed to read settings file: %w", err)
	}

	// Parse settings
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return "", fmt.Errorf("failed to parse settings file: %w", err)
	}

	// Extract hooks
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok || len(hooks) == 0 {
		return "{}", nil
	}

	// Format as JSON
	output, err := json.MarshalIndent(hooks, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal hooks: %w", err)
	}

	return string(output), nil
}

// hookCommand represents a single hook command.
type hookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// hookEntry represents a hook entry in the new format with matcher and hooks array.
type hookEntry struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []hookCommand `json:"hooks"`
}

// findE5Score looks up the Score for a memory ID in the query results.
func findE5Score(memoryID int64, queryResults []QueryResult) float64 {
	for _, qr := range queryResults {
		if qr.ID == memoryID {
			return qr.Score
		}
	}

	return 0.0
}

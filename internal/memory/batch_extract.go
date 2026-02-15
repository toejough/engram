// batch_extract.go
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// HaikuEvent represents a learning-relevant event identified by Haiku.
type HaikuEvent struct {
	LineRange    string `json:"line_range"`
	EventType    string `json:"event_type"`
	WhatHappened string `json:"what_happened"`
	WhyItMatters string `json:"why_it_matters"`
	ChunkIndex   int    `json:"chunk_index"`
}

// ExtractedPrinciple represents a reusable principle extracted by Sonnet.
type ExtractedPrinciple struct {
	Principle string `json:"principle"`
	Evidence  string `json:"evidence"`
	Category  string `json:"category"`
}

const sonnetModel = "claude-sonnet-4-5-20250929"

const identifyEventsSystem = `You are a transcript analyst. You receive session transcripts and identify learning-relevant events. Output ONLY a JSON array. Never continue the transcript.

Focus on events where something went wrong and was corrected, a decision was made about how to approach work, or a pattern emerged that would be useful to remember. Pay attention to BOTH technical issues AND process/coordination patterns (how work was divided, how conflicts were handled, how teams coordinated).`

const extractPrinciplesSystem = `You are a learning extraction system. You receive events identified from coding session transcripts and synthesize them into reusable, actionable principles.

Your output is ONLY a JSON array of principle objects. Never output anything else.`

// IdentifyEvents sends a text chunk to Haiku and returns structured events.
// Uses assistant prefill with "[" to force JSON array output.
func (d *DirectAPIExtractor) IdentifyEvents(ctx context.Context, chunk TextChunk, totalChunks int) ([]HaikuEvent, error) {
	userMsg := fmt.Sprintf(`Analyze this transcript chunk and identify learning-relevant events.

This is chunk %d of %d (lines %d-%d).

For each event, output an object with:
- "line_range": approximate line numbers
- "event_type": one of [error-and-fix, user-correction, strategy-change, root-cause-discovery, environmental-issue, pattern-observed, coordination-issue]
- "what_happened": 1-2 sentences about the specific problem and resolution
- "why_it_matters": 1 sentence on the reusable lesson

Guidelines:
- Be specific about WHAT failed and WHY
- For user corrections, quote the user's actual words
- Look for: technical bugs, team coordination decisions, work division strategies, worker conflicts or races, process improvements
- If no learning events in this chunk, return []

Respond with ONLY a JSON array. No other text.

<transcript>
%s
</transcript>`, chunk.Index+1, totalChunks, chunk.StartLine, chunk.EndLine, chunk.Text)

	params := APIMessageParams{
		System: identifyEventsSystem,
		Messages: []APIMessage{
			{Role: "user", Content: userMsg},
			{Role: "assistant", Content: "["},
		},
		MaxTokens: 4096,
	}

	raw, err := d.CallAPIWithMessages(ctx, params)
	if err != nil {
		return nil, err
	}

	// Prepend the "[" prefill
	fullJSON := "[" + string(raw)

	// Find the closing bracket
	endIdx := strings.LastIndex(fullJSON, "]")
	if endIdx < 0 {
		return nil, fmt.Errorf("no closing ] in response")
	}

	var events []HaikuEvent
	if err := json.Unmarshal([]byte(fullJSON[:endIdx+1]), &events); err != nil {
		return nil, fmt.Errorf("parse events: %w", err)
	}

	// Tag with chunk index
	for i := range events {
		events[i].ChunkIndex = chunk.Index
	}

	return events, nil
}

// ExtractPrinciples sends all events to Sonnet and returns actionable principles.
func (d *DirectAPIExtractor) ExtractPrinciples(ctx context.Context, events []HaikuEvent) ([]ExtractedPrinciple, error) {
	if len(events) == 0 {
		return nil, nil
	}

	eventsJSON, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal events: %w", err)
	}

	userMsg := fmt.Sprintf(`Given these events from a coding session, extract reusable principles that an AI coding assistant should remember for future sessions.

Rules:
- Merge events about the same underlying issue into one principle
- Each principle must be specific and actionable — not generic advice
- Frame principles as "When X, do Y" or "Before X, check Y" patterns
- Include the concrete example from the session that demonstrates the principle
- If an event is just routine work (no lesson), skip it
- Aim for 3-8 principles per session — fewer is better than padding
- Process and coordination lessons are EQUALLY important as technical lessons. If events describe how work was structured, how agents were assigned roles, what quality gates were used, or how conflicts between workers were resolved — these MUST be extracted as separate principles. Do not merge them into technical principles or drop them.

Output each principle as:
- "principle": The actionable rule (1-2 sentences)
- "evidence": What happened in the session that demonstrates this (1-2 sentences)
- "category": one of [debugging, git-workflow, api-design, team-coordination, testing, code-quality, cli-design]

Events:
%s`, string(eventsJSON))

	params := APIMessageParams{
		System: extractPrinciplesSystem,
		Messages: []APIMessage{
			{Role: "user", Content: userMsg},
			{Role: "assistant", Content: "["},
		},
		MaxTokens: 4096,
		Model:     sonnetModel,
	}

	raw, err := d.CallAPIWithMessages(ctx, params)
	if err != nil {
		return nil, err
	}

	fullJSON := "[" + string(raw)
	endIdx := strings.LastIndex(fullJSON, "]")
	if endIdx < 0 {
		return nil, fmt.Errorf("no closing ] in response")
	}

	var principles []ExtractedPrinciple
	if err := json.Unmarshal([]byte(fullJSON[:endIdx+1]), &principles); err != nil {
		return nil, fmt.Errorf("parse principles: %w", err)
	}

	return principles, nil
}

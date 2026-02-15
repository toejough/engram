// batch_extract.go
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
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

// BatchExtractResult holds the full pipeline output.
type BatchExtractResult struct {
	StrippedSize  int
	ChunkCount    int
	ChunkFailures int
	Events        []HaikuEvent
	Principles    []ExtractedPrinciple
}

const sonnetModel = "claude-sonnet-4-5-20250929"
const defaultChunkSize = 25000 // 25KB
const maxParallelChunks = 4

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

// BatchExtractSession runs the full extraction pipeline on a session transcript.
func BatchExtractSession(ctx context.Context, sessionPath string, ext *DirectAPIExtractor) (*BatchExtractResult, error) {
	// Stage 1: Strip
	stripped, err := StripSession(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("strip session: %w", err)
	}

	if len(stripped) == 0 {
		return &BatchExtractResult{}, nil
	}

	// Stage 2: Chunk
	chunks := ChunkText(stripped, defaultChunkSize)

	// Stage 3: Haiku event identification (parallel)
	type chunkResult struct {
		events []HaikuEvent
		err    error
		index  int
	}

	results := make(chan chunkResult, len(chunks))
	sem := make(chan struct{}, maxParallelChunks)
	var wg sync.WaitGroup

	for _, chunk := range chunks {
		wg.Add(1)
		go func(c TextChunk) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			events, err := ext.IdentifyEvents(ctx, c, len(chunks))
			results <- chunkResult{events: events, err: err, index: c.Index}
		}(chunk)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allEvents []HaikuEvent
	failures := 0
	for r := range results {
		if r.err != nil {
			failures++
			continue
		}
		allEvents = append(allEvents, r.events...)
	}

	// Sort events by chunk index then line range for stable ordering
	sortEvents(allEvents)

	// Stage 4: Sonnet principle extraction
	principles, err := ext.ExtractPrinciples(ctx, allEvents)
	if err != nil {
		return nil, fmt.Errorf("extract principles: %w", err)
	}

	return &BatchExtractResult{
		StrippedSize:  len(stripped),
		ChunkCount:    len(chunks),
		ChunkFailures: failures,
		Events:        allEvents,
		Principles:    principles,
	}, nil
}

func sortEvents(events []HaikuEvent) {
	// Simple sort by chunk index, then line range string
	for i := 1; i < len(events); i++ {
		for j := i; j > 0; j-- {
			if events[j].ChunkIndex < events[j-1].ChunkIndex ||
				(events[j].ChunkIndex == events[j-1].ChunkIndex && events[j].LineRange < events[j-1].LineRange) {
				events[j], events[j-1] = events[j-1], events[j]
			} else {
				break
			}
		}
	}
}

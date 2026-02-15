// batch_extract.go
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	EndOffset     int64
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
- Each principle MUST start with one of these words: Always, Never, Prefer, Avoid, Use, Ensure, Check, Verify, Validate, Test, When, Before, After, If, Do not, Follow, Apply, Set, Configure, Add, Remove, Create, Build, Run, Fix, Update, Replace, Delete, Execute, Install, Deploy, Compile, Implement
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

	// Scale max tokens with event count — more events means richer evidence sections.
	// Base 4096 for ≤20 events, +100 per event beyond that, capped at 16384.
	maxTokens := 4096
	if len(events) > 20 {
		maxTokens += (len(events) - 20) * 100
	}
	if maxTokens > 16384 {
		maxTokens = 16384
	}

	params := APIMessageParams{
		System: extractPrinciplesSystem,
		Messages: []APIMessage{
			{Role: "user", Content: userMsg},
			{Role: "assistant", Content: "["},
		},
		MaxTokens: maxTokens,
		Model:     sonnetModel,
	}

	raw, err := d.CallAPIWithMessages(ctx, params)
	if err != nil {
		return nil, err
	}

	principles, err := parsePrinciplesJSON("[" + string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse principles: %w", err)
	}

	return principles, nil
}

// parsePrinciplesJSON parses a JSON array of principles, recovering partial results
// from truncated output (e.g., when MaxTokens is hit mid-response).
func parsePrinciplesJSON(fullJSON string) ([]ExtractedPrinciple, error) {
	// Try clean parse first — response has a proper closing ]
	endIdx := strings.LastIndex(fullJSON, "]")
	if endIdx >= 0 {
		var principles []ExtractedPrinciple
		if err := json.Unmarshal([]byte(fullJSON[:endIdx+1]), &principles); err == nil {
			return principles, nil
		}
	}

	// Truncated response — find the last complete JSON object by looking for "}".
	// Walk backward to find a position where the array parses successfully.
	lastBrace := strings.LastIndex(fullJSON, "}")
	for lastBrace > 0 {
		candidate := strings.TrimRight(fullJSON[:lastBrace+1], ", \n\t") + "]"
		var principles []ExtractedPrinciple
		if err := json.Unmarshal([]byte(candidate), &principles); err == nil {
			return principles, nil
		}
		// Try the previous }
		lastBrace = strings.LastIndex(fullJSON[:lastBrace], "}")
	}

	return nil, fmt.Errorf("unexpected end of JSON input")
}

// BatchExtractSession runs the full extraction pipeline on a session transcript.
// If startOffset > 0, only processes content from that byte position onward.
// If progress is non-nil, one-line status updates are written to it.
func BatchExtractSession(ctx context.Context, sessionPath string, ext *DirectAPIExtractor, startOffset int64, progress io.Writer) (*BatchExtractResult, error) {
	logf := func(format string, args ...any) {
		if progress != nil {
			fmt.Fprintf(progress, "  "+format+"\n", args...)
		}
	}

	// Stage 1: Strip
	logf("stripping %s (offset %d)...", sessionPath, startOffset)
	stripped, endOffset, err := StripSession(sessionPath, startOffset)
	if err != nil {
		return nil, fmt.Errorf("strip session: %w", err)
	}

	if len(stripped) == 0 {
		logf("nothing new to process")
		return &BatchExtractResult{EndOffset: endOffset}, nil
	}

	logf("stripped to %d bytes", len(stripped))

	// Stage 2: Chunk
	chunks := ChunkText(stripped, defaultChunkSize)

	// Stage 3: Haiku event identification (parallel)
	logf("identifying events with haiku (%d chunks)...", len(chunks))

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

	logf("identified %d events in %d chunks", len(allEvents), len(chunks))

	if len(allEvents) == 0 {
		return &BatchExtractResult{
			StrippedSize:  len(stripped),
			ChunkCount:    len(chunks),
			ChunkFailures: failures,
			EndOffset:     endOffset,
		}, nil
	}

	// Stage 4: Sonnet principle extraction
	logf("extracting principles with sonnet...")
	principles, err := ext.ExtractPrinciples(ctx, allEvents)
	if err != nil {
		return nil, fmt.Errorf("extract principles: %w", err)
	}

	logf("extracted %d principles", len(principles))

	return &BatchExtractResult{
		StrippedSize:  len(stripped),
		ChunkCount:    len(chunks),
		ChunkFailures: failures,
		Events:        allEvents,
		Principles:    principles,
		EndOffset:     endOffset,
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

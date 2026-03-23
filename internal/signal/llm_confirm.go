package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"engram/internal/memory"
)

// LLMConfirmer asks an LLM to confirm that memory clusters share a principle.
type LLMConfirmer struct {
	llmCaller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
}

// NewLLMConfirmer creates an LLMConfirmer with the given LLM caller.
func NewLLMConfirmer(
	caller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error),
) *LLMConfirmer {
	return &LLMConfirmer{llmCaller: caller}
}

// ConfirmClusters builds a prompt from the query and candidates, asks the LLM
// to group them by shared principle and exclude contradictions, then parses the response.
func (c *LLMConfirmer) ConfirmClusters(
	ctx context.Context,
	query *memory.MemoryRecord,
	candidates []ScoredCandidate,
) ([]ConfirmedCluster, error) {
	userPrompt := buildConfirmPrompt(query, candidates)

	response, err := c.llmCaller(ctx, confirmerModel, confirmSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("confirming clusters: %w", err)
	}

	parsed, parseErr := parseConfirmResponse(response)
	if parseErr != nil {
		return nil, fmt.Errorf("confirming clusters: parsing response: %w", parseErr)
	}

	allMemories := buildAllMemories(query, candidates)
	contradictions := buildContradictionSet(parsed.Contradictions)

	return buildConfirmedClusters(parsed.Clusters, allMemories, contradictions), nil
}

// LLMExtractor creates a generalized memory record from a confirmed cluster via LLM.
type LLMExtractor struct {
	llmCaller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
}

// NewLLMExtractor creates an LLMExtractor with the given LLM caller.
func NewLLMExtractor(
	caller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error),
) *LLMExtractor {
	return &LLMExtractor{llmCaller: caller}
}

// ExtractPrinciple builds a prompt from the cluster members and asks the LLM
// to synthesize a generalized memory record.
func (e *LLMExtractor) ExtractPrinciple(
	ctx context.Context,
	cluster ConfirmedCluster,
) (*memory.MemoryRecord, error) {
	userPrompt := buildExtractPrompt(cluster)

	response, err := e.llmCaller(ctx, confirmerModel, extractSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("extracting principle: %w", err)
	}

	record, parseErr := parseExtractResponse(response)
	if parseErr != nil {
		return nil, fmt.Errorf("extracting principle: parsing response: %w", parseErr)
	}

	return record, nil
}

// unexported constants.
const (
	confirmSystemPrompt = "You are a memory consolidation assistant. " +
		"Given a query memory and candidate memories, group candidates that share " +
		"a genuine underlying principle with the query. Exclude contradictions. " +
		"Return ONLY JSON with this structure:\n" +
		`{"clusters":[{"member_indices":[0,1],"principle":"..."}],"contradictions":[2]}` + "\n" +
		"member_indices are zero-based indices into the combined list " +
		"(query=0, candidates=1..N). " +
		"contradictions is an array of indices that contradict the cluster principle."
	confirmerModel      = "claude-haiku-4-5-20251001"
	extractSystemPrompt = "You are a memory synthesis assistant. " +
		"Given a cluster of related memories, synthesize a single generalized memory. " +
		"Return ONLY JSON with these fields:\n" +
		`{"title":"...","principle":"...","anti_pattern":"...",` +
		`"content":"...","keywords":[...],"concepts":[...],"generalizability":4}` + "\n" +
		"generalizability is 1-5: 1=session-specific, 5=universal."
)

//nolint:tagliatelle // LLM prompt specifies snake_case JSON field names.
type confirmClusterJSON struct {
	MemberIndices []int  `json:"member_indices"`
	Principle     string `json:"principle"`
}

type confirmResponseJSON struct {
	Clusters       []confirmClusterJSON `json:"clusters"`
	Contradictions []int                `json:"contradictions"`
}

//nolint:tagliatelle // LLM prompt specifies snake_case JSON field names.
type extractResponseJSON struct {
	Title            string   `json:"title"`
	Principle        string   `json:"principle"`
	AntiPattern      string   `json:"anti_pattern"`
	Content          string   `json:"content"`
	Keywords         []string `json:"keywords"`
	Concepts         []string `json:"concepts"`
	Generalizability int      `json:"generalizability"`
}

// buildAllMemories creates an ordered list: query at index 0, then candidates.
func buildAllMemories(
	query *memory.MemoryRecord,
	candidates []ScoredCandidate,
) []*memory.MemoryRecord {
	allMemories := make([]*memory.MemoryRecord, 0, len(candidates)+1)
	allMemories = append(allMemories, query)

	for idx := range candidates {
		allMemories = append(allMemories, candidates[idx].Memory)
	}

	return allMemories
}

func buildConfirmPrompt(query *memory.MemoryRecord, candidates []ScoredCandidate) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Query memory:\n")
	fmt.Fprintf(&sb, "  Title: %s\n", query.Title)
	fmt.Fprintf(&sb, "  Principle: %s\n", query.Principle)
	fmt.Fprintf(&sb, "  Keywords: %s\n\n", strings.Join(query.Keywords, ", "))

	fmt.Fprintf(&sb, "Candidates:\n")

	for idx, candidate := range candidates {
		mem := candidate.Memory
		fmt.Fprintf(&sb, "[%d] Title: %s\n", idx+1, mem.Title)
		fmt.Fprintf(&sb, "    Principle: %s\n", mem.Principle)
		fmt.Fprintf(&sb, "    Keywords: %s\n", strings.Join(mem.Keywords, ", "))
		fmt.Fprintf(&sb, "    Score: %.2f\n\n", candidate.Score)
	}

	return sb.String()
}

// buildConfirmedClusters maps raw cluster JSON into ConfirmedCluster values,
// excluding contradicted members and out-of-bounds indices.
func buildConfirmedClusters(
	rawClusters []confirmClusterJSON,
	allMemories []*memory.MemoryRecord,
	contradictions map[int]struct{},
) []ConfirmedCluster {
	clusters := make([]ConfirmedCluster, 0, len(rawClusters))

	for _, rawCluster := range rawClusters {
		members := filterClusterMembers(rawCluster.MemberIndices, allMemories, contradictions)

		if len(members) > 0 {
			clusters = append(clusters, ConfirmedCluster{
				Members:   members,
				Principle: rawCluster.Principle,
			})
		}
	}

	return clusters
}

// buildContradictionSet creates a set of indices flagged as contradictions.
func buildContradictionSet(indices []int) map[int]struct{} {
	contradictions := make(map[int]struct{}, len(indices))
	for _, idx := range indices {
		contradictions[idx] = struct{}{}
	}

	return contradictions
}

func buildExtractPrompt(cluster ConfirmedCluster) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Cluster principle: %s\n\n", cluster.Principle)
	fmt.Fprintf(&sb, "Members:\n")

	for idx, mem := range cluster.Members {
		fmt.Fprintf(&sb, "[%d] Title: %s\n", idx, mem.Title)
		fmt.Fprintf(&sb, "    Content: %s\n", mem.Content)
		fmt.Fprintf(&sb, "    Principle: %s\n", mem.Principle)
		fmt.Fprintf(&sb, "    Keywords: %s\n\n", strings.Join(mem.Keywords, ", "))
	}

	return sb.String()
}

// filterClusterMembers returns non-contradicted, in-bounds members for a cluster.
func filterClusterMembers(
	indices []int,
	allMemories []*memory.MemoryRecord,
	contradictions map[int]struct{},
) []*memory.MemoryRecord {
	members := make([]*memory.MemoryRecord, 0, len(indices))

	for _, idx := range indices {
		if _, excluded := contradictions[idx]; excluded {
			continue
		}

		if idx >= 0 && idx < len(allMemories) {
			members = append(members, allMemories[idx])
		}
	}

	return members
}

func parseConfirmResponse(response string) (confirmResponseJSON, error) {
	cleaned := stripMarkdownFenceLocal(response)

	var parsed confirmResponseJSON

	err := json.Unmarshal([]byte(cleaned), &parsed)
	if err != nil {
		return confirmResponseJSON{}, fmt.Errorf("parsing confirm JSON: %w", err)
	}

	return parsed, nil
}

func parseExtractResponse(response string) (*memory.MemoryRecord, error) {
	cleaned := stripMarkdownFenceLocal(response)

	var parsed extractResponseJSON

	err := json.Unmarshal([]byte(cleaned), &parsed)
	if err != nil {
		return nil, fmt.Errorf("parsing extract JSON: %w", err)
	}

	return &memory.MemoryRecord{
		Title:            parsed.Title,
		Principle:        parsed.Principle,
		AntiPattern:      parsed.AntiPattern,
		Content:          parsed.Content,
		Keywords:         parsed.Keywords,
		Concepts:         parsed.Concepts,
		Generalizability: parsed.Generalizability,
	}, nil
}

// stripMarkdownFenceLocal removes markdown code fences that LLMs sometimes wrap around JSON.
func stripMarkdownFenceLocal(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}

	firstNewline := strings.Index(trimmed, "\n")
	if firstNewline < 0 {
		return trimmed
	}

	trimmed = trimmed[firstNewline+1:]

	if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
		trimmed = trimmed[:idx]
	}

	return strings.TrimSpace(trimmed)
}

package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SkillTestResult captures the outcome of a single test run (RED or GREEN).
type SkillTestResult struct {
	Scenario           TestScenario
	WithSkill          bool   // true for GREEN phase, false for RED phase
	Response           string // LLM response text
	SuccessCriteriaMet bool   // true if success pattern was found
	FailureCriteriaMet bool   // true if failure pattern was found
	Error              string // error message if test failed to run
}

// EvaluateTestResults determines if a skill candidate passes the RED/GREEN protocol.
// Pass criteria: >=N-1 RED failures AND >=N-1 GREEN successes
// Returns (pass bool, reasoning string explaining the decision)
func EvaluateTestResults(redResults, greenResults []SkillTestResult) (pass bool, reasoning string) {
	if len(redResults) == 0 || len(greenResults) == 0 {
		return false, "cannot evaluate: missing RED or GREEN results"
	}

	N := len(redResults)

	minRequired := max(N-1, 0)

	// Count RED failures (should fail without skill)
	redFailures := 0

	for _, result := range redResults {
		if result.FailureCriteriaMet && !result.SuccessCriteriaMet {
			redFailures++
		}
	}

	// Count GREEN successes (should succeed with skill)
	greenSuccesses := 0

	for _, result := range greenResults {
		if result.SuccessCriteriaMet && !result.FailureCriteriaMet {
			greenSuccesses++
		}
	}

	// Both criteria must be met
	redPass := redFailures >= minRequired
	greenPass := greenSuccesses >= minRequired

	if redPass && greenPass {
		return true, fmt.Sprintf("RED: %d/%d failures (>=%d required), GREEN: %d/%d successes (>=%d required) → PASS",
			redFailures, N, minRequired, greenSuccesses, len(greenResults), minRequired)
	}

	var reasons []string
	if !redPass {
		reasons = append(reasons, fmt.Sprintf("RED failed: only %d/%d failures (>=%d required)", redFailures, N, minRequired))
	}

	if !greenPass {
		reasons = append(reasons, fmt.Sprintf("GREEN failed: only %d/%d successes (>=%d required)", greenSuccesses, len(greenResults), minRequired))
	}

	return false, "FAIL: " + joinReasons(reasons)
}

// TestSkillCandidate runs RED/GREEN testing protocol for a skill candidate.
// RED phase: N runs WITHOUT skill in system prompt (should fail/show failure mode)
// GREEN phase: N runs WITH skill content injected into system prompt (should succeed)
// Uses direct Anthropic API with temperature=0.0 and Haiku model.
func TestSkillCandidate(ctx context.Context, scenario TestScenario, runs int, apiKey string) (redResults, greenResults []SkillTestResult, err error) {
	if runs <= 0 {
		return nil, nil, fmt.Errorf("runs must be > 0, got %d", runs)
	}

	// Create API client for testing
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// RED phase: test WITHOUT skill
	redResults = make([]SkillTestResult, 0, runs)
	for range runs {
		result := runSingleTest(ctx, client, apiKey, scenario, false)
		redResults = append(redResults, result)
	}

	// GREEN phase: test WITH skill
	greenResults = make([]SkillTestResult, 0, runs)
	for range runs {
		result := runSingleTest(ctx, client, apiKey, scenario, true)
		greenResults = append(greenResults, result)
	}

	return redResults, greenResults, nil
}

// callAnthropicAPI sends a request to the Anthropic API and returns the response text.
func callAnthropicAPI(ctx context.Context, client *http.Client, apiKey, systemPrompt, userPrompt string) (string, error) {
	reqBody := map[string]any{
		"model":       "claude-haiku-4-5-20251001",
		"max_tokens":  512,
		"temperature": 0.0,
		"system":      systemPrompt,
		"messages": []map[string]any{
			{"role": "user", "content": userPrompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Anthropic-Version", "2023-06-01")
	req.Header.Set("Anthropic-Beta", "oauth-2025-04-20")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}

	if resp == nil {
		return "", errors.New("API request returned nil response")
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return "", errors.New("empty response content")
	}

	return apiResp.Content[0].Text, nil
}

func joinReasons(reasons []string) string {
	result := ""

	var resultSb167 strings.Builder

	for i, r := range reasons {
		if i > 0 {
			resultSb167.WriteString("; ")
		}

		resultSb167.WriteString(r)
	}

	result += resultSb167.String()

	return result
}

// runSingleTest executes a single test run with or without the skill content.
func runSingleTest(ctx context.Context, client *http.Client, apiKey string, scenario TestScenario, withSkill bool) SkillTestResult {
	result := SkillTestResult{
		Scenario:  scenario,
		WithSkill: withSkill,
	}

	// Build system prompt
	systemPrompt := "You are a helpful assistant."
	if withSkill {
		systemPrompt = "You are a helpful assistant.\n\n" + scenario.SkillContent
	}

	// Build test prompt from scenario description
	userPrompt := scenario.Description

	// Call API
	response, err := callAnthropicAPI(ctx, client, apiKey, systemPrompt, userPrompt)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Response = response

	// Check criteria
	responseLower := strings.ToLower(response)
	successLower := strings.ToLower(scenario.SuccessCriteria)
	failureLower := strings.ToLower(scenario.FailureCriteria)

	result.SuccessCriteriaMet = strings.Contains(responseLower, successLower)
	result.FailureCriteriaMet = strings.Contains(responseLower, failureLower)

	return result
}

//go:build targ

package eval

import (
	"encoding/json"
	"fmt"
)

// Layer1 projects the cost/efficiency metrics.
func (rs ResultSummary) Layer1() Layer1Metrics {
	return Layer1Metrics{
		DurationMS:  rs.DurationMS,
		Turns:       rs.NumTurns,
		TotalTokens: rs.Usage.InputTokens + rs.Usage.OutputTokens,
		CostUSD:     rs.TotalCost,
	}
}

// ParseResult decodes the headless result JSON.
func ParseResult(raw []byte) (ResultSummary, error) {
	var rs ResultSummary
	if err := json.Unmarshal(raw, &rs); err != nil {
		return ResultSummary{}, fmt.Errorf("parsing result json: %w", err)
	}

	if rs.Type != "result" {
		return ResultSummary{}, fmt.Errorf("unexpected result type %q", rs.Type)
	}

	return rs, nil
}

// marshalResult serializes a RunResult to JSON for JSONL storage.
func marshalResult(r RunResult) ([]byte, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("marshaling run result: %w", err)
	}

	return data, nil
}

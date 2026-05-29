//go:build targ

package eval

import (
	"bufio"
	"bytes"
	"encoding/json"
)

// ParseBashCommands returns, in order, every Bash tool_use command string
// in a Claude Code session JSONL. Malformed lines are skipped.
func ParseBashCommands(raw []byte) []string {
	var cmds []string
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, initialScanBuf), maxScanLine)
	for scanner.Scan() {
		var line transcriptLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type != "assistant" {
			continue
		}
		for _, block := range line.Message.Content {
			if block.Type == "tool_use" && block.Name == "Bash" && block.Input.Command != "" {
				cmds = append(cmds, block.Input.Command)
			}
		}
	}
	return cmds
}

// unexported constants.
const (
	initialScanBuf = 64 * 1024
	maxScanLine    = 16 * 1024 * 1024
)

type transcriptLine struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type  string `json:"type"`
			Name  string `json:"name"`
			Input struct {
				Command string `json:"command"`
			} `json:"input"`
		} `json:"content"`
	} `json:"message"`
}

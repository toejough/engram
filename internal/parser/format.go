package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// tomlPattern matches TOML key = value pattern
var tomlPattern = regexp.MustCompile(`^\s*\w+\s*=`)

// DetectFormat determines whether content uses YAML or TOML format.
// Returns "yaml" for YAML frontmatter, "toml" for TOML key=value pairs.
func DetectFormat(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", fmt.Errorf("empty content")
	}

	// Check for YAML frontmatter delimiter
	if strings.HasPrefix(trimmed, "---") {
		return "yaml", nil
	}

	// Check for TOML key=value pattern
	lines := strings.Split(trimmed, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if tomlPattern.MatchString(line) {
			return "toml", nil
		}
		break
	}

	return "", fmt.Errorf("unrecognized format")
}

package parser

import (
	"errors"
	"regexp"
	"strings"
)

// DetectFormat determines whether content uses YAML or TOML format.
// Returns "yaml" for YAML frontmatter, "toml" for TOML key=value pairs.
func DetectFormat(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", errors.New("empty content")
	}

	// Check for YAML frontmatter delimiter
	if strings.HasPrefix(trimmed, "---") {
		return "yaml", nil
	}

	// Check for TOML key=value pattern
	lines := strings.SplitSeq(trimmed, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if tomlPattern.MatchString(line) {
			return "toml", nil
		}

		break
	}

	return "", errors.New("unrecognized format")
}

// unexported variables.
var (
	tomlPattern = regexp.MustCompile(`^\s*\w+\s*=`)
)

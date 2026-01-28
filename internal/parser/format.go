package parser

import "fmt"

// DetectFormat determines whether content uses YAML or TOML format.
// Returns "yaml" for YAML frontmatter, "toml" for TOML key=value pairs.
func DetectFormat(content string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

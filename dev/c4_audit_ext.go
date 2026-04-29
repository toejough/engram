//go:build targ

package dev

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// unexported constants.
const (
	catalogCellsForSource     = 7
	propertyEnforcedCellIndex = 4
	propertyMinCellCount      = 8
	propertyTestedCellIndex   = 5
)

// unexported variables.
var (
	inlineYAMLArrayRe = regexp.MustCompile(`\[(.*?)\]`)
	markdownLinkRe    = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	propertyLedgerRe  = regexp.MustCompile(`(?m)^##\s+Property Ledger\s*$`)
)

func brokenLinksInCell(cell, dir string, lineNum int) []Finding {
	matches := markdownLinkRe.FindAllStringSubmatch(cell, -1)
	findings := []Finding{}
	for _, match := range matches {
		target := stripFragment(match[2])
		if target == "" {
			continue
		}
		resolved := filepath.Join(dir, target)
		if _, err := os.Stat(resolved); err == nil {
			continue
		}
		findings = append(findings, Finding{
			ID:     "property_link_unresolved",
			Line:   lineNum,
			Detail: fmt.Sprintf("link target %q does not resolve from %s", match[2], dir),
		})
	}
	return findings
}

// checkChildren validates each entry of the front-matter `children` array
// against the level-derived expected prefix (`c{level+1}-`). L4 is special-
// cased: any non-empty children entry is invalid.
func checkChildren(matter frontMatter) []Finding {
	if !matter.hasChildren || !matter.hasLevel || matter.level < 1 {
		return nil
	}
	if matter.level >= 4 {
		findings := []Finding{}
		for _, child := range matter.children {
			findings = append(findings, Finding{
				ID:   "child_prefix_invalid",
				Line: matter.childrenLine,
				Detail: fmt.Sprintf(
					"L4 cannot have children, but children includes %q", child),
			})
		}
		return findings
	}
	expectedPrefix := fmt.Sprintf("c%d-", matter.level+1)
	findings := []Finding{}
	for _, child := range matter.children {
		if !strings.HasPrefix(child, expectedPrefix) {
			findings = append(findings, Finding{
				ID:   "child_prefix_invalid",
				Line: matter.childrenLine,
				Detail: fmt.Sprintf(
					"child %q must start with %q (level %d expects %s* children)",
					child, expectedPrefix, matter.level, expectedPrefix),
			})
		}
	}
	return findings
}

// checkPropertyLinks scans an L4 ledger's Property Ledger table and emits a
// finding for any markdown link in the Enforced-at or Tested-at columns whose
// path does not resolve. Links resolved relative to the markdown file's
// directory; the **⚠ UNTESTED** marker (no link present in Tested-at) is not
// a finding by design.
func checkPropertyLinks(matter frontMatter, raw []byte, mdPath string) []Finding {
	if matter.level != 4 {
		return nil
	}
	text := string(raw)
	loc := propertyLedgerRe.FindStringIndex(text)
	if loc == nil {
		return nil
	}
	tail := text[loc[1]:]
	startLine := 1 + strings.Count(text[:loc[0]], "\n")
	findings := []Finding{}
	for offset, line := range strings.Split(tail, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			break
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		findings = append(findings, propertyLinkFindings(line, mdPath, startLine+offset+1)...)
	}
	return findings
}

// checkSourcePaths walks an L1/L2/L3 markdown's Element Catalog table,
// locates the "Source" cell (column 5 in a 7-piece split — empty | id | name
// | type | responsibility | source | empty), extracts the markdown link
// target if present, and emits `source_path_unresolved` for any path that
// does not stat. Free-text Source values render plain (no link) and are
// ignored by this check; only path-like values are rendered as links by the
// L1/L2/L3 builders, so this finder catches dead repo-relative paths only.
//
// Returns nil for L4 files; L4 has its own property-link audit.
func checkSourcePaths(matter frontMatter, raw []byte, mdPath string) []Finding {
	if matter.level < 1 || matter.level > 3 {
		return nil
	}
	text := string(raw)
	loc := catalogHeaderRe.FindStringIndex(text)
	if loc == nil {
		return nil
	}
	tail := text[loc[1]:]
	startLine := 1 + strings.Count(text[:loc[0]], "\n")
	findings := []Finding{}
	for offset, line := range strings.Split(tail, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			break
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		finding := sourcePathFindingForRow(line, mdPath, startLine+offset+1)
		if finding != nil {
			findings = append(findings, *finding)
		}
	}
	return findings
}

// parseInlineYAMLArray parses a YAML inline array like
//
//	["foo.md", "bar.md"]
//
// into its individual entries. It tolerates single quotes, double quotes, and
// unquoted entries. Returns nil for malformed input or an empty array.
func parseInlineYAMLArray(value string) []string {
	matches := inlineYAMLArrayRe.FindStringSubmatch(value)
	if len(matches) < 2 {
		return nil
	}
	inner := strings.TrimSpace(matches[1])
	if inner == "" {
		return nil
	}
	parts := strings.Split(inner, ",")
	out := make([]string, 0, len(parts))
	for _, raw := range parts {
		entry := strings.TrimSpace(raw)
		entry = strings.Trim(entry, `"'`)
		if entry == "" {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func propertyLinkFindings(line, mdPath string, lineNum int) []Finding {
	cells := strings.Split(line, "|")
	if len(cells) < propertyMinCellCount {
		return nil
	}
	if !strings.Contains(cells[1], "P") || strings.Contains(cells[1], "---") {
		return nil
	}
	dir := filepath.Dir(mdPath)
	findings := []Finding{}
	for _, cellIdx := range []int{propertyEnforcedCellIndex, propertyTestedCellIndex} {
		findings = append(findings, brokenLinksInCell(cells[cellIdx], dir, lineNum)...)
	}
	return findings
}

// sourcePathFindingForRow returns a finding when the catalog row's source
// cell contains a markdown link to a non-existent path. Returns nil for
// header rows, separator rows, rows without enough cells, or rows whose
// source cell has no link (free-text values are not validated).
func sourcePathFindingForRow(line, mdPath string, lineNum int) *Finding {
	cells := strings.Split(line, "|")
	if len(cells) < catalogCellsForSource {
		return nil
	}
	sourceCell := cells[5]
	match := markdownLinkRe.FindStringSubmatch(sourceCell)
	if len(match) < 3 {
		return nil
	}
	target := strings.TrimSpace(match[2])
	if target == "" {
		return nil
	}
	resolved := filepath.Join(filepath.Dir(mdPath), target)
	if _, err := os.Stat(resolved); err == nil {
		return nil
	}
	return &Finding{
		ID:   "source_path_unresolved",
		Line: lineNum,
		Detail: fmt.Sprintf(
			"source path %q resolves to %q but does not exist", target, resolved),
	}
}

func stripFragment(target string) string {
	if hashIndex := strings.Index(target, "#"); hashIndex >= 0 {
		return target[:hashIndex]
	}
	return target
}

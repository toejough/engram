//go:build targ

package dev

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// unexported constants.
const (
	catalogCellsForCodePointer = 7
)

// unexported variables.
var (
	inlineYAMLArrayRe = regexp.MustCompile(`\[(.*?)\]`)
	markdownLinkRe    = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	propertyLedgerRe  = regexp.MustCompile(`(?m)^##\s+Property Ledger\s*$`)
)

const (
	propertyEnforcedCellIndex = 4
	propertyTestedCellIndex   = 5
	propertyMinCellCount      = 8
)

// catalogIDName is one (id, name) pair extracted from the audited markdown's
// Element Catalog along with the source line for finding placement.
type catalogIDName struct {
	id   string
	name string
	line int
}

// catalogPairFromRow parses one Element Catalog markdown row into a
// catalogIDName. Returns false for header, separator, and non-data rows.
func catalogPairFromRow(line string, lineNum int) (catalogIDName, bool) {
	cells := strings.Split(line, "|")
	if len(cells) < 4 {
		return catalogIDName{}, false
	}
	idCell := strings.TrimSpace(cells[1])
	nameCell := strings.TrimSpace(cells[2])
	idCell = anchorInCellRe.ReplaceAllString(idCell, "")
	idCell = strings.TrimSpace(idCell)
	if !mermaidIDPrefix.MatchString(idCell) {
		return catalogIDName{}, false
	}
	id := mermaidIDPrefix.FindString(idCell)
	if id == "" || nameCell == "" {
		return catalogIDName{}, false
	}
	return catalogIDName{id: id, name: nameCell, line: lineNum}, true
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

// checkCodePointers walks an L3 markdown's Element Catalog table, locates the
// "Code Pointer" cell (column 5 in a 7-piece split — empty | id | name | type
// | responsibility | code_pointer | empty), extracts the markdown link target,
// and emits `code_pointer_unresolved` for any path that does not stat.
//
// Returns nil for non-L3 files; only L3 catalogs use a code-pointer column.
func checkCodePointers(matter frontMatter, raw []byte, mdPath string) []Finding {
	if matter.level != 3 {
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
		finding := codePointerFindingForRow(line, mdPath, startLine+offset+1)
		if finding != nil {
			findings = append(findings, *finding)
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

func stripFragment(target string) string {
	if hashIndex := strings.Index(target, "#"); hashIndex >= 0 {
		return target[:hashIndex]
	}
	return target
}

// checkRegistryCrossCheck derives the registry from the audited markdown's
// parent directory and verifies that every (id, name) pair in the markdown's
// Element Catalog matches the registry. Emits:
//
//   - `registry_orphan` (single, line 1) when the dir has spec JSONs but no
//     JSON matches this markdown's basename;
//   - `id_name_drift` per-row when a markdown id has no registry entry, or the
//     registry's name(s) for that id don't include the markdown's name.
//
// When the dir has zero `c*.json` files, the audit is skipped (no findings).
func checkRegistryCrossCheck(ctx context.Context, raw []byte, mdPath string) []Finding {
	dir := filepath.Dir(mdPath)
	files, records, err := scanRegistryDir(ctx, dir)
	if err != nil || len(files) == 0 {
		return nil
	}
	base := strings.TrimSuffix(filepath.Base(mdPath), ".md")
	matchingJSON := base + ".json"
	if !containsString(files, matchingJSON) {
		return []Finding{{
			ID:   "registry_orphan",
			Line: 1,
			Detail: fmt.Sprintf(
				"no matching %s among scanned specs %v", matchingJSON, files),
		}}
	}
	view := deriveRegistry(dir, files, records)
	elementByID := map[string]RegistryElement{}
	for _, element := range view.Elements {
		elementByID[element.ID] = element
	}
	pairs := parseCatalogIDNames(raw)
	findings := []Finding{}
	for _, pair := range pairs {
		entry, ok := elementByID[pair.id]
		if !ok {
			findings = append(findings, Finding{
				ID:   "id_name_drift",
				Line: pair.line,
				Detail: fmt.Sprintf(
					"markdown declares %s but no JSON in %s does", pair.id, dir),
			})
			continue
		}
		if !containsString(entry.Names, pair.name) {
			findings = append(findings, Finding{
				ID:   "id_name_drift",
				Line: pair.line,
				Detail: fmt.Sprintf(
					"markdown declares %s = %q but registry has %v",
					pair.id, pair.name, entry.Names),
			})
		}
	}
	return findings
}

// codePointerFindingForRow returns a finding when the catalog row's code-
// pointer cell contains a markdown link to a non-existent path. Returns nil
// for header rows, separator rows, rows without enough cells, or rows whose
// code-pointer cell has no link.
func codePointerFindingForRow(line, mdPath string, lineNum int) *Finding {
	cells := strings.Split(line, "|")
	if len(cells) < catalogCellsForCodePointer {
		return nil
	}
	codeCell := cells[5]
	match := markdownLinkRe.FindStringSubmatch(codeCell)
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
		ID:   "code_pointer_unresolved",
		Line: lineNum,
		Detail: fmt.Sprintf(
			"code pointer %q resolves to %q but does not exist", target, resolved),
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// parseCatalogIDNames extracts (id, name) pairs from the Element Catalog. The
// id comes from the first cell after the leading empty cell (stripping any
// `<a id="..."></a>` anchor); the name is column 2.
func parseCatalogIDNames(raw []byte) []catalogIDName {
	text := string(raw)
	loc := catalogHeaderRe.FindStringIndex(text)
	if loc == nil {
		return nil
	}
	tail := text[loc[1]:]
	startLine := 1 + strings.Count(text[:loc[0]], "\n")
	pairs := []catalogIDName{}
	for offset, line := range strings.Split(tail, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			break
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		pair, ok := catalogPairFromRow(line, startLine+offset+1)
		if !ok {
			continue
		}
		pairs = append(pairs, pair)
	}
	return pairs
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

package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/vaultgraph"
)

// CountArgs holds parsed flags for `engram count`. The command aggregates over
// note frontmatter attributes and exposes wikilink in-degree; it is read-only
// and deliberately off the recall/similarity path.
type CountArgs struct {
	Vault       string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"` //nolint:lll // unbreakable env+desc struct-tag string
	GroupBy     string   `targ:"flag,name=group-by,desc=frontmatter attribute to count note membership over"`
	Filter      []string `targ:"flag,name=filter,desc=attr=value predicate (repeatable and AND-ed; scalar equality or list-contains)"` //nolint:lll // single unbreakable struct-tag string
	BacklinksOf string   `targ:"flag,name=backlinks-of,desc=note basename: print wikilink in-degree and sorted linkers"`
}

// CountDeps holds injected dependencies for RunCount. The command is read-only.
// ListMD/ReadFile drive the frontmatter group-by/filter path; Scan drives the
// wikilink backlinks path (a struct-of-funcs does not satisfy
// vaultgraph.VaultFS, so ScanVault cannot be fed CountDeps directly).
type CountDeps struct {
	ListMD   func(vault string) ([]string, error)
	ReadFile func(path string) ([]byte, error)
	Scan     func(vault string) ([]vaultgraph.Note, error)
}

// RunCount dispatches to the backlinks-of path when --backlinks-of is set, and
// otherwise to the frontmatter group-by/filter path. The two modes are mutually
// exclusive — they measure different things (frontmatter membership vs wikilink
// in-degree, which legitimately diverge by the count of non-member linkers such
// as MOC/index pages), so requesting both is an error. args.Vault must already
// be resolved by the caller via resolveVault. Read-only; no writes.
func RunCount(args CountArgs, deps CountDeps, stdout io.Writer) error {
	if args.GroupBy != "" && args.BacklinksOf != "" {
		return errCountBothModes
	}

	if args.BacklinksOf != "" {
		return runCountBacklinks(args, deps, stdout)
	}

	return runCountGroupBy(args, deps, stdout)
}

// unexported constants.
const (
	// filterDelimiter separates the attribute and value of a --filter predicate.
	filterDelimiter = "="
)

// unexported variables.
var (
	errCountBadFilter = errors.New("count: --filter must be attr=value")
	errCountBothModes = errors.New("count: --group-by and --backlinks-of are mutually exclusive")
	errCountNoMode    = errors.New("count: specify --group-by <attr> or --backlinks-of <basename>")
)

// countFilter is a parsed --filter predicate: a note matches when the value is
// one of the (deduped) values of attr — scalar equality or list membership.
type countFilter struct {
	attr  string
	value string
}

// attrValues returns the distinct string values of attr in the frontmatter map
// and whether the key is present. A scalar yields one value; a list yields one
// per distinct element (dedup within-note); an empty/null value yields none.
func attrValues(attrs map[string]any, attr string) (values []string, present bool) {
	raw, present := attrs[attr]
	if !present {
		return nil, false
	}

	list, isList := raw.([]any)
	if !isList {
		if raw == nil {
			return nil, true
		}

		return []string{fmt.Sprint(raw)}, true
	}

	seen := make(map[string]struct{}, len(list))
	values = make([]string, 0, len(list))

	for _, elem := range list {
		text := fmt.Sprint(elem)
		if _, dup := seen[text]; dup {
			continue
		}

		seen[text] = struct{}{}
		values = append(values, text)
	}

	return values, true
}

// matchesAllFilters reports whether attrs satisfies every filter (AND).
func matchesAllFilters(attrs map[string]any, filters []countFilter) bool {
	for _, filter := range filters {
		values, _ := attrValues(attrs, filter.attr)
		if !slices.Contains(values, filter.value) {
			return false
		}
	}

	return true
}

// newCountDeps wires RunCount from the injected CLI capabilities — pure
// composition over EdgeFS (#700).
func newCountDeps(deps Deps) CountDeps {
	fsys := newVaultFS(deps.FS)

	return CountDeps{
		ListMD:   fsys.ListMD,
		ReadFile: fsys.ReadFile,
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(fsys, vault)
		},
	}
}

// parseCountFilters parses each raw "attr=value" predicate. Returns
// errCountBadFilter when an entry has no '=' separator.
func parseCountFilters(raw []string) ([]countFilter, error) {
	filters := make([]countFilter, 0, len(raw))

	for _, entry := range raw {
		attr, value, ok := strings.Cut(entry, filterDelimiter)
		if !ok {
			return nil, fmt.Errorf("%w: %q", errCountBadFilter, entry)
		}

		filters = append(filters, countFilter{attr: attr, value: value})
	}

	return filters, nil
}

// readNoteAttrs reads a note's frontmatter into an attribute map. Returns
// (nil, false) when the note is unreadable, has no frontmatter, or fails to
// parse — those notes are not countable and are skipped by callers.
func readNoteAttrs(readFile func(string) ([]byte, error), path string) (map[string]any, bool) {
	raw, readErr := readFile(path)
	if readErr != nil {
		return nil, false
	}

	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return nil, false
	}

	attrs := map[string]any{}

	unmarshalErr := yaml.Unmarshal(frontmatter, &attrs)
	if unmarshalErr != nil {
		return nil, false
	}

	return attrs, true
}

// renderCountGroupBy writes the group-by report: "value\tcount" lines sorted by
// count desc then value asc, an "(attr absent): N" line when any in-set note
// lacks the attr, and a "total: N" line. An empty in-set (total == 0) prints
// nothing.
func renderCountGroupBy(stdout io.Writer, attr string, counts map[string]int, absent, total int) {
	if total == 0 {
		return
	}

	values := make([]string, 0, len(counts))
	for value := range counts {
		values = append(values, value)
	}

	sort.Slice(values, func(i, j int) bool {
		if counts[values[i]] != counts[values[j]] {
			return counts[values[i]] > counts[values[j]]
		}

		return values[i] < values[j]
	})

	for _, value := range values {
		_, _ = fmt.Fprintf(stdout, "%s\t%d\n", value, counts[value])
	}

	if absent > 0 {
		_, _ = fmt.Fprintf(stdout, "(%s absent): %d\n", attr, absent)
	}

	_, _ = fmt.Fprintf(stdout, "total: %d\n", total)
}

// runCountBacklinks prints the wikilink in-degree of the target basename plus
// its linkers, sorted ascending.
func runCountBacklinks(args CountArgs, deps CountDeps, stdout io.Writer) error {
	notes, scanErr := deps.Scan(args.Vault)
	if scanErr != nil {
		return fmt.Errorf("count: scan: %w", scanErr)
	}

	graph := vaultgraph.BuildGraph(notes)
	target := args.BacklinksOf

	linkers := make([]string, 0, len(graph.Incoming[target]))
	for source := range graph.Incoming[target] {
		linkers = append(linkers, source)
	}

	sort.Strings(linkers)

	_, _ = fmt.Fprintf(stdout, "in-degree: %d\n", graph.InDegree(target))

	for _, linker := range linkers {
		_, _ = fmt.Fprintln(stdout, linker)
	}

	return nil
}

// runCountGroupBy counts distinct frontmatter membership per attribute value
// over the notes that pass every filter, then renders the report.
func runCountGroupBy(args CountArgs, deps CountDeps, stdout io.Writer) error {
	if args.GroupBy == "" {
		return errCountNoMode
	}

	filters, parseErr := parseCountFilters(args.Filter)
	if parseErr != nil {
		return parseErr
	}

	names, listErr := deps.ListMD(args.Vault)
	if listErr != nil {
		return fmt.Errorf("count: listing vault: %w", listErr)
	}

	counts := make(map[string]int)

	var absent, total int

	for _, name := range names {
		attrs, ok := readNoteAttrs(deps.ReadFile, filepath.Join(args.Vault, name))
		if !ok || !matchesAllFilters(attrs, filters) {
			continue
		}

		total++

		values, present := attrValues(attrs, args.GroupBy)
		if !present {
			absent++

			continue
		}

		for _, value := range values {
			counts[value]++
		}
	}

	renderCountGroupBy(stdout, args.GroupBy, counts, absent, total)

	return nil
}

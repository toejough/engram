package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// CheckArgs holds parsed flags for `engram check`.
type CheckArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
}

// CheckDeps holds injected dependencies for RunCheck. The check is read-only.
// ReadNote is optional: a nil ReadNote skips the content-level checks (e.g.
// situation-presence) and runs only the graph-level checks. ReadSidecar is
// likewise optional: a nil ReadSidecar skips the S1 sidecar-schema invariant.
type CheckDeps struct {
	Scan        func(vault string) ([]vaultgraph.Note, error)
	ReadNote    func(path string) ([]byte, error)
	ReadSidecar func(path string) ([]byte, error)
}

// RunCheck runs the vault-invariant checks read-only over the vault, writes a
// per-invariant PASS/FAIL report to stdout, and returns errCheckFailed when any
// FAIL-class invariant is violated.
func RunCheck(_ context.Context, args CheckArgs, deps CheckDeps, stdout io.Writer) error {
	notes, err := deps.Scan(args.VaultPath)
	if err != nil {
		return fmt.Errorf("check: scan: %w", err)
	}

	failed := false

	failed = checkGraphResolution(notes, stdout) || failed

	if deps.ReadNote != nil {
		failed = checkSituationPresence(notes, deps.ReadNote, args.VaultPath, stdout) || failed
	}

	if deps.ReadSidecar != nil {
		failed = checkSidecars(notes, deps.ReadSidecar, args.VaultPath, stdout) || failed
	}

	if failed {
		return errCheckFailed
	}

	return nil
}

// unexported constants.
const (
	maxCheckExamples = 10
)

// unexported variables.
var (
	// bareLuhmannIDRE matches a bare Luhmann id like 105 or 1a2.
	bareLuhmannIDRE = regexp.MustCompile(`^[0-9]+[a-z0-9]*$`)
	errCheckFailed  = errors.New("check: FAIL-class invariant violations found")
)

// checkGraphResolution verifies G0: every authored wikilink resolves to a note.
// Returns true if the invariant is violated.
func checkGraphResolution(notes []vaultgraph.Note, stdout io.Writer) bool {
	leadingIDs := make(map[string]struct{}, len(notes))
	for _, note := range notes {
		id, _, _ := strings.Cut(note.Basename, ".")
		leadingIDs[id] = struct{}{}
	}

	resolverBroken := make([]vaultgraph.UnresolvedLink, 0)
	dangling := make([]vaultgraph.UnresolvedLink, 0)

	for _, link := range vaultgraph.UnresolvedTargets(notes) {
		_, idExists := leadingIDs[link.Target]
		if idExists && bareLuhmannIDRE.MatchString(link.Target) {
			resolverBroken = append(resolverBroken, link) // could resolve, wrong form → G0
		} else {
			dangling = append(dangling, link) // target is no note at all → G3
		}
	}

	if len(resolverBroken) == 0 {
		_, _ = fmt.Fprintln(stdout, "PASS  G0 graph-resolution: every authored link resolves by form")
	} else {
		_, _ = fmt.Fprintf(stdout, "FAIL  G0 graph-resolution: %d bare-id links that should resolve\n", len(resolverBroken))
		printLinkExamples(stdout, resolverBroken)
	}

	if len(dangling) > 0 {
		_, _ = fmt.Fprintf(stdout, "WARN  G3 dangling: %d authored links target no note\n", len(dangling))
		printLinkExamples(stdout, dangling)
	}

	return len(resolverBroken) > 0
}

// checkSidecars verifies S1: every situation-bearing note's sidecar parses
// under the current schema. An old-schema or malformed sidecar FAILs (a
// re-embed is required); a missing sidecar WARNs (embed status covers it).
// Returns true if the FAIL-class invariant is violated.
func checkSidecars(
	notes []vaultgraph.Note,
	readSidecar func(path string) ([]byte, error),
	vault string,
	stdout io.Writer,
) bool {
	stale := make([]string, 0)
	missing := 0

	for _, note := range notes {
		if note.IsMOC {
			continue
		}

		notePath := note.Basename + ".md"

		scBytes, err := readSidecar(filepath.Join(vault, embed.SidecarPath(notePath)))
		if err != nil {
			missing++

			continue
		}

		_, parseErr := embed.UnmarshalSidecar(scBytes)
		if parseErr != nil {
			stale = append(stale, note.Basename)
		}
	}

	if missing > 0 {
		_, _ = fmt.Fprintf(stdout,
			"WARN  S1 sidecar-schema: %d note(s) missing a sidecar (run `engram embed apply`)\n", missing)
	}

	if len(stale) > 0 {
		_, _ = fmt.Fprintf(stdout,
			"FAIL  S1 sidecar-schema: %d sidecar(s) on an old/invalid schema (run `engram embed apply --force`)\n",
			len(stale))
		printNoteExamples(stdout, stale)

		return true
	}

	_, _ = fmt.Fprintln(stdout, "PASS  S1 sidecar-schema: every sidecar parses under the current schema")

	return false
}

// checkSituationPresence verifies M5: every fact/feedback note names a
// non-empty situation — the field recall matches on and the embedding is
// shaped around. MOC notes are skipped (not situation-bearing). A note whose
// file is unreadable or whose frontmatter does not parse is skipped here;
// those failures surface in other checks. Returns true if violated.
func checkSituationPresence(
	notes []vaultgraph.Note,
	readNote func(path string) ([]byte, error),
	vault string,
	stdout io.Writer,
) bool {
	missing := make([]string, 0)

	for _, note := range notes {
		if note.IsMOC {
			continue
		}

		raw, err := readNote(filepath.Join(vault, note.Basename+".md"))
		if err != nil {
			continue
		}

		frontmatter, ok := splitFrontmatter(raw)
		if !ok {
			continue
		}

		noteType, situation := frontmatterTypeAndSituation(frontmatter)
		if isSituationBearing(noteType) && strings.TrimSpace(situation) == "" {
			missing = append(missing, note.Basename)
		}
	}

	if len(missing) > 0 {
		_, _ = fmt.Fprintf(stdout, "FAIL  M5 situation-presence: %d note(s) missing a situation\n", len(missing))
		printNoteExamples(stdout, missing)

		return true
	}

	_, _ = fmt.Fprintln(stdout, "PASS  M5 situation-presence: every fact/feedback names a situation")

	return false
}

// frontmatterTypeAndSituation extracts the top-level type and situation from a
// frontmatter YAML block. Returns empty strings when the block does not parse.
func frontmatterTypeAndSituation(frontmatter []byte) (noteType, situation string) {
	var probe struct {
		Type      string `yaml:"type"`
		Situation string `yaml:"situation"`
	}

	err := yaml.Unmarshal(frontmatter, &probe)
	if err != nil {
		return "", ""
	}

	return probe.Type, probe.Situation
}

// isSituationBearing reports whether a note type is required to carry a
// situation: facts and feedback are; MOCs and bare notes are not.
func isSituationBearing(noteType string) bool {
	return noteType == typeFact || noteType == typeFeedback
}

// newOsCheckDeps wires RunCheck to the real filesystem vault scanner.
func newOsCheckDeps() CheckDeps {
	fsys := &osVaultFS{}

	return CheckDeps{
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(fsys, vault)
		},
		ReadNote:    fsys.ReadFile,
		ReadSidecar: fsys.ReadFile,
	}
}

// printLinkExamples writes up to maxCheckExamples offending links to stdout.
func printLinkExamples(stdout io.Writer, links []vaultgraph.UnresolvedLink) {
	for i, link := range links {
		if i >= maxCheckExamples {
			_, _ = fmt.Fprintf(stdout, "        … and %d more\n", len(links)-maxCheckExamples)

			break
		}

		_, _ = fmt.Fprintf(stdout, "        %s → [[%s]]\n", link.Source, link.Target)
	}
}

// printNoteExamples writes up to maxCheckExamples offending note basenames.
func printNoteExamples(stdout io.Writer, names []string) {
	for i, name := range names {
		if i >= maxCheckExamples {
			_, _ = fmt.Fprintf(stdout, "        … and %d more\n", len(names)-maxCheckExamples)

			break
		}

		_, _ = fmt.Fprintf(stdout, "        %s\n", name)
	}
}

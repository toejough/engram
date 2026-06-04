package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/toejough/engram/internal/vaultgraph"
)

// CheckArgs holds parsed flags for `engram check`.
type CheckArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
}

// CheckDeps holds injected dependencies for RunCheck. The check is read-only.
type CheckDeps struct {
	Scan func(vault string) ([]vaultgraph.Note, error)
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

// newOsCheckDeps wires RunCheck to the real filesystem vault scanner.
func newOsCheckDeps() CheckDeps {
	return CheckDeps{
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(&osVaultFS{}, vault)
		},
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

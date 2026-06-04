package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

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
	errCheckFailed = errors.New("check: FAIL-class invariant violations found")
)

// checkGraphResolution verifies G0: every authored wikilink resolves to a note.
// Returns true if the invariant is violated.
func checkGraphResolution(notes []vaultgraph.Note, stdout io.Writer) bool {
	unresolved := vaultgraph.UnresolvedTargets(notes)

	if len(unresolved) == 0 {
		_, _ = fmt.Fprintln(stdout, "PASS  G0 graph-resolution: all authored links resolve")

		return false
	}

	_, _ = fmt.Fprintf(stdout, "FAIL  G0 graph-resolution: %d authored links resolve to no note\n", len(unresolved))

	for i, link := range unresolved {
		if i >= maxCheckExamples {
			_, _ = fmt.Fprintf(stdout, "        … and %d more\n", len(unresolved)-maxCheckExamples)

			break
		}

		_, _ = fmt.Fprintf(stdout, "        %s → [[%s]]\n", link.Source, link.Target)
	}

	return true
}

// newOsCheckDeps wires RunCheck to the real filesystem vault scanner.
func newOsCheckDeps() CheckDeps {
	return CheckDeps{
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(&osVaultFS{}, vault)
		},
	}
}

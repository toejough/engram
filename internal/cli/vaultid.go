package cli

import (
	"errors"
	"fmt"
	"io"
)

// VaultIDArgs holds parsed flags for `engram vault-id` (design D2, r3-4).
type VaultIDArgs struct {
	Vault      string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	Regenerate bool   `targ:"flag,name=regenerate,desc=this vault is a copy: mint a new vault ID (no note changes)"`
	Claim      bool   `targ:"flag,name=claim,desc=this is the same vault moved or re-cloned: rewrite its location record"`
}

// RunVaultID implements `engram vault-id [--regenerate | --claim]`. With no
// flag it prints the ID and the location check result, writing nothing.
func RunVaultID(args VaultIDArgs, deps Deps, stdout io.Writer) error {
	if args.Regenerate && args.Claim {
		return errVaultIDFlagsExclusive
	}

	state := exchangeStateFromDeps(deps)

	switch {
	case args.Regenerate:
		id, regenErr := regenerateVaultID(state, args.Vault)
		if regenErr != nil {
			return regenErr
		}

		_, _ = fmt.Fprintf(stdout, "vault_id: %s (regenerated)\n", id)

		return nil
	case args.Claim:
		id, claimErr := claimVaultLocation(state, args.Vault)
		if claimErr != nil {
			return claimErr
		}

		_, _ = fmt.Fprintf(stdout, "vault_id: %s (location claimed)\n", id)

		return nil
	default:
		return printVaultID(state, args.Vault, stdout)
	}
}

// unexported variables.
var (
	errVaultIDFlagsExclusive = errors.New("vault-id: --regenerate and --claim are mutually exclusive")
)

// printVaultID prints the ID (or "none") and the location check result;
// a failed check also names both remedies.
func printVaultID(state exchangeState, vault string, stdout io.Writer) error {
	id, found, idErr := readVaultID(state, vault)
	if idErr != nil {
		return idErr
	}

	if !found {
		id = "none"
	}

	status, checkErr := checkVaultLocation(state, vault)
	if checkErr != nil {
		return checkErr
	}

	_, _ = fmt.Fprintf(stdout, "vault_id: %s\nlocation: %s\n", id, status)

	if status == locationMissing || status == locationMismatch {
		_, _ = fmt.Fprintln(stdout,
			"remedy: if this vault is a copy, run `engram vault-id --regenerate`; "+
				"if it is the same vault moved or re-cloned, run `engram vault-id --claim`")
	}

	return nil
}

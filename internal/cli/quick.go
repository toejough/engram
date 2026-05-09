// Package cli — quick subcommand: writes a fleeting note file.
package cli

import (
	"errors"
	"fmt"
	"io"
	"regexp"
)

var (
	errSlugEmpty   = errors.New("quick: slug is required")
	errSlugInvalid = errors.New("quick: slug must match [a-z0-9-]+")

	slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// validateSlug returns nil if slug is non-empty kebab-case lowercase.
func validateSlug(slug string) error {
	if slug == "" {
		return errSlugEmpty
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("%w: got %q", errSlugInvalid, slug)
	}
	return nil
}

const envVaultDir = "ENGRAM_VAULT_DIR"

var errVaultUnset = errors.New("quick: vault path is required (--vault flag or ENGRAM_VAULT_DIR env)")

// resolveVault returns the vault path: flag wins, env is fallback, error if neither set.
func resolveVault(flagValue string, getenv func(string) string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if env := getenv(envVaultDir); env != "" {
		return env, nil
	}
	return "", errVaultUnset
}

var (
	errContentBoth    = errors.New("quick: provide --content OR stdin, not both")
	errContentNeither = errors.New("quick: --content flag or stdin required")
)

// resolveContent picks content from flag XOR stdin. Errors on both or neither.
func resolveContent(flagValue string, stdin io.Reader) (string, error) {
	stdinBytes, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("quick: reading stdin: %w", err)
	}
	hasFlag := flagValue != ""
	hasStdin := len(stdinBytes) > 0
	if hasFlag && hasStdin {
		return "", errContentBoth
	}
	if !hasFlag && !hasStdin {
		return "", errContentNeither
	}
	if hasFlag {
		return flagValue, nil
	}
	return string(stdinBytes), nil
}

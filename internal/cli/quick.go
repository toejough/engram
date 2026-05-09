package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"time"
)

// QuickDeps holds injected dependencies for runQuick. All fields required.
type QuickDeps struct {
	Now      func() time.Time
	Stdin    io.Reader
	Getenv   func(string) string
	StatDir  func(string) error
	WriteNew func(path string, data []byte) error // must error with fs.ErrExist if file exists
}

// unexported constants.
const (
	dateFormat     = "2006-01-02"
	envVaultDir    = "ENGRAM_VAULT_DIR"
	fleetingSubdir = "Fleeting"
)

// unexported variables.
var (
	errContentNeither = errors.New("quick: --content flag or stdin required")
	errFileExists     = errors.New("quick: target file already exists")
	errSlugEmpty      = errors.New("quick: slug is required")
	errSlugInvalid    = errors.New("quick: slug must match [a-z0-9-]+")
	errVaultUnset     = errors.New("quick: vault path is required (--vault flag or ENGRAM_VAULT_DIR env)")
	slugPattern       = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// fleetingPath builds the absolute path for a fleeting note file.
func fleetingPath(vault, slug string, when time.Time) string {
	filename := fmt.Sprintf("%s.%s.md", when.Format(dateFormat), slug)

	return filepath.Join(vault, fleetingSubdir, filename)
}

// requireVaultDirs verifies vault root exists, then Fleeting/ subdir exists.
// Two-stage check produces a clear message for whichever level is missing.
func requireVaultDirs(vault string, statDir func(string) error) error {
	err := statDir(vault)
	if err != nil {
		return fmt.Errorf("quick: vault directory not accessible at %s: %w", vault, err)
	}

	fleeting := filepath.Join(vault, fleetingSubdir)

	err = statDir(fleeting)
	if err != nil {
		return fmt.Errorf("quick: vault Fleeting directory not accessible at %s: %w", fleeting, err)
	}

	return nil
}

// resolveContent picks content from flag or stdin. Flag wins without reading stdin.
// Reading stdin only when the flag is empty avoids consuming/blocking on the parent's stdin.
func resolveContent(flagValue string, stdin io.Reader) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	stdinBytes, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("quick: reading stdin: %w", err)
	}

	if len(stdinBytes) == 0 {
		return "", errContentNeither
	}

	return string(stdinBytes), nil
}

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

// runQuick orchestrates the quick subcommand: validates inputs, derives the path, writes the file.
func runQuick(_ context.Context, args QuickArgs, deps QuickDeps, stdout io.Writer) error {
	slugErr := validateSlug(args.Slug)
	if slugErr != nil {
		return slugErr
	}

	vault, err := resolveVault(args.Vault, deps.Getenv)
	if err != nil {
		return err
	}

	dirErr := requireVaultDirs(vault, deps.StatDir)
	if dirErr != nil {
		return dirErr
	}

	content, contentErr := resolveContent(args.Content, deps.Stdin)
	if contentErr != nil {
		return contentErr
	}

	path := fleetingPath(vault, args.Slug, deps.Now())

	writeErr := deps.WriteNew(path, []byte(content))
	if writeErr != nil {
		if errors.Is(writeErr, fs.ErrExist) {
			return fmt.Errorf("%w: %s", errFileExists, path)
		}

		return fmt.Errorf("quick: writing %s: %w", path, writeErr)
	}

	_, _ = fmt.Fprintln(stdout, path)

	return nil
}

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

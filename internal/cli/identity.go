package cli

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/toejough/engram/internal/update"
)

// unexported constants.
const (
	defaultVaultName = "personal"
	envVaultName     = "ENGRAM_VAULT_NAME"
)

// identityStamp carries the repo/user/vault provenance values freshly
// detected for one learn, amend, or backfill write, threaded down to the
// frontmatter override/render step so it stays pure of I/O.
type identityStamp struct {
	Repo  string
	User  string
	Vault string
}

// detectRepo resolves the repo: frontmatter field: the working directory's
// git origin remote URL, falling back to the git root directory's basename
// when no origin remote is configured, empty when the working directory
// isn't inside a git repository at all (or any step fails). Never fails the
// caller — every error path resolves to "".
func detectRepo(
	ctx context.Context,
	getwd func() (string, error),
	commander update.Commander,
) string {
	dir, wdErr := getwd()
	if wdErr != nil {
		return ""
	}

	origin, _, remoteErr := commander.Run(ctx, dir, "git", "remote", "get-url", "origin")
	if remoteErr == nil {
		if url := strings.TrimSpace(string(origin)); url != "" {
			return url
		}
	}

	top, _, rootErr := commander.Run(ctx, dir, "git", "rev-parse", "--show-toplevel")
	if rootErr != nil {
		return ""
	}

	return filepath.Base(strings.TrimSpace(string(top)))
}

// detectUser resolves the user: frontmatter field: git config user.email
// (global config, independent of cwd — dir is deliberately empty), falling
// back to the machine's OS username when git config resolves to nothing.
// Never fails the caller — every error path resolves to "".
func detectUser(
	ctx context.Context,
	commander update.Commander,
	username func() (string, error),
) string {
	email, _, configErr := commander.Run(ctx, "", "git", "config", "user.email")
	if configErr == nil {
		if trimmed := strings.TrimSpace(string(email)); trimmed != "" {
			return trimmed
		}
	}

	if username == nil {
		return ""
	}

	name, userErr := username()
	if userErr != nil {
		return ""
	}

	return name
}

// repoWithProjectFallback prefers a note's existing project: field over a
// freshly detected repo when project is non-empty. Used only by identity
// backfill: a single backfill invocation runs from one working directory but
// may touch notes that originated in different repos, whereas an amend is
// assumed to run from the repo it's actually about, so amend's repo:
// re-stamp always uses fresh detection directly and never calls this.
func repoWithProjectFallback(project, freshRepo string) string {
	if project != "" {
		return project
	}

	return freshRepo
}

// resolveVaultName resolves the vault: frontmatter field: flag -> env
// (ENGRAM_VAULT_NAME) -> default "personal", mirroring resolveVault's
// flag/env/default order for the vault path.
func resolveVaultName(flagValue string, getenv func(string) string) string {
	if flagValue != "" {
		return flagValue
	}

	if env := getenv(envVaultName); env != "" {
		return env
	}

	return defaultVaultName
}

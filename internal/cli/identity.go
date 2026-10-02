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

	// basename and warn are set only on the amend path (amendIdentity), so
	// stampPreservingUser can name the note and log a warning when user
	// detection resolves empty (design D3, #776). Every other construction
	// site (learn, pulldown, serve_learn, identity backfill) leaves them
	// zero and never calls stampPreservingUser — those paths have no prior
	// value to keep (Non-Goals).
	basename string
	warn     func(string, ...any)
}

// stamp overwrites repo, user and vault with the stamp's values. A nil stamp
// (a bookkeeping amend) leaves them as they are.
func (s *identityStamp) stamp(repo, user, vault *string) {
	if s == nil {
		return
	}

	*repo, *user, *vault = s.Repo, s.User, s.Vault
}

// stampPreservingUser overwrites repo and vault the same as stamp. User is
// overwritten with the freshly detected value, UNLESS that detected value is
// empty (both git config user.email and the OS username lookup failed) and
// the note already carries a non-empty user: — then the note's existing
// value is kept instead of being blanked to user: "", and a warning naming
// the stamp's basename is emitted through warn (design D3, #776; a nil stamp
// is a no-op, matching stamp). The user-preserving behavior itself does not
// depend on warn: a nil warn (no basename/warn set — not the amend path)
// still keeps the prior user, it only skips logging the warning.
func (s *identityStamp) stampPreservingUser(repo, user, vault *string) {
	if s == nil {
		return
	}

	priorUser := *user

	s.stamp(repo, user, vault)

	if s.User == "" && priorUser != "" {
		*user = priorUser

		if s.warn != nil {
			s.warn("amend: user detection resolved empty; keeping user: %s on %s", priorUser, s.basename)
		}
	}
}

// detectRepo resolves the repo: frontmatter field: the working directory's
// git origin remote URL, falling back to the git root directory's basename
// when no origin remote is configured, empty when the working directory
// isn't inside a git repository at all (or any step fails). Never fails the
// caller — every error path resolves to "". The same git probes feed the
// skill-source project identity (probeProjectIdentity).
func detectRepo(
	ctx context.Context,
	getwd func() (string, error),
	commander update.Commander,
) string {
	dir, wdErr := getwd()
	if wdErr != nil {
		return ""
	}

	if url, found := gitOriginURL(ctx, commander, dir); found {
		return url
	}

	top, inRepo := gitTopLevel(ctx, commander, dir)
	if !inRepo {
		return ""
	}

	return filepath.Base(top)
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

// firstWriteIdentity is the identity a note's first write stamps (learn, a
// served learn's in-place rewrite, a pull-down): repo:/user: freshly
// detected, vault: as resolved by the caller. When user detection resolves
// empty, user: is left empty, so the writer omits the key (user,omitempty)
// instead of writing user: "", and one warning names the command (#789
// design D6). Backfill or a later re-stamping amend fills it in.
func firstWriteIdentity(ctx context.Context, deps LearnDeps, vaultName, command string) identityStamp {
	stamp := identityStamp{Repo: deps.DetectRepo(ctx), User: deps.DetectUser(ctx), Vault: vaultName}

	if stamp.User == "" && deps.LogWarning != nil {
		deps.LogWarning("%s: user detection resolved empty (no git user.email and no OS username); "+
			"writing the note without user:", command)
	}

	return stamp
}

// gitOriginURL returns dir's `git remote get-url origin`, trimmed; found is
// false when the command fails or prints nothing.
func gitOriginURL(ctx context.Context, commander update.Commander, dir string) (string, bool) {
	url, ok := runGit(ctx, commander, dir, "remote", "get-url", "origin")

	return url, ok && url != ""
}

// gitTopLevel returns dir's `git rev-parse --show-toplevel`, trimmed; ok is
// false when the command fails (dir is not inside a work tree).
func gitTopLevel(ctx context.Context, commander update.Commander, dir string) (string, bool) {
	return runGit(ctx, commander, dir, "rev-parse", "--show-toplevel")
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

// runGit runs `git <args>` in dir and returns its trimmed stdout; ok is false
// when the command failed.
func runGit(ctx context.Context, commander update.Commander, dir string, args ...string) (string, bool) {
	out, _, err := commander.Run(ctx, dir, "git", args...)
	if err != nil {
		return "", false
	}

	return strings.TrimSpace(string(out)), true
}

package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/debuglog"
)

// CommonLearnArgs holds shared flags for learn subcommands.
type CommonLearnArgs struct {
	Slug      string   `targ:"flag,name=slug,desc=kebab-case tag for the filename"`
	Vault     string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	Target    string   `targ:"flag,name=target,desc=Luhmann ID this note relates to (empty for top-level)"`
	Position  string   `targ:"flag,name=position,desc=top|continuation|sibling"`
	Source    string   `targ:"flag,name=source,required,desc=provenance string for the source field (required)"`
	Relations []string `targ:"flag,name=relation,desc=related note as <wikilink-target>|<rationale> (repeatable)"`
	Project   string   `targ:"flag,name=project,desc=kebab-case project slug for cross-project filtering (optional)"`
	Issue     string   `targ:"flag,name=issue,desc=originating issue ID (optional)"`
}

// LearnEpisodeArgs holds parsed flags for the learn episode subcommand.
// Episodes are L1 evidence — the noise-filtered transcript chunk that
// captures what happened during a discrete segment of work. See
// docs/superpowers/research/2026-05-26-l1-episode-fix-spec.md.
type LearnEpisodeArgs struct {
	CommonLearnArgs

	Situation           string   `targ:"flag,name=situation,required,desc=retrieval-shaped topic phrase (required)"`
	BoundaryRationale   string   `targ:"flag,name=boundary-rationale,required,desc=why this chunk's bounds (required)"`
	FromTranscriptRange []string `targ:"flag,name=from-transcript-range,desc=<session>:<start>..<end>"`
	TranscriptText      string   `targ:"flag,name=transcript-text,desc=literal transcript chunk content"`
	Sessions            []string `targ:"flag,name=session,required,desc=provenance.sessions entry (required)"`
	TranscriptRange     string   `targ:"flag,name=transcript-range,required,desc=<start>..<end> RFC3339 (required)"`
}

// LearnFactArgs holds parsed flags for the learn fact subcommand.
type LearnFactArgs struct {
	CommonLearnArgs

	Situation string `targ:"flag,name=situation,desc=context when this applies"`
	Subject   string `targ:"flag,name=subject,desc=subject of the fact"`
	Predicate string `targ:"flag,name=predicate,desc=relationship or verb"`
	Object    string `targ:"flag,name=object,desc=object of the fact"`
}

// LearnFeedbackArgs holds parsed flags for the learn feedback subcommand.
type LearnFeedbackArgs struct {
	CommonLearnArgs

	Situation string `targ:"flag,name=situation,desc=context when this applies"`
	Behavior  string `targ:"flag,name=behavior,desc=observed behavior"`
	Impact    string `targ:"flag,name=impact,desc=impact of the behavior"`
	Action    string `targ:"flag,name=action,desc=recommended action"`
}

// DataDirFromHome returns the standard engram data directory for a given home path.
// It respects $XDG_DATA_HOME if set, otherwise defaults to $HOME/.local/share/engram.
// getenv is injected so callers control environment access (pass os.Getenv in production).
func DataDirFromHome(home string, getenv func(string) string) string {
	if xdg := getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "engram")
	}

	return filepath.Join(home, ".local", "share", "engram")
}

// ProjectSlugFromPath converts a filesystem path to a project slug by replacing
// path separators with dashes, matching the shell convention: echo "$PWD" | tr '/' '-'.
func ProjectSlugFromPath(path string) string {
	return strings.ReplaceAll(path, string(filepath.Separator), "-")
}

// Targets returns all targ targets for the engram CLI. The logger is
// attached to each handler's ctx so downstream code can call debuglog.Log
// without an explicit logger argument.
func Targets(stdout, stderr io.Writer, exit func(int), logger *debuglog.Logger) []any {
	errHandler := newErrHandler(stderr, exit)

	withLog := func(ctx context.Context) context.Context {
		return debuglog.WithLogger(ctx, logger)
	}

	homeOrEmpty := func() string {
		home, _ := os.UserHomeDir()

		return home
	}

	return []any{
		targ.Targ(func(ctx context.Context, a TranscriptArgs) {
			cwd, _ := os.Getwd()
			finder, reader := newTranscriptDeps(cwd)
			errHandler(runTranscript(withLog(ctx), a, finder, reader, stdout))
		}).Name("transcript").Description("Read session transcripts in a date range"),
		targ.Group("learn",
			targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) {
				a.Vault = resolveVault(a.Vault, homeOrEmpty(), os.Getenv)
				errHandler(runLearnFromFeedbackArgs(withLog(ctx), a, stdout))
			}).Name("feedback").Description("Write a feedback note to Permanent/"),
			targ.Targ(func(ctx context.Context, a LearnFactArgs) {
				a.Vault = resolveVault(a.Vault, homeOrEmpty(), os.Getenv)
				errHandler(runLearnFromFactArgs(withLog(ctx), a, stdout))
			}).Name("fact").Description("Write a fact note to Permanent/"),
			targ.Targ(func(ctx context.Context, a LearnEpisodeArgs) {
				a.Vault = resolveVault(a.Vault, homeOrEmpty(), os.Getenv)
				errHandler(runLearnFromEpisodeArgs(withLog(ctx), a, stdout))
			}).Name("episode").Description("Write an episode note to Permanent/"),
		),
		targ.Targ(func(ctx context.Context, a UpdateArgs) {
			errHandler(runUpdate(withLog(ctx), a, stdout))
		}).Name("update").Description("Refresh engram binary and harness skills"),
		targ.Group("embed",
			targ.Targ(func(ctx context.Context, a EmbedApplyArgs) {
				a.VaultPath = resolveVault(a.VaultPath, homeOrEmpty(), os.Getenv)
				errHandler(RunEmbedApply(withLog(ctx), a, newOsEmbedDeps(), stdout))
			}).Name("apply").Description("Embed notes (default: missing only)"),
			targ.Targ(func(ctx context.Context, a EmbedStatusArgs) {
				a.VaultPath = resolveVault(a.VaultPath, homeOrEmpty(), os.Getenv)
				errHandler(RunEmbedStatus(withLog(ctx), a, newOsEmbedDeps(), stdout))
			}).Name("status").Description("Report embedding state counts"),
		),
		targ.Targ(func(ctx context.Context, a QueryArgs) {
			a.VaultPath = resolveVault(a.VaultPath, homeOrEmpty(), os.Getenv)
			errHandler(RunQuery(withLog(ctx), a, newOsQueryDeps(), stdout))
		}).Name("query").Description("Semantic search over the vault (YAML output)"),
	}
}

// newErrHandler returns a function that prints err to stderr and signals
// failure via exit(1). When err is nil the handler is a no-op.
func newErrHandler(stderr io.Writer, exit func(int)) func(error) {
	return func(err error) {
		if err == nil {
			return
		}

		_, _ = fmt.Fprintln(stderr, err)

		exit(1)
	}
}

package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/debuglog"
)

// CommonLearnArgs holds shared flags for learn subcommands.
type CommonLearnArgs struct {
	Slug       string   `targ:"flag,name=slug,desc=kebab-case tag for the filename"`
	Vault      string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"` //nolint:lll // unbreakable env+desc struct-tag string
	Target     string   `targ:"flag,name=target,desc=Luhmann ID this note relates to (empty for top-level)"`
	Position   string   `targ:"flag,name=position,desc=top|continuation|sibling"`
	Source     string   `targ:"flag,name=source,required,desc=provenance string for the source field (required)"`
	Supersedes []string `targ:"flag,name=supersedes,desc=supersession: <note>|<type>|<claim> (updates/narrows/refutes)"`
	Project    string   `targ:"flag,name=project,desc=kebab-case project slug for cross-project filtering (optional)"`
	Issue      string   `targ:"flag,name=issue,desc=originating issue ID (optional)"`
	Tier       string   `targ:"flag,name=tier,desc=tier: L2 (active/default) or L1|L3 (legacy); default derived from type"`

	ChunkSources []string `targ:"flag,name=chunk-source,desc=chunk id (source#anchor) recorded as provenance (repeatable)"`
	Tags         []string `targ:"flag,name=tag,desc=categorical tag: <family> or <family>/<value> (kebab-case; repeatable)"` //nolint:lll // single unbreakable struct-tag string
}

// LearnFactArgs holds parsed flags for the learn fact subcommand.
type LearnFactArgs struct {
	CommonLearnArgs

	Situation string `targ:"flag,name=situation,required,desc=context when this applies (required)"`
	Subject   string `targ:"flag,name=subject,desc=subject of the fact"`
	Predicate string `targ:"flag,name=predicate,desc=relationship or verb"`
	Object    string `targ:"flag,name=object,desc=object of the fact"`
}

// LearnFeedbackArgs holds parsed flags for the learn feedback subcommand.
type LearnFeedbackArgs struct {
	CommonLearnArgs

	Situation string `targ:"flag,name=situation,required,desc=context when this applies (required)"`
	Behavior  string `targ:"flag,name=behavior,desc=observed behavior"`
	Impact    string `targ:"flag,name=impact,desc=impact of the behavior"`
	Action    string `targ:"flag,name=action,desc=recommended action"`
}

// CacheDirFromHome returns the engram model cache directory for a given home path
// and model ID. It respects $XDG_CACHE_HOME if set, otherwise defaults to
// $HOME/.cache/engram/models/<modelID>. getenv is injected so callers control
// environment access (pass os.Getenv in production).
func CacheDirFromHome(home, modelID string, getenv func(string) string) string {
	if xdg := getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "engram", "models", modelID)
	}

	return filepath.Join(home, ".cache", "engram", "models", modelID)
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

// Targets returns all targ targets for the engram CLI, wired from the
// single production capability carrier (#700). The debug logger is
// constructed from deps.DebugLog (nil sink → no-op logger) and attached to
// each handler's ctx so downstream code can call debuglog.Log without an
// explicit logger argument.
func Targets(deps Deps) []any {
	errHandler := newErrHandler(deps.Stderr, deps.Exit)
	logger := debuglog.New(deps.DebugLog, "engram", deps.Now)

	withLog := func(ctx context.Context) context.Context {
		return debuglog.WithLogger(ctx, logger)
	}

	return append(
		coreTargets(deps, withLog, errHandler),
		maintenanceTargets(deps, withLog, errHandler)...,
	)
}

// amendResituateTargets returns the amend and resituate subcommands. Split out
// of maintenanceTargets to stay within the per-function length budget.
func amendResituateTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
	home string,
) []any {
	return []any{
		targ.Targ(func(ctx context.Context, a ResituateArgs) {
			a.Vault = resolveVault(a.Vault, home, deps.Getenv)
			errHandler(RunResituate(withLog(ctx), a, newResituateDeps(deps), deps.Stdout))
		}).Name("resituate").Description("Rewrite a note's situation in sync (frontmatter + body + sidecar) (D4/INV-S2)"),
		targ.Targ(func(ctx context.Context, a AmendArgs) {
			a.Vault = resolveVault(a.Vault, home, deps.Getenv)
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunAmend(withLog(ctx), a, newAmendDeps(deps), deps.Stdout))
		}).Name("amend").Description("Amend a note in place: supersedes, provenance-merge, field-replacement, activate"),
	}
}

// coreTargets returns the primary subcommands (learn, update, embed, query,
// show, check). Split from Targets to stay within the per-function length
// budget; the wiring mirrors maintenanceTargets exactly.
func coreTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
) []any {
	return append(
		ingestQueryTargets(deps, withLog, errHandler),
		learnUpdateTargets(deps, withLog, errHandler)...,
	)
}

// homeOrEmpty returns the user's home directory via the injected capability,
// or "" when it cannot be resolved (or is unwired, as in minimal test Deps).
// resolveVault tolerates an empty home (it falls back to env / XDG), so the
// error is intentionally discarded.
func homeOrEmpty(deps Deps) string {
	if deps.UserHomeDir == nil {
		return ""
	}

	home, _ := deps.UserHomeDir()

	return home
}

// ingestQueryTargets returns the read/write-vault subcommands (query, ingest,
// query-chunks, activate, show, check). Split from coreTargets to stay within
// the per-function length budget.
func ingestQueryTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
) []any {
	home := homeOrEmpty(deps)

	return []any{
		targ.Targ(func(ctx context.Context, a QueryArgs) {
			a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunQuery(withLog(ctx), a, newQueryDeps(deps), deps.Stdout))
		}).Name("query").Description("Semantic search over vault + chunk index (YAML output)"),
		targ.Targ(func(ctx context.Context, a IngestArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunIngest(withLog(ctx), a, newIngestDeps(deps), deps.Stdout))
		}).Name("ingest").Description("Chunk+embed transcripts/markdown into a chunk index (zero-LLM)"),
		targ.Targ(func(ctx context.Context, a PruneArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunPrune(withLog(ctx), a, newPruneDeps(deps), deps.Stdout))
		}).Name("prune").Description(
			"Detach chunk entries whose source file is gone: drop the stale manifest entry, " +
				"keep the embedded chunks (still searchable)"),
		targ.Targ(func(ctx context.Context, a ChunkQueryArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunChunkQuery(withLog(ctx), a, newChunkQueryDeps(deps), deps.Stdout))
		}).Name("query-chunks").Description("Semantic search over the chunk index (YAML output)"),
		targ.Targ(func(_ context.Context, a ActivateArgs) {
			a.Vault = resolveVault(a.Vault, home, deps.Getenv)
			errHandler(RunActivate(a, newActivateDeps(deps)))
		}).Name("activate").Description("Mark note(s) as recently used (bumps LastUsed in sidecar)"),
		targ.Targ(func(_ context.Context, a CountArgs) {
			a.Vault = resolveVault(a.Vault, home, deps.Getenv)
			errHandler(RunCount(a, newCountDeps(deps), deps.Stdout))
		}).Name("count").Description(
			"Count notes by a frontmatter attribute or a note's wikilink in-degree (read-only)"),
		targ.Targ(func(ctx context.Context, a ShowArgs) {
			a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
			errHandler(RunShow(withLog(ctx), a, newShowDeps(deps), deps.Stdout))
		}).Name("show").Description("Print a note and its outbound wikilink targets (read-only)"),
		targ.Targ(func(ctx context.Context, a ShowChunkArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunShowChunk(withLog(ctx), a, newShowChunkDeps(deps), deps.Stdout))
		}).Name("show-chunk").Description("Print a chunk's text by its source#anchor id (read-only)"),
		targ.Targ(func(ctx context.Context, a CheckArgs) {
			a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
			errHandler(RunCheck(withLog(ctx), a, newCheckDeps(deps), deps.Stdout))
		}).Name("check").Description("Run vault-invariant checks (exit non-zero on FAIL)"),
	}
}

// learnUpdateTargets returns the learn and update subcommands (learn group,
// update, embed group). Split from coreTargets to stay within the
// per-function length budget.
func learnUpdateTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
) []any {
	home := homeOrEmpty(deps)

	return []any{
		targ.Group("learn",
			targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(runLearnFromFeedbackArgs(withLog(ctx), a, deps, deps.Stdout))
			}).Name("feedback").Description("Write a feedback note to the vault"),
			targ.Targ(func(ctx context.Context, a LearnFactArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(runLearnFromFactArgs(withLog(ctx), a, deps, deps.Stdout))
			}).Name("fact").Description("Write a fact note to the vault"),
			targ.Targ(func(ctx context.Context, a LearnQAArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunLearnQA(withLog(ctx), a, newQaDeps(deps), deps.Stdout))
			}).Name("qa").Description("Write a QA pair (Q+A notes) to the vault"),
		),
		targ.Targ(func(ctx context.Context, a UpdateArgs) {
			errHandler(runUpdate(withLog(ctx), a, deps.Stdout))
		}).Name("update").Description("Refresh engram binary and harness skills"),
		targ.Group("embed",
			targ.Targ(func(ctx context.Context, a EmbedApplyArgs) {
				a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
				errHandler(RunEmbedApply(withLog(ctx), a, newOsEmbedDeps(), deps.Stdout))
			}).Name("apply").Description("Embed notes (default: missing only)"),
			targ.Targ(func(ctx context.Context, a EmbedStatusArgs) {
				a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
				errHandler(RunEmbedStatus(withLog(ctx), a, newOsEmbedDeps(), deps.Stdout))
			}).Name("status").Description("Report embedding state counts"),
		),
	}
}

// maintenanceTargets returns the vault-maintenance subcommands (resituate,
// amend, vocab). Split out of Targets to keep each function within the length budget;
// the wiring mirrors the other targets exactly.
func maintenanceTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
) []any {
	home := homeOrEmpty(deps)

	return append(
		amendResituateTargets(deps, withLog, errHandler, home),
		vocabTargets(deps, withLog, errHandler, home)...,
	)
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

// vocabTargets returns the vocab group subcommands (bootstrap, stats,
// propose, refit).
func vocabTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
	home string,
) []any {
	return []any{
		targ.Group("vocab",
			targ.Targ(func(ctx context.Context, a VocabBootstrapArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunVocabBootstrap(withLog(ctx), a, newVocabDeps(deps), deps.Stdout))
			}).Name("bootstrap").Description("Seed vocab term notes + tag all existing notes (idempotent)"),
			targ.Targ(func(_ context.Context, a VocabStatsArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunVocabStats(a, newVocabStatsDeps(deps), deps.Stdout))
			}).Name("stats").Description("Print vocab health report (per-term counts, hubs, orphans, untagged rate)"),
			targ.Targ(func(ctx context.Context, a VocabProposeArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunVocabPropose(withLog(ctx), a, newVocabDeps(deps), deps.Stdout))
			}).Name("propose").Description("Add a new vocab term note + minor version bump (LLM gate runs agent-side)"),
			targ.Targ(func(ctx context.Context, a VocabRefitArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunVocabRefit(withLog(ctx), a, newVocabDeps(deps), deps.Stdout))
			}).Name("refit").Description("Apply a refit plan: renames, removals, additions, re-tag, major version bump"),
		),
	}
}

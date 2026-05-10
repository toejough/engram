package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/toejough/targ"
)

// CommonLearnArgs holds shared flags for learn subcommands.
type CommonLearnArgs struct {
	Situation  string `targ:"flag,name=situation,desc=context when this applies"`
	Source     string `targ:"flag,name=source,desc=human or agent"`
	DataDir    string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	NoDupCheck bool   `targ:"flag,name=no-dup-check,desc=skip duplicate/contradiction detection"`
	LLMCmd     string `targ:"flag,name=llm-cmd,desc=command to invoke for LLM calls (overrides ENGRAM_LLM_CMD)"`
}

// CommonPromoteArgs holds shared flags for promote subcommands.
type CommonPromoteArgs struct {
	Slug           string `targ:"flag,name=slug,desc=kebab-case tag for the filename"`
	Vault          string `targ:"flag,name=vault,env=ENGRAM_VAULT_DIR,desc=vault root directory"`
	Target         string `targ:"flag,name=target,desc=Luhmann ID this note relates to (empty for top-level)"`
	Relation       string `targ:"flag,name=relation,desc=top|continuation|sibling"`
	Source         string `targ:"flag,name=source,desc=provenance string for the source field"`
	DeleteFleeting string `targ:"flag,name=delete-fleeting,desc=path to fleeting note to delete after success"`
}

// LearnFactArgs holds parsed flags for the learn fact subcommand.
type LearnFactArgs struct {
	CommonLearnArgs

	Subject   string `targ:"flag,name=subject,desc=subject of the fact"`
	Predicate string `targ:"flag,name=predicate,desc=relationship or verb"`
	Object    string `targ:"flag,name=object,desc=object of the fact"`
}

// LearnFeedbackArgs holds parsed flags for the learn feedback subcommand.
type LearnFeedbackArgs struct {
	CommonLearnArgs

	Behavior string `targ:"flag,name=behavior,desc=observed behavior"`
	Impact   string `targ:"flag,name=impact,desc=impact of the behavior"`
	Action   string `targ:"flag,name=action,desc=recommended action"`
}

// ListArgs holds parsed flags for the list subcommand.
type ListArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}

// PromoteFactArgs holds parsed flags for the promote fact subcommand.
type PromoteFactArgs struct {
	CommonPromoteArgs

	Situation string `targ:"flag,name=situation,desc=context when this applies"`
	Subject   string `targ:"flag,name=subject,desc=subject of the fact"`
	Predicate string `targ:"flag,name=predicate,desc=relationship or verb"`
	Object    string `targ:"flag,name=object,desc=object of the fact"`
}

// PromoteFeedbackArgs holds parsed flags for the promote feedback subcommand.
type PromoteFeedbackArgs struct {
	CommonPromoteArgs

	Situation string `targ:"flag,name=situation,desc=context when this applies"`
	Behavior  string `targ:"flag,name=behavior,desc=observed behavior"`
	Impact    string `targ:"flag,name=impact,desc=impact of the behavior"`
	Action    string `targ:"flag,name=action,desc=recommended action"`
}

// PromoteMOCArgs holds parsed flags for the promote moc subcommand.
type PromoteMOCArgs struct {
	CommonPromoteArgs

	Topic string `targ:"flag,name=topic,desc=cluster topic name"`
}

// QuickArgs holds parsed flags for the quick subcommand.
type QuickArgs struct {
	Slug    string `targ:"flag,name=slug,desc=kebab-case tag for the fleeting note"`
	Content string `targ:"flag,name=content,desc=full body markdown (or pipe via stdin)"`
	Vault   string `targ:"flag,name=vault,env=ENGRAM_VAULT_DIR,desc=vault root directory"`
}

// RecallArgs holds parsed flags for the recall subcommand.
type RecallArgs struct {
	DataDir       string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ProjectSlug   string `targ:"flag,name=project-slug,desc=project directory slug"`
	Query         string `targ:"flag,name=query,desc=search query (omit for summary mode)"`
	MemoriesOnly  bool   `targ:"flag,name=memories-only,desc=search only memory files"`
	Limit         int    `targ:"flag,name=limit,desc=max memories to return (default 10)"`
	TranscriptDir string `targ:"flag,name=transcript-dir,env=ENGRAM_TRANSCRIPT_DIR,desc=override transcript directory"`
	LLMCmd        string `targ:"flag,name=llm-cmd,desc=command to invoke for LLM calls (overrides ENGRAM_LLM_CMD)"`
}

// ShowArgs holds parsed flags for the show subcommand.
type ShowArgs struct {
	Name    string `targ:"flag,name=name,desc=memory slug to display"`
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}

// StartingPointsArgs holds parsed flags for the starting-points subcommand.
type StartingPointsArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=path to agent-memory vault root"`
}

// UpdateArgs holds parsed flags for the update subcommand.
type UpdateArgs struct {
	Name      string `targ:"flag,name=name,required,desc=memory slug (required)"`
	DataDir   string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Situation string `targ:"flag,name=situation,desc=context when this applies"`
	Behavior  string `targ:"flag,name=behavior,desc=observed behavior"`
	Impact    string `targ:"flag,name=impact,desc=impact of the behavior"`
	Action    string `targ:"flag,name=action,desc=recommended action"`
	Subject   string `targ:"flag,name=subject,desc=subject of the fact"`
	Predicate string `targ:"flag,name=predicate,desc=relationship or verb"`
	Object    string `targ:"flag,name=object,desc=object of the fact"`
	Source    string `targ:"flag,name=source,desc=human or agent"`
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

// Targets returns all targ targets for the engram CLI.
func Targets(stdout, stderr io.Writer, exit func(int)) []any {
	errHandler := newErrHandler(stderr, exit)

	return []any{
		targ.Targ(func(ctx context.Context, a RecallArgs) {
			errHandler(runRecall(ctx, a, stdout))
		}).Name("recall").Description("Recall recent session context"),
		targ.Targ(func(ctx context.Context, a CycleArgs) {
			errHandler(RunCycle(ctx, a, stdout))
		}).Name("cycle").Description("Run a learn-and-recall evaluation cycle"),
		targ.Targ(func(ctx context.Context, a ShowArgs) {
			errHandler(runShow(ctx, a, stdout))
		}).Name("show").Description("Display full memory details"),
		targ.Targ(func(ctx context.Context, a ListArgs) {
			errHandler(runList(ctx, a, stdout))
		}).Name("list").Description("List all memories with type, name, and situation"),
		targ.Targ(func(ctx context.Context, a StartingPointsArgs) {
			errHandler(runStartingPoints(ctx, a, stdout))
		}).Name("starting-points").Description("Emit vault graph traversal entry points (one wikilink per line)"),
		targ.Group("learn",
			targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) {
				errHandler(runLearnFeedback(ctx, a, stdout))
			}).Name("feedback").Description("Learn from behavioral feedback"),
			targ.Targ(func(ctx context.Context, a LearnFactArgs) {
				errHandler(runLearnFact(ctx, a, stdout))
			}).Name("fact").Description("Learn a factual statement"),
		),
		targ.Group("promote",
			targ.Targ(func(ctx context.Context, a PromoteFeedbackArgs) {
				errHandler(runPromoteFromFeedbackArgs(ctx, a, stdout))
			}).Name("feedback").Description("Promote a feedback note to Permanent/"),
			targ.Targ(func(ctx context.Context, a PromoteFactArgs) {
				errHandler(runPromoteFromFactArgs(ctx, a, stdout))
			}).Name("fact").Description("Promote a fact note to Permanent/"),
			targ.Targ(func(ctx context.Context, a PromoteMOCArgs) {
				errHandler(runPromoteFromMOCArgs(ctx, a, stdout))
			}).Name("moc").Description("Promote a MOC note to MOCs/"),
		),
		targ.Targ(func(ctx context.Context, a QuickArgs) {
			fsAdapter := &osQuickFS{}
			deps := QuickDeps{
				Now:      time.Now,
				Stdin:    os.Stdin,
				Getenv:   os.Getenv,
				StatDir:  fsAdapter.StatDir,
				WriteNew: fsAdapter.WriteNew,
			}
			errHandler(runQuick(ctx, a, deps, stdout))
		}).Name("quick").Description("Write a fleeting note to the agent-memory vault"),
		targ.Targ(func(ctx context.Context, a UpdateArgs) {
			errHandler(runUpdate(ctx, a, stdout))
		}).Name("update").Description("Update an existing memory"),
		targ.Targ(func(ctx context.Context, a ReminderArgs) {
			errHandler(runReminder(ctx, a, stdout))
		}).Name("reminder").Description("Emit canonical reminder text"),
		targ.Targ(func(ctx context.Context, a BuildSelfArgs) {
			errHandler(runBuildSelf(ctx, a, stdout))
		}).Name("build-self").Description("Build the engram binary"),
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

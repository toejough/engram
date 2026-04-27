package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/toejough/targ"
)

// CommonLearnArgs holds shared flags for learn subcommands.
type CommonLearnArgs struct {
	Situation  string `targ:"flag,name=situation,desc=context when this applies"`
	Source     string `targ:"flag,name=source,desc=human or agent"`
	DataDir    string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	NoDupCheck bool   `targ:"flag,name=no-dup-check,desc=skip duplicate/contradiction detection"`
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

// RecallArgs holds parsed flags for the recall subcommand.
type RecallArgs struct {
	DataDir      string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ProjectSlug  string `targ:"flag,name=project-slug,desc=project directory slug"`
	Query        string `targ:"flag,name=query,desc=search query (omit for summary mode)"`
	MemoriesOnly bool   `targ:"flag,name=memories-only,desc=search only memory files"`
	Limit        int    `targ:"flag,name=limit,desc=max memories to return (default 10)"`
}

// ShowArgs holds parsed flags for the show subcommand.
type ShowArgs struct {
	Name    string `targ:"flag,name=name,desc=memory slug to display"`
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
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
func Targets(stdout, stderr io.Writer) []any {
	errHandler := func(err error) {
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
		}
	}

	return []any{
		targ.Targ(func(ctx context.Context, a RecallArgs) {
			errHandler(runRecall(ctx, a, stdout))
		}).Name("recall").Description("Recall recent session context"),
		targ.Targ(func(ctx context.Context, a ShowArgs) {
			errHandler(runShow(ctx, a, stdout))
		}).Name("show").Description("Display full memory details"),
		targ.Targ(func(ctx context.Context, a ListArgs) {
			errHandler(runList(ctx, a, stdout))
		}).Name("list").Description("List all memories with type, name, and situation"),
		targ.Group("learn",
			targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) {
				errHandler(runLearnFeedback(ctx, a, stdout))
			}).Name("feedback").Description("Learn from behavioral feedback"),
			targ.Targ(func(ctx context.Context, a LearnFactArgs) {
				errHandler(runLearnFact(ctx, a, stdout))
			}).Name("fact").Description("Learn a factual statement"),
		),
		targ.Targ(func(ctx context.Context, a UpdateArgs) {
			errHandler(runUpdate(ctx, a, stdout))
		}).Name("update").Description("Update an existing memory"),
	}
}

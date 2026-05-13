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
	Slug      string   `targ:"flag,name=slug,desc=kebab-case tag for the filename"`
	Vault     string   `targ:"flag,name=vault,env=ENGRAM_VAULT_DIR,desc=vault root directory"`
	Target    string   `targ:"flag,name=target,desc=Luhmann ID this note relates to (empty for top-level)"`
	Position  string   `targ:"flag,name=position,desc=top|continuation|sibling"`
	Source    string   `targ:"flag,name=source,required,desc=provenance string for the source field (required)"`
	Relations []string `targ:"flag,name=relation,desc=related note as <wikilink-target>|<rationale> (repeatable)"`
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

// LearnMOCArgs holds parsed flags for the learn moc subcommand.
type LearnMOCArgs struct {
	CommonLearnArgs

	Topic   string `targ:"flag,name=topic,desc=cluster topic name"`
	Framing string `targ:"flag,name=framing,desc=framing paragraph(s) for the MOC body"`
}

// RecallArgs holds parsed flags for the recall subcommand. Recall is
// vault-only; three modes are mutually selected by flag:
//
//   - No args: emit structural anchors (MOCs + per-component winners).
//   - --recent: emit the most-recent notes by filename date prefix.
//   - --follow: emit cascade frontier expansion (outgoing + backlinks).
type RecallArgs struct {
	VaultPath   string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=path to vault root (required)"`
	Recent      bool     `targ:"flag,name=recent,desc=emit most-recent notes by filename date instead of anchors"`
	Limit       int      `targ:"flag,name=limit,desc=cap for --recent output (default 20)"`
	Follow      []string `targ:"flag,name=follow,desc=basenames to expand (outgoing + backlinks)"`
	AlreadyRead []string `targ:"flag,name=already-read,desc=basenames to subtract from --follow output"`
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

	return []any{
		targ.Targ(func(ctx context.Context, a RecallArgs) {
			errHandler(runRecall(withLog(ctx), a, stdout))
		}).Name("recall").Description("Recall recent session context"),
		targ.Targ(func(ctx context.Context, a TranscriptArgs) {
			errHandler(runTranscript(withLog(ctx), a, stdout))
		}).Name("transcript").Description("Read session transcripts in a date range"),
		targ.Group("learn",
			targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) {
				errHandler(runLearnFromFeedbackArgs(withLog(ctx), a, stdout))
			}).Name("feedback").Description("Write a feedback note to Permanent/"),
			targ.Targ(func(ctx context.Context, a LearnFactArgs) {
				errHandler(runLearnFromFactArgs(withLog(ctx), a, stdout))
			}).Name("fact").Description("Write a fact note to Permanent/"),
			targ.Targ(func(ctx context.Context, a LearnMOCArgs) {
				errHandler(runLearnFromMOCArgs(withLog(ctx), a, stdout))
			}).Name("moc").Description("Write a MOC note to MOCs/"),
		),
		targ.Targ(func(ctx context.Context, a UpdateArgs) {
			errHandler(runUpdate(withLog(ctx), a, stdout))
		}).Name("update").Description("Refresh engram binary and harness skills"),
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

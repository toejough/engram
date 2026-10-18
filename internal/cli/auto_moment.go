package cli

import (
	"context"
	"io"
)

// AutoMomentArgs holds parsed flags for `engram auto moment`.
type AutoMomentArgs struct {
	SessionsDir string `targ:"flag,name=sessions,desc=directory of recorded sessions (default ENGRAM_TRANSCRIPT_DIR then CLAUDE_CONFIG_DIR/projects then ~/.claude/projects)"` //nolint:lll // single unbreakable struct-tag string
}

// AutoMomentDeps holds injected dependencies for RunAutoMoment.
type AutoMomentDeps struct {
	// Stdin carries the moment: one JSON object.
	Stdin io.Reader
	// ReadRecord reads the record of a session in the session-log directory.
	ReadRecord RecordRead
}

// RecordRead returns the record of a session, which Claude Code keeps in a directory of records, or nothing.
type RecordRead func(dir, session string) []byte

// RunAutoMoment reads a moment from standard input and prints the request for the model that judges it.
func RunAutoMoment(_ context.Context, _ AutoMomentArgs, _ AutoMomentDeps, _ io.Writer) error {
	return nil
}

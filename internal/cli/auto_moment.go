package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/toejough/engram/internal/auto"
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

// RunAutoMoment reads a moment from standard input and prints the request for the model that judges it, and the carry
// for the next step, or nothing when the prompt is not one the user sends (855d4a). The record of the session is read
// once, and a session with no record is judged without conversation lines.
func RunAutoMoment(_ context.Context, args AutoMomentArgs, deps AutoMomentDeps, stdout io.Writer) error {
	var input auto.MomentInput

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&input)
	if decodeErr != nil {
		return fmt.Errorf("auto moment: reading the moment: %w", decodeErr)
	}

	if !auto.UserSent(input.Origin) {
		return nil
	}

	encoded, marshalErr := json.Marshal(auto.Moment(deps.ReadRecord(args.SessionsDir, input.Session), input))
	if marshalErr != nil {
		return fmt.Errorf("auto moment: encoding the request: %w", marshalErr)
	}

	_, writeErr := fmt.Fprintln(stdout, string(encoded))
	if writeErr != nil {
		return fmt.Errorf("auto moment: writing the request: %w", writeErr)
	}

	return nil
}

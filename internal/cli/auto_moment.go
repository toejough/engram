package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/toejough/engram/internal/auto"
	"github.com/toejough/engram/internal/debuglog"
)

// AutoMomentArgs holds parsed flags for `engram auto moment`.
type AutoMomentArgs struct {
	SessionsDir string `targ:"flag,name=sessions,desc=directory of recorded sessions (default ENGRAM_TRANSCRIPT_DIR then CLAUDE_CONFIG_DIR/projects then ~/.claude/projects)"`                            //nolint:lll // single unbreakable struct-tag string
	CaptureSDK  bool   `targ:"flag,name=capture-sdk,env=ENGRAM_AUTO_CAPTURE_SDK,desc=judge the turns of headless prompts (origin sdk) too; off by default so a script run with claude -p writes no note"` //nolint:lll // single unbreakable struct-tag string
	RecordsDir  string `targ:"flag,name=records,desc=directory of the records of what automatic memory did (default $XDG_DATA_HOME/engram/auto)"`                                                         //nolint:lll // single unbreakable struct-tag string
}

// AutoMomentDeps holds injected dependencies for RunAutoMoment.
type AutoMomentDeps struct {
	// Stdin carries the moment: one JSON object.
	Stdin io.Reader
	// ReadRecord reads the record of a session in the session-log directory.
	ReadRecord RecordRead
	// Now is the time a record is made at.
	Now func() time.Time
	// WriteRecord writes one record, a file of its own, and makes its directory.
	WriteRecord func(path string, data []byte) error
}

// RecordRead returns the record of a session, which Claude Code keeps in a directory of records, or nothing.
type RecordRead func(dir, session string) []byte

// RunAutoMoment reads a moment from standard input and prints the request for the model that judges it, and the carry
// for the next step, or nothing when the prompt is not one the user sends, or, at the end of a turn, when the turn is
// a headless one that the user did not ask to capture, which a record tells (855d4a, 855d4m). The record of the
// session is read once, and a session with no record is judged without conversation lines.
func RunAutoMoment(ctx context.Context, args AutoMomentArgs, deps AutoMomentDeps, stdout io.Writer) error {
	var input auto.MomentInput

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&input)
	if decodeErr != nil {
		return fmt.Errorf("auto moment: reading the moment: %w", decodeErr)
	}

	if !auto.UserSent(input.Origin) || (input.Cue == auto.CueTurnEnd && !auto.Judged(input.Origin, args.CaptureSDK)) {
		// a turn of a prompt the user sends that is not judged is a headless one that the user did not ask to capture
		if auto.UserSent(input.Origin) {
			recordLeftOut(deps.Now, deps.WriteRecord, args.RecordsDir, input)
		}

		// the end of a capture that no write follows is told by this line (855b52)
		if input.Cue == auto.CueTurnEnd {
			debuglog.Log(ctx, "auto.moment", "session=%s nothing", input.Session)
		}

		return nil
	}

	return printStep(stdout, "auto moment", auto.Moment(deps.ReadRecord(args.SessionsDir, input.Session), input))
}

// newAutoMomentDeps wires RunAutoMoment from the injected CLI capabilities - pure composition (#700).
func newAutoMomentDeps(d Deps) AutoMomentDeps {
	return AutoMomentDeps{
		Stdin: d.Stdin,
		ReadRecord: func(dir, session string) []byte {
			return recordOf(d.FS.ReadDir, d.FS.ReadFile, dir, session)
		},
		Now:         d.Now,
		WriteRecord: recordWriterFrom(d),
	}
}

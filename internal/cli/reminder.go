package cli

import (
	"context"
	"fmt"
	"io"

	"engram/internal/reminders"
)

// ReminderArgs holds parsed flags for the reminder subcommand.
type ReminderArgs struct {
	Kind string `targ:"positional,required,desc=reminder kind (session-start | user-prompt | post-tool | system)"`
}

func runReminder(_ context.Context, args ReminderArgs, stdout io.Writer) error {
	text, err := reminders.Get(args.Kind)
	if err != nil {
		return fmt.Errorf("reminder: %w", err)
	}

	_, writeErr := fmt.Fprint(stdout, text)
	if writeErr != nil {
		return fmt.Errorf("reminder: %w", writeErr)
	}

	return nil
}

package log

import (
	"encoding/json"
	"fmt"
	"time"
)

// ReadArgs holds arguments for the log read command.
type ReadArgs struct {
	Dir   string `targ:"flag,short=d,required,desc=Project directory"`
	Model string `targ:"flag,desc=Filter by model (haiku|sonnet|opus)"`
}

// WriteArgs holds arguments for the log write command.
type WriteArgs struct {
	Dir             string `targ:"flag,short=d,required,desc=Project directory"`
	Level           string `targ:"flag,short=l,required,desc=Log level (verbose|status|phase)"`
	Subject         string `targ:"flag,short=s,required,desc=Log subject (thinking|skill-result|etc)"`
	Message         string `targ:"flag,short=m,required,desc=Log message"`
	Task            string `targ:"flag,desc=Task ID (e.g. TASK-004)"`
	Phase           string `targ:"flag,desc=Current phase"`
	Model           string `targ:"flag,desc=Model used (haiku|sonnet|opus)"`
	Tokens          int    `targ:"flag,desc=Override token estimate with known value"`
	ContextEstimate int    `targ:"flag,short=c,desc=Current context usage estimate (tokens)"`
}

// RunRead reads and prints log entries.
func RunRead(args ReadArgs) error {
	entries, err := Read(args.Dir, ReadOpts{
		Model: args.Model,
	}, RealFS{})
	if err != nil {
		return err
	}

	for _, entry := range entries {
		line, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal entry: %w", err)
		}

		fmt.Println(string(line))
	}

	return nil
}

// RunWrite writes a log entry.
func RunWrite(args WriteArgs) error {
	err := Write(args.Dir, args.Level, args.Subject, args.Message, WriteOpts{
		Task:            args.Task,
		Phase:           args.Phase,
		Model:           args.Model,
		Tokens:          args.Tokens,
		ContextEstimate: args.ContextEstimate,
	}, time.Now, RealFS{})
	if err != nil {
		return err
	}

	fmt.Println("Log entry written")

	return nil
}

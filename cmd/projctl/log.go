package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/toejough/projctl/internal/log"
)

type logWriteArgs struct {
	Dir     string `targ:"flag,short=d,required,desc=Project directory"`
	Level   string `targ:"flag,short=l,required,desc=Log level (verbose|status|phase)"`
	Subject string `targ:"flag,short=s,required,desc=Log subject (thinking|skill-result|etc)"`
	Message string `targ:"flag,short=m,required,desc=Log message"`
	Task    string `targ:"flag,desc=Task ID (e.g. TASK-004)"`
	Phase   string `targ:"flag,desc=Current phase"`
	Model   string `targ:"flag,desc=Model used (haiku|sonnet|opus)"`
}

func logWrite(args logWriteArgs) error {
	err := log.Write(args.Dir, args.Level, args.Subject, args.Message, log.WriteOpts{
		Task:  args.Task,
		Phase: args.Phase,
		Model: args.Model,
	}, time.Now)
	if err != nil {
		return err
	}

	fmt.Println("Log entry written")

	return nil
}

type logReadArgs struct {
	Dir   string `targ:"flag,short=d,required,desc=Project directory"`
	Model string `targ:"flag,desc=Filter by model (haiku|sonnet|opus)"`
}

func logRead(args logReadArgs) error {
	entries, err := log.Read(args.Dir, log.ReadOpts{
		Model: args.Model,
	})
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

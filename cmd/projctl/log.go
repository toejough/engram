package main

import (
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
}

func logWrite(args logWriteArgs) error {
	err := log.Write(args.Dir, args.Level, args.Subject, args.Message, log.WriteOpts{
		Task:  args.Task,
		Phase: args.Phase,
	}, time.Now)
	if err != nil {
		return err
	}

	fmt.Println("Log entry written")

	return nil
}

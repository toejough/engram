package main

import (
	"fmt"

	"github.com/toejough/projctl/internal/context"
)

type contextWriteArgs struct {
	Dir   string `targ:"flag,short=d,required,desc=Project directory"`
	Task  string `targ:"flag,short=t,required,desc=Task ID (e.g. TASK-004)"`
	Skill string `targ:"flag,short=s,required,desc=Skill name (e.g. tdd-red)"`
	File  string `targ:"flag,short=f,required,desc=Path to TOML context file"`
}

func contextWrite(args contextWriteArgs) error {
	path, err := context.Write(args.Dir, args.Task, args.Skill, args.File)
	if err != nil {
		return err
	}

	fmt.Printf("Context written: %s\n", path)

	return nil
}

type contextReadArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Project directory"`
	Task   string `targ:"flag,short=t,required,desc=Task ID (e.g. TASK-004)"`
	Skill  string `targ:"flag,short=s,required,desc=Skill name (e.g. tdd-red)"`
	Result bool   `targ:"flag,short=r,desc=Read result file instead of context file"`
}

func contextRead(args contextReadArgs) error {
	content, err := context.Read(args.Dir, args.Task, args.Skill, args.Result)
	if err != nil {
		return err
	}

	fmt.Print(content)

	return nil
}

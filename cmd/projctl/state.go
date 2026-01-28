package main

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/toejough/projctl/internal/state"
)

type stateInitArgs struct {
	Name string `targ:"flag,short=n,required,desc=Project name"`
	Dir  string `targ:"flag,short=d,required,desc=Project directory"`
}

func stateInit(args stateInitArgs) error {
	s, err := state.Init(args.Dir, args.Name, time.Now)
	if err != nil {
		return err
	}

	fmt.Printf("Initialized project %q in %s (phase: %s)\n", s.Project.Name, args.Dir, s.Project.Phase)

	return nil
}

type stateGetArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func stateGet(args stateGetArgs) error {
	s, err := state.Get(args.Dir)
	if err != nil {
		return err
	}

	return toml.NewEncoder(os.Stdout).Encode(s)
}

type stateTransitionArgs struct {
	Dir      string `targ:"flag,short=d,required,desc=Project directory"`
	To       string `targ:"flag,short=t,required,desc=Target phase"`
	Task     string `targ:"flag,desc=Current task ID (e.g. TASK-004)"`
	Subphase string `targ:"flag,desc=Current subphase (e.g. tdd-green)"`
}

func stateTransition(args stateTransitionArgs) error {
	s, err := state.Transition(args.Dir, args.To, state.TransitionOpts{
		Task:     args.Task,
		Subphase: args.Subphase,
	}, time.Now)
	if err != nil {
		return err
	}

	fmt.Printf("Transitioned to %q (task: %s, subphase: %s)\n",
		s.Project.Phase,
		s.Progress.CurrentTask,
		s.Progress.CurrentSubphase,
	)

	return nil
}

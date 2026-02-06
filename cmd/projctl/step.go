package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/toejough/projctl/internal/step"
)

type stepNextArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func stepNext(args stepNextArgs) error {
	result, err := step.Next(args.Dir)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}

	fmt.Println(string(data))

	return nil
}

type stepCompleteArgs struct {
	Dir        string `targ:"flag,short=d,required,desc=Project directory"`
	Action     string `targ:"flag,short=a,required,desc=Completed action (spawn-producer, spawn-qa, commit, transition)"`
	Status     string `targ:"flag,short=s,required,desc=Result status (done, failed)"`
	QAVerdict  string `targ:"flag,short=v,desc=QA verdict (approved, improvement-request, escalate-phase, escalate-user)"`
	QAFeedback string `targ:"flag,short=f,desc=QA feedback text"`
	Phase      string `targ:"flag,short=p,desc=Target phase (for transition action)"`
}

func stepComplete(args stepCompleteArgs) error {
	err := step.Complete(args.Dir, step.CompleteResult{
		Action:     args.Action,
		Status:     args.Status,
		QAVerdict:  args.QAVerdict,
		QAFeedback: args.QAFeedback,
		Phase:      args.Phase,
	}, time.Now)
	if err != nil {
		return err
	}

	fmt.Println("Step completed successfully")

	return nil
}

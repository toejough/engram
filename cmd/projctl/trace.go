package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/toejough/projctl/internal/trace"
)

type traceAddArgs struct {
	Dir  string `targ:"flag,short=d,required,desc=Project directory"`
	From string `targ:"flag,short=f,required,desc=Source ID (e.g. REQ-001)"`
	To   string `targ:"flag,short=t,required,desc=Target ID(s) comma-separated (e.g. DES-001,ARCH-001)"`
}

func traceAdd(args traceAddArgs) error {
	targets := strings.Split(args.To, ",")
	for i := range targets {
		targets[i] = strings.TrimSpace(targets[i])
	}

	err := trace.Add(args.Dir, args.From, targets)
	if err != nil {
		return err
	}

	fmt.Printf("Trace link added: %s → %s\n", args.From, args.To)

	return nil
}

type traceValidateArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func traceValidate(args traceValidateArgs) error {
	result, err := trace.Validate(args.Dir)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}

	fmt.Println(string(data))

	if !result.Pass {
		os.Exit(1)
	}

	return nil
}

type traceImpactArgs struct {
	Dir     string `targ:"flag,short=d,required,desc=Project directory"`
	ID      string `targ:"flag,short=i,required,desc=Traceability ID (e.g. REQ-003)"`
	Reverse bool   `targ:"flag,short=r,desc=Backward (upstream) analysis"`
}

func traceImpact(args traceImpactArgs) error {
	result, err := trace.Impact(args.Dir, args.ID, args.Reverse)
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

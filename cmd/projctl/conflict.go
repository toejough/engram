package main

import (
	"encoding/json"
	"fmt"

	"github.com/toejough/projctl/internal/conflict"
)

type conflictCreateArgs struct {
	Dir          string `targ:"flag,short=d,required,desc=Project directory"`
	Skills       string `targ:"flag,short=s,required,desc=Involved skills (e.g. pm,architect)"`
	Traceability string `targ:"flag,short=t,required,desc=Traceability IDs (e.g. REQ-001,ARCH-003)"`
	Description  string `targ:"flag,short=m,required,desc=Conflict description"`
}

func conflictCreate(args conflictCreateArgs) error {
	id, err := conflict.Create(args.Dir, args.Skills, args.Traceability, args.Description, conflict.RealFS{})
	if err != nil {
		return err
	}

	fmt.Printf("Conflict created: %s\n", id)

	return nil
}

type conflictCheckArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func conflictCheck(args conflictCheckArgs) error {
	result, err := conflict.Check(args.Dir, conflict.RealFS{})
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

type conflictListArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Project directory"`
	Status string `targ:"flag,short=s,desc=Filter by status (open|resolved|negotiating)"`
}

func conflictList(args conflictListArgs) error {
	result, err := conflict.List(args.Dir, args.Status, conflict.RealFS{})
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

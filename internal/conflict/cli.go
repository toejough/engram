package conflict

import (
	"encoding/json"
	"fmt"
)

// CheckArgs holds arguments for the conflict check command.
type CheckArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

// CreateArgs holds arguments for the conflict create command.
type CreateArgs struct {
	Dir          string `targ:"flag,short=d,required,desc=Project directory"`
	Skills       string `targ:"flag,short=s,required,desc=Involved skills (e.g. pm;architect)"`
	Traceability string `targ:"flag,short=t,required,desc=Traceability IDs (e.g. REQ-001;ARCH-003)"`
	Description  string `targ:"flag,short=m,required,desc=Conflict description"`
}

// ListArgs holds arguments for the conflict list command.
type ListArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Project directory"`
	Status string `targ:"flag,short=s,desc=Filter by status (open|resolved|negotiating)"`
}

// RunCheck checks for unresolved conflicts.
func RunCheck(args CheckArgs) error {
	result, err := Check(args.Dir, RealFS{})
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

// RunCreate creates a conflict record.
func RunCreate(args CreateArgs) error {
	id, err := Create(args.Dir, args.Skills, args.Traceability, args.Description, RealFS{})
	if err != nil {
		return err
	}

	fmt.Printf("Conflict created: %s\n", id)

	return nil
}

// RunList lists all conflicts.
func RunList(args ListArgs) error {
	result, err := List(args.Dir, args.Status, RealFS{})
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

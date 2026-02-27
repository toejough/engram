package escalation

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ListArgs holds arguments for the escalation list command.
type ListArgs struct {
	Dir    string `targ:"flag,short=d,desc=Project directory (default: current)"`
	File   string `targ:"flag,short=f,desc=Escalation file path (default: escalations.md)"`
	Status string `targ:"flag,short=s,desc=Filter by status (pending / resolved / deferred / issue)"`
	JSON   bool   `targ:"flag,short=j,desc=Output as JSON"`
}

// RealCommandExecutor implements CommandExecutor using os/exec.
type RealCommandExecutor struct{}

func (r *RealCommandExecutor) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RealEscalationFS implements EscalationFS using the real file system.
type RealEscalationFS struct{}

func (r *RealEscalationFS) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (r *RealEscalationFS) WriteFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// ResolveArgs holds arguments for the escalation resolve command.
type ResolveArgs struct {
	Dir    string `targ:"flag,short=d,desc=Project directory (default: current)"`
	File   string `targ:"flag,short=f,desc=Escalation file path (default: escalations.md)"`
	ID     string `targ:"flag,desc=Escalation ID to resolve (e.g. ESC-001),required"`
	Status string `targ:"flag,short=s,required,desc=New status: resolved / deferred / issue"`
	Notes  string `targ:"flag,short=n,desc=Resolution notes"`
}

// ReviewArgs holds arguments for the escalation review command.
type ReviewArgs struct {
	Dir  string `targ:"flag,short=d,desc=Project directory (default: current)"`
	File string `targ:"flag,short=f,desc=Escalation file path (default: escalations.md)"`
}

// WriteArgs holds arguments for the escalation write command.
type WriteArgs struct {
	Dir      string `targ:"flag,short=d,desc=Project directory (default: current)"`
	File     string `targ:"flag,short=f,desc=Output file path (default: escalations.md)"`
	ID       string `targ:"flag,desc=Escalation ID (e.g. ESC-001),required"`
	Category string `targ:"flag,short=c,required,desc=Category: requirement / design / architecture / task"`
	Context  string `targ:"flag,desc=What was being analyzed,required"`
	Question string `targ:"flag,short=q,desc=The question needing resolution,required"`
}

// RunList lists escalations with optional status filter.
func RunList(args ListArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	filePath := args.File
	if filePath == "" {
		filePath = filepath.Join(dir, "escalations.md")
	}

	fs := &RealEscalationFS{}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if args.JSON {
			fmt.Println("[]")
		} else {
			fmt.Println("No escalations found")
		}

		return nil
	}

	escalations, err := ParseEscalationFile(filePath, fs)
	if err != nil {
		return fmt.Errorf("failed to parse escalations: %w", err)
	}

	if args.Status != "" {
		var filtered []Escalation

		for _, e := range escalations {
			if e.Status == args.Status {
				filtered = append(filtered, e)
			}
		}

		escalations = filtered
	}

	if args.JSON {
		data, err := json.MarshalIndent(escalations, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}

		fmt.Println(string(data))
	} else {
		if len(escalations) == 0 {
			fmt.Println("No escalations found")
			return nil
		}

		for _, e := range escalations {
			fmt.Printf("%s [%s] %s\n", e.ID, e.Status, e.Question)
		}
	}

	return nil
}

// RunResolve resolves an escalation by ID.
func RunResolve(args ResolveArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	filePath := args.File
	if filePath == "" {
		filePath = filepath.Join(dir, "escalations.md")
	}

	fs := &RealEscalationFS{}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("escalation file does not exist: %s", filePath)
	}

	escalations, err := ParseEscalationFile(filePath, fs)
	if err != nil {
		return fmt.Errorf("failed to parse escalations: %w", err)
	}

	updated, err := Resolve(escalations, args.ID, args.Status, args.Notes)
	if err != nil {
		return fmt.Errorf("failed to resolve: %w", err)
	}

	if err := WriteEscalationFile(filePath, updated, fs); err != nil {
		return fmt.Errorf("failed to write escalations: %w", err)
	}

	fmt.Printf("Resolved %s with status %q\n", args.ID, args.Status)

	return nil
}

// RunReview reviews pending escalations in an editor.
func RunReview(args ReviewArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	filePath := args.File
	if filePath == "" {
		filePath = filepath.Join(dir, "escalations.md")
	}

	fs := &RealEscalationFS{}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("escalation file does not exist: %s", filePath)
	}

	existing, err := ParseEscalationFile(filePath, fs)
	if err != nil {
		return fmt.Errorf("failed to parse escalations: %w", err)
	}

	executor := &RealCommandExecutor{}

	reviewed, err := ReviewEscalations(existing, filePath, os.Getenv, executor, fs)
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	result := ApplyResolutions(reviewed)

	fmt.Printf("Review complete:\n")
	fmt.Printf("  Applied: %d\n", len(result.Applied))
	fmt.Printf("  Issues:  %d\n", len(result.Issues))
	fmt.Printf("  Pending: %d\n", len(result.Pending))

	return nil
}

// RunWrite adds a new escalation to the file.
func RunWrite(args WriteArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	filePath := args.File
	if filePath == "" {
		filePath = filepath.Join(dir, "escalations.md")
	}

	fs := &RealEscalationFS{}

	var escalations []Escalation

	if _, err := os.Stat(filePath); err == nil {
		existing, err := ParseEscalationFile(filePath, fs)
		if err != nil {
			return fmt.Errorf("failed to parse existing escalations: %w", err)
		}

		escalations = existing
	}

	newEsc := Escalation{
		ID:       args.ID,
		Category: args.Category,
		Context:  args.Context,
		Question: args.Question,
		Status:   "pending",
		Notes:    "",
	}
	escalations = append(escalations, newEsc)

	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	err = WriteEscalationFile(filePath, escalations, fs)
	if err != nil {
		return fmt.Errorf("failed to write escalations: %w", err)
	}

	fmt.Printf("Added escalation %s to %s\n", args.ID, filePath)

	return nil
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/toejough/projctl/internal/escalation"
)

// realEscalationFS implements escalation.EscalationFS using the real file system.
type realEscalationFS struct{}

func (r *realEscalationFS) WriteFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func (r *realEscalationFS) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type escalationWriteArgs struct {
	Dir      string `targ:"flag,short=d,desc=Project directory (default: current)"`
	File     string `targ:"flag,short=f,desc=Output file path (default: escalations.md)"`
	ID       string `targ:"flag,desc=Escalation ID (e.g. ESC-001),required"`
	Category string `targ:"flag,short=c,desc=Category: requirement, design, architecture, task,required"`
	Context  string `targ:"flag,desc=What was being analyzed,required"`
	Question string `targ:"flag,short=q,desc=The question needing resolution,required"`
}

func escalationWrite(args escalationWriteArgs) error {
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

	fs := &realEscalationFS{}

	// Load existing escalations if file exists
	var escalations []escalation.Escalation
	if _, err := os.Stat(filePath); err == nil {
		existing, err := escalation.ParseEscalationFile(filePath, fs)
		if err != nil {
			return fmt.Errorf("failed to parse existing escalations: %w", err)
		}
		escalations = existing
	}

	// Add new escalation
	newEsc := escalation.Escalation{
		ID:       args.ID,
		Category: args.Category,
		Context:  args.Context,
		Question: args.Question,
		Status:   "pending",
		Notes:    "",
	}
	escalations = append(escalations, newEsc)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := escalation.WriteEscalationFile(filePath, escalations, fs); err != nil {
		return fmt.Errorf("failed to write escalations: %w", err)
	}

	fmt.Printf("Added escalation %s to %s\n", args.ID, filePath)
	return nil
}

type escalationReviewArgs struct {
	Dir  string `targ:"flag,short=d,desc=Project directory (default: current)"`
	File string `targ:"flag,short=f,desc=Escalation file path (default: escalations.md)"`
}

func escalationReview(args escalationReviewArgs) error {
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

	fs := &realEscalationFS{}

	// Load existing escalations
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("escalation file does not exist: %s", filePath)
	}

	existing, err := escalation.ParseEscalationFile(filePath, fs)
	if err != nil {
		return fmt.Errorf("failed to parse escalations: %w", err)
	}

	// Open in editor for review
	executor := &realCommandExecutor{}
	reviewed, err := escalation.ReviewEscalations(existing, filePath, os.Getenv, executor, fs)
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	// Apply resolutions
	result := escalation.ApplyResolutions(reviewed)

	fmt.Printf("Review complete:\n")
	fmt.Printf("  Applied: %d\n", len(result.Applied))
	fmt.Printf("  Issues:  %d\n", len(result.Issues))
	fmt.Printf("  Pending: %d\n", len(result.Pending))

	return nil
}

// realCommandExecutor implements escalation.CommandExecutor
type realCommandExecutor struct{}

func (r *realCommandExecutor) Run(name string, args ...string) error {
	return runCommand(name, args...)
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type escalationListArgs struct {
	Dir    string `targ:"flag,short=d,desc=Project directory (default: current)"`
	File   string `targ:"flag,short=f,desc=Escalation file path (default: escalations.md)"`
	Status string `targ:"flag,short=s,desc=Filter by status (pending, resolved, deferred, issue)"`
	JSON   bool   `targ:"flag,short=j,desc=Output as JSON"`
}

type escalationResolveArgs struct {
	Dir    string `targ:"flag,short=d,desc=Project directory (default: current)"`
	File   string `targ:"flag,short=f,desc=Escalation file path (default: escalations.md)"`
	ID     string `targ:"flag,desc=Escalation ID to resolve (e.g. ESC-001),required"`
	Status string `targ:"flag,short=s,desc=New status: resolved, deferred, issue,required"`
	Notes  string `targ:"flag,short=n,desc=Resolution notes"`
}

func escalationResolve(args escalationResolveArgs) error {
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

	fs := &realEscalationFS{}

	// Load existing escalations
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("escalation file does not exist: %s", filePath)
	}

	escalations, err := escalation.ParseEscalationFile(filePath, fs)
	if err != nil {
		return fmt.Errorf("failed to parse escalations: %w", err)
	}

	// Resolve the escalation
	updated, err := escalation.Resolve(escalations, args.ID, args.Status, args.Notes)
	if err != nil {
		return fmt.Errorf("failed to resolve: %w", err)
	}

	// Write back
	if err := escalation.WriteEscalationFile(filePath, updated, fs); err != nil {
		return fmt.Errorf("failed to write escalations: %w", err)
	}

	fmt.Printf("Resolved %s with status %q\n", args.ID, args.Status)
	return nil
}

func escalationList(args escalationListArgs) error {
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

	fs := &realEscalationFS{}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if args.JSON {
			fmt.Println("[]")
		} else {
			fmt.Println("No escalations found")
		}
		return nil
	}

	escalations, err := escalation.ParseEscalationFile(filePath, fs)
	if err != nil {
		return fmt.Errorf("failed to parse escalations: %w", err)
	}

	// Filter by status if specified
	if args.Status != "" {
		var filtered []escalation.Escalation
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

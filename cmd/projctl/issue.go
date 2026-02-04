package main

import (
	"fmt"
	"os"
	"time"

	"github.com/toejough/projctl/internal/issue"
)

type issueCreateArgs struct {
	Dir      string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Title    string `targ:"flag,short=t,required,desc=Issue title"`
	Priority string `targ:"flag,short=p,desc=Priority (High, Medium, Low)"`
	Body     string `targ:"flag,short=b,desc=Issue body/description"`
}

func issueCreate(args issueCreateArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	i, err := issue.Create(dir, issue.CreateOpts{
		Title:    args.Title,
		Priority: args.Priority,
		Body:     args.Body,
	}, time.Now)
	if err != nil {
		return err
	}

	fmt.Printf("Created %s: %s\n", i.ID, i.Title)
	return nil
}

type issueUpdateArgs struct {
	Dir     string `targ:"flag,short=d,desc=Project directory (default: current)"`
	ID      string `targ:"flag,short=i,required,desc=Issue ID (e.g. ISSUE-042)"`
	Status  string `targ:"flag,short=s,desc=New status (Open, Closed, etc.)"`
	Comment string `targ:"flag,short=c,desc=Comment to append"`
	Force   bool   `targ:"flag,short=f,desc=Force close even if AC incomplete"`
}

func issueUpdate(args issueUpdateArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	if args.Status == "" && args.Comment == "" {
		return fmt.Errorf("at least one of --status or --comment must be specified")
	}

	if err := issue.Update(dir, args.ID, issue.UpdateOpts{
		Status:  args.Status,
		Comment: args.Comment,
		Force:   args.Force,
	}); err != nil {
		return err
	}

	fmt.Printf("Updated %s\n", args.ID)
	return nil
}

type issueListArgs struct {
	Dir    string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Status string `targ:"flag,short=s,desc=Filter by status (e.g. Open)"`
}

func issueList(args issueListArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	issues, err := issue.List(dir, args.Status)
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		fmt.Println("No issues found")
		return nil
	}

	for _, i := range issues {
		fmt.Printf("%s: %s [%s] (%s)\n", i.ID, i.Title, i.Status, i.Priority)
	}

	return nil
}

type issueGetArgs struct {
	Dir string `targ:"flag,short=d,desc=Project directory (default: current)"`
	ID  string `targ:"flag,short=i,required,desc=Issue ID (e.g. ISSUE-042)"`
}

func issueGet(args issueGetArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	i, err := issue.Get(dir, args.ID)
	if err != nil {
		return err
	}

	fmt.Printf("## %s: %s\n\n", i.ID, i.Title)
	fmt.Printf("**Priority:** %s\n", i.Priority)
	fmt.Printf("**Status:** %s\n", i.Status)
	fmt.Printf("**Created:** %s\n", i.Created)
	if i.Body != "" {
		fmt.Printf("\n%s\n", i.Body)
	}

	return nil
}

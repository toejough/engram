//go:build targ

package dev

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/toejough/projctl/internal/issue"
	"github.com/toejough/targ"
	_ "github.com/toejough/targ/dev"
)

func init() {
	targ.Register(InstallSkills, InstallProjctl, InstallHooks, InstallBinary, Install, GitStatus)
	targ.Register(IssueCreate, IssueUpdate, IssueList, IssueGet)
}

// Exported variables.
var (
	GitStatus      = targ.Targ("git status").Name("git-status")
	Install        = targ.Targ("targ install-projctl install-skills install-hooks").Name("install")
	InstallBinary  = targ.Targ("targ install-projctl install-hooks").Name("install-binary")
	InstallHooks   = targ.Targ("projctl memory hooks install").Name("install-hooks")
	InstallProjctl = targ.Targ("go install -tags sqlite_fts5 ./cmd/projctl").Name("install-projctl")
	InstallSkills  = targ.Targ("projctl skills install").Name("install-skills")
	IssueCreate    = targ.Targ(func(_ context.Context, args issueCreateArgs) error {
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
	}).Name("issue-create")
	IssueGet = targ.Targ(func(_ context.Context, args issueGetArgs) error {
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
	}).Name("issue-get")
	IssueList = targ.Targ(func(_ context.Context, args issueListArgs) error {
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
	}).Name("issue-list")
	IssueUpdate = targ.Targ(func(_ context.Context, args issueUpdateArgs) error {
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
	}).Name("issue-update")
)

type issueCreateArgs struct {
	Dir      string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Title    string `targ:"flag,short=t,required,desc=Issue title"`
	Priority string `targ:"flag,short=p,enum=High|Medium|Low,desc=Issue priority"`
	Body     string `targ:"flag,short=b,desc=Issue body/description"`
}

type issueGetArgs struct {
	Dir string `targ:"flag,short=d,desc=Project directory (default: current)"`
	ID  string `targ:"flag,short=i,required,desc=Issue ID (e.g. ISSUE-42)"`
}

type issueListArgs struct {
	Dir    string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Status string `targ:"flag,short=s,desc=Filter by status (e.g. Open)"`
}

type issueUpdateArgs struct {
	Dir     string `targ:"flag,short=d,desc=Project directory (default: current)"`
	ID      string `targ:"flag,short=i,required,desc=Issue ID (e.g. ISSUE-42)"`
	Status  string `targ:"flag,short=s,desc=New status"`
	Comment string `targ:"flag,short=c,desc=Comment to append"`
	Force   bool   `targ:"flag,short=f,desc=Force close even if AC incomplete"`
}

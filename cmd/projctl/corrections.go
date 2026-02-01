package main

import (
	"fmt"
	"os"

	"github.com/toejough/projctl/internal/corrections"
)

type correctionsLogArgs struct {
	Dir     string `targ:"flag,short=d,desc=Project directory (omit for global)"`
	Message string `targ:"flag,short=m,required,desc=Correction message"`
	Context string `targ:"flag,short=c,desc=Context for the correction"`
	Session string `targ:"flag,short=s,desc=Session ID (optional)"`
}

func correctionsLog(args correctionsLogArgs) error {
	opts := corrections.LogOpts{
		SessionID: args.Session,
	}

	if args.Dir == "" {
		// Global corrections
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		return corrections.LogGlobal(args.Message, args.Context, opts, homeDir, nil)
	}

	// Project-specific corrections
	return corrections.Log(args.Dir, args.Message, args.Context, opts, nil)
}

type correctionsCountArgs struct {
	Dir     string `targ:"flag,short=d,desc=Project directory (omit for global)"`
	Since   string `targ:"flag,desc=Filter to entries since timestamp (RFC3339)"`
	Session string `targ:"flag,short=s,desc=Filter to specific session"`
}

func correctionsCount(args correctionsCountArgs) error {
	var entries []corrections.Entry
	var err error

	if args.Dir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		entries, err = corrections.ReadGlobal(homeDir)
	} else {
		entries, err = corrections.Read(args.Dir)
	}

	if err != nil {
		return err
	}

	// Filter by session if specified
	if args.Session != "" {
		filtered := make([]corrections.Entry, 0)
		for _, e := range entries {
			if e.SessionID == args.Session {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	// Filter by since timestamp if specified
	if args.Since != "" {
		filtered := make([]corrections.Entry, 0)
		for _, e := range entries {
			if e.Timestamp >= args.Since {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	fmt.Println(len(entries))
	return nil
}

package corrections

import (
	"fmt"
	"os"
)

// AnalyzeArgs holds arguments for the corrections analyze command.
type AnalyzeArgs struct {
	Dir            string `targ:"flag,short=d,desc=Project directory (omit for global)"`
	MinOccurrences int    `targ:"flag,short=n,desc=Minimum occurrences to report (default: 2)"`
}

// CountArgs holds arguments for the corrections count command.
type CountArgs struct {
	Dir     string `targ:"flag,short=d,desc=Project directory (omit for global)"`
	Since   string `targ:"flag,desc=Filter to entries since timestamp (RFC3339)"`
	Session string `targ:"flag,short=s,desc=Filter to specific session"`
}

// LogArgs holds arguments for the corrections log command.
type LogArgs struct {
	Dir     string `targ:"flag,short=d,desc=Project directory (omit for global)"`
	Message string `targ:"flag,short=m,required,desc=Correction message"`
	Context string `targ:"flag,short=c,desc=Context for the correction"`
	Session string `targ:"flag,short=s,desc=Session ID (optional)"`
}

// RunAnalyze detects patterns in corrections.
func RunAnalyze(args AnalyzeArgs) error {
	opts := AnalyzeOpts{}
	if args.MinOccurrences > 0 {
		opts.MinOccurrences = args.MinOccurrences
	}

	var (
		patterns []Pattern
		err      error
	)

	if args.Dir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		patterns, err = AnalyzeGlobal(homeDir, opts, RealFS{})
		if err != nil {
			return err
		}
	} else {
		patterns, err = Analyze(args.Dir, opts, RealFS{})
		if err != nil {
			return err
		}
	}

	if len(patterns) == 0 {
		fmt.Println("No patterns found.")
		return nil
	}

	fmt.Printf("Found %d correction patterns:\n\n", len(patterns))

	for i, p := range patterns {
		fmt.Printf("%d. **%s** (count: %d)\n", i+1, p.Message, p.Count)
		fmt.Printf("   Proposed rule: %s\n\n", p.Proposal)
	}

	return nil
}

// RunCount counts correction entries with optional filters.
func RunCount(args CountArgs) error {
	var (
		entries []Entry
		err     error
	)

	if args.Dir == "" {
		homeDir, errHome := os.UserHomeDir()
		if errHome != nil {
			return fmt.Errorf("failed to get home directory: %w", errHome)
		}

		entries, err = ReadGlobal(homeDir, RealFS{})
	} else {
		entries, err = Read(args.Dir, RealFS{})
	}

	if err != nil {
		return err
	}

	if args.Session != "" {
		filtered := make([]Entry, 0)

		for _, e := range entries {
			if e.SessionID == args.Session {
				filtered = append(filtered, e)
			}
		}

		entries = filtered
	}

	if args.Since != "" {
		filtered := make([]Entry, 0)

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

// RunLog logs a correction entry.
func RunLog(args LogArgs) error {
	opts := LogOpts{
		SessionID: args.Session,
	}

	if args.Dir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		return LogGlobal(args.Message, args.Context, opts, homeDir, nil, RealFS{})
	}

	return Log(args.Dir, args.Message, args.Context, opts, nil, RealFS{})
}
